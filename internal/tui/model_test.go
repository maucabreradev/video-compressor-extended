package tui

import (
	"context"
	"errors"
	"fmt"
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

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	if !model.workflow.editing {
		t.Fatal("e on quick setup did not open the input editor")
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

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(*Model)
	if !model.workflow.editing {
		t.Fatal("e should open the input editor on the quick setup screen")
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
