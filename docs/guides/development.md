# Development Setup Guide

This guide will help you set up a local development environment for contributing to the crawler project.

## Prerequisites

### Required

- **Go 1.24.2 or higher**
  - Download from [golang.org](https://golang.org/dl/)
  - Verify installation: `go version`
  - The project uses Go 1.24.2 with toolchain go1.24.4

- **Git**
  - For cloning the repository

### Optional

- **Playwright** (for JavaScript crawling tests)
  - Auto-installs on first use when running with `--engine playwright`
  - Or manually install:
    ```bash
    go install github.com/playwright-community/playwright-go/cmd/playwright@latest
    playwright install
    ```

- **Task runner** (optional convenience)
  - Install: `go install github.com/go-task/task/v3/cmd/task@latest`
  - Provides shortcuts for common development tasks

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/crawler.git
cd crawler
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Project

```bash
go build -o crawler .
```

This creates the `crawler` binary in the current directory.

### 4. Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/config
go test ./cmd

# Run a specific test
go test ./cmd -run TestConfigLoading
```

## Development Commands

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Build to check for compilation errors
go build ./...
```

### Running the Crawler

```bash
# Basic crawl
./crawler https://example.com

# With verbose output
./crawler --verbose https://example.com

# Limit pages
./crawler --max-pages 10 https://example.com

# Test mobile emulation (requires Playwright)
./crawler --mobile https://example.com

# Force specific engine
./crawler --engine colly https://example.com
./crawler --engine playwright https://example.com
```

### Testing Quick Reference

```bash
# Test Colly engine (static sites)
./crawler --engine colly --max-pages 1 https://httpbin.org/html

# Test Playwright engine (JavaScript sites)
./crawler --engine playwright --max-pages 1 https://example.com

# Test with verbose logging (no UI)
./crawler --verbose --engine colly --max-pages 1 https://httpbin.org/html
```

## Project Structure

```
crawler/
├── api/                    # Public interfaces
│   └── crawler.go
├── cmd/                    # CLI implementation (Cobra)
│   ├── root.go
│   └── root_test.go
├── internal/
│   ├── config/            # Configuration management (Viper)
│   │   ├── config.go
│   │   └── config_test.go
│   ├── crawlers/          # Core crawling logic
│   │   ├── engines.go            # CrawlEngine interface
│   │   ├── colly_engine.go       # HTTP crawler
│   │   ├── playwright_engine.go  # Browser crawler
│   │   ├── engine_crawler.go     # Orchestration
│   │   └── factory.go            # Component factory
│   └── utils/              # Shared utilities
│       ├── url.go
│       └── url_test.go
├── ui/                     # Terminal UI (Bubbletea)
│   ├── unified.go
│   └── styles.go
├── docs/                   # Documentation
│   ├── guides/
│   ├── api/
│   └── examples/
├── go.mod
├── go.sum
└── main.go
```

## Development Workflow

### 1. Make Changes

Edit the relevant files. The project follows standard Go layout conventions.

### 2. Run Tests

```bash
go test ./...
```

### 3. Format and Vet

```bash
go fmt ./...
go vet ./...
```

### 4. Build

```bash
go build -o crawler .
```

### 5. Test Your Changes

```bash
# Quick test with verbose output
./crawler --verbose --max-pages 5 https://example.com
```

## Test Coverage

Current test coverage:
- URL utils: 53.7%
- Config: 70.9%
- CMD: 9.7%

When adding new features, please include tests. Run tests with coverage:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Configuration

The crawler supports configuration via:

1. **CLI flags** (highest priority)
2. **YAML config file** at `./crawl.yml` or `~/.config/crawler/crawl.yml`
3. **Environment variable** `$CRAWLER_CONFIG`
4. **Default values**

Example config file:

```yaml
concurrency: 5
delay: 1.0
maxPages: 100
engine: "colly"
ignorePatterns:
  - "\\.pdf$"
  - "/api/"
```

## Troubleshooting

### Build Errors

- Ensure Go 1.24.2+ is installed: `go version`
- Clean build cache: `go clean -cache`
- Re-download dependencies: `go mod download`

### Test Failures

- Run specific test with verbose output: `go test -v ./cmd -run TestName`
- Check Go version compatibility
- Ensure all dependencies are downloaded

### Playwright Issues

- First run auto-installs browsers (may take time)
- Manual install: `playwright install`
- Check system dependencies for your OS

### Import Errors

- Run `go mod tidy` to clean up dependencies
- Verify `go.mod` is up to date

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Add tests for new functionality
5. Run `go test ./...` and ensure all tests pass
6. Run `go fmt ./...` and `go vet ./...`
7. Commit with conventional commit message: `feat: add amazing feature`
8. Push to branch: `git push origin feature/amazing-feature`
9. Open a Pull Request

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for detailed guidelines.

## Additional Resources

- [README](../../README.md) - Project overview and usage
- [Paths Reference](../paths.md) - File locations
- [Tasks Guide](../tasks.md) - Common development tasks
- [Error Solutions](../errors.md) - Known issues and fixes
- [Gotchas](../gotchas.md) - Non-obvious behaviors
