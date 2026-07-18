package crawlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/internal/utils"
	"github.com/dotcommander/crawler/ui"

	"github.com/sony/gobreaker"
	"golang.org/x/sync/semaphore"
)

// EngineCrawler is a new crawler implementation that uses pluggable engines
type EngineCrawler struct {
	config        *config.CrawlerConfig
	engine        CrawlEngine
	reporter      ProgressReporter
	exporter      exporters.Exporter
	semaphore     *semaphore.Weighted
	visited       session.VisitedStore
	queue         chan *QueueItem
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	activeWorkers atomic.Int64
	queueClosed   sync.Once
	seedingDone   chan struct{}

	// Statistics
	stats struct {
		PagesVisited    atomic.Int64
		PagesFailed     atomic.Int64
		PDFsDownloaded  atomic.Int64
		BytesDownloaded atomic.Int64
		LinksDropped    atomic.Int64
	}

	// Circuit breaker for failing domains
	circuitBreaker *DomainCircuitBreaker

	// Per-domain rate limiter (non-blocking across domains; respects ctx)
	rateLimiter *DomainRateLimiter

	// Parsed seed URLs for domain-boundary checking
	seedURLs []*url.URL

	// HTTP client for fetching external JS files
	httpClient *http.Client

	// Tracks fetched external JS URLs to avoid re-fetching
	fetchedJS sync.Map

	// Pre-compiled CSS selectors for --extract
	compiledSelectors []compiledSelector

	// Start time for duration calculation
	startTime time.Time

	// robotsAllowed, when non-nil, is consulted in shouldSkipURL to enforce
	// robots.txt at the discovery boundary. Signature mirrors
	// seeders.RobotsResult.IsAllowed(path, userAgent). Nil = no enforcement.
	robotsAllowed func(path, userAgent string) bool
}

// NewEngineCrawler creates a new engine-based crawler
func NewEngineCrawler(cfg *config.CrawlerConfig, reporter ProgressReporter, engineType string, store session.VisitedStore) (*EngineCrawler, error) {
	if reporter == nil {
		reporter = &NoOpReporter{}
	}

	// Determine engine type
	var engineTypeEnum EngineType
	switch strings.ToLower(engineType) {
	case "colly":
		engineTypeEnum = EngineTypeColly
	case "playwright":
		engineTypeEnum = EngineTypePlaywright
	case "":
		// Auto-select based on configuration
		engineTypeEnum = SelectEngine(cfg)
	default:
		return nil, fmt.Errorf("unknown engine type: %s", engineType)
	}

	// Create the appropriate engine
	engine, err := CreateEngine(cfg, engineTypeEnum)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s engine: %w", engineTypeEnum, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if store == nil || isNilVisitedStore(store) {
		store = session.NewMemoryStore()
	}

	crawler := &EngineCrawler{
		config:         cfg,
		engine:         engine,
		reporter:       reporter,
		semaphore:      semaphore.NewWeighted(int64(cfg.Concurrency)),
		visited:        store,
		queue:          make(chan *QueueItem, cfg.Concurrency*100),
		ctx:            ctx,
		cancel:         cancel,
		seedingDone:    make(chan struct{}),
		circuitBreaker: NewDomainCircuitBreaker(3, 5*time.Minute),
		rateLimiter:    NewDomainRateLimiter(cfg.DefaultDelay),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		startTime:      time.Now(),
	}

	// Register any per-domain delay overrides on the rate limiter
	for domain, delay := range cfg.DomainDelays {
		crawler.rateLimiter.SetDomainDelay(domain, delay)
	}

	// Pre-parse seed URLs for domain-boundary checking
	seedSources := cfg.StartURLs
	if len(seedSources) == 0 {
		seedSources = []string{cfg.StartURL}
	}
	for _, rawURL := range seedSources {
		if parsed, err := url.Parse(rawURL); err == nil {
			crawler.seedURLs = append(crawler.seedURLs, parsed)
		}
	}

	// Pre-compile CSS selectors for --extract
	if len(cfg.ExtractSelectors) > 0 {
		crawler.compiledSelectors = CompileSelectors(cfg.ExtractSelectors)
	}

	// Log which engine is being used
	if !cfg.Quiet {
		log.Printf("Using %s engine for crawling", engine.GetEngineType())
	}

	return crawler, nil
}

func isNilVisitedStore(store session.VisitedStore) bool {
	v := reflect.ValueOf(store)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// SetExporter sets the exporter for structured output
func (c *EngineCrawler) SetExporter(exp exporters.Exporter) {
	c.exporter = exp
}

// SetRobotsChecker installs a robots.txt allow predicate consulted for every
// discovered URL in shouldSkipURL. fn(path, userAgent) reports whether path is
// allowed. Passing nil disables enforcement (default).
func (c *EngineCrawler) SetRobotsChecker(fn func(path, userAgent string) bool) {
	c.robotsAllowed = fn
}

// Start begins the crawling process
func (c *EngineCrawler) Start() error {
	// Ensure output directory exists
	if err := os.MkdirAll(c.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Start worker goroutines before seeding so large seed sets cannot fill the
	// queue and block startup before any consumer exists.
	for i := 0; i < c.config.Concurrency; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// Start progress reporting and idle detection.
	go c.progressReporter()
	go c.cancelWhenIdle()

	// Add all seed URLs to queue.
	seeds := c.config.StartURLs
	if len(seeds) == 0 {
		seeds = []string{c.config.StartURL}
	}
	for _, seedURL := range seeds {
		if _, err := url.Parse(seedURL); err != nil {
			close(c.seedingDone)
			c.cancel()
			return fmt.Errorf("invalid start URL %q: %w", seedURL, err)
		}
		if !c.enqueue(&QueueItem{URL: seedURL, Depth: 0}) {
			close(c.seedingDone)
			return c.ctx.Err()
		}
	}
	if len(seeds) == 1 {
		c.reporter.Log("INFO", fmt.Sprintf("Starting crawl of %s", seeds[0]))
	} else {
		c.reporter.Log("INFO", fmt.Sprintf("Starting crawl of %d seed URLs", len(seeds)))
	}

	// Add pre-seeded URLs from sitemap/robots.txt discovery
	seeded := 0
	for _, seedURL := range c.config.SeedURLs {
		if c.enqueueNonBlocking(&QueueItem{URL: seedURL, Depth: 1}) {
			seeded++
			continue
		}
		c.reporter.Log("WARN", fmt.Sprintf("Queue full, seeded %d of %d sitemap URLs", seeded, len(c.config.SeedURLs)))
		break
	}
	if seeded > 0 {
		c.reporter.Log("INFO", fmt.Sprintf("Seeded %d URLs from sitemap discovery", seeded))
	}
	close(c.seedingDone)

	// Wait for completion or cancellation
	c.wg.Wait()
	c.queueClosed.Do(func() {
		close(c.queue)
	})

	return nil
}

func (c *EngineCrawler) enqueue(item *QueueItem) bool {
	select {
	case <-c.ctx.Done():
		return false
	case c.queue <- item:
		return true
	}
}

func (c *EngineCrawler) enqueueNonBlocking(item *QueueItem) bool {
	select {
	case <-c.ctx.Done():
		return false
	case c.queue <- item:
		return true
	default:
		return false
	}
}

func (c *EngineCrawler) cancelWhenIdle() {
	<-c.seedingDone

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if len(c.queue) == 0 && c.activeWorkers.Load() == 0 {
				c.cancel()
				return
			}
		}
	}
}

// worker processes items from the queue
func (c *EngineCrawler) worker(workerID int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			c.reporter.UpdateWorker(workerID, "stopped", "")
			return
		case item := <-c.queue:
			if item == nil {
				return // Channel closed
			}

			// Count as active the instant we own this dequeued item, BEFORE the
			// MaxPages/semaphore gates. Otherwise cancelWhenIdle can observe a
			// false idle state (empty queue + zero active) in the window between
			// dequeue and processing, cancel the crawler, and lose this item.
			c.activeWorkers.Add(1)

			// Check if we've reached max pages limit
			if c.config.MaxPages > 0 && c.stats.PagesVisited.Load() >= int64(c.config.MaxPages) {
				c.activeWorkers.Add(-1)
				c.reporter.UpdateWorker(workerID, "limit reached", "")
				continue
			}

			// Acquire semaphore
			if err := c.semaphore.Acquire(c.ctx, 1); err != nil {
				c.activeWorkers.Add(-1)
				return // Context cancelled
			}

			c.processItem(workerID, item)
			c.activeWorkers.Add(-1)
			c.semaphore.Release(1)
		}
	}
}

// processItem processes a single crawl item
func (c *EngineCrawler) processItem(workerID int, item *QueueItem) {
	c.reporter.UpdateWorker(workerID, "crawling", item.URL)

	// Validate URL and check if already visited
	parsedURL, normalizedURL, shouldSkip := c.validateAndCheckURL(workerID, item.URL)
	if shouldSkip {
		return
	}

	// Apply rate limiting (full URL — limiter derives the domain)
	c.applyRateLimit(item.URL)

	// Perform the actual crawl
	result := c.performCrawl(item, parsedURL)
	if result == nil {
		return
	}

	// Process successful crawl result
	c.processCrawlResult(item, result, parsedURL, normalizedURL)

	c.reporter.UpdateWorker(workerID, "idle", "")
}

// validateAndCheckURL validates the URL and checks if it should be processed.
// Returns the parsed URL and the normalized URL string for later use.
func (c *EngineCrawler) validateAndCheckURL(workerID int, urlStr string) (*url.URL, string, bool) {
	// Normalize URL to avoid duplicates from directory listing sort parameters
	normalizedURL := c.normalizeURL(urlStr)

	// Check if already visited (using normalized URL)
	if c.visited.MarkVisited(normalizedURL) {
		c.reporter.UpdateWorker(workerID, "already visited", urlStr)
		return nil, "", true
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		c.reporter.Log("ERROR", fmt.Sprintf("Invalid URL %s: %v", urlStr, err))
		c.stats.PagesFailed.Add(1)
		return nil, "", true
	}

	// Check circuit breaker
	if c.circuitBreaker.IsBlocked(parsedURL.Host) {
		c.reporter.Log("WARN", fmt.Sprintf("Domain %s is circuit broken, skipping %s", parsedURL.Host, urlStr))
		c.stats.PagesFailed.Add(1)
		return nil, "", true
	}

	return parsedURL, normalizedURL, false
}

// performCrawl executes the crawl operation using the configured engine with circuit breaker protection
func (c *EngineCrawler) performCrawl(item *QueueItem, parsedURL *url.URL) *CrawlResult {
	var result *CrawlResult

	// Execute crawl operation through circuit breaker
	err := c.circuitBreaker.Execute(parsedURL.Host, func() error {
		var crawlErr error
		result, crawlErr = c.engine.CrawlPage(c.ctx, item)
		if crawlErr != nil {
			c.reporter.Log("ERROR", fmt.Sprintf("Engine error for %s: %v", item.URL, crawlErr))
			c.stats.PagesFailed.Add(1)
			return crawlErr
		}

		if !result.Success {
			c.reporter.Log("ERROR", fmt.Sprintf("Failed to crawl %s: %v", item.URL, result.Error))
			c.stats.PagesFailed.Add(1)
			return fmt.Errorf("crawl failed: %v", result.Error)
		}

		return nil
	})

	if err != nil {
		// Circuit breaker is open or operation failed
		if err == gobreaker.ErrOpenState {
			c.reporter.Log("WARN", fmt.Sprintf("Circuit breaker open for domain %s, skipping %s", parsedURL.Host, item.URL))
		}
		return nil
	}

	return result
}

// processCrawlResult handles a successful crawl result
func (c *EngineCrawler) processCrawlResult(item *QueueItem, result *CrawlResult, parsedURL *url.URL, normalizedURL string) {
	// Update statistics
	c.stats.PagesVisited.Add(1)
	c.stats.BytesDownloaded.Add(result.ContentLength)

	// Record result in visited store (persists status code for resume)
	if err := c.visited.RecordResult(normalizedURL, result.StatusCode); err != nil {
		c.reporter.Log("WARN", fmt.Sprintf("Failed to record result for %s: %v", item.URL, err))
	}

	// Export record if exporter is set
	if c.exporter != nil {
		record := exporters.PageRecord{
			URL:         result.URL,
			Title:       result.Title,
			StatusCode:  result.StatusCode,
			ContentType: result.ContentType,
			LinksFound:  len(result.Links),
			CrawledAt:   time.Now(),
		}
		if len(c.compiledSelectors) > 0 && len(result.Content) > 0 {
			record.Extracted = ExtractFields(result.Content, c.compiledSelectors)
		}
		if err := c.exporter.WriteRecord(record); err != nil {
			c.reporter.Log("ERROR", fmt.Sprintf("Failed to export record for %s: %v", result.URL, err))
		}
	}

	// Save content if present
	c.saveResultContent(result)

	// Extract JS endpoints if enabled
	if c.config.JSCrawl && len(result.Content) > 0 {
		jsEndpoints := ExtractJSEndpoints(result.Content, parsedURL)
		if len(jsEndpoints) > 0 {
			c.reporter.Log("INFO", fmt.Sprintf("JS extracted %d endpoints from %s", len(jsEndpoints), utils.TruncateURL(item.URL, 60)))
			result.Links = append(result.Links, jsEndpoints...)
		}

		// Fetch and parse external JS files referenced by <script src="...">
		externalEndpoints := c.fetchExternalJSEndpoints(result.Content, parsedURL)
		if len(externalEndpoints) > 0 {
			c.reporter.Log("INFO", fmt.Sprintf("External JS extracted %d endpoints from %s", len(externalEndpoints), utils.TruncateURL(item.URL, 60)))
			result.Links = append(result.Links, externalEndpoints...)
		}
	}

	// Log success message
	c.logCrawlSuccess(item, result)

	// Add discovered links to queue if within depth limit
	if item.Depth < c.config.MaxDepth {
		c.addLinksToQueue(result.Links, item.Depth+1, parsedURL)
	}
}

// saveResultContent saves crawled content to file
func (c *EngineCrawler) saveResultContent(result *CrawlResult) {
	if len(result.Content) == 0 {
		return
	}

	filePath := c.getFilePath(result.URL)
	if err := c.saveContent(filePath, result.Content); err != nil {
		c.reporter.Log("ERROR", fmt.Sprintf("Failed to save %s: %v", result.URL, err))
	} else {
		c.reporter.Log("INFO", fmt.Sprintf("Saved: %s", utils.TruncateURL(filePath, 60)))
	}
}

// logCrawlSuccess logs appropriate success message based on content type
func (c *EngineCrawler) logCrawlSuccess(item *QueueItem, result *CrawlResult) {
	if result.IsPDF {
		c.stats.PDFsDownloaded.Add(1)
		c.reporter.Log("SUCCESS", fmt.Sprintf("Downloaded PDF: %s (%s)",
			utils.TruncateURL(item.URL, 60),
			utils.FormatBytes(result.ContentLength, true)))
	} else {
		c.reporter.Log("SUCCESS", fmt.Sprintf("Crawled: %s (%s, %d links)",
			utils.TruncateURL(item.URL, 60),
			utils.FormatBytes(result.ContentLength, true),
			len(result.Links)))
	}
}

// addLinksToQueue adds discovered links to the crawl queue
func (c *EngineCrawler) addLinksToQueue(links []string, depth int, baseURL *url.URL) {
	dropped := 0
	for _, link := range links {
		// Parse and validate link
		linkURL, err := url.Parse(link)
		if err != nil {
			continue
		}

		// Convert relative URLs to absolute
		if !linkURL.IsAbs() {
			linkURL = baseURL.ResolveReference(linkURL)
		}

		// Apply filtering
		if c.shouldSkipURL(linkURL.String()) {
			continue
		}

		// Skip Apache directory listing sort links entirely
		if c.isApacheSortLink(linkURL) {
			continue
		}

		// Check if normalized URL was already visited or queued
		normalizedURL := c.normalizeURL(linkURL.String())
		if c.visited.IsVisited(normalizedURL) {
			continue
		}

		if !c.enqueueNonBlocking(&QueueItem{URL: linkURL.String(), Depth: depth}) {
			// Queue is full; the link is lost (no retry). Count it so the drop
			// is visible instead of silent.
			dropped++
		}
	}
	if dropped > 0 {
		c.stats.LinksDropped.Add(int64(dropped))
		c.reporter.Log("WARN", fmt.Sprintf("Queue full; dropped %d of %d discovered links from %s (total dropped: %d)",
			dropped, len(links), utils.TruncateURL(baseURL.String(), 60), c.stats.LinksDropped.Load()))
	}
}

// shouldSkipURL determines if a URL should be skipped based on patterns with enhanced security
func (c *EngineCrawler) shouldSkipURL(urlStr string) bool {
	// Use secure URL validation and parsing
	u, err := utils.ValidateAndParseURL(urlStr)
	if err != nil {
		return true // Skip invalid or insecure URLs
	}

	// Skip non-HTTP(S) schemes (already validated in ValidateAndParseURL)
	if utils.ShouldSkipURL(u) {
		return true
	}

	// Check if URL belongs to any seed URL's domain and base path
	allowed := false
	for _, seedURL := range c.seedURLs {
		if u.Host == seedURL.Host && c.isWithinBasePath(u, seedURL) {
			allowed = true
			break
		}
	}
	if !allowed {
		return true
	}

	// Check exclude patterns
	if utils.MatchesExcludePatterns(urlStr, c.config.ExcludePatterns) {
		return true
	}

	// Enforce robots.txt at the discovery boundary (links found mid-crawl,
	// not just seeds). Nil checker = no enforcement.
	if c.robotsAllowed != nil && !c.robotsAllowed(u.Path, c.config.UserAgent) {
		c.reporter.Log("WARN", fmt.Sprintf("Skipping %s: disallowed by robots.txt", utils.TruncateURL(urlStr, 60)))
		return true
	}
	return false
}

// applyRateLimit blocks until the per-domain rate limiter permits a request
// for rawURL, or until the crawler context is cancelled. Unlike a per-worker
// time.Sleep, this only paces requests to the SAME domain — workers crawling
// other domains proceed in parallel, and the wait respects ctx cancellation.
func (c *EngineCrawler) applyRateLimit(rawURL string) {
	if err := c.rateLimiter.Wait(c.ctx, rawURL); err != nil {
		// ctx cancelled or limiter error — stop pacing; the caller's
		// subsequent ctx checks / crawl will short-circuit.
		return
	}
}

// progressReporter periodically reports crawling progress
func (c *EngineCrawler) progressReporter() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.reporter.UpdateStats(ui.StatsMsg{
				PagesVisited:    c.stats.PagesVisited.Load(),
				PagesFailed:     c.stats.PagesFailed.Load(),
				PDFsDownloaded:  c.stats.PDFsDownloaded.Load(),
				BytesDownloaded: c.stats.BytesDownloaded.Load(),
				QueueSize:       len(c.queue),
				ActiveWorkers:   int(c.activeWorkers.Load()),
			})
		}
	}
}

// Cancel gracefully stops the crawler
func (c *EngineCrawler) Cancel() {
	c.cancel()

	// Drain queued work so blocked sends can observe the cancelled context.
	// The drain exits when Start closes c.queue after all workers return.
	go func() {
		for range c.queue {
			// Just drain, don't process
		}
	}()
}

// Close cancels the crawl, waits (bounded) for workers to drain, then
// releases the engine and visited store. Waiting prevents closing the engine
// while workers are still using it — which loses in-flight pages and, for
// browser engines, triggers operations on a page being torn down.
func (c *EngineCrawler) Close() {
	c.cancel()

	// Workers exit on ctx.Done, but a browser-engine worker mid-navigation may
	// not observe cancellation until its own timeout, so bound the wait rather
	// than block forever.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}

	// Ensure the queue is closed so any Cancel() drain goroutine unblocks even
	// if Start never reached its own close (e.g. force-quit path). Idempotent.
	c.queueClosed.Do(func() { close(c.queue) })

	if c.engine != nil {
		c.engine.Close()
	}
	if c.visited != nil {
		c.visited.Close()
	}
}

// getFilePath determines the file path for saving a URL's content
func (c *EngineCrawler) getFilePath(urlStr string) string {
	return utils.GenerateFilePath(c.config.OutputDir, urlStr)
}

// saveContent saves content to a file
func (c *EngineCrawler) saveContent(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(filePath, content, 0644)
}

// normalizeURL removes Apache directory listing sort parameters to avoid duplicates
func (c *EngineCrawler) normalizeURL(urlStr string) string {
	return utils.NormalizeURLString(urlStr)
}

// isApacheSortLink checks if a URL is an Apache directory listing sort link
func (c *EngineCrawler) isApacheSortLink(u *url.URL) bool {
	return utils.IsApacheSortLink(u)
}

// isWithinBasePath checks if a URL is within the base path of the start URL
func (c *EngineCrawler) isWithinBasePath(u, startURL *url.URL) bool {
	return utils.IsWithinBasePath(u, startURL)
}

// fetchExternalJSEndpoints extracts <script src="..."> URLs from HTML,
// fetches the external JS files, and parses them for API endpoints.
func (c *EngineCrawler) fetchExternalJSEndpoints(htmlContent []byte, baseURL *url.URL) []string {
	scriptURLs := ExtractScriptSrcURLs(htmlContent, baseURL)
	if len(scriptURLs) == 0 {
		return nil
	}

	var allEndpoints []string

	for _, scriptURL := range scriptURLs {
		// Skip if already fetched
		if _, loaded := c.fetchedJS.LoadOrStore(scriptURL, true); loaded {
			continue
		}

		// Respect rate limiting for the script's domain (full URL — limiter derives the domain)
		c.applyRateLimit(scriptURL)

		// Check context cancellation
		if c.ctx.Err() != nil {
			break
		}

		jsContent, err := c.fetchJSFile(scriptURL)
		if err != nil {
			c.reporter.Log("WARN", fmt.Sprintf("Failed to fetch JS: %s: %v", utils.TruncateURL(scriptURL, 60), err))
			continue
		}

		endpoints := ExtractJSFromRawSource(jsContent, baseURL)
		if len(endpoints) > 0 {
			c.reporter.Log("INFO", fmt.Sprintf("Parsed %s: %d endpoints", utils.TruncateURL(scriptURL, 60), len(endpoints)))
			allEndpoints = append(allEndpoints, endpoints...)
		}
	}

	return allEndpoints
}

// fetchJSFile fetches the content of an external JavaScript file.
func (c *EngineCrawler) fetchJSFile(jsURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, jsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// Limit read to 5MB to prevent memory issues with large bundles
	const maxJSSize = 5 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSSize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}
