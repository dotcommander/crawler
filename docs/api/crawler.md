# API Reference: Crawler Interface

The `api.Crawler` interface provides the public API for programmatically
controlling web crawling operations.

## Interface Definition

```go
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
```

## Methods

### `Start() error`

Begins the crawling process using the configuration provided during crawler
creation.

**Returns:**

- `error` - Returns an error if the crawl fails to start or encounters a
  fatal error during execution. Returns `nil` on successful completion.

**Behavior:**

- Creates the output directory if it doesn't exist
- Validates the start URL
- Spawns worker goroutines based on the configured concurrency level
- Processes URLs until the queue is empty, max pages limit is reached,
  or context is cancelled
- Blocks until crawling completes or is cancelled

### `Cancel()`

Gracefully stops the crawling process by cancelling the internal context.

**Returns:**

- None

**Behavior:**

- Signals all active workers to stop processing
- Drains the URL queue to unblock workers
- Workers complete their current page before exiting
- Does not wait for workers to finish (use `Close()` for full cleanup)

### `Close()`

Cleans up all resources associated with the crawler.

**Returns:**

- None

**Behavior:**

- Cancels the internal context if not already cancelled
- Closes the underlying crawl engine (e.g., shuts down browser
  instances for Playwright)
- Should be called as a deferred function or after `Cancel()` to
  ensure proper cleanup

## Usage Example

```go
package main

import (
    "fmt"
    "log"

    "crawler/api"
    "crawler/internal/config"
    "crawler/internal/crawlers"
)

func main() {
    // Create configuration
    cfg := &config.CrawlerConfig{
        StartURL:    "https://example.com",
        OutputDir:   "./output",
        MaxDepth:    2,
        MaxPages:    100,
        Concurrency: 5,
    }

    // Create crawler (uses factory for proper initialization)
    crawler, err := crawlers.CreateCrawler(cfg, true, "colly")
    if err != nil {
        log.Fatalf("Failed to create crawler: %v", err)
    }

    // Ensure cleanup happens
    defer crawler.Close()

    // Start crawling (blocking)
    if err := crawler.Start(); err != nil {
        log.Fatalf("Crawl failed: %v", err)
    }

    fmt.Println("Crawling completed successfully")
}
```

## Concurrent Cancellation Example

```go
// Start crawler in background
go func() {
    if err := crawler.Start(); err != nil {
        log.Printf("Crawl stopped: %v", err)
    }
}()

// Wait for user input or signal
<-sigChan

// Gracefully stop the crawler
crawler.Cancel()

// Allow time for cleanup
time.Sleep(2 * time.Second)
crawler.Close()
```

## Implementation Notes

- The interface is implemented by `EngineCrawler` (internal) and
  `CrawlerWithUI` (with UI)
- Use `crawlers.CreateCrawler()` factory function for proper instantiation
- Always call `Close()` as a deferred function to prevent resource leaks
- `Cancel()` is optional if you let the crawl complete naturally
- The factory automatically selects the appropriate engine (Colly or
  Playwright) based on configuration
