package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dotcommander/crawler/internal/utils"
)

// StandardUI implements UI with the standard Bubble Tea presentation.
type StandardUI struct {
	baseUI
}

func newStandardUI(startURL, outputDir string, maxDepth, concurrency int) *StandardUI {
	ui := &StandardUI{}
	initBaseUI(&ui.baseUI, startURL, outputDir, maxDepth, concurrency)
	return ui
}

// Init returns the initial Bubbletea commands for standard mode
func (ui *StandardUI) Init() tea.Cmd {
	return tea.Batch(ui.spinner.Tick, tickCmd())
}

// Update handles incoming Bubbletea messages
func (ui *StandardUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ui.width = msg.Width
		ui.height = msg.Height
		ui.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return ui, tea.Quit
		}

	case TickMsg:
		ui.animFrame++
		return ui, tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		ui.spinner, cmd = ui.spinner.Update(msg)
		return ui, cmd

	case StatsMsg:
		ui.updateStatsInternal(msg)

	case LogMsg:
		ui.addLog(msg.Level, msg.Message)

	case WorkerMsg:
		ui.updateWorker(msg.ID, msg.Status, msg.URL)

	case DoneMsg:
		ui.done = true
	}

	return ui, nil
}

// View renders the standard Bubbletea UI
func (ui *StandardUI) View() tea.View {
	view := tea.NewView(ui.renderStandardView())
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

// IsBubbletea returns true — standard mode uses Bubbletea
func (ui *StandardUI) IsBubbletea() bool {
	return true
}

// PrintHeader is a no-op for Bubbletea modes
func (ui *StandardUI) PrintHeader() {}

// UpdateStats is a no-op for Bubbletea modes (handled via Update)
func (ui *StandardUI) UpdateStats(_ StatsMsg) {}

// PrintLog is a no-op for Bubbletea modes (handled via Update)
func (ui *StandardUI) PrintLog(_, _ string) {}

// PrintFinalStats is a no-op for Bubbletea modes (handled via Update)
func (ui *StandardUI) PrintFinalStats(_ StatsMsg) {}

// renderStandardView assembles the full standard-mode view
func (ui *StandardUI) renderStandardView() string {
	if !ui.ready {
		return "\n  Initializing..."
	}

	sections := []string{
		ui.renderHeader(),
		ui.renderStats(),
		ui.renderWorkers(),
		ui.renderLogs(),
	}

	if ui.done {
		sections = append(sections, ui.renderFinalStats())
	} else {
		sections = append(sections, ui.renderHelp())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (ui *StandardUI) renderHeader() string {
	elapsed := time.Since(ui.startTime)

	title := " 🛸 HYPERCRAWLER v1.0 "
	subtitle := fmt.Sprintf("Scanning: %s", ui.startURL)
	timer := fmt.Sprintf("⏱  %s", utils.FormatDuration(elapsed))

	titleBox := TitleStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			URLStyle.Render(subtitle),
			timer,
		),
	)

	return titleBox
}

func (ui *StandardUI) renderStats() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	var rows []string

	// Progress bar for standard mode
	progress := ui.calculateProgress()
	progressBar := ui.renderProgressBar(progress)
	rows = append(rows, progressBar)
	rows = append(rows, "")

	// Stats grid
	stats := [][]string{
		{
			StatLabelStyle.Render("Pages Visited:"),
			StatValueStyle.Render(fmt.Sprintf("%d", ui.stats.PagesVisited)),
			StatLabelStyle.Render("Queue Size:"),
			StatValueStyle.Render(fmt.Sprintf("%d", ui.stats.QueueSize)),
		},
		{
			StatLabelStyle.Render("Pages Failed:"),
			StatValueStyle.Render(fmt.Sprintf("%d", ui.stats.PagesFailed)),
			StatLabelStyle.Render("Active Workers:"),
			StatValueStyle.Render(fmt.Sprintf("%d", ui.stats.ActiveWorkers)),
		},
		{
			StatLabelStyle.Render("PDFs Downloaded:"),
			StatValueStyle.Render(fmt.Sprintf("%d", ui.stats.PDFsDownloaded)),
			StatLabelStyle.Render("Data Downloaded:"),
			StatValueStyle.Render(utils.FormatBytes(ui.stats.BytesDownloaded, true)),
		},
	}

	for _, row := range stats {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return BoxStyle.Render(content)
}

func (ui *StandardUI) renderWorkers() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	var lines []string
	lines = append(lines, StatusStyle.Render("⚡ WORKER STATUS"))
	lines = append(lines, "")

	for i := 0; i < ui.concurrency; i++ {
		worker := ui.workers[i]
		icon := SpinnerFrames[ui.animFrame%len(SpinnerFrames)]
		active := worker.Status != "idle" && worker.Status != "done"
		if !active {
			icon = "○"
		}

		status := fmt.Sprintf("%s Worker %d: %s", icon, i, worker.Status)
		if worker.URL != "" {
			status += fmt.Sprintf(" - %s", utils.TruncateURL(worker.URL, 50))
		}

		style := LogStyle
		if active {
			style = SuccessStyle
		}

		lines = append(lines, style.Render(status))
	}

	return BoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (ui *StandardUI) renderLogs() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	var lines []string
	lines = append(lines, StatusStyle.Render("📡 ACTIVITY LOG"))
	lines = append(lines, "")

	// Display last 10 logs
	displayLogs := ui.logs
	if len(displayLogs) > 10 {
		displayLogs = displayLogs[len(displayLogs)-10:]
	}

	for _, log := range displayLogs {
		style := LogStyle
		icon := "•"

		switch log.Level {
		case "ERROR":
			style = ErrorStyle
			icon = "✗"
		case "WARN":
			style = WarningStyle
			icon = "!"
		case "SUCCESS":
			style = SuccessStyle
			icon = "✓"
		}

		entry := fmt.Sprintf("%s %s %s",
			log.Time.Format("15:04:05"),
			icon,
			log.Message,
		)
		lines = append(lines, style.Render(entry))
	}

	if len(lines) == 2 {
		lines = append(lines, LogStyle.Render("Waiting for activity..."))
	}

	return BoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (ui *StandardUI) renderHelp() string {
	return HelpStyle.Render("Press 'q' or 'ctrl+c' to quit")
}

// renderProgressBar renders a styled progress bar for standard mode
func (ui *StandardUI) renderProgressBar(progress float64) string {
	width := 50
	filled := int(progress * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percentage := fmt.Sprintf(" %3.0f%%", progress*100)

	return ProgressBarStyle.Render(bar) + percentage
}
