package katanaengine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/ui"
)

// countingStore wraps a MemoryStore and counts stored pages for
// assertions; MemoryStore exposes no listing.
type countingStore struct {
	*session.MemoryStore
	mu   sync.Mutex
	urls []string
}

func newCountingStore() *countingStore {
	return &countingStore{MemoryStore: session.NewMemoryStore()}
}

func (c *countingStore) MarkVisited(url string) (bool, error) {
	already, err := c.MemoryStore.MarkVisited(url)
	if err == nil && !already {
		c.mu.Lock()
		c.urls = append(c.urls, url)
		c.mu.Unlock()
	}
	return already, err
}

func (c *countingStore) stored() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.urls...)
}

type captureReporter struct {
	mu    sync.Mutex
	stats []ui.StatsMsg
	logs  []string
}

func (r *captureReporter) Log(level, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, level+": "+message)
}

func (r *captureReporter) UpdateStats(stats ui.StatsMsg) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats = append(r.stats, stats)
}

func (r *captureReporter) UpdateWorker(workerID int, status, url string) {}

func (r *captureReporter) lastStats() ui.StatsMsg {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.stats) == 0 {
		return ui.StatsMsg{}
	}
	return r.stats[len(r.stats)-1]
}

// testSite serves pages and records per-path hit counts. The root page
// links to every other served path.
type testSite struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func newTestSite(t *testing.T, robotsBody string, pages map[string]string) *testSite {
	t.Helper()
	site := &testSite{hits: make(map[string]int)}
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		site.count(r.URL.Path)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(robotsBody))
	})
	var links strings.Builder
	for p := range pages {
		fmt.Fprintf(&links, `<a href="%s">%s</a>`, p, p)
	}
	for p := range pages {
		p := p
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			site.count(p)
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<html><head><title>%s</title></head><body>%s</body></html>`, p, links.String())
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		site.count("/")
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<html><head><title>root</title></head><body>%s</body></html>`, links.String())
	})
	site.Server = httptest.NewServer(mux)
	t.Cleanup(site.Close)
	return site
}

func (s *testSite) count(path string) {
	s.mu.Lock()
	s.hits[path]++
	s.mu.Unlock()
}

func (s *testSite) hitCount(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

func TestKatanaEngineCrawlStoresResults(t *testing.T) {
	t.Parallel()

	site := newTestSite(t, "", map[string]string{
		"/a": "<html></html>",
		"/b": "<html></html>",
	})
	store := newCountingStore()
	reporter := &captureReporter{}
	cfg := &config.CrawlerConfig{
		StartURL:    site.URL,
		MaxDepth:    2,
		Concurrency: 2,
		UserAgent:   "crawler-engine-test",
	}
	engine := NewKatanaEngine(cfg, reporter, store, WithTimeout(1))

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := engine.Crawl(ctx, cfg.StartURL); err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	stored := store.stored()
	for _, want := range []string{site.URL + "/a", site.URL + "/b"} {
		found := false
		for _, u := range stored {
			if u == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stored URLs %v missing %q", stored, want)
		}
	}
	stats := reporter.lastStats()
	if stats.PagesVisited < 2 {
		t.Errorf("PagesVisited = %d, want >= 2", stats.PagesVisited)
	}
	if stats.PagesFailed != 0 {
		t.Errorf("PagesFailed = %d, want 0", stats.PagesFailed)
	}
}

func TestKatanaEngineRobotsCompliance(t *testing.T) {
	t.Parallel()

	site := newTestSite(t, "User-agent: *\nDisallow: /private\n", map[string]string{
		"/public":  "<html></html>",
		"/private": "<html></html>",
	})
	store := newCountingStore()
	reporter := &captureReporter{}
	cfg := &config.CrawlerConfig{
		StartURL:  site.URL,
		MaxDepth:  2,
		UserAgent: "crawler-engine-test",
	}
	engine := NewKatanaEngine(cfg, reporter, store, WithTimeout(1))

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := engine.Crawl(ctx, cfg.StartURL); err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	if hits := site.hitCount("/private"); hits != 0 {
		t.Errorf("/private was fetched %d times despite robots disallow", hits)
	}
	for _, u := range store.stored() {
		if strings.Contains(u, "/private") {
			t.Errorf("robots-disallowed URL stored: %s", u)
		}
	}
	if !store.IsVisited(site.URL + "/public") {
		t.Errorf("public page should be stored, have %v", store.stored())
	}
}

func TestKatanaEngineMaxPages(t *testing.T) {
	t.Parallel()

	site := newTestSite(t, "", map[string]string{
		"/p1": "<html></html>",
		"/p2": "<html></html>",
		"/p3": "<html></html>",
	})
	store := newCountingStore()
	reporter := &captureReporter{}
	cfg := &config.CrawlerConfig{
		StartURL:  site.URL,
		MaxDepth:  2,
		MaxPages:  2,
		UserAgent: "crawler-engine-test",
	}
	engine := NewKatanaEngine(cfg, reporter, store, WithTimeout(1))

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := engine.Crawl(ctx, cfg.StartURL); err != nil {
		t.Fatalf("Crawl with page limit should stop cleanly, got %v", err)
	}

	if got := len(store.stored()); got != 2 {
		t.Errorf("stored %d pages, want exactly 2 (MaxPages)", got)
	}
}

func TestKatanaEngineNoRobotsFetchesDisallowed(t *testing.T) {
	t.Parallel()

	site := newTestSite(t, "User-agent: *\nDisallow: /private\n", map[string]string{
		"/private": "<html></html>",
	})
	store := newCountingStore()
	cfg := &config.CrawlerConfig{
		StartURL:  site.URL,
		MaxDepth:  2,
		NoRobots:  true,
		UserAgent: "crawler-engine-test",
	}
	engine := NewKatanaEngine(cfg, &captureReporter{}, store, WithTimeout(1))

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := engine.Crawl(ctx, cfg.StartURL); err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	if !store.IsVisited(site.URL + "/private") {
		t.Errorf("NoRobots should allow /private, stored %v", store.stored())
	}
}

func TestKatanaEngineNoSeeds(t *testing.T) {
	t.Parallel()
	engine := NewKatanaEngine(&config.CrawlerConfig{StartURL: "https://example.com/"}, &captureReporter{}, session.NewMemoryStore())
	if err := engine.Crawl(t.Context()); err == nil {
		t.Fatal("expected error for no seeds")
	}
}
