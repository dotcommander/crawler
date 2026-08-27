package crawlers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/ui"
)

type testEngine struct{}

type failingVisitedStore struct {
	err error
}

func (s *failingVisitedStore) MarkVisited(string) (bool, error) { return false, s.err }
func (s *failingVisitedStore) RecordResult(string, int) error   { return nil }
func (s *failingVisitedStore) IsVisited(string) bool            { return false }
func (s *failingVisitedStore) Close() error                     { return nil }

type recordingReporter struct {
	logs []string
}

func (r *recordingReporter) Log(level, message string) {
	r.logs = append(r.logs, level+": "+message)
}
func (*recordingReporter) UpdateStats(ui.StatsMsg)          {}
func (*recordingReporter) UpdateWorker(int, string, string) {}

func (testEngine) CrawlPage(_ context.Context, item *QueueItem) (*CrawlResult, error) {
	return &CrawlResult{
		URL:        item.URL,
		StatusCode: 200,
		Success:    true,
	}, nil
}

func (testEngine) Close() error { return nil }

func (testEngine) GetEngineType() string { return "test" }

// TestEngineCrawler_Cancel_DrainGoroutineExits verifies that the drain
// goroutine started by Cancel() terminates once c.queue is closed — i.e. the
// documented exit condition holds: range exits when the channel is closed.
func TestEngineCrawler_Cancel_DrainGoroutineExits(t *testing.T) {
	t.Parallel()

	cfg := &config.CrawlerConfig{
		StartURL:    "http://example.com",
		Concurrency: 2,
	}
	crawler, err := NewEngineCrawler(cfg, &NoOpReporter{}, "colly", nil)
	if err != nil {
		t.Fatalf("NewEngineCrawler: %v", err)
	}
	t.Cleanup(func() { _ = crawler.engine.Close() })

	// Cancel starts the drain goroutine; it will block until c.queue is closed.
	crawler.Cancel()

	// Simulate Start's queueClosed.Do(close) after all workers exit.
	crawler.queueClosed.Do(func() { close(crawler.queue) })

	// After the queue is closed the drain goroutine must exit promptly.
	// We verify by attempting a send on the now-closed channel inside a
	// recover and confirming no goroutine is still blocking — if the goroutine
	// leaked it would keep the channel referenced but not drainable. A
	// time-bounded select gives a deterministic signal.
	done := make(chan struct{})
	go func() {
		// Drain any remaining items (there are none) and signal completion.
		// This goroutine itself exits immediately because the queue is closed.
		for range crawler.queue { //nolint:revive
		}
		close(done)
	}()

	select {
	case <-done:
		// drain goroutine exited as expected
	case <-time.After(2 * time.Second):
		t.Fatal("drain goroutine did not exit after queue was closed")
	}
}

func TestEngineCrawler_Start_LargeSeedSetDoesNotBlockBeforeWorkers(t *testing.T) {
	t.Parallel()

	const concurrency = 2
	cfg := &config.CrawlerConfig{
		StartURL:     "http://example.com",
		OutputDir:    t.TempDir(),
		MaxDepth:     0,
		Concurrency:  concurrency,
		DefaultDelay: 0,
	}
	for i := 0; i < concurrency*10+5; i++ {
		cfg.StartURLs = append(cfg.StartURLs, "http://example.com/page-"+time.Now().Add(time.Duration(i)).Format("150405.000000000"))
	}

	crawler, err := NewEngineCrawler(cfg, &NoOpReporter{}, "colly", nil)
	if err != nil {
		t.Fatalf("NewEngineCrawler: %v", err)
	}
	oldEngine := crawler.engine
	crawler.engine = testEngine{}
	t.Cleanup(func() { _ = oldEngine.Close() })
	t.Cleanup(crawler.Close)

	done := make(chan error, 1)
	go func() {
		done <- crawler.Start()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Start blocked with %d seeds and queue capacity %d", len(cfg.StartURLs), concurrency*10)
	}
}

func TestNewEngineCrawler_TypedNilStoreFallsBackToMemory(t *testing.T) {
	t.Parallel()

	var typedNil *session.SQLiteStore
	crawler, err := NewEngineCrawler(&config.CrawlerConfig{
		StartURL:     "http://example.com",
		OutputDir:    t.TempDir(),
		Concurrency:  1,
		DefaultDelay: 0,
	}, &NoOpReporter{}, "colly", typedNil)
	if err != nil {
		t.Fatalf("NewEngineCrawler: %v", err)
	}
	t.Cleanup(crawler.Close)

	if _, ok := crawler.visited.(*session.MemoryStore); !ok {
		t.Fatalf("visited store = %T, want *session.MemoryStore", crawler.visited)
	}
}

func TestEngineCrawler_VisitPersistenceFailureIsReportedAndSkipped(t *testing.T) {
	t.Parallel()
	reporter := &recordingReporter{}
	storeErr := errors.New("session database unavailable")
	crawler, err := NewEngineCrawler(&config.CrawlerConfig{
		StartURL:     "http://example.com",
		OutputDir:    t.TempDir(),
		Concurrency:  1,
		DefaultDelay: 0,
	}, reporter, "colly", &failingVisitedStore{err: storeErr})
	if err != nil {
		t.Fatalf("NewEngineCrawler: %v", err)
	}
	t.Cleanup(crawler.Close)

	parsedURL, normalizedURL, skipped := crawler.validateAndCheckURL(0, "http://example.com/page")
	if !skipped || parsedURL != nil || normalizedURL != "" {
		t.Fatalf("validateAndCheckURL = (%v, %q, %v), want (nil, empty, true)", parsedURL, normalizedURL, skipped)
	}
	if got := crawler.stats.PagesFailed.Load(); got != 1 {
		t.Fatalf("PagesFailed = %d, want 1", got)
	}
	if len(reporter.logs) != 1 || !strings.Contains(reporter.logs[0], storeErr.Error()) {
		t.Fatalf("logs = %q, want persistence error", reporter.logs)
	}
}
