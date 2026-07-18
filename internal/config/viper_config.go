package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ViperConfigManager handles configuration using Viper for better integration
type ViperConfigManager struct {
	v *viper.Viper
}

// NewViperConfigManager creates a new Viper-based config manager
func NewViperConfigManager() *ViperConfigManager {
	v := viper.New()

	// Set up configuration structure and defaults
	v.SetConfigName("crawl")
	v.SetConfigType("yaml")

	// Add configuration search paths with XDG compliance
	v.AddConfigPath(getXDGConfigDir())
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/crawler")

	// Set up environment variable support
	v.SetEnvPrefix("CRAWLER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set sensible defaults
	setDefaults(v)

	return &ViperConfigManager{v: v}
}

// setDefaults configures reasonable default values
func setDefaults(v *viper.Viper) {
	v.SetDefault("depth", 3)
	v.SetDefault("delay", 1.0)
	v.SetDefault("maxRetries", 2)
	v.SetDefault("concurrency", 5)
	v.SetDefault("mobile", false)
	v.SetDefault("force", false)
	v.SetDefault("maxPages", 0)
	v.SetDefault("userAgent", "")
	v.SetDefault("mobileUserAgent", "")
	v.SetDefault("headers", map[string]string{})
	v.SetDefault("domainDelays", map[string]float64{})
	v.SetDefault("ignorePatterns", []string{})
	v.SetDefault("waitStrategy", "networkidle")
	v.SetDefault("extraWaitTime", "500ms")
}

// getXDGConfigDir returns the XDG-compliant config directory
func getXDGConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "crawler")
	}

	// Fall back to OS-specific paths
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".config", "crawler")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "crawler")
	default: // linux and others
		return filepath.Join(home, ".config", "crawler")
	}
}

// LoadConfig loads configuration with optional config file override
func (vcm *ViperConfigManager) LoadConfig(configFile string) error {
	if configFile != "" {
		// Use specific config file
		vcm.v.SetConfigFile(configFile)
	}

	// Read config file
	if err := vcm.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; using defaults and environment variables
			return nil
		}
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// BuildCrawlerConfig creates a CrawlerConfig from the loaded Viper configuration
func (vcm *ViperConfigManager) BuildCrawlerConfig(startURL, outputDir string, mobile bool, maxPages int) (*CrawlerConfig, error) {
	// CLI flags override config file values
	if mobile {
		vcm.v.Set("mobile", true)
	}
	if maxPages > 0 {
		vcm.v.Set("maxPages", maxPages)
	}

	// Set default output directory if not specified
	if outputDir == "" {
		outputDir = GetDefaultOutputDir()
	}

	// Ensure output directory is absolute
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}

	// Parse domain delays
	domainDelaysRaw := vcm.v.GetStringMapString("domainDelays")
	domainDelays := make(map[string]time.Duration)
	for domain, delayStr := range domainDelaysRaw {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			domainDelays[domain] = delay
		} else {
			// Try parsing as float seconds (backward compatibility)
			if delayFloat := vcm.v.GetFloat64("domainDelays." + domain); delayFloat > 0 {
				domainDelays[domain] = time.Duration(delayFloat * float64(time.Second))
			}
		}
	}

	// Parse extra wait time
	extraWaitTime, err := time.ParseDuration(vcm.v.GetString("extraWaitTime"))
	if err != nil {
		extraWaitTime = 500 * time.Millisecond // fallback
	}

	config := &CrawlerConfig{
		StartURL:        startURL,
		OutputDir:       absOutputDir,
		CacheDir:        GetCacheDir(),
		MaxDepth:        vcm.v.GetInt("depth"),
		Concurrency:     vcm.v.GetInt("concurrency"),
		DefaultDelay:    time.Duration(vcm.v.GetFloat64("delay") * float64(time.Second)),
		MaxRetries:      vcm.v.GetInt("maxRetries"),
		Force:           vcm.v.GetBool("force"),
		DomainDelays:    domainDelays,
		ExcludePatterns: vcm.v.GetStringSlice("ignorePatterns"),
		UserAgent:       vcm.v.GetString("userAgent"),
		MobileUserAgent: vcm.v.GetString("mobileUserAgent"),
		Headers:         vcm.v.GetStringMapString("headers"),
		Mobile:          vcm.v.GetBool("mobile"),
		MaxPages:        vcm.v.GetInt("maxPages"),
		WaitStrategy:    vcm.v.GetString("waitStrategy"),
		ExtraWaitTime:   extraWaitTime,
	}

	return config, nil
}

// ApplyProfile applies preset configurations for different use cases
func (vcm *ViperConfigManager) ApplyProfile(profile string) {
	switch profile {
	case "fast":
		vcm.v.Set("delay", 0.5)
		vcm.v.Set("concurrency", 10)
		vcm.v.Set("depth", 2)
		vcm.v.Set("extraWaitTime", "200ms")
		vcm.v.Set("waitStrategy", "domcontentloaded")
	case "safe":
		vcm.v.Set("delay", 2.0)
		vcm.v.Set("concurrency", 3)
		vcm.v.Set("depth", 5)
		vcm.v.Set("extraWaitTime", "1s")
		vcm.v.Set("waitStrategy", "networkidle")
	case "thorough":
		vcm.v.Set("delay", 3.0)
		vcm.v.Set("concurrency", 2)
		vcm.v.Set("depth", 10)
		vcm.v.Set("extraWaitTime", "2s")
		vcm.v.Set("waitStrategy", "networkidle")
	}
}

// GetConfigFileUsed returns the config file that was actually used
func (vcm *ViperConfigManager) GetConfigFileUsed() string {
	return vcm.v.ConfigFileUsed()
}

// WriteConfigExample writes an example configuration file
func (vcm *ViperConfigManager) WriteConfigExample(path string) error {
	exampleConfig := `# Crawler Configuration Example
# Place this file in ~/.config/crawler/crawl.yaml

# Basic crawling settings
depth: 3
delay: 1.0  # seconds between requests
maxRetries: 2
concurrency: 5

# Page limits
maxPages: 100  # 0 for unlimited

# User agent and headers
# userAgent: ""   # blank uses the built-in desktop default; set to override
# mobileUserAgent: ""   # blank uses the built-in mobile default; set to override
userAgent: ""
headers:
  Accept: "text/html,application/xhtml+xml"

# Mobile device emulation
mobile: false

# Domain-specific delays (seconds)
domainDelays:
  slow-site.com: 5.0
  api.example.com: 2.0

# URL patterns to ignore
ignorePatterns:
  - "*.pdf"
  - "*logout*"
  - "*admin*"

# Playwright settings
waitStrategy: "networkidle"  # commit, load, domcontentloaded, networkidle
extraWaitTime: "500ms"

# Other settings
force: false  # Overwrite existing files
`

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(exampleConfig), 0644)
}
