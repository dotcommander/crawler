# Basic Crawling Example

A step-by-step guide to performing basic web crawling operations.

## Quick Start

The simplest way to crawl a website:

```bash
./crawler https://example.com
```

This will crawl `example.com` using default settings:
- **Engine**: Colly (fast HTTP-based crawler)
- **Depth**: 2 levels from the start URL
- **Concurrency**: 5 workers
- **Output**: `~/.config/crawler/storage/`

## Common Examples

### Crawl a Single Page

Limit the crawl to just the starting page:

```bash
./crawler --max-pages 1 https://example.com
```

**Expected output:**
```
▌ Crawler starting...
  Engine: colly
  Target: https://example.com
  Max pages: 1

▌ Crawling https://example.com...
  ✓ Fetched (2.3KB)

▌ Complete!
  Pages: 1
  Time: 1.2s
```

### Crawl with Verbose Logging

See detailed progress information:

```bash
./crawler --verbose https://example.com
```

**Expected output:**
```
[INFO] Starting crawler with engine: colly
[INFO] Start URL: https://example.com
[INFO] Output directory: /Users/you/.config/crawler/storage
[INFO] Concurrency: 5 workers
[INFO] Max depth: 2
[INFO] Max pages: 0 (unlimited)
[INFO] Starting crawl...
[INFO] Fetching: https://example.com
[INFO] Status: 200 OK
[INFO] Size: 2.3KB
[INFO] Found 5 links
[INFO] Fetching: https://example.com/about
[INFO] Status: 200 OK
...
```

### Crawl Specific Path

Crawl only a specific section of a website:

```bash
./crawler https://example.com/docs/
```

The crawler automatically restricts itself to:
- Same domain (`example.com`)
- Same base path (`/docs/`)

**Expected output:**
```
✓ https://example.com/docs/
✓ https://example.com/docs/api.html
✓ https://example.com/docs/tutorial.html
✗ https://example.com/blog/ (skipped - outside base path)
✗ https://other.com/page (skipped - different domain)
```

### Crawl with Custom Settings

Adjust crawl behavior with common options:

```bash
./crawler --depth 3 --concurrency 10 --delay 0.5 https://example.com
```

- `--depth 3`: Follow links up to 3 levels deep
- `--concurrency 10`: Use 10 parallel workers
- `--delay 0.5`: Wait 0.5 seconds between requests

### Mobile Emulation Crawl

Crawl as a mobile device (automatically selects Playwright engine):

```bash
./crawler --mobile --max-pages 5 https://example.com
```

**Expected output:**
```
▌ Crawler starting...
  Engine: playwright (auto-selected for mobile)
  Device: iPhone 14
  Viewport: 390x844

▌ Crawling https://example.com...
  ✓ Fetched (45.2KB) - mobile optimized
```

## Engine Selection

### Colly Engine (Default)

Fast HTTP-based crawling for static content:

```bash
./crawler --engine colly https://example.com
```

**Best for:**
- Static HTML sites
- High-performance crawling
- Limited memory usage

**Not for:**
- JavaScript-rendered content
- Sites requiring browser interaction

### Playwright Engine

Full browser automation for dynamic content:

```bash
./crawler --engine playwright https://spa-example.com
```

**Best for:**
- Single-page applications (SPAs)
- JavaScript-heavy sites
- Sites requiring user interaction

**Expected output:**
```
▌ Crawler starting...
  Engine: playwright
  Browser: headless chromium

▌ Crawling https://spa-example.com...
  ✓ Waiting for network idle...
  ✓ Page rendered (156.7KB)
  ✓ JavaScript executed
```

## Output Location

Crawled content is saved to:

```bash
# macOS
~/.config/crawler/storage/

# Linux
~/.config/crawler/storage/

# Custom location
./crawler --output ./my-output https://example.com
```

**Directory structure:**
```
storage/
├── example.com/
│   ├── index.html
│   ├── about/
│   │   └── index.html
│   └── contact/
│       └── index.html
└── .cache/
    └── crawler_cache.txt
```

## Configuration File

For reusable settings, create a `crawl.yml` file:

```yaml
# crawl.yml
depth: 3
concurrency: 10
delay: 1.0
maxPages: 100
output: "./output"

# Per-domain rate limiting
domainDelays:
  api.example.com: 2.0
  slow-site.com: 5.0

# Exclude specific patterns
excludePatterns:
  - "*/admin/*"
  - "*.pdf"
  - "*/login"
```

Then run:

```bash
./crawler --config crawl.yml https://example.com
```

## Troubleshooting

### Crawler Seems Stuck

Some pages take time to load. Use `--verbose` to see what's happening:

```bash
./crawler --verbose https://slow-site.com
```

### Too Many Pages Crawled

Limit the scope with `--max-pages`:

```bash
./crawler --max-pages 10 https://example.com
```

### JavaScript Not Rendering

Switch to Playwright engine:

```bash
./crawler --engine playwright https://spa-site.com
```

### Rate Limiting Issues

Increase delay between requests:

```bash
./crawler --delay 2.0 https://example.com
```

Or use per-domain delays in config:

```yaml
domainDelays:
  example.com: 3.0
```

## Next Steps

- [Configuration Reference](/api/config) - All configuration options
- [API Reference](/api/crawler) - Programmatic usage
- [URL Filtering Guide](/guides/url-filtering) - Advanced filtering patterns
