package api

// Crawler defines the interface for web crawlers
type Crawler interface {
	// Start begins the crawling process
	Start() error

	// Cancel gracefully stops the crawler
	Cancel()

	// Close cleans up resources
	Close()
}
