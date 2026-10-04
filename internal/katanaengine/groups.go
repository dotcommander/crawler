package katanaengine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/ui"
)

// WantsHeadless reports whether the configuration requires a browser
// engine, mirroring the legacy auto-selection: mobile emulation, a wait
// strategy other than networkidle, or a long extra wait. Everything
// else runs on the standard HTTP engine.
func WantsHeadless(cfg *config.CrawlerConfig) bool {
	if cfg.Mobile {
		return true
	}
	if ws := cfg.WaitStrategy; ws != "" && ws != "networkidle" {
		return true
	}
	return cfg.ExtraWaitTime > 500*time.Millisecond
}

// SeedGroup bundles the seed URLs that share one request-pacing delay.
type SeedGroup struct {
	Delay time.Duration
	Seeds []string
}

// GroupSeeds partitions seeds by the DomainDelays entry matching each
// seed's host exactly (hostname match, like the legacy rate limiter);
// hosts without an entry use DefaultDelay. Groups are ordered by
// ascending delay with seeds keeping input order. Crawling is confined
// to each group's own seeds (see RunGroups), so links crossing delay
// groups are not followed — a documented deviation from the legacy
// crawler, which paced cross-domain discovery by the target domain's
// delay inside one crawl.
func GroupSeeds(cfg *config.CrawlerConfig, seeds []string) []SeedGroup {
	if len(cfg.DomainDelays) == 0 {
		return []SeedGroup{{Delay: cfg.DefaultDelay, Seeds: seeds}}
	}
	byDelay := make(map[time.Duration][]string)
	for _, seed := range seeds {
		delay := cfg.DefaultDelay
		if host := hostOf(seed); host != "" {
			if override, ok := cfg.DomainDelays[host]; ok {
				delay = override
			}
		}
		byDelay[delay] = append(byDelay[delay], seed)
	}
	delays := make([]int64, 0, len(byDelay))
	for d := range byDelay {
		delays = append(delays, int64(d))
	}
	sort.Slice(delays, func(i, j int) bool { return delays[i] < delays[j] })
	groups := make([]SeedGroup, 0, len(delays))
	for _, d := range delays {
		groups = append(groups, SeedGroup{Delay: time.Duration(d), Seeds: byDelay[time.Duration(d)]})
	}
	return groups
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// offsetReporter adds completed-group totals to a following group's
// stats so counters stay monotonic across the per-group engines.
type offsetReporter struct {
	inner Reporter
	off   ui.StatsMsg
}

func (r *offsetReporter) Log(level, message string)             { r.inner.Log(level, message) }
func (r *offsetReporter) UpdateWorker(id int, status, u string) { r.inner.UpdateWorker(id, status, u) }
func (r *offsetReporter) UpdateStats(stats ui.StatsMsg) {
	stats.PagesVisited += r.off.PagesVisited
	stats.PagesFailed += r.off.PagesFailed
	stats.PDFsDownloaded += r.off.PDFsDownloaded
	stats.BytesDownloaded += r.off.BytesDownloaded
	r.inner.UpdateStats(stats)
}

// RunGroups runs one katana engine per delay group over the seeds,
// sharing the visited store, reporter, and options. Groups run
// sequentially (the library-wide katana lock serializes concurrent
// engines anyway; see libraryMu in engine.go). Each group's crawl is
// scoped to its own seeds, which is how per-domain delays are enforced
// in katana's single-rate-per-engine model.
func RunGroups(ctx context.Context, cfg *config.CrawlerConfig, seeds []string, reporter Reporter, store session.VisitedStore, opts ...Option) error {
	if len(seeds) == 0 {
		return fmt.Errorf("katanaengine: no seed URLs")
	}
	if reporter == nil {
		return fmt.Errorf("katanaengine: nil reporter")
	}
	groups := GroupSeeds(cfg, seeds)
	var carry ui.StatsMsg
	var errs []error
	for i, group := range groups {
		if ctx.Err() != nil {
			break
		}
		groupCfg := *cfg
		groupCfg.DefaultDelay = group.Delay
		groupCfg.DomainDelays = nil
		rep := reporter
		if i > 0 {
			rep = &offsetReporter{inner: reporter, off: carry}
		}
		if len(groups) > 1 {
			reporter.Log("INFO", fmt.Sprintf("Crawling delay group %d/%d (%d seeds, %s pacing)",
				i+1, len(groups), len(group.Seeds), group.Delay))
		}
		engine := NewKatanaEngine(&groupCfg, rep, store, opts...)
		if err := engine.Crawl(ctx, group.Seeds...); err != nil {
			errs = append(errs, fmt.Errorf("delay group %d: %w", i+1, err))
		}
		final := engine.Stats()
		carry.PagesVisited += final.PagesVisited
		carry.PagesFailed += final.PagesFailed
		carry.PDFsDownloaded += final.PDFsDownloaded
		carry.BytesDownloaded += final.BytesDownloaded
	}
	reporter.UpdateStats(carry)
	return errors.Join(errs...)
}
