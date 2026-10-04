package katanaengine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/ui"
)

func TestGroupSeedsNoDelays(t *testing.T) {
	t.Parallel()

	cfg := &config.CrawlerConfig{DefaultDelay: 2 * time.Second}
	groups := GroupSeeds(cfg, []string{"https://a.example/", "https://b.example/"})
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Delay != 2*time.Second {
		t.Errorf("delay = %v, want 2s", groups[0].Delay)
	}
	if len(groups[0].Seeds) != 2 {
		t.Errorf("seeds = %v, want both", groups[0].Seeds)
	}
}

func TestGroupSeedsPartitionsByHostDelay(t *testing.T) {
	t.Parallel()

	cfg := &config.CrawlerConfig{
		DefaultDelay: time.Second,
		DomainDelays: map[string]time.Duration{
			"slow.example": 5 * time.Second,
		},
	}
	groups := GroupSeeds(cfg, []string{
		"https://fast.example/",
		"https://slow.example/",
		"https://other.example/",
	})
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(groups), groups)
	}
	// Ascending delay order: default group (1s) first.
	if groups[0].Delay != time.Second || len(groups[0].Seeds) != 2 {
		t.Errorf("default group = %+v", groups[0])
	}
	if groups[1].Delay != 5*time.Second || len(groups[1].Seeds) != 1 || groups[1].Seeds[0] != "https://slow.example/" {
		t.Errorf("slow group = %+v", groups[1])
	}
}

func TestRunGroupsCrawlsAllGroupsAndKeepsStatsMonotonic(t *testing.T) {
	t.Parallel()

	siteA := newSinglePageServer(t, "a")
	defer siteA.Close()
	siteB := newSinglePageServer(t, "b")
	defer siteB.Close()

	cfg := &config.CrawlerConfig{
		MaxDepth:             1,
		Concurrency:          2,
		EngineTimeoutSeconds: 1,
		DomainDelays: map[string]time.Duration{
			hostOf(siteB.URL): 10 * time.Millisecond,
		},
	}
	store := newCountingStore()
	reporter := &captureReporter{}

	err := RunGroups(t.Context(), cfg, []string{siteA.URL, siteB.URL}, reporter, store, WithTimeout(1))
	if err != nil {
		t.Fatalf("RunGroups: %v", err)
	}

	visited := map[string]bool{}
	for _, u := range store.stored() {
		visited[u] = true
	}
	if !visited[siteA.URL+"/"] || !visited[siteB.URL+"/"] {
		t.Errorf("stored %v, want both group seeds", store.stored())
	}

	// Counters across the two sequential engines must never go backwards.
	reporter.mu.Lock()
	stats := append([]ui.StatsMsg(nil), reporter.stats...)
	reporter.mu.Unlock()
	for i := 1; i < len(stats); i++ {
		if stats[i].PagesVisited < stats[i-1].PagesVisited {
			t.Fatalf("stats went backwards at %d: %+v -> %+v", i, stats[i-1], stats[i])
		}
	}
	if last := stats[len(stats)-1]; last.PagesVisited < 2 {
		t.Errorf("final stats = %+v, want at least 2 visited", last)
	}
}

func newSinglePageServer(t *testing.T, name string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>" + name + "</title></head><body>" + name + "</body></html>"))
	})
	return httptest.NewServer(mux)
}
