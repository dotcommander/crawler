# Error Handling Guide

Comprehensive guide to understanding, diagnosing, and resolving crawler errors.

## Overview

The crawler implements a multi-layered error handling strategy:

- **Circuit Breaker Pattern** - Prevents cascading failures from problematic domains
- **Engine-Specific Handlers** - Colly and Playwright have tailored error recovery
- **Progress Reporting** - Real-time error visibility via UI or verbose logging
- **Graceful Shutdown** - Clean cancellation and resource cleanup

---

## Common Error Types

### 1. Circuit Breaker Errors

#### Symptom
```
WARN: Domain example.com is circuit broken, skipping https://example.com/page
WARN: Circuit breaker open for domain example.com, skipping https://example.com/page
```

#### Causes
- **3 consecutive failures** on same domain triggers circuit breaker
- Domain temporarily down or overloaded
- Rate limiting triggered (429 Too Many Requests)
- Network connectivity issues specific to domain
- DNS resolution failures

#### Circuit Breaker Thresholds

| Setting | Value | Location |
|---------|-------|----------|
| Max Failures | 3 | `internal/crawlers/engine_crawler.go:87` |
| Reset Timeout | 5 minutes | `internal/crawlers/engine_crawler.go:87` |
| State Check | Per-request | `internal/crawlers/engine_crawler.go:208` |

The circuit breaker uses `gobreaker` library with these settings:
```go
gobreaker.Settings{
    MaxRequests: 3,              // Max requests in half-open state
    Interval:    5 * time.Minute, // Statistical interval
    Timeout:     5 * time.Minute, // Time to try again
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures >= 3
    },
}
```

#### Recovery Steps

1. **Wait for automatic reset** (5 minutes):
   ```bash
   # Circuit breaker automatically transitions to half-open state
   # after 5 minutes, allowing test requests
   ```

2. **Restart crawler** (resets circuit state):
   ```bash
   # Press 'q' to gracefully stop
   # Then restart with same command
   ```

3. **Investigate root cause**:
   ```bash
   # Check if domain is accessible
   curl -I https://example.com

   # Check DNS resolution
   nslookup example.com

   # Test with Playwright engine (bypasses some HTTP issues)
   ./crawler --engine playwright --max-pages 1 https://example.com
   ```

4. **Adjust rate limiting** (if 429 errors):
   ```bash
   # Create config with per-domain delay
   cat > crawl.yml <<EOF
   domainDelays:
     example.com: 2000  # 2 second delay
   EOF

   ./crawler --config crawl.yml https://example.com
   ```

5. **Verify with verbose logging**:
   ```bash
   # See detailed error messages
   ./crawler --verbose --max-pages 1 https://example.com 2>&1 | grep -E "ERROR|WARN"
   ```

---

### 2. Engine Initialization Errors

#### Colly Engine Errors

**Symptom:**
```
ERROR: Engine error for https://example.com: invalid URL
ERROR: Invalid URL %s: %v
```

**Causes:**
- Malformed URL syntax
- Unsupported URL schemes (only `http://` and `https://` allowed)
- Invalid URL encoding

**Recovery:**
```bash
# Validate URL syntax
curl -I "https://example.com/path"

# Check for special characters needing encoding
# Use verbose mode to see exact URL being processed
./crawler --verbose https://example.com 2>&1 | grep "Invalid URL"
```

#### Playwright Engine Errors

**Symptom:**
```
ERROR: could not start playwright: playwright error: executable doesn't exist
ERROR: failed to launch browser: could not start playwright
```

**Causes:**
- Playwright browsers not installed
- Missing Chromium dependencies
- Incompatible system architecture

**Recovery:**
```bash
# Install Playwright browsers
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# Install specific browser only
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium

# Verify installation
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --help
```

---

### 3. Navigation and Timeout Errors

#### Symptom
```
ERROR: Engine error for https://example.com: context deadline exceeded
ERROR: failed to navigate to https://example.com: Timeout 30000ms exceeded
```

#### Causes
- **Slow page load** - Large assets, poor server performance
- **JavaScript rendering** - SPA takes time to hydrate
- **Network issues** - High latency, packet loss
- **Infinite redirects** - Misconfigured server redirects

#### Timeout Configuration

| Engine | Default Timeout | Configuration |
|--------|-----------------|---------------|
| Colly | 30 seconds | `internal/crawlers/colly_engine.go:60` |
| Playwright | 30 seconds | `internal/crawlers/playwright_engine.go:85` |

#### Recovery Steps

1. **Use verbose mode** to identify slow pages:
   ```bash
   ./crawler --verbose --max-pages 5 https://example.com 2>&1 | grep "Crawled:"
   ```

2. **Increase timeout for JavaScript sites**:
   ```bash
   # Use Playwright with extra wait time
   cat > crawl.yml <<EOF
   engine: playwright
   extraWaitTime: 5000  # Additional 5 seconds
   waitStrategy: "networkidle"
   EOF

   ./crawler --config crawl.yml https://spa-example.com
   ```

3. **Switch to faster engine** for static content:
   ```bash
   # Colly is 2-5x faster for static sites
   ./crawler --engine colly https://static-site.com
   ```

4. **Add domain-specific delays**:
   ```bash
   # Slow down requests to problematic domain
   cat > crawl.yml <<EOF
   domainDelays:
     slow-example.com: 3000  # 3 seconds between requests
   EOF
   ```

---

### 4. Rate Limiting Errors

#### Symptom
```
WARN: Domain example.com is circuit broken, skipping...
ERROR: Engine error: HTTP 429 Too Many Requests
```

#### Causes
- Sending requests too quickly
- Server-side rate limiting thresholds
- Concurrent requests overwhelming server

#### Rate Limiting Mechanisms

The crawler implements **two-layer rate limiting**:

1. **Per-domain delays** (configurable):
   ```go
   // engine_crawler.go:374-382
   func (c *EngineCrawler) applyRateLimit(domain string) {
       delay := c.config.DefaultDelay
       if domainDelay, exists := c.config.DomainDelays[domain]; exists {
           delay = domainDelay
       }
       if delay > 0 {
           time.Sleep(delay)
       }
   }
   ```

2. **Engine-specific limits**:
   - **Colly**: Built-in `LimitRule` with domain-specific parallelism (colly_engine.go:52-57)
   - **Playwright**: Semaphore-controlled concurrency (engine_crawler.go:83)

#### Recovery Steps

1. **Increase default delay**:
   ```bash
   cat > crawl.yml <<EOF
   defaultDelay: 2000  # 2 seconds between requests
   EOF
   ```

2. **Add per-domain delays**:
   ```bash
   cat > crawl.yml <<EOF
   domainDelays:
     api.example.com: 5000     # Slower for API endpoints
     www.example.com: 1000     # Normal for web pages
   EOF
   ```

3. **Reduce concurrency**:
   ```bash
   ./crawler --concurrency 2 https://example.com
   ```

4. **Use exponential backoff** (via circuit breaker):
   ```bash
   # Circuit breaker automatically implements backoff
   # after 3 consecutive failures, waits 5 minutes
   ```

---

### 5. URL Filtering Errors

#### Symptom
```
ERROR: Invalid URL %s: parse "http://": empty host
UpdateWorker: "already visited" for valid URLs
```

#### Causes
- Invalid URL syntax (missing host, malformed scheme)
- URL filtering too aggressive
- Duplicate URL detection false positives

#### URL Filtering Rules

The crawler enforces **strict URL filtering** (`engine_crawler.go:338-371`):

1. **Scheme validation** - Only `http://` and `https://` allowed
2. **Domain restriction** - Must match start URL's host
3. **Path restriction** - Must stay within base path
4. **Exclude patterns** - Configurable regex exclusions
5. **Normalization** - Removes query parameters like `?C=N;O=A`

#### Recovery Steps

1. **Verify URL meets filtering criteria**:
   ```bash
   # Check domain matches
   echo "https://example.com/path" | awk -F/ '{print $3}'

   # Check path is within base
   ./crawler --verbose https://example.com/base/path 2>&1 | grep -i "skip"
   ```

2. **Adjust exclude patterns**:
   ```bash
   # Reduce aggressive filtering
   cat > crawl.yml <<EOF
   excludePatterns: []
   EOF
   ```

3. **Check normalization**:
   ```bash
   # Apache directory links are auto-normalized
   # ?C=N;O=A parameters are stripped
   grep -r "NormalizeURL" internal/utils/
   ```

---

### 6. Filesystem Errors

#### Symptom
```
ERROR: failed to create output directory: permission denied
ERROR: Failed to save https://example.com: no space left on device
```

#### Causes
- Insufficient permissions on `~/.config/crawler/storage`
- Disk full
- Readonly filesystem
- Path length exceeds system limits

#### Default Storage Location

| Platform | Default Path | Override |
|----------|--------------|----------|
| macOS/Linux | `~/.config/crawler/storage/` | `--output` flag |
| Environment Variable | `CRAWLER_OUTPUT_DIR` | `CRAWLER_OUTPUT_DIR=/tmp/crawl` |

#### Recovery Steps

1. **Check permissions**:
   ```bash
   # Fix storage directory permissions
   chmod -R 755 ~/.config/crawler/

   # Or use alternative location
   ./crawler --output /tmp/crawl-output https://example.com
   ```

2. **Check disk space**:
   ```bash
   # Verify available space
   df -h ~/.config/crawler/storage

   # Clean up old crawls
   du -sh ~/.config/crawler/storage/* | sort -h
   rm -rf ~/.config/crawler/storage/old-domain.com/
   ```

3. **Use temporary directory**:
   ```bash
   ./crawler --output /tmp/crawl https://example.com
   ```

4. **Verify path length** (macOS limit: 255 characters):
   ```bash
   # Check generated path length
   ./crawler --verbose --max-pages 1 https://example.com/very/long/path 2>&1 | grep "Saved:"
   ```

---

### 7. Concurrency and Resource Errors

#### Symptom
```
ERROR: too many open files
ERROR: failed to acquire page: page pool exhausted
```

#### Causes
- System file descriptor limit exceeded
- Concurrency too high for system resources
- Memory exhaustion from concurrent browser instances

#### Concurrency Limits

| Component | Default Limit | Configuration |
|-----------|---------------|---------------|
| Worker goroutines | `config.Concurrency` | `--concurrency` flag |
| Semaphore permits | Same as concurrency | `engine_crawler.go:83` |
| Playwright page pool | Same as concurrency | `pagepool.go` |

#### Recovery Steps

1. **Check system limits**:
   ```bash
   # Check file descriptor limit
   ulimit -n

   # Increase limit (temporary)
   ulimit -n 4096

   # For permanent fix, add to ~/.bashrc or ~/.zshrc:
   # ulimit -n 4096
   ```

2. **Reduce concurrency**:
   ```bash
   # Lower worker count
   ./crawler --concurrency 2 https://example.com
   ```

3. **Use Colly engine** (lower memory footprint):
   ```bash
   # Colly uses ~50MB per worker vs ~200MB for Playwright
   ./crawler --engine colly --concurrency 10 https://example.com
   ```

4. **Monitor memory usage**:
   ```bash
   # Watch memory in real-time
   watch -n 1 'ps aux | grep -E "crawler|chromium" | grep -v grep'

   # Use verbose mode for progress
   ./crawler --verbose https://example.com
   ```

---

### 8. Content Extraction Errors

#### Symptom
```
ERROR: failed to extract links: execution context was destroyed
ERROR: Engine error: no HTML content found
```

#### Causes
- Page redirects to non-HTML content (PDF, binary)
- JavaScript crashes or unhandled exceptions
- Page navigation during extraction
- CORS or CSP restrictions

#### Recovery Steps

1. **Switch wait strategy**:
   ```bash
   cat > crawl.yml <<EOF
   engine: playwright
   waitStrategy: "networkidle"  # Wait for network to settle
   extraWaitTime: 2000          # Additional 2 seconds
   EOF
   ```

2. **Check content type**:
   ```bash
   # Verify page returns HTML
   curl -I https://example.com | grep "Content-Type"

   # Skip non-HTML URLs with exclude patterns
   cat > crawl.yml <<EOF
   excludePatterns:
     - '\\.pdf$'
     - '\\.jpg$'
     - '\\.png$'
   EOF
   ```

3. **Use Colly for problematic pages**:
   ```bash
   # Colly handles redirects better
   ./crawler --engine colly --max-pages 1 https://problematic-site.com
   ```

---

## Debugging Techniques

### Enable Verbose Logging

```bash
# See all log messages (bypasses UI)
./crawler --verbose https://example.com

# Filter for errors only
./crawler --verbose https://example.com 2>&1 | grep -E "ERROR|WARN"

# Monitor circuit breaker state
./crawler --verbose https://example.com 2>&1 | grep "circuit"
```

### Test Single Page

```bash
# Test one page to isolate issues
./crawler --max-pages 1 --verbose https://example.com/specific-page

# Compare engines
./crawler --engine colly --max-pages 1 https://example.com
./crawler --engine playwright --max-pages 1 https://example.com
```

### Check Circuit Breaker State

```bash
# Monitor when domains get blocked
./crawler --verbose https://example.com 2>&1 | grep "Domain .* is circuit broken"

# Verify automatic reset (wait 5 minutes)
./crawler --max-pages 1 https://example.com
# Wait 5 minutes
./crawler --max-pages 1 https://example.com
```

### Inspect Generated Files

```bash
# Verify files are being saved correctly
./crawler --max-pages 5 https://example.com
find ~/.config/crawler/storage -name "*.html" -mtime -1

# Check file content for issues
find ~/.config/crawler/storage -name "*.html" -exec sh -c 'echo "File: $1"; head -n 5 "$1"' _ {} \;
```

---

## Error Prevention Best Practices

### 1. Use Appropriate Engine

| Site Type | Recommended Engine | Reason |
|-----------|-------------------|--------|
| Static HTML | Colly | 2-5x faster, lower memory |
| JavaScript SPA | Playwright | Renders dynamic content |
| PDF-heavy | Either | Both handle downloads |
| Rate-limited | Colly | Better delay control |

### 2. Configure Rate Limiting

```yaml
# crawl.yml
defaultDelay: 1000  # 1 second between requests
domainDelays:
  api.example.com: 5000   # Slower for APIs
  www.example.com: 1000   # Normal for pages
concurrency: 5            # Limit concurrent workers
```

### 3. Set Realistic Timeouts

```yaml
# For slow sites
extraWaitTime: 5000
waitStrategy: "networkidle"

# For fast sites (Colly default)
# No extra configuration needed
```

### 4. Monitor Progress

```bash
# Use verbose mode for production crawls
./crawler --verbose --output /data/crawl https://example.com > crawl.log 2>&1

# Check log for errors
grep -E "ERROR|WARN" crawl.log | sort | uniq -c
```

### 5. Handle Edge Cases

```yaml
# Exclude problematic paths
excludePatterns:
  - '/admin'
  - '/api/'
  - '\\.pdf$'
  - '\\.(jpg|png|gif)$'

# Limit crawl scope
maxDepth: 3
maxPages: 1000
```

---

## Error Recovery Flowchart

```
Error Occurs
    |
    v
Is it a circuit breaker error?
    |
    +-- Yes --> Wait 5 minutes OR Restart crawler
    |            Check domain accessibility with curl
    |            Adjust rate limiting if needed
    |
    +-- No --> Is it a timeout error?
                 |
                 +-- Yes --> Increase extraWaitTime
                 |           Switch to Playwright engine
                 |           Reduce concurrency
                 |
                 +-- No --> Is it a validation error?
                              |
                              +-- Yes --> Verify URL syntax
                              |           Check filtering rules
                              |           Use verbose mode to inspect
                              |
                              +-- No --> Is it a filesystem error?
                                           |
                                           +-- Yes --> Check permissions
                                           |           Verify disk space
                                           |           Use --output flag
                                           |
                                           +-- No --> Check verbose logs
                                                     Verify engine installation
                                                     Review exclude patterns
```

---

## Getting Help

If errors persist after following this guide:

1. **Enable verbose logging** and capture output:
   ```bash
   ./crawler --verbose https://problematic-site.com > debug.log 2>&1
   ```

2. **Check existing issues**:
   ```bash
   # Review docs/errors.md for known issues
   cat docs/errors.md
   ```

3. **Verify engine installation**:
   ```bash
   # Test Playwright installation
   go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --help

   # Test Colly (should work if Go is installed)
   go run -C . --version
   ```

4. **Review configuration**:
   ```bash
   # Dump config to verify settings
   ./crawler --help  # See all flags and defaults
   ```

---

## References

- **Circuit Breaker Implementation**: `internal/crawlers/crawler.go:48-104`
- **Error Handling**: `internal/crawlers/engine_crawler.go:217-249`
- **Colly Engine**: `internal/crawlers/colly_engine.go`
- **Playwright Engine**: `internal/crawlers/playwright_engine.go`
- **URL Filtering**: `internal/crawlers/engine_crawler.go:338-371`
- **Known Issues**: `docs/errors.md`
