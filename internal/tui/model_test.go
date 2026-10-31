package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
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

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	if !model.workflow.editing {
		t.Fatal("p on quick setup did not open manual path entry")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("chosen")})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("media")})
	model = updated.(*Model)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("saving the selected input should prepare the run")
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
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("quick setup did not schedule its run preview")
	}
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

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	for index, field := range workflowFields {
		if field.name == "encode_workers" {
			model.workflow.focused = index
			break
		}
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 42, Height: 9})
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

func TestWorkflowQuickPathReviewsDefaultsThenStartsAfterConfirmation(t *testing.T) {
	cfg := config.Defaults()
	cfg.InputRoot = `C:\media\to-process`
	var prepared config.Config
	started := false
	model := NewWorkflowModel(cfg, func(_ context.Context, got config.Config) (app.RunPreview, error) {
		prepared = got
		return app.RunPreview{
			Plans:              []outputplan.OutputPlan{{Source: `C:\media\to-process\clip.mkv`, Video: `C:\media\to-process\vcx_processed\clip.mp4`, Subtitle: `C:\media\to-process\vcx_processed\clip.srt`}},
			Diagnostics:        diagnostics.Report{Checks: []diagnostics.Check{{Name: "Whisper model", Status: diagnostics.StatusOK, Message: "ready"}}},
			CompressionEnabled: true, SubtitlesEnabled: true,
			EncoderBackend: ffmpeg.BackendAMF, EncoderName: "h264_amf",
		}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.startRun = func(config.Config) { started = true }

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("Enter on quick setup should prepare the run for review")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if model.workflow.phase != workflowReview {
		t.Fatalf("phase = %v, want review", model.workflow.phase)
	}
	if prepared.InputRoot != cfg.InputRoot || !prepared.CompressionEnabled || !prepared.SubtitlesEnabled ||
		prepared.VideoCodec != "h264" || prepared.MaxWidth != 1280 || prepared.MaxHeight != 720 ||
		prepared.FrameRate != 0 || prepared.ResourceProfile != config.ResourceProfileConservative ||
		prepared.WhisperModelPath != "" {
		t.Fatalf("quick path did not retain plug-and-play defaults: %+v", prepared)
	}
	view := model.View()
	for _, want := range []string{"clip.mkv", "clip.mp4", "clip.srt", "h264_amf", "1280x720", "source frame rate", "subtitles"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(want)) {
			t.Errorf("quick review is missing %q: %q", want, view)
		}
	}
	if started {
		t.Fatal("review must not start processing before confirmation")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !started || model.workflow.phase != workflowRunning {
		t.Fatalf("confirmation did not start the run: started=%v phase=%v", started, model.workflow.phase)
	}
}

func TestWorkflowQuickSetupCanEditInputBeforePreparing(t *testing.T) {
	cfg := config.Defaults()
	cfg.InputRoot = `C:\media\old`
	preparedInput := ""
	model := NewWorkflowModel(cfg, func(_ context.Context, got config.Config) (app.RunPreview, error) {
		preparedInput = got.InputRoot
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	if !model.workflow.editing {
		t.Fatal("p should open manual path entry on the quick setup screen")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(`D:\New Media`)})
	model = updated.(*Model)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("Enter after editing the selected input should prepare the run")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if preparedInput != `D:\New Media` || model.workflow.phase != workflowReview {
		t.Fatalf("edited input was not reviewed: input=%q phase=%v", preparedInput, model.workflow.phase)
	}
}

func TestWorkflowSoftwareFallbackRequiresSeparateExplicitConfirmation(t *testing.T) {
	cfg := config.Defaults()
	cfg.AllowSoftwareEncoding = true
	started := false
	model := NewWorkflowModel(cfg, func(context.Context, config.Config) (app.RunPreview, error) {
		return app.RunPreview{
			Diagnostics:                          diagnostics.Report{Checks: []diagnostics.Check{{Name: "software H.264 probe", Status: diagnostics.StatusOK}}},
			SoftwareFallbackAvailable:            true,
			SoftwareFallbackConfirmationRequired: false,
			SoftwareFallbackWarning:              "CPU encoding is slower and uses more CPU.",
			EncoderBackend:                       ffmpeg.BackendSoftware, EncoderName: "libx264",
		}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.startRun = func(got config.Config) {
		started = true
		if !got.AllowSoftwareEncoding {
			t.Error("explicit software consent was not applied to the run configuration")
		}
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("Enter on quick setup should prepare the run for review")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if !strings.Contains(model.View(), "slower") || !strings.Contains(model.View(), "CPU") {
		t.Fatalf("review did not disclose software fallback trade-off: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if started {
		t.Fatal("review confirmation alone must not authorize software encoding")
	}
	if !strings.Contains(model.View(), "confirm") {
		t.Fatalf("software fallback confirmation prompt is missing: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(*Model)
	if !started || model.workflow.phase != workflowRunning {
		t.Fatalf("explicit fallback confirmation did not start the run: started=%v phase=%v", started, model.workflow.phase)
	}
}

func TestWorkflowAdvancedSettingsAreOptional(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "ADVANCED SETTINGS") {
		t.Fatalf("advanced settings were not available on demand: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "SELECT MEDIA") {
		t.Fatalf("back from advanced settings did not return to quick setup: %q", model.View())
	}
}

func TestWorkflowQuickSetupRemainsUsableAfterTerminalResize(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	for _, size := range [][2]int{{38, 8}, {120, 32}} {
		updated, _ := model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		model = updated.(*Model)
		view := model.View()
		if model.width != size[0] || model.height != size[1] ||
			!strings.Contains(view, "SELECT MEDIA") || !strings.Contains(view, "enter review") {
			t.Fatalf("quick setup did not remain usable at %dx%d: %q", size[0], size[1], view)
		}
	}
}

func TestWorkflowQuickSetupUsesResponsiveSectionsWithoutOverflow(t *testing.T) {
	for _, size := range [][2]int{{38, 8}, {40, 12}, {80, 24}, {120, 32}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
			updated, _ := model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			model = updated.(*Model)

			lines := strings.Split(model.View(), "\n")
			for _, want := range []string{"MEDIA INPUT", "RUN PROFILE", "enter review"} {
				if !strings.Contains(model.View(), want) {
					t.Errorf("responsive selection screen is missing %q: %q", want, model.View())
				}
			}
			if len(lines) > size[1] {
				t.Errorf("selection screen has %d rows for a %d-row terminal", len(lines), size[1])
			}
			for index, line := range lines {
				if width := lipgloss.Width(line); width > size[0] {
					t.Errorf("line %d is %d columns wide for a %d-column terminal: %q", index+1, width, size[0], line)
				}
			}
			if size[1] <= 12 {
				model.workflow.error = "preview failed"
				model.workflow.preferenceError = "preferences unavailable"
				if view := model.View(); !strings.Contains(view, "enter review") {
					t.Errorf("compact error state hid the primary control: %q", view)
				}
			}
		})
	}
}

func TestWorkflowEditInputOpensFilteredMediaBrowser(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Media é with spaces")
	if err := os.MkdirAll(filepath.Join(root, "clips"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sample é.mp4", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	model.workflow.cfg.InputRoot = root
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("opening the media browser did not start directory loading")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)

	view := model.View()
	for _, want := range []string{"BROWSE MEDIA", "clips", "sample é.mp4"} {
		if !strings.Contains(view, want) {
			t.Errorf("media browser is missing %q: %q", want, view)
		}
	}
	if strings.Contains(view, "notes.txt") {
		t.Fatalf("media browser showed an unsupported file: %q", view)
	}
	cursor := model.workflow.pickerCursor
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	model = updated.(*Model)
	if model.workflow.pickerCursor != cursor || !strings.Contains(model.View(), "BROWSE MEDIA") {
		t.Fatalf("terminal resize lost the active media-browser position: cursor=%d view=%q", model.workflow.pickerCursor, model.View())
	}

	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("entering a folder did not start directory loading")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("going to the parent folder did not start directory loading")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if model.workflow.phase != workflowQuickSetup || model.workflow.cfg.InputRoot != filepath.Join(root, "sample é.mp4") {
		t.Fatalf("choosing a media file did not return its path to setup: phase=%v input=%q", model.workflow.phase, model.workflow.cfg.InputRoot)
	}
}

func TestWorkflowMediaBrowserSelectsCurrentFolderRecursively(t *testing.T) {
	root := t.TempDir()
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	model.workflow.cfg.InputRoot = root
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(*Model)
	if model.workflow.phase != workflowQuickSetup || model.workflow.cfg.InputRoot != root {
		t.Fatalf("choosing the current folder did not return it for recursive discovery: phase=%v input=%q", model.workflow.phase, model.workflow.cfg.InputRoot)
	}
}

func TestWorkflowMediaBrowserMarksAndUnmarksFiles(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(root, "sample é.mp4")
	if err := os.WriteFile(mediaPath, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := openWorkflowMediaPicker(t, root)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 1 || model.workflow.cfg.InputPaths[0] != "sample é.mp4" {
		t.Fatalf("space did not mark the highlighted media file: root=%q paths=%q", model.workflow.cfg.InputRoot, model.workflow.cfg.InputPaths)
	}
	if model.workflow.cfg.InputRoot != root || !strings.Contains(model.View(), "Selected: 1") {
		t.Fatalf("selection root or count was not shown: root=%q view=%q", model.workflow.cfg.InputRoot, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 0 {
		t.Fatalf("space did not unmark the highlighted media file: %q", model.workflow.cfg.InputPaths)
	}
}

func TestWorkflowMediaBrowserAddsFoldersRecursivelyWithoutOverlappingSelections(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "nested")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first.mp4", "second.mkv"} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(folder, "deeper"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "deeper", "third.webm"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "notes.txt"), []byte("not media"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := openWorkflowMediaPicker(t, root)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("opening the nested folder did not start directory loading")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 1 || model.workflow.cfg.InputPaths[0] != filepath.Join("nested", "first.mp4") {
		t.Fatalf("marking a file did not preserve its path relative to the common root: %q", model.workflow.cfg.InputPaths)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("returning to the root did not start directory loading")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("recursive folder add did not start")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 3 {
		t.Fatalf("recursive folder add did not include nested supported videos without duplicating the marked file: %q", model.workflow.cfg.InputPaths)
	}
	wantPaths := map[string]bool{
		filepath.Join("nested", "first.mp4"):            true,
		filepath.Join("nested", "second.mkv"):           true,
		filepath.Join("nested", "deeper", "third.webm"): true,
	}
	for _, path := range model.workflow.cfg.InputPaths {
		if !wantPaths[path] {
			t.Errorf("recursive selection included an unexpected or duplicate path %q in %q", path, model.workflow.cfg.InputPaths)
		}
		delete(wantPaths, path)
	}
	if len(wantPaths) != 0 {
		t.Errorf("recursive selection omitted expected paths: %v", wantPaths)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 3 {
		t.Fatalf("adding the same folder again was not idempotent: %q", model.workflow.cfg.InputPaths)
	}
}

func TestWorkflowSelectionReviewAllowsRemovalAndRejectsEmptySelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first.mp4", "second.mkv"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	previewCalls := 0
	model := NewWorkflowModel(config.Defaults(), func(context.Context, config.Config) (app.RunPreview, error) {
		previewCalls++
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.workflow.cfg.InputRoot = root
	model.startRun = func(config.Config) { t.Fatal("an empty selection started processing") }
	openPicker := func() {
		updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		model = updated.(*Model)
		updated, _ = model.Update(command())
		model = updated.(*Model)
	}
	openPicker()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "SELECTED MEDIA") || !strings.Contains(model.View(), "Selected items: 2") {
		t.Fatalf("selection review did not show selected items and count: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 1 {
		t.Fatalf("remove control did not remove one selected item: %q", model.workflow.cfg.InputPaths)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "SELECTED MEDIA") || previewCalls != 0 || strings.TrimSpace(model.workflow.error) == "" {
		t.Fatalf("empty selection was not rejected before readiness checks: phase=%v previews=%d error=%q", model.workflow.phase, previewCalls, model.workflow.error)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(*Model)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command != nil || previewCalls != 0 || !strings.Contains(model.workflow.error, "Select at least one") {
		t.Fatalf("quick setup started or prepared an empty multi-selection: command=%v previews=%d error=%q", command != nil, previewCalls, model.workflow.error)
	}
}

func TestWorkflowSelectionStaysReadableAndOperableAfterResize(t *testing.T) {
	root := t.TempDir()
	name := "café_long-selection-name_0123456789.mp4"
	if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := openWorkflowMediaPicker(t, root)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 38, Height: 10})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "Selected: 1") || !strings.Contains(model.View(), "café") {
		t.Fatalf("picker selection was not visible after terminal resize: %q", model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(*Model)
	for _, size := range [][2]int{{38, 8}, {120, 32}} {
		updated, _ = model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		model = updated.(*Model)
		view := model.View()
		if !strings.Contains(view, "SELECTED MEDIA") || !strings.Contains(view, "Selected items: 1") || !strings.Contains(view, "café") {
			t.Fatalf("selection was not legible at %dx%d: %q", size[0], size[1], view)
		}
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(*Model)
	if len(model.workflow.cfg.InputPaths) != 0 {
		t.Fatalf("selection could not be removed after resizing: %q", model.workflow.cfg.InputPaths)
	}
}

func TestWorkflowSelectionRejectsPathsOutsideCommonRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	outside := filepath.Join(filepath.Dir(root), "media-neighbor", "clip.mp4")
	if _, err := relativeMediaSelectionPath(root, outside); err == nil {
		t.Fatalf("path outside the selected common root was accepted: root=%q path=%q", root, outside)
	}
}

func TestWorkflowSelectionReviewPrecedesReadinessChecks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "clip.mp4"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	previewCalls := 0
	model := NewWorkflowModel(config.Defaults(), func(_ context.Context, cfg config.Config) (app.RunPreview, error) {
		previewCalls++
		if cfg.InputRoot != root || len(cfg.InputPaths) != 1 || cfg.InputPaths[0] != "clip.mp4" {
			t.Fatalf("preview did not receive the reviewed selection: %+v", cfg)
		}
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.workflow.cfg.InputRoot = root
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if model.workflow.phase != workflowQuickSetup || previewCalls != 0 {
		t.Fatalf("reviewing the selection started readiness checks instead of returning to setup: phase=%v previews=%d", model.workflow.phase, previewCalls)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if previewCalls != 1 || model.workflow.phase != workflowReview {
		t.Fatalf("confirmed selection did not transition into readiness review: phase=%v previews=%d", model.workflow.phase, previewCalls)
	}
}

func TestWorkflowConfiguredSelectionMustBeReviewedBeforeReadinessChecks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "clip.mp4"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.InputRoot = root
	cfg.InputPaths = []string{"clip.mp4"}
	previewCalls := 0
	model := NewWorkflowModel(cfg, func(context.Context, config.Config) (app.RunPreview, error) {
		previewCalls++
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command != nil || previewCalls != 0 || !strings.Contains(model.View(), "SELECTED MEDIA") {
		t.Fatalf("configured selection reached readiness before selection review: command=%v previews=%d view=%q", command != nil, previewCalls, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if previewCalls != 0 || model.workflow.phase != workflowQuickSetup {
		t.Fatalf("confirming the selection did not return to quick setup: phase=%v previews=%d", model.workflow.phase, previewCalls)
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("reviewed configured selection did not start readiness checks")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if previewCalls != 1 || model.workflow.phase != workflowReview {
		t.Fatalf("reviewed configured selection did not reach run review: phase=%v previews=%d", model.workflow.phase, previewCalls)
	}
}

func openWorkflowMediaPicker(t *testing.T, root string) *Model {
	t.Helper()
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	model.workflow.cfg.InputRoot = root
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("opening the media browser did not start directory loading")
	}
	updated, _ = model.Update(command())
	return updated.(*Model)
}

func TestWorkflowMediaBrowserShowsEmptyAndMissingFolderStates(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: t.TempDir(), want: "No supported media or subfolders"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), want: "read media folder"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
			model.workflow.cfg.InputRoot = test.path
			updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			model = updated.(*Model)
			updated, _ = model.Update(command())
			model = updated.(*Model)
			if view := model.View(); !strings.Contains(view, test.want) {
				t.Fatalf("media browser did not explain %s state: %q", test.name, view)
			}
		})
	}
}

func TestWorkflowShowsDependencyPreparationFailureAndCanRetry(t *testing.T) {
	calls := 0
	model := NewWorkflowModel(config.Defaults(), func(context.Context, config.Config) (app.RunPreview, error) {
		calls++
		if calls == 1 {
			return app.RunPreview{}, errors.New("download refused: missing Whisper runtime")
		}
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if !strings.Contains(model.View(), "download refused") || model.workflow.phase != workflowQuickSetup {
		t.Fatalf("dependency setup failure was not actionable: phase=%v view=%q", model.workflow.phase, model.View())
	}
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if calls != 2 || model.workflow.phase != workflowReview {
		t.Fatalf("workflow could not retry dependency setup: calls=%d phase=%v", calls, model.workflow.phase)
	}
}

func TestWorkflowShowsRuntimeDownloadConsentAndProgress(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	model.workflow.phase = workflowPreparing
	reply := make(chan bool, 1)
	asset := runtimeassets.Asset{
		ID: "whisper-model", Version: "1.2.3", Target: "windows-amd64",
		SourceURL: "https://example.com/project", DownloadURL: "https://example.com/model.bin",
		License: "MIT", SizeBytes: 2 << 20,
	}
	updated, _ := model.Update(workflowAssetConsentMsg{asset: asset, reply: reply})
	model = updated.(*Model)
	view := model.View()
	for _, want := range []string{"whisper-model", "1.2.3", "windows-amd64", "https://example.com/project", "MIT", "2.0 MiB", "y allow", "n decline"} {
		if !strings.Contains(view, want) {
			t.Errorf("dependency consent prompt is missing %q: %q", want, view)
		}
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(*Model)
	if accepted := <-reply; !accepted || model.workflow.phase != workflowPreparing {
		t.Fatalf("consent answer was not forwarded: accepted=%v phase=%v", accepted, model.workflow.phase)
	}
	updated, _ = model.Update(workflowAssetProgressMsg{progress: runtimeassets.DownloadProgress{
		AssetID: "whisper-model", ReceivedBytes: 1 << 20, TotalBytes: 2 << 20,
	}})
	model = updated.(*Model)
	if !strings.Contains(model.View(), "Downloading whisper-model: 1.0 MiB / 2.0 MiB") {
		t.Fatalf("dependency download progress is not visible: %q", model.View())
	}
}

func TestWorkflowReviewRemainsReadableAndScrollableAtCompactAndWideSizes(t *testing.T) {
	preview := app.RunPreview{
		Plans: make([]outputplan.OutputPlan, 12),
		Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{
			{Name: "AMD hardware encode probe", Status: diagnostics.StatusOK, Message: "verified and ready"},
			{Name: "Whisper model", Status: diagnostics.StatusOK, Message: "verified and ready"},
		}},
		CompressionEnabled: true, SubtitlesEnabled: true,
		EncoderBackend: ffmpeg.BackendAMF, EncoderName: "h264_amf",
	}
	for index := range preview.Plans {
		preview.Plans[index] = outputplan.OutputPlan{
			Source:   fmt.Sprintf("C:/media/long source directory/clip-%02d-世界.mkv", index),
			Video:    fmt.Sprintf("C:/media/processed/nested/clip-%02d.mp4", index),
			Subtitle: fmt.Sprintf("C:/media/processed/nested/clip-%02d.srt", index),
		}
	}
	for _, size := range [][2]int{{38, 8}, {120, 30}} {
		model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
		updated, _ := model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		model = updated.(*Model)
		updated, _ = model.Update(workflowPreviewMsg{preview: preview})
		model = updated.(*Model)
		view := model.View()
		assertResponsiveScreen(t, view, size[0], size[1])
		for _, want := range []string{"Files to process: 12", "Stages:", "h264_amf", "Resources:", "enter continue"} {
			if !strings.Contains(view, want) {
				t.Errorf("review at %dx%d is missing %q: %q", size[0], size[1], want, view)
			}
		}
		for index := 0; index < 300; index++ {
			updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
			model = updated.(*Model)
		}
		if !strings.Contains(model.View(), "Dependency checks") || !strings.Contains(model.View(), "Whisper model") {
			t.Errorf("review plans or readiness checks were not reachable by scrolling at %dx%d: %q", size[0], size[1], model.View())
		}
	}
}

func TestWorkflowPreparationConsentAndFallbackScreensFitCompactTerminals(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 38, Height: 8})
	model = updated.(*Model)
	model.workflow.phase = workflowPreparing
	model.workflow.progress = &runtimeassets.DownloadProgress{AssetID: "whisper-model", ReceivedBytes: 1 << 20, TotalBytes: 2 << 20}
	assertResponsiveScreen(t, model.View(), 38, 8)
	for _, want := range []string{"PREPARING RUN", "whisper-model", "1.0 MiB / 2.0 MiB", "q quit"} {
		if !strings.Contains(model.View(), want) {
			t.Errorf("compact preparation screen is missing %q: %q", want, model.View())
		}
	}
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model = updated.(*Model)
	assertResponsiveScreen(t, model.View(), 120, 30)
	if !strings.Contains(model.View(), "1.0 MiB / 2.0 MiB") {
		t.Fatalf("preparation progress disappeared after resizing: %q", model.View())
	}
	quitModel := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
	quitModel.workflow.phase = workflowPreparing
	updated, command := quitModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	quitModel = updated.(*Model)
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatal("keyboard quit stopped working during dependency preparation")
	}

	reply := make(chan bool, 1)
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 38, Height: 8})
	model = updated.(*Model)
	asset := runtimeassets.Asset{
		ID: "whisper-model", Version: "1.2.3", Target: "windows-amd64",
		SourceURL:   "https://example.com/a/very/long/project/source/page",
		DownloadURL: "https://example.com/a/very/long/path/default-model.bin",
		License:     "MIT", SizeBytes: 2 << 20,
	}
	updated, _ = model.Update(workflowAssetConsentMsg{asset: asset, reply: reply})
	model = updated.(*Model)
	view := model.View()
	assertResponsiveScreen(t, view, 38, 8)
	if !strings.Contains(view, "y allow") || !strings.Contains(view, "n decline") {
		t.Fatalf("consent actions are not visible at compact size: %q", view)
	}
	var scrolled strings.Builder
	scrolled.WriteString(view)
	for index := 0; index < 40; index++ {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(*Model)
		scrolled.WriteString(model.View())
	}
	visibleDetails := strings.NewReplacer("\n", "", " ", "").Replace(scrolled.String())
	for _, want := range []string{"1.2.3", "windows-amd64", "https://example.com/a/very/long/project/source/page", "https://example.com/a/very/long/path/default-model.bin", "MIT", "2.0 MiB"} {
		if !strings.Contains(visibleDetails, strings.ReplaceAll(want, " ", "")) {
			t.Errorf("scrollable consent details are missing %q", want)
		}
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(*Model)
	if accepted := <-reply; !accepted || model.workflow.phase != workflowPreparing {
		t.Fatalf("consent action stopped working after scrolling: accepted=%v phase=%v", accepted, model.workflow.phase)
	}

	model.workflow.phase = workflowReview
	model.workflow.preview.SoftwareFallbackWarning = "Software H.264 uses the CPU and can be substantially slower on this system; confirm before continuing."
	model.workflow.preview.SoftwareFallbackConfirmationRequired = true
	model.setWorkflowViewportContent(workflowReviewContent(model.workflow.preview, model.workflow.cfg), true)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	view = model.View()
	assertResponsiveScreen(t, view, 38, 8)
	if !strings.Contains(view, "y confirm") || !strings.Contains(view, "n review") {
		t.Fatalf("fallback confirmation actions are not visible at compact size: %q", view)
	}
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model = updated.(*Model)
	view = model.View()
	assertResponsiveScreen(t, view, 120, 30)
	if !strings.Contains(view, "y confirm") || !strings.Contains(view, "n review") || !strings.Contains(view, "substantially slower") {
		t.Fatalf("wide fallback confirmation lost its warning or actions: %q", view)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = updated.(*Model)
	if model.workflow.phase != workflowReview {
		t.Fatalf("fallback decline did not return to run review: %v", model.workflow.phase)
	}
}

func TestRecoveryScreenShowsJobStateAllowsNewRunAndResumesOnlyReadySelection(t *testing.T) {
	cfg := config.Defaults()
	cfg.InputRoot = "selected media"
	model := NewWorkflowModel(cfg, nil, scheduler.NewDispatchControl(), func() {})
	model.width, model.height = 48, 12
	model.setRecoveryJobs([]app.RecoveryJob{
		{ID: "ready-job-id", SourceSummary: "clip.mkv", OutputSummary: "clip.mp4", CompressionState: "staged 1", SubtitleState: "pending 1", Status: app.RecoveryJobReady},
		{ID: "active-job-id", SourceSummary: "active.mkv", Status: app.RecoveryJobActive, Detail: "Active in another VCX process."},
		{ID: "stale-job-id", SourceSummary: "changed.mkv", Status: app.RecoveryJobStale, Detail: "Source changed."},
	}, nil)
	view := model.View()
	for _, want := range []string{"RECOVER SAVED JOB", "ready-job-id", "active", "n new run"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(want)) {
			t.Errorf("recovery screen is missing %q: %q", want, view)
		}
	}
	assertResponsiveScreen(t, view, 48, 12)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*Model)
	if !strings.Contains(strings.ToLower(model.View()), "stale") {
		t.Fatalf("stale job state was not reachable by selection: %q", model.View())
	}

	started := false
	model = NewWorkflowModel(cfg, nil, scheduler.NewDispatchControl(), func() {})
	model.setRecoveryJobs([]app.RecoveryJob{{ID: "ready-job-id", Status: app.RecoveryJobReady}}, nil)
	model.startResume = func(jobID string) { started = jobID == "ready-job-id" }
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !started || model.workflow.phase != workflowPreparing {
		t.Fatalf("ready job was not resumed: started=%v phase=%v", started, model.workflow.phase)
	}

	newRun := NewWorkflowModel(cfg, nil, scheduler.NewDispatchControl(), func() {})
	newRun.setRecoveryJobs([]app.RecoveryJob{{ID: "active-job-id", Status: app.RecoveryJobActive}}, nil)
	updated, _ = newRun.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	newRun = updated.(*Model)
	if newRun.workflow.phase != workflowQuickSetup || newRun.workflow.cfg.InputRoot != cfg.InputRoot {
		t.Fatalf("new-run choice changed existing media selection: phase=%v config=%+v", newRun.workflow.phase, newRun.workflow.cfg)
	}
}

func TestRecoveryScreenBlocksActiveJobsAndFitsCompactAndWideTerminals(t *testing.T) {
	for _, size := range [][2]int{{38, 8}, {120, 30}} {
		model := NewWorkflowModel(config.Defaults(), nil, scheduler.NewDispatchControl(), func() {})
		model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		model.setRecoveryJobs([]app.RecoveryJob{{
			ID: "active-job-id", SourceSummary: "clip.mkv", OutputSummary: "clip.mp4",
			CompressionState: "running 1", Status: app.RecoveryJobActive,
			Detail: "This job is active in another VCX process.",
		}}, nil)
		resumed := false
		model.startResume = func(string) { resumed = true }
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(*Model)
		if resumed || model.workflow.phase != workflowRecovery || !strings.Contains(model.workflow.error, "active") {
			t.Fatalf("active job was resumable at %dx%d: resumed=%v phase=%v error=%q", size[0], size[1], resumed, model.workflow.phase, model.workflow.error)
		}
		assertResponsiveScreen(t, model.View(), size[0], size[1])
	}
}

func TestBatchScreenFitsCompactAndWideSizesWithProgressAndErrors(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := NewModel(scheduler.NewDispatchControl(), func() {})
	for _, size := range [][2]int{{38, 8}, {120, 30}} {
		updated, _ := model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		model = updated.(*Model)
		for _, event := range []app.RunEvent{
			{Kind: app.RunStageStateChanged, Source: "C:/media/very long/世界 clip.mkv", Stage: scheduler.Compression, State: scheduler.Running},
			{Kind: app.RunStageStateChanged, Source: "C:/media/very long/世界 clip.mkv", Stage: scheduler.Transcription, State: scheduler.Queued},
			{Kind: app.RunProgress, Source: "C:/media/very long/世界 clip.mkv", Stage: scheduler.Compression, DurationKnown: true, Progress: process.FFmpegProgress{Frame: 120, Percent: 42.5}},
			{Kind: app.RunStageStateChanged, Source: "C:/media/very long/世界 clip.mkv", Stage: scheduler.Compression, State: scheduler.Failed, Err: errors.New("encoder device unavailable; check the AMD driver and selected output path")},
		} {
			updated, _ = model.Update(runEventMsg{event: event})
			model = updated.(*Model)
		}
		view := model.View()
		assertResponsiveScreen(t, view, size[0], size[1])
		if strings.Contains(view, "\x1b[") {
			t.Errorf("NO_COLOR view contains terminal color escapes at %dx%d: %q", size[0], size[1], view)
		}
		for _, want := range []string{"BATCH QUEUE", "elapsed", "Encode", "Subtitles", "42.5%", "encoder device unavailable", "p pause", "c cancel"} {
			if !strings.Contains(view, want) {
				t.Errorf("batch screen at %dx%d is missing %q: %q", size[0], size[1], want, view)
			}
		}
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	if !model.paused || !model.control.IsPaused() {
		t.Fatal("responsive batch controls stopped pausing queued work")
	}
}

func assertResponsiveScreen(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Errorf("screen rendered %d rows for a %d-row terminal: %q", len(lines), height, view)
	}
	for index, line := range lines {
		if lineWidth := lipgloss.Width(line); lineWidth > width {
			t.Errorf("screen line %d is %d columns wide for a %d-column terminal: %q", index+1, lineWidth, width, line)
		}
	}
}

func TestTruncateDisplayPreservesANSIAndUnicodeCellWidth(t *testing.T) {
	styled := "\x1b[31m世界 and a long styled label\x1b[0m"
	got := truncateDisplay(styled, 12)
	if width := lipgloss.Width(got); width > 12 {
		t.Fatalf("truncated styled text is %d cells wide, want at most 12: %q", width, got)
	}
	if !strings.HasPrefix(got, "\x1b[31m") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("truncation broke the ANSI style boundary: %q", got)
	}
}

func TestWorkflowResetRestoresOriginalSettingsAndConfirmationPersistsPreferences(t *testing.T) {
	base := config.Defaults()
	base.InputRoot = `C:\media\source`
	cfg := base
	cfg.VideoCodec = "hevc"
	resetCalled := false
	saved := false
	model := NewWorkflowModel(cfg, func(context.Context, config.Config) (app.RunPreview, error) {
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	model.workflow.baseConfig = base
	model.startRun = func(config.Config) {}
	model.workflow.resetPreferences = func() error { resetCalled = true; return nil }
	model.workflow.savePreferences = func(got config.Config) error {
		saved = true
		if got.InputRoot != base.InputRoot || got.VideoCodec != cfg.VideoCodec {
			t.Errorf("preferences saved from unexpected configuration: %+v", got)
		}
		return nil
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(*Model)
	if !resetCalled || model.workflow.cfg.VideoCodec != base.VideoCodec || !strings.Contains(model.View(), "reset") {
		t.Fatalf("reset did not restore the initial settings: called=%v cfg=%+v view=%q", resetCalled, model.workflow.cfg, model.View())
	}
	// Restore the edited choice to verify that a confirmed run saves the settings
	// actually used for that run, rather than its initial snapshot.
	model.workflow.cfg.VideoCodec = "hevc"
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("quick setup did not schedule review")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !saved {
		t.Fatal("confirmed run did not persist non-sensitive preferences")
	}
}

func TestWorkflowPreferenceWriteFailureDoesNotPreventConfirmedProcessing(t *testing.T) {
	model := NewWorkflowModel(config.Defaults(), func(context.Context, config.Config) (app.RunPreview, error) {
		return app.RunPreview{Diagnostics: diagnostics.Report{Checks: []diagnostics.Check{{Status: diagnostics.StatusOK}}}}, nil
	}, scheduler.NewDispatchControl(), func() {})
	started := false
	saveCalls := 0
	model.startRun = func(config.Config) { started = true }
	model.workflow.savePreferences = func(config.Config) error {
		saveCalls++
		return errors.New("read-only preferences directory")
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	updated, _ = model.Update(command())
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if started || !strings.Contains(model.View(), "run without saving") {
		t.Fatalf("preference write failure was not explained before continuing: started=%v view=%q", started, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if !started || saveCalls != 1 || model.workflow.phase != workflowRunning {
		t.Fatalf("user could not continue after preference write failure: started=%v saves=%d phase=%v", started, saveCalls, model.workflow.phase)
	}
}
