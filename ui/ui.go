package ui

import (
	"fmt"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dotcommander/crawler/internal/utils"
)

// UIMode represents the different UI display modes
type UIMode string

const (
	ModeSimple   UIMode = "simple"   // Direct terminal output (no Bubbletea)
	ModeStandard UIMode = "standard" // Basic Bubbletea UI
	ModeEnhanced UIMode = "enhanced" // Full-featured Bubbletea UI with viewport
)

// Message types for Bubbletea
type (
	// TickMsg is sent every tick for animations
	TickMsg time.Time

	// StatsMsg updates statistics
	StatsMsg struct {
		PagesVisited    int64
		PagesFailed     int64
		PDFsDownloaded  int64
		BytesDownloaded int64
		QueueSize       int
		ActiveWorkers   int
	}

	// LogMsg adds a log entry
	LogMsg struct {
		Level   string
		Message string
	}

	// WorkerMsg updates worker status
	WorkerMsg struct {
		ID     int
		Status string
		URL    string
	}

	// DoneMsg signals crawl completion
	DoneMsg struct{}
)

// LogEntry represents a single log entry
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

// WorkerStatus represents the current status of a worker
type WorkerStatus struct {
	Status string
	URL    string
}

// UI is the interface that all UI implementations satisfy
type UI interface {
	tea.Model
	IsBubbletea() bool
	PrintHeader()
	UpdateStats(StatsMsg)
	PrintLog(string, string)
	PrintFinalStats(StatsMsg)
}

// NewUnifiedUI creates a UI instance in the specified mode and returns the UI interface
func NewUnifiedUI(mode UIMode, startURL, outputDir string, maxDepth, concurrency int) UI {
	switch mode {
	case ModeSimple:
		return newSimpleUI(startURL, outputDir, maxDepth, concurrency)
	case ModeStandard:
		return newStandardUI(startURL, outputDir, maxDepth, concurrency)
	default:
		return newEnhancedUI(startURL, outputDir, maxDepth, concurrency)
	}
}

// tickCmd creates a tick command for animations
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// getLogIcon returns an icon string for a given log level
func getLogIcon(level string) string {
	switch level {
	case "ERROR":
		return "✗"
	case "WARN":
		return "⚠"
	case "SUCCESS":
		return "✓"
	default:
		return "•"
	}
}

// formatBytes is a wrapper for utils.FormatBytes
func formatBytes(bytes int64) string {
	return utils.FormatBytes(bytes, true)
}

// formatDuration is a wrapper for utils.FormatDuration
func formatDuration(d time.Duration) string {
	return utils.FormatDuration(d)
}

// baseUI holds shared Bubbletea state used by StandardUI and EnhancedUI
type baseUI struct {
	// Configuration
	startURL    string
	outputDir   string
	maxDepth    int
	concurrency int

	// Shared state
	stats     StatsMsg
	workers   map[int]WorkerStatus
	logs      []LogEntry
	startTime time.Time
	done      bool
	mu        sync.RWMutex

	// Animation state
	animFrame int
	spinner   spinner.Model

	// Terminal dimensions
	width, height int
	ready         bool
}

// initBaseUI populates an already-allocated baseUI in place, avoiding
// copying a sync.RWMutex by value.
func initBaseUI(b *baseUI, startURL, outputDir string, maxDepth, concurrency int) {
	b.startURL = startURL
	b.outputDir = outputDir
	b.maxDepth = maxDepth
	b.concurrency = concurrency
	b.workers = make(map[int]WorkerStatus)
	b.logs = make([]LogEntry, 0, 100)
	b.startTime = time.Now()

	for i := 0; i < concurrency; i++ {
		b.workers[i] = WorkerStatus{Status: "idle", URL: ""}
	}

	b.spinner = spinner.New()
	b.spinner.Spinner = spinner.Spinner{
		Frames: SpinnerFrames,
		FPS:    time.Second / 10,
	}
	b.spinner.Style = SpinnerStyle
}

// updateStatsInternal updates stats under a write lock
func (b *baseUI) updateStatsInternal(stats StatsMsg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stats = stats
}

// addLog appends a log entry, keeping at most 100 entries
func (b *baseUI) addLog(level, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logs = append(b.logs, LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	})

	if len(b.logs) > 100 {
		b.logs = b.logs[len(b.logs)-100:]
	}
}

// updateWorker updates a worker's status by ID
func (b *baseUI) updateWorker(id int, status, url string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.workers[id]; exists {
		b.workers[id] = WorkerStatus{
			Status: status,
			URL:    url,
		}
	}
}

// calculateProgress returns a 0–1 progress fraction (caller must hold at least RLock)
func (b *baseUI) calculateProgress() float64 {
	total := b.stats.PagesVisited + b.stats.PagesFailed + int64(b.stats.QueueSize)
	if total == 0 {
		return 0
	}
	completed := b.stats.PagesVisited + b.stats.PagesFailed
	return float64(completed) / float64(total)
}

// renderFinalStats renders the completion summary for Bubbletea modes
func (b *baseUI) renderFinalStats() string {
	elapsed := time.Since(b.startTime)
	pagesPerSecond := float64(b.stats.PagesVisited) / elapsed.Seconds()

	title := TitleStyle.Render(" 🎉 CRAWL COMPLETE ")

	stats := []string{
		"",
		fmt.Sprintf("📁 Output Directory: %s", b.outputDir),
		fmt.Sprintf("⏱  Time Elapsed: %s", utils.FormatDuration(elapsed)),
		fmt.Sprintf("📄 Pages Visited: %d", b.stats.PagesVisited),
		fmt.Sprintf("❌ Pages Failed: %d", b.stats.PagesFailed),
		fmt.Sprintf("📑 PDFs Downloaded: %d", b.stats.PDFsDownloaded),
		fmt.Sprintf("💾 Data Downloaded: %s", utils.FormatBytes(b.stats.BytesDownloaded, true)),
		fmt.Sprintf("⚡ Pages/Second: %.2f", pagesPerSecond),
		"",
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		BoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, stats...)),
	)

	return content
}
