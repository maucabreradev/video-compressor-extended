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
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

type workflowPhase uint8

const (
	workflowQuickSetup workflowPhase = iota
	workflowAdvanced
	workflowPreparing
	workflowConsent
	workflowReview
	workflowSoftwareConfirmation
	workflowRunning
)

type workflowState struct {
	ctx                  context.Context
	prepareWithCallbacks func(context.Context, config.Config, app.RuntimeSetupCallbacks) (app.RunPreview, error)
	cfg                  config.Config
	baseConfig           config.Config
	phase                workflowPhase
	returnToAdvanced     bool
	focused              int
	editing              bool
	editor               string
	preview              app.RunPreview
	error                string
	pendingAsset         runtimeassets.Asset
	consentReply         chan bool
	progress             *runtimeassets.DownloadProgress
	resetPreferences     func() error
	savePreferences      func(config.Config) error
	preferenceError      string
	skipPreferenceSave   bool
}

type workflowPreviewMsg struct {
	preview app.RunPreview
	err     error
}

type workflowAssetConsentMsg struct {
	asset runtimeassets.Asset
	reply chan bool
}

type workflowAssetProgressMsg struct {
	progress runtimeassets.DownloadProgress
}

type workflowField struct {
	name  string
	label string
}

var workflowFields = []workflowField{
	{name: "input_root", label: "Input file or directory"},
	{name: "output_dir", label: "Output directory"},
	{name: "compression_enabled", label: "Enable compression (true / false)"},
	{name: "subtitles_enabled", label: "Enable subtitles (true / false)"},
	{name: "video_codec", label: "Video codec (h264 / hevc)"},
	{name: "max_width", label: "Maximum width"},
	{name: "max_height", label: "Maximum height"},
	{name: "frame_rate", label: "Frame rate (0 preserves source)"},
	{name: "whisper_model_path", label: "Whisper model path (blank uses managed default)"},
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
		ctx:        context.Background(),
		cfg:        cfg,
		baseConfig: cfg,
		phase:      workflowQuickSetup,
	}
	model.workflow.prepareWithCallbacks = func(ctx context.Context, cfg config.Config, _ app.RuntimeSetupCallbacks) (app.RunPreview, error) {
		if prepare == nil {
			return app.RunPreview{}, fmt.Errorf("run preview is unavailable")
		}
		return prepare(ctx, cfg)
	}
	return model
}

// SetWorkflowPreviewWithCallbacks configures dependency setup reporting for the
// interactive preview. It must be called before the Bubble Tea program starts.
func (model *Model) SetWorkflowPreviewWithCallbacks(
	prepare func(context.Context, config.Config, app.RuntimeSetupCallbacks) (app.RunPreview, error),
) {
	if model.workflow != nil && prepare != nil {
		model.workflow.prepareWithCallbacks = prepare
	}
}

func (model *Model) updateWorkflow(message tea.Msg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	switch message := message.(type) {
	case workflowPreviewMsg:
		workflow.error = ""
		workflow.progress = nil
		workflow.consentReply = nil
		if message.err != nil {
			if workflow.returnToAdvanced {
				workflow.phase = workflowAdvanced
			} else {
				workflow.phase = workflowQuickSetup
			}
			workflow.error = message.err.Error()
			return model, nil
		}
		workflow.preview = message.preview
		if workflow.preview.SoftwareFallbackAvailable {
			workflow.preview.SoftwareFallbackConfirmationRequired = true
		}
		workflow.phase = workflowReview
		model.viewport.Width = max(1, model.width-2)
		model.viewport.Height = max(1, model.height-7)
		model.viewport.SetContent(workflowReviewContent(message.preview, workflow.cfg))
		return model, nil
	case workflowAssetConsentMsg:
		if workflow.phase != workflowPreparing {
			message.reply <- false
			return model, nil
		}
		workflow.pendingAsset = message.asset
		workflow.consentReply = message.reply
		workflow.phase = workflowConsent
		return model, nil
	case workflowAssetProgressMsg:
		progress := message.progress
		workflow.progress = &progress
		return model, nil
	case tickMsg:
		return model, model.tick()
	case tea.WindowSizeMsg:
		model.width = max(1, message.Width)
		model.height = max(1, message.Height)
		model.viewport.Width = max(1, model.width-2)
		model.viewport.Height = max(1, model.height-7)
		if workflow.phase == workflowReview {
			model.viewport.SetContent(workflowReviewContent(workflow.preview, workflow.cfg))
		}
		return model, nil
	case externalCancelMsg:
		model.declinePendingAsset()
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
	if workflow.phase == workflowConsent {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyCtrlC {
			model.declinePendingAsset()
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
		}
		switch strings.ToLower(strings.TrimSpace(string(key.Runes))) {
		case "y":
			model.answerAssetConsent(true)
			workflow.phase = workflowPreparing
		case "n":
			model.answerAssetConsent(false)
			workflow.phase = workflowPreparing
		case "q":
			model.declinePendingAsset()
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
		}
		return model, nil
	}
	if workflow.phase == workflowReview {
		switch key.Type {
		case tea.KeyEnter:
			if workflow.preview.SoftwareFallbackConfirmationRequired {
				workflow.phase = workflowSoftwareConfirmation
				workflow.error = ""
				return model, nil
			}
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
				if workflow.returnToAdvanced {
					workflow.phase = workflowAdvanced
				} else {
					workflow.phase = workflowQuickSetup
				}
				workflow.error = ""
			case "q":
				return model, tea.Quit
			}
		}
		return model, nil
	}
	if workflow.phase == workflowSoftwareConfirmation {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyCtrlC {
			workflow.phase = workflowReview
			return model, nil
		}
		switch strings.ToLower(strings.TrimSpace(string(key.Runes))) {
		case "y":
			workflow.cfg.AllowSoftwareEncoding = true
			return model, model.confirmWorkflow()
		case "n", "b":
			workflow.phase = workflowReview
			workflow.cfg.AllowSoftwareEncoding = workflow.baseConfig.AllowSoftwareEncoding
			workflow.error = ""
			workflow.skipPreferenceSave = false
		case "q":
			return model, tea.Quit
		}
		return model, nil
	}
	if workflow.phase == workflowQuickSetup {
		return model.updateQuickSetupKey(key)
	}
	if workflow.phase == workflowPreparing {
		if key.Type == tea.KeyCtrlC || key.Type == tea.KeyEsc || (key.Type == tea.KeyRunes && strings.EqualFold(string(key.Runes), "q")) {
			model.declinePendingAsset()
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
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
	if workflow.phase == workflowAdvanced && key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "b":
			workflow.phase = workflowQuickSetup
			workflow.focused = 0
			workflow.error = ""
			return model, nil
		case "p":
			workflow.returnToAdvanced = true
			return model, model.startWorkflowPreview()
		case "q":
			return model, tea.Quit
		}
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
	case "q":
		return model, tea.Quit
	}
	return model, nil
}

func (model *Model) updateQuickSetupKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	if workflow.editing {
		switch key.Type {
		case tea.KeyEnter:
			if err := setWorkflowField(&workflow.cfg, "input_root", workflow.editor); err != nil {
				workflow.error = err.Error()
				return model, nil
			}
			workflow.editing = false
			workflow.editor = ""
			workflow.returnToAdvanced = false
			return model, model.startWorkflowPreview()
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
	if key.Type == tea.KeyEnter {
		workflow.returnToAdvanced = false
		return model, model.startWorkflowPreview()
	}
	if key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "e":
			workflow.editing = true
			workflow.editor = workflow.cfg.InputRoot
			workflow.error = ""
		case "a":
			workflow.phase = workflowAdvanced
			workflow.focused = 0
			workflow.error = ""
		case "r":
			if workflow.resetPreferences == nil {
				workflow.cfg = workflow.baseConfig
				workflow.error = "No saved preferences to reset."
				return model, nil
			}
			if err := workflow.resetPreferences(); err != nil {
				workflow.error = fmt.Sprintf("Could not reset saved preferences: %v", err)
				return model, nil
			}
			workflow.cfg = workflow.baseConfig
			workflow.error = "Saved preferences reset."
			workflow.preferenceError = ""
		case "q":
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model *Model) startWorkflowPreview() tea.Cmd {
	workflow := model.workflow
	workflow.skipPreferenceSave = false
	if err := workflow.cfg.Validate(); err != nil {
		workflow.error = fmt.Sprintf("Invalid settings: %v", err)
		return nil
	}
	if workflow.prepareWithCallbacks == nil {
		workflow.error = "Run preview is unavailable"
		return nil
	}
	workflow.phase = workflowPreparing
	workflow.error = ""
	workflow.progress = nil
	cfg, ctx, prepare := workflow.cfg, workflow.ctx, workflow.prepareWithCallbacks
	return func() tea.Msg {
		preview, err := prepare(ctx, cfg, app.RuntimeSetupCallbacks{})
		return workflowPreviewMsg{preview: preview, err: err}
	}
}

func (model *Model) answerAssetConsent(accepted bool) {
	if model.workflow.consentReply != nil {
		model.workflow.consentReply <- accepted
		model.workflow.consentReply = nil
	}
}

func (model *Model) declinePendingAsset() {
	model.answerAssetConsent(false)
}

func (model *Model) workflowView() string {
	workflow := model.workflow
	switch workflow.phase {
	case workflowPreparing:
		lines := []string{
			titleStyle.Render("VCX · PREPARING RUN"),
			"Discovering media and checking dependencies…",
		}
		if workflow.progress != nil {
			lines = append(lines, fmt.Sprintf("Downloading %s: %s / %s", workflow.progress.AssetID,
				formatAssetSize(workflow.progress.ReceivedBytes), formatAssetSize(workflow.progress.TotalBytes)))
		}
		lines = append(lines, "q quit")
		return strings.Join(lines, "\n")
	case workflowConsent:
		asset := workflow.pendingAsset
		return strings.Join([]string{
			titleStyle.Render("VCX · DEPENDENCY DOWNLOAD CONSENT"),
			fmt.Sprintf("Download %s version %s for %s?", asset.ID, asset.Version, asset.Target),
			"Source: " + asset.SourceURL,
			"Download: " + asset.DownloadURL,
			"License: " + asset.License,
			"Size: " + formatAssetSize(asset.SizeBytes),
			dimStyle.Render("y allow this pinned download · n decline · q quit"),
		}, "\n")
	case workflowReview:
		controls := "enter continue · b back · ↑/↓ scroll · q quit"
		if workflow.preview.Diagnostics.HasFailures() {
			controls = "Resolve failed checks, then press b to change settings and preview again · q quit"
		}
		return strings.Join([]string{
			titleStyle.Render("VCX · REVIEW RUN"),
			"Planned files and required dependencies",
			model.viewport.View(),
			errorStyle.Render(workflow.error),
			dimStyle.Render(controls),
		}, "\n")
	case workflowSoftwareConfirmation:
		warning := workflow.preview.SoftwareFallbackWarning
		if warning == "" {
			warning = "Software H.264 uses the CPU and may be slower than AMD AMF."
		}
		return strings.Join([]string{
			titleStyle.Render("VCX · CONFIRM SOFTWARE H.264"),
			warning,
			"This run will use libx264 and may use substantial CPU resources.",
			errorStyle.Render(singleLine(workflow.error)),
			dimStyle.Render("y confirm and start · n return to review · q quit"),
		}, "\n")
	default:
		if workflow.phase == workflowQuickSetup {
			return model.workflowQuickSetupView()
		}
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
		titleStyle.Render("VCX · ADVANCED SETTINGS"),
		"Defaults keep both stages enabled and limit resource use. Enter edits a value.",
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
	controls := "↑/↓ or j/k select · enter edit · p preview · b back · q quit"
	if workflow.editing {
		controls = "type value · enter save · esc cancel"
	}
	lines = append(lines, dimStyle.Render(controls))
	return strings.Join(lines, "\n")
}

func (model *Model) workflowQuickSetupView() string {
	workflow := model.workflow
	layout := newWorkflowLayout(model.width, model.height)
	input := workflow.cfg.InputRoot
	if workflow.editing {
		input = workflow.editor + "▏"
	}
	modelSummary := "managed default Whisper model"
	if workflow.cfg.WhisperModelPath != "" {
		modelSummary = workflow.cfg.WhisperModelPath
	}
	inputContent := []string{"Choose a media folder or enter a path", input}
	profileContent := []string{
		fmt.Sprintf("Compression: %s · subtitles: %s", enabledLabel(workflow.cfg.CompressionEnabled), enabledLabel(workflow.cfg.SubtitlesEnabled)),
		fmt.Sprintf("%s · up to %dx%d", strings.ToUpper(workflow.cfg.VideoCodec), workflow.cfg.MaxWidth, workflow.cfg.MaxHeight),
		"Source frame rate preserved",
		"Resources: " + workflow.cfg.ResourceProfile,
		"Output: " + workflow.cfg.OutputDir,
		"Whisper: " + modelSummary,
	}
	var lines []string
	if layout.compact {
		lines = []string{
			titleStyle.Render("VCX · SELECT MEDIA"),
			sectionStyle.Render("MEDIA INPUT"),
			infoStyle.Render(truncateDisplay(input, layout.width)),
			sectionStyle.Render("RUN PROFILE"),
			infoStyle.Render(fmt.Sprintf("Compress %s · subtitles %s", enabledLabel(workflow.cfg.CompressionEnabled), enabledLabel(workflow.cfg.SubtitlesEnabled))),
			infoStyle.Render(fmt.Sprintf("%s · max %dx%d", strings.ToUpper(workflow.cfg.VideoCodec), workflow.cfg.MaxWidth, workflow.cfg.MaxHeight)),
		}
		if layout.height >= 10 {
			lines = append(lines, infoStyle.Render("Resources: "+workflow.cfg.ResourceProfile))
		}
	} else {
		lines = []string{
			titleStyle.Render("VCX · SELECT MEDIA"),
			infoStyle.Render("STEP 1 OF 3  ·  SELECT MEDIA  →  REVIEW  →  PROCESS"),
		}
		if layout.split {
			leftWidth := max(1, (layout.width-3)/2)
			rightWidth := max(1, layout.width-leftWidth-3)
			panels := lipgloss.JoinHorizontal(lipgloss.Top,
				layout.panel("MEDIA INPUT", inputContent, leftWidth),
				" ",
				layout.panel("RUN PROFILE", profileContent, rightWidth),
			)
			lines = append(lines, "", panels)
		} else {
			lines = append(lines, "", layout.panel("MEDIA INPUT", inputContent, layout.width), layout.panel("RUN PROFILE", profileContent, layout.width))
		}
	}
	if workflow.preferenceError != "" {
		lines = append(lines, errorStyle.Render(singleLine(workflow.preferenceError)))
	}
	if workflow.error != "" {
		lines = append(lines, errorStyle.Render(singleLine(workflow.error)))
	}
	controls := layout.controls()
	if workflow.editing {
		controls = []string{selectedStyle.Render("type path · enter review · esc cancel · ctrl+u clear")}
	}
	lines = append(lines, controls...)
	if len(lines) > layout.height {
		bodyEnd := len(lines) - len(controls)
		bodyRows := max(0, layout.height-len(controls))
		lines = append(lines[:min(bodyEnd, bodyRows)], controls...)
	}
	for index, line := range lines {
		if width := lipgloss.Width(line); width > layout.width {
			lines[index] = truncateDisplay(line, layout.width)
		}
	}
	return strings.Join(lines, "\n")
}

func workflowReviewContent(preview app.RunPreview, cfg config.Config) string {
	lines := []string{fmt.Sprintf("Files to process: %d", len(preview.Plans))}
	lines = append(lines, fmt.Sprintf("Stages: compression %s · subtitles %s",
		enabledLabel(preview.CompressionEnabled), enabledLabel(preview.SubtitlesEnabled)))
	if preview.CompressionEnabled {
		lines = append(lines, fmt.Sprintf("Encoding: %s (%s) · max %dx%d · %s", preview.EncoderBackend, preview.EncoderName,
			cfg.MaxWidth, cfg.MaxHeight, frameRateLabel(cfg.FrameRate)))
	}
	lines = append(lines, fmt.Sprintf("Resources: %s · output directory: %s", cfg.ResourceProfile, cfg.OutputDir))
	if preview.SoftwareFallbackAvailable {
		lines = append(lines, "", errorStyle.Render("Software H.264 fallback: "+preview.SoftwareFallbackWarning))
	}
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
	case "compression_enabled":
		return strconv.FormatBool(cfg.CompressionEnabled)
	case "subtitles_enabled":
		return strconv.FormatBool(cfg.SubtitlesEnabled)
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
	case "compression_enabled":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("compression enabled must be true or false: %w", err)
		}
		cfg.CompressionEnabled = parsed
	case "subtitles_enabled":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("subtitles enabled must be true or false: %w", err)
		}
		cfg.SubtitlesEnabled = parsed
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
	if workflow.savePreferences != nil && !workflow.skipPreferenceSave {
		if err := workflow.savePreferences(workflow.cfg); err != nil {
			workflow.skipPreferenceSave = true
			workflow.error = fmt.Sprintf("Could not save preferences: %v. Press the confirmation key again to run without saving.", err)
			return nil
		}
	}
	workflow.phase = workflowRunning
	workflow.error = ""
	model.startRun(workflow.cfg)
	return nil
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func frameRateLabel(frameRate int) string {
	if frameRate == 0 {
		return "source frame rate"
	}
	return fmt.Sprintf("%d fps", frameRate)
}

func formatAssetSize(size int64) string {
	if size >= 1<<30 {
		return fmt.Sprintf("%.1f GiB", float64(size)/(1<<30))
	}
	if size >= 1<<20 {
		return fmt.Sprintf("%.1f MiB", float64(size)/(1<<20))
	}
	if size >= 1<<10 {
		return fmt.Sprintf("%.1f KiB", float64(size)/(1<<10))
	}
	return fmt.Sprintf("%d B", size)
}
