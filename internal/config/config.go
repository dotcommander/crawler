package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// CrawlerConfig represents the final configuration for the crawler
type CrawlerConfig struct {
	StartURL         string
	StartURLs        []string // All seed URLs (includes StartURL); used for multi-URL crawling
	OutputDir        string
	CacheDir         string // Directory for cache files
	MaxDepth         int
	Concurrency      int
	DefaultDelay     time.Duration
	MaxRetries       int
	Force            bool
	DomainDelays     map[string]time.Duration
	ExcludePatterns  []string
	UserAgent        string
	MobileUserAgent  string
	Headers          map[string]string
	Mobile           bool
	MaxPages         int
	WaitStrategy     string            // Playwright wait strategy: "commit", "load", "domcontentloaded", "networkidle"
	ExtraWaitTime    time.Duration     // Additional wait after page load
	JSCrawl          bool              // Extract endpoints from JavaScript content
	ExportFormat     string            // Structured output format: jsonl, csv, sitemap
	ExportFile       string            // Export output file path (empty = stdout)
	Resume           bool              // Resume a previous crawl session
	NoRobots         bool              // Skip robots.txt/sitemap seeding
	SeedURLs         []string          // Pre-seed URLs from sitemap/robots.txt
	ExtractSelectors map[string]string // CSS selectors for field extraction
	Quiet            bool              // Pipeline mode: suppress all output except export
}

// LoadConfigWithViper loads configuration using the new Viper-based system
func LoadConfigWithViper(configFile, startURL, outputDir string, profile string, mobile bool, maxPages int) (*CrawlerConfig, error) {
	vcm := NewViperConfigManager()

	if err := vcm.LoadConfig(configFile); err != nil {
		return nil, err
	}

	if profile != "" {
		vcm.ApplyProfile(profile)
	}

	return vcm.BuildCrawlerConfig(startURL, outputDir, mobile, maxPages)
}

// GetCacheDir returns the appropriate cache directory for the crawler
func GetCacheDir() string {
	// Check if CRAWLER_CACHE_DIR is set
	if cacheDir := os.Getenv("CRAWLER_CACHE_DIR"); cacheDir != "" {
		return cacheDir
	}

	// Check for XDG_CACHE_HOME first (Linux/Unix standard)
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "crawler")
	}

	// Fall back to OS-specific paths
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, use temp directory
		return filepath.Join(os.TempDir(), "crawler-cache")
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "crawler")
	case "windows":
		return filepath.Join(home, "AppData", "Local", "crawler", "cache")
	default: // linux and others
		return filepath.Join(home, ".cache", "crawler")
	}
}

// GetDefaultOutputDir returns the default output directory for crawled content
func GetDefaultOutputDir() string {
	// Check if CRAWLER_OUTPUT_DIR is set
	if outputDir := os.Getenv("CRAWLER_OUTPUT_DIR"); outputDir != "" {
		return outputDir
	}

	// Always use ~/.config/crawler/storage for all platforms
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "crawler-storage")
	}
	return filepath.Join(home, ".config", "crawler", "storage")
}
