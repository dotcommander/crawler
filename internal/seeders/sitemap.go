package seeders

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dotcommander/crawler/internal/utils"
)

type sitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

const (
	maxSitemapDepth = 3
	maxSitemapSize  = 10 * 1024 * 1024
	maxSeedURLs     = 10000
)

// FetchSitemapURLs fetches sitemap(s) and returns discovered URLs filtered by domain boundary.
func FetchSitemapURLs(ctx context.Context, startURL string, sitemapURLs []string, excludePatterns []string, verbose bool) ([]string, error) {
	startParsed, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("parse start URL: %w", err)
	}

	if len(sitemapURLs) == 0 {
		defaultSitemap := fmt.Sprintf("%s://%s/sitemap.xml", startParsed.Scheme, startParsed.Host)
		sitemapURLs = []string{defaultSitemap}
	}

	seen := make(map[string]struct{})
	var seeds []string

	for _, smURL := range sitemapURLs {
		if len(seeds) >= maxSeedURLs {
			break
		}
		urls, err := fetchSitemap(ctx, smURL, 0, verbose)
		if err != nil {
			if verbose {
				log.Printf("[WARN] Failed to fetch sitemap %s: %v", smURL, err)
			}
			continue
		}

		for _, u := range urls {
			if len(seeds) >= maxSeedURLs {
				break
			}
			if _, exists := seen[u]; exists {
				continue
			}
			seen[u] = struct{}{}

			parsed, err := url.Parse(u)
			if err != nil {
				continue
			}
			if parsed.Host != startParsed.Host {
				continue
			}
			// Check base path
			if !strings.HasPrefix(parsed.Path, startParsed.Path) {
				continue
			}
			// Check exclude patterns
			if utils.MatchesExcludePatterns(u, excludePatterns) {
				continue
			}

			seeds = append(seeds, u)
		}
	}

	if verbose {
		log.Printf("[INFO] Sitemap seeding: discovered %d URLs", len(seeds))
	}

	return seeds, nil
}

func fetchSitemap(ctx context.Context, sitemapURL string, depth int, verbose bool) ([]string, error) {
	if depth > maxSitemapDepth {
		return nil, fmt.Errorf("sitemap nesting depth exceeded (%d)", maxSitemapDepth)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSitemapSize)))
	if err != nil {
		return nil, fmt.Errorf("read sitemap body: %w", err)
	}

	// Try parsing as sitemap index first
	var index sitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		if verbose {
			log.Printf("[INFO] Sitemap index %s: %d nested sitemaps", sitemapURL, len(index.Sitemaps))
		}
		var allURLs []string
		for _, entry := range index.Sitemaps {
			loc := strings.TrimSpace(entry.Loc)
			if loc == "" {
				continue
			}
			nested, err := fetchSitemap(ctx, loc, depth+1, verbose)
			if err != nil {
				if verbose {
					log.Printf("[WARN] Failed to fetch nested sitemap %s: %v", loc, err)
				}
				continue
			}
			allURLs = append(allURLs, nested...)
		}
		return allURLs, nil
	}

	// Parse as regular sitemap
	var us urlSet
	if err := xml.Unmarshal(body, &us); err != nil {
		return nil, fmt.Errorf("parse sitemap XML: %w", err)
	}

	urls := make([]string, 0, len(us.URLs))
	for _, u := range us.URLs {
		loc := strings.TrimSpace(u.Loc)
		if loc != "" {
			urls = append(urls, loc)
		}
	}

	if verbose {
		log.Printf("[INFO] Sitemap %s: %d URLs", sitemapURL, len(urls))
	}

	return urls, nil
}
