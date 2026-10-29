package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
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

	signalContext, stopSignals := signal.NotifyContext(ctx, os.Interrupt)
	defer stopSignals()
	runContext, cancel := context.WithCancel(signalContext)
	defer cancel()

	control := scheduler.NewDispatchControl()
	model := NewWorkflowModel(cfg, application.Preview, control, cancel)
	model.workflow.ctx = runContext
	program := tea.NewProgram(model,
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
