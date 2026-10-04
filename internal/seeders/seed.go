package seeders

import (
	"context"
	"log"
	"net/url"
	"time"
)

// SeedResult contains the output of the seeding process.
type SeedResult struct {
	URLs            []string
	DisallowedCount int
}

// Seed uses the caller-provided robots.txt result (fetching one itself
// only when nil) together with sitemaps for the given start URL, returning
// seed URLs filtered by domain boundaries and robots.txt rules.
func Seed(ctx context.Context, startURL, userAgent string, excludePatterns []string, robots *RobotsResult, verbose bool) (*SeedResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if robots == nil {
		fetched, err := FetchRobotsTxt(ctx, startURL, verbose)
		if err != nil {
			if verbose {
				log.Printf("[WARN] robots.txt fetch failed: %v", err)
			}
			fetched = &RobotsResult{}
		}
		robots = fetched
	}

	sitemapURLs, err := FetchSitemapURLs(ctx, startURL, robots.SitemapURLs, excludePatterns, verbose)
	if err != nil {
		if verbose {
			log.Printf("[WARN] Sitemap fetch failed: %v", err)
		}
		sitemapURLs = nil
	}

	var filtered []string
	disallowed := 0
	for _, seedURL := range sitemapURLs {
		parsed, err := url.Parse(seedURL)
		if err != nil {
			continue
		}
		if !robots.IsAllowed(parsed.Path, userAgent) {
			disallowed++
			continue
		}
		filtered = append(filtered, seedURL)
	}

	if verbose && disallowed > 0 {
		log.Printf("[INFO] Filtered %d URLs disallowed by robots.txt", disallowed)
	}

	return &SeedResult{
		URLs:            filtered,
		DisallowedCount: disallowed,
	}, nil
}
