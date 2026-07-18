# Gotchas and Non-Obvious Behaviors

## URL Path Restrictions Are Strict

**The Gotcha**: When you crawl `https://site.com/docs/api.html`, the crawler will ONLY crawl within `/docs/`, not the entire site.

**Why It's Surprising**: You might expect it to crawl the whole domain, but it treats the starting URL's path as a boundary.

**Example**:
```bash
# Starting from a file
./crawler https://example.com/blog/post.html
# ✅ Will crawl: /blog/other-post.html, /blog/archive/
# ❌ Won't crawl: /about.html, /contact.html

# Starting from a directory  
./crawler https://example.com/blog/
# ✅ Will crawl: Everything under /blog/
# ❌ Won't crawl: Parent directories
```

**Workaround**: To crawl entire domain, start from root:
```bash
./crawler https://example.com/
```

## Apache Directory Listings Create Duplicates

**The Gotcha**: Apache directory listings have sort links (`?C=N;O=D`) that look like different pages but aren't.

**Why It's Surprising**: You'll see the same directory crawled multiple times with different query parameters.

**What Happens**:
```
Crawling /files/
Crawling /files/?C=N;O=D  (same content, sorted by name desc)
Crawling /files/?C=M;O=A  (same content, sorted by modified asc)
```

**The Fix**: Already implemented - crawler filters these, but you'll still see them in logs.

## Colly Engine Doesn't Execute JavaScript

**The Gotcha**: The default Colly engine is fast but can't see JavaScript-rendered content.

**Why It's Surprising**: Modern sites often load content dynamically, so you get empty pages.

**Symptoms**:
- Pages save but are mostly empty
- Missing content that you see in browser
- Zero links found on JavaScript-heavy sites

**Solution**:
```bash
# Force Playwright engine for JS sites
./crawler --engine playwright https://react-app.com

# Or set in config for specific scenarios
# Mobile mode auto-selects Playwright
./crawler --mobile https://example.com
```

## File Save Location Isn't Where You Run From

**The Gotcha**: Files save to `~/.config/crawler/storage/`, not `./output/` or current directory.

**Why It's Surprising**: Many tools save relative to where you run them.

**To Find Your Files**:
```bash
# macOS/Linux
cd ~/.config/crawler/storage/

# Or use custom location
./crawler --output ./my-crawl https://example.com
```

## Quit Key ('q') Needs Graceful Shutdown Time

**The Gotcha**: Pressing 'q' starts shutdown but may take 2-3 seconds to clean up.

**Why It's Surprising**: Feels like it's hanging when it's actually cleaning up properly.

**What's Happening**:
1. UI exits immediately
2. Workers finish current pages
3. Files are saved
4. Resources cleaned up

**Note**: This is improved in latest version with 2-second timeout.

## Max Pages Includes Failed Pages

**The Gotcha**: `--max-pages 10` might only successfully crawl 7 pages if 3 fail.

**Why It's Surprising**: You expect 10 successful pages, not 10 attempts.

**Example**:
```bash
./crawler --max-pages 5 https://flaky-site.com
# Might get: 3 successful, 2 failed = 5 total (stops)
```

**Workaround**: Set higher limit if expecting failures.

## Relative Import Paths Break Go Modules

**The Gotcha**: Import paths must use full module name, not relative paths.

**Wrong**:
```go
import "../config"  // Won't work
```

**Right**:
```go
import "crawler/internal/config"  // Full module path
```

**Why It's Surprising**: Other languages allow relative imports.

## Cache Persists Between Runs

**The Gotcha**: Crawler remembers visited URLs even after restart.

**Why It's Surprising**: You might expect fresh crawl each run.

**Impact**:
```bash
./crawler https://example.com  # Crawls 50 pages
./crawler https://example.com  # Crawls 0 pages (all cached)
```

**To Force Recrawl**:
```bash
# Use force flag
./crawler --force https://example.com

# Or clear cache
rm -rf ~/Library/Caches/crawler/  # macOS
```

## Binary Name Conflicts with Shell Alias

**The Gotcha**: If you have an alias/function named 'crawler', it overrides the binary.

**Symptoms**:
```bash
./crawler: command not found
# But file exists and is executable
```

**Fix**:
```bash
# Check for conflicts
type crawler
alias | grep crawler

# Use full path
./crawler https://example.com

# Or unalias
unalias crawler
```

## Config File Search Is Silent

**The Gotcha**: Crawler silently ignores missing config files.

**Why It's Surprising**: No error if crawl.yml doesn't exist.

**What Happens**:
1. Looks for `./crawl.yml`
2. Looks for `~/.config/crawler/crawl.yml`
3. Uses defaults if neither found

**To Debug**:
```bash
# See what config is being used
./crawler --verbose https://example.com 2>&1 | head -20
```

## URL Normalization Can Skip Intended Pages

**The Gotcha**: URLs are normalized, which might skip variations you want.

**Example**:
```
https://example.com/page == https://example.com/page/ (trailing slash)
https://example.com == https://example.com/index.html
```

**Impact**: Might miss content if server treats these differently.

<!-- Last updated: 2024-07-14 -->
<!-- Source: Implementation quirks and user confusion points -->