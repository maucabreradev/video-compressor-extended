package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

type workflowPhase uint8

const (
	workflowSettings workflowPhase = iota
	workflowPreparing
	workflowReview
	workflowRunning
)

type workflowState struct {
	ctx     context.Context
	prepare func(context.Context, config.Config) (app.RunPreview, error)
	cfg     config.Config
	phase   workflowPhase
	focused int
	editing bool
	editor  string
	preview app.RunPreview
	error   string
}

type workflowPreviewMsg struct {
	preview app.RunPreview
	err     error
}

type workflowField struct {
	name  string
	label string
}

var workflowFields = []workflowField{
	{name: "input_root", label: "Input file or directory"},
	{name: "output_dir", label: "Output directory"},
	{name: "video_codec", label: "Video codec (h264 / hevc)"},
	{name: "max_width", label: "Maximum width"},
	{name: "max_height", label: "Maximum height"},
	{name: "frame_rate", label: "Frame rate (0 preserves source)"},
	{name: "whisper_model_path", label: "Whisper model (blank disables subtitles)"},
	{name: "language", label: "Subtitle language"},
	{name: "resource_profile", label: "Resource profile (conservative / custom)"},
	{name: "encode_workers", label: "Encode workers"},
	{name: "subtitle_workers", label: "Subtitle workers"},
	{name: "whisper_threads", label: "Whisper CPU threads"},
	{name: "process_priority", label: "Process priority"},
	{name: "whisper_gpu", label: "Whisper GPU (true / false)"},
}

// NewWorkflowModel creates an interactive setup model. The preview callback
// must validate and inspect a run without starting media processing.
func NewWorkflowModel(
	cfg config.Config,
	prepare func(context.Context, config.Config) (app.RunPreview, error),
	control *scheduler.DispatchControl,
	cancel context.CancelFunc,
) *Model {
	model := NewModel(control, cancel)
	model.workflow = &workflowState{
		ctx:     context.Background(),
		prepare: prepare,
		cfg:     cfg,
		phase:   workflowSettings,
	}
	return model
}

func (model *Model) updateWorkflow(message tea.Msg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	switch message := message.(type) {
	case workflowPreviewMsg:
		workflow.error = ""
		if message.err != nil {
			workflow.phase = workflowSettings
			workflow.error = message.err.Error()
			return model, nil
		}
		workflow.preview = message.preview
		workflow.phase = workflowReview
		model.viewport.Width = max(1, model.width-2)
		model.viewport.Height = max(1, model.height-7)
		model.viewport.SetContent(workflowReviewContent(message.preview))
		return model, nil
	case tickMsg:
		return model, model.tick()
	case tea.WindowSizeMsg:
		model.width = max(1, message.Width)
		model.height = max(1, message.Height)
		model.viewport.Width = max(1, model.width-2)
		model.viewport.Height = max(1, model.height-7)
		if workflow.phase == workflowReview {
			model.viewport.SetContent(workflowReviewContent(workflow.preview))
		}
		return model, nil
	case externalCancelMsg:
		if model.cancel != nil {
			model.cancel()
		}
		return model, tea.Quit
	case tea.KeyMsg:
		return model.updateWorkflowKey(message)
	default:
		if workflow.phase == workflowReview {
			var command tea.Cmd
			model.viewport, command = model.viewport.Update(message)
			return model, command
		}
		return model, nil
	}
}

func (model *Model) updateWorkflowKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	if workflow.phase == workflowReview {
		switch key.Type {
		case tea.KeyEnter:
			return model, model.confirmWorkflow()
		case tea.KeyUp, tea.KeyDown:
			var command tea.Cmd
			model.viewport, command = model.viewport.Update(key)
			return model, command
		case tea.KeyEsc, tea.KeyCtrlC:
			return model, tea.Quit
		case tea.KeyRunes:
			switch strings.ToLower(string(key.Runes)) {
			case "b":
				workflow.phase = workflowSettings
				workflow.error = ""
			case "q":
				return model, tea.Quit
			}
		}
		return model, nil
	}
	if workflow.editing {
		switch key.Type {
		case tea.KeyEnter:
			if err := setWorkflowField(&workflow.cfg, workflowFields[workflow.focused].name, workflow.editor); err != nil {
				workflow.error = err.Error()
				return model, nil
			}
			workflow.editing = false
			workflow.editor = ""
			workflow.error = ""
			return model, nil
		case tea.KeyEsc:
			workflow.editing = false
			workflow.editor = ""
			workflow.error = ""
			return model, nil
		case tea.KeyBackspace, tea.KeyDelete:
			runes := []rune(workflow.editor)
			if len(runes) > 0 {
				workflow.editor = string(runes[:len(runes)-1])
			}
			return model, nil
		case tea.KeyCtrlU:
			workflow.editor = ""
			return model, nil
		case tea.KeySpace:
			workflow.editor += " "
			return model, nil
		case tea.KeyRunes:
			workflow.editor += string(key.Runes)
			return model, nil
		}
		return model, nil
	}

	if key.Type == tea.KeyCtrlC || key.Type == tea.KeyEsc {
		return model, tea.Quit
	}
	if key.Type == tea.KeyUp || (key.Type == tea.KeyRunes && string(key.Runes) == "k") {
		workflow.focused = (workflow.focused + len(workflowFields) - 1) % len(workflowFields)
		workflow.error = ""
		return model, nil
	}
	if key.Type == tea.KeyDown || key.Type == tea.KeyTab || (key.Type == tea.KeyRunes && string(key.Runes) == "j") {
		workflow.focused = (workflow.focused + 1) % len(workflowFields)
		workflow.error = ""
		return model, nil
	}
	if key.Type == tea.KeyEnter {
		workflow.editor = ""
		workflow.editing = true
		workflow.error = ""
		return model, nil
	}
	if key.Type != tea.KeyRunes {
		return model, nil
	}
	switch strings.ToLower(string(key.Runes)) {
	case "p":
		if err := workflow.cfg.Validate(); err != nil {
			workflow.error = fmt.Sprintf("Invalid settings: %v", err)
			return model, nil
		}
		if workflow.prepare == nil {
			workflow.error = "Run preview is unavailable"
			return model, nil
		}
		workflow.phase = workflowPreparing
		workflow.error = ""
		cfg := workflow.cfg
		ctx := workflow.ctx
		prepare := workflow.prepare
		return model, func() tea.Msg {
			preview, err := prepare(ctx, cfg)
			return workflowPreviewMsg{preview: preview, err: err}
		}
	case "q":
		return model, tea.Quit
	}
	return model, nil
}

func (model *Model) workflowView() string {
	workflow := model.workflow
	switch workflow.phase {
	case workflowPreparing:
		return strings.Join([]string{
			titleStyle.Render("VCX · PREPARING RUN"),
			"Discovering media and checking dependencies…",
			"q quit",
		}, "\n")
	case workflowReview:
		controls := "enter start · b back to settings · ↑/↓ scroll · q quit"
		if workflow.preview.Diagnostics.HasFailures() {
			controls = "Resolve the failed checks, then press b to edit settings and preview again · q quit"
		}
		return strings.Join([]string{
			titleStyle.Render("VCX · REVIEW RUN"),
			"Planned files and required dependencies",
			model.viewport.View(),
			errorStyle.Render(workflow.error),
			dimStyle.Render(controls),
		}, "\n")
	default:
		return model.workflowSettingsView()
	}
}

func (model *Model) workflowSettingsView() string {
	workflow := model.workflow
	maxVisible := max(1, model.height-7)
	start := max(0, workflow.focused-maxVisible/2)
	if start+maxVisible > len(workflowFields) {
		start = max(0, len(workflowFields)-maxVisible)
	}
	end := min(len(workflowFields), start+maxVisible)
	lines := []string{
		titleStyle.Render("VCX · RUN SETUP"),
		"Choose inputs and settings. Press enter to edit the selected value.",
	}
	for index := start; index < end; index++ {
		field := workflowFields[index]
		prefix := "  "
		if index == workflow.focused {
			prefix = "> "
		}
		value := workflowFieldValue(workflow.cfg, field.name)
		if index == workflow.focused && workflow.editing {
			if workflow.editor == "" {
				value = "type replacement (current: " + value + ")"
			} else {
				value = workflow.editor + "▏"
			}
		} else if value == "" {
			value = "(not set)"
		}
		line := fmt.Sprintf("%s%-43s %s", prefix, field.label, value)
		if model.width < 62 {
			line = prefix + field.label + ": " + value
		}
		lines = append(lines, lipgloss.NewStyle().Width(max(1, model.width-2)).Render(line))
	}
	if workflow.error != "" {
		lines = append(lines, errorStyle.Render(singleLine(workflow.error)))
	}
	controls := "↑/↓ or j/k select · enter edit · p preview · q quit"
	if workflow.editing {
		controls = "type value · enter save · esc cancel"
	}
	lines = append(lines, dimStyle.Render(controls))
	return strings.Join(lines, "\n")
}

func workflowReviewContent(preview app.RunPreview) string {
	lines := []string{fmt.Sprintf("Files to process: %d", len(preview.Plans))}
	for _, plan := range preview.Plans {
		lines = append(lines, "INPUT  "+plan.Source, "VIDEO  "+plan.Video)
		if preview.SubtitlesEnabled && plan.Subtitle != "" {
			lines = append(lines, "SRT    "+plan.Subtitle)
		}
		lines = append(lines, "")
	}
	lines = append(lines, "Dependency checks")
	for _, check := range preview.Diagnostics.Checks {
		line := fmt.Sprintf("%-24s %-15s %s", check.Name, check.Status, check.Message)
		if check.Status == diagnostics.StatusMissing || check.Status == diagnostics.StatusFailed || check.Status == diagnostics.StatusNotConfigured {
			line = errorStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func workflowFieldValue(cfg config.Config, name string) string {
	switch name {
	case "input_root":
		return cfg.InputRoot
	case "output_dir":
		return cfg.OutputDir
	case "video_codec":
		return cfg.VideoCodec
	case "max_width":
		return strconv.Itoa(cfg.MaxWidth)
	case "max_height":
		return strconv.Itoa(cfg.MaxHeight)
	case "frame_rate":
		return strconv.Itoa(cfg.FrameRate)
	case "whisper_model_path":
		return cfg.WhisperModelPath
	case "language":
		return cfg.Language
	case "resource_profile":
		return cfg.ResourceProfile
	case "encode_workers":
		return strconv.Itoa(cfg.EncodeWorkers)
	case "subtitle_workers":
		return strconv.Itoa(cfg.SubtitleWorkers)
	case "whisper_threads":
		return strconv.Itoa(cfg.WhisperThreads)
	case "process_priority":
		return cfg.ProcessPriority
	case "whisper_gpu":
		return strconv.FormatBool(cfg.WhisperGPU)
	default:
		return ""
	}
}

func setWorkflowField(cfg *config.Config, name, value string) error {
	value = strings.TrimSpace(value)
	switch name {
	case "input_root":
		cfg.InputRoot = value
	case "output_dir":
		cfg.OutputDir = value
	case "video_codec":
		cfg.VideoCodec = value
	case "max_width":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("maximum width must be an integer: %w", err)
		}
		cfg.MaxWidth = parsed
	case "max_height":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("maximum height must be an integer: %w", err)
		}
		cfg.MaxHeight = parsed
	case "frame_rate":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("frame rate must be an integer: %w", err)
		}
		cfg.FrameRate = parsed
	case "whisper_model_path":
		cfg.WhisperModelPath = value
	case "language":
		cfg.Language = value
	case "resource_profile":
		cfg.ResourceProfile = value
	case "encode_workers":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("encode workers must be an integer: %w", err)
		}
		cfg.EncodeWorkers = parsed
	case "subtitle_workers":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("subtitle workers must be an integer: %w", err)
		}
		cfg.SubtitleWorkers = parsed
	case "whisper_threads":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Whisper threads must be an integer: %w", err)
		}
		cfg.WhisperThreads = parsed
	case "process_priority":
		cfg.ProcessPriority = value
	case "whisper_gpu":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("Whisper GPU must be true or false: %w", err)
		}
		cfg.WhisperGPU = parsed
	default:
		return fmt.Errorf("unknown setting %q", name)
	}
	return nil
}

func (model *Model) confirmWorkflow() tea.Cmd {
	workflow := model.workflow
	if workflow.preview.Diagnostics.HasFailures() {
		workflow.error = "Dependency checks failed; processing cannot start."
		return nil
	}
	if model.startRun == nil {
		workflow.error = "Run handler is unavailable"
		return nil
	}
	workflow.phase = workflowRunning
	workflow.error = ""
	model.startRun(workflow.cfg)
	return nil
}
