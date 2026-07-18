package crawlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotcommander/crawler/internal/config"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// RodEngine implements the CrawlEngine interface using Rod
type RodEngine struct {
	config  *config.CrawlerConfig
	browser *rod.Browser
	pool    rod.Pool[rod.Page]
}

type rodTitleFinder interface {
	Has(selector string) (bool, *rod.Element, error)
}

func readRodTitle(page rodTitleFinder) (string, error) {
	found, titleElement, err := page.Has("title")
	if err != nil {
		return "", fmt.Errorf("find page title: %w", err)
	}
	if !found {
		return "", nil
	}
	if titleElement == nil {
		return "", fmt.Errorf("find page title: title element is nil")
	}
	title, err := titleElement.Text()
	if err != nil {
		return "", fmt.Errorf("read page title: %w", err)
	}
	return title, nil
}

func setRodTitle(page rodTitleFinder, result *CrawlResult) {
	title, err := readRodTitle(page)
	if err == nil {
		result.Title = title
	}
}

// NewRodEngine creates a new Rod-based crawler engine
func NewRodEngine(cfg *config.CrawlerConfig) (*RodEngine, error) {
	// Launch browser with optimized flags
	browser := rod.New()
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("could not connect rod browser: %w", err)
	}

	// Create pool for concurrent page reuse
	pool := rod.NewPagePool(cfg.Concurrency)

	return &RodEngine{
		config:  cfg,
		browser: browser,
		pool:    pool,
	}, nil
}

// CrawlPage crawls a single page using Rod
func (e *RodEngine) CrawlPage(ctx context.Context, item *QueueItem) (*CrawlResult, error) {
	result := &CrawlResult{
		URL:     item.URL,
		Links:   make([]string, 0),
		IsPDF:   false,
		Success: false,
	}

	// Check if this is a PDF URL
	if strings.HasSuffix(strings.ToLower(item.URL), ".pdf") {
		result.IsPDF = true
		return e.downloadPDF(ctx, item.URL, result)
	}

	// Get a page from the pool
	page, err := e.pool.Get(func() (*rod.Page, error) {
		p, err := e.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return nil, fmt.Errorf("failed to create page: %w", err)
		}
		// Configure mobile settings if enabled
		if e.config.Mobile {
			err := proto.PageSetDeviceMetricsOverride{
				Width:             MobileViewportWidth,
				Height:            MobileViewportHeight,
				DeviceScaleFactor: 2.0,
				Mobile:            true,
			}.Call(p)
			if err != nil {
				return nil, fmt.Errorf("failed to set viewport: %w", err)
			}

			// Set mobile user agent
			err = proto.NetworkSetUserAgentOverride{UserAgent: e.config.MobileUserAgent}.Call(p)
			if err != nil {
				return nil, fmt.Errorf("failed to set user agent: %w", err)
			}
		} else if e.config.UserAgent != "" {
			// Set custom user agent if provided
			err := proto.NetworkSetUserAgentOverride{UserAgent: e.config.UserAgent}.Call(p)
			if err != nil {
				return nil, fmt.Errorf("failed to set user agent: %w", err)
			}
		}
		return p, nil
	})
	if err != nil {
		result.Error = fmt.Errorf("failed to get page from pool: %w", err)
		return result, nil
	}
	defer e.pool.Put(page)

	// Navigate to the page with networkidle wait
	err = page.Context(ctx).Timeout(30 * time.Second).Navigate(item.URL)
	if err != nil {
		result.Error = fmt.Errorf("failed to navigate to %s: %w", item.URL, err)
		return result, nil
	}

	// Wait for page load and network idle
	err = page.Context(ctx).Timeout(30 * time.Second).WaitLoad()
	if err != nil {
		result.Error = fmt.Errorf("failed to wait for load: %w", err)
		return result, nil
	}

	err = page.Context(ctx).Timeout(30 * time.Second).WaitIdle(2 * time.Second)
	if err != nil {
		result.Error = fmt.Errorf("failed to wait for network idle: %w", err)
		return result, nil
	}

	// Set default status code (navigation succeeded)
	result.StatusCode = 200

	// Wait for additional content if configured
	if e.config.ExtraWaitTime > 0 {
		time.Sleep(e.config.ExtraWaitTime)
	}

	// Get page content length
	content, err := page.HTML()
	if err != nil {
		result.Error = fmt.Errorf("failed to read page HTML: %w", err)
		return result, nil
	}
	result.ContentLength = int64(len(content))

	// Extract page title with caller cancellation and a bounded query timeout.
	// Title is optional metadata, so lookup errors do not fail the crawl.
	setRodTitle(page.Context(ctx).Timeout(5*time.Second), result)

	// Extract links using Rod
	links, err := page.Elements("a[href]")
	if err != nil {
		result.Error = fmt.Errorf("failed to extract links: %w", err)
		return result, nil
	}
	for _, link := range links {
		href, err := link.Attribute("href")
		if err != nil {
			result.Error = fmt.Errorf("failed to read link href: %w", err)
			return result, nil
		}
		if href != nil && *href != "" {
			result.Links = append(result.Links, *href)
		}
	}

	result.Success = true
	return result, nil
}

// downloadPDF handles PDF file downloads using Rod
func (e *RodEngine) downloadPDF(ctx context.Context, pdfURL string, result *CrawlResult) (*CrawlResult, error) {
	return downloadPDFWithHTTP(ctx, e.config, pdfURL, result)
}

// Close cleans up resources used by the Rod engine
func (e *RodEngine) Close() error {
	var errs []error

	if e.pool != nil {
		e.pool.Cleanup(func(page *rod.Page) {
			if err := page.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close page: %w", err))
			}
		})
	}

	if e.browser != nil {
		if err := e.browser.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close browser: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during cleanup: %v", errs)
	}

	return nil
}

// GetEngineType returns the engine type for logging/stats
func (e *RodEngine) GetEngineType() string {
	return string(EngineTypeRod)
}
