# Contributing to Crawler

Thank you for considering contributing to the crawler project! This document provides guidelines and workflows for submitting quality pull requests.

## Table of Contents

- [Code Style](#code-style)
- [Commit Message Format](#commit-message-format)
- [Development Workflow](#development-workflow)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Review Expectations](#review-expectations)

## Code Style

This project follows standard Go conventions and community best practices.

### Formatting

All code MUST be formatted with `go fmt`:

```bash
go fmt ./...
```

This is enforced in CI. Commits with unformatted code will not be accepted.

### Linting

All code MUST pass `go vet`:

```bash
go vet ./...
```

`go vet` catches common mistakes like:
- Unreachable code
- Suspicious constructs
- Printf format string errors
- Incorrect struct tags

### Naming Conventions

- **Package names**: Lowercase, single word, no underscores
- **Exported identifiers**: PascalCase (e.g., `CrawlEngine`, `StartCrawling`)
- **Unexported identifiers**: camelCase (e.g., `localVariable`, `internalFunc`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Interfaces**: Should be descriptive names ending in `-er` when possible (e.g., `Crawler`, `Reporter`)

### File Organization

- One package per directory
- Keep files focused on a single responsibility
- Package documentation at the top of each package
- Exported functions should have godoc comments

### Error Handling

- Always handle errors, never ignore them
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return early on errors to reduce nesting
- Use sentinel errors for expected error types

```go
// Good
result, err := process()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Bad
result, _ := process()
```

### Comments

- Exported functions MUST have godoc comments
- Comments should be complete sentences
- Comments should explain **why**, not **what**
- Keep comments near the code they describe

```go
// Crawl initiates the crawling process for the given URL.
// It returns the number of pages crawled and any error encountered.
func Crawl(url string) (int, error) {
    // ...
}
```

## Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) format:

```
type(scope): description

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring without functional changes
- `docs`: Documentation changes
- `test`: Test additions or modifications
- `chore`: Maintenance tasks, dependencies, tooling
- `perf`: Performance improvements
- `style`: Code style changes (formatting, etc.)

### Scopes

Common scopes include:
- `crawler`: Core crawling logic
- `ui`: User interface components
- `config`: Configuration system
- `engine`: Crawler engines (colly, playwright)
- `cmd`: CLI commands
- `docs`: Documentation

### Examples

```bash
feat(crawler): add support for custom wait strategies
fix(ui): correct progress bar update race condition
refactor(config): consolidate YAML and CLI flag handling
docs(contributing): add code style guidelines
test(engine): add table-driven tests for URL validation
chore(deps): bump colly to v2.0
```

### Guidelines

- Use lowercase for type and scope
- Keep description under 72 characters
- Use imperative mood ("add" not "added" or "adds")
- Limit body lines to 72 characters
- Reference issues in footer: `Closes #123`

## Development Workflow

### 1. Set Up Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/crawler.git
cd crawler

# Add upstream remote
git remote add upstream https://github.com/ORIGINAL_OWNER/crawler.git

# Install dependencies
go mod download

# Run tests to verify setup
go test ./...
```

### 2. Create a Feature Branch

```bash
# Update main from upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 3. Make Changes

- Write clean, idiomatic Go code
- Add tests for new functionality
- Update documentation as needed
- Run formatters and linters frequently

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run tests
go test ./...

# Build to verify compilation
go build ./...
```

### 4. Commit Your Changes

```bash
git add .
git commit -m "feat(scope): description of changes"
```

### 5. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 6. Create Pull Request

- Go to the repository on GitHub
- Click "Compare & pull request"
- Fill in the PR template
- Link related issues
- Request review from maintainers

## Testing Requirements

### Test Coverage

- New features MUST include tests
- Bug fixes SHOULD include regression tests
- Aim for >70% coverage for new code
- Critical paths (config, URL handling) should have >80% coverage

### Test Structure

Use table-driven tests for multiple cases:

```go
func TestIsValidURL(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        want    bool
    }{
        {"valid HTTP", "http://example.com", true},
        {"valid HTTPS", "https://example.com", true},
        {"invalid scheme", "ftp://example.com", false},
        {"empty string", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := IsValidURL(tt.url); got != tt.want {
                t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/config

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...

# Run with race detector
go test -race ./...

# Run verbose output
go test -v ./...
```

### Test Naming

- Test functions: `Test<FunctionName>`
- Benchmark functions: `Benchmark<FunctionName>`
- Example functions: `Example<FunctionName>`

```go
func TestCrawlEngine_Start(t *testing.T) { /* ... */ }
func BenchmarkCrawlEngine_Start(b *testing.B) { /* ... */ }
func ExampleCrawlEngine_Start() { /* ... */ }
```

## Pull Request Process

### Before Submitting

- [ ] Code follows style guidelines (`go fmt`, `go vet`)
- [ ] All tests pass (`go test ./...`)
- [ ] New features have tests
- [ ] Documentation is updated
- [ ] Commit messages follow format
- [ ] Branch is up to date with main

### PR Description Template

```markdown
## Summary
Brief description of changes (2-3 bullet points)

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings generated
```

### PR Title Format

PR titles should follow commit message format:

```
feat(scope): brief description
fix(scope): brief description
docs(scope): brief description
```

### Merge Requirements

- At least one maintainer approval
- All CI checks pass
- No merge conflicts
- Discussion resolved (if any)

## Review Expectations

### For Contributors

- Be responsive to review feedback
- Address all reviewer comments
- Push new commits to the same branch (not force push unless asked)
- Mark conversations as resolved when addressed
- Ask questions if feedback is unclear

### For Reviewers

- Provide constructive, specific feedback
- Explain **why** a change is suggested
- Approve when only minor/nitpick issues remain
- Request changes when substantial work is needed
- Test the changes locally if possible

### Review Categories

**Must Fix** (block merge):
- Bugs or incorrect behavior
- Breaking changes without documentation
- Security vulnerabilities
- Test failures
- Style violations

**Should Fix** (strongly encouraged):
- Performance concerns
- Error handling improvements
- Missing edge cases
- Documentation gaps

**Nice to Have** (optional):
- Variable naming
- Code organization suggestions
- Minor refactoring opportunities
- Additional test coverage

### Timeline

- Maintainers aim to review within 3-5 business days
- Contributors should respond within 7 days to feedback
- PRs inactive for 30+ days may be closed

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open an issue with reproduction steps
- **Features**: Open an issue with the `enhancement` label
- **Security**: Email maintainers privately

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](../LICENSE).
