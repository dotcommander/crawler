package crawlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dotcommander/crawler/internal/config"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

// CollyEngine implements the CrawlEngine interface using Colly
type CollyEngine struct {
	mu         sync.Mutex
	collectors map[string]*colly.Collector // Domain-specific collectors for rate limiting
	config     *config.CrawlerConfig
}

// NewCollyEngine creates a new Colly-based crawler engine
func NewCollyEngine(cfg *config.CrawlerConfig) (*CollyEngine, error) {
	engine := &CollyEngine{
		collectors: make(map[string]*colly.Collector),
		config:     cfg,
	}

	return engine, nil
}

// getOrCreateCollector returns a domain-specific collector with appropriate rate limiting
func (e *CollyEngine) getOrCreateCollector(domain string) *colly.Collector {
	e.mu.Lock()
	defer e.mu.Unlock()

	if collector, exists := e.collectors[domain]; exists {
		return collector
	}

	// Create new collector for this domain.
	// Cap response body at 10MB to bound memory (workspace rule: never buffer
	// unbounded input). Set explicitly rather than relying on colly's default.
	c := colly.NewCollector(
		colly.UserAgent(e.getUserAgent()),
		colly.MaxBodySize(10*1024*1024),
	)

	// Set up debugging if verbose mode is enabled
	if false { // TODO: Add verbose flag support
		c.OnRequest(func(r *colly.Request) {
			fmt.Printf("Visiting %s\n", r.URL.String())
		})
		c.SetDebugger(&debug.LogDebugger{})
	}

	// Configure limits
	limit := e.getDomainDelay(domain)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*" + domain + "*",
		Parallelism: 1, // Prevent multiple simultaneous requests to same domain
		Delay:       limit,
	})

	// Set request timeout
	c.SetRequestTimeout(30 * time.Second)

	// Configure headers
	if e.config.Headers != nil {
		c.OnRequest(func(r *colly.Request) {
			for key, value := range e.config.Headers {
				r.Headers.Set(key, value)
			}
		})
	}

	// Handle redirects
	c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
		// Follow up to 10 redirects
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	})

	// Store collector for reuse
	e.collectors[domain] = c
	return c
}

// CrawlPage crawls a single page using Colly
func (e *CollyEngine) CrawlPage(ctx context.Context, item *QueueItem) (*CrawlResult, error) {
	result := &CrawlResult{
		URL:     item.URL,
		Links:   make([]string, 0),
		IsPDF:   false,
		Success: false,
	}

	parsedURL, err := url.Parse(item.URL)
	if err != nil {
		result.Error = fmt.Errorf("invalid URL: %w", err)
		return result, nil
	}

	// Check if this is a PDF URL
	if strings.HasSuffix(strings.ToLower(parsedURL.Path), ".pdf") {
		return e.downloadPDF(ctx, item.URL, result)
	}

	// Get domain-specific collector
	collector := e.getOrCreateCollector(parsedURL.Host)

	// Clone collector for this request to avoid conflicts
	c := collector.Clone()

	// Pre-visit cancellation check at the request boundary: if the caller
	// already cancelled (or the deadline passed), don't start a request.
	if err := ctx.Err(); err != nil {
		result.Error = fmt.Errorf("crawl cancelled for %s: %w", item.URL, err)
		return result, nil
	}

	// Handle successful responses
	c.OnHTML("html", func(e *colly.HTMLElement) {
		result.ContentLength = int64(len(e.Response.Body))
		result.StatusCode = e.Response.StatusCode
		result.Content = e.Response.Body
		result.ContentType = e.Response.Headers.Get("Content-Type")

		// Extract title
		result.Title = e.ChildText("head > title")

		// Extract links
		e.ForEach("a[href]", func(_ int, el *colly.HTMLElement) {
			link := el.Attr("href")
			if link != "" {
				// Convert relative URLs to absolute
				if absoluteURL := e.Request.AbsoluteURL(link); absoluteURL != "" {
					result.Links = append(result.Links, absoluteURL)
				}
			}
		})

		result.Success = true
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		result.Error = fmt.Errorf("colly crawl failed for %s: %w", item.URL, err)
		result.StatusCode = r.StatusCode
	})

	// Handle non-HTML responses
	c.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		result.ContentLength = int64(len(r.Body))
		result.ContentType = r.Headers.Get("Content-Type")

		ct := r.Headers.Get("Content-Type")
		// Only retain the body for content we actually process downstream
		// (HTML or PDF). For other types (binary, images, archives, etc.)
		// skip storing the body to avoid holding large/rogue payloads.
		if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/pdf") {
			result.Content = r.Body
		}

		// If it's not HTML, we still consider it successful
		if !strings.Contains(ct, "text/html") {
			result.Success = true
		}
	})

	// Run the visit in a goroutine and select on ctx.Done() so the caller
	// stops blocking immediately on cancellation.
	// Limitation: Colly v2's Visit/Wait accept no context, so the in-flight
	// HTTP request itself is not aborted — it still runs to the 30s
	// SetRequestTimeout. We only stop waiting on it here.
	done := make(chan error, 1)
	go func() {
		visitErr := c.Visit(item.URL)
		c.Wait()
		done <- visitErr
	}()

	select {
	case <-ctx.Done():
		result.Error = fmt.Errorf("crawl cancelled for %s: %w", item.URL, ctx.Err())
		return result, nil
	case err = <-done:
		if err != nil {
			result.Error = fmt.Errorf("failed to visit URL %s: %w", item.URL, err)
			return result, nil
		}
	}

	return result, nil
}

// downloadPDF handles PDF file downloads
func (e *CollyEngine) downloadPDF(ctx context.Context, pdfURL string, result *CrawlResult) (*CrawlResult, error) {
	parsedURL, _ := url.Parse(pdfURL)
	collector := e.getOrCreateCollector(parsedURL.Host)

	c := collector.Clone()

	// Pre-visit cancellation check at the request boundary.
	if err := ctx.Err(); err != nil {
		result.Error = fmt.Errorf("PDF download cancelled for %s: %w", pdfURL, err)
		return result, nil
	}

	c.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		result.ContentLength = int64(len(r.Body))
		result.Content = r.Body
		result.ContentType = r.Headers.Get("Content-Type")
		result.IsPDF = true

		if r.StatusCode == 200 {
			result.Success = true
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		result.Error = fmt.Errorf("PDF download failed for %s: %w", pdfURL, err)
		result.StatusCode = r.StatusCode
	})

	// Same goroutine+select pattern as CrawlPage; the in-flight request is
	// not aborted (Colly limitation) but the caller no longer blocks on it.
	done := make(chan error, 1)
	go func() {
		visitErr := c.Visit(pdfURL)
		c.Wait()
		done <- visitErr
	}()

	select {
	case <-ctx.Done():
		result.Error = fmt.Errorf("PDF download cancelled for %s: %w", pdfURL, ctx.Err())
		return result, nil
	case err := <-done:
		if err != nil {
			result.Error = fmt.Errorf("failed to download PDF %s: %w", pdfURL, err)
			return result, nil
		}
	}
	return result, nil
}

// getUserAgent returns the configured user agent string. The desktop default
// is seeded via viper (config.defaultDesktopUserAgent); an empty value here
// means a caller built CrawlerConfig directly without one — Colly then uses
// its own built-in agent, which is acceptable.
func (e *CollyEngine) getUserAgent() string {
	return e.config.UserAgent
}

// getDomainDelay returns the appropriate delay for the given domain
func (e *CollyEngine) getDomainDelay(domain string) time.Duration {
	// Check for domain-specific delays
	if delay, exists := e.config.DomainDelays[domain]; exists {
		return delay
	}

	// Use default delay
	return e.config.DefaultDelay
}

// Close cleans up resources used by the Colly engine
func (e *CollyEngine) Close() error {
	// Colly collectors don't need explicit cleanup
	// Clear the collectors map to help with garbage collection
	e.collectors = nil
	return nil
}

// GetEngineType returns the engine type for logging/stats
func (e *CollyEngine) GetEngineType() string {
	return string(EngineTypeColly)
}
