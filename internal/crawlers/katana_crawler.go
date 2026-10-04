package crawlers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/katanaengine"
	"github.com/dotcommander/crawler/internal/seeders"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/internal/utils"
)

// KatanaCrawler implements the api.Crawler surface on top of the katana
// engine adapter. It owns the surfaces katana does not: structured
// export records, on-disk content saving, CSS-selector extraction,
// sitemap pre-seeding, and the caller-supplied robots checker.
type KatanaCrawler struct {
	cfg               *config.CrawlerConfig
	reporter          ProgressReporter
	store             session.VisitedStore
	exporter          exporters.Exporter
	compiledSelectors []compiledSelector
	robotsResult      *seeders.RobotsResult

	ctx    context.Context
	cancel context.CancelFunc
	// writeMu serializes exporter and file writes across katana's
	// concurrent result callbacks.
	writeMu sync.Mutex
	started atomic.Bool
	done    chan struct{}
}

// NewKatanaCrawler creates a crawler backed by the katana engine. A nil
// store falls back to an in-memory visited store.
func NewKatanaCrawler(cfg *config.CrawlerConfig, reporter ProgressReporter, store session.VisitedStore) (*KatanaCrawler, error) {
	if reporter == nil {
		reporter = &NoOpReporter{}
	}
	if store == nil || isNilVisitedStore(store) {
		store = session.NewMemoryStore()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &KatanaCrawler{
		cfg:      cfg,
		reporter: reporter,
		store:    store,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	if len(cfg.ExtractSelectors) > 0 {
		c.compiledSelectors = CompileSelectors(cfg.ExtractSelectors)
	}
	return c, nil
}

func isNilVisitedStore(store session.VisitedStore) bool {
	v := reflect.ValueOf(store)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// SetExporter sets the exporter for structured output.
func (c *KatanaCrawler) SetExporter(exp exporters.Exporter) {
	c.exporter = exp
}

// SetRobotsResult installs a caller-fetched robots.txt result that
// replaces the engine's own robots handling (discovery-level OutOfScope
// translation plus the result-level allow gate).
func (c *KatanaCrawler) SetRobotsResult(res *seeders.RobotsResult) {
	c.robotsResult = res
}

// Start crawls all configured seeds (plus sitemap pre-seeds) and blocks
// until the crawl finishes or is cancelled. A second call returns an
// error instead of racing the first run.
func (c *KatanaCrawler) Start() error {
	if !c.started.CompareAndSwap(false, true) {
		return fmt.Errorf("crawler already started")
	}
	defer close(c.done)
	if err := os.MkdirAll(c.cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	seeds := c.cfg.StartURLs
	if len(seeds) == 0 {
		seeds = []string{c.cfg.StartURL}
	}
	seeds = append(slices.Clone(seeds), c.cfg.SeedURLs...)

	c.reporter.Log("INFO", fmt.Sprintf("Using %s engine for crawling", engineKind(c.cfg)))

	opts := []katanaengine.Option{katanaengine.WithOnPage(c.handleResult)}
	if c.robotsResult != nil {
		opts = append(opts, katanaengine.WithRobotsResult(c.robotsResult))
	}
	return katanaengine.RunGroups(c.ctx, c.cfg, seeds, c.reporter, c.store, opts...)
}

// Cancel stops an in-flight crawl.
func (c *KatanaCrawler) Cancel() {
	c.cancel()
}

// Close cancels the crawl, waits (bounded) for it to unwind so result
// callbacks finish their writes, then releases the visited store.
func (c *KatanaCrawler) Close() {
	c.cancel()
	select {
	case <-c.done:
	case <-time.After(10 * time.Second):
	}
	if c.store != nil {
		c.store.Close()
	}
}

func engineKind(cfg *config.CrawlerConfig) string {
	if katanaengine.WantsHeadless(cfg) {
		return "katana-headless"
	}
	return "katana-standard"
}

// handleResult exports, saves, and logs one successfully crawled page.
// It runs on katana worker goroutines; writeMu keeps exporter and file
// writes serialized.
func (c *KatanaCrawler) handleResult(pr katanaengine.PageResult) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	isHTML := strings.Contains(strings.ToLower(pr.ContentType), "text/html")
	title := ""
	if isHTML {
		title = pageTitle(pr.Body)
	}

	if c.exporter != nil {
		record := exporters.PageRecord{
			URL:         pr.URL,
			Title:       title,
			StatusCode:  pr.StatusCode,
			ContentType: pr.ContentType,
			// katana does not expose per-page discovered links; the
			// exported field stays for schema compatibility but is
			// always zero (documented deviation in the migration spec).
			LinksFound: 0,
			CrawledAt:  time.Now(),
		}
		if isHTML && len(c.compiledSelectors) > 0 {
			record.Extracted = ExtractFields(pr.Body, c.compiledSelectors)
		}
		if err := c.exporter.WriteRecord(record); err != nil {
			c.reporter.Log("ERROR", fmt.Sprintf("Failed to export record for %s: %v", pr.URL, err))
		}
	}

	if len(pr.Body) > 0 {
		filePath := utils.GenerateFilePath(c.cfg.OutputDir, pr.URL)
		if err := saveContent(filePath, pr.Body); err != nil {
			c.reporter.Log("ERROR", fmt.Sprintf("Failed to save %s: %v", pr.URL, err))
		} else {
			c.reporter.Log("INFO", fmt.Sprintf("Saved: %s", utils.TruncateURL(filePath, 60)))
		}
	}

	if pr.IsPDF {
		c.reporter.Log("SUCCESS", fmt.Sprintf("Downloaded PDF: %s (%s)",
			utils.TruncateURL(pr.URL, 60), utils.FormatBytes(int64(len(pr.Body)), true)))
	} else {
		c.reporter.Log("SUCCESS", fmt.Sprintf("Crawled: %s (%s)",
			utils.TruncateURL(pr.URL, 60), utils.FormatBytes(int64(len(pr.Body)), true)))
	}
}

// pageTitle extracts the document title, mirroring the legacy engines'
// head > title extraction.
func pageTitle(body []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Find("head > title").First().Text())
}

func saveContent(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(filePath, content, 0o644)
}
