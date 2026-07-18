package crawlers

import (
	"context"
	"time"

	"github.com/dotcommander/crawler/internal/config"
)

// CrawlResult represents the result of crawling a single page
type CrawlResult struct {
	URL           string
	Title         string
	StatusCode    int
	ContentLength int64
	Links         []string
	IsPDF         bool
	Success       bool
	Error         error
	Content       []byte
	ContentType   string
}

// CrawlEngine defines the interface for different crawling implementations
type CrawlEngine interface {
	// CrawlPage crawls a single page and returns the result
	CrawlPage(ctx context.Context, item *QueueItem) (*CrawlResult, error)

	// Close cleans up resources used by the engine
	Close() error

	// GetEngineType returns the type of the engine for logging/stats
	GetEngineType() string
}

// EngineType represents the different crawler engine types
type EngineType string

const (
	EngineTypeColly      EngineType = "colly"
	EngineTypeRod        EngineType = "rod"
	EngineTypePlaywright EngineType = "playwright"
)

// EngineSelector determines which engine to use based on configuration
func SelectEngine(cfg *config.CrawlerConfig) EngineType {
	// Force Rod for mobile emulation
	if cfg.Mobile {
		return EngineTypeRod
	}

	// Force Rod for custom wait strategies
	if cfg.WaitStrategy != "" && cfg.WaitStrategy != "networkidle" {
		return EngineTypeRod
	}

	// Force Rod for extra wait time (indicates JavaScript heavy sites)
	if cfg.ExtraWaitTime > 500*time.Millisecond {
		return EngineTypeRod
	}

	// Default to Colly for better performance
	return EngineTypeColly
}

// CreateEngine creates the appropriate crawler engine based on configuration
func CreateEngine(cfg *config.CrawlerConfig, engineType EngineType) (CrawlEngine, error) {
	switch engineType {
	case EngineTypeColly:
		return NewCollyEngine(cfg)
	case EngineTypeRod:
		return NewRodEngine(cfg)
	case EngineTypePlaywright:
		return NewPlaywrightEngine(cfg)
	default:
		// Auto-select based on configuration
		selectedType := SelectEngine(cfg)
		return CreateEngine(cfg, selectedType)
	}
}
