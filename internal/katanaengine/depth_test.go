package katanaengine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
)

// TestDepthReachCrawlsExactHopCount pins the MaxDepth translation: a
// chain site (root -> /a -> /b -> /c -> /d) must crawl exactly cfg hops
// below the seed — katana counts hops from the seed the same way crawler
// always has, so the value passes through unchanged.
//
// It also guards against phantom depth notices: katana emits
// Output(request, nil, ErrMaxDepthReached) for depth-exceeded discoveries
// (e.g. sitemap-listed pages); those must neither be stored as visits nor
// suppress the real result for the same URL.
func TestDepthReachCrawlsExactHopCount(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	chain := []string{"/", "/a", "/b", "/c", "/d"}
	for i, p := range chain {
		next := ""
		if i+1 < len(chain) {
			next = fmt.Sprintf(`<a href="%s">next</a>`, chain[i+1])
		}
		body := fmt.Sprintf(`<html><body>page %s %s</body></html>`, p, next)
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		})
	}
	site := httptest.NewServer(mux)
	t.Cleanup(site.Close)

	// chain[i] sits i hops below the seed; cfg.MaxDepth must crawl
	// exactly the pages at hops <= maxDepth.
	tests := []struct{ maxDepth int }{
		{0}, {1}, {2}, {3},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("maxDepth=%d", tt.maxDepth), func(t *testing.T) {
			// Subtests stay sequential: katana engine construction is
			// serialized process-wide (libraryMu), so parallel subtests
			// would just burn their contexts waiting for the lock.
			var mu sync.Mutex
			var paths []string
			cfg := &config.CrawlerConfig{
				StartURL:             site.URL + "/",
				OutputDir:            t.TempDir(),
				MaxDepth:             tt.maxDepth,
				Concurrency:          1,
				UserAgent:            "CrawlerTest/1.0",
				EngineTimeoutSeconds: 1,
			}
			eng := NewKatanaEngine(cfg, nil, nil, WithOnPage(func(pr PageResult) {
				mu.Lock()
				paths = append(paths, pr.URL)
				mu.Unlock()
			}))
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			if err := eng.Crawl(ctx, site.URL+"/"); err != nil {
				t.Fatalf("Crawl: %v", err)
			}

			mu.Lock()
			defer mu.Unlock()
			wantCount := tt.maxDepth + 1 // seed plus maxDepth hops
			if len(paths) != wantCount {
				t.Fatalf("maxDepth=%d crawled %d pages (%v), want exactly %d", tt.maxDepth, len(paths), paths, wantCount)
			}
			for i := 0; i <= tt.maxDepth; i++ {
				want := site.URL + chain[i]
				if !slices.Contains(paths, want) {
					t.Errorf("maxDepth=%d: page %s not crawled (got %v)", tt.maxDepth, want, paths)
				}
			}
		})
	}
}
