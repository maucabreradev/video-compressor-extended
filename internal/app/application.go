package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/jobmanifest"
	"github.com/maucabreradev/video-compressor-extended/internal/media/compression"
	"github.com/maucabreradev/video-compressor-extended/internal/media/discovery"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/media/subtitle"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
	"github.com/maucabreradev/video-compressor-extended/internal/workspace"
)

// Application coordinates media discovery, dependency checks, and processing
// use cases shared by CLI and TUI entry points.
type Application struct {
	runner           process.CommandRunner
	output           io.Writer
	runEventHandler  RunEventHandler
	runEventMu       sync.Mutex
	platform         string
	amdDeviceFinder  func() ([]string, error)
	runtimeAssets    RuntimeAssetManager
	assetConsent     func(runtimeassets.Asset) bool
	workspaceManager workspace.Manager
	jobManifests     *jobmanifest.Store
}

type RuntimeAssetManager interface {
	Prepare(context.Context, string, func(runtimeassets.Asset) bool, func(runtimeassets.DownloadProgress)) (runtimeassets.PreparedAsset, error)
}

// RuntimeSetupCallbacks report managed dependency consent and download progress
// to the interactive workflow.
type RuntimeSetupCallbacks struct {
	Consent  func(runtimeassets.Asset) bool
	Progress func(runtimeassets.DownloadProgress)
}

// RunEventKind identifies an event emitted while a batch run processes a file.
type RunEventKind string

const (
	RunStarted           RunEventKind = "started"
	RunProgress          RunEventKind = "progress"
	RunCompleted         RunEventKind = "completed"
	RunStageStateChanged RunEventKind = "stage_state"
)

// RunEvent carries structured batch state to a presentation layer.
type RunEvent struct {
	Kind          RunEventKind
	Source        string
	Destination   string
	Stage         scheduler.Stage
	State         scheduler.State
	Err           error
	DurationKnown bool
	Progress      process.FFmpegProgress
}

// RunEventHandler consumes one structured batch event. Callback errors are
// collected and returned after scheduled work finishes.
type RunEventHandler func(RunEvent) error

// RunPreview contains the work plan and dependency readiness shown before an
// interactive run begins.
type RunPreview struct {
	Plans                                []outputplan.OutputPlan
	Diagnostics                          diagnostics.Report
	CompressionEnabled                   bool
	SubtitlesEnabled                     bool
	EncoderBackend                       ffmpeg.Backend
	EncoderName                          string
	SoftwareFallbackAvailable            bool
	SoftwareFallbackConfirmationRequired bool
	SoftwareFallbackWarning              string
}

// New creates an application service. A nil runner selects the system process
// runner; a nil output writer discards diagnostic report output.
func New(runner process.CommandRunner, output io.Writer) *Application {
	if runner == nil {
		runner = process.Runner{}
	}
	if output == nil {
		output = io.Discard
	}
	return &Application{runner: runner, output: output, platform: runtime.GOOS}
}

// SetRunEventHandler sets the presentation callback for batch lifecycle, stage,
// and FFmpeg progress events. Calls to the handler are serialized. A nil
// handler disables batch event reporting.
func (application *Application) SetRunEventHandler(handler RunEventHandler) {
	application.runEventHandler = handler
}

// SetRuntimeAssetManager configures managed dependency resolution. Passing nil
// restores system-path lookup for callers that supply their own executables.
func (application *Application) SetRuntimeAssetManager(manager RuntimeAssetManager) {
	application.runtimeAssets = manager
}

// SetRuntimeAssetConsentHandler sets the caller that presents and obtains
// consent for a pinned runtime download. A nil handler declines missing assets.
func (application *Application) SetRuntimeAssetConsentHandler(handler func(runtimeassets.Asset) bool) {
	application.assetConsent = handler
}

func (application *Application) runtimeSetupCallbacks() RuntimeSetupCallbacks {
	return RuntimeSetupCallbacks{Consent: application.assetConsent}
}

type runtimePaths struct {
	ffmpeg          string
	ffprobe         string
	whisper         string
	model           string
	ffmpegRevision  string
	whisperRevision string
	modelSHA256     string
}

func (application *Application) subtitlesEnabled(cfg config.Config) bool {
	if !cfg.SubtitlesEnabled {
		return false
	}
	// Callers without managed runtime support retain the explicit-model legacy
	// path. The shipped CLI supplies a manager and uses the default model.
	return application.runtimeAssets != nil || strings.TrimSpace(cfg.WhisperModelPath) != ""
}

func (application *Application) prepareRuntimeAssets(ctx context.Context, cfg config.Config, subtitlesEnabled bool, callbacks RuntimeSetupCallbacks) (runtimePaths, error) {
	paths := runtimePaths{ffmpeg: "ffmpeg", ffprobe: "ffprobe", whisper: "whisper-cli", model: cfg.WhisperModelPath}
	paths.ffmpegRevision = "system:" + paths.ffmpeg
	paths.whisperRevision = "system:" + paths.whisper
	if application.runtimeAssets == nil {
		return paths, nil
	}
	ffmpegAsset, err := application.runtimeAssets.Prepare(ctx, "ffmpeg", callbacks.Consent, callbacks.Progress)
	if err != nil {
		return runtimePaths{}, fmt.Errorf("prepare FFmpeg runtime: %w", err)
	}
	paths.ffmpeg, err = requiredAssetFile(ffmpegAsset, "ffmpeg")
	if err != nil {
		return runtimePaths{}, err
	}
	paths.ffmpegRevision = ffmpegAsset.Asset.Revision
	paths.ffprobe, err = requiredAssetFile(ffmpegAsset, "ffprobe")
	if err != nil {
		return runtimePaths{}, err
	}
	if !subtitlesEnabled {
		return paths, nil
	}
	whisperAsset, err := application.runtimeAssets.Prepare(ctx, "whisper-cli", callbacks.Consent, callbacks.Progress)
	if err != nil {
		return runtimePaths{}, fmt.Errorf("prepare whisper.cpp runtime: %w", err)
	}
	paths.whisper, err = requiredAssetFile(whisperAsset, "whisper-cli")
	if err != nil {
		return runtimePaths{}, err
	}
	paths.whisperRevision = whisperAsset.Asset.Revision
	if strings.TrimSpace(paths.model) == "" {
		modelAsset, err := application.runtimeAssets.Prepare(ctx, "whisper-model", callbacks.Consent, callbacks.Progress)
		if err != nil {
			return runtimePaths{}, fmt.Errorf("prepare default Whisper model: %w", err)
		}
		paths.model, err = requiredAssetFile(modelAsset, "model")
		if err != nil {
			return runtimePaths{}, err
		}
		paths.modelSHA256 = modelAsset.Asset.SHA256
	}
	return paths, nil
}

func requiredAssetFile(asset runtimeassets.PreparedAsset, name string) (string, error) {
	path := asset.Files[name]
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("runtime asset %q does not provide required file %q", asset.Asset.ID, name)
	}
	return path, nil
}

func selectedEncoder(report diagnostics.Report, codec string, compressionEnabled bool) (ffmpeg.Backend, string, bool) {
	if !compressionEnabled {
		return "", "", false
	}
	for _, check := range report.Checks {
		if check.Name == "AMD hardware encode probe" && check.Status == diagnostics.StatusOK {
			return ffmpeg.BackendAMF, report.EncoderName, false
		}
	}
	if report.SoftwareH264Ready && strings.EqualFold(strings.TrimSpace(codec), "h264") {
		return ffmpeg.BackendSoftware, "libx264", true
	}
	return "", report.EncoderName, false
}

func softwareFallbackWarning(available bool) string {
	if !available {
		return ""
	}
	return "AMD AMF is unavailable. Software H.264 may be slower and use more CPU; confirm before starting."
}

// Doctor reports tool and AMD encoder readiness without opening user media.
func (application *Application) Doctor(ctx context.Context, cfg config.Config) error {
	if err := application.CheckPlatform(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	subtitlesEnabled := cfg.SubtitlesEnabled
	paths, err := application.prepareRuntimeAssets(ctx, cfg, subtitlesEnabled, application.runtimeSetupCallbacks())
	if err != nil {
		return fmt.Errorf("prepare runtime dependencies: %w", err)
	}
	report := application.diagnose(ctx, diagnostics.Options{
		FFmpegCommand: paths.ffmpeg, FFprobeCommand: paths.ffprobe, WhisperCommand: paths.whisper,
		VideoCodec: cfg.VideoCodec, WhisperModelPath: paths.model,
		SkipWhisper: !subtitlesEnabled, SkipFFprobe: !cfg.CompressionEnabled,
		SkipEncoder:                !cfg.CompressionEnabled,
		EnableSoftwareH264Fallback: cfg.CompressionEnabled && strings.EqualFold(cfg.VideoCodec, "h264"),
	})
	if err := report.WriteReport(application.output); err != nil {
		return fmt.Errorf("write diagnostics report: %w", err)
	}
	if report.HasFailures() {
		return fmt.Errorf("one or more dependencies or hardware capabilities are unavailable")
	}
	return nil
}

// Run processes discovered inputs through independent, bounded compression and
// transcription stages. It never falls back to CPU encoding.
func (application *Application) Run(ctx context.Context, cfg config.Config) error {
	return application.RunWithControl(ctx, cfg, nil, nil)
}

// Preview validates settings, discovers selected media, plans destinations,
// and checks the dependencies required by the requested stages. It never
// starts encoding or transcription.
func (application *Application) Preview(ctx context.Context, cfg config.Config) (RunPreview, error) {
	return application.PreviewWithCallbacks(ctx, cfg, application.runtimeSetupCallbacks())
}

// PreviewWithCallbacks previews a run while reporting consent requests and
// dependency acquisition progress to the caller. It never starts media work.
func (application *Application) PreviewWithCallbacks(ctx context.Context, cfg config.Config, callbacks RuntimeSetupCallbacks) (RunPreview, error) {
	if callbacks.Consent == nil {
		callbacks.Consent = application.assetConsent
	}
	if err := application.CheckPlatform(); err != nil {
		return RunPreview{}, err
	}
	if err := cfg.Validate(); err != nil {
		return RunPreview{}, fmt.Errorf("validate configuration: %w", err)
	}
	if ctx == nil {
		return RunPreview{}, fmt.Errorf("preview context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return RunPreview{}, err
	}
	outputRoot, err := resolveOutputRoot(cfg.InputRoot, cfg.OutputDir)
	if err != nil {
		return RunPreview{}, err
	}
	inputs, err := discovery.Discover(configuredInputPaths(cfg), outputRoot)
	if err != nil {
		return RunPreview{}, err
	}
	if len(inputs) == 0 {
		return RunPreview{}, fmt.Errorf("no supported video files were found under %q", cfg.InputRoot)
	}
	plans, err := outputplan.PlanOutputs(cfg.InputRoot, outputRoot, inputs)
	if err != nil {
		return RunPreview{}, fmt.Errorf("plan output paths: %w", err)
	}
	subtitlesEnabled := application.subtitlesEnabled(cfg)
	paths, err := application.prepareRuntimeAssets(ctx, cfg, subtitlesEnabled, callbacks)
	if err != nil {
		return RunPreview{}, fmt.Errorf("prepare runtime dependencies: %w", err)
	}
	report := application.diagnose(ctx, diagnostics.Options{
		FFmpegCommand: paths.ffmpeg, FFprobeCommand: paths.ffprobe, WhisperCommand: paths.whisper,
		VideoCodec: cfg.VideoCodec, WhisperModelPath: paths.model,
		SkipWhisper: !subtitlesEnabled, SkipFFprobe: !cfg.CompressionEnabled,
		SkipEncoder:                !cfg.CompressionEnabled,
		EnableSoftwareH264Fallback: cfg.CompressionEnabled && strings.EqualFold(cfg.VideoCodec, "h264"),
	})
	if err := ctx.Err(); err != nil {
		return RunPreview{}, err
	}
	backend, encoder, softwareFallback := selectedEncoder(report, cfg.VideoCodec, cfg.CompressionEnabled)
	return RunPreview{
		Plans: plans, Diagnostics: report, CompressionEnabled: cfg.CompressionEnabled,
		SubtitlesEnabled: subtitlesEnabled, EncoderBackend: backend, EncoderName: encoder,
		SoftwareFallbackAvailable:            softwareFallback,
		SoftwareFallbackConfirmationRequired: softwareFallback && !cfg.AllowSoftwareEncoding,
		SoftwareFallbackWarning:              softwareFallbackWarning(softwareFallback),
	}, nil
}

// RunWithControl runs media work with an optional dispatch control and
// per-run event handler. Pausing holds queued tasks but does not suspend
// processes that are already running.
func (application *Application) RunWithControl(
	ctx context.Context,
	cfg config.Config,
	control *scheduler.DispatchControl,
	handler RunEventHandler,
) error {
	if err := application.CheckPlatform(); err != nil {
		return err
	}
	if handler == nil {
		handler = application.runEventHandler
	}
	emitRunEvent := func(event RunEvent) error {
		if handler == nil {
			return nil
		}
		application.runEventMu.Lock()
		defer application.runEventMu.Unlock()
		return handler(event)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	if ctx == nil {
		return fmt.Errorf("run context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	outputRoot, err := resolveOutputRoot(cfg.InputRoot, cfg.OutputDir)
	if err != nil {
		return err
	}
	inputs, err := discovery.Discover(configuredInputPaths(cfg), outputRoot)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return fmt.Errorf("no supported video files were found under %q", cfg.InputRoot)
	}
	plans, err := outputplan.PlanOutputs(cfg.InputRoot, outputRoot, inputs)
	if err != nil {
		return fmt.Errorf("plan output paths: %w", err)
	}
	subtitlesEnabled := application.subtitlesEnabled(cfg)
	paths, err := application.prepareRuntimeAssets(ctx, cfg, subtitlesEnabled, application.runtimeSetupCallbacks())
	if err != nil {
		return fmt.Errorf("prepare runtime dependencies: %w", err)
	}
	report := application.diagnose(ctx, diagnostics.Options{
		FFmpegCommand: paths.ffmpeg, FFprobeCommand: paths.ffprobe, WhisperCommand: paths.whisper,
		VideoCodec: cfg.VideoCodec, WhisperModelPath: paths.model,
		SkipWhisper: application.runtimeAssets == nil || !subtitlesEnabled,
		SkipFFprobe: !cfg.CompressionEnabled, SkipEncoder: !cfg.CompressionEnabled,
		EnableSoftwareH264Fallback: cfg.CompressionEnabled && strings.EqualFold(cfg.VideoCodec, "h264"),
	})
	if err := ctx.Err(); err != nil {
		return err
	}
	backend, _, softwareFallback := selectedEncoder(report, cfg.VideoCodec, cfg.CompressionEnabled)
	if softwareFallback && !cfg.AllowSoftwareEncoding {
		return fmt.Errorf("AMD AMF is unavailable; software H.264 is ready but may be slower and use more CPU; explicit confirmation is required (set allow_software_encoding=true or --allow-software-encoding)")
	}
	if report.HasFailures() {
		var details bytes.Buffer
		if err := report.WriteReport(&details); err != nil {
			return fmt.Errorf("write diagnostics report: %w", err)
		}
		return fmt.Errorf("processing prerequisites are unavailable:\n%s", details.String())
	}
	if cfg.CompressionEnabled && backend == "" {
		return fmt.Errorf("no verified compression backend is available")
	}

	var whisperPath string
	var whisperLookupErr error
	if subtitlesEnabled {
		whisperPath = paths.whisper
		if application.runtimeAssets == nil {
			whisperPath, whisperLookupErr = application.runner.LookPath(paths.whisper)
		}
	}
	resources := cfg.EffectiveResources()
	encoding := ffmpeg.Options{
		Backend: backend, Codec: cfg.VideoCodec, VAAPIDevice: report.EncoderDevice,
		MaxWidth: cfg.MaxWidth, MaxHeight: cfg.MaxHeight, FrameRate: cfg.FrameRate,
	}
	resume := make([]struct {
		videoSkipped    bool
		subtitleSkipped bool
	}, len(plans))
	reservedDestinations := make([]string, 0, len(plans)*2)
	for _, plan := range plans {
		if cfg.CompressionEnabled {
			reservedDestinations = append(reservedDestinations, plan.Video)
		}
		if subtitlesEnabled {
			reservedDestinations = append(reservedDestinations, plan.Subtitle)
		}
	}
	for index := range plans {
		plan := &plans[index]
		if cfg.CompressionEnabled {
			videoPath, videoSkipped, err := outputplan.ResolveArtifactPath(
				plan.Video, reservedDestinations,
				func(path string) error {
					return compression.ValidateOutput(ctx, application.runner, report.FFprobePath, path, encoding)
				},
			)
			if err != nil {
				return fmt.Errorf("inspect resumable video output for %q: %w", plan.Source, err)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			plan.Video = videoPath
			resume[index].videoSkipped = videoSkipped
			reservedDestinations = append(reservedDestinations, videoPath)
		}
		if subtitlesEnabled {
			subtitlePath, subtitleSkipped, err := outputplan.ResolveArtifactPath(
				plan.Subtitle, reservedDestinations,
				subtitle.ValidateOutput,
			)
			if err != nil {
				return fmt.Errorf("inspect resumable subtitle output for %q: %w", plan.Source, err)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			plan.Subtitle = subtitlePath
			resume[index].subtitleSkipped = subtitleSkipped
			reservedDestinations = append(reservedDestinations, subtitlePath)
		}
	}
	manifestJobID := ""
	if cfg.CompressionEnabled || subtitlesEnabled {
		manifestJobID, err = application.createJobManifest(cfg, plans, backend, paths, resume, subtitlesEnabled)
		if err != nil {
			return fmt.Errorf("create job manifest: %w", err)
		}
	}

	tasks := make([]scheduler.Task, 0, len(plans)*2)
	for index, outputPlan := range plans {
		plan := outputPlan
		if cfg.CompressionEnabled {
			tasks = append(tasks, scheduler.Task{
				Source: plan.Source, Destination: plan.Video,
				Stage: scheduler.Compression, Skip: resume[index].videoSkipped,
				Run: func(taskContext context.Context) error {
					shortWorkspace, err := application.workspaceManager.Create(filepath.Dir(plan.Video))
					if err != nil {
						return errors.Join(err, application.updateJobStage(manifestJobID, index, scheduler.Compression, jobmanifest.StageFailed, ""))
					}
					workspaceID := filepath.Base(shortWorkspace.Path())
					if err := application.updateJobStage(manifestJobID, index, scheduler.Compression, jobmanifest.StageRunning, workspaceID); err != nil {
						return errors.Join(err, shortWorkspace.Remove())
					}
					duration := probeMediaDuration(taskContext, application.runner, report.FFprobePath, plan.Source)
					var progressErr error
					var progressErrMu sync.Mutex
					var onProgress func(process.FFmpegProgress)
					if handler != nil {
						onProgress = func(progress process.FFmpegProgress) {
							err := emitRunEvent(RunEvent{
								Kind: RunProgress, Source: plan.Source, Stage: scheduler.Compression,
								DurationKnown: duration > 0, Progress: progress,
							})
							if err != nil {
								progressErrMu.Lock()
								progressErr = errors.Join(progressErr, err)
								progressErrMu.Unlock()
							}
						}
					}
					err = compression.Compress(taskContext, application.runner, plan, compression.Options{
						FFmpegPath:    report.FFmpegPath,
						FFprobePath:   report.FFprobePath,
						WorkspacePath: shortWorkspace.Path(),
						Duration:      duration,
						Encoding:      encoding,
						Priority:      process.Priority(resources.ProcessPriority),
					}, onProgress)
					cleanupErr := shortWorkspace.Remove()
					stageState := jobmanifest.StagePublished
					if err != nil {
						stageState = jobmanifest.StageFailed
						if errors.Is(err, context.Canceled) || errors.Is(taskContext.Err(), context.Canceled) {
							stageState = jobmanifest.StageCancelled
						}
					}
					manifestErr := application.updateJobStage(manifestJobID, index, scheduler.Compression, stageState, "")
					progressErrMu.Lock()
					defer progressErrMu.Unlock()
					return errors.Join(err, cleanupErr, manifestErr, progressErr)
				},
			})
		}
		if subtitlesEnabled {
			tasks = append(tasks, scheduler.Task{
				Source: plan.Source, Destination: plan.Subtitle,
				Stage: scheduler.Transcription, Skip: resume[index].subtitleSkipped,
				Run: func(taskContext context.Context) error {
					if whisperLookupErr != nil {
						err := fmt.Errorf("whisper.cpp executable \"whisper-cli\" was not found; install it and add it to PATH: %w", whisperLookupErr)
						return errors.Join(err, application.updateJobStage(manifestJobID, index, scheduler.Transcription, jobmanifest.StageFailed, ""))
					}
					shortWorkspace, err := application.workspaceManager.Create(filepath.Dir(plan.Subtitle))
					if err != nil {
						return errors.Join(err, application.updateJobStage(manifestJobID, index, scheduler.Transcription, jobmanifest.StageFailed, ""))
					}
					workspaceID := filepath.Base(shortWorkspace.Path())
					if err := application.updateJobStage(manifestJobID, index, scheduler.Transcription, jobmanifest.StageRunning, workspaceID); err != nil {
						return errors.Join(err, shortWorkspace.Remove())
					}
					generationErr := subtitle.Generate(taskContext, application.runner, plan, subtitle.Options{
						FFmpegPath: report.FFmpegPath, WhisperPath: whisperPath,
						ModelPath: paths.model, WorkspacePath: shortWorkspace.Path(), Language: cfg.Language,
						Threads: resources.WhisperThreads, UseGPU: resources.WhisperGPU,
						Priority: process.Priority(resources.ProcessPriority),
					})
					cleanupErr := shortWorkspace.Remove()
					stageState := jobmanifest.StagePublished
					if generationErr != nil {
						stageState = jobmanifest.StageFailed
						if errors.Is(generationErr, context.Canceled) || errors.Is(taskContext.Err(), context.Canceled) {
							stageState = jobmanifest.StageCancelled
						}
					}
					manifestErr := application.updateJobStage(manifestJobID, index, scheduler.Transcription, stageState, "")
					return errors.Join(generationErr, cleanupErr, manifestErr)
				},
			})
		}
	}

	results, schedulerErr := scheduler.RunWithControl(ctx, tasks, scheduler.Limits{
		EncodeWorkers: resources.EncodeWorkers, SubtitleWorkers: resources.SubtitleWorkers,
	}, control, func(event scheduler.Event) error {
		var runErrors []error
		if event.State == scheduler.Running && (event.Stage == scheduler.Compression ||
			(!cfg.CompressionEnabled && event.Stage == scheduler.Transcription)) {
			if err := emitRunEvent(RunEvent{
				Kind: RunStarted, Source: event.Source, Stage: event.Stage, State: event.State,
			}); err != nil {
				runErrors = append(runErrors, err)
			}
		}
		if err := emitRunEvent(RunEvent{
			Kind: RunStageStateChanged, Source: event.Source, Destination: event.Destination,
			Stage: event.Stage,
			State: event.State, Err: event.Err,
		}); err != nil {
			runErrors = append(runErrors, err)
		}
		return errors.Join(runErrors...)
	})

	failedSources := make(map[string]bool)
	var runErrors []error
	if schedulerErr != nil {
		runErrors = append(runErrors, schedulerErr)
	}
	for _, result := range results {
		if result.Err == nil {
			continue
		}
		failedSources[result.Source] = true
		switch result.Stage {
		case scheduler.Compression:
			runErrors = append(runErrors, fmt.Errorf("compress %q: %w", result.Source, result.Err))
		case scheduler.Transcription:
			runErrors = append(runErrors, fmt.Errorf("generate subtitles for %q: %w", result.Source, result.Err))
		}
	}
	for _, plan := range plans {
		if failedSources[plan.Source] {
			continue
		}
		destination := plan.Video
		if !cfg.CompressionEnabled {
			destination = plan.Subtitle
		}
		if err := emitRunEvent(RunEvent{
			Kind: RunCompleted, Source: plan.Source, Destination: destination,
		}); err != nil {
			runErrors = append(runErrors, fmt.Errorf("report completion of %q: %w", plan.Source, err))
		}
	}
	return errors.Join(runErrors...)
}

func (application *Application) createJobManifest(
	cfg config.Config,
	plans []outputplan.OutputPlan,
	backend ffmpeg.Backend,
	paths runtimePaths,
	resume []struct {
		videoSkipped    bool
		subtitleSkipped bool
	},
	subtitlesEnabled bool,
) (string, error) {
	modelSHA256 := paths.modelSHA256
	if subtitlesEnabled && modelSHA256 == "" {
		modelInfo, statErr := os.Stat(paths.model)
		if statErr != nil || !modelInfo.Mode().IsRegular() || modelInfo.Size() == 0 {
			// An unavailable subtitle model must not prevent independent encoding.
			// The subtitle task will report the prerequisite failure itself.
			return "", nil
		}
		var err error
		modelSHA256, err = fileSHA256(paths.model)
		if err != nil {
			return "", fmt.Errorf("identify Whisper model: %w", err)
		}
	}
	items := make([]jobmanifest.Item, 0, len(plans))
	for index, plan := range plans {
		item, err := jobmanifest.NewItem(plan.Source, plan.Video, plan.Subtitle)
		if err != nil {
			return "", fmt.Errorf("create manifest item for %q: %w", plan.Source, err)
		}
		if index < len(resume) && resume[index].subtitleSkipped {
			item.Subtitles.State = jobmanifest.StageSkipped
		}
		if index < len(resume) && resume[index].videoSkipped {
			item.Compression.State = jobmanifest.StageSkipped
		}
		items = append(items, item)
	}
	identity := jobmanifest.ProcessingIdentity{
		CompressionEnabled: cfg.CompressionEnabled,
		SubtitlesEnabled:   subtitlesEnabled,
	}
	if cfg.CompressionEnabled {
		identity.VideoBackend = string(backend)
		identity.VideoCodec = cfg.VideoCodec
		identity.MaxWidth = cfg.MaxWidth
		identity.MaxHeight = cfg.MaxHeight
		identity.FrameRate = cfg.FrameRate
		identity.FFmpegRevision = paths.ffmpegRevision
	}
	if subtitlesEnabled {
		resources := cfg.EffectiveResources()
		identity.Language = cfg.Language
		identity.WhisperThreads = resources.WhisperThreads
		identity.WhisperGPU = resources.WhisperGPU
		identity.WhisperRevision = paths.whisperRevision
		identity.ModelSHA256 = modelSHA256
	}
	manifest, err := jobmanifest.NewManifest(identity, items)
	if err != nil {
		return "", err
	}
	store := application.jobManifests
	if store == nil {
		store, err = jobmanifest.NewStore("")
		if err != nil {
			return "", err
		}
		application.jobManifests = store
	}
	if err := store.Create(manifest); err != nil {
		return "", err
	}
	return manifest.JobID, nil
}

func (application *Application) updateJobStage(jobID string, itemIndex int, stage scheduler.Stage, state jobmanifest.StageState, workspaceID string) error {
	if jobID == "" {
		return nil
	}
	if application.jobManifests == nil {
		return fmt.Errorf("subtitle job manifest is unavailable")
	}
	return application.jobManifests.Update(jobID, func(manifest *jobmanifest.Manifest) error {
		if itemIndex < 0 || itemIndex >= len(manifest.Items) {
			return fmt.Errorf("job manifest item index %d is out of range", itemIndex)
		}
		record := jobmanifest.StageRecord{State: state}
		if workspaceID != "" {
			record.WorkspaceID = workspaceID
			switch stage {
			case scheduler.Compression:
				record.ArtifactName = "video.mp4"
			case scheduler.Transcription:
				record.ArtifactName = "transcript.srt"
			default:
				return fmt.Errorf("unsupported job stage %q", stage)
			}
		}
		switch stage {
		case scheduler.Compression:
			manifest.Items[itemIndex].Compression = record
		case scheduler.Transcription:
			manifest.Items[itemIndex].Subtitles = record
		default:
			return fmt.Errorf("unsupported job stage %q", stage)
		}
		return nil
	})
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %q: %w", path, err)
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(hasher, file)
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return "", fmt.Errorf("hash %q: %w", path, err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// configuredInputPaths preserves the legacy input_root behavior while resolving
// relative explicit selections against the common input root.
func configuredInputPaths(cfg config.Config) []string {
	if len(cfg.InputPaths) == 0 {
		return []string{cfg.InputRoot}
	}
	inputs := make([]string, len(cfg.InputPaths))
	for index, input := range cfg.InputPaths {
		if !filepath.IsAbs(input) {
			input = filepath.Join(cfg.InputRoot, input)
		}
		inputs[index] = input
	}
	return inputs
}
func probeMediaDuration(ctx context.Context, runner process.CommandRunner, ffprobePath, source string) time.Duration {
	stdout, _, err := runner.Run(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		source,
	)
	if err != nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(stdout)), 64)
	maxSeconds := float64(time.Duration(1<<63-1)) / float64(time.Second)
	if err != nil || seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > maxSeconds {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func (application *Application) diagnose(ctx context.Context, options diagnostics.Options) diagnostics.Report {
	return (diagnostics.Checker{
		Runner:          application.runner,
		Platform:        application.platform,
		AMDDeviceFinder: application.amdDeviceFinder,
	}).Diagnose(ctx, options)
}

func resolveOutputRoot(inputRoot, outputDir string) (string, error) {
	if filepath.IsAbs(outputDir) {
		return filepath.Clean(outputDir), nil
	}
	info, err := os.Stat(inputRoot)
	if err != nil {
		return "", fmt.Errorf("inspect input root %q: %w", inputRoot, err)
	}
	base := inputRoot
	if !info.IsDir() {
		base = filepath.Dir(inputRoot)
	}
	return filepath.Join(base, outputDir), nil
}

// CheckPlatform reports whether the current runtime is supported by VCX 2.0.
func (application *Application) CheckPlatform() error {
	if application.platform == "windows" {
		return nil
	}
	return fmt.Errorf("VCX 2.0 currently supports Windows only; platform %q is not supported", application.platform)
}
