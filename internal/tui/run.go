package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

// Run lets users choose inputs, review configuration and dependencies, and
// confirm the batch before the application starts processing media.
func Run(ctx context.Context, application *app.Application, cfg config.Config, input io.Reader, output io.Writer) error {
	if ctx == nil {
		return errors.New("TUI context must not be nil")
	}
	if application == nil {
		return errors.New("TUI application must not be nil")
	}
	if err := application.CheckPlatform(); err != nil {
		return err
	}
	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = os.Stdout
	}
	baseConfig := cfg
	preferencesPath, preferencePathErr := userPreferencesPath()
	var preferenceError string
	if preferencePathErr != nil {
		preferenceError = preferencePathErr.Error()
	} else {
		preferences, err := loadPreferences(preferencesPath)
		if err != nil {
			preferenceError = fmt.Sprintf("Saved preferences could not be loaded: %v. Press r to reset them.", err)
		} else {
			cfg = applyPreferences(cfg, preferences)
		}
	}

	signalContext, stopSignals := signal.NotifyContext(ctx, os.Interrupt)
	defer stopSignals()
	runContext, cancel := context.WithCancel(signalContext)
	defer cancel()

	control := scheduler.NewDispatchControl()
	model := NewWorkflowModel(cfg, application.Preview, control, cancel)
	jobs, recoveryListErr := application.ListRecoveryJobs(runContext)
	model.setRecoveryJobs(jobs, recoveryListErr)
	model.workflow.baseConfig = baseConfig
	model.workflow.preferenceError = preferenceError
	if preferencePathErr == nil {
		model.workflow.savePreferences = func(runConfig config.Config) error {
			return savePreferences(preferencesPath, runConfig)
		}
		model.workflow.resetPreferences = func() error {
			return resetPreferences(preferencesPath)
		}
	}
	model.workflow.ctx = runContext
	var program *tea.Program
	model.SetWorkflowPreviewWithCallbacks(func(previewContext context.Context, previewConfig config.Config, callbacks app.RuntimeSetupCallbacks) (app.RunPreview, error) {
		var lastProgress time.Time
		callbacks.Consent = func(asset runtimeassets.Asset) bool {
			reply := make(chan bool, 1)
			program.Send(workflowAssetConsentMsg{asset: asset, reply: reply})
			select {
			case accepted := <-reply:
				return accepted
			case <-previewContext.Done():
				return false
			}
		}
		callbacks.Progress = func(progress runtimeassets.DownloadProgress) {
			now := time.Now()
			if progress.ReceivedBytes < progress.TotalBytes && !lastProgress.IsZero() && now.Sub(lastProgress) < 150*time.Millisecond {
				return
			}
			lastProgress = now
			program.Send(workflowAssetProgressMsg{progress: progress})
		}
		return application.PreviewWithCallbacks(previewContext, previewConfig, callbacks)
	})
	program = tea.NewProgram(model,
		tea.WithInput(input),
		tea.WithOutput(output),
		tea.WithAltScreen(),
		tea.WithoutSignalHandler(),
	)
	model.startRun = func(runConfig config.Config) {
		go func() {
			err := application.RunWithControl(runContext, runConfig, control, func(event app.RunEvent) error {
				program.Send(runEventMsg{event: event})
				return nil
			})
			program.Send(runFinishedMsg{err: err})
		}()
	}
	model.startResume = func(jobID string) {
		go func() {
			var lastProgress time.Time
			callbacks := app.RuntimeSetupCallbacks{
				Consent: func(asset runtimeassets.Asset) bool {
					reply := make(chan bool, 1)
					program.Send(workflowAssetConsentMsg{asset: asset, reply: reply})
					select {
					case accepted := <-reply:
						return accepted
					case <-runContext.Done():
						return false
					}
				},
				Progress: func(progress runtimeassets.DownloadProgress) {
					now := time.Now()
					if progress.ReceivedBytes < progress.TotalBytes && !lastProgress.IsZero() && now.Sub(lastProgress) < 150*time.Millisecond {
						return
					}
					lastProgress = now
					program.Send(workflowAssetProgressMsg{progress: progress})
				},
			}
			err := application.ResumeSavedJobWithCallbacks(runContext, jobID, control, func(event app.RunEvent) error {
				program.Send(runEventMsg{event: event})
				return nil
			}, callbacks)
			program.Send(runFinishedMsg{err: err})
		}()
	}
	go func() {
		<-signalContext.Done()
		program.Send(externalCancelMsg{})
	}()

	finalModel, programErr := program.Run()
	if programErr != nil {
		cancel()
	}
	var runErr error
	if finalModel, ok := finalModel.(*Model); ok {
		runErr = finalModel.runErr
	}
	return errors.Join(programErr, runErr)
}
