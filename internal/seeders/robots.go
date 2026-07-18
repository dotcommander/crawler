package seeders

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsResult contains parsed robots.txt data.
type RobotsResult struct {
	SitemapURLs     []string
	DisallowedPaths []string
	robotsData      *robotstxt.RobotsData
}

// IsAllowed checks whether a given path is allowed for the configured user-agent.
func (r *RobotsResult) IsAllowed(path, userAgent string) bool {
	if r == nil || r.robotsData == nil {
		return true
	}
	agent := userAgent
	if agent == "" {
		agent = "*"
	}
	return r.robotsData.TestAgent(path, agent)
}

// FetchRobotsTxt fetches and parses robots.txt for the given base URL.
func FetchRobotsTxt(ctx context.Context, baseURL string, verbose bool) (*RobotsResult, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return &RobotsResult{}, fmt.Errorf("parse base URL: %w", err)
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", u.Scheme, u.Host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return &RobotsResult{}, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if verbose {
			log.Printf("[INFO] Could not fetch robots.txt: %v", err)
		}
		return &RobotsResult{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if verbose {
			log.Printf("[INFO] robots.txt returned HTTP %d, skipping", resp.StatusCode)
		}
		return &RobotsResult{}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return &RobotsResult{}, fmt.Errorf("read robots.txt body: %w", err)
	}

	data, err := robotstxt.FromBytes(body)
	if err != nil {
		if verbose {
			log.Printf("[WARN] Failed to parse robots.txt: %v", err)
		}
		return &RobotsResult{}, nil
	}

	result := &RobotsResult{
		SitemapURLs: data.Sitemaps,
		robotsData:  data,
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "disallow:") {
			path := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			if path != "" {
				result.DisallowedPaths = append(result.DisallowedPaths, path)
			}
		}
	}

	if verbose {
		log.Printf("[INFO] robots.txt: %d sitemap(s), %d disallow rule(s)",
			len(result.SitemapURLs), len(result.DisallowedPaths))
	}

	return result, nil
}
