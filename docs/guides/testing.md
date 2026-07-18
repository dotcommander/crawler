# Testing Guide

This guide explains the testing conventions and patterns used in the crawler project.

## Table of Contents

- [Test File Conventions](#test-file-conventions)
- [Table-Driven Tests](#table-driven-tests)
- [Test Coverage](#test-coverage)
- [Common Testing Patterns](#common-testing-patterns)
- [Running Tests](#running-tests)

## Test File Conventions

### File Naming

Test files follow the standard Go convention: `<package>_test.go`

Examples:
- `cmd/root_test.go` - Tests for the `cmd` package
- `internal/utils/url_test.go` - Tests for URL utilities
- `internal/config/config_test.go` - Tests for configuration management

### File Location

Test files are placed alongside the code they test:

```
cmd/
├── root.go
└── root_test.go

internal/utils/
├── url.go
└── url_test.go
```

### Test Naming

- Test functions: `Test<FunctionName>` for unit tests
- Subtests: Use `t.Run()` with descriptive names
- Benchmark functions: `Benchmark<FunctionName>`

## Table-Driven Tests

Table-driven tests are the preferred pattern for testing multiple scenarios. They provide:

- **Clear test cases** - Each case is explicitly defined
- **Easy extension** - Add new cases by adding struct entries
- **Isolated failures** - One failing case doesn't block others
- **Readable output** - Descriptive names for each scenario

### Basic Pattern

```go
func TestNormalizeURL(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "basic URL",
            input:    "https://example.com/path",
            expected: "https://example.com/path",
        },
        {
            name:     "URL with fragment",
            input:    "https://example.com/path#section",
            expected: "https://example.com/path",
        },
        {
            name:     "URL with query params",
            input:    "https://example.com/path?foo=bar",
            expected: "https://example.com/path",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := NormalizeURL(tt.input)
            if result != tt.expected {
                t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### Error Testing Pattern

For functions that return errors, include error validation:

```go
func TestValidateURL(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        shouldErr bool
        errMsg    string
    }{
        {
            name:      "valid HTTPS URL",
            input:     "https://example.com/path",
            shouldErr: false,
        },
        {
            name:      "invalid scheme",
            input:     "ftp://example.com/path",
            shouldErr: true,
            errMsg:    "only HTTP and HTTPS schemes allowed",
        },
        {
            name:      "empty URL",
            input:     "",
            shouldErr: true,
            errMsg:    "URL cannot be empty",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateURL(tt.input)

            if tt.shouldErr {
                if err == nil {
                    t.Errorf("ValidateURL(%q) expected error, got nil", tt.input)
                }
                if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("ValidateURL(%q) error = %q, want containing %q", tt.input, err.Error(), tt.errMsg)
                }
            } else {
                if err != nil {
                    t.Errorf("ValidateURL(%q) unexpected error: %v", tt.input, err)
                }
            }
        })
    }
}
```

### Testing with Setup/Teardown

For tests requiring setup or cleanup:

```go
func TestConfigLoading(t *testing.T) {
    tests := []struct {
        name    string
        config  string
        checkFn func(*CrawlerConfig) error
    }{
        {
            name: "fast profile",
            config: `
concurrency: 10
max-depth: 2
`,
            checkFn: func(c *CrawlerConfig) error {
                if c.Concurrency != 10 {
                    return fmt.Errorf("expected Concurrency=10, got %d", c.Concurrency)
                }
                return nil
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup: Create temporary directory and config file
            tempDir := t.TempDir()
            configPath := filepath.Join(tempDir, "config.yml")
            if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
                t.Fatalf("failed to write config: %v", err)
            }

            // Execute: Load and validate config
            cfg, err := LoadConfig(configPath)
            if err != nil {
                t.Fatalf("LoadConfig failed: %v", err)
            }

            // Verify: Run validation function
            if err := tt.checkFn(cfg); err != nil {
                t.Error(err)
            }
        })
    }
}
```

**Key points:**
- Use `t.TempDir()` for temporary files (automatically cleaned up)
- Use `t.Fatalf()` for setup failures (immediate stop)
- Use `t.Errorf()` for test failures (continue testing other cases)

## Test Coverage

### Current Coverage Status

As of the latest test run:

| Package | Coverage | Status |
|---------|----------|--------|
| `internal/config` | 70.9% | ✅ Good |
| `internal/utils` | 53.7% | ⚠️ Moderate |
| `cmd` | 9.7% | ⚠️ Low |
| `internal/crawlers` | 0.0% | ❌ None |
| `ui` | 0.0% | ❌ None |

### Coverage Goals

- **Priority 1**: Core logic (`internal/crawlers`) - Target: 60%+
- **Priority 2**: UI components (`ui`) - Target: 40%+ (challenging for Bubbletea)
- **Priority 3**: CLI commands (`cmd`) - Target: 30%+

### Measuring Coverage

```bash
# Coverage for all packages
go test ./... -cover

# Detailed coverage by package
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Common Testing Patterns

### CLI Testing

Test CLI commands by setting arguments and executing:

```go
func TestRootCommand(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantErr  bool
        setupFn  func() string
        cleanup  func(string)
    }{
        {
            name: "valid URL with config",
            args: []string{"--config", "test.yml", "https://example.com"},
            wantErr: false,
            setupFn: func() string {
                // Create test config
                tempDir := t.TempDir()
                configPath := filepath.Join(tempDir, "test.yml")
                // ... write config
                return configPath
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupFn != nil {
                configPath := tt.setupFn()
                tt.args[1] = configPath
            }

            rootCmd.SetArgs(tt.args)
            err := rootCmd.Execute()

            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### HTTP Client Mocking

For testing HTTP interactions without making real calls:

```go
func TestCrawlerWithMockServer(t *testing.T) {
    // Start test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`<html><body><a href="/page2">Link</a></body></html>`))
    }))
    defer server.Close()

    // Test with server URL
    crawler := NewCrawler(server.URL)
    err := crawler.Start()

    if err != nil {
        t.Errorf("Crawler.Start() error = %v", err)
    }
}
```

### Benchmark Tests

For performance-critical code:

```go
func BenchmarkNormalizeURL(b *testing.B) {
    url := "https://example.com/path?query=1#fragment"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        NormalizeURL(url)
    }
}
```

Run benchmarks with:
```bash
go test -bench=. -benchmem
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./internal/utils

# Run specific test
go test ./internal/utils -run TestNormalizeURL

# Run with verbose output
go test ./... -v

# Run with coverage
go test ./... -cover

# Race detection
go test ./... -race
```

### Test Flags

- `-v` - Verbose output (shows all test names)
- `-run <regex>` - Run tests matching pattern
- `-cover` - Show coverage percentage
- `-race` - Detect race conditions
- `-short` - Skip long-running tests

### Continuous Integration

Tests run automatically on:
- Every pull request
- Main branch commits

Ensure all tests pass before submitting PRs:

```bash
# Full CI test suite
go test ./... -race -coverprofile=coverage.out
```

## Best Practices

1. **Write tests first** (TDD) when possible
2. **One assertion per test case** - Makes failures clearer
3. **Use table-driven tests** for multiple scenarios
4. **Test edge cases** - Empty inputs, nil values, boundaries
5. **Keep tests fast** - Use `t.Short()` to skip slow tests in CI
6. **Avoid sleep** - Use channels or sync primitives for timing
7. **Clean up resources** - Use `defer` and `t.TempDir()`
8. **Descriptive names** - Test names should explain what is being tested

## Resources

- [Go Testing Package](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify Assertions](https://github.com/stretchr/testify) (used in docs tests)
