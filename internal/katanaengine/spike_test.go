package katanaengine

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/projectdiscovery/katana/pkg/engine/standard"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"
)

// TestSpikeKatanaStandardEngine is the Phase 1 pin check: it proves the
// pinned katana version drives the standard engine as a library against
// a local httptest origin, delivering results through OnResult.
func TestSpikeKatanaStandardEngine(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`<html><head><title>root</title></head><body><a href="/sub">sub</a></body></html>`))
			return
		}
		_, _ = w.Write([]byte(`<html><head><title>sub</title></head><body></body></html>`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var crawled []string
	options := &types.Options{
		URLs:               []string{server.URL},
		MaxDepth:           1,
		FieldScope:         "",
		BodyReadSize:       math.MaxInt,
		RateLimit:          150,
		Silent:             true,
		DisableUpdateCheck: true,
		Timeout:            1,
		Strategy:           "breadth-first",
		Context:            context.Background(),
		OnResult: func(r output.Result) {
			if r.Request == nil {
				return
			}
			// Runs under libraryMu (whole-run lock); worker callbacks
			// complete before Crawl returns.
			crawled = append(crawled, r.Request.URL)
		},
	}

	// libraryMu is held across construction, crawl, and close; see its
	// comment in engine.go.
	libraryMu.Lock()
	defer libraryMu.Unlock()

	crawlerOptions, err := types.NewCrawlerOptions(options)
	if err != nil {
		t.Fatalf("NewCrawlerOptions: %v", err)
	}

	crawler, err := standard.New(crawlerOptions)
	if err != nil {
		t.Fatalf("standard.New: %v", err)
	}

	if err := crawler.Crawl(server.URL); err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if err := crawlerOptions.Close(); err != nil {
		t.Logf("close crawler options: %v", err)
	}
	if err := crawler.Close(); err != nil {
		t.Logf("close crawler: %v", err)
	}

	if len(crawled) == 0 {
		t.Fatal("no URLs crawled")
	}
	if !slices.Contains(crawled, server.URL+"/sub") {
		t.Errorf("crawled URLs %v do not include %s", crawled, server.URL+"/sub")
	}
}
