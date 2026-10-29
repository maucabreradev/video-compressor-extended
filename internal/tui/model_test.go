package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

func TestModelTracksQueueStageProgressAndActionableErrors(t *testing.T) {
	model := NewModel(scheduler.NewDispatchControl(), func() {})
	source := `C:\media\clip.mkv`
	for _, event := range []app.RunEvent{
		{Kind: app.RunStageStateChanged, Source: source, Stage: scheduler.Compression, State: scheduler.Queued},
		{Kind: app.RunStageStateChanged, Source: source, Stage: scheduler.Transcription, State: scheduler.Running},
		{Kind: app.RunProgress, Source: source, Stage: scheduler.Compression, DurationKnown: true,
			Progress: process.FFmpegProgress{Frame: 8, Percent: 40}},
		{Kind: app.RunStageStateChanged, Source: source, Stage: scheduler.Compression, State: scheduler.Failed, Err: errors.New("encoder device unavailable")},
	} {
		updated, _ := model.Update(runEventMsg{event: event})
		model = updated.(*Model)
	}

	job := model.jobs[source]
	if job == nil {
		t.Fatal("event did not create a queue row")
	}
	if job.compression.state != scheduler.Failed || job.compression.err != "encoder device unavailable" {
		t.Fatalf("compression state did not retain its actionable error: %+v", job.compression)
	}
	if job.transcription.state != scheduler.Running {
		t.Fatalf("transcription state was not tracked independently: %+v", job.transcription)
	}
	if job.progress == nil || job.progress.Frame != 8 || job.progress.Percent != 40 {
		t.Fatalf("compression progress was not retained: %+v", job.progress)
	}
	view := model.View()
	for _, want := range []string{
		"clip.mkv", "encoder device unavailable", "40.0%",
		"Files 1", "compression", "transcription", "failed 1", "running 1", "elapsed",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("TUI view is missing %q: %q", want, view)
		}
	}
}

func TestModelPauseKeyControlsSchedulerDispatch(t *testing.T) {
	control := scheduler.NewDispatchControl()
	model := NewModel(control, func() {})
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updated, _ := model.Update(key)
	model = updated.(*Model)
	if !model.paused || !control.IsPaused() {
		t.Fatal("first pause key did not pause queued dispatch")
	}
	updated, _ = model.Update(key)
	model = updated.(*Model)
	if model.paused || control.IsPaused() {
		t.Fatal("second pause key did not resume queued dispatch")
	}
}

func TestModelCancelWaitsForRunCompletionBeforeQuitting(t *testing.T) {
	cancelled := false
	model := NewModel(scheduler.NewDispatchControl(), func() { cancelled = true })
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model = updated.(*Model)
	if !cancelled || !model.cancelRequested {
		t.Fatal("cancel key did not request run cancellation")
	}
	if cmd != nil {
		t.Fatal("cancel key should wait for active processes before quitting")
	}
	updated, _ = model.Update(runFinishedMsg{err: context.Canceled})
	model = updated.(*Model)
	if !model.finished {
		t.Fatal("run completion was not recorded")
	}
	if !strings.Contains(model.View(), "CANCELLED") {
		t.Fatalf("cancellation was not clearly communicated: %q", model.View())
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q after completion should quit the terminal program")
	}
	_ = updated
}

func TestModelEscapeAndCtrlCRequestCancellationWithoutQuitting(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEsc}, {Type: tea.KeyCtrlC},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
	} {
		cancelled := false
		model := NewModel(scheduler.NewDispatchControl(), func() { cancelled = true })
		updated, command := model.Update(key)
		model = updated.(*Model)
		if !cancelled || !model.cancelRequested {
			t.Errorf("key %v did not request cancellation", key.Type)
		}
		if command != nil {
			t.Errorf("key %v quit before run cleanup", key.Type)
		}
	}
}

func TestModelShowsExternalCancellationAndKeepsWaitingForCleanup(t *testing.T) {
	model := NewModel(scheduler.NewDispatchControl(), func() {})
	updated, command := model.Update(externalCancelMsg{})
	model = updated.(*Model)
	if !model.cancelRequested || model.finished {
		t.Fatalf("external cancellation should request cancel and wait for cleanup: %+v", model)
	}
	if command != nil || !strings.Contains(model.View(), "CANCELLING") {
		t.Fatalf("external cancellation was not communicated safely: view=%q cmd=%v", model.View(), command)
	}
}

func TestModelResizesQueueViewport(t *testing.T) {
	model := NewModel(scheduler.NewDispatchControl(), func() {})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 92, Height: 28})
	model = updated.(*Model)
	if model.width != 92 || model.height != 28 {
		t.Fatalf("terminal size = %dx%d, want 92x28", model.width, model.height)
	}
	if model.viewport.Width <= 0 || model.viewport.Height <= 0 {
		t.Fatalf("viewport was not resized: %dx%d", model.viewport.Width, model.viewport.Height)
	}
}

func TestWorkflowEditsInputPreviewsPlanAndStartsOnlyAfterConfirmation(t *testing.T) {
	cfg := config.Defaults()
	cfg.InputRoot = "old-input"
	preparedConfig := ""
	started := false
	model := NewWorkflowModel(cfg, func(_ context.Context, prepared config.Config) (app.RunPreview, error) {
		preparedConfig = prepared.InputRoot
		return app.RunPreview{
			Plans:       []outputplan.OutputPlan{{Source: "chosen/clip.mkv", Video: "out/clip.mp4", Subtitle: "out/clip.srt"}},
			Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Name: "FFmpeg", Status: diagnostics.StatusOK}}},
		}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.startRun = func(config.Config) { started = true }

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !model.workflow.editing {
		t.Fatal("Enter on the selected input did not open its editor")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("chosen")})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("media")})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if model.workflow.cfg.InputRoot != "chosen media" {
		t.Fatalf("selected input = %q, want chosen media", model.workflow.cfg.InputRoot)
	}

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("preview key did not schedule preparation")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if model.workflow.phase != workflowReview || preparedConfig != "chosen media" {
		t.Fatalf("preview did not show the selected input plan: phase=%v input=%q", model.workflow.phase, preparedConfig)
	}
	if started || !strings.Contains(model.View(), "chosen/clip.mkv") || strings.Contains(model.View(), "clip.srt") {
		t.Fatalf("preview should display work without starting it: started=%v view=%q", started, model.View())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !started || model.workflow.phase != workflowRunning {
		t.Fatalf("confirmation did not start processing: started=%v phase=%v", started, model.workflow.phase)
	}
}

func TestWorkflowKeepsQueueClockCommandAliveAfterStarting(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), func(context.Context, config.Config) (app.RunPreview, error) {
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.startRun = func(config.Config) {}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, command = model.Update(tickMsg{})
	if _, ok := updated.(*Model); !ok || command == nil {
		t.Fatal("queue clock did not schedule its next tick after workflow confirmation")
	}
}

func TestWorkflowBlocksConfirmationWhenDiagnosticsFail(t *testing.T) {
	cfg := config.Defaults()
	started := false
	model := NewWorkflowModel(cfg, func(context.Context, config.Config) (app.RunPreview, error) {
		return app.RunPreview{
			Plans: []outputplan.OutputPlan{{Source: "clip.mkv", Video: "clip.mp4"}},
			Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{
				Name: "AMD hardware encode probe", Status: diagnostics.StatusFailed, Message: "device unavailable",
			}}},
		}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.startRun = func(config.Config) { started = true }

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if !strings.Contains(model.View(), "device unavailable") {
		t.Fatalf("preflight failure was not shown: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if started || model.workflow.phase != workflowReview {
		t.Fatalf("failed diagnostics must block start: started=%v phase=%v", started, model.workflow.phase)
	}
}

func TestWorkflowShowsConfigurationErrorsWithoutStartingPreparation(t *testing.T) {
	cfg := config.Defaults()
	prepared := false
	model := NewWorkflowModel(cfg, func(context.Context, config.Config) (app.RunPreview, error) {
		prepared = true
		return app.RunPreview{}, nil
	}, scheduler.NewDispatchControl(), func() {})
	for index := 0; index < 9; index++ {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(*Model)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	if command != nil {
		t.Fatal("invalid settings should be rejected before launching dependency checks")
	}
	if prepared || !strings.Contains(model.View(), "encode_workers") {
		t.Fatalf("invalid settings were not explained before preparation: prepared=%v view=%q", prepared, model.View())
	}
}

func TestWorkflowKeyboardNavigationKeepsSelectionVisibleAtCompactSize(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 42, Height: 9})
	model = updated.(*Model)
	for index := 0; index < len(workflowFields)-1; index++ {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(*Model)
	}
	view := model.View()
	if model.workflow.focused != len(workflowFields)-1 || !strings.Contains(view, "Whisper GPU") {
		t.Fatalf("compact settings view lost keyboard focus: focus=%d view=%q", model.workflow.focused, view)
	}
	if model.viewport.Width < 1 || model.viewport.Height < 1 {
		t.Fatalf("compact terminal produced invalid viewport: %dx%d", model.viewport.Width, model.viewport.Height)
	}
}
