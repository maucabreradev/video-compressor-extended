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
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/media/compression"
	"github.com/maucabreradev/video-compressor-extended/internal/media/discovery"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/media/subtitle"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

// Application coordinates media discovery, dependency checks, and processing
// use cases shared by CLI and TUI entry points.
type Application struct {
	runner          process.CommandRunner
	output          io.Writer
	runEventHandler RunEventHandler
	platform        string
	amdDeviceFinder func() ([]string, error)
}

// RunEventKind identifies an event emitted while a batch run processes a file.
type RunEventKind string

const (
	RunStarted   RunEventKind = "started"
	RunProgress  RunEventKind = "progress"
	RunCompleted RunEventKind = "completed"
)

// RunEvent carries structured batch state to a presentation layer.
type RunEvent struct {
	Kind          RunEventKind
	Source        string
	Destination   string
	DurationKnown bool
	Progress      process.FFmpegProgress
}

// RunEventHandler consumes one structured batch event. Returning an error
// stops the batch before starting the next file.
type RunEventHandler func(RunEvent) error

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

// SetRunEventHandler sets the presentation callback for batch lifecycle and
// FFmpeg progress events. A nil handler disables batch event reporting.
func (application *Application) SetRunEventHandler(handler RunEventHandler) {
	application.runEventHandler = handler
}

// Doctor reports tool and AMD encoder readiness without opening user media.
func (application *Application) Doctor(ctx context.Context, cfg config.Config) error {
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

// Run compresses each discovered input sequentially through the verified AMD
// backend. Subtitle generation and concurrent scheduling are added by later
// tasks; this use case never falls back to CPU encoding.
func (application *Application) Run(ctx context.Context, cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
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

	var runErrors []error
	for _, plan := range plans {
		if err := application.emitRunEvent(RunEvent{Kind: RunStarted, Source: plan.Source}); err != nil {
			return fmt.Errorf("report start of %q: %w", plan.Source, err)
		}
		duration := probeMediaDuration(ctx, application.runner, report.FFprobePath, plan.Source)
		var callbackErr error
		var onProgress func(process.FFmpegProgress)
		if application.runEventHandler != nil {
			onProgress = func(progress process.FFmpegProgress) {
				if callbackErr != nil {
					return
				}
				callbackErr = application.emitRunEvent(RunEvent{
					Kind: RunProgress, Source: plan.Source,
					DurationKnown: duration > 0, Progress: progress,
				})
			}
		}
		err := compression.Compress(ctx, application.runner, plan, compression.Options{
			FFmpegPath:  report.FFmpegPath,
			FFprobePath: report.FFprobePath,
			Duration:    duration,
			Encoding: ffmpeg.Options{
				Backend:     backend,
				Codec:       cfg.VideoCodec,
				VAAPIDevice: report.EncoderDevice,
				MaxWidth:    cfg.MaxWidth,
				MaxHeight:   cfg.MaxHeight,
				FrameRate:   cfg.FrameRate,
			},
		}, onProgress)
		var fileErrors []error
		if err != nil {
			fileErrors = append(fileErrors, fmt.Errorf("compress %q: %w", plan.Source, err))
		}
		if callbackErr != nil {
			fileErrors = append(fileErrors, fmt.Errorf("report progress for %q: %w", plan.Source, callbackErr))
		}
		if strings.TrimSpace(cfg.WhisperModelPath) != "" {
			whisperPath, err := application.runner.LookPath("whisper-cli")
			if err != nil {
				fileErrors = append(fileErrors, fmt.Errorf("generate subtitles for %q: whisper.cpp executable \"whisper-cli\" was not found; install it and add it to PATH: %w", plan.Source, err))
			} else if err := subtitle.Generate(ctx, application.runner, plan, subtitle.Options{
				FFmpegPath: report.FFmpegPath, WhisperPath: whisperPath,
				ModelPath: cfg.WhisperModelPath, Language: cfg.Language, Threads: cfg.WhisperThreads,
			}); err != nil {
				fileErrors = append(fileErrors, fmt.Errorf("generate subtitles for %q: %w", plan.Source, err))
			}
		}
		if len(fileErrors) > 0 {
			runErrors = append(runErrors, errors.Join(fileErrors...))
			continue
		}
		if err := application.emitRunEvent(RunEvent{
			Kind: RunCompleted, Source: plan.Source, Destination: plan.Video,
		}); err != nil {
			return fmt.Errorf("report completion of %q: %w", plan.Source, err)
		}
	}
	return errors.Join(runErrors...)
}

func (application *Application) emitRunEvent(event RunEvent) error {
	if application.runEventHandler == nil {
		return nil
	}
	return application.runEventHandler(event)
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
