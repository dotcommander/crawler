# Glossary

This glossary defines key terms and concepts used throughout the crawler project.

## A

### Atomic Operations
Thread-safe operations using Go's `sync/atomic` package for counters and
statistics. Ensures data consistency across concurrent workers without mutex
overhead.

**See also**: [Architecture Overview](../architecture/overview.md),
[Concurrency](../architecture/overview.md#concurrency-and-performance)

## B

### Bubbletea
A terminal UI framework used for the crawler's real-time interface. Provides
three modes: Simple (direct terminal output), Standard (basic UI), and Enhanced
(full-featured with animations).

**See also**: [UI Architecture](../architecture/ui.md),
[Development Guide](guides/development.md)

## C

### Circuit Breaker
A fault-tolerance pattern that prevents cascading failures by blocking requests
to domains with 3+ consecutive failures. Protects the crawler from wasting
resources on unreachable or hostile sites.

**Implementation**: `internal/crawlers/engine_crawler.go:42-50`

**See also**: [Reliability Patterns](../architecture/reliability.md#fault-tolerance)

### Collector
In Colly context, a domain-specific scraping instance with configurable
callbacks, rules, and rate limiting. Each domain gets its own collector to
isolate failures and respect per-domain rate limits.

**See also**: [Colly Engine](../architecture/engines.md#colly-engine),
[URL Filtering](../architecture/engines.md#url-filtering)

### Configuration Hierarchy
The precedence order for settings: **CLI flags > YAML config file > defaults**.
Allows flexible override behavior for different crawling scenarios.

**See also**: [Configuration Guide](guides/configuration.md),
[Development Guide](guides/development.md#configuration)

### CrawlEngine
The core abstraction interface that both Colly and Playwright implement.
Provides unified API (`CrawlPage`, `Close`, `GetEngineType`) for engine-agnostic
crawling logic.

**Interface**: `internal/crawlers/engines.go:9-27`

**See also**: [Engine Architecture](../architecture/engines.md),
[Adding Engines](guides/development.md#adding-new-engines)

### Crawler
The main orchestrator that coordinates workers, enforces rate limits, manages
circuit breakers, and reports progress. Wraps a specific `CrawlEngine`
implementation.

**Public API**: `api/crawler.go`

**See also**: [Architecture Overview](../architecture/overview.md)

## D

### Domain Delays
Per-domain rate limiting configuration (in milliseconds) enforced via
`time.Sleep()` after each page request. Prevents overwhelming servers and
avoids IP bans.

**Config field**: `domain-delays` in YAML

**See also**: [Configuration Guide](guides/configuration.md#rate-limiting),
[Best Practices](guides/best-practices.md#rate-limiting)

### Dual Engine Architecture
The design pattern supporting two interchangeable crawling engines (Colly for
speed, Playwright for JavaScript) with automatic selection based on configuration
flags.

**See also**: [Engine Architecture](../architecture/engines.md),
[Engine Selection](../architecture/engines.md#selection-logic)

## E

### Engine Auto-Selection
Automatic choice of Colly or Playwright based on:
- `Mobile: true` → Playwright (device emulation)
- Custom `WaitStrategy` → Playwright (complex timing)
- `ExtraWaitTime > 500ms` → Playwright (JS-heavy indicator)
- Default → Colly (optimal performance)

**Implementation**: `internal/crawlers/engines.go:60-82`

**See also**: [Engine Selection](../architecture/engines.md#selection-logic)

### EngineCrawler
The orchestration layer that wraps a `CrawlEngine` and coordinates workers,
circuit breakers, rate limiting, and progress reporting.

**Source**: `internal/crawlers/engine_crawler.go`

**See also**: [Architecture Overview](../architecture/overview.md#core-architecture-flow)

### Exclude Patterns
Regular expression patterns for filtering out unwanted URLs (e.g., `/api/`,
`/admin`, logout links). Applied in both engines for consistent filtering.

**Config field**: `exclude-patterns` in YAML

**See also**: [URL Filtering](../architecture/engines.md#url-filtering),
[Configuration Guide](guides/configuration.md#url-filtering)

## F

### Factory Pattern
The `CreateCrawler()` function that assembles the crawler: determines UI mode,
selects engine, creates reporter, and wires components together.

**Source**: `internal/crawlers/factory.go`

**See also**: [Architecture Overview](../architecture/overview.md#core-architecture-flow),
[Development Guide](guides/development.md#extension-points)

## H

### Headless Mode
Browser automation without visible UI. Playwright runs headless by default for
efficiency; use `--verbose` flag for plain text output in non-interactive
environments.

**See also**: [Testing Guide](guides/testing.md#headless-environments),
[Development Guide](guides/development.md#running-tests)

## I

### Interface Abstraction
The separation of concerns through Go interfaces:
- `api.Crawler` - Main public interface
- `CrawlEngine` - Pluggable engine interface
- `ProgressReporter` - Decoupled progress reporting

**See also**: [Architecture Overview](../architecture/overview.md#interface-abstraction),
[Design Patterns](../architecture/design-patterns.md)

## M

### Mobile Emulation
Playwright engine capability to simulate mobile devices (default: iPhone 14) for
testing responsive layouts and mobile-specific content. Automatically selects
Playwright engine.

**Config flag**: `--mobile`

**See also**: [Engine Selection](../architecture/engines.md#selection-logic),
[Configuration Guide](guides/configuration.md#mobile-emulation)

## P

### PagePool
A Playwright-specific optimization that reuses browser pages across requests,
maintaining warm connections to reduce overhead. Similar to HTTP connection
pooling.

**Implementation**: `internal/crawlers/playwright_engine.go`

**See also**: [Playwright Engine](../architecture/engines.md#playwright-engine),
[Performance](../architecture/performance.md)

### Playwright Engine
Browser automation engine using Microsoft Playwright for JavaScript-heavy sites.
Slower than Colly but handles dynamic content, custom wait strategies, and mobile
emulation.

**Source**: `internal/crawlers/playwright_engine.go`

**See also**: [Engine Architecture](../architecture/engines.md#playwright-engine),
[Engine Selection](../architecture/engines.md#selection-logic)

### ProgressReporter
Interface for decoupling progress tracking from UI implementation. Methods:
`Log()`, `UpdateStats()`, `UpdateWorker()`.

**Interface**: `internal/crawlers/reporter.go`

**See also**: [UI Architecture](../architecture/ui.md),
[Extension Points](guides/development.md#extension-points)

## S

### Semaphore
Concurrency control mechanism limiting active workers to `config.Concurrency`.
Prevents resource exhaustion by capping parallel requests.

**See also**: [Concurrency](../architecture/overview.md#concurrency-and-performance),
[Configuration Guide](guides/configuration.md#concurrency)

### Standard Library
Go's built-in packages (`fmt`, `net/http`, `sync/atomic`, etc.) preferred over
external dependencies unless functionality is unavailable (e.g., browser
automation).

**See also**: [Project Principles](../architecture/principles.md),
[Best Practices](guides/best-practices.md#dependencies)

## U

### UnifiedUI
Single UI class supporting three modes (Simple, Standard, Enhanced) based on environment variables (`CRAWLER_LEGACY_UI`, `CRAWLER_STANDARD_UI`).

**Source**: `ui/unified.go`

**See also**: [UI Architecture](../architecture/ui.md), [Development Guide](guides/development.md#ui-modes)

### URL Filtering
Constraint ensuring crawler stays within same domain AND base path. Prevents drifting to external sites or overwhelming subdirectories.

**Implementation**: `internal/crawlers/engine_crawler.go:52-78`

**See also**: [Colly Engine](../architecture/engines.md#colly-engine), [Best Practices](guides/best-practices.md#scope-control)

## V

### Viper
Configuration library for unifying YAML config files and CLI flags. Provides the configuration hierarchy: flags > YAML > defaults.

**See also**: [Configuration Guide](guides/configuration.md), [Development Guide](guides/development.md#configuration)

### Verbose Mode
Plain text logging bypassing the Bubbletea UI. Useful for debugging, headless environments, or log file redirection.

**CLI flag**: `--verbose`

**See also**: [Testing Guide](guides/testing.md#headless-environments), [Development Guide](guides/development.md#running-the-crawler)

## W

### WaitStrategy
Playwright-specific configuration for complex timing scenarios (e.g., waiting for network idle, specific selectors, or custom conditions). Forces Playwright engine selection.

**Options**: `network-idle`, `selector`, `timeout`, custom

**See also**: [Engine Selection](../architecture/engines.md#selection-logic), [Configuration Guide](guides/configuration.md#wait-strategies)

### Workers
Concurrent goroutines that execute page requests coordinated by a semaphore. Each worker gets URLs from the queue, processes them, and reports progress.

**Config field**: `concurrency` in YAML (default: 5)

**See also**: [Concurrency](../architecture/overview.md#concurrency-and-performance), [Configuration Guide](guides/configuration.md#concurrency)

## Y

### YAML Configuration
File-based configuration (`~/.config/crawler/config.yaml`) for default settings. Overridden by CLI flags but overrides built-in defaults.

**See also**: [Configuration Guide](guides/configuration.md), [Development Guide](guides/development.md#configuration)
