package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dotcommander/crawler/internal/utils"
)

// EnhancedUI implements the full-featured Bubbletea UI with alt-screen and viewport
type EnhancedUI struct {
	baseUI

	progressBar  progress.Model
	viewport     viewport.Model
	helpViewport viewport.Model
	showHelp     bool
	focusedPane  int
}

func newEnhancedUI(startURL, outputDir string, maxDepth, concurrency int) *EnhancedUI {
	eui := &EnhancedUI{
		progressBar: progress.New(),
	}
	eui.progressBar.ShowPercentage = true
	initBaseUI(&eui.baseUI, startURL, outputDir, maxDepth, concurrency)
	return eui
}

// Init returns the initial Bubbletea commands for enhanced mode (enters alt screen)
func (ui *EnhancedUI) Init() tea.Cmd {
	return tea.Batch(ui.spinner.Tick, tickCmd())
}

// Update handles incoming Bubbletea messages
func (ui *EnhancedUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ui.handleWindowResize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if ui.showHelp {
			return ui.handleHelpKeys(msg)
		}
		return ui.handleMainKeys(msg)

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

// View renders the enhanced Bubbletea UI
func (ui *EnhancedUI) View() tea.View {
	view := tea.NewView(ui.renderEnhancedView())
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

// IsBubbletea returns true — enhanced mode uses Bubbletea
func (ui *EnhancedUI) IsBubbletea() bool {
	return true
}

// PrintHeader is a no-op for Bubbletea modes
func (ui *EnhancedUI) PrintHeader() {}

// UpdateStats is a no-op for Bubbletea modes (handled via Update)
func (ui *EnhancedUI) UpdateStats(_ StatsMsg) {}

// PrintLog is a no-op for Bubbletea modes (handled via Update)
func (ui *EnhancedUI) PrintLog(_, _ string) {}

// PrintFinalStats is a no-op for Bubbletea modes (handled via Update)
func (ui *EnhancedUI) PrintFinalStats(_ StatsMsg) {}

func (ui *EnhancedUI) handleWindowResize(width, height int) {
	ui.width = width
	ui.height = height
	ui.ready = true

	headerHeight := 10
	footerHeight := 3
	contentHeight := height - headerHeight - footerHeight

	if !ui.showHelp {
		ui.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(contentHeight))
		ui.viewport.SetContent(ui.renderMainContent())
	} else {
		ui.helpViewport = viewport.New(viewport.WithWidth(width-4), viewport.WithHeight(height-6))
		ui.helpViewport.SetContent(ui.renderHelpContent())
	}
}

func (ui *EnhancedUI) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return ui, tea.Quit
	case "?", "h":
		ui.showHelp = true
	case "tab":
		ui.focusedPane = (ui.focusedPane + 1) % 2
	}
	return ui, nil
}

func (ui *EnhancedUI) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?", "h":
		ui.showHelp = false
		return ui, nil
	}

	// Handle viewport scrolling
	var cmd tea.Cmd
	ui.helpViewport, cmd = ui.helpViewport.Update(msg)
	return ui, cmd
}

func (ui *EnhancedUI) renderEnhancedView() string {
	if !ui.ready {
		return "\n  Initializing..."
	}

	if ui.showHelp {
		return ui.renderHelpView()
	}

	header := ui.renderEnhancedHeader()
	footer := ui.renderFooter()

	if ui.done {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			ui.renderFinalStats(),
			footer,
		)
	}

	// Update viewport content
	ui.viewport.SetContent(ui.renderMainContent())

	// Progress section for enhanced mode
	progressSection := ui.renderProgressSection()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		progressSection,
		ui.viewport.View(),
		footer,
	)
}

func (ui *EnhancedUI) renderEnhancedHeader() string {
	elapsed := time.Since(ui.startTime)
	sparkleFrames := []string{"✦", "✧", "✦", "✧", "⬥", "◆", "◇"}
	sparkle := sparkleFrames[ui.animFrame%len(sparkleFrames)]

	title := fmt.Sprintf("%s HYPERCRAWLER v2.0 %s", sparkle, sparkle)
	subtitle := fmt.Sprintf("Scanning: %s", utils.TruncateURL(ui.startURL, 60))
	timer := fmt.Sprintf("⏱  Elapsed: %s", utils.FormatDuration(elapsed))

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff41")).
		Background(lipgloss.Color("#0a0a0a")).
		Padding(0, 2).
		Render(title)

	subtitleStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff006e")).
		Italic(true).
		Render(subtitle)

	timerStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#39ff14")).
		Render(timer)

	header := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyled,
		subtitleStyled,
		timerStyled,
	)

	return lipgloss.NewStyle().
		Width(ui.width).
		Align(lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#00ff41")).
		Render(header)
}

func (ui *EnhancedUI) renderMainContent() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.renderEnhancedStats(),
		"",
		ui.renderEnhancedWorkers(),
		"",
		ui.renderEnhancedLogs(),
	)
}

func (ui *EnhancedUI) renderProgressSection() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	prog := ui.calculateProgress()

	progressWidth := ui.width - 20
	if progressWidth < 10 {
		progressWidth = 10
	}

	ui.progressBar.SetWidth(progressWidth)
	progressView := ui.progressBar.ViewAs(prog)

	percentage := fmt.Sprintf(" %3.0f%%", prog*100)
	stats := fmt.Sprintf(" | %d pages | %d in queue", ui.stats.PagesVisited, ui.stats.QueueSize)

	progressLine := progressView +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff41")).Bold(true).Render(percentage) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#606060")).Render(stats)

	return lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Render(progressLine)
}

func (ui *EnhancedUI) renderEnhancedStats() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	statsGrid := [][]string{
		{
			ui.renderStatItem("Pages Visited", fmt.Sprintf("%d", ui.stats.PagesVisited), "success"),
			ui.renderStatItem("Queue Size", fmt.Sprintf("%d", ui.stats.QueueSize), "info"),
		},
		{
			ui.renderStatItem("Pages Failed", fmt.Sprintf("%d", ui.stats.PagesFailed), "error"),
			ui.renderStatItem("Active Workers", fmt.Sprintf("%d/%d", ui.stats.ActiveWorkers, ui.concurrency), "info"),
		},
		{
			ui.renderStatItem("PDFs Downloaded", fmt.Sprintf("%d", ui.stats.PDFsDownloaded), "success"),
			ui.renderStatItem("Data Downloaded", utils.FormatBytes(ui.stats.BytesDownloaded, true), "info"),
		},
	}

	var rows []string
	rows = append(rows, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff41")).
		Bold(true).
		Render("📊 STATISTICS"))
	rows = append(rows, "")

	for _, row := range statsGrid {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00ff41")).
		Padding(1, 2).
		Width(ui.width - 4).
		Render(content)
}

func (ui *EnhancedUI) renderStatItem(label, value, style string) string {
	var valueColor string
	switch style {
	case "success":
		valueColor = "#00ff41"
	case "error":
		valueColor = "#ff0040"
	case "warning":
		valueColor = "#ffb700"
	default:
		valueColor = "#39ff14"
	}

	labelStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Width(20).
		Render(label + ":")

	valueStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(valueColor)).
		Bold(true).
		Width(15).
		Align(lipgloss.Right).
		Render(value)

	return labelStyled + valueStyled + "  "
}

func (ui *EnhancedUI) renderEnhancedWorkers() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	var lines []string
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#39ff14")).
		Bold(true).
		Render("⚡ WORKER STATUS"))
	lines = append(lines, "")

	waveFrames := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
	wave := waveFrames[ui.animFrame%len(waveFrames)]

	for i := 0; i < ui.concurrency; i++ {
		worker := ui.workers[i]
		active := worker.Status != "idle" && worker.Status != "done"

		var icon, statusColor string
		if active {
			icon = ui.spinner.View()
			statusColor = "#00ff41"
		} else {
			icon = "○"
			statusColor = "#606060"
		}

		status := fmt.Sprintf("%s Worker %d: %s", icon, i, worker.Status)

		if active && worker.URL != "" {
			urlPart := fmt.Sprintf(" %s %s", wave, utils.TruncateURL(worker.URL, 40))
			status += lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff006e")).
				Render(urlPart)
		}

		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render(status))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#39ff14")).
		Padding(1, 2).
		Width(ui.width - 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (ui *EnhancedUI) renderEnhancedLogs() string {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	var lines []string

	title := "📡 ACTIVITY LOG"
	if ui.focusedPane == 1 {
		title += " [FOCUSED]"
	}

	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffb700")).
		Bold(true).
		Render(title))
	lines = append(lines, "")

	displayLogs := ui.logs
	maxLogs := 15
	if len(displayLogs) > maxLogs {
		displayLogs = displayLogs[len(displayLogs)-maxLogs:]
	}

	for _, log := range displayLogs {
		var style lipgloss.Style
		var icon string

		switch log.Level {
		case "ERROR":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0040"))
			icon = "✗"
		case "WARN":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb700"))
			icon = "⚠"
		case "SUCCESS":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff41"))
			icon = "✓"
		case "INFO":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#39ff14"))
			icon = "•"
		default:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#606060"))
			icon = "·"
		}

		entry := fmt.Sprintf("%s %s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#606060")).Render(log.Time.Format("15:04:05")),
			icon,
			log.Message,
		)
		lines = append(lines, style.Render(entry))
	}

	if len(lines) == 2 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true).
			Render("Waiting for activity..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	borderColor := "#ffb700"
	if ui.focusedPane == 1 {
		borderColor = "#00ff41"
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(ui.width - 4).
		Render(content)
}

func (ui *EnhancedUI) renderFooter() string {
	help := []string{
		"[q]uit",
		"[?]help",
		"[tab]switch pane",
		"[↑↓]scroll",
	}

	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Render(strings.Join(help, " • "))

	var statusText string
	if ui.stats.ActiveWorkers > 0 {
		statusText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff41")).
			Bold(true).
			Render("● CRAWLING")
	} else if ui.stats.QueueSize > 0 {
		statusText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffb700")).
			Bold(true).
			Render("● PROCESSING")
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Render("● IDLE")
	}

	spacerWidth := ui.width - lipgloss.Width(helpText) - lipgloss.Width(statusText) - 4
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	footer := lipgloss.JoinHorizontal(
		lipgloss.Top,
		helpText,
		strings.Repeat(" ", spacerWidth),
		statusText,
	)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("#606060")).
		Padding(0, 2).
		Render(footer)
}

func (ui *EnhancedUI) renderHelpView() string {
	helpBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#00ff41")).
		Padding(1, 2).
		Width(ui.width - 4).
		Height(ui.height - 4).
		Render(ui.helpViewport.View())

	return lipgloss.Place(
		ui.width,
		ui.height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}

func (ui *EnhancedUI) renderHelpContent() string {
	help := `
🛸 HYPERCRAWLER HELP

KEYBOARD SHORTCUTS:
  q, Ctrl+C    Quit the application
  ?            Toggle this help screen
  Tab          Switch between panes
  ↑/↓, j/k     Scroll current pane
  PgUp/PgDn    Page up/down

INTERFACE SECTIONS:
  • Header      Shows title, URL, and elapsed time
  • Progress    Real-time crawling progress
  • Statistics  Live metrics and counters
  • Workers     Status of concurrent workers
  • Logs        Recent activity and errors

STATUS INDICATORS:
  ● CRAWLING   Active crawling in progress
  ● PROCESSING Queue has items to process
  ● IDLE       No active operations

WORKER STATES:
  ○            Idle/waiting for work
  ◐◓◑◒         Active/processing
  ✓            Successfully completed
  ✗            Failed with error

Press '?' or ESC to close this help`

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c0c0c0")).
		Render(help)
}
