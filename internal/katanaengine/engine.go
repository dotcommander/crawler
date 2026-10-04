package katanaengine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/seeders"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/ui"
	kateng "github.com/projectdiscovery/katana/pkg/engine"
	"github.com/projectdiscovery/katana/pkg/engine/headless"
	"github.com/projectdiscovery/katana/pkg/engine/standard"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"
)

// Reporter is the reporting contract this engine drives. It is satisfied
// structurally by crawlers.ProgressReporter implementations, avoiding an
// import cycle with the legacy package that this adapter replaces.
type Reporter interface {
	Log(level, message string)
	UpdateStats(stats ui.StatsMsg)
	UpdateWorker(workerID int, status, url string)
}

// KatanaEngine runs crawls through the katana standard engine, mapping
// results onto crawler's reporter and visited-store surfaces.
type KatanaEngine struct {
	cfg      *config.CrawlerConfig
	reporter Reporter
	store    session.VisitedStore
	settings engineSettings

	mu              sync.Mutex
	running         bool
	pagesVisited    int64
	pagesFailed     int64
	bytesDownloaded int64
	pdfsDownloaded  int64
	limitHit        bool

	robots *seeders.RobotsResult
	cancel context.CancelFunc
	eng    kateng.Engine
}

// Option customizes a KatanaEngine.
type Option func(*engineSettings)

type engineSettings struct {
	timeoutSec   int
	onPage       func(PageResult)
	robotsResult *seeders.RobotsResult
}

// WithTimeout overrides the katana engine timeout window in seconds. In
// the standard engine this value doubles as the queue's minimum lifetime:
// a natural crawl end is detected only after the queue has been empty
// past this window, so small values make bounded crawls finish sooner.
// Zero keeps katana's upstream default.
func WithTimeout(seconds int) Option {
	return func(s *engineSettings) {
		s.timeoutSec = seconds
	}
}

// WithOnPage registers a callback invoked once per successfully crawled,
// newly visited page (after the visit and status are persisted). The
// callback runs on katana worker goroutines and must be safe for
// concurrent use.
func WithOnPage(fn func(PageResult)) Option {
	return func(s *engineSettings) {
		s.onPage = fn
	}
}

// WithRobotsResult installs a caller-fetched robots.txt result that
// replaces the engine's own robots fetch. Its disallowed paths feed the
// discovery-level OutOfScope translation and the result-level gate.
func WithRobotsResult(res *seeders.RobotsResult) Option {
	return func(s *engineSettings) {
		s.robotsResult = res
	}
}

// NewKatanaEngine constructs a katana-backed engine. Call Crawl to run;
// Close releases engine resources.
func NewKatanaEngine(cfg *config.CrawlerConfig, reporter Reporter, store session.VisitedStore, opts ...Option) *KatanaEngine {
	settings := engineSettings{}
	for _, opt := range opts {
		opt(&settings)
	}
	if store == nil || isNilStore(store) {
		store = session.NewMemoryStore()
	}
	if reporter == nil {
		reporter = noopReporter{}
	}
	return &KatanaEngine{
		cfg:      cfg,
		reporter: reporter,
		store:    store,
		settings: settings,
	}
}

func isNilStore(store session.VisitedStore) bool {
	v := reflect.ValueOf(store)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// noopReporter is the fallback reporter when none is provided.
type noopReporter struct{}

func (noopReporter) Log(string, string)               {}
func (noopReporter) UpdateStats(ui.StatsMsg)          {}
func (noopReporter) UpdateWorker(int, string, string) {}
func (noopReporter) UpdateQueueSize(int)              {}
func (noopReporter) SetPhase(string)                  {}
func (noopReporter) Close()                           {}

// GetEngineType identifies the engine for logging and stats.
func (e *KatanaEngine) GetEngineType() string {
	if WantsHeadless(e.cfg) {
		return "katana-headless"
	}
	return "katana-standard"
}

// libraryMu serializes katana library use process-wide. Upstream
// NewCrawlerOptions calls ConfigureOutput, which writes the global
// gologger level, while engine runs read it from worker goroutines;
// concurrent katana engines in one process therefore race inside the
// library. Holding this lock across construction and the crawl loop
// makes runs sound at the cost of serializing concurrent engines (a
// fix worth contributing upstream).
var libraryMu sync.Mutex

// discardWriter replaces katana's default output writer, which prints
// every result URL to stdout. Results are consumed exclusively through
// Options.OnResult; Write must succeed so Output invokes OnResult.
type discardWriter struct{}

func (discardWriter) Write(*output.Result) error   { return nil }
func (discardWriter) WriteErr(*output.Error) error { return nil }
func (discardWriter) Close() error                 { return nil }
func (discardWriter) GetResultCount() int64        { return 0 }

// Crawl crawls the given seeds. The context bounds the whole run and is
// propagated into katana via Options.Context.
func (e *KatanaEngine) Crawl(ctx context.Context, seeds ...string) error {
	if len(seeds) == 0 {
		return fmt.Errorf("katanaengine: no seed URLs")
	}
	if err := e.beginRun(); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancel = cancel
	e.mu.Unlock()
	defer func() {
		cancel()
		e.mu.Lock()
		e.running = false
		e.cancel = nil
		e.eng = nil
		e.mu.Unlock()
	}()

	robots := &seeders.RobotsResult{}
	switch {
	case e.settings.robotsResult != nil:
		robots = e.settings.robotsResult
	case !e.cfg.NoRobots:
		fetched, err := seeders.FetchRobotsTxt(runCtx, seeds[0], false)
		if err == nil && fetched != nil {
			robots = fetched
		}
	}
	e.mu.Lock()
	e.robots = robots
	e.mu.Unlock()

	opts, err := TranslateOptions(e.cfg, seeds)
	if err != nil {
		return err
	}
	opts.OutOfScope = append(opts.OutOfScope, OutOfScopeFromRobots(seeds, robots.DisallowedPaths)...)
	opts.Context = runCtx
	if e.settings.timeoutSec > 0 {
		opts.Timeout = e.settings.timeoutSec
	}
	opts.OnResult = e.onResult
	opts.OnSkipURL = func(u string) {
		e.reporter.Log("WARN", fmt.Sprintf("Skipped by engine: %s", u))
	}

	// libraryMu is held for the whole run; see its comment.
	libraryMu.Lock()
	defer libraryMu.Unlock()

	crawlerOptions, err := types.NewCrawlerOptions(opts)
	if err != nil {
		return fmt.Errorf("katanaengine: build crawler options: %w", err)
	}
	crawlerOptions.OutputWriter = discardWriter{}
	defer func() {
		if err := crawlerOptions.Close(); err != nil {
			e.reporter.Log("WARN", fmt.Sprintf("Close crawler options: %v", err))
		}
	}()

	// Engine choice: standard for everything in this slice; headless and
	// per-delay groups land with the D3 phase.
	eng, err := newEngine(e.cfg, crawlerOptions)
	if err != nil {
		return fmt.Errorf("katanaengine: create engine: %w", err)
	}
	defer func() {
		if err := eng.Close(); err != nil {
			e.reporter.Log("WARN", fmt.Sprintf("Close engine: %v", err))
		}
	}()
	e.mu.Lock()
	e.eng = eng
	e.mu.Unlock()

	e.reporter.Log("INFO", fmt.Sprintf("Starting crawl of %d seed URL(s) (engine=%s)", len(seeds), e.GetEngineType()))

	var errs []error
	for _, seed := range seeds {
		if runCtx.Err() != nil {
			break
		}
		if err := eng.Crawl(seed); err != nil {
			if e.isLimitStop(err) {
				break
			}
			errs = append(errs, fmt.Errorf("crawl %s: %w", seed, err))
		}
	}
	// Cancellation (user cancel, MaxPages limit, or parent deadline) ends
	// the run cleanly, mirroring the legacy crawler's Start semantics.
	if runCtx.Err() != nil {
		return nil
	}
	e.reporter.UpdateStats(e.Stats())
	return errors.Join(errs...)
}

// Cancel stops an in-flight crawl.
func (e *KatanaEngine) Cancel() {
	e.mu.Lock()
	cancel := e.cancel
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Close releases engine resources and stops any in-flight crawl.
func (e *KatanaEngine) Close() error {
	e.Cancel()
	e.mu.Lock()
	eng := e.eng
	e.mu.Unlock()
	if eng != nil {
		return eng.Close()
	}
	return nil
}

func (e *KatanaEngine) beginRun() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return fmt.Errorf("katanaengine: crawl already in progress")
	}
	e.running = true
	e.pagesVisited = 0
	e.pagesFailed = 0
	e.bytesDownloaded = 0
	e.pdfsDownloaded = 0
	e.limitHit = false
	return nil
}

func (e *KatanaEngine) isLimitStop(err error) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.limitHit && errors.Is(err, context.Canceled)
}

// newEngine selects the katana engine implementation: headless Chrome
// where the legacy auto-selection used a browser engine (mobile,
// non-networkidle wait strategies, long extra waits), standard HTTP
// fetch otherwise.
func newEngine(cfg *config.CrawlerConfig, co *types.CrawlerOptions) (kateng.Engine, error) {
	if WantsHeadless(cfg) {
		return headless.New(co)
	}
	return standard.New(co)
}

// Stats returns a snapshot of the run's counters.
func (e *KatanaEngine) Stats() ui.StatsMsg {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.statsLocked()
}

func (e *KatanaEngine) statsLocked() ui.StatsMsg {
	return ui.StatsMsg{
		PagesVisited:    e.pagesVisited,
		PagesFailed:     e.pagesFailed,
		PDFsDownloaded:  e.pdfsDownloaded,
		BytesDownloaded: e.bytesDownloaded,
		QueueSize:       0, // katana owns the queue; size is not exported
		ActiveWorkers:   0,
	}
}
