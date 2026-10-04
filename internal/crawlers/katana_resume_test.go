package crawlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/session"
)

// TestKatanaCrawlerResumeStoresEachPageOnce covers the resume e2e path
// (migration spec D2 fallback branch): a crawl stopped before completion
// (MaxPages cutoff) persists done pages in the SQLite session; a resumed
// run reattaches to the same session (resume reopens the visited store
// with state='done' rows) and must not store, export, or re-record any
// completed page while still covering the whole site.
//
// katana's own resume state (Options.Resume) is CLI-layer only and never
// consumed by pkg/engine, so the session store is the sole dedup
// authority — parity is scoped to "no duplicate stored pages".
func TestKatanaCrawlerResumeStoresEachPageOnce(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	pages := map[string]string{
		"/":   `<html><head><title>Home</title></head><body><a href="/p1">p1</a> <a href="/p2">p2</a></body></html>`,
		"/p1": `<html><head><title>P1</title></head><body><a href="/p3">p3</a></body></html>`,
		"/p2": `<html><head><title>P2</title></head><body><a href="/p4">p4</a></body></html>`,
		"/p3": `<html><head><title>P3</title></head><body>three</body></html>`,
		"/p4": `<html><head><title>P4</title></head><body>four</body></html>`,
	}
	for path, body := range pages {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(body))
		})
	}
	site := httptest.NewServer(mux)
	t.Cleanup(site.Close)

	wantURLs := []string{site.URL + "/", site.URL + "/p1", site.URL + "/p2", site.URL + "/p3", site.URL + "/p4"}

	sessionsDir := t.TempDir()
	outputDir := t.TempDir()

	newCfg := func(maxPages int) *config.CrawlerConfig {
		return &config.CrawlerConfig{
			StartURL:             site.URL,
			OutputDir:            outputDir,
			MaxDepth:             2,
			Concurrency:          2,
			DefaultDelay:         10 * time.Millisecond,
			EngineTimeoutSeconds: 1,
			MaxPages:             maxPages,
			UserAgent:            "CrawlerTest/1.0",
		}
	}

	// runPass opens the session store (reattaching when resume is set),
	// crawls, and returns the exported record URLs in order.
	runPass := func(t *testing.T, resume bool, maxPages int) []string {
		t.Helper()
		store, err := session.NewSQLiteStore(sessionsDir, site.URL, resume)
		if err != nil {
			t.Fatalf("NewSQLiteStore(resume=%v): %v", resume, err)
		}
		var buf bytes.Buffer
		exp, err := exporters.New("jsonl", &buf)
		if err != nil {
			store.Close()
			t.Fatalf("exporters.New: %v", err)
		}
		if testing.Short() {
			t.Skip("integration: resume e2e")
		}
		crawler, err := NewKatanaCrawler(newCfg(maxPages), &NoOpReporter{}, store)
		if err != nil {
			t.Fatalf("NewKatanaCrawler: %v", err)
		}
		t.Cleanup(crawler.Close)
		crawler.SetExporter(exp)
		if err := crawler.Start(); err != nil {
			t.Fatalf("Start(resume=%v): %v", resume, err)
		}
		crawler.Close()

		var urls []string
		for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
			if len(line) == 0 {
				continue
			}
			var rec struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(line, &rec); err != nil {
				t.Fatalf("parse jsonl record %q: %v", line, err)
			}
			urls = append(urls, rec.URL)
		}
		return urls
	}

	// MaxPages is a soft cap: with parallel workers a bounded number of
	// in-flight pages can slip past the limit before cancellation, so
	// assert a partial crawl (cutoff happened, most pages still missing)
	// rather than an exact count.
	pass1 := runPass(t, false, 2)
	if len(pass1) < 2 || len(pass1) >= len(wantURLs) {
		t.Fatalf("pass 1 exported %d pages (%v), want a partial crawl of at least 2 pages short of the full site", len(pass1), pass1)
	}

	// Pages the first pass completed (state='done') are the ones a
	// resumed run must not re-store; pages cut mid-flight may be
	// legitimately re-exported (fallback semantics).
	done := doneURLs(t, sessionsDir, site.URL)
	if len(done) == 0 {
		t.Fatal("no completed pages recorded after pass 1")
	}

	pass2 := runPass(t, true, 0)
	if len(pass2) == 0 {
		t.Fatal("pass 2 exported no pages")
	}
	for _, u := range pass2 {
		if slices.Contains(done, u) {
			t.Errorf("resumed run re-exported completed page %s (done=%v pass2=%v)", u, done, pass2)
		}
	}
	if repeated := repeatedEntries(pass2); len(repeated) > 0 {
		t.Errorf("resumed run exported duplicates within itself: %v", repeated)
	}

	union := slices.Concat(pass1, pass2)
	for _, want := range wantURLs {
		if !slices.Contains(union, want) {
			t.Errorf("page %s never exported across both passes (union=%v)", want, union)
		}
	}

	// Reattach once more and confirm every page is marked visited.
	verify, err := session.NewSQLiteStore(sessionsDir, site.URL, true)
	if err != nil {
		t.Fatalf("NewSQLiteStore(verify): %v", err)
	}
	defer verify.Close()
	for _, want := range wantURLs {
		if !verify.IsVisited(want) {
			t.Errorf("page %s not marked visited after resume", want)
		}
	}
}

// doneURLs reads the session database directly and returns the URLs
// whose crawl state is 'done'.
func doneURLs(t *testing.T, sessionsDir, startURL string) []string {
	t.Helper()
	sum := sha256.Sum256([]byte(startURL))
	dbPath := filepath.Join(sessionsDir, fmt.Sprintf("%x.db", sum[:8]))
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open session db: %v", err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT url FROM visited WHERE state = 'done'`)
	if err != nil {
		t.Fatalf("query visited: %v", err)
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			t.Fatalf("scan visited: %v", err)
		}
		urls = append(urls, u)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate visited: %v", err)
	}
	return urls
}

// repeatedEntries returns entries occurring more than once in urls.
func repeatedEntries(urls []string) []string {
	seen := make(map[string]bool, len(urls))
	var repeated []string
	for _, u := range urls {
		if seen[u] {
			repeated = append(repeated, u)
		}
		seen[u] = true
	}
	return repeated
}
