package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	sectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	successStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	warningStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	errorStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

type stageView struct {
	state scheduler.State
	err   string
}

type jobView struct {
	source        string
	compression   stageView
	transcription stageView
	progress      *process.FFmpegProgress
	durationKnown bool
	completed     bool
}

type runEventMsg struct {
	event app.RunEvent
}

type runFinishedMsg struct {
	err error
}

type externalCancelMsg struct{}
type tickMsg struct{}

// Model is the Bubble Tea state for one VCX batch run.
type Model struct {
	control         *scheduler.DispatchControl
	cancel          context.CancelFunc
	startRun        func(config.Config)
	workflow        *workflowState
	jobs            map[string]*jobView
	order           []string
	viewport        viewport.Model
	width           int
	height          int
	paused          bool
	cancelRequested bool
	finished        bool
	runErr          error
	startedAt       time.Time
	elapsed         time.Duration
}

// NewModel creates a queue model without starting terminal I/O.
func NewModel(control *scheduler.DispatchControl, cancel context.CancelFunc) *Model {
	if control == nil {
		control = scheduler.NewDispatchControl()
	}
	view := viewport.New(80, 12)
	model := &Model{
		control:  control,
		cancel:   cancel,
		jobs:     make(map[string]*jobView),
		viewport: view,
		width:    80,
		height:   20,
	}
	model.refreshViewport()
	return model
}

// Init starts the viewport's internal commands.
func (model *Model) Init() tea.Cmd {
	return tea.Batch(model.viewport.Init(), model.tick())
}

// Update applies job events and keyboard controls without doing media work.
func (model *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if model.workflow != nil && model.workflow.phase != workflowRunning {
		return model.updateWorkflow(message)
	}
	switch message := message.(type) {
	case runEventMsg:
		model.updateRunEvent(message.event)
		return model, nil
	case runFinishedMsg:
		model.finished = true
		model.runErr = message.err
		if !model.startedAt.IsZero() {
			model.elapsed = time.Since(model.startedAt)
		}
		model.refreshViewport()
		return model, nil
	case tickMsg:
		if model.finished {
			return model, nil
		}
		if !model.startedAt.IsZero() {
			model.elapsed = time.Since(model.startedAt)
		}
		model.refreshViewport()
		return model, model.tick()
	case externalCancelMsg:
		if !model.finished {
			model.cancelRequested = true
			model.refreshViewport()
		}
		return model, nil
	case tea.WindowSizeMsg:
		model.width = max(1, message.Width)
		model.height = max(1, message.Height)
		model.refreshViewport()
		return model, nil
	case tea.KeyMsg:
		return model.updateKey(message)
	default:
		var command tea.Cmd
		model.viewport, command = model.viewport.Update(message)
		return model, command
	}
}

// View renders queue state, stage progress, errors, and available controls.
func (model *Model) View() string {
	if model.workflow != nil && model.workflow.phase != workflowRunning {
		return model.workflowView()
	}
	layout := newWorkflowLayout(model.width, model.height)
	header := model.queueHeader(layout)
	controls := model.queueControls(layout)
	return composeResponsiveScreen(header, strings.Split(model.viewport.View(), "\n"), controls, layout.width, layout.height)
}

func (model *Model) updateRunEvent(event app.RunEvent) {
	if model.startedAt.IsZero() {
		model.startedAt = time.Now()
	}
	job := model.ensureJob(event.Source)
	switch event.Kind {
	case app.RunStarted:
		job.compression.state = scheduler.Running
	case app.RunProgress:
		progress := event.Progress
		job.progress = &progress
		job.durationKnown = event.DurationKnown
	case app.RunStageStateChanged:
		stage := &job.compression
		if event.Stage == scheduler.Transcription {
			stage = &job.transcription
		}
		stage.state = event.State
		if event.Err != nil {
			stage.err = event.Err.Error()
		}
	case app.RunCompleted:
		job.completed = true
	}
	model.refreshViewport()
}

func (model *Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (model *Model) updateKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlC || key.Type == tea.KeyEsc {
		if model.finished {
			return model, tea.Quit
		}
		model.requestCancel()
		return model, nil
	}
	if key.Type == tea.KeyRunes {
		switch strings.ToLower(string(key.Runes)) {
		case "p":
			if !model.finished && !model.cancelRequested {
				model.paused = !model.paused
				if model.paused {
					model.control.Pause()
				} else {
					model.control.Resume()
				}
				model.refreshViewport()
			}
			return model, nil
		case "c":
			model.requestCancel()
			return model, nil
		case "q":
			if model.finished {
				return model, tea.Quit
			}
			model.requestCancel()
			return model, nil
		}
	}

	var command tea.Cmd
	model.viewport, command = model.viewport.Update(key)
	return model, command
}

func (model *Model) requestCancel() {
	if model.finished || model.cancelRequested {
		return
	}
	model.cancelRequested = true
	if model.cancel != nil {
		model.cancel()
	}
	model.refreshViewport()
}

func (model *Model) ensureJob(source string) *jobView {
	if job := model.jobs[source]; job != nil {
		return job
	}
	job := &jobView{source: source}
	model.jobs[source] = job
	model.order = append(model.order, source)
	return job
}

func (model *Model) queueCounts() string {
	queued, running, completed, failed, cancelled := model.queueStateCounts()
	return fmt.Sprintf("Files %d · queued %d · running %d · done %d · failed %d · cancelled %d", len(model.jobs), queued, running, completed, failed, cancelled)
}

func (model *Model) queueStateCounts() (queued, running, completed, failed, cancelled int) {
	for _, job := range model.jobs {
		switch {
		case job.completed:
			completed++
		case job.compression.state == scheduler.Failed || job.transcription.state == scheduler.Failed:
			failed++
		case job.compression.state == scheduler.Cancelled || job.transcription.state == scheduler.Cancelled:
			cancelled++
		case job.compression.state == scheduler.Running || job.transcription.state == scheduler.Running:
			running++
		default:
			queued++
		}
	}
	return queued, running, completed, failed, cancelled
}

func (model *Model) stageCounts(stage scheduler.Stage) string {
	counts := map[scheduler.State]int{}
	for _, job := range model.jobs {
		state := job.compression.state
		if stage == scheduler.Transcription {
			state = job.transcription.state
		}
		if state != "" {
			counts[state]++
		}
	}
	return fmt.Sprintf("queued %d · running %d · done %d · failed %d · cancelled %d · skipped %d",
		counts[scheduler.Queued], counts[scheduler.Running], counts[scheduler.Completed],
		counts[scheduler.Failed], counts[scheduler.Cancelled], counts[scheduler.Skipped])
}

func (model *Model) refreshViewport() {
	model.resizeQueueViewport()
	lines := make([]string, 0, len(model.order)*3)
	for _, source := range model.order {
		job := model.jobs[source]
		lines = append(lines, fmt.Sprintf("%s  encode: %s  subtitles: %s",
			filepath.Base(source), stateLabel(job.compression.state), stateLabel(job.transcription.state)))
		if job.progress != nil {
			progress := *job.progress
			if job.durationKnown || progress.Done {
				lines = append(lines, fmt.Sprintf("  encode progress: %.1f%% · frame %d · %s", progress.Percent, progress.Frame, progress.OutTime))
			} else {
				lines = append(lines, fmt.Sprintf("  encode progress: frame %d · %s", progress.Frame, progress.OutTime))
			}
		}
		for _, stage := range []stageView{job.compression, job.transcription} {
			if stage.err != "" {
				lines = append(lines, errorStyle.Render("  error: "+singleLine(stage.err)))
			}
		}
	}
	if model.runErr != nil && !errors.Is(model.runErr, context.Canceled) {
		lines = append(lines, errorStyle.Render("Run error: "+singleLine(model.runErr.Error())))
	}
	if len(lines) == 0 {
		lines = append(lines, dimStyle.Render("Waiting for queue events…"))
	}
	wasAtBottom := model.viewport.AtBottom()
	previousOffset := model.viewport.YOffset
	model.viewport.SetContent(wrapTerminalText(strings.Join(lines, "\n"), model.viewport.Width))
	if wasAtBottom {
		model.viewport.GotoBottom()
	} else {
		model.viewport.SetYOffset(previousOffset)
	}
}

func (model *Model) resizeQueueViewport() {
	layout := newWorkflowLayout(model.width, model.height)
	header := model.queueHeader(layout)
	controls := model.queueControls(layout)
	model.viewport.Width = layout.width
	model.viewport.Height = max(1, layout.height-len(header)-len(controls))
	model.viewport.SetYOffset(model.viewport.YOffset)
}

func (model *Model) queueHeader(layout workflowLayout) []string {
	header := []string{titleStyle.Render("VCX · BATCH QUEUE")}
	status := model.queueStatus(layout)
	if layout.compact {
		header = append(header, fmt.Sprintf("%s · elapsed %s · %d files", status, model.elapsed.Round(time.Second), len(model.jobs)))
		if summary := model.latestQueueError(); summary != "" {
			header = append(header, errorStyle.Render("Error: "+singleLine(summary)))
		} else if model.paused {
			header = append(header, "Active processes continue")
		} else {
			header = append(header, model.queueCountsCompact())
		}
		header = append(header, model.stageCountsCompact(scheduler.Compression), model.stageCountsCompact(scheduler.Transcription))
		return header
	}
	header = append(header,
		fmt.Sprintf("%s  elapsed %s  %s", status, model.elapsed.Round(time.Second), model.queueCounts()),
		"Encode (compression): "+model.stageCounts(scheduler.Compression),
		"Subtitles (transcription): "+model.stageCounts(scheduler.Transcription),
	)
	if summary := model.latestQueueError(); summary != "" {
		header = append(header, errorStyle.Render("Error: "+singleLine(summary)))
	}
	return header
}

func (model *Model) queueStatus(layout workflowLayout) string {
	switch {
	case model.finished && errors.Is(model.runErr, context.Canceled):
		return "CANCELLED"
	case model.finished && model.runErr != nil:
		return "FINISHED WITH ERRORS"
	case model.finished:
		return "COMPLETE"
	case model.cancelRequested:
		return "CANCELLING"
	case model.paused:
		if !layout.compact {
			return "PAUSED — active processes continue"
		}
		return "PAUSED"
	default:
		return "RUNNING"
	}
}

func (model *Model) queueControls(layout workflowLayout) []string {
	if model.finished {
		return []string{dimStyle.Render("q quit · ↑/↓ scroll")}
	}
	if layout.compact {
		return []string{selectedStyle.Render("p pause · c cancel"), dimStyle.Render("q quit · ↑/↓ scroll")}
	}
	return []string{dimStyle.Render("p pause/resume · c cancel · q cancel/quit · ↑/↓ scroll")}
}

func (model *Model) queueCountsCompact() string {
	queued, running, completed, failed, cancelled := model.queueStateCounts()
	return fmt.Sprintf("Q/R/D/F/C %d/%d/%d/%d/%d", queued, running, completed, failed, cancelled)
}

func (model *Model) latestQueueError() string {
	if model.runErr != nil && !errors.Is(model.runErr, context.Canceled) {
		return model.runErr.Error()
	}
	for index := len(model.order) - 1; index >= 0; index-- {
		job := model.jobs[model.order[index]]
		if job.transcription.err != "" {
			return job.transcription.err
		}
		if job.compression.err != "" {
			return job.compression.err
		}
	}
	return ""
}

func (model *Model) stageCountsCompact(stage scheduler.Stage) string {
	counts := map[scheduler.State]int{}
	for _, job := range model.jobs {
		state := job.compression.state
		if stage == scheduler.Transcription {
			state = job.transcription.state
		}
		if state != "" {
			counts[state]++
		}
	}
	name := "Encode"
	if stage == scheduler.Transcription {
		name = "Subtitles"
	}
	line := fmt.Sprintf("%s Q/R/D/F/C: %d/%d/%d/%d/%d", name,
		counts[scheduler.Queued], counts[scheduler.Running], counts[scheduler.Completed], counts[scheduler.Failed], counts[scheduler.Cancelled])
	if stage == scheduler.Compression {
		if progress := model.latestEncodeProgress(); progress != nil {
			line += fmt.Sprintf(" · %.1f%%", progress.Percent)
		}
	}
	return line
}

func (model *Model) latestEncodeProgress() *process.FFmpegProgress {
	for index := len(model.order) - 1; index >= 0; index-- {
		if progress := model.jobs[model.order[index]].progress; progress != nil {
			return progress
		}
	}
	return nil
}

func stateLabel(state scheduler.State) string {
	if state == "" {
		return "not requested"
	}
	return string(state)
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
