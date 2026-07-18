# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-07-18

### Changed
- Public-release hygiene: squashed history to a single commit; removed
  internal/cross-repo markdown, legacy migration scripts, and private-use-case
  Taskfile targets and example domains.
- `go.mod`: retracted `v0.1.0` (its published module carried internal
  scaffolding). `@latest` now resolves to `v0.1.1`.

## [0.1.0] - 2026-07-18

First public release.

### Added
- Dual-engine crawler: Colly (HTTP) by default; Rod (browser) auto-selected for
  mobile, custom wait strategies, and JS-heavy pages. A Playwright engine is
  also included.
- Scope-safe crawling within the seed domain and base path, with `robots.txt`
  and sitemap discovery (`--no-robots` to opt out).
- Per-domain rate limiting and a per-domain circuit breaker; semaphore-bounded
  workers; bounded response bodies.
- Resumable sessions persisted to SQLite (`--resume`).
- Structured export: `--format jsonl|csv|sitemap` with `--extract` CSS selectors.
- JavaScript API-endpoint mining (`--jc`) from inline and external scripts.
- Page `<title>` extraction across all engines.
- Three UI modes: enhanced Bubbletea TUI, `--verbose` plain logs, and `--quiet`
  JSONL pipeline mode.
- `--version` flag (overridable at build time via ldflags).
- `serve` subcommand to browse captured content.

### Fixed
- Race on the Colly engine's per-domain collector map (could panic under
  concurrent multi-domain crawling).
- Idle-detection race that could drop the last dequeued pages of a crawl.
- Shutdown now drains workers before closing the engine and page pool,
  preventing operations on a closing page and writes to a closed exporter.
- Discovered links that overflow the bounded queue are now counted and logged
  instead of dropped silently.
- Base-path scope is derived from the URL directory, so a dotted seed segment
  such as `/v1.2` no longer collapses the crawl scope.
- Exclude-pattern globs (e.g. `*.pdf`) now match nested paths.

### Known limitations
- Extreme bursts beyond `concurrency * 100` may still overflow the queue
  (counted and logged, not retried).
- Browser-engine page operations do not abort the instant the context is
  cancelled; graceful shutdown waits up to ~10s for in-flight pages.

[Unreleased]: https://github.com/dotcommander/crawler/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/dotcommander/crawler/releases/tag/v0.1.1
[0.1.0]: https://github.com/dotcommander/crawler/releases/tag/v0.1.0
