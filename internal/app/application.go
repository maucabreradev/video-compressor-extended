package app

import (
	"bytes"
	"context"
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
	"github.com/maucabreradev/video-compressor-extended/internal/media/compression"
	"github.com/maucabreradev/video-compressor-extended/internal/media/discovery"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/media/subtitle"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

// Application coordinates media discovery, dependency checks, and processing
// use cases shared by CLI and TUI entry points.
type Application struct {
	runner          process.CommandRunner
	output          io.Writer
	runEventHandler RunEventHandler
	runEventMu      sync.Mutex
	platform        string
	amdDeviceFinder func() ([]string, error)
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
	Plans            []outputplan.OutputPlan
	Diagnostics      diagnostics.Report
	SubtitlesEnabled bool
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

// Doctor reports tool and AMD encoder readiness without opening user media.
func (application *Application) Doctor(ctx context.Context, cfg config.Config) error {
	if err := application.CheckPlatform(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	report := application.diagnose(ctx, diagnostics.Options{
		VideoCodec:       cfg.VideoCodec,
		WhisperModelPath: cfg.WhisperModelPath,
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
	inputs, err := discovery.Discover([]string{cfg.InputRoot}, outputRoot)
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
	report := application.diagnose(ctx, diagnostics.Options{
		VideoCodec:       cfg.VideoCodec,
		WhisperModelPath: cfg.WhisperModelPath,
		SkipWhisper:      strings.TrimSpace(cfg.WhisperModelPath) == "",
	})
	if err := ctx.Err(); err != nil {
		return RunPreview{}, err
	}
	return RunPreview{
		Plans: plans, Diagnostics: report,
		SubtitlesEnabled: strings.TrimSpace(cfg.WhisperModelPath) != "",
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
	inputs, err := discovery.Discover([]string{cfg.InputRoot}, outputRoot)
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
	report := application.diagnose(ctx, diagnostics.Options{
		VideoCodec:  cfg.VideoCodec,
		SkipWhisper: true,
	})
	if err := ctx.Err(); err != nil {
		return err
	}
	if report.HasFailures() {
		var details bytes.Buffer
		if err := report.WriteReport(&details); err != nil {
			return fmt.Errorf("write diagnostics report: %w", err)
		}
		return fmt.Errorf("compression prerequisites are unavailable:\n%s", details.String())
	}
	backend, err := backendForPlatform(application.platform)
	if err != nil {
		return err
	}

	var whisperPath string
	var whisperLookupErr error
	if strings.TrimSpace(cfg.WhisperModelPath) != "" {
		whisperPath, whisperLookupErr = application.runner.LookPath("whisper-cli")
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
		reservedDestinations = append(reservedDestinations, plan.Video, plan.Subtitle)
	}
	for index := range plans {
		plan := &plans[index]
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
		if strings.TrimSpace(cfg.WhisperModelPath) == "" {
			continue
		}
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

	tasks := make([]scheduler.Task, 0, len(plans)*2)
	for index, outputPlan := range plans {
		plan := outputPlan
		tasks = append(tasks, scheduler.Task{
			Source: plan.Source, Destination: plan.Video,
			Stage: scheduler.Compression, Skip: resume[index].videoSkipped,
			Run: func(taskContext context.Context) error {
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
				err := compression.Compress(taskContext, application.runner, plan, compression.Options{
					FFmpegPath:  report.FFmpegPath,
					FFprobePath: report.FFprobePath,
					Duration:    duration,
					Encoding:    encoding,
					Priority:    process.Priority(resources.ProcessPriority),
				}, onProgress)
				progressErrMu.Lock()
				defer progressErrMu.Unlock()
				return errors.Join(err, progressErr)
			},
		})
		if strings.TrimSpace(cfg.WhisperModelPath) != "" {
			tasks = append(tasks, scheduler.Task{
				Source: plan.Source, Destination: plan.Subtitle,
				Stage: scheduler.Transcription, Skip: resume[index].subtitleSkipped,
				Run: func(taskContext context.Context) error {
					if whisperLookupErr != nil {
						return fmt.Errorf("whisper.cpp executable \"whisper-cli\" was not found; install it and add it to PATH: %w", whisperLookupErr)
					}
					return subtitle.Generate(taskContext, application.runner, plan, subtitle.Options{
						FFmpegPath: report.FFmpegPath, WhisperPath: whisperPath,
						ModelPath: cfg.WhisperModelPath, Language: cfg.Language,
						Threads: resources.WhisperThreads, UseGPU: resources.WhisperGPU,
						Priority: process.Priority(resources.ProcessPriority),
					})
				},
			})
		}
	}

	results, schedulerErr := scheduler.RunWithControl(ctx, tasks, scheduler.Limits{
		EncodeWorkers: resources.EncodeWorkers, SubtitleWorkers: resources.SubtitleWorkers,
	}, control, func(event scheduler.Event) error {
		var runErrors []error
		if event.State == scheduler.Running && event.Stage == scheduler.Compression {
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
		if err := emitRunEvent(RunEvent{
			Kind: RunCompleted, Source: plan.Source, Destination: plan.Video,
		}); err != nil {
			runErrors = append(runErrors, fmt.Errorf("report completion of %q: %w", plan.Source, err))
		}
	}
	return errors.Join(runErrors...)
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

func backendForPlatform(platform string) (ffmpeg.Backend, error) {
	switch platform {
	case "windows":
		return ffmpeg.BackendAMF, nil
	case "linux":
		return ffmpeg.BackendVAAPI, nil
	default:
		return "", fmt.Errorf("AMD hardware encoding is not supported on %s", platform)
	}
}

// CheckPlatform reports whether the current runtime is supported by VCX 2.0.
func (application *Application) CheckPlatform() error {
	if application.platform == "windows" {
		return nil
	}
	return fmt.Errorf("VCX 2.0 currently supports Windows only; platform %q is not supported", application.platform)
}
