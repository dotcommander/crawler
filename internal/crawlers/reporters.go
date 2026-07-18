package crawlers

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/dotcommander/crawler/ui"
)

// LogReporter sends progress to the standard logger (for headless operation)
type LogReporter struct{}

func (r *LogReporter) Log(level, message string) {
	log.Printf("[%s] %s", level, message)
}

func (r *LogReporter) UpdateStats(stats ui.StatsMsg) {
	// Log progress every so often
	if stats.PagesVisited%10 == 0 {
		log.Printf("Progress: %d pages visited, %d failed, %d PDFs downloaded, %d queued",
			stats.PagesVisited, stats.PagesFailed, stats.PDFsDownloaded, stats.QueueSize)
	}
}

func (r *LogReporter) UpdateWorker(workerID int, status, url string) {
	// Only log important worker status changes
	switch status {
	case "starting", "finished", "cancelled":
		log.Printf("Worker %d: %s", workerID, status)
	}
}

// UIReporter sends progress updates to the BubbleTea program
type UIReporter struct {
	program *tea.Program
}

func NewUIReporter(program *tea.Program) *UIReporter {
	return &UIReporter{program: program}
}

func (r *UIReporter) Log(level, message string) {
	r.program.Send(ui.LogMsg{
		Level:   level,
		Message: message,
	})
}

func (r *UIReporter) UpdateStats(stats ui.StatsMsg) {
	r.program.Send(stats)
}

func (r *UIReporter) UpdateWorker(workerID int, status, url string) {
	r.program.Send(ui.WorkerMsg{
		ID:     workerID,
		Status: status,
		URL:    url,
	})
}

// UnifiedUIReporter sends progress to the unified UI implementation
type UnifiedUIReporter struct {
	ui         ui.UI
	program    *tea.Program
	simpleMode bool
}

func (r *UnifiedUIReporter) Log(level, message string) {
	if r.simpleMode {
		r.ui.PrintLog(level, message)
	} else {
		r.program.Send(ui.LogMsg{
			Level:   level,
			Message: message,
		})
	}
}

func (r *UnifiedUIReporter) UpdateStats(stats ui.StatsMsg) {
	if r.simpleMode {
		r.ui.UpdateStats(stats)
	} else {
		r.program.Send(stats)
	}
}

func (r *UnifiedUIReporter) UpdateWorker(workerID int, status, url string) {
	if !r.simpleMode {
		r.program.Send(ui.WorkerMsg{
			ID:     workerID,
			Status: status,
			URL:    url,
		})
	}
}
