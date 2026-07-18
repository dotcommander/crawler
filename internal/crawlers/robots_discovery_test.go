package crawlers

import (
	"net/url"
	"testing"

	"github.com/dotcommander/crawler/internal/config"
)

func TestShouldSkipURL_RobotsDiscoveryEnforcement(t *testing.T) {
	t.Parallel()

	seed, _ := url.Parse("https://example.com/")
	c := &EngineCrawler{
		config:   &config.CrawlerConfig{UserAgent: "TestBot"},
		reporter: &NoOpReporter{},
		seedURLs: []*url.URL{seed},
	}

	// No checker installed: in-domain URL must NOT be skipped (backward-compat).
	if c.shouldSkipURL("https://example.com/page") {
		t.Fatal("expected in-domain URL allowed when no robots checker is set")
	}

	// Install a checker that disallows /private.
	c.SetRobotsChecker(func(path, userAgent string) bool {
		return path != "/private"
	})

	if c.shouldSkipURL("https://example.com/page") {
		t.Fatal("expected /page allowed by robots checker")
	}
	if !c.shouldSkipURL("https://example.com/private") {
		t.Fatal("expected /private skipped by robots checker")
	}
}
