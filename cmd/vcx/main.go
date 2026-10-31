package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/cli"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/tui"
)

var version = "2.0.0-dev"

func main() {
	os.Exit(runApplication(os.Args[1:], process.Runner{}, os.Stdout, os.Stderr))
}

func runApplication(args []string, runner process.CommandRunner, stdout, stderr io.Writer) int {
	application := app.New(runner, stdout)
	if executable, err := os.Executable(); err == nil {
		application.SetRuntimeAssetManager(runtimeassets.Manager{
			BundleRoot: filepath.Join(filepath.Dir(executable), "runtime-assets"),
		})
	}
	application.SetRunEventHandler(cli.NewBatchOutputReporter(stdout).Report)
	return executeCLI(args, applicationHandlers(application), stdout, stderr)
}

func applicationHandlers(application *app.Application) cli.Handlers {
	return cli.Handlers{
		TUI: func(ctx context.Context, cfg config.Config) error {
			return tui.Run(ctx, application, cfg, os.Stdin, os.Stdout)
		},
		Run:    application.Run,
		Doctor: application.Doctor,
		ListJobs: func(ctx context.Context) ([]cli.RecoveryJob, error) {
			jobs, err := application.ListRecoveryJobs(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]cli.RecoveryJob, 0, len(jobs))
			for _, job := range jobs {
				result = append(result, cli.RecoveryJob{
					ID: job.ID, UpdatedAt: job.UpdatedAt, SourceSummary: job.SourceSummary,
					OutputSummary: job.OutputSummary, CompressionState: job.CompressionState,
					SubtitleState: job.SubtitleState, Status: recoveryCLIStatus(job), Detail: job.Detail,
				})
			}
			return result, nil
		},
		ResumeJob: func(ctx context.Context, jobID string) error {
			return application.ResumeSavedJob(ctx, jobID, nil, nil)
		},
	}
}

func recoveryCLIStatus(job app.RecoveryJob) string {
	switch job.Status {
	case app.RecoveryJobReady:
		if isCompletedStageSummary(job.CompressionState) && isCompletedStageSummary(job.SubtitleState) {
			return "completed"
		}
		return "resumable"
	case app.RecoveryJobActive:
		return "active"
	default:
		return "invalid"
	}
}

func isCompletedStageSummary(summary string) bool {
	if summary == "not requested" {
		return true
	}
	if summary == "" {
		return false
	}
	for _, state := range strings.Split(summary, ", ") {
		if !strings.HasPrefix(state, "published ") && !strings.HasPrefix(state, "skipped ") {
			return false
		}
	}
	return true
}

func executeCLI(args []string, handlers cli.Handlers, stdout, stderr io.Writer) int {
	command := cli.NewRootCommand(version, handlers)
	command.SetArgs(args)
	command.SetOut(stdout)
	command.SetErr(stderr)
	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
