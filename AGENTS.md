# Project Assistant Context

Read and apply `/Users/vampire/.agents/AGENTS.md` completely, then
`/Users/vampire/code/go/src/AGENTS.md`, before this file. This file contains
repository-specific guidance only.

## Quick Start
**What**: Single-site web crawler
**Type**: CLI tool (+ tiny `serve` subcommand)
**Stack**: Go 1.26, embedded [katana](https://github.com/projectdiscovery/katana) v1.7.0 engine, Kong CLI, Viper config, Bubbletea TUI
**Run**: `go build -o crawler . && ./crawler --max-pages 5 <url>`
**Test**: `go test ./...` (e2e subset: `go test ./cmd/ -run TestE2E`)
**Install**: `go install .`

## Navigation Index

- `docs/paths.md`, `docs/tasks.md`, `docs/errors.md`, `docs/gotchas.md` — where things live, common tasks, known issues
- `docs/guides/README-go.md` — full feature list and usage
- README.md — current operating model (quick start, config keys, limits)
- `.work/katana-migration-spec.md` — why the engine layer looks the way it does (upstream race workaround, KnownFiles rationale, resume semantics)

## Critical Context

- **Engine**: `internal/katanaengine` translates `CrawlerConfig` → katana
  `Options`; `internal/crawlers/katana_crawler.go` orchestrates. There is
  **no `--engine` flag**: standard (HTTP) vs headless (Chromium) mode is
  derived from `mobile` / `waitStrategy` / `extraWaitTime`. The former
  colly/rod/playwright engines and `engine_crawler.go` orchestration are
  gone — do not resurrect them.
- **katana is used as a library**, not as the CLI binary. Upstream's
  gologger writes a global log level while engines read it (data race);
  `internal/katanaengine` serializes library use with a process-wide lock.
  katana's `KnownFiles` (robots/sitemap discovery) is deliberately disabled
  — its parsers fetch at the *seed path* (RFC 9309 violation) and cause
  duplicate fetches; seeding/compliance are crawler-owned (`internal/seeders`).
- **Scope**: seed host **and** base path; robots.txt honoured by default
  (`--no-robots` to opt out); sitemap URLs from robots `Sitemap:` lines are
  pre-seeded.
- **Resume**: visited URLs persist to SQLite under `~/.config/crawler/sessions/`.
  Resume re-traverses from the seed (no persisted frontier) but deduplicates
  at result level — a page is exported/saved at most once across runs.
- **Config**: Viper keys in `crawl.yml` (`ignorePatterns`, not
  `excludePatterns`; `maxPages`, `extraWaitTime` as duration string,
  `engineTimeoutSeconds` floors every crawl's exit by ~10 s). Env prefix
  `CRAWLER_`. Defaults in `internal/config/viper_config.go`.
- **Known deviation**: `links_found` in exported records is always `0`
  (katana adapter; kept for schema compatibility).
- Default storage: `~/.config/crawler/storage/<host>/` (not `./tmp/`).
- Press <kbd>q</kbd> or <kbd>Ctrl-C</kbd> to stop (5 s graceful drain).

## Verification

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run ./...
```
