# Glossary

This glossary defines key terms and concepts used throughout the crawler project.

## A

### Atomic Operations
Thread-safe operations using Go's `sync/atomic` package for counters and
statistics. Ensures data consistency across concurrent workers without mutex
overhead.


## B

### Bubbletea
A terminal UI framework used for the crawler's real-time interface. Provides
three modes: Simple (direct terminal output), Standard (basic UI), and Enhanced
(full-featured with animations).


## C

### Configuration Hierarchy
The precedence order for settings: **CLI flags > YAML config file > defaults**.
Allows flexible override behavior for different crawling scenarios.

**See also**: [Configuration Guide](guides/configuration.md),
[Development Guide](guides/development.md#configuration)

### Crawler
The main orchestrator (`KatanaCrawler`) that owns structured exports, content
saving, CSS extraction, and sitemap seeding on top of the katana engine
adapter, and reports progress through the reporter interface.

**Public API**: `api/crawler.go`

## D

### Domain Delays
Per-domain rate limiting configuration (in milliseconds) enforced via
`time.Sleep()` after each page request. Prevents overwhelming servers and
avoids IP bans.

**Config field**: `domain-delays` in YAML

**See also**: [Configuration Guide](guides/configuration.md#rate-limiting),
[Best Practices](guides/best-practices.md#rate-limiting)

### Delay Groups
Seed URLs grouped by their `domainDelays` entry; each group runs as its own
katana engine with the group's delay, preserving per-domain rate limiting
(katana has one global delay per engine).

## E

### Engine Auto-Selection
Automatic choice of katana engine mode based on:
- `Mobile: true` → headless (device emulation)
- Custom `WaitStrategy` → headless (complex timing)
- `ExtraWaitTime > 500ms` → headless (JS-heavy indicator)
- Default → standard HTTP engine (optimal performance)

**Implementation**: `internal/katanaengine/groups.go` (`WantsHeadless`)

### KatanaEngine
The adapter (`internal/katanaengine`) translating `CrawlerConfig` into katana
`types.Options`, running katana's standard or headless engine, and mapping its
results onto reporter, visited-store, and page-callback surfaces.

**Source**: `internal/katanaengine/engine.go`

### Exclude Patterns
Regular expression patterns for filtering out unwanted URLs (e.g., `/api/`,
`/admin`, logout links). Applied in both engines for consistent filtering.

**Config field**: `exclude-patterns` in YAML


## F

### Factory Pattern
The `CreateCrawler()` function that assembles the crawler: determines UI mode,
selects engine, creates reporter, and wires components together.

**Source**: `internal/crawlers/factory.go`


## H

### Headless Mode
Browser automation without visible UI. The katana headless engine launches a
managed Chromium on demand; use `--verbose` flag for plain text output in
non-interactive environments.

**See also**: [Testing Guide](guides/testing.md#headless-environments),
[Development Guide](guides/development.md#running-tests)

## I

### Interface Abstraction
The separation of concerns through Go interfaces:
- `api.Crawler` - Main public interface
- `Reporter` - Engine-level progress reporting contract
- `ProgressReporter` - UI-level progress reporting


## M

### Mobile Emulation
Headless-engine capability to simulate mobile devices (mobile user agent plus
viewport/touch arguments) for testing responsive layouts and mobile-specific
content. Automatically selects the headless engine.

**Config flag**: `--mobile`

## P

### Page Load Strategy
katana headless-engine option controlling when a page is considered loaded
(`none`, `eager`, `normal`); mapped 1:1 from crawler's `WaitStrategy`.

**Config field**: `waitStrategy`


### ProgressReporter
Interface for decoupling progress tracking from UI implementation. Methods:
`Log()`, `UpdateStats()`, `UpdateWorker()`.

**Interface**: `internal/crawlers/reporter.go`


## S

### Semaphore
Concurrency control mechanism limiting active workers to `config.Concurrency`.
Prevents resource exhaustion by capping parallel requests.


### Standard Library
Go's built-in packages (`fmt`, `net/http`, `sync/atomic`, etc.) preferred over
external dependencies unless functionality is unavailable (e.g., browser
automation).


## U

### UnifiedUI
Single UI class supporting three modes (Simple, Standard, Enhanced) based on environment variables (`CRAWLER_LEGACY_UI`, `CRAWLER_STANDARD_UI`).

**Source**: `ui/unified.go`
, [Development Guide](guides/development.md#ui-modes)

### URL Filtering
Constraint ensuring crawler stays within same domain AND base path. Prevents drifting to external sites or overwhelming subdirectories.

**Implementation**: `internal/katanaengine/translate.go` (`ScopeRegexes`)
, [Best Practices](guides/best-practices.md#scope-control)

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
Headless-engine configuration for complex load timing (waiting for network
idle or a specific load state). Non-default values select the headless engine.

**Options**: `networkidle` (default), `commit`, `load`, `domcontentloaded`

### Workers
Concurrent goroutines that execute page requests coordinated by a semaphore. Each worker gets URLs from the queue, processes them, and reports progress.

**Config field**: `concurrency` in YAML (default: 5)
, [Configuration Guide](guides/configuration.md#concurrency)

## Y

### YAML Configuration
File-based configuration (`~/.config/crawler/config.yaml`) for default settings. Overridden by CLI flags but overrides built-in defaults.

**See also**: [Configuration Guide](guides/configuration.md), [Development Guide](guides/development.md#configuration)
