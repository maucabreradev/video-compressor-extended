package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maucabreradev/video-compressor-extended/internal/cli"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
)

var version = "2.0.0-dev"

func main() {
	command := cli.NewRootCommand(version, cli.Handlers{
		Doctor: func(ctx context.Context, cfg config.Config) error {
			report := (diagnostics.Checker{}).Diagnose(ctx, diagnostics.Options{
				VideoCodec:       cfg.VideoCodec,
				WhisperModelPath: cfg.WhisperModelPath,
			})
			if err := report.WriteReport(os.Stdout); err != nil {
				return fmt.Errorf("write diagnostics report: %w", err)
			}
			if report.HasFailures() {
				return fmt.Errorf("one or more dependencies or hardware capabilities are unavailable")
			}
			return nil
		},
	})
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
