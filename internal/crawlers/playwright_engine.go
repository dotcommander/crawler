package crawlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotcommander/crawler/internal/config"
	"github.com/playwright-community/playwright-go"
)

// PlaywrightEngine implements the CrawlEngine interface using Playwright
type PlaywrightEngine struct {
	config   *config.CrawlerConfig
	pw       *playwright.Playwright
	browser  playwright.Browser
	pagePool *PagePool
}

type playwrightTitleReader interface {
	Title() (string, error)
}

func setPlaywrightTitle(page playwrightTitleReader, result *CrawlResult) error {
	title, err := page.Title()
	if err != nil {
		return fmt.Errorf("read page title: %w", err)
	}
	result.Title = title
	return nil
}

// NewPlaywrightEngine creates a new Playwright-based crawler engine
func NewPlaywrightEngine(cfg *config.CrawlerConfig) (*PlaywrightEngine, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-dev-shm-usage",
			"--disable-extensions",
			"--disable-gpu",
			"--no-sandbox",
			"--disable-setuid-sandbox",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	// Create page pool for performance
	pagePool, err := NewPagePool(browser, cfg.Concurrency, cfg)
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("could not create page pool: %w", err)
	}

	return &PlaywrightEngine{
		config:   cfg,
		pw:       pw,
		browser:  browser,
		pagePool: pagePool,
	}, nil
}

// CrawlPage crawls a single page using Playwright
func (e *PlaywrightEngine) CrawlPage(ctx context.Context, item *QueueItem) (*CrawlResult, error) {
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
	page, err := e.pagePool.Acquire()
	if err != nil {
		result.Error = fmt.Errorf("failed to acquire page: %w", err)
		return result, nil
	}
	defer e.pagePool.Release(page)

	// Navigate to the page
	response, err := page.Goto(item.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000), // 30 second timeout
	})

	if err != nil {
		result.Error = fmt.Errorf("failed to navigate to %s: %w", item.URL, err)
		return result, nil
	}

	// Get response information
	if response != nil {
		result.StatusCode = response.Status()
	}

	// Wait for additional content if configured
	if e.config.ExtraWaitTime > 0 {
		time.Sleep(e.config.ExtraWaitTime)
	}

	// Title is optional metadata; a lookup failure must not discard a page that
	// loaded successfully or prevent link discovery.
	_ = setPlaywrightTitle(page, result)

	// Get page content length
	content, err := page.Content()
	if err == nil {
		result.ContentLength = int64(len(content))
	}

	// Extract links
	links, err := page.Locator("a[href]").EvaluateAll(`links => links.map(link => link.href)`)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract links: %w", err)
		return result, nil
	}

	// Convert interface{} slice to string slice
	if linkSlice, ok := links.([]interface{}); ok {
		for _, link := range linkSlice {
			if linkStr, ok := link.(string); ok && linkStr != "" {
				result.Links = append(result.Links, linkStr)
			}
		}
	}

	result.Success = true
	return result, nil
}

// downloadPDF handles PDF file downloads using Playwright
func (e *PlaywrightEngine) downloadPDF(ctx context.Context, pdfURL string, result *CrawlResult) (*CrawlResult, error) {
	return downloadPDFWithHTTP(ctx, e.config, pdfURL, result)
}

// Close cleans up resources used by the Playwright engine
func (e *PlaywrightEngine) Close() error {
	var errs []error

	if e.pagePool != nil {
		e.pagePool.Close()
	}

	if e.browser != nil {
		if err := e.browser.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close browser: %w", err))
		}
	}

	if e.pw != nil {
		if err := e.pw.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop playwright: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during cleanup: %v", errs)
	}

	return nil
}

// GetEngineType returns the engine type for logging/stats
func (e *PlaywrightEngine) GetEngineType() string {
	return string(EngineTypePlaywright)
}
