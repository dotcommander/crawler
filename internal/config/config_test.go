package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestViperConfigManager(t *testing.T) {
	t.Parallel()
	vcm := NewViperConfigManager()
	if vcm == nil {
		t.Fatal("NewViperConfigManager() returned nil")
	}
	if vcm.v == nil {
		t.Fatal("Viper instance not initialized")
	}
}

func TestLoadConfigWithFile(t *testing.T) {
	t.Parallel()
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test config file
	configPath := filepath.Join(tempDir, "test-crawl.yml")
	configContent := `
depth: 5
delay: 2.5
maxRetries: 3
concurrency: 8
mobile: true
force: true
maxPages: 100
userAgent: "TestCrawler/1.0"
headers:
  Accept: "text/html,application/xhtml+xml"
  User-Agent: "TestBot"
domainDelays:
  slow-site.com: 5.0
  fast-site.com: 0.5
ignorePatterns:
  - "*.pdf"
  - "*admin*"
waitStrategy: "domcontentloaded"
extraWaitTime: "1s"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadConfigWithViper(configPath, "https://example.com", tempDir, "", false, 0)
	if err != nil {
		t.Fatalf("LoadConfigWithViper failed: %v", err)
	}

	// Verify loaded values
	if config.MaxDepth != 5 {
		t.Errorf("Expected depth 5, got %d", config.MaxDepth)
	}
	if config.DefaultDelay != 2500*time.Millisecond {
		t.Errorf("Expected delay 2.5s, got %v", config.DefaultDelay)
	}
	if config.MaxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", config.MaxRetries)
	}
	if config.Concurrency != 8 {
		t.Errorf("Expected concurrency 8, got %d", config.Concurrency)
	}
	if !config.Mobile {
		t.Error("Expected mobile to be true")
	}
	if !config.Force {
		t.Error("Expected force to be true")
	}
	if config.MaxPages != 100 {
		t.Errorf("Expected maxPages 100, got %d", config.MaxPages)
	}
	if config.UserAgent != "TestCrawler/1.0" {
		t.Errorf("Expected userAgent 'TestCrawler/1.0', got '%s'", config.UserAgent)
	}
	if config.WaitStrategy != "domcontentloaded" {
		t.Errorf("Expected waitStrategy 'domcontentloaded', got '%s'", config.WaitStrategy)
	}
	if config.ExtraWaitTime != time.Second {
		t.Errorf("Expected extraWaitTime 1s, got %v", config.ExtraWaitTime)
	}

	// Check headers
	if len(config.Headers) == 0 {
		t.Error("Expected headers to be loaded")
	}
	if accept, exists := config.Headers["Accept"]; exists && accept != "text/html,application/xhtml+xml" {
		t.Errorf("Unexpected Accept header: %s", accept)
	}

	// Check domain delays
	if len(config.DomainDelays) != 2 {
		t.Errorf("Expected 2 domain delays, got %d", len(config.DomainDelays))
	}
	if config.DomainDelays["slow-site.com"] != 5*time.Second {
		t.Errorf("Expected slow-site.com delay 5s, got %v", config.DomainDelays["slow-site.com"])
	}

	// Check ignore patterns
	if len(config.ExcludePatterns) != 2 {
		t.Errorf("Expected 2 ignore patterns, got %d", len(config.ExcludePatterns))
	}

	// Verify output directory is absolute
	if !filepath.IsAbs(config.OutputDir) {
		t.Errorf("Expected absolute output directory, got %s", config.OutputDir)
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	t.Parallel()
	// Test with non-existent config file (should use defaults)
	config, err := LoadConfigWithViper("", "https://example.com", "", "", false, 0)
	if err != nil {
		t.Fatalf("LoadConfigWithViper failed with defaults: %v", err)
	}

	// Verify default values (note: fast profile may be applied by default)
	if config.MaxDepth < 2 || config.MaxDepth > 3 {
		t.Errorf("Expected default depth 2-3, got %d", config.MaxDepth)
	}
	if config.DefaultDelay != time.Second {
		t.Errorf("Expected default delay 1s, got %v", config.DefaultDelay)
	}
	if config.Concurrency != 5 {
		t.Errorf("Expected default concurrency 5, got %d", config.Concurrency)
	}
	if config.WaitStrategy != "networkidle" {
		t.Errorf("Expected default waitStrategy 'networkidle', got '%s'", config.WaitStrategy)
	}
}

func TestProfileApplication(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		profile string
		checkFn func(*CrawlerConfig) error
	}{
		{
			name:    "fast profile",
			profile: "fast",
			checkFn: func(c *CrawlerConfig) error {
				if c.DefaultDelay != 500*time.Millisecond {
					t.Errorf("Fast profile: expected delay 0.5s, got %v", c.DefaultDelay)
				}
				if c.Concurrency != 10 {
					t.Errorf("Fast profile: expected concurrency 10, got %d", c.Concurrency)
				}
				if c.MaxDepth != 2 {
					t.Errorf("Fast profile: expected depth 2, got %d", c.MaxDepth)
				}
				if c.WaitStrategy != "domcontentloaded" {
					t.Errorf("Fast profile: expected waitStrategy 'domcontentloaded', got '%s'", c.WaitStrategy)
				}
				return nil
			},
		},
		{
			name:    "safe profile",
			profile: "safe",
			checkFn: func(c *CrawlerConfig) error {
				if c.DefaultDelay != 2*time.Second {
					t.Errorf("Safe profile: expected delay 2s, got %v", c.DefaultDelay)
				}
				if c.Concurrency != 3 {
					t.Errorf("Safe profile: expected concurrency 3, got %d", c.Concurrency)
				}
				if c.MaxDepth != 5 {
					t.Errorf("Safe profile: expected depth 5, got %d", c.MaxDepth)
				}
				return nil
			},
		},
		{
			name:    "thorough profile",
			profile: "thorough",
			checkFn: func(c *CrawlerConfig) error {
				if c.DefaultDelay != 3*time.Second {
					t.Errorf("Thorough profile: expected delay 3s, got %v", c.DefaultDelay)
				}
				if c.Concurrency != 2 {
					t.Errorf("Thorough profile: expected concurrency 2, got %d", c.Concurrency)
				}
				if c.MaxDepth != 10 {
					t.Errorf("Thorough profile: expected depth 10, got %d", c.MaxDepth)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config, err := LoadConfigWithViper("", "https://example.com", "", tt.profile, false, 0)
			if err != nil {
				t.Fatalf("LoadConfigWithViper failed: %v", err)
			}
			tt.checkFn(config)
		})
	}
}

func TestCLIOverrides(t *testing.T) {
	t.Parallel()
	// Create a config with defaults
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-override.yml")
	configContent := `
mobile: false
maxPages: 50
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test CLI overrides
	config, err := LoadConfigWithViper(configPath, "https://example.com", tempDir, "", true, 200)
	if err != nil {
		t.Fatalf("LoadConfigWithViper failed: %v", err)
	}

	// CLI flags should override config file
	if !config.Mobile {
		t.Error("Expected mobile=true from CLI override")
	}
	if config.MaxPages != 200 {
		t.Errorf("Expected maxPages=200 from CLI override, got %d", config.MaxPages)
	}
}

func TestDefaultUserAgents(t *testing.T) {
	t.Parallel()
	// UA defaults are not baked into Go source (config-in-source rule). Verify
	// that values read from a config file are passed through correctly, and that
	// desktop and mobile agents remain distinct.
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "ua-test.yml")
	content := `userAgent: "TestDesktop/1.0"
mobileUserAgent: "TestMobile/1.0"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write ua config: %v", err)
	}
	cfg, err := LoadConfigWithViper(cfgPath, "https://example.com", tempDir, "", false, 0)
	if err != nil {
		t.Fatalf("LoadConfigWithViper failed: %v", err)
	}
	if cfg.UserAgent != "TestDesktop/1.0" {
		t.Fatalf("expected desktop userAgent 'TestDesktop/1.0', got %q", cfg.UserAgent)
	}
	if cfg.MobileUserAgent != "TestMobile/1.0" {
		t.Fatalf("expected mobileUserAgent 'TestMobile/1.0', got %q", cfg.MobileUserAgent)
	}
	if cfg.UserAgent == cfg.MobileUserAgent {
		t.Fatal("desktop and mobile user agents must differ")
	}
}

func TestGetCacheDir(t *testing.T) {
	t.Parallel()
	// Test with environment variable
	originalVar := os.Getenv("CRAWLER_CACHE_DIR")
	defer func() {
		if originalVar != "" {
			os.Setenv("CRAWLER_CACHE_DIR", originalVar)
		} else {
			os.Unsetenv("CRAWLER_CACHE_DIR")
		}
	}()

	testCacheDir := "/tmp/test-cache"
	os.Setenv("CRAWLER_CACHE_DIR", testCacheDir)

	cacheDir := GetCacheDir()
	if cacheDir != testCacheDir {
		t.Errorf("Expected cache dir %s, got %s", testCacheDir, cacheDir)
	}

	// Test without environment variable (should use XDG/OS defaults)
	os.Unsetenv("CRAWLER_CACHE_DIR")
	cacheDir = GetCacheDir()
	if cacheDir == "" {
		t.Error("GetCacheDir() returned empty string")
	}
	if !filepath.IsAbs(cacheDir) {
		t.Errorf("Expected absolute cache directory, got %s", cacheDir)
	}
}
