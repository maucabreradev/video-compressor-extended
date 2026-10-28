package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/media/compression"
	"github.com/maucabreradev/video-compressor-extended/internal/media/discovery"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

// Application coordinates media discovery, dependency checks, and processing
// use cases shared by CLI and TUI entry points.
type Application struct {
	runner          process.CommandRunner
	output          io.Writer
	platform        string
	amdDeviceFinder func() ([]string, error)
}

// New creates an application service. A nil runner selects the system process
// runner; a nil output writer discards informational output.
func New(runner process.CommandRunner, output io.Writer) *Application {
	if runner == nil {
		runner = process.Runner{}
	}
	if output == nil {
		output = io.Discard
	}
	return &Application{runner: runner, output: output, platform: runtime.GOOS}
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

	for _, plan := range plans {
		if _, err := fmt.Fprintf(application.output, "Compressing %s\n", filepath.Base(plan.Source)); err != nil {
			return fmt.Errorf("write progress: %w", err)
		}
		err := compression.Compress(ctx, application.runner, plan, compression.Options{
			FFmpegPath:  report.FFmpegPath,
			FFprobePath: report.FFprobePath,
			Encoding: ffmpeg.Options{
				Backend:     backend,
				Codec:       cfg.VideoCodec,
				VAAPIDevice: report.EncoderDevice,
				MaxWidth:    cfg.MaxWidth,
				MaxHeight:   cfg.MaxHeight,
				FrameRate:   cfg.FrameRate,
			},
		}, nil)
		if err != nil {
			return fmt.Errorf("compress %q: %w", plan.Source, err)
		}
		if _, err := fmt.Fprintf(application.output, "Completed %s -> %s\n", filepath.Base(plan.Source), plan.Video); err != nil {
			return fmt.Errorf("write completion status: %w", err)
		}
	}
	return nil
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
