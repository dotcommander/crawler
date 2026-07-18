# URL Filtering Guide

Comprehensive guide to controlling crawl scope using URL filtering and pattern matching.

## Overview

The crawler implements **multi-layer URL filtering** to control crawl scope and prevent runaway crawling:

- **Domain Restriction** - Only crawls pages on the same domain as the start URL
- **Path Restriction** - Stays within the base path of the start URL
- **Scheme Validation** - Only HTTP/HTTPS protocols allowed
- **Exclude Patterns** - Configurable pattern matching for filtering
- **Normalization** - Removes duplicate URLs from query parameter variations

---

## Filtering Rules

### 1. Domain Restriction

The crawler **only follows links on the same domain** as the start URL.

#### Implementation

Location: `internal/crawlers/engine_crawler.go:356-359`

```go
// Skip external domains (different host than start URL)
if u.Host != "" && u.Host != startURL.Host {
    return true
}
```

#### Examples

```bash
# Starting from https://example.com
./crawler https://example.com/docs/

# ✅ Crawled: https://example.com/docs/api.html
# ✅ Crawled: https://example.com/about.html
# ✅ Crawled: https://example.com/products/item
# ❌ Skipped: https://other-site.com/page.html
# ❌ Skipped: https://api.example.com/endpoint
# ❌ Skipped: https://subdomain.example.com/page
```

**Note**: Subdomains are considered different domains. `www.example.com` ≠ `example.com`.

---

### 2. Path Restriction

The crawler **stays within the base path** of the start URL.

#### Implementation

Location: `internal/utils/url.go:141-164`

```go
// IsWithinBasePath checks if URL is within the start URL's base path
func IsWithinBasePath(u, startURL *url.URL) bool {
    // Start URL's base path includes all subdirectories
    // but excludes parent and sibling directories
    basePath := path.Dir(startURL.Path)
    if basePath == "." {
        basePath = "/"
    }
    return strings.HasPrefix(u.Path, basePath)
}
```

#### Examples

```bash
# Starting from specific documentation section
./crawler https://example.com/docs/guides/tutorial.html

# Base path: /docs/
# ✅ Crawled: https://example.com/docs/api.html
# ✅ Crawled: https://example.com/docs/guides/advanced.html
# ✅ Crawled: https://example.com/docs/reference/
# ❌ Skipped: https://example.com/blog/post
# ❌ Skipped: https://example.com/about.html
# ❌ Skipped: https://example.com/ (root is outside /docs/)
```

#### Edge Cases

| Start URL | Base Path | Crawled | Skipped |
|-----------|-----------|---------|---------|
| `https://example.com/` | `/` | All paths on domain | N/A (root level) |
| `https://example.com/docs/` | `/docs` | `/docs/*`, `/docs/api/*` | `/blog/*`, `/about` |
| `https://example.com/docs/api.html` | `/docs` | `/docs/*` | `/blog/*`, `/` |
| `https://example.com/a/b/c.html` | `/a/b` | `/a/b/*`, `/a/b/c/*` | `/a/*`, `/` |

---

### 3. Scheme Validation

The crawler **only accepts HTTP and HTTPS** URLs. All other schemes are rejected.

#### Implementation

Location: `internal/utils/url.go:242-249`

```go
// ShouldSkipURL checks if a URL should be skipped based on its scheme
func ShouldSkipURL(u *url.URL) bool {
    // Skip non-HTTP(S) schemes (mailto:, tel:, ftp:, javascript:, etc.)
    if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
        return true
    }
    return false
}
```

#### Automatically Filtered Schemes

| Scheme | Example | Reason |
|--------|---------|---------|
| `mailto:` | `mailto:user@example.com` | Email addresses |
| `tel:` | `tel:+1234567890` | Phone numbers |
| `javascript:` | `javascript:void(0)` | Script links |
| `ftp://` | `ftp://files.example.com` | FTP protocol |
| `file://` | `file:///local/path` | Local filesystem |
| `data:` | `data:text/html,<html>...` | Embedded data |
| `ws://`, `wss://` | `ws://example.com/socket` | WebSocket connections |

These are **automatically skipped** without configuration.

---

### 4. Exclude Patterns

Configure **custom pattern matching** to filter specific URLs.

#### Implementation

Location: `internal/utils/url.go:251-264`

```go
// MatchesExcludePatterns checks if a URL matches any exclude patterns
func MatchesExcludePatterns(urlStr string, patterns []string) bool {
    for _, pattern := range patterns {
        // Try both path.Match (glob patterns) and string contains
        if matched, _ := path.Match(pattern, urlStr); matched {
            return true
        }
        // Also check if pattern is contained in URL (for simpler patterns)
        if strings.Contains(urlStr, pattern) {
            return true
        }
    }
    return false
}
```

#### Pattern Matching Behavior

The crawler supports **two pattern types**:

1. **Glob Patterns** (via `path.Match`):
   - `*.pdf` - Matches any URL ending with `.pdf`
   - `*/admin/*` - Matches URLs with `/admin/` in the path
   - `???.jpg` - Matches 3-character filenames ending with `.jpg`

2. **Substring Matching** (via `strings.Contains`):
   - `admin` - Matches any URL containing "admin"
   - `logout` - Matches any URL containing "logout"
   - `/api/v1/` - Matches any URL containing this substring

#### Configuration

**Via YAML** (`crawl.yml`):

```yaml
# Using ignorePatterns (older name, maps to ExcludePatterns)
ignorePatterns:
  - "*.pdf"
  - "*/admin/*"
  - "logout"
  - "/api/"

# Or using excludePatterns directly (newer name)
excludePatterns:
  - "*.pdf"
  - "*/admin/*"
  - "logout"
  - "/api/"
```

**Via CLI** (not currently supported - use config file):

```bash
# Create config file first
cat > crawl.yml <<EOF
excludePatterns:
  - "*.pdf"
  - "*/admin/*"
EOF

./crawler --config crawl.yml https://example.com
```

---

## Common Patterns

### File Extensions

Exclude specific file types from crawling. Common file types to exclude: `*.pdf`, `*.zip`, `*.jpg`, `*.png`, `*.mp4`, and other media formats:

```yaml
excludePatterns:
  - "*.pdf"        # PDF documents
  - "*.zip"        # Zip archives
  - "*.tar.gz"     # Compressed archives
  - "*.jpg"        # JPEG images
  - "*.png"        # PNG images
  - "*.gif"        # GIF images
  - "*.svg"        # SVG graphics
  - "*.mp4"        # MP4 videos
  - "*.webm"       # WebM videos
```

**Note**: The crawler **automatically downloads PDFs** detected via content type. Use `*.pdf` pattern to prevent downloading.

### Administrative Areas

Exclude admin and backend areas. Common patterns: `*/admin/*`, `*/backend/*`, `*/wp-admin/*`, `*/user/*`, and similar control interfaces:

```yaml
excludePatterns:
  - "*/admin/*"        # Admin panels
  - "*/backend/*"      # Backend interfaces
  - "*/wp-admin/*"     # WordPress admin
  - "*/administrator/*" # Joomla admin
  - "*/manager/*"      # MODX manager
  - "*/user/*"         # User account areas
```

### API Endpoints

Exclude API endpoints to avoid crawling JSON responses. Key patterns: `/api/`, `/api/v1/*`, `/graphql`, `*.json`, and REST endpoint paths:

```yaml
excludePatterns:
  - "/api/"            # All API endpoints
  - "/api/v1/*"        # API v1 specifically
  - "/graphql"         # GraphQL endpoints
  - "/rest/"           # REST APIs
  - "*.json"           # JSON files
```

### Authentication and Sessions

Exclude authentication-related URLs. Common patterns: `login`, `logout`, `register`, `password`, `/auth/*`, and session management paths:

```yaml
excludePatterns:
  - "login"            # Login pages
  - "logout"           # Logout actions
  - "register"         # Registration pages
  - "password"         # Password reset flows
  - "/auth/*"          # Auth endpoints
  - "/session/*"       # Session management
  - "/oauth/*"         # OAuth flows
```

### Search and Filter URLs

Exclude search results and filtered views. Common patterns: `/search*`, `?query=`, `?filter=`, `?sort=`, and pagination parameters:

```yaml
excludePatterns:
  - "/search*"         # Search pages
  - "?query="          # Search query parameters
  - "?filter="         # Filter parameters
  - "?sort="           # Sort parameters
  - "?page="           # Pagination (be careful!)
```

**Warning**: Excluding `?page=` may prevent crawling paginated content. Use with caution.

### Development and Testing

Exclude development/staging environments. Common patterns: `/dev/*`, `/staging/*`, `/test/*`, `/debug/*`, and similar non-production paths:

```yaml
excludePatterns:
  - "/dev/*"           # Development areas
  - "/staging/*"       # Staging environments
  - "/test/*"          # Test directories
  - "/spec/*"          # Test specifications
  - "/debug/*"         # Debug endpoints
```

---

## URL Normalization

The crawler **normalizes URLs** to avoid duplicate crawling from parameter variations.

### Apache Directory Listing Parameters

Removes Apache sort parameters like `C=N;O=A` and `C=M;O=D` from URLs to prevent duplicate crawling of the same directory listing.

Location: `internal/utils/url.go:266-284`

```go
// NormalizeURLString removes Apache directory listing sort parameters
func NormalizeURLString(urlStr string) string {
    // Remove ?C=N;O=A (sort by name, ascending)
    // Remove ?C=M;O=D (sort by modified date, descending)
    // etc.
    u, err := url.Parse(urlStr)
    if err != nil {
        return urlStr
    }

    // Remove Apache sort parameters from query
    q := u.Query()
    delete(q, "C")  // Sort column
    delete(q, "O")  // Sort order

    u.RawQuery = q.Encode()
    return u.String()
}
```

#### Normalization Examples

| Original URL | Normalized URL | Reason |
|--------------|----------------|--------|
| `https://example.com/docs/?C=N;O=A` | `https://example.com/docs/` | Apache sort by name |
| `https://example.com/docs/?C=M;O=D` | `https://example.com/docs/` | Apache sort by date |
| `https://example.com/page?id=123` | `https://example.com/page?id=123` | Preserves non-Apache params |

---

## Configuration Examples

### Minimal Config (No Exclusions)

```yaml
# crawl.yml
concurrency: 5
depth: 2
excludePatterns: []
```

### Blog Archive Crawl

Crawl blog archives but skip media files:

```yaml
# crawl.yml
concurrency: 3
depth: 4
excludePatterns:
  - "*.jpg"
  - "*.png"
  - "*.gif"
  - "*.mp4"
  - "*.webm"
```

### Documentation Crawl

Crawl documentation while excluding API endpoints and admin areas:

```yaml
# crawl.yml
concurrency: 5
depth: 5
excludePatterns:
  - "/api/"
  - "/admin/*"
  - "login"
  - "logout"
  - "*.pdf"
```

### E-commerce Product Catalog

Crawl product pages, skip checkout and account areas:

```yaml
# crawl.yml
concurrency: 10
depth: 3
excludePatterns:
  - "/cart/*"
  - "/checkout/*"
  - "/account/*"
  - "login"
  - "register"
  - "/wishlist/*"
```

---

## Testing URL Filtering

### Verbose Mode

Use `--verbose` to see which URLs are being filtered. Pipe through `grep -E "skip|Skip"` to focus on filtered URLs:

```bash
./crawler --verbose https://example.com 2>&1 | grep -E "skip|Skip"
```

### Dry Run with Max Pages

Test filtering with a limited crawl using `--max-pages 5` to cap the crawl at 5 pages:

```bash
# Crawl only first 5 pages to verify filtering
./crawler --max-pages 5 --verbose https://example.com
```

### Single Page Test

Test a specific URL's filtering behavior using `--max-pages 1` to fetch only the start page:

```bash
# Test if a specific URL would be crawled
./crawler --max-pages 1 --verbose https://example.com/admin/test
```

---

## Filtering Flowchart

```
Link Discovered
    |
    v
Parse URL
    |
    +-- Invalid syntax? → Skip (ERROR log)
    |
    v
Check Scheme
    |
    +-- Not HTTP/HTTPS? → Skip (mailto:, tel:, etc.)
    |
    v
Check Domain
    |
    +-- Different host? → Skip (external domain)
    |
    v
Check Base Path
    |
    +-- Outside base path? → Skip (different directory)
    |
    v
Check Exclude Patterns
    |
    +-- Matches pattern? → Skip (config exclusion)
    |
    v
Check Normalization
    |
    +-- Already visited? → Skip (duplicate URL)
    |
    v
Add to Queue ✅
```

---

## Best Practices

### 1. Start Broad, Then Refine

Begin with minimal exclusions, then add patterns based on crawl results:

```bash
# Initial crawl - see what gets discovered
./crawler --max-pages 10 --verbose https://example.com > initial.log

# Review log, identify unwanted URLs
grep "Crawled:" initial.log | grep -E "\.pdf|admin|api"

# Add patterns to exclude these
cat > crawl.yml <<EOF
excludePatterns:
  - "*.pdf"
  - "*/admin/*"
  - "/api/"
EOF

# Re-crawl with exclusions
./crawler --config crawl.yml https://example.com
```

### 2. Use Specific Patterns

Prefer specific patterns over broad ones:

```yaml
# ❌ Too broad - may exclude valid content
excludePatterns:
  - "api"

# ✅ Specific - only API endpoints
excludePatterns:
  - "/api/"
```

### 3. Test Patterns Incrementally

Add patterns one at a time and test:

```yaml
# Start with one pattern
excludePatterns:
  - "*.pdf"

# Test crawl
./crawler --max-pages 5 https://example.com

# If working, add next pattern
excludePatterns:
  - "*.pdf"
  - "*/admin/*"
```

### 4. Use Glob Patterns for Paths

Use glob syntax for path matching:

```yaml
# ✅ Good: Glob patterns for paths
excludePatterns:
  - "*/admin/*"
  - "*/api/*"

# ⚠️ Acceptable: Substring matching
excludePatterns:
  - "admin"
  - "api"
```

### 5. Consider Page Type

Different crawls require different patterns:

| Crawl Type | Exclude These |
|------------|---------------|
| **Content Archive** | `*.pdf`, `*.zip`, media files |
| **Link Analysis** | None (keep all for analysis) |
| **API Discovery** | `/api/` (if discovering pages, not endpoints) |
| **Product Catalog** | `/cart/*`, `/checkout/*`, `/account/*` |

---

## Troubleshooting

### URLs Not Being Excluded

**Problem**: URLs matching exclude patterns are still being crawled.

**Solution**: Check pattern syntax:

```bash
# Test pattern matching manually
cat > test_pattern.go <<'EOF'
package main
import (
    "fmt"
    "path"
    "strings"
)
func main() {
    url := "https://example.com/admin/settings"
    pattern := "*/admin/*"
    matched, _ := path.Match(pattern, url)
    fmt.Printf("Glob match: %v\n", matched)
    fmt.Printf("Contains match: %v\n", strings.Contains(url, "admin"))
}
EOF

go run test_pattern.go
```

### Too Many URLs Skipped

**Problem**: Valid URLs are being excluded.

**Solution**: Review patterns for over-matching:

```yaml
# ❌ Problem: Matches "admin-settings.html"
excludePatterns:
  - "admin"

# ✅ Better: More specific
excludePatterns:
  - "*/admin/*"
```

### URLs Outside Base Path

**Problem**: URLs that should be crawled are being skipped as outside base path.

**Solution**: Verify base path calculation:

```bash
# Check what the base path is
echo "https://example.com/docs/api.html" | awk -F/ '{print "/" $2 "/" $3}'

# Result: /docs
# Any URL starting with /docs/ will be crawled
# URLs like /blog/ or /about will be skipped
```

---

## References

- **URL Filtering Implementation**: `internal/crawlers/engine_crawler.go:338-371`
- **Path Validation**: `internal/utils/url.go:141-164`
- **Pattern Matching**: `internal/utils/url.go:251-264`
- **Normalization**: `internal/utils/url.go:266-284`
- **Scheme Validation**: `internal/utils/url.go:242-249`
- **Configuration**: `internal/config/viper_config.go:153`
