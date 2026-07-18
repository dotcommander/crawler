package ui

import (
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Simple mode ANSI color constants
const (
	reset                = "\033[0m"
	primaryColorSimple   = "\033[38;5;46m"  // Bright green
	secondaryColorSimple = "\033[38;5;82m"  // Yellow-green
	accentColorSimple    = "\033[38;5;201m" // Hot pink
	warningColorSimple   = "\033[38;5;214m" // Orange
	errorColorSimple     = "\033[38;5;196m" // Bright red
	successColorSimple   = "\033[38;5;46m"  // Green
	dimColorSimple       = "\033[38;5;240m" // Gray
)

// Simple mode style render helpers
func primaryColorSimpleRender(text string) string {
	return primaryColorSimple + text + reset
}

func secondaryColorSimpleRender(text string) string {
	return secondaryColorSimple + text + reset
}

func accentColorSimpleRender(text string) string {
	return accentColorSimple + text + reset
}

func warningColorSimpleRender(text string) string {
	return warningColorSimple + text + reset
}

func errorColorSimpleRender(text string) string {
	return errorColorSimple + text + reset
}

func successColorSimpleRender(text string) string {
	return successColorSimple + text + reset
}

func dimColorSimpleRender(text string) string {
	return dimColorSimple + text + reset
}

// styleWrapper wraps a render function to expose a Render method
type styleWrapper struct {
	renderFunc func(string) string
}

func (s styleWrapper) Render(text string) string {
	return s.renderFunc(text)
}

// Simple mode style wrappers
var (
	successStyleSimple = styleWrapper{successColorSimpleRender}
	errorStyleSimple   = styleWrapper{errorColorSimpleRender}
	warningStyleSimple = styleWrapper{warningColorSimpleRender}
)

// SimpleUI implements UI for direct terminal output without Bubbletea
type SimpleUI struct {
	startURL    string
	outputDir   string
	maxDepth    int
	concurrency int

	stats        StatsMsg
	startTime    time.Time
	spinnerFrame int
	lastUpdate   time.Time
	mu           sync.Mutex
}

func newSimpleUI(startURL, outputDir string, maxDepth, concurrency int) *SimpleUI {
	return &SimpleUI{
		startURL:    startURL,
		outputDir:   outputDir,
		maxDepth:    maxDepth,
		concurrency: concurrency,
		startTime:   time.Now(),
	}
}

// tea.Model interface — no-ops for simple mode

func (ui *SimpleUI) Init() tea.Cmd {
	return nil
}

func (ui *SimpleUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return ui, nil
}

func (ui *SimpleUI) View() tea.View {
	return tea.NewView("")
}

// IsBubbletea returns false — simple mode uses direct terminal output
func (ui *SimpleUI) IsBubbletea() bool {
	return false
}

// PrintHeader clears the screen and prints the ASCII art header
func (ui *SimpleUI) PrintHeader() {
	fmt.Print("\033[2J\033[H")

	fmt.Println(primaryColorSimpleRender(`
 ╦ ╦┌─┐┌┐   ╔═╗┬─┐┌─┐┬ ┬┬  ┌─┐┬─┐
 ║║║├┤ ├┴┐  ║  ├┬┘├─┤││││  ├┤ ├┬┘
 ╚╩╝└─┘└─┘  ╚═╝┴└─┴ ┴└┴┘┴─┘└─┘┴└─`))

	fmt.Printf("\n%s %s\n", secondaryColorSimpleRender("Target URL:"), ui.startURL)
	fmt.Printf("%s %s\n", secondaryColorSimpleRender("Output Dir:"), ui.outputDir)
	fmt.Printf("%s %d | %s %d\n\n",
		secondaryColorSimpleRender("Max Depth:"), ui.maxDepth,
		secondaryColorSimpleRender("Concurrency:"), ui.concurrency)
}

// UpdateStats prints live statistics to the terminal
func (ui *SimpleUI) UpdateStats(stats StatsMsg) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Throttle updates
	now := time.Now()
	if now.Sub(ui.lastUpdate) < 100*time.Millisecond {
		return
	}
	ui.lastUpdate = now

	ui.stats = stats
	ui.spinnerFrame = (ui.spinnerFrame + 1) % len(SciFiSpinner)

	// Move cursor up and clear lines
	fmt.Print("\033[9A")

	// Calculate progress
	progress := ui.calculateProgress()

	// Render stats
	fmt.Printf("\033[K%s Crawling in progress...\n\n", accentColorSimpleRender(SciFiSpinner[ui.spinnerFrame]))
	fmt.Printf("\033[K%s\n", primaryColorSimpleRender("┌─ Statistics ─────────────────────────────────┐"))
	fmt.Printf("\033[K%s Pages: %s Failed: %s PDFs: %s  %s\n",
		primaryColorSimpleRender("│"),
		successStyleSimple.Render(fmt.Sprintf("%-6d", stats.PagesVisited)),
		errorStyleSimple.Render(fmt.Sprintf("%-6d", stats.PagesFailed)),
		warningStyleSimple.Render(fmt.Sprintf("%-6d", stats.PDFsDownloaded)),
		primaryColorSimpleRender("│"))
	fmt.Printf("\033[K%s Queue: %s Active: %s Size: %s %s\n",
		primaryColorSimpleRender("│"),
		accentColorSimpleRender(fmt.Sprintf("%-6d", stats.QueueSize)),
		secondaryColorSimpleRender(fmt.Sprintf("%-6d", stats.ActiveWorkers)),
		dimColorSimpleRender(fmt.Sprintf("%-10s", formatBytes(stats.BytesDownloaded))),
		primaryColorSimpleRender("│"))
	fmt.Printf("\033[K%s\n\n", primaryColorSimpleRender("└──────────────────────────────────────────────┘"))

	// Progress bar
	fmt.Printf("\033[K%s\n", ui.renderProgressBar(progress))
	fmt.Printf("\033[K\n")
}

// PrintLog prints a single log entry to the terminal
func (ui *SimpleUI) PrintLog(level, message string) {
	// Save cursor position, print log, restore cursor
	fmt.Print("\033[s")

	timestamp := time.Now().Format("15:04:05")
	icon := getLogIcon(level)

	var styledMessage string
	switch level {
	case "ERROR":
		styledMessage = errorStyleSimple.Render(fmt.Sprintf("%s %s", icon, message))
	case "WARN":
		styledMessage = warningStyleSimple.Render(fmt.Sprintf("%s %s", icon, message))
	case "SUCCESS":
		styledMessage = successStyleSimple.Render(fmt.Sprintf("%s %s", icon, message))
	default:
		styledMessage = dimColorSimpleRender(fmt.Sprintf("%s %s", icon, message))
	}

	fmt.Printf("\n%s %s\n",
		dimColorSimpleRender(fmt.Sprintf("[%s]", timestamp)),
		styledMessage)

	fmt.Print("\033[u")
}

// PrintFinalStats prints the completion summary to the terminal
func (ui *SimpleUI) PrintFinalStats(stats StatsMsg) {
	duration := time.Since(ui.startTime)

	fmt.Print("\033[2J\033[H") // Clear screen
	fmt.Println(successStyleSimple.Render(`
 ╔═╗┬─┐┌─┐┬ ┬┬    ╔═╗┌─┐┌┬┐┌─┐┬  ┌─┐┌┬┐┌─┐
 ║  ├┬┘├─┤││││    ║  │ ││││├─┘│  ├┤  │ ├┤
 ╚═╝┴└─┴ ┴└┴┘┴─┘  ╚═╝└─┘┴ ┴┴  ┴─┘└─┘ ┴ └─┘`))

	fmt.Printf("\n%s\n", secondaryColorSimpleRender("📊 Final Statistics"))
	fmt.Printf("├─ Pages visited: %s\n", successStyleSimple.Render(fmt.Sprintf("%d", stats.PagesVisited)))
	fmt.Printf("├─ Pages failed: %s\n", errorStyleSimple.Render(fmt.Sprintf("%d", stats.PagesFailed)))
	fmt.Printf("├─ PDFs downloaded: %s\n", warningStyleSimple.Render(fmt.Sprintf("%d", stats.PDFsDownloaded)))
	fmt.Printf("├─ Total size: %s\n", accentColorSimpleRender(formatBytes(stats.BytesDownloaded)))
	fmt.Printf("└─ Duration: %s\n", primaryColorSimpleRender(formatDuration(duration)))
}

// calculateProgress returns a 0–1 progress fraction
func (ui *SimpleUI) calculateProgress() float64 {
	total := ui.stats.PagesVisited + ui.stats.PagesFailed + int64(ui.stats.QueueSize)
	if total == 0 {
		return 0
	}
	completed := ui.stats.PagesVisited + ui.stats.PagesFailed
	return float64(completed) / float64(total)
}

// renderProgressBar renders an ASCII progress bar for simple mode
func (ui *SimpleUI) renderProgressBar(progress float64) string {
	width := 40
	filled := int(progress * float64(width))
	bar := ""

	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return fmt.Sprintf("%s [%s] %.1f%%",
		primaryColorSimpleRender("Progress:"), bar, progress*100)
}
