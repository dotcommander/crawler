package crawlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/seeders"
)

// katanaTestSite serves a small site with robots.txt and records hits.
type katanaTestSite struct {
	*httptest.Server
	mu   chan struct{} // serialized hit counting
	hits map[string]int
}

func newKatanaTestSite(t *testing.T, robotsBody string) *katanaTestSite {
	t.Helper()
	site := &katanaTestSite{mu: make(chan struct{}, 1), hits: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(robotsBody))
	})
	pages := map[string]string{
		"/":        `<html><head><title>Home</title></head><body><h1>Welcome</h1><a href="/sub">sub</a> <a href="/private">private</a></body></html>`,
		"/sub":     `<html><head><title>Sub</title></head><body><h1>Sub page</h1></body></html>`,
		"/private": `<html><head><title>Private</title></head><body>hidden</body></html>`,
	}
	for path, body := range pages {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			site.mu <- struct{}{}
			site.hits[r.URL.Path]++
			<-site.mu
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(body))
		})
	}
	site.Server = httptest.NewServer(mux)
	t.Cleanup(site.Close)
	return site
}

func (s *katanaTestSite) hitCount(path string) int {
	s.mu <- struct{}{}
	defer func() { <-s.mu }()
	return s.hits[path]
}

func newKatanaTestConfig(t *testing.T, startURL, outputDir string) *config.CrawlerConfig {
	t.Helper()
	return &config.CrawlerConfig{
		StartURL:             startURL,
		OutputDir:            outputDir,
		MaxDepth:             2,
		Concurrency:          2,
		DefaultDelay:         10 * time.Millisecond,
		EngineTimeoutSeconds: 1,
		UserAgent:            "CrawlerTest/1.0",
	}
}

func TestKatanaCrawlerExportsSavesAndHonorsRobots(t *testing.T) {
	t.Parallel()

	site := newKatanaTestSite(t, "User-agent: *\nDisallow: /private\n")
	cfg := newKatanaTestConfig(t, site.URL, t.TempDir())
	cfg.ExtractSelectors = map[string]string{"heading": "h1"}

	crawler, err := NewKatanaCrawler(cfg, &NoOpReporter{}, nil)
	if err != nil {
		t.Fatalf("NewKatanaCrawler: %v", err)
	}
	t.Cleanup(crawler.Close)

	var buf bytes.Buffer
	exp, err := exporters.New("jsonl", &buf, "heading")
	if err != nil {
		t.Fatalf("exporters.New: %v", err)
	}
	crawler.SetExporter(exp)

	if err := crawler.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Export records: home + sub crawled, private disallowed.
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("bad JSONL line %q: %v", line, err)
		}
		records = append(records, rec)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d (%v), want 2", len(records), buf.String())
	}
	first := records[0]
	if first["title"] != "Home" {
		t.Errorf("title = %v, want Home", first["title"])
	}
	if first["content_type"] != "text/html" {
		t.Errorf("content_type = %v", first["content_type"])
	}
	extracted, _ := first["extracted"].(map[string]any)
	if extracted["heading"] != "Welcome" {
		t.Errorf("extracted = %v, want heading=Welcome", first["extracted"])
	}

	// Robots: /private never fetched.
	if hits := site.hitCount("/private"); hits != 0 {
		t.Errorf("/private fetched %d times, want 0", hits)
	}

	// Content saved under the output directory for both pages.
	var saved []string
	err = filepath.WalkDir(cfg.OutputDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			saved = append(saved, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk output dir: %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("saved files = %v, want 2", saved)
	}
}

func TestKatanaCrawlerRobotsResultOverride(t *testing.T) {
	t.Parallel()

	// Site robots disallows /sub; the crawler is handed the fetched
	// result directly (the cmd wiring path) instead of fetching again.
	site := newKatanaTestSite(t, "User-agent: *\nDisallow: /sub\n")
	cfg := newKatanaTestConfig(t, site.URL, t.TempDir())

	robots, err := seeders.FetchRobotsTxt(t.Context(), site.URL, false)
	if err != nil {
		t.Fatalf("FetchRobotsTxt: %v", err)
	}

	crawler, err := NewKatanaCrawler(cfg, &NoOpReporter{}, nil)
	if err != nil {
		t.Fatalf("NewKatanaCrawler: %v", err)
	}
	t.Cleanup(crawler.Close)
	crawler.SetRobotsResult(robots)

	var buf bytes.Buffer
	exp, err := exporters.New("jsonl", &buf)
	if err != nil {
		t.Fatalf("exporters.New: %v", err)
	}
	crawler.SetExporter(exp)

	if err := crawler.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if strings.Contains(buf.String(), "/sub") {
		t.Errorf("records contain disallowed /sub: %s", buf.String())
	}
	if hits := site.hitCount("/sub"); hits != 0 {
		t.Errorf("/sub fetched %d times, want 0", hits)
	}
}

func TestKatanaCrawlerMaxPages(t *testing.T) {
	t.Parallel()

	site := newKatanaTestSite(t, "User-agent: *\nDisallow: /private\n")
	cfg := newKatanaTestConfig(t, site.URL, t.TempDir())
	cfg.MaxPages = 1

	crawler, err := NewKatanaCrawler(cfg, &NoOpReporter{}, nil)
	if err != nil {
		t.Fatalf("NewKatanaCrawler: %v", err)
	}
	t.Cleanup(crawler.Close)

	var buf bytes.Buffer
	exp, err := exporters.New("jsonl", &buf)
	if err != nil {
		t.Fatalf("exporters.New: %v", err)
	}
	crawler.SetExporter(exp)

	if err := crawler.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(buf.String()), "\n") + 1
	if lines != 1 {
		t.Errorf("exported %d records, want 1 (MaxPages=1)", lines)
	}
}
