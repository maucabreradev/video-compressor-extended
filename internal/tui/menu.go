package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
)

type workflowMenuAction uint8

const (
	menuStartRun workflowMenuAction = iota
	menuBrowseMedia
	menuResumeSession
	menuSettings
	menuQuit
	menuShowRecoveryQueue
	menuCancelAndPreserve
)

type workflowMenuItem struct {
	key      rune
	label    string
	action   workflowMenuAction
	disabled bool
}

var (
	menuTextStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE"))
	menuMutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7A7A7"))
	menuHeaderStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EEEEEE"))
	menuArtAccentStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2474A8"))
	menuSelectedStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#2474A8"))
	menuEnabledStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE")).Background(lipgloss.Color("#1D1D1D"))
	menuDisabledStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7A7A7")).Background(lipgloss.Color("#272727"))
	menuErrorTextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E34949")).Bold(true)
	menuWarningTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2B563")).Bold(true)
)

var (
	vcxASCIIArt = []string{
		".#   #.   .#####.   .#   #.",
		".#   #.  ##     ##   .# #.",
		".#   #.  ##           ###",
		" .# #.   ##          .#",
		"  .#.     .#####.   .#   #.",
	}
	recoveryQuestionASCIIArt = []string{
		" .#####.   .#####.   .#####.",
		"##     ## ##     ## ##     ##",
		"       ##        ##        ##",
		"     .##.      .##.      .##.",
		"    .##.      .##.      .##.",
		"    ##        ##        ##",
		"",
		"    ##        ##        ##",
	}
)

func homeMenuItems(hasRecovery bool) []workflowMenuItem {
	items := []workflowMenuItem{
		{key: 'v', label: "Start a new run", action: menuStartRun},
		{key: 'b', label: "Browse media", action: menuBrowseMedia},
	}
	if hasRecovery {
		items = append(items, workflowMenuItem{key: 'r', label: "Resume previous session", action: menuResumeSession})
	}
	return append(items,
		workflowMenuItem{key: 's', label: "Settings", action: menuSettings},
		workflowMenuItem{key: 'q', label: "Quit", action: menuQuit},
	)
}

func recoveryMenuItems(jobReady bool) []workflowMenuItem {
	return []workflowMenuItem{
		{key: 'v', label: "Resume VCX", action: menuResumeSession, disabled: !jobReady},
		{key: 'b', label: "Show queue", action: menuShowRecoveryQueue},
		{key: 's', label: "Settings", action: menuSettings},
		{key: 'c', label: "Cancel and preserve", action: menuCancelAndPreserve},
	}
}

func renderWorkflowMenu(
	width, height int,
	heading, subtitle string,
	details []string,
	items []workflowMenuItem,
	cursor int,
	footer []string,
) string {
	layout := newWorkflowLayout(width, height)
	menuWidth := min(layout.width, 64)
	center := lipgloss.NewStyle().Width(menuWidth).Align(lipgloss.Center)
	var brandLines, titleLines []string

	if layout.compact {
		compactHeading := "VCX / " + strings.ToUpper(heading)
		if heading == "Resume Previous Session" {
			compactHeading = "VCX / RECOVERY"
		}
		titleLines = append(titleLines, menuHeaderStyle.Render(truncateDisplay(compactHeading, menuWidth)))
	} else {
		art := vcxASCIIArt
		if heading == "Resume Previous Session" {
			art = recoveryQuestionASCIIArt
		}
		for _, row := range art {
			brandLines = append(brandLines, center.Render(renderASCIIArtLine(row)))
		}
		if heading != "Resume Previous Session" {
			titleLines = append(titleLines, center.Render(menuMutedStyle.Render("Video Compressor eXtended")))
		}
		titleLines = append(titleLines, center.Render(menuTextStyle.Render(heading)))
	}
	if subtitle != "" && !layout.compact {
		titleLines = append(titleLines, center.Render(menuMutedStyle.Render(truncateDisplay(subtitle, menuWidth))))
	}
	var detailLines, menuLines, footerLines []string
	for _, detail := range details {
		style := menuMutedStyle
		if strings.HasPrefix(detail, "ERROR:") {
			style = menuErrorTextStyle
		} else if strings.HasPrefix(detail, "WARNING:") {
			style = menuWarningTextStyle
		}
		for _, wrapped := range strings.Split(wrapTerminalText(detail, menuWidth), "\n") {
			detailLines = append(detailLines, center.Render(style.Render(truncateDisplay(wrapped, menuWidth))))
		}
	}
	for index, item := range items {
		marker := "  "
		if index == cursor {
			marker = "> "
		}
		label := fmt.Sprintf("%s%c) %s", marker, item.key, item.label)
		style := menuEnabledStyle
		if item.disabled {
			label += " [unavailable]"
			style = menuDisabledStyle
		} else if index == cursor {
			style = menuSelectedStyle
		}
		row := style.Width(max(1, menuWidth-2)).Padding(0, 1).Render(truncateDisplay(label, max(1, menuWidth-2)))
		menuLines = append(menuLines, row)
	}
	for _, hint := range footer {
		for _, wrapped := range strings.Split(wrapTerminalText(hint, menuWidth), "\n") {
			footerLines = append(footerLines, center.Render(menuMutedStyle.Render(truncateDisplay(wrapped, menuWidth))))
		}
	}

	mandatoryRows := len(brandLines) + len(titleLines) + len(menuLines) + len(footerLines)
	availableRows := max(0, layout.height-mandatoryRows)
	introLines := append(append([]string(nil), titleLines...), detailLines...)
	sectionCount := 0
	for _, section := range [][]string{brandLines, introLines, menuLines, footerLines} {
		if len(section) > 0 {
			sectionCount++
		}
	}
	gapLimit := 3
	if layout.compact {
		gapLimit = 2
		if availableRows <= 1 {
			gapLimit = 0
		}
	}
	gapRows := min(min(max(0, sectionCount-1), availableRows), gapLimit)
	rowsAfterSectionGaps := max(0, availableRows-gapRows)
	rowGapCandidates := 0
	if !layout.compact {
		rowGapCandidates = max(0, len(menuLines)-1)
	}
	reservedDetailRows := 0
	if len(detailLines) > 0 && rowsAfterSectionGaps > 0 {
		reservedDetailRows = 1
	}
	menuRowGaps := min(rowGapCandidates, max(0, rowsAfterSectionGaps-reservedDetailRows))
	detailRows := min(len(detailLines), max(0, rowsAfterSectionGaps-menuRowGaps))
	introLines = append(append([]string(nil), titleLines...), detailLines[:detailRows]...)
	spacedMenuLines := make([]string, 0, len(menuLines)+menuRowGaps)
	for index, row := range menuLines {
		spacedMenuLines = append(spacedMenuLines, row)
		if index < len(menuLines)-1 && index < menuRowGaps {
			spacedMenuLines = append(spacedMenuLines, "")
		}
	}

	sections := [][]string{brandLines, introLines, spacedMenuLines, footerLines}
	lines := make([]string, 0, layout.height)
	addedGaps := 0
	lastSectionAdded := false
	for _, section := range sections {
		if len(section) == 0 {
			continue
		}
		if lastSectionAdded && addedGaps < gapRows {
			lines = append(lines, "")
			addedGaps++
		}
		lines = append(lines, section...)
		lastSectionAdded = true
	}

	for index, line := range lines {
		lines[index] = truncateDisplay(line, menuWidth)
	}
	return lipgloss.Place(layout.width, layout.height, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
}

func renderASCIIArtLine(line string) string {
	var rendered strings.Builder
	for _, char := range line {
		switch char {
		case '.':
			rendered.WriteString(menuArtAccentStyle.Render(string(char)))
		case '#':
			rendered.WriteString(menuHeaderStyle.Render(string(char)))
		default:
			rendered.WriteRune(char)
		}
	}
	return rendered.String()
}

func firstReadyRecoveryJob(jobs []app.RecoveryJob) int {
	for index, job := range jobs {
		if job.Status == app.RecoveryJobReady {
			return index
		}
	}
	return 0
}

func recoveryJobDetails(job app.RecoveryJob) []string {
	if job.ID == "" {
		return []string{"No saved jobs could be read. You can start a new run or quit."}
	}
	status := strings.ToUpper(string(job.Status))
	source := filepath.Base(strings.TrimSpace(job.SourceSummary))
	if source == "" || source == "." {
		source = "saved job"
	}
	details := []string{fmt.Sprintf("%s · %s", status, source)}
	if job.Status == app.RecoveryJobReady {
		details = append(details, fmt.Sprintf("Encode: %s · Subtitles: %s", job.CompressionState, job.SubtitleState))
	} else if job.Status == app.RecoveryJobActive {
		details = append(details, "Active in another VCX process.")
	} else if job.Status == app.RecoveryJobStale {
		details = append(details, "Saved source or plan changed. Show queue for details.")
	} else if job.Status == app.RecoveryJobInvalid {
		details = append(details, "Saved job cannot be resumed. Show queue for details.")
	}
	if job.Status == app.RecoveryJobReady && job.Detail != "" {
		details = append(details, job.Detail)
	}
	return details
}

func updateMenuCursor(cursor, length, delta int) int {
	if length == 0 {
		return 0
	}
	return (cursor + delta + length) % length
}

func findMenuItem(items []workflowMenuItem, key rune) (workflowMenuItem, bool) {
	for _, item := range items {
		if item.key == key {
			return item, true
		}
	}
	return workflowMenuItem{}, false
}

func isMenuUp(key tea.KeyMsg) bool {
	return key.Type == tea.KeyUp || (key.Type == tea.KeyRunes && strings.EqualFold(string(key.Runes), "k"))
}

func isMenuDown(key tea.KeyMsg) bool {
	return key.Type == tea.KeyDown || (key.Type == tea.KeyRunes && strings.EqualFold(string(key.Runes), "j"))
}

func (model *Model) homeMenuView() string {
	workflow := model.workflow
	items := homeMenuItems(len(workflow.recoveryJobs) > 0 || workflow.recoveryError != "")
	workflow.menuCursor = min(workflow.menuCursor, len(items)-1)
	details := []string(nil)
	if workflow.error != "" {
		details = []string{"ERROR: " + workflow.error}
	}
	footer := []string{"Up/Down select · Enter open", "Letter shortcuts are shown in rows"}
	if newWorkflowLayout(model.width, model.height).compact {
		footer = []string{"Arrows/Enter select", "Esc/q quit"}
	}
	return renderWorkflowMenu(model.width, model.height, "Home", "",
		details, items, workflow.menuCursor, footer)
}

func (model *Model) recoveryLandingView() string {
	workflow := model.workflow
	items := recoveryMenuItems(model.selectedRecoveryJobReady())
	workflow.recoveryMenuCursor = min(workflow.recoveryMenuCursor, len(items)-1)
	var details []string
	if len(workflow.recoveryJobs) == 0 {
		if workflow.recoveryError != "" {
			details = []string{"ERROR: " + workflow.recoveryError}
		} else {
			details = []string{"No saved jobs were found."}
		}
	} else if workflow.recoveryCursor < len(workflow.recoveryJobs) {
		details = recoveryJobDetails(workflow.recoveryJobs[workflow.recoveryCursor])
	}
	if workflow.error != "" {
		details = []string{"ERROR: " + workflow.error}
	}
	footer := []string{"Up/Down select · Enter open", "Esc back · q quit"}
	if newWorkflowLayout(model.width, model.height).compact {
		footer = []string{"Arrows/Enter select", "Esc back · q quit"}
	}
	return renderWorkflowMenu(model.width, model.height, "Resume Previous Session",
		"Previous session work was interrupted.", details, items, workflow.recoveryMenuCursor,
		footer)
}

func (model *Model) updateHomeKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	items := homeMenuItems(len(workflow.recoveryJobs) > 0 || workflow.recoveryError != "")
	workflow.menuCursor = min(workflow.menuCursor, len(items)-1)
	if isMenuUp(key) {
		workflow.menuCursor = updateMenuCursor(workflow.menuCursor, len(items), -1)
		return model, nil
	}
	if isMenuDown(key) {
		workflow.menuCursor = updateMenuCursor(workflow.menuCursor, len(items), 1)
		return model, nil
	}
	if key.Type == tea.KeyEsc || key.Type == tea.KeyCtrlC {
		return model, tea.Quit
	}
	if key.Type == tea.KeyEnter {
		return model, model.activateHomeMenuItem(items[workflow.menuCursor])
	}
	if key.Type != tea.KeyRunes {
		return model, nil
	}
	value := strings.ToLower(string(key.Runes))
	if value == "q" {
		return model, tea.Quit
	}
	if len(value) != 1 {
		return model, nil
	}
	item, found := findMenuItem(items, rune(value[0]))
	if !found {
		return model, nil
	}
	return model, model.activateHomeMenuItem(item)
}

func (model *Model) activateHomeMenuItem(item workflowMenuItem) tea.Cmd {
	workflow := model.workflow
	workflow.error = ""
	switch item.action {
	case menuStartRun:
		workflow.phase = workflowQuickSetup
	case menuBrowseMedia:
		return model.openMediaPicker()
	case menuResumeSession:
		workflow.phase = workflowRecoveryHome
		workflow.recoveryMenuCursor = 0
	case menuSettings:
		workflow.settingsReturnPhase = workflowHome
		workflow.phase = workflowAdvanced
		workflow.focused = 0
	case menuQuit:
		return tea.Quit
	}
	return nil
}

func (model *Model) updateRecoveryHomeKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	workflow := model.workflow
	items := recoveryMenuItems(model.selectedRecoveryJobReady())
	if isMenuUp(key) {
		workflow.recoveryMenuCursor = updateMenuCursor(workflow.recoveryMenuCursor, len(items), -1)
		return model, nil
	}
	if isMenuDown(key) {
		workflow.recoveryMenuCursor = updateMenuCursor(workflow.recoveryMenuCursor, len(items), 1)
		return model, nil
	}
	if key.Type == tea.KeyEsc {
		workflow.phase = workflowHome
		workflow.error = ""
		return model, nil
	}
	if key.Type == tea.KeyCtrlC {
		return model, tea.Quit
	}
	if key.Type == tea.KeyEnter {
		return model, model.activateRecoveryMenuItem(items[workflow.recoveryMenuCursor])
	}
	if key.Type != tea.KeyRunes {
		return model, nil
	}
	value := strings.ToLower(string(key.Runes))
	if value == "q" {
		return model, tea.Quit
	}
	if len(value) != 1 {
		return model, nil
	}
	item, found := findMenuItem(items, rune(value[0]))
	if !found {
		return model, nil
	}
	return model, model.activateRecoveryMenuItem(item)
}

func (model *Model) selectedRecoveryJobReady() bool {
	workflow := model.workflow
	return workflow.recoveryCursor >= 0 && workflow.recoveryCursor < len(workflow.recoveryJobs) &&
		workflow.recoveryJobs[workflow.recoveryCursor].Status == app.RecoveryJobReady
}

func (model *Model) activateRecoveryMenuItem(item workflowMenuItem) tea.Cmd {
	workflow := model.workflow
	workflow.error = ""
	if item.disabled {
		workflow.error = "No selected saved job is ready to resume. Choose Show queue to inspect job states."
		return nil
	}
	switch item.action {
	case menuResumeSession:
		if model.startResume == nil {
			workflow.error = "Job recovery is unavailable."
			return nil
		}
		job := workflow.recoveryJobs[workflow.recoveryCursor]
		workflow.phase = workflowPreparing
		workflow.error = "Resuming saved job…"
		model.startResume(job.ID)
	case menuShowRecoveryQueue:
		workflow.phase = workflowRecovery
		workflow.error = workflow.recoveryError
		model.showRecoverySelection()
	case menuSettings:
		workflow.settingsReturnPhase = workflowRecoveryHome
		workflow.phase = workflowAdvanced
		workflow.focused = 0
	case menuCancelAndPreserve:
		workflow.phase = workflowHome
		workflow.menuCursor = 0
		workflow.error = ""
	}
	return nil
}
