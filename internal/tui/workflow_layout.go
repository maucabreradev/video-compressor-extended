package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	ellipsis := "…"
	limit := max(0, width-lipgloss.Width(ellipsis))
	var result strings.Builder
	used := 0
	for _, character := range value {
		part := string(character)
		partWidth := lipgloss.Width(part)
		if used+partWidth > limit {
			break
		}
		result.WriteString(part)
		used += partWidth
	}
	return result.String() + ellipsis
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
