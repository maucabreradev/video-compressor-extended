package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
	workflowMediaPicker
	workflowSelectionReview
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
	pickerPath           string
	pickerEntries        []mediaPickerEntry
	pickerCursor         int
	pickerError          string
	pickerLoading        bool
	pickerSelecting      bool
	selectionRoot        string
	selectionActive      bool
	selectionReviewed    bool
	selectionCursor      int
}

type mediaPickerEntry struct {
	name  string
	path  string
	isDir bool
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

type mediaPickerDirectoryMsg struct {
	path          string
	preferredName string
	entries       []mediaPickerEntry
	err           error
}

type mediaPickerSelectionMsg struct {
	root  string
	paths []string
	err   error
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
		ctx:             context.Background(),
		cfg:             cfg,
		baseConfig:      cfg,
		phase:           workflowQuickSetup,
		selectionActive: len(cfg.InputPaths) > 0,
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
		model.setWorkflowViewportContent(workflowReviewContent(message.preview, workflow.cfg), true)
		return model, nil
	case workflowAssetConsentMsg:
		if workflow.phase != workflowPreparing {
			message.reply <- false
			return model, nil
		}
		workflow.pendingAsset = message.asset
		workflow.consentReply = message.reply
		workflow.phase = workflowConsent
		model.setWorkflowViewportContent(workflowAssetConsentContent(message.asset), true)
		return model, nil
	case workflowAssetProgressMsg:
		progress := message.progress
		workflow.progress = &progress
		return model, nil
	case mediaPickerDirectoryMsg:
		workflow.pickerLoading = false
		workflow.pickerPath = message.path
		workflow.pickerError = ""
		workflow.pickerEntries = message.entries
		workflow.pickerCursor = 0
		if message.err != nil {
			workflow.pickerError = message.err.Error()
			return model, nil
		}
		for index, entry := range message.entries {
			if entry.name == message.preferredName {
				workflow.pickerCursor = index
				break
			}
		}
		return model, nil
	case mediaPickerSelectionMsg:
		workflow.pickerSelecting = false
		workflow.pickerError = ""
		if message.err != nil {
			workflow.pickerError = message.err.Error()
			return model, nil
		}
		if len(message.paths) == 0 {
			workflow.pickerError = "No supported media files found in that folder."
			return model, nil
		}
		workflow.selectionRoot = message.root
		workflow.cfg.InputRoot = message.root
		workflow.cfg.InputPaths = normalizeSelectionPaths(append(workflow.cfg.InputPaths, message.paths...))
		workflow.selectionActive = true
		workflow.selectionReviewed = false
		return model, nil
	case tickMsg:
		return model, model.tick()
	case tea.WindowSizeMsg:
		model.width = max(1, message.Width)
		model.height = max(1, message.Height)
		switch workflow.phase {
		case workflowReview:
			model.setWorkflowViewportContent(workflowReviewContent(workflow.preview, workflow.cfg), false)
		case workflowConsent:
			model.setWorkflowViewportContent(workflowAssetConsentContent(workflow.pendingAsset), false)
		case workflowSoftwareConfirmation:
			model.setWorkflowViewportContent(workflowSoftwareConfirmationContent(workflow.preview), false)
		default:
			model.viewport.Width = max(1, model.width)
			model.viewport.Height = max(1, model.height-7)
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
	if workflow.phase == workflowMediaPicker {
		return model.updateMediaPickerKey(key)
	}
	if workflow.phase == workflowSelectionReview {
		return model.updateSelectionReviewKey(key)
	}
	if workflow.phase == workflowConsent {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyCtrlC {
			model.declinePendingAsset()
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
		}
		if key.Type == tea.KeyUp || key.Type == tea.KeyDown || key.Type == tea.KeyPgUp || key.Type == tea.KeyPgDown {
			var command tea.Cmd
			model.viewport, command = model.viewport.Update(key)
			return model, command
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
				model.setWorkflowViewportContent(workflowSoftwareConfirmationContent(workflow.preview), true)
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
			model.setWorkflowViewportContent(workflowReviewContent(workflow.preview, workflow.cfg), false)
			return model, nil
		}
		if key.Type == tea.KeyUp || key.Type == tea.KeyDown || key.Type == tea.KeyPgUp || key.Type == tea.KeyPgDown {
			var command tea.Cmd
			model.viewport, command = model.viewport.Update(key)
			return model, command
		}
		switch strings.ToLower(strings.TrimSpace(string(key.Runes))) {
		case "y":
			workflow.cfg.AllowSoftwareEncoding = true
			return model, model.confirmWorkflow()
		case "n", "b":
			workflow.phase = workflowReview
			model.setWorkflowViewportContent(workflowReviewContent(workflow.preview, workflow.cfg), false)
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
			workflow.cfg.InputPaths = nil
			workflow.selectionActive = false
			workflow.selectionRoot = ""
			workflow.selectionReviewed = false
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
			return model, model.openMediaPicker()
		case "p":
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

func (model *Model) openMediaPicker() tea.Cmd {
	workflow := model.workflow
	path := strings.TrimSpace(workflow.cfg.InputRoot)
	if path == "" {
		path = "."
	}
	preferredName := ""
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		preferredName = filepath.Base(path)
		path = filepath.Dir(path)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		workflow.error = fmt.Sprintf("Resolve media folder %q: %v", path, err)
		return nil
	}
	path = filepath.Clean(absolutePath)
	selectionRoot := path
	if workflow.selectionActive {
		selectionRoot, err = filepath.Abs(workflow.cfg.InputRoot)
		if err != nil {
			workflow.error = fmt.Sprintf("Resolve selection root %q: %v", workflow.cfg.InputRoot, err)
			return nil
		}
	}
	workflow.phase = workflowMediaPicker
	workflow.pickerPath = path
	workflow.selectionRoot = filepath.Clean(selectionRoot)
	workflow.pickerEntries = nil
	workflow.pickerCursor = 0
	workflow.pickerError = ""
	workflow.pickerLoading = true
	workflow.pickerSelecting = false
	return loadMediaPickerDirectory(path, preferredName)
}

func loadMediaPickerDirectory(path, preferredName string) tea.Cmd {
	return func() tea.Msg {
		entries, err := readMediaPickerDirectory(path)
		return mediaPickerDirectoryMsg{path: path, preferredName: preferredName, entries: entries, err: err}
	}
}

func readMediaPickerDirectory(path string) ([]mediaPickerEntry, error) {
	directoryEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read media folder %q: %w", path, err)
	}
	entries := make([]mediaPickerEntry, 0, len(directoryEntries))
	for _, entry := range directoryEntries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			entries = append(entries, mediaPickerEntry{name: entry.Name(), path: fullPath, isDir: true})
			continue
		}
		if !isSupportedPickerMedia(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect media file %q: %w", fullPath, err)
		}
		if info.Mode().IsRegular() {
			entries = append(entries, mediaPickerEntry{name: entry.Name(), path: fullPath})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})
	return entries, nil
}

func isSupportedPickerMedia(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".ts", ".webm", ".wmv":
		return true
	default:
		return false
	}
}

func (model *Model) updateMediaPickerKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	if workflow.pickerLoading || workflow.pickerSelecting {
		return model, nil
	}
	switch key.Type {
	case tea.KeyUp:
		workflow.pickerCursor = max(0, workflow.pickerCursor-1)
	case tea.KeyDown:
		workflow.pickerCursor = min(max(0, len(workflow.pickerEntries)-1), workflow.pickerCursor+1)
	case tea.KeyBackspace:
		parent := filepath.Dir(filepath.Clean(workflow.pickerPath))
		if parent != filepath.Clean(workflow.pickerPath) {
			if !workflow.selectionActive {
				workflow.selectionRoot = parent
			}
			return model, model.loadMediaPicker(parent, filepath.Base(workflow.pickerPath))
		}
	case tea.KeyEnter:
		if len(workflow.pickerEntries) == 0 {
			return model, nil
		}
		entry := workflow.pickerEntries[workflow.pickerCursor]
		if entry.isDir {
			return model, model.loadMediaPicker(entry.path, "")
		}
		workflow.cfg.InputRoot = entry.path
		workflow.cfg.InputPaths = nil
		workflow.selectionActive = false
		workflow.selectionRoot = ""
		workflow.selectionReviewed = false
		workflow.pickerError = ""
		workflow.phase = workflowQuickSetup
	case tea.KeySpace:
		if len(workflow.pickerEntries) == 0 {
			return model, nil
		}
		entry := workflow.pickerEntries[workflow.pickerCursor]
		if entry.isDir {
			workflow.pickerError = "Use 'a' to add a folder recursively."
			return model, nil
		}
		if err := model.toggleMediaSelection(entry.path); err != nil {
			workflow.pickerError = err.Error()
		} else {
			workflow.pickerError = ""
		}
	case tea.KeyEsc:
		workflow.phase = workflowQuickSetup
		workflow.pickerError = ""
	case tea.KeyRunes:
		switch strings.ToLower(string(key.Runes)) {
		case "j":
			workflow.pickerCursor = min(max(0, len(workflow.pickerEntries)-1), workflow.pickerCursor+1)
		case "k":
			workflow.pickerCursor = max(0, workflow.pickerCursor-1)
		case "f":
			workflow.cfg.InputRoot = workflow.pickerPath
			workflow.cfg.InputPaths = nil
			workflow.selectionActive = false
			workflow.selectionRoot = ""
			workflow.selectionReviewed = false
			workflow.pickerError = ""
			workflow.phase = workflowQuickSetup
		case "a":
			if len(workflow.pickerEntries) == 0 || !workflow.pickerEntries[workflow.pickerCursor].isDir {
				workflow.pickerError = "Highlight a folder to add its media recursively."
				return model, nil
			}
			workflow.pickerSelecting = true
			workflow.pickerError = ""
			return model, loadMediaPickerSelection(workflow.selectionRoot, workflow.pickerEntries[workflow.pickerCursor].path)
		case "r":
			workflow.phase = workflowSelectionReview
			workflow.selectionCursor = 0
			workflow.error = ""
		case "p":
			workflow.phase = workflowQuickSetup
			workflow.editing = true
			workflow.editor = workflow.cfg.InputRoot
		case "q":
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model *Model) updateSelectionReviewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	switch key.Type {
	case tea.KeyEsc, tea.KeyBackspace:
		workflow.phase = workflowMediaPicker
		workflow.error = ""
	case tea.KeyUp:
		workflow.selectionCursor = max(0, workflow.selectionCursor-1)
	case tea.KeyDown:
		workflow.selectionCursor = min(max(0, len(workflow.cfg.InputPaths)-1), workflow.selectionCursor+1)
	case tea.KeyEnter:
		if len(workflow.cfg.InputPaths) == 0 {
			workflow.error = "Select at least one media file before continuing."
			return model, nil
		}
		workflow.selectionReviewed = true
		workflow.returnToAdvanced = false
		workflow.phase = workflowQuickSetup
		workflow.error = ""
	case tea.KeyRunes:
		switch strings.ToLower(string(key.Runes)) {
		case "j":
			workflow.selectionCursor = min(max(0, len(workflow.cfg.InputPaths)-1), workflow.selectionCursor+1)
		case "k":
			workflow.selectionCursor = max(0, workflow.selectionCursor-1)
		case "x":
			if len(workflow.cfg.InputPaths) > 0 {
				workflow.cfg.InputPaths = append(workflow.cfg.InputPaths[:workflow.selectionCursor], workflow.cfg.InputPaths[workflow.selectionCursor+1:]...)
				workflow.selectionCursor = min(workflow.selectionCursor, max(0, len(workflow.cfg.InputPaths)-1))
			}
			workflow.selectionReviewed = false
			workflow.error = ""
		case "c":
			workflow.cfg.InputPaths = nil
			workflow.selectionCursor = 0
			workflow.selectionActive = true
			workflow.selectionReviewed = false
			workflow.error = ""
		case "e":
			workflow.phase = workflowMediaPicker
			workflow.error = ""
		case "q":
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model *Model) loadMediaPicker(path, preferredName string) tea.Cmd {
	model.workflow.pickerPath = path
	model.workflow.pickerEntries = nil
	model.workflow.pickerCursor = 0
	model.workflow.pickerError = ""
	model.workflow.pickerLoading = true
	return loadMediaPickerDirectory(path, preferredName)
}

func loadMediaPickerSelection(root, directory string) tea.Cmd {
	return func() tea.Msg {
		paths, err := discoverPickerMedia(root, directory)
		return mediaPickerSelectionMsg{root: root, paths: paths, err: err}
	}
}

func discoverPickerMedia(root, directory string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve selection root %q: %w", root, err)
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve media folder %q: %w", directory, err)
	}
	if _, err := relativeMediaSelectionPath(root, directory); err != nil {
		return nil, err
	}
	var paths []string
	err = filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan media folder %q: %w", path, walkErr)
		}
		if path != directory && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !isSupportedPickerMedia(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect media file %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, err := relativeMediaSelectionPath(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, relative)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return normalizeSelectionPaths(paths), nil
}

func (model *Model) toggleMediaSelection(path string) error {
	workflow := model.workflow
	relative, err := relativeMediaSelectionPath(workflow.selectionRoot, path)
	if err != nil {
		return err
	}
	workflow.cfg.InputRoot = workflow.selectionRoot
	workflow.selectionActive = true
	workflow.selectionReviewed = false
	for index, selected := range workflow.cfg.InputPaths {
		if selectionPathKey(filepath.Clean(selected)) == selectionPathKey(filepath.Clean(relative)) {
			workflow.cfg.InputPaths = append(workflow.cfg.InputPaths[:index], workflow.cfg.InputPaths[index+1:]...)
			workflow.selectionCursor = min(workflow.selectionCursor, max(0, len(workflow.cfg.InputPaths)-1))
			return nil
		}
	}
	workflow.cfg.InputPaths = normalizeSelectionPaths(append(workflow.cfg.InputPaths, relative))
	return nil
}

func relativeMediaSelectionPath(root, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve selection root %q: %w", root, err)
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve selected media path %q: %w", path, err)
	}
	relative, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", fmt.Errorf("make media path relative to selection root: %w", err)
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("selected media path %q is outside selection root %q", path, root)
	}
	return filepath.Clean(relative), nil
}

func normalizeSelectionPaths(paths []string) []string {
	unique := make(map[string]string, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "." || path == "" {
			continue
		}
		key := selectionPathKey(path)
		if _, exists := unique[key]; !exists {
			unique[key] = path
		}
	}
	normalized := make([]string, 0, len(unique))
	for _, path := range unique {
		normalized = append(normalized, path)
	}
	sort.Slice(normalized, func(i, j int) bool {
		left, right := strings.ToLower(normalized[i]), strings.ToLower(normalized[j])
		if left == right {
			return normalized[i] < normalized[j]
		}
		return left < right
	})
	return normalized
}

func selectionPathKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func (model *Model) startWorkflowPreview() tea.Cmd {
	workflow := model.workflow
	workflow.skipPreferenceSave = false
	if workflow.selectionActive {
		if len(workflow.cfg.InputPaths) == 0 {
			workflow.error = "Select at least one media file before continuing."
			return nil
		}
		if !workflow.selectionReviewed {
			workflow.phase = workflowSelectionReview
			workflow.selectionCursor = 0
			workflow.returnToAdvanced = false
			workflow.error = "Review selected media before checking dependencies."
			return nil
		}
	}
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
	layout := newWorkflowLayout(model.width, model.height)
	switch workflow.phase {
	case workflowPreparing:
		body := []string{"Checking media and dependencies…"}
		if workflow.progress != nil {
			if layout.compact {
				body = append(body, fmt.Sprintf("Downloading %s", workflow.progress.AssetID),
					fmt.Sprintf("%s / %s", formatAssetSize(workflow.progress.ReceivedBytes), formatAssetSize(workflow.progress.TotalBytes)))
			} else {
				body = append(body, fmt.Sprintf("Downloading %s: %s / %s", workflow.progress.AssetID,
					formatAssetSize(workflow.progress.ReceivedBytes), formatAssetSize(workflow.progress.TotalBytes)))
			}
		}
		body = strings.Split(wrapTerminalText(strings.Join(body, "\n"), layout.width), "\n")
		return composeResponsiveScreen(
			[]string{titleStyle.Render("VCX · PREPARING RUN")}, body,
			workflowPhaseControls(workflowPreparing, layout, false), layout.width, layout.height)
	case workflowConsent:
		return model.workflowViewportScreen("VCX · DEPENDENCY DOWNLOAD CONSENT", workflowPhaseControls(workflowConsent, layout, false))
	case workflowMediaPicker:
		return model.mediaPickerView()
	case workflowSelectionReview:
		return model.selectionReviewView()
	case workflowReview:
		return model.workflowViewportScreen("VCX · REVIEW RUN", workflowPhaseControls(workflowReview, layout, workflow.preview.Diagnostics.HasFailures()))
	case workflowSoftwareConfirmation:
		return model.workflowViewportScreen("VCX · CONFIRM SOFTWARE H.264", workflowPhaseControls(workflowSoftwareConfirmation, layout, false))
	default:
		if workflow.phase == workflowQuickSetup {
			return model.workflowQuickSetupView()
		}
		return model.workflowSettingsView()
	}
}

func workflowPhaseControls(phase workflowPhase, layout workflowLayout, hasFailures bool) []string {
	var controls []string
	switch phase {
	case workflowPreparing:
		controls = []string{"q quit"}
	case workflowConsent:
		if layout.compact {
			controls = []string{"y allow · n decline", "↑/↓ details · q quit"}
		} else {
			controls = []string{"y allow · n decline · ↑/↓ scroll · q quit"}
		}
	case workflowSoftwareConfirmation:
		if layout.compact {
			controls = []string{"y confirm · n review", "↑/↓ details · q quit"}
		} else {
			controls = []string{"y confirm · n review · ↑/↓ scroll · q quit"}
		}
	case workflowReview:
		if hasFailures && layout.compact {
			controls = []string{"Resolve checks · b settings", "q quit"}
		} else if hasFailures {
			controls = []string{"Resolve failed checks · b settings · q quit"}
		} else if layout.compact {
			controls = []string{"enter continue · b back", "↑/↓ scroll · q quit"}
		} else {
			controls = []string{"enter continue · b back · ↑/↓ scroll · q quit"}
		}
	}
	for index, control := range controls {
		controls[index] = dimStyle.Render(control)
	}
	return controls
}

func (model *Model) setWorkflowViewportContent(content string, reset bool) {
	layout := newWorkflowLayout(model.width, model.height)
	controls := workflowPhaseControls(model.workflow.phase, layout, model.workflow.preview.Diagnostics.HasFailures())
	reserved := 1 + len(controls)
	if model.workflow.error != "" {
		reserved += len(strings.Split(wrapTerminalText(model.workflow.error, layout.width), "\n"))
	}
	model.viewport.Width = layout.width
	model.viewport.Height = max(1, layout.height-reserved)
	model.viewport.SetContent(wrapTerminalText(content, layout.width))
	if reset {
		model.viewport.GotoTop()
	} else {
		model.viewport.SetYOffset(model.viewport.YOffset)
	}
}

func (model *Model) workflowViewportScreen(title string, controls []string) string {
	header := []string{titleStyle.Render(title)}
	body := make([]string, 0)
	if model.workflow.error != "" {
		for _, line := range strings.Split(wrapTerminalText(singleLine(model.workflow.error), model.width), "\n") {
			body = append(body, errorStyle.Render(line))
		}
	}
	body = append(body, strings.Split(model.viewport.View(), "\n")...)
	return composeResponsiveScreen(header, body, controls, model.width, model.height)
}

func workflowAssetConsentContent(asset runtimeassets.Asset) string {
	return strings.Join([]string{
		fmt.Sprintf("Asset: %s", asset.ID),
		fmt.Sprintf("Version: %s", asset.Version),
		fmt.Sprintf("Target: %s", asset.Target),
		"Source: " + asset.SourceURL,
		"Download: " + asset.DownloadURL,
		"License: " + asset.License,
		"Size: " + formatAssetSize(asset.SizeBytes),
		"Allow this pinned runtime download?",
	}, "\n")
}

func workflowSoftwareConfirmationContent(preview app.RunPreview) string {
	warning := preview.SoftwareFallbackWarning
	if warning == "" {
		warning = "Software H.264 uses the CPU and may be slower than AMD AMF."
	}
	return strings.Join([]string{
		"Software H.264 fallback warning:",
		warning,
		"This run will use libx264 and may use substantial CPU resources.",
	}, "\n")
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
	if workflow.selectionActive {
		inputContent = append(inputContent, fmt.Sprintf("Selected items: %d", len(workflow.cfg.InputPaths)))
		for _, path := range workflow.cfg.InputPaths[:min(2, len(workflow.cfg.InputPaths))] {
			inputContent = append(inputContent, "· "+path)
		}
	}
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
		}
		if workflow.selectionActive && layout.height >= 9 {
			lines = append(lines, infoStyle.Render(fmt.Sprintf("Selected items: %d", len(workflow.cfg.InputPaths))))
		}
		lines = append(lines,
			sectionStyle.Render("RUN PROFILE"),
			infoStyle.Render(fmt.Sprintf("Compress %s · subtitles %s", enabledLabel(workflow.cfg.CompressionEnabled), enabledLabel(workflow.cfg.SubtitlesEnabled))),
			infoStyle.Render(fmt.Sprintf("%s · max %dx%d", strings.ToUpper(workflow.cfg.VideoCodec), workflow.cfg.MaxWidth, workflow.cfg.MaxHeight)),
		)
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
	if layout.compact {
		controls = []string{
			selectedStyle.Render("enter review · e browse"),
			dimStyle.Render("p type path · a options · q quit"),
		}
	} else if lipgloss.Width("enter review · e browse · p type path · a advanced settings · r reset preferences · q quit") <= layout.width {
		controls = []string{dimStyle.Render("enter review · e browse · p type path · a advanced settings · r reset preferences · q quit")}
	} else {
		controls = []string{
			selectedStyle.Render("enter review · e browse · p type path · a advanced settings"),
			dimStyle.Render("r reset preferences · q quit"),
		}
	}
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

func (model *Model) mediaPickerView() string {
	workflow := model.workflow
	layout := newWorkflowLayout(model.width, model.height)
	controls := []string{"↑/↓ move · space mark file", "a add folder recursively · r review selection", "enter open · f folder · p path · esc back · q quit"}
	if layout.compact {
		controls = []string{"↑/↓ move · space mark · a add folder", "r review · enter open · f folder", "p path · esc back · q quit"}
	}
	lines := []string{
		titleStyle.Render("VCX · BROWSE MEDIA"),
		infoStyle.Render("Folder: " + truncateDisplay(workflow.pickerPath, layout.width)),
		infoStyle.Render(fmt.Sprintf("Selected: %d", len(workflow.cfg.InputPaths))),
	}
	if workflow.pickerLoading {
		lines = append(lines, infoStyle.Render("Loading folder…"))
	} else if workflow.pickerSelecting {
		lines = append(lines, infoStyle.Render("Adding folder media…"))
	} else if workflow.pickerError != "" {
		lines = append(lines, errorStyle.Render(truncateDisplay(workflow.pickerError, layout.width)))
	} else if len(workflow.pickerEntries) == 0 {
		lines = append(lines, infoStyle.Render("No supported media or subfolders here."))
	} else {
		visibleRows := max(1, layout.height-len(lines)-len(controls))
		start := max(0, workflow.pickerCursor-visibleRows+1)
		end := min(len(workflow.pickerEntries), start+visibleRows)
		for index := start; index < end; index++ {
			entry := workflow.pickerEntries[index]
			marker, selected, kind := "  ", "[ ]", "[VID]"
			if index == workflow.pickerCursor {
				marker = "> "
			}
			if entry.isDir {
				kind = "[DIR]"
			} else if model.mediaPickerEntrySelected(entry) {
				selected = "[x]"
			}
			row := truncateDisplay(marker+selected+" "+kind+" "+entry.name, layout.width)
			if index == workflow.pickerCursor {
				row = selectedStyle.Render(row)
			} else {
				row = infoStyle.Render(row)
			}
			lines = append(lines, row)
		}
	}
	bodyRows := max(0, layout.height-len(controls))
	if len(lines) > bodyRows {
		lines = lines[:bodyRows]
	}
	lines = append(lines, controls...)
	for index, line := range lines {
		if width := lipgloss.Width(line); width > layout.width {
			lines[index] = truncateDisplay(line, layout.width)
		}
	}
	return strings.Join(lines, "\n")
}

func (model *Model) mediaPickerEntrySelected(entry mediaPickerEntry) bool {
	if entry.isDir || model.workflow.selectionRoot == "" {
		return false
	}
	relative, err := relativeMediaSelectionPath(model.workflow.selectionRoot, entry.path)
	if err != nil {
		return false
	}
	key := selectionPathKey(relative)
	for _, selected := range model.workflow.cfg.InputPaths {
		if selectionPathKey(filepath.Clean(selected)) == key {
			return true
		}
	}
	return false
}

func (model *Model) selectionReviewView() string {
	workflow := model.workflow
	layout := newWorkflowLayout(model.width, model.height)
	root := workflow.selectionRoot
	if root == "" {
		root = workflow.cfg.InputRoot
	}
	controls := []string{"↑/↓ choose · x remove · c clear", "enter continue · e browse · esc back · q quit"}
	if layout.compact {
		controls = []string{"x remove · c clear · ↑/↓ choose", "enter continue · e browse · esc back"}
	}
	lines := []string{
		titleStyle.Render("VCX · SELECTED MEDIA"),
		infoStyle.Render(fmt.Sprintf("Selected items: %d", len(workflow.cfg.InputPaths))),
		infoStyle.Render("Common root: " + truncateDisplay(root, layout.width)),
	}
	if len(workflow.cfg.InputPaths) == 0 {
		lines = append(lines, infoStyle.Render("No media selected."))
	} else {
		workflow.selectionCursor = min(workflow.selectionCursor, len(workflow.cfg.InputPaths)-1)
		visibleRows := max(1, layout.height-len(lines)-len(controls)-1)
		start := max(0, workflow.selectionCursor-visibleRows+1)
		end := min(len(workflow.cfg.InputPaths), start+visibleRows)
		for index := start; index < end; index++ {
			marker := "  "
			if index == workflow.selectionCursor {
				marker = "> "
			}
			row := truncateDisplay(marker+workflow.cfg.InputPaths[index], layout.width)
			if index == workflow.selectionCursor {
				row = selectedStyle.Render(row)
			} else {
				row = infoStyle.Render(row)
			}
			lines = append(lines, row)
		}
	}
	if workflow.error != "" {
		lines = append(lines, errorStyle.Render(truncateDisplay(workflow.error, layout.width)))
	}
	bodyRows := max(0, layout.height-len(controls))
	if len(lines) > bodyRows {
		lines = lines[:bodyRows]
	}
	lines = append(lines, controls...)
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
	model.viewport.GotoTop()
	model.refreshViewport()
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
