// Package katanaengine adapts crawler's configuration, reporting, and
// session surfaces onto the projectdiscovery/katana engine used as a
// library. It replaces the per-page Colly/Rod/Playwright engines and the
// former in-house orchestration with katana's own queueing, scope
// control, and pacing.
package katanaengine

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	urlutils "github.com/dotcommander/crawler/internal/utils"
	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/katana/pkg/types"
	"github.com/projectdiscovery/katana/pkg/utils/queue"
)

// TranslateOptions maps CrawlerConfig and seed URLs onto katana options.
// Returned options are pre-populated from katana's DefaultOptions so that
// engine timeouts and body limits stay upstream-consistent.
func TranslateOptions(cfg *config.CrawlerConfig, seeds []string) (*types.Options, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("katanaengine: at least one seed URL is required")
	}
	// Root seeds arrive as bare origins; the same page discovered via
	// links resolves to the trailing-slash form, and katana's enqueue
	// dedup is string-based — seeding the canonical with-slash form
	// keeps the root from being fetched (and depth-charged) twice.
	canonical := canonicalSeeds(seeds)
	scopeRegexes, err := ScopeRegexes(canonical)
	if err != nil {
		return nil, fmt.Errorf("katanaengine: build scope: %w", err)
	}
	outOfScope := ExcludeOutOfScope(cfg.ExcludePatterns)

	opts := types.DefaultOptions // copy; defaults stay upstream-consistent
	opts.URLs = goflags.StringSlice(canonical)
	// katana depth counts hops from the seed (seed = 0), matching
	// crawler's MaxDepth semantics; enqueue rejects depth > MaxDepth.
	opts.MaxDepth = cfg.MaxDepth
	if cfg.Concurrency > 0 {
		opts.Concurrency = cfg.Concurrency
		opts.Parallelism = cfg.Concurrency
	}
	hostRate, hostRateMinute := pacingFromDelay(cfg.DefaultDelay)
	opts.HostRateLimit = hostRate
	opts.HostRateLimitMinute = hostRateMinute
	// Legacy behavior read full response bodies into memory (PDFs capped
	// at 50MB); keep byte parity by reading up to the same bound.
	const maxBodyReadSize = 50 * 1024 * 1024
	opts.BodyReadSize = maxBodyReadSize
	// Legacy behavior recorded and re-linked every fetched page; katana's
	// unique-content filter would silently drop byte-identical bodies
	// (including their discovered links), so it is disabled for parity.
	opts.DisableUniqueFilter = true
	opts.Timeout = engineTimeout(cfg)
	opts.Retries = cfg.MaxRetries
	opts.CustomHeaders = customHeaders(cfg)
	opts.HeadlessOptionalArguments = headlessArgs(cfg)
	opts.Strategy = queue.BreadthFirst.String()
	opts.FieldScope = "" // replaced by explicit per-seed Scope regexes
	opts.Scope = goflags.StringSlice(scopeRegexes)
	opts.OutOfScope = goflags.StringSlice(outOfScope)
	// Katana's own robots.txt/sitemap discovery (KnownFiles) stays
	// disabled: its parsers fetch `<seed>/robots.txt` and
	// `<seed>/sitemap.xml` at the seed path instead of the host root
	// (RFC 9309), enqueue every Allow/Disallow directive as a fresh
	// request, and bypass the queue's dedupe under concurrency, causing
	// duplicate fetches. Seeding and robots compliance are owned here:
	// cmd/root.go seeds from robots.txt/sitemaps, and compliance runs
	// through the robots->OutOfScope translation plus the result-level
	// gate.
	opts.KnownFiles = ""
	opts.ScrapeJSResponses = cfg.JSCrawl
	opts.PageLoadStrategy = pageLoadStrategy(cfg.WaitStrategy)
	if cfg.ExtraWaitTime > 0 {
		opts.DOMWaitTime = int(math.Ceil(cfg.ExtraWaitTime.Seconds()))
	}
	opts.Silent = true
	opts.DisableUpdateCheck = true
	return &opts, nil
}

// ScopeRegexes confines crawling to each seed's scheme, host, and base
// path, mirroring crawler's same-domain-and-base-path rule (utils.
// baseDirectory semantics: the longest prefix ending in "/"; root-file
// seeds allow the whole host). Multiple seeds on different origins are
// joined into one anchored alternation. Bare-origin URLs (no path) are
// explicitly allowed so the seed result itself is in scope.
func ScopeRegexes(seeds []string) ([]string, error) {
	prefixes := make([]string, 0, len(seeds))
	seen := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		u, err := url.Parse(seed)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("invalid seed URL %q", seed)
		}
		baseDir := "/"
		if u.Path != "" && u.Path != "/" {
			switch {
			case strings.HasSuffix(u.Path, "/"):
				baseDir = u.Path
			case strings.LastIndex(u.Path, "/") > 0:
				baseDir = u.Path[:strings.LastIndex(u.Path, "/")+1]
			}
		}
		var alt string
		if baseDir == "/" {
			// Whole host, including the bare-origin form of the seed.
			alt = regexp.QuoteMeta(u.Scheme+"://"+u.Host) + "(?:/|$)"
		} else {
			alt = regexp.QuoteMeta(u.Scheme + "://" + u.Host + baseDir)
		}
		if _, dup := seen[alt]; dup {
			continue
		}
		seen[alt] = struct{}{}
		prefixes = append(prefixes, alt)
	}
	return []string{"^(?:" + strings.Join(prefixes, "|") + ")"}, nil
}

// ExcludeOutOfScope converts crawler exclude patterns (path.Match globs or
// plain substrings, see urlutils.MatchesExcludePatterns) into katana
// OutOfScope regexes. Translation is exact for glob matches against the
// URL path and last path segment; bare substring patterns remain
// unanchored and therefore also match the host and query portions of a
// URL. That deviation only ever over-blocks, never under-blocks.
func ExcludeOutOfScope(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if hasGlobMeta(p) {
			g := globToRegex(p)
			fullPath := `://[^/?#]*` + g + `(?:[?#]|$)`
			lastSegment := `://[^?#]*/` + g + `(?:[?#]|$)`
			out = append(out, `(?:`+fullPath+`|`+lastSegment+`|`+regexp.QuoteMeta(p)+`)`)
			continue
		}
		out = append(out, regexp.QuoteMeta(p))
	}
	return out
}

func hasGlobMeta(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// globToRegex translates a path.Match glob into an unanchored regex.
// path.Match's "*" and "?" do not cross "/", which [^/] preserves.
func globToRegex(glob string) string {
	var b strings.Builder
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}

// pacingFromDelay converts the per-host default delay into katana's
// per-host rate limits. Sub-second delays map to requests per second;
// delays of a second or more map to requests per minute, which keeps
// fractional rates such as 0.5/s expressible.
func pacingFromDelay(d time.Duration) (hostRate int, hostRateMinute int) {
	if d <= 0 {
		return 0, 0
	}
	secs := d.Seconds()
	switch {
	case secs < 1:
		return int(math.Ceil(1 / secs)), 0
	default:
		return 0, int(math.Round(60 / secs))
	}
}

// pageLoadStrategy maps crawler wait strategies onto katana's
// PageLoadStrategy vocabulary. "commit" has no katana equivalent and maps
// to its nearest successor event, domcontentloaded.
func pageLoadStrategy(waitStrategy string) string {
	switch strings.ToLower(waitStrategy) {
	case "commit":
		return "domcontentloaded"
	case "load", "domcontentloaded", "networkidle":
		return strings.ToLower(waitStrategy)
	default:
		return ""
	}
}

// engineTimeout returns the katana engine window in seconds. The
// window floors a crawl's runtime (natural end is detected after the
// queue has been empty for the remainder of the window), so it must
// stay comfortably above the slowest expected page fetch to avoid
// dropping links discovered by in-flight workers near the window edge.
func engineTimeout(cfg *config.CrawlerConfig) int {
	if cfg.EngineTimeoutSeconds > 0 {
		return cfg.EngineTimeoutSeconds
	}
	return 10 // upstream katana default
}

// MobileViewportWidth/Height mirror the viewport the legacy Rod engine
// emulated for --mobile. Headless Chrome args approximate it (user agent
// + window size); full touch/viewport emulation needs CDP, which katana
// does not expose — documented deviation in the migration spec (D3).
const (
	MobileViewportWidth  = 390
	MobileViewportHeight = 844
)

func headlessArgs(cfg *config.CrawlerConfig) goflags.StringSlice {
	if !cfg.Mobile {
		return nil
	}
	return goflags.StringSlice{
		fmt.Sprintf("--window-size=%d,%d", MobileViewportWidth, MobileViewportHeight),
		"--lang=en-US",
	}
}

func customHeaders(cfg *config.CrawlerConfig) goflags.StringSlice {
	headers := make(goflags.StringSlice, 0, len(cfg.Headers)+1)
	userAgent := cfg.UserAgent
	if cfg.Mobile && cfg.MobileUserAgent != "" {
		userAgent = cfg.MobileUserAgent
	}
	if userAgent != "" {
		headers = append(headers, "User-Agent: "+userAgent)
	}
	for k, v := range cfg.Headers {
		headers = append(headers, k+": "+v)
	}
	return headers
}

// normalizeURL re-exports the URL normalization used for visited-store
// keys so results and legacy sessions share one key space.
func normalizeURL(raw string) string {
	normalized := urlutils.NormalizeURLString(raw)
	// A site root arrives both as the bare origin (seed form) and with a
	// trailing slash (discovered-link form); canonicalize to the
	// with-slash form so both collapse to one visited-store key.
	// Sub-directory trailing slashes are already stripped by
	// NormalizeURLString.
	if u, err := url.Parse(normalized); err == nil && u.Path == "" {
		u.Path = "/"
		return u.String()
	}
	return normalized
}

// canonicalSeeds returns seeds with bare-origin roots normalized to
// their trailing-slash form.
func canonicalSeeds(seeds []string) []string {
	out := make([]string, len(seeds))
	for i, s := range seeds {
		if u, err := url.Parse(s); err == nil && u.Path == "" {
			s = u.String() + "/"
		}
		out[i] = s
	}
	return out
}
