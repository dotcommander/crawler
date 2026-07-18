package crawlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dotcommander/crawler/internal/config"

	"github.com/playwright-community/playwright-go"
)

// PagePool manages a pool of browser pages for efficient resource usage
type PagePool struct {
	pages     chan playwright.Page
	closePage func(playwright.Page)
	mu        sync.Mutex
	closed    bool
	releaseWG sync.WaitGroup // tracks in-flight Release goroutines so Close can drain them
}

// newContextOptions builds BrowserNewContextOptions from crawler config.
// All pages in the pool share the same options, so compute once.
func newContextOptions(cfg *config.CrawlerConfig) playwright.BrowserNewContextOptions {
	opts := playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
	}
	if cfg.Mobile {
		opts.Viewport = &playwright.Size{Width: MobileViewportWidth, Height: MobileViewportHeight}
		opts.UserAgent = playwright.String(cfg.MobileUserAgent)
		opts.HasTouch = playwright.Bool(true)
		opts.DeviceScaleFactor = playwright.Float(2.0)
	} else {
		opts.Viewport = &playwright.Size{Width: 1280, Height: 720}
		if cfg.UserAgent != "" {
			opts.UserAgent = playwright.String(cfg.UserAgent)
		}
	}
	if len(cfg.Headers) > 0 {
		opts.ExtraHttpHeaders = cfg.Headers
	}
	return opts
}

// routeFilter blocks heavyweight resources while allowing PDFs.
func routeFilter(route playwright.Route) {
	req := route.Request()
	if strings.HasSuffix(strings.ToLower(req.URL()), ".pdf") {
		route.Continue()
		return
	}
	switch req.ResourceType() {
	case "image", "stylesheet", "font", "script":
		route.Abort()
	default:
		route.Continue()
	}
}

// NewPagePool creates a new page pool with the specified size
func NewPagePool(browser playwright.Browser, size int, cfg *config.CrawlerConfig) (*PagePool, error) {
	pool := &PagePool{
		pages:     make(chan playwright.Page, size),
		closePage: closePageContext,
	}

	contextOptions := newContextOptions(cfg)

	for i := 0; i < size; i++ {
		context, err := browser.NewContext(contextOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create context: %w", err)
		}

		page, err := context.NewPage()
		if err != nil {
			return nil, fmt.Errorf("failed to create page: %w", err)
		}

		if err = page.Route("**/*", routeFilter); err != nil {
			return nil, fmt.Errorf("failed to set up routing: %w", err)
		}

		pool.pages <- page
	}

	return pool, nil
}

// Acquire gets a page from the pool. It never returns a nil page: if the pool
// is closed (or closes while waiting) it returns an error, preventing a nil
// dereference in the caller's subsequent page operations during shutdown.
func (p *PagePool) Acquire() (playwright.Page, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("page pool closed")
	}
	p.mu.Unlock()

	t := time.NewTimer(30 * time.Second)
	defer t.Stop()
	select {
	case page, ok := <-p.pages:
		if !ok {
			return nil, fmt.Errorf("page pool closed")
		}
		return page, nil
	case <-t.C:
		return nil, fmt.Errorf("timeout acquiring page from pool")
	}
}

// Release returns a page to the pool
func (p *PagePool) Release(page playwright.Page) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		p.close(page)
		return
	}
	p.releaseWG.Add(1)
	p.mu.Unlock()

	// Clear page state in goroutine.
	// Exit condition: Goto completes (bounded by Playwright timeout), then
	// the closed-check under p.mu prevents send-on-closed-channel; Close()
	// drains via releaseWG before close(p.pages) so this send is safe.
	go func() {
		defer p.releaseWG.Done()
		page.Goto("about:blank", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
		})

		shouldClose := false
		p.mu.Lock()
		if !p.closed {
			select {
			case p.pages <- page:
				// Successfully released
			default:
				shouldClose = true
			}
		} else {
			shouldClose = true
		}
		p.mu.Unlock()
		if shouldClose {
			p.close(page)
		}
	}()
}

func (p *PagePool) close(page playwright.Page) {
	if p.closePage != nil {
		p.closePage(page)
		return
	}
	closePageContext(page)
}

func closePageContext(page playwright.Page) {
	if page == nil || page.Context() == nil {
		return
	}
	_ = page.Context().Close()
}

// Close shuts down the page pool
func (p *PagePool) Close() {
	// Mark closed under the lock so new Release calls bail before launching
	// goroutines, then release the lock so in-flight Release goroutines can
	// reach their p.mu.Lock() and observe p.closed.
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	// Wait for all in-flight Release goroutines to finish before closing the
	// channel — prevents send-on-closed-channel panic.
	p.releaseWG.Wait()

	p.mu.Lock()
	close(p.pages)
	p.mu.Unlock()

	// Close all pages — no lock needed; channel is closed and releaseWG drained.
	for page := range p.pages {
		p.close(page)
	}
}
