# Crawler

[![Go Version](https://img.shields.io/github/go-mod/go-version/dotcommander/crawler)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/dotcommander/crawler.svg)](https://pkg.go.dev/github.com/dotcommander/crawler)
[![Go Report Card](https://goreportcard.com/badge/github.com/dotcommander/crawler)](https://goreportcard.com/report/github.com/dotcommander/crawler)

Crawler walks one website at a time: it seeds from `robots.txt` and sitemaps,
fetches only URLs under the seed's domain and base path, saves every fetched
body under `~/.config/crawler/storage/<host>/`, and can stream each page as a
JSONL or CSV record. The fetching engine is
[projectdiscovery/katana](https://github.com/projectdiscovery/katana) v1.7.0,
embedded as a Go library — plain HTTP by default, switching to headless
Chromium when you ask for mobile rendering or custom page-wait behavior.

## Contents

| Section | What you get |
|---|---|
| [Quick start](#quick-start) | Build, first crawl, where the files land |
| [Configuration and state](#configuration-and-state) | `crawl.yml` keys, presets, env vars, directories |
| [Non-goals](#non-goals) | What this tool deliberately does not do |
| [Capabilities](#capabilities) | Scope, robots, exports, extraction, resume, serve, library use |
| [Verify and contribute](#verify-and-contribute) | Build, test, lint, PRs |
| [Limits](#limits) | Known deviations and failure modes |

## Quick start

Prerequisite: Go 1.26 or later (`go.mod` pins the module to 1.26). Install
the latest tagged release (`v0.2.0`, katana engine):

```bash
go install github.com/dotcommander/crawler@latest   # -> $(go env GOPATH)/bin/crawler
crawler --max-pages 5 https://example.com
```

Or build from source (development):

```bash
git clone https://github.com/dotcommander/crawler.git
cd crawler
go build -o crawler .
./crawler --max-pages 5 https://example.com
```

Expected observable result: live progress in the terminal (a Bubbletea TUI on
a real terminal; plain log lines when stdout is a pipe, as in CI), then

```bash
ls ~/.config/crawler/storage/example.com/
# index.html  ...plus every in-scope page linked from it
```

Mechanism, in crawl order:

1. The seed URL is validated (http/https only) and becomes the crawl scope
   anchor: same host **and** same base path.
2. `https://<host>/robots.txt` is fetched once; its `Sitemap:` lines are
   followed and every sitemap URL inside scope is pre-seeded. If `robots.txt`
   is missing or unparseable, the crawl simply proceeds.
3. katana fetches pages — 5 workers, 1 s default delay, depth 3 by default —
   and each fetched body is saved as
   `~/.config/crawler/storage/<host>/<path>` (extension-less paths get
   `index.html`; query strings and fragments are dropped).
4. A finished crawl lingers up to `engineTimeoutSeconds` (default 10) waiting
   for the queue to stay empty before exiting — a small crawl still holds the
   terminal for those seconds; that floor, not a hang, is what you are seeing.

Next safe variation — pipeline mode, no UI, one JSON record per page on
stdout:

```bash
./crawler --quiet --max-pages 5 https://example.com | head -2
```

Press <kbd>q</kbd> or <kbd>Ctrl-C</kbd> to stop a crawl; workers drain for up
to 5 s before shutdown completes.

## Configuration and state

Configuration comes from CLI flags (see `crawler --help`), then a YAML file,
then built-in defaults. The file is named `crawl.yml` (see
[`crawl.yml.example`](crawl.yml.example)); discovery order: `--config` flag,
`$CRAWLER_CONFIG`, `./crawl.yml`, then the OS config dir
(`~/Library/Application Support/crawler/` on macOS, `~/.config/crawler/` on
Linux, `%APPDATA%\crawler\` on Windows).

Any key below can also be set through an environment variable with a
`CRAWLER_` prefix, e.g. `CRAWLER_CONCURRENCY=10`.

| Key | Default | Meaning |
|---|---|---|
| `depth` | `3` | Link hops to follow from the seed |
| `concurrency` | `5` | Parallel workers |
| `delay` | `1.0` | Seconds between requests |
| `maxRetries` | `2` | Retries per failed request |
| `maxPages` | `0` | Stop after N pages, best effort (0 = unlimited) |
| `mobile` | `false` | Mobile emulation (selects headless engine) |
| `userAgent` / `mobileUserAgent` | built-in | Override the sent UA |
| `headers` | `{}` | Extra HTTP headers |
| `domainDelays` | `{}` | Per-domain delay overrides, e.g. `api.github.com: 2.0` |
| `ignorePatterns` | `[]` | Regexes; matching URLs are never fetched |
| `waitStrategy` | `networkidle` | `commit`, `load`, `domcontentloaded`, `networkidle`; a non-default value selects the headless engine |
| `extraWaitTime` | `500ms` | Extra wait after load; >500 ms selects the headless engine |
| `engineTimeoutSeconds` | `10` | Minimum seconds the queue must stay empty before the crawl ends |
| `force` | `false` | Overwrite existing saved files |

`--profile` applies a preset before flags:

| Profile | Delay | Workers | Depth | Wait strategy | Use case |
|---|---|---|---|---|---|
| `fast` | 0.5 s | 10 | 2 | `domcontentloaded` | Robust sites, quick scan |
| `safe` | 2.0 s | 3 | 5 | `networkidle` | Fragile or legacy sites |
| `thorough` | 3.0 s | 2 | 10 | `networkidle` | Deep documentation crawl |

Files and directories:

| What | Default | Override |
|---|---|---|
| Crawled content | `~/.config/crawler/storage/` | `--output`, `CRAWLER_OUTPUT_DIR` |
| Resume sessions (SQLite) | `~/.config/crawler/sessions/` | `CRAWLER_SESSIONS_DIR` |
| Cache | `~/Library/Caches/crawler` (macOS), `~/.cache/crawler` (Linux) | `CRAWLER_CACHE_DIR` |
| Config file | discovery order above | `--config`, `CRAWLER_CONFIG` |

(Windows uses `%APPDATA%\crawler\` for config and sessions and
`%LOCALAPPDATA%\crawler\cache` for cache.)

## Non-goals

- **No cross-site crawling.** Links outside the seed host and base path are
  filtered out; this is a single-site tool, not a harvester.
- **No engine flag.** There is intentionally no `--engine`; the mode is
  derived deterministically from `mobile`, `waitStrategy`, and
  `extraWaitTime`.
- **No JavaScript execution in plain mode.** The default engine is HTTP-only;
  JS runs only when the headless engine is selected (which needs a local
  Chromium/Chrome for katana's headless mode).
- **Not distributed.** One process, one SQLite session per host.

## Capabilities

### Scope and politeness

Scope is the seed's host plus base path — a crawl of
`https://docs.example.com/go/` never fetches `https://docs.example.com/blog`
or `https://cdn.example.com`. Response bodies are size-capped (50 MB per page
via katana's `BodyReadSize`; `robots.txt` reads cap at 512 KB), `delay` spaces
requests, and `domainDelays` raises the delay for specific hosts. Depth
defaults to 3; `--max-pages`/`-p` stops a crawl early on a best-effort basis
(in-flight requests may overshoot slightly).

### Robots and sitemap seeding

On by default: `robots.txt` is fetched at the host root, disallowed paths are
excluded, and `Sitemap:` entries are parsed and seeded (URLs outside scope are
filtered). `--no-robots` skips all of it. Compliance is enforced at both
discovery and result level — a link that becomes disallowed mid-crawl is still
dropped.

### Exports and extraction

`--format jsonl|csv|sitemap` with `--export-file` (sitemap **requires**
`--export-file`; otherwise it fails with `Error: failed to set up exporter:
sitemap format requires --export-file`). Without `--export-file`, records go
to stdout. `--quiet` implies `--format jsonl` and silences the UI, for
pipelines.

JSONL record keys (runtime-verified):
`url`, `title`, `status_code`, `content_type`, `links_found`, `crawled_at`,
plus `extracted` when `--extract` is used. CSV uses the same columns as its
header. `--extract` takes CSS selectors as `key=selector,...`:

```bash
./crawler --quiet --format csv --extract "title=h1,desc=.summary" \
  --export-file pages.csv https://example.com
```

### JavaScript endpoint mining

`--jc` scans inline and external `<script>` sources of fetched pages for
URL-like endpoints; discovered endpoints inside scope are fetched like any
other page (verified: an API path referenced only in JavaScript is reached
with `--jc` and not without it).

### Sessions and resume

Each crawl records visited URLs in a SQLite database under the sessions dir
(keyed by host). `--resume` continues a previous crawl of the same seed:
already-completed pages are re-fetched (the engine queue is not persisted)
but never re-exported or re-saved — a page is processed at most once across
runs.

### Serving captured content

```bash
crawler serve [directory]        # browse captured content
crawler serve --port 9000 .      # default: localhost:8080
```

Serves stored files for browsing (directory listing + file contents); path
traversal out of the served directory is blocked.

### Engine modes

| Options | Engine | Why |
|---|---|---|
| (default) | katana standard | Fast plain HTTP; no JavaScript |
| `--mobile` | katana headless | Device emulation needs a browser |
| `waitStrategy` ≠ `networkidle` | katana headless | Custom load timing |
| `extraWaitTime` > 500 ms | katana headless | Indicates a JS-heavy page |

### Library use

The module path is `github.com/dotcommander/crawler`; `api.Crawler` is the
public interface and `crawlers.CreateCrawler(cfg, verbose, store)` the
factory. See [`docs/api/crawler.md`](docs/api/crawler.md) for a worked
example.

## Verify and contribute

```bash
go build ./...            # build everything
go test ./...             # full test suite
go vet ./...              # vet
golangci-lint run ./...   # lint (config in .golangci.yml)
go test ./cmd/ -run TestE2E   # end-to-end tests (local HTTP servers, ~15 s)
```

A [`Taskfile.yml`](Taskfile.yml) wraps the common ones (`task build`,
`task test`, `task install`).

Pull requests welcome — see [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md).
Run the test suite and linter before submitting, and use Conventional
Commits-style messages. Vulnerability reports go to
[`docs/SECURITY.md`](docs/SECURITY.md); the code of conduct is
[`docs/CODE_OF_CONDUCT.md`](docs/CODE_OF_CONDUCT.md).

## Limits

- **`links_found` is always `0`** in exported records — a known deviation of
  the katana adapter, kept for schema compatibility.
- **`--max-pages` is best effort.** Enforcement happens when results arrive,
  so up to `concurrency`−1 in-flight fetches can land beyond the limit.
- **Resume re-traverses the network.** Completed pages are deduplicated from
  output, but their bytes are fetched again; pages discovered but not yet
  fetched when a run stopped are reached by re-walking from the seed.
- **Every crawl pays the `engineTimeoutSeconds` tail** (default 10 s) before
  the process exits, even for a 1-page crawl. Lower it in `crawl.yml` for
  short scripted runs.
- **Headless mode needs a Chromium/Chrome** available to katana's headless
  engine; without one, `--mobile` and custom wait strategies cannot run.

## License

MIT — see [LICENSE](LICENSE).
