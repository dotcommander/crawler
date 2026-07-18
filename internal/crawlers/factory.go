package crawlers

import (
	"fmt"
	"os"
	"time"

	"github.com/dotcommander/crawler/api"
	"github.com/dotcommander/crawler/internal/config"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/session"
	"github.com/dotcommander/crawler/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-isatty"
)

// CreateCrawler creates a crawler instance based on the configuration, UI preference, and engine type
func CreateCrawler(cfg *config.CrawlerConfig, useVerboseMode bool, engineType string, store session.VisitedStore) (api.Crawler, error) {
	var mode ui.UIMode

	if cfg.Quiet {
		return NewEngineCrawler(cfg, &NoOpReporter{}, engineType, store)
	}

	// Verbose or no TTY: plain log output (no ANSI/alt-screen corruption in piped/CI).
	noTTY := !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd())
	if useVerboseMode || noTTY {
		return NewEngineCrawler(cfg, &LogReporter{}, engineType, store)
	}

	// Determine UI mode
	if os.Getenv("CRAWLER_LEGACY_UI") == "1" {
		mode = ui.ModeSimple
	} else if os.Getenv("CRAWLER_STANDARD_UI") == "1" {
		mode = ui.ModeStandard
	} else {
		mode = ui.ModeEnhanced // Default to enhanced UI
	}

	// Create unified UI instance
	unifiedUI := ui.NewUnifiedUI(mode, cfg.StartURL, cfg.OutputDir, cfg.MaxDepth, cfg.Concurrency)

	if !unifiedUI.IsBubbletea() {
		// Simple mode doesn't use Bubbletea
		return NewCrawlerWithSimpleUI(cfg, unifiedUI, engineType, store)
	}

	// Standard and Enhanced modes use Bubbletea
	return NewCrawlerWithBubbletea(cfg, unifiedUI, engineType, store)
}

// CrawlerWithUI wraps a crawler with a UI implementation
type CrawlerWithUI struct {
	Crawler api.Crawler
	uiImpl  ui.UI
	program *tea.Program
}

// NewCrawlerWithSimpleUI creates a crawler with simple terminal UI
func NewCrawlerWithSimpleUI(config *config.CrawlerConfig, uiImpl ui.UI, engineType string, store session.VisitedStore) (*CrawlerWithUI, error) {
	reporter := &UnifiedUIReporter{ui: uiImpl, simpleMode: true}
	crawler, err := NewEngineCrawler(config, reporter, engineType, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine crawler with simple UI: %w", err)
	}

	return &CrawlerWithUI{
		Crawler: &crawlerAdapter{crawler},
		uiImpl:  uiImpl,
	}, nil
}

// NewCrawlerWithBubbletea creates a crawler with Bubbletea UI
func NewCrawlerWithBubbletea(config *config.CrawlerConfig, uiImpl ui.UI, engineType string, store session.VisitedStore) (*CrawlerWithUI, error) {
	program := tea.NewProgram(uiImpl)
	reporter := &UnifiedUIReporter{ui: uiImpl, program: program}
	crawler, err := NewEngineCrawler(config, reporter, engineType, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine crawler with Bubbletea UI: %w", err)
	}

	return &CrawlerWithUI{
		Crawler: &crawlerAdapter{crawler},
		uiImpl:  uiImpl,
		program: program,
	}, nil
}

// crawlerAdapter adapts EngineCrawler to the api.Crawler interface
type crawlerAdapter struct {
	*EngineCrawler
}

// SetExporter forwards to the embedded EngineCrawler
func (a *crawlerAdapter) SetExporter(exp exporters.Exporter) {
	a.EngineCrawler.SetExporter(exp)
}

// Start runs the crawler with UI
func (c *CrawlerWithUI) Start() error {
	if c.program != nil {
		// Bubbletea mode
		errChan := make(chan error, 1)
		crawlerDone := make(chan struct{})
		// programDone is closed once program.Run() returns so any goroutine
		// trying to Send into the program after UI shutdown bails out cleanly.
		programDone := make(chan struct{})

		// Exit condition: crawler finishes, then either Send succeeds OR
		// programDone fires (UI already exited) — never blocks on a dead program.
		go func() {
			err := c.Crawler.Start()
			close(crawlerDone)
			errChan <- err
			select {
			case <-programDone:
				// UI exited before crawl finished; skip Send to avoid send-after-close race.
			default:
				c.program.Send(ui.DoneMsg{})
			}
		}()
		defer close(programDone)

		if _, err := c.program.Run(); err != nil {
			// If UI exits with error, cancel crawler
			c.Crawler.Cancel()
			<-crawlerDone // Wait for crawler to finish
			return err
		}

		// Check if crawler is still running (UI quit via 'q')
		select {
		case err := <-errChan:
			// Crawler finished naturally
			return err
		default:
			// UI quit but crawler still running - cancel it
			c.Crawler.Cancel()
			// Wait for crawler to finish with timeout
			t := time.NewTimer(2 * time.Second)
			defer t.Stop()
			select {
			case err := <-errChan:
				return err
			case <-t.C:
				// Force close if not stopped within timeout
				c.Crawler.Close()
				return nil
			}
		}
	}

	// Simple mode
	c.uiImpl.PrintHeader()
	err := c.Crawler.Start()

	return err
}

// Cancel stops the crawler
func (c *CrawlerWithUI) Cancel() {
	c.Crawler.Cancel()
	if c.program != nil {
		c.program.Send(ui.DoneMsg{})
	}
}

// Close cleans up resources
func (c *CrawlerWithUI) Close() {
	c.Crawler.Close()
	if c.program != nil {
		c.program.Quit()
	}
}

// GetInnerCrawler returns the underlying crawler for configuration
func (c *CrawlerWithUI) GetInnerCrawler() api.Crawler {
	return c.Crawler
}
