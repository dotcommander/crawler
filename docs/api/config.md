# API Reference: Config Package

The `config.CrawlerConfig` struct defines all configuration options for the crawler.
Configuration values are loaded from YAML files, environment variables, and CLI flags
with the following precedence: CLI flags > environment variables > config file > defaults.

## Type Definition

```go
package config

// CrawlerConfig represents the final configuration for the crawler
type CrawlerConfig struct {
    StartURL        string
    OutputDir       string
    CacheDir        string
    MaxDepth        int
    Concurrency     int
    DefaultDelay    time.Duration
    MaxRetries      int
    Force           bool
    DomainDelays    map[string]time.Duration
    ExcludePatterns []string
    UserAgent       string
    Headers         map[string]string
    Mobile          bool
    MaxPages        int
    WaitStrategy    string
    ExtraWaitTime   time.Duration
}
```

## Field Reference

| Field | Type | Default | CLI Flag | Env Var | Config Key | Description |
|-------|------|---------|----------|---------|-------------|-------------|
| `StartURL` | `string` | *required* | positional arg | — | — | The URL where crawling begins. Must be a valid HTTP/HTTPS URL. |
| `OutputDir` | `string` | `~/.config/crawler/storage` | `--output` | `CRAWLER_OUTPUT_DIR` | — | Directory where crawled content is saved. Resolved to absolute path. |
| `CacheDir` | `string` | XDG cache path | — | `CRAWLER_CACHE_DIR` | — | Directory for cache files. Follows XDG Base Directory specification. |
| `MaxDepth` | `int` | `3` | `--depth` | `CRAWLER_DEPTH` | `depth` | Maximum crawl depth from start URL. `0` means only start page. |
| `Concurrency` | `int` | `5` | `--concurrency` | `CRAWLER_CONCURRENCY` | `concurrency` | Number of concurrent workers for crawling. Higher values = faster but more resource intensive. |
| `DefaultDelay` | `time.Duration` | `1s` | `--delay` | `CRAWLER_DELAY` | `delay` | Default delay between requests to the same domain. Formatted as duration string (e.g., "500ms", "2s"). |
| `MaxRetries` | `int` | `2` | `--max-retries` | `CRAWLER_MAXRETRIES` | `maxRetries` | Maximum number of retry attempts for failed requests. |
| `Force` | `bool` | `false` | `--force` | `CRAWLER_FORCE` | `force` | Overwrite existing files in output directory. When `false`, existing files are skipped. |
| `DomainDelays` | `map[string]time.Duration` | `{}` | — | — | `domainDelays` | Per-domain delay overrides. Keys are domain names, values are duration strings. Example: `{"slow.com": "5s"}`. |
| `ExcludePatterns` | `[]string` | `[]` | — | — | `ignorePatterns` | URL patterns to exclude from crawling. Supports glob patterns. Examples: `"*.pdf"`, `"*logout*"`. |
| `UserAgent` | `string` | `""` | `--user-agent` | `CRAWLER_USERAGENT` | `userAgent` | Custom User-Agent string for HTTP requests. Empty string uses engine default. |
| `Headers` | `map[string]string` | `{}` | — | — | `headers` | Custom HTTP headers to include in all requests. Key-value pairs of header names and values. |
| `Mobile` | `bool` | `false` | `--mobile` | `CRAWLER_MOBILE` | `mobile` | Enable mobile device emulation. When `true`, forces Playwright engine with iPhone 14 user agent. |
| `MaxPages` | `int` | `0` | `--max-pages` | `CRAWLER_MAXPAGES` | `maxPages` | Maximum number of pages to crawl. `0` means unlimited. Stops crawler when limit is reached. |
| `WaitStrategy` | `string` | `"networkidle"` | — | — | `waitStrategy` | Playwright wait strategy: `"commit"`, `"load"`, `"domcontentloaded"`, `"networkidle"`. Non-default values force Playwright engine. |
| `ExtraWaitTime` | `time.Duration` | `500ms` | — | — | `extraWaitTime` | Additional wait time after page load completes. Values >500ms force Playwright engine. |

## Configuration Precedence

Configuration values are merged in the following order (later sources override earlier ones):

1. **Defaults** - Hardcoded default values in `setDefaults()`
2. **Config file** - YAML file from search paths
3. **Environment variables** - `CRAWLER_*` prefixed variables
4. **CLI flags** - Command-line arguments take precedence

### Config File Search Paths

The crawler searches for `crawl.yaml` in the following order:

1. `./` (current directory)
2. `$HOME/.config/crawler/` (XDG config home)
3. Path specified by `--config` flag

### Environment Variable Format

Environment variables use `CRAWLER_` prefix and uppercase names with underscores:

```bash
export CRAWLER_DEPTH=5
export CRAWLER_CONCURRENCY=10
export CRAWLER_DELAY="2s"
export CRAWLER_OUTPUT_DIR="/tmp/crawl-output"
```

Nested config keys use double underscore:

```bash
export CRAWLER_DOMAIN_DELAYS__SLOW_COM="5s"
```

## Usage Examples

### Basic Programmatic Usage

```go
package main

import (
    "crawler/internal/config"
    "crawler/internal/crawlers"
    "log"
    "time"
)

func main() {
    cfg := &config.CrawlerConfig{
        StartURL:     "https://example.com",
        OutputDir:    "./output",
        MaxDepth:     2,
        MaxPages:     100,
        Concurrency:  5,
        DefaultDelay: 1 * time.Second,
    }

    crawler, err := crawlers.CreateCrawler(cfg, true, "colly")
    if err != nil {
        log.Fatal(err)
    }
    defer crawler.Close()

    if err := crawler.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Loading from Config File

```go
// Load from ~/.config/crawler/crawl.yaml or custom path
cfg, err := config.LoadConfigWithViper(
    "",               // configFile (empty = use search paths)
    "https://example.com",
    "./output",
    "",               // profile (empty = no profile)
    false,            // mobile
    0,                // maxPages (0 = unlimited)
)
if err != nil {
    log.Fatal(err)
}
```

### Using Profiles

```go
vcm := config.NewViperConfigManager()
vcm.LoadConfig("")

// Apply preset profile
vcm.ApplyProfile("fast")   // or "safe", "thorough"

cfg, err := vcm.BuildCrawlerConfig(startURL, outputDir, mobile, maxPages)
```

**Available Profiles:**

| Profile | Settings |
|---------|----------|
| `fast` | delay=0.5s, concurrency=10, depth=2, waitStrategy=domcontentloaded |
| `safe` | delay=2s, concurrency=3, depth=5, waitStrategy=networkidle |
| `thorough` | delay=3s, concurrency=2, depth=10, waitStrategy=networkidle |

### Domain-Specific Configuration

```go
cfg := &config.CrawlerConfig{
    StartURL:     "https://example.com",
    DefaultDelay: 1 * time.Second,
    DomainDelays: map[string]time.Duration{
        "slow-api.example.com": 5 * time.Second,
        "rate-limited.com":     2 * time.Second,
    },
}
```

### URL Exclusion Patterns

```go
cfg := &config.CrawlerConfig{
    StartURL:        "https://example.com",
    ExcludePatterns: []string{"*.pdf", "*logout*", "/admin/*"},
}
```

## Engine Selection Impact

Certain configuration fields affect automatic engine selection:

| Config | Effect |
|--------|--------|
| `Mobile: true` | Forces Playwright engine (iPhone 14 emulation) |
| `WaitStrategy != "networkidle"` | Forces Playwright engine |
| `ExtraWaitTime > 500ms` | Suggests Playwright for JavaScript-heavy sites |

## XDG Base Directory Compliance

The crawler follows XDG Base Directory specification for configuration and cache locations:

### Config Directories

| Platform | Config Path | Cache Path |
|----------|-------------|------------|
| Linux | `~/.config/crawler/` | `~/.cache/crawler/` |
| macOS | `~/.config/crawler/` | `~/Library/Caches/crawler/` |
| Windows | `%APPDATA%\crawler\` | `%LOCALAPPDATA%\crawler\cache\` |

Environment variable overrides:
- `XDG_CONFIG_HOME` - Config directory location
- `XDG_CACHE_HOME` - Cache directory location
- `CRAWLER_CACHE_DIR` - Direct override for cache directory
- `CRAWLER_OUTPUT_DIR` - Direct override for output directory

## Functions

### `LoadConfigWithViper(configFile, startURL, outputDir string, profile string, mobile bool, maxPages int) (*CrawlerConfig, error)`

Loads configuration using Viper with support for YAML files, environment variables,
and profiles.

**Parameters:**
- `configFile` - Optional path to config file (empty uses search paths)
- `startURL` - Required starting URL for crawling
- `outputDir` - Output directory (empty uses default)
- `profile` - Optional profile name ("fast", "safe", "thorough")
- `mobile` - Enable mobile emulation
- `maxPages` - Maximum pages to crawl (0 = unlimited)

**Returns:** `*CrawlerConfig` and error

### `GetCacheDir() string`

Returns the XDG-compliant cache directory path.

**Returns:** Absolute path to cache directory

## Configuration File Example

```yaml
# ~/.config/crawler/crawl.yaml

# Basic crawling settings
depth: 3
delay: 1.0
maxRetries: 2
concurrency: 5

# Page limits
maxPages: 100  # 0 for unlimited

# User agent and headers
userAgent: "Mozilla/5.0 (compatible; Crawler/1.0)"
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
waitStrategy: "networkidle"
extraWaitTime: "500ms"

# Other settings
force: false
```
