package crawlers

import "github.com/dotcommander/crawler/ui"

// ProgressReporter defines the interface for reporting crawl progress.
// This allows decoupling the core crawling logic from different UI implementations.
type ProgressReporter interface {
	// Log reports a message with the specified level
	Log(level, message string)

	// UpdateStats reports current crawling statistics
	UpdateStats(stats ui.StatsMsg)

	// UpdateWorker reports worker status changes
	UpdateWorker(workerID int, status, url string)
}

// NoOpReporter is a no-operation reporter for testing or headless operation
type NoOpReporter struct{}

func (r *NoOpReporter) Log(level, message string)                     {}
func (r *NoOpReporter) UpdateStats(stats ui.StatsMsg)                 {}
func (r *NoOpReporter) UpdateWorker(workerID int, status, url string) {}
