# Crawler

[![Go Version](https://img.shields.io/github/go-mod/go-version/dotcommander/crawler)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/dotcommander/crawler.svg)](https://pkg.go.dev/github.com/dotcommander/crawler)
[![Go Report Card](https://goreportcard.com/badge/github.com/dotcommander/crawler)](https://goreportcard.com/report/github.com/dotcommander/crawler)

A high-performance web crawler in Go with a dual-engine architecture that picks
the fastest method that can render the page: plain HTTP ([Colly](https://github.com/gocolly/colly))
for static content, and a real browser ([Rod](https://github.com/go-rod/rod)) for
JavaScript-heavy or mobile-emulated sites.

## Features

- **Dual engine, auto-selected** — Colly (HTTP) by default; Rod (browser) when you
  request `--mobile`, a custom `waitStrategy`, or an `extraWaitTime > 500ms`.
- **Scope-safe crawling** — stays within the seed's domain and base path; honours
  `robots.txt` and sitemap discovery by default (`--no-robots` to opt out).
- **Polite by construction** — per-domain rate limiting, a per-domain circuit
  breaker, semaphore-bounded workers, and bounded response bodies.
- **Resumable** — visited-URL state persists to SQLite under
  `~/.config/crawler/sessions/`; re-run with `--resume` to continue.
- **Structured export** — `--format jsonl|csv|sitemap` with `--extract` CSS
  selectors (e.g. `title=h1,desc=.summary`) emitted as columns/fields.
- **JavaScript endpoint mining** — `--jc` extracts API endpoints from inline and
  external `<script>` sources.
- **Resilient shutdown** — cancellation and signals drain workers before the
  engine is torn down; press <kbd>q</kbd> (TUI) or <kbd>Ctrl-C</kbd> to stop.
- **Three UI modes** — enhanced Bubbletea TUI by default; `--verbose` for plain
  log output (CI/pipes); `--quiet` for JSONL-to-stdout pipeline mode.

## Requirements

- Go 1.25 or later.
- For browser (Rod) crawls: a Chromium is launched on demand (headless).

## Install

```bash
go install github.com/dotcommander/crawler@latest
```

Or build from source:

```bash
git clone https://github.com/dotcommander/crawler.git
cd crawler
go build -o crawler .
```

## Quick start

```bash
# Crawl with smart defaults (Colly, in-scope)
crawler https://example.com

# Cap a large site
crawler --max-pages 50 https://docs.example.com

# Mobile rendering (auto-selects the Rod browser engine)
crawler --mobile https://m.example.com

# Pipeline mode: JSONL records to stdout, no UI
crawler --quiet --format jsonl https://example.com > pages.jsonl

# Resume an interrupted crawl
crawler --resume https://example.com
```

## Command reference

```
crawler [options] <URL>
crawler serve [directory]   # browse captured content
```

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Directory to save crawled content |
| `--max-pages` | `-p` | Stop after N pages (`0` = unlimited) |
| `--mobile` | `-m` | Crawl as a mobile device (selects Rod) |
| `--config` | `-c` | Path to a YAML config file |
| `--profile` | | Preset: `fast`, `safe`, or `thorough` |
| `--url-list` | | File of newline-delimited seed URLs |
| `--format` | | Export format: `jsonl`, `csv`, `sitemap` |
| `--export-file` | | Export output file (default: stdout) |
| `--extract` | | CSS selectors as `key=selector,...` |
| `--jc` | | Extract API endpoints from JavaScript |
| `--resume` | | Resume from the persisted session |
| `--no-robots` | | Skip robots.txt / sitemap seeding |
| `--verbose` | `-v` | Plain progress logging (no TUI) |
| `--quiet` | `-q` | Pipeline mode: JSONL to stdout, no UI |
| `--version` | | Print version and exit |

There is intentionally no `--engine` flag: the engine is auto-selected from the
options above. Engine choice is deterministic — see *Architecture*.

## Configuration

The crawler reads `crawl.yml` (see [`crawl.yml.example`](crawl.yml.example) for
the full reference), searching in order: `$CRAWLER_CONFIG`, `./crawl.yml`, then
the OS config dir (`~/Library/Application Support/crawler/` on macOS,
`~/.config/crawler/` on Linux, `%APPDATA%\crawler\` on Windows). CLI flags
override config values.

Common fields: `depth`, `concurrency` (default 5), `delay`, `maxRetries`,
`mobile`, `extraWaitTime`, `waitStrategy`, `domainDelays` (per-domain rate
limits), `excludePatterns`, `headers`, and `userAgent`.

### Presets

`--profile` selects a tuned preset:

| Profile | Workers | Delay | Depth | Use case |
|---------|---------|-------|-------|----------|
| `fast` | 10 | 0.5s | 2 | Robust sites, quick scan |
| `safe` | 2 | 2.0s | 3 | Fragile or legacy sites |
| `thorough` | 5 | 1.0s | 5 | Deep documentation crawl |

### Files and environment

| What | Default | Override |
|------|---------|----------|
| Crawled content | `~/.config/crawler/storage/<host>/` | `--output`, `CRAWLER_OUTPUT_DIR` |
| Cache | OS cache dir (e.g. `~/Library/Caches/crawler`, `~/.cache/crawler`) | `CRAWLER_CACHE_DIR` |
| Config file | `crawl.yml` (search order above) | `CRAWLER_CONFIG` |
| Resume sessions | `~/.config/crawler/sessions/` | `CRAWLER_SESSIONS_DIR` |

If the TUI renders garbled (SSH, CI, piped output), use `--verbose` for plain
log output or `--quiet` for JSONL.

## Architecture

### Engine selection

| Option | Engine | Why |
|--------|--------|-----|
| (default) | **Colly** | Fast HTTP; no JavaScript |
| `--mobile` | **Rod** | Device emulation needs a browser |
| `waitStrategy` ≠ `networkidle` | **Rod** | Custom load timing |
| `extraWaitTime > 500ms` | **Rod** | Indicates a JS-heavy page |

A [Playwright](https://github.com/playwright-community/playwright-go) engine is
also implemented in `internal/crawlers/`; the auto-selector uses Colly or Rod.

### Layout

```
api/                     Public Crawler interface
cmd/                     Kong CLI (root, serve) and config helpers
internal/
  config/                Viper-based config + defaults
  crawlers/              Engines, orchestration, page pool, rate limiter,
                         circuit breaker, JS/HTML extractors
  exporters/             jsonl / csv / sitemap writers
  seeders/               robots.txt + sitemap discovery
  session/               Visited store (SQLite / memory) for resume
  utils/                 URL normalization, validation, path safety
ui/                      Bubbletea TUI (enhanced / standard / simple)
```

### Safety and resilience

- **Scope** — links are followed only within the seed domain and base path; URLs
  are validated (scheme, traversal, encoded traversal).
- **Backpressure** — discovered links that overflow the bounded queue are counted
  and logged rather than dropped silently.
- **Bounded I/O** — HTTP/JS/PDF/sitemap bodies are read through `io.LimitReader`.
- **Shutdown** — `Cancel`/signals drain workers (bounded wait) before the engine
  and visited store are closed.

## Development

```bash
go build ./...          # build
go test ./...           # tests
go vet ./...            # vet
golangci-lint run ./... # lint (config in .golangci.yml)
```

A [`Taskfile.yml`](Taskfile.yml) wraps common tasks (`task build`, `task test`,
`task install`). Run `task --list` to see them all.

### Known limitations

- Under extreme bursts of discovered links (far exceeding `concurrency * 100`),
  overflow links are counted and logged but not retried.
- Browser-engine page operations do not abort the instant the context is
  cancelled; graceful shutdown waits up to ~10s for in-flight pages.

## Contributing

Pull requests welcome. See [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md). Please
run `go test ./...` and `golangci-lint run` before submitting, and use
[Conventional Commits](https://www.conventionalcommits.org/) messages.

## Security

Found a vulnerability? See [`docs/SECURITY.md`](docs/SECURITY.md) for private
reporting. Community standards are in
[`docs/CODE_OF_CONDUCT.md`](docs/CODE_OF_CONDUCT.md).

## License

MIT — see [LICENSE](LICENSE).
