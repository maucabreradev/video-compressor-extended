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

// Run displays the current configured batch in an interactive terminal. Input
// selection and configuration review are left to the later interactive-workflow
// task.
func Run(ctx context.Context, application *app.Application, cfg config.Config, input io.Reader, output io.Writer) error {
	if ctx == nil {
		return errors.New("TUI context must not be nil")
	}
	if application == nil {
		return errors.New("TUI application must not be nil")
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
	model := NewModel(control, cancel)
	program := tea.NewProgram(model,
		tea.WithInput(input),
		tea.WithOutput(output),
		tea.WithAltScreen(),
		tea.WithoutSignalHandler(),
	)
	runResult := make(chan error, 1)
	go func() {
		err := application.RunWithControl(runContext, cfg, control, func(event app.RunEvent) error {
			program.Send(runEventMsg{event: event})
			return nil
		})
		program.Send(runFinishedMsg{err: err})
		runResult <- err
	}()

	go func() {
		<-signalContext.Done()
		program.Send(externalCancelMsg{})
	}()

	_, programErr := program.Run()
	if programErr != nil {
		cancel()
	}
	runErr := <-runResult
	return errors.Join(programErr, runErr)
}
