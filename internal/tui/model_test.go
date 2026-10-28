package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
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
