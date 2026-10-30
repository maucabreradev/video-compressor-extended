package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type workflowLayout struct {
	width   int
	height  int
	compact bool
	split   bool
}

func newWorkflowLayout(width, height int) workflowLayout {
	width = max(1, width)
	height = max(1, height)
	return workflowLayout{
		width:   width,
		height:  height,
		compact: width < 60 || height < 14,
		split:   width >= 100 && height >= 18,
	}
}

func (layout workflowLayout) panel(title string, content []string, width int) string {
	innerWidth := max(1, width-4)
	lines := []string{sectionStyle.Render(title)}
	for _, line := range content {
		lines = append(lines, infoStyle.Render(truncateDisplay(line, innerWidth)))
	}
	return lipgloss.NewStyle().
		Width(innerWidth).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Render(strings.Join(lines, "\n"))
}

func truncateDisplay(value string, width int) string {
	if width < 1 || lipgloss.Width(value) <= width {
		if width < 1 {
			return ""
		}
		return value
	}
	return ansi.TruncateWc(value, width, "…")
}

func wrapTerminalText(value string, width int) string {
	if width < 1 {
		return ""
	}
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = ansi.HardwrapWc(line, width, true)
	}
	return strings.Join(lines, "\n")
}

func composeResponsiveScreen(header, body, footer []string, width, height int) string {
	layout := newWorkflowLayout(width, height)
	header = append([]string(nil), header...)
	body = append([]string(nil), body...)
	footer = append([]string(nil), footer...)
	if len(header)+len(footer) > layout.height {
		keepHeader := max(1, layout.height-len(footer))
		if keepHeader < len(header) {
			header = header[:keepHeader]
		}
		if len(header)+len(footer) > layout.height {
			footer = footer[:max(0, layout.height-len(header))]
		}
	}
	bodyRows := max(0, layout.height-len(header)-len(footer))
	if len(body) > bodyRows {
		body = body[:bodyRows]
	}
	lines := append(header, body...)
	lines = append(lines, footer...)
	for index, line := range lines {
		lines[index] = truncateDisplay(line, layout.width)
	}
	return strings.Join(lines, "\n")
}

func (layout workflowLayout) controls() []string {
	if layout.compact {
		return []string{
			selectedStyle.Render("enter review · e edit"),
			dimStyle.Render("a options · r reset · q quit"),
		}
	}
	controls := "enter review · e edit input · a advanced settings · r reset preferences · q quit"
	if lipgloss.Width(controls) <= layout.width {
		return []string{dimStyle.Render(controls)}
	}
	return []string{
		selectedStyle.Render("enter review · e edit input · a advanced settings"),
		dimStyle.Render("r reset preferences · q quit"),
	}
}
