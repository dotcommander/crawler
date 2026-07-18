# Web Crawler (Go Version)

A high-performance web crawler written in Go using Playwright for browser automation. Features mobile device emulation, concurrent crawling, and advanced page limits.

## Features

- 🚀 Concurrent crawling with configurable workers
- 📱 Mobile device emulation (iPhone 14)
- 🎯 Max pages limit to control crawl scope
- 🔄 Circuit breaker for failing domains
- ⚡ Per-domain rate limiting
- 💾 Persistent cache for visited URLs
- 📊 Real-time progress reporting
- 🎨 Enhanced UI with worker visualization
- 🔒 Smart URL filtering:
  - Stays within the same domain as start URL
  - Stays within the base path of start URL
  - Ignores non-HTTP schemes (mailto:, tel:, javascript:, ftp:, etc.)
  - Filters Apache directory listing sort parameters

## Installation

```bash
go build -o crawler .
```

## Usage

### Basic Usage

```bash
./crawler [options] <startUrl>
```

### Options

- `-v, --verbose`         Enable verbose logging
- `-f, --force`           Force re-crawl even if URL was visited
- `-d, --depth <n>`       Maximum crawl depth (default: 2)
- `-l, --delay <s>`       Delay between requests in seconds (default: 1.0)
- `-r, --max-retries <n>` Maximum retries for failed requests (default: 2)
- `-c, --concurrency <n>` Number of concurrent workers (default: 5)
- `-C, --config <file>`   Config file path (default: crawl.yml)
- `-o, --output <dir>`    Output directory for crawled content
- `-m, --mobile`          Enable mobile browser emulation
- `--max-pages <n>`       Maximum number of pages to crawl (0 = unlimited)

### Mobile Crawling

Enable mobile device emulation to crawl sites as they appear on mobile devices:

```bash
./crawler --mobile https://example.com
```

Mobile mode configures:
- User Agent: iOS Safari on iPhone
- Viewport: 390x844 pixels (iPhone 14)
- Touch: Enabled
- Device Scale Factor: 2.0x (Retina display)

### Limiting Crawl Scope

Use `--max-pages` to limit the total number of pages crawled:

```bash
# Crawl only the homepage
./crawler --max-pages 1 https://example.com

# Crawl up to 10 pages
./crawler --max-pages 10 https://example.com
```

### Examples

```bash
# Basic crawl with verbose output
./crawler -v https://example.com

# Mobile crawl of single page
./crawler --mobile --max-pages 1 https://example.com

# Deep crawl with custom settings
./crawler --depth 5 --concurrency 10 --delay 0.5 https://example.com

# Force re-crawl with mobile emulation
./crawler -f --mobile https://example.com
```

## Configuration

Create a `crawl.yml` file for persistent configuration:

```yaml
force: false
depth: 3
delay: 1.0
maxRetries: 3
concurrency: 5
mobile: false
maxPages: 0
userAgent: "Custom User Agent"
headers:
  Authorization: "Bearer token"
domainDelays:
  api.example.com: 2.0
  slow-site.com: 5.0
ignorePatterns:
  - "*/admin/*"
  - "*.pdf"
```

## Page Comparison Tools

### HTTP-based Comparison

Quick comparison using HTTP requests:

```bash
./cmd/compare-page/compare-page [--mobile] <url>
```

### Browser-based Comparison

Full browser-based comparison with JavaScript execution:

```bash
./cmd/compare-page-crawl/compare-page-crawl [--mobile] <url>
```

## Output

Crawled content is saved to the output directory with the following structure:

```
output/
├── example.com/
│   ├── index.html
│   ├── about/
│   │   └── index.html
│   └── products/
│       ├── index.html
│       └── widget.html
└── cache/
    └── crawl_cache.txt
```

## Cache

The crawler maintains a cache of visited URLs to avoid re-crawling. Cache location:
- macOS: `~/Library/Caches/crawler/`
- Linux: `~/.cache/crawler/`
- Windows: `%LOCALAPPDATA%\crawler\cache\`

Override with `CRAWLER_CACHE_DIR` environment variable.

## URL Filtering and Scope Control

The crawler automatically filters URLs to stay within scope:

### Domain Restriction
The crawler only follows links on the same domain as the start URL:
```bash
# Starting from https://example.com will only crawl example.com
./crawler https://example.com/docs/
# ✅ https://example.com/docs/api.html
# ✅ https://example.com/about.html
# ❌ https://other-site.com/page.html
```

### Path Restriction
The crawler stays within the base path of the start URL:
```bash
# Starting from a specific path
./crawler https://example.com/docs/guide.html
# Base path is /docs/
# ✅ https://example.com/docs/api.html
# ✅ https://example.com/docs/tutorial/
# ❌ https://example.com/blog/
# ❌ https://example.com/about.html
```

### Non-HTTP Schemes
The crawler automatically ignores non-HTTP links:
- ❌ `mailto:` links (email addresses)
- ❌ `tel:` links (phone numbers)
- ❌ `javascript:` links
- ❌ `ftp:` links
- ❌ Any other non-HTTP/HTTPS schemes

### Example Scenarios

```bash
# Crawl only the /blog section
./crawler https://site.com/blog/

# Crawl starting from a specific page
./crawler https://site.com/docs/api.html
# Will crawl /docs/* but not parent directories

# Crawl with pattern exclusions
./crawler --config crawl.yml https://site.com
# Use ignorePatterns in config to exclude specific paths
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/your-repo/crawler.git
cd crawler

# Build the main crawler
go build -o crawler .

# Build comparison tools
go build -o compare-page ./cmd/compare-page
go build -o compare-page-crawl ./cmd/compare-page-crawl

# Run tests
go test ./...
```

## License

MIT