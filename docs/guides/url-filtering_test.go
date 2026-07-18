package guides_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestURLFilteringGuide_FileExists verifies the URL filtering guide file exists and is readable
func TestURLFilteringGuide_FileExists(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "url-filtering.md should exist and be readable")
	assert.Greater(t, len(content), 0, "File should not be empty")
}

// TestURLFilteringGuide_ExcludePatternsDocumented verifies ExcludePatterns usage is documented with examples
func TestURLFilteringGuide_ExcludePatternsDocumented(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "ExcludePatterns section header",
			mustExist: []string{
				"### 4. Exclude Patterns",
				"Configure **custom pattern matching** to filter specific URLs",
			},
			reason: "Should have a clear section header explaining ExcludePatterns purpose",
		},
		{
			name: "ExcludePatterns implementation reference",
			mustExist: []string{
				"Location: `internal/utils/url.go:251-264`",
				"MatchesExcludePatterns",
			},
			reason: "Should reference the implementation location",
		},
		{
			name: "ExcludePatterns code examples",
			mustExist: []string{
				"```go",
				"func MatchesExcludePatterns(urlStr string, patterns []string) bool",
				"path.Match(pattern, urlStr)",
				"strings.Contains(urlStr, pattern)",
			},
			reason: "Should include Go code implementation example",
		},
		{
			name: "YAML configuration examples",
			mustExist: []string{
				"```yaml",
				"excludePatterns:",
				"ignorePatterns:",
			},
			reason: "Should show YAML configuration with both old and new pattern names",
		},
		{
			name: "Pattern types explanation",
			mustExist: []string{
				"**Glob Patterns**",
				"**Substring Matching**",
				"`*.pdf`",
				"`*/admin/*`",
			},
			reason: "Should explain glob and substring matching patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_SameDomainBehavior verifies same-domain filtering is documented with examples
func TestURLFilteringGuide_SameDomainBehavior(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Domain restriction section",
			mustExist: []string{
				"### 1. Domain Restriction",
				"**only follows links on the same domain**",
			},
			reason: "Should have clear section header explaining same-domain behavior",
		},
		{
			name: "Implementation reference",
			mustExist: []string{
				"Location: `internal/crawlers/engine_crawler.go:356-359`",
				"u.Host != startURL.Host",
			},
			reason: "Should reference the domain checking implementation",
		},
		{
			name: "Code example with URL.Host comparison",
			mustExist: []string{
				"```go",
				"if u.Host != \"\" && u.Host != startURL.Host {",
				"return true",
			},
			reason: "Should show Go code for domain validation",
		},
		{
			name: "Practical examples with crawled vs skipped",
			mustExist: []string{
				"# ✅ Crawled:",
				"# ❌ Skipped:",
				"https://other-site.com/page.html",
				"https://api.example.com/endpoint",
			},
			reason: "Should provide concrete examples of what gets crawled vs skipped",
		},
		{
			name: "Subdomain explanation",
			mustExist: []string{
				"**Note**: Subdomains are considered different domains",
				"`www.example.com` ≠ `example.com`",
			},
			reason: "Should explain subdomain handling clearly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_BasePathBehavior verifies base-path filtering is documented with examples
func TestURLFilteringGuide_BasePathBehavior(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Path restriction section",
			mustExist: []string{
				"### 2. Path Restriction",
				"**stays within the base path**",
			},
			reason: "Should have clear section header for path restriction",
		},
		{
			name: "Implementation reference",
			mustExist: []string{
				"Location: `internal/utils/url.go:141-164`",
				"IsWithinBasePath",
				"path.Dir(startURL.Path)",
				"strings.HasPrefix(u.Path, basePath)",
			},
			reason: "Should reference the path validation implementation",
		},
		{
			name: "Code example",
			mustExist: []string{
				"```go",
				"func IsWithinBasePath(u, startURL *url.URL) bool",
				"basePath := path.Dir(startURL.Path)",
			},
			reason: "Should show Go code for base path validation",
		},
		{
			name: "Practical examples",
			mustExist: []string{
				"# Base path: /docs/",
				"✅ Crawled: https://example.com/docs/api.html",
				"❌ Skipped: https://example.com/blog/post",
			},
			reason: "Should show concrete examples of base path behavior",
		},
		{
			name: "Edge cases table",
			mustExist: []string{
				"#### Edge Cases",
				"| Start URL | Base Path | Crawled | Skipped |",
				"`https://example.com/`",
				"`https://example.com/docs/`",
				"`https://example.com/docs/api.html`",
			},
			reason: "Should provide a table with edge cases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_CommonPatterns verifies common regex/glob patterns are documented
func TestURLFilteringGuide_CommonPatterns(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Common patterns section",
			mustExist: []string{
				"## Common Patterns",
			},
			reason: "Should have a dedicated section for common patterns",
		},
		{
			name: "File extension patterns",
			mustExist: []string{
				"### File Extensions",
				"`*.pdf`",
				"`*.zip`",
				"`*.jpg`",
				"`*.png`",
				"`*.mp4`",
			},
			reason: "Should document file extension patterns",
		},
		{
			name: "Administrative area patterns",
			mustExist: []string{
				"### Administrative Areas",
				"`*/admin/*`",
				"`*/backend/*`",
				"`*/wp-admin/*`",
				"`*/user/*`",
			},
			reason: "Should document admin area patterns",
		},
		{
			name: "API endpoint patterns",
			mustExist: []string{
				"### API Endpoints",
				"`/api/`",
				"`/api/v1/*`",
				"`/graphql`",
				"`*.json`",
			},
			reason: "Should document API endpoint patterns",
		},
		{
			name: "Authentication patterns",
			mustExist: []string{
				"### Authentication and Sessions",
				"`login`",
				"`logout`",
				"`register`",
				"`password`",
				"`/auth/*`",
			},
			reason: "Should document authentication-related patterns",
		},
		{
			name: "Search and filter patterns",
			mustExist: []string{
				"### Search and Filter URLs",
				"`/search*`",
				"`?query=`",
				"`?filter=`",
				"`?sort=`",
			},
			reason: "Should document search/filter patterns",
		},
		{
			name: "Development environment patterns",
			mustExist: []string{
				"### Development and Testing",
				"`/dev/*`",
				"`/staging/*`",
				"`/test/*`",
				"`/debug/*`",
			},
			reason: "Should document development environment patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_PatternSyntax verifies pattern syntax is explained
func TestURLFilteringGuide_PatternSyntax(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Glob pattern explanation",
			mustExist: []string{
				"**Glob Patterns**",
				"via `path.Match`",
				"`*.pdf` - Matches any URL ending with `.pdf`",
				"`*/admin/*` - Matches URLs with `/admin/` in the path",
				"`???.jpg` - Matches 3-character filenames",
			},
			reason: "Should explain glob pattern syntax with examples",
		},
		{
			name: "Substring matching explanation",
			mustExist: []string{
				"**Substring Matching**",
				"via `strings.Contains`",
				"`admin` - Matches any URL containing \"admin\"",
				"`logout` - Matches any URL containing \"logout\"",
			},
			reason: "Should explain substring matching with examples",
		},
		{
			name: "Pattern behavior section",
			mustExist: []string{
				"#### Pattern Matching Behavior",
				"The crawler supports **two pattern types**",
			},
			reason: "Should have a dedicated section explaining pattern behavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_SchemeValidation verifies scheme validation is documented
func TestURLFilteringGuide_SchemeValidation(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Scheme validation section",
			mustExist: []string{
				"### 3. Scheme Validation",
				"**only accepts HTTP and HTTPS**",
			},
			reason: "Should document scheme validation",
		},
		{
			name: "Filtered schemes table",
			mustExist: []string{
				"#### Automatically Filtered Schemes",
				"| Scheme | Example | Reason |",
				"`mailto:`",
				"`tel:`",
				"`javascript:`",
				"`ftp://`",
				"`file://`",
				"`data:`",
			},
			reason: "Should provide a table of filtered schemes",
		},
		{
			name: "Implementation reference",
			mustExist: []string{
				"Location: `internal/utils/url.go:242-249`",
				"ShouldSkipURL",
			},
			reason: "Should reference scheme validation implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_URLNormalization verifies URL normalization is documented
func TestURLFilteringGuide_URLNormalization(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "URL normalization section",
			mustExist: []string{
				"## URL Normalization",
				"**normalizes URLs** to avoid duplicate crawling",
			},
			reason: "Should document URL normalization",
		},
		{
			name: "Apache parameter removal",
			mustExist: []string{
				"### Apache Directory Listing Parameters",
				"NormalizeURLString",
				"`C=N;O=A`",
				"`C=M;O=D`",
			},
			reason: "Should explain Apache parameter normalization",
		},
		{
			name: "Normalization examples table",
			mustExist: []string{
				"#### Normalization Examples",
				"| Original URL | Normalized URL | Reason |",
			},
			reason: "Should provide normalization examples",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_ConfigurationExamples verifies configuration examples are provided
func TestURLFilteringGuide_ConfigurationExamples(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Configuration examples section",
			mustExist: []string{
				"## Configuration Examples",
			},
			reason: "Should have a section for configuration examples",
		},
		{
			name: "Minimal config example",
			mustExist: []string{
				"### Minimal Config",
				"```yaml",
				"excludePatterns: []",
			},
			reason: "Should show minimal configuration",
		},
		{
			name: "Blog archive config",
			mustExist: []string{
				"### Blog Archive Crawl",
				"```yaml",
				"excludePatterns:",
				"  - \"*.jpg\"",
			},
			reason: "Should show blog archive configuration",
		},
		{
			name: "Documentation crawl config",
			mustExist: []string{
				"### Documentation Crawl",
				"```yaml",
				"- \"/api/\"",
				"- \"/admin/*\"",
			},
			reason: "Should show documentation crawl configuration",
		},
		{
			name: "E-commerce config",
			mustExist: []string{
				"### E-commerce Product Catalog",
				"- \"/cart/*\"",
				"- \"/checkout/*\"",
			},
			reason: "Should show e-commerce configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_BestPractices verifies best practices are documented
func TestURLFilteringGuide_BestPractices(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Best practices section",
			mustExist: []string{
				"## Best Practices",
			},
			reason: "Should have a best practices section",
		},
		{
			name: "Start broad practice",
			mustExist: []string{
				"### 1. Start Broad, Then Refine",
				"Begin with minimal exclusions",
			},
			reason: "Should recommend starting with minimal exclusions",
		},
		{
			name: "Use specific patterns practice",
			mustExist: []string{
				"### 2. Use Specific Patterns",
				"Prefer specific patterns over broad ones",
			},
			reason: "Should recommend using specific patterns",
		},
		{
			name: "Test incrementally practice",
			mustExist: []string{
				"### 3. Test Patterns Incrementally",
				"Add patterns one at a time",
			},
			reason: "Should recommend incremental testing",
		},
		{
			name: "Use glob patterns practice",
			mustExist: []string{
				"### 4. Use Glob Patterns for Paths",
				"Use glob syntax for path matching",
			},
			reason: "Should recommend glob patterns for paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_Troubleshooting verifies troubleshooting guidance is provided
func TestURLFilteringGuide_Troubleshooting(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Troubleshooting section",
			mustExist: []string{
				"## Troubleshooting",
			},
			reason: "Should have a troubleshooting section",
		},
		{
			name: "URLs not excluded problem",
			mustExist: []string{
				"### URLs Not Being Excluded",
				"**Problem**: URLs matching exclude patterns are still being crawled",
				"**Solution**: Check pattern syntax",
			},
			reason: "Should provide troubleshooting for pattern matching",
		},
		{
			name: "Too many skipped problem",
			mustExist: []string{
				"### Too Many URLs Skipped",
				"**Problem**: Valid URLs are being excluded",
			},
			reason: "Should provide troubleshooting for over-matching",
		},
		{
			name: "Base path problem",
			mustExist: []string{
				"### URLs Outside Base Path",
				"**Problem**: URLs that should be crawled are being skipped",
			},
			reason: "Should provide troubleshooting for base path issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_FilteringFlowchart verifies filtering flowchart is included
func TestURLFilteringGuide_FilteringFlowchart(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Filtering flowchart section",
			mustExist: []string{
				"## Filtering Flowchart",
			},
			reason: "Should include a filtering flowchart",
		},
		{
			name: "Flowchart steps",
			mustExist: []string{
				"Link Discovered",
				"Parse URL",
				"Check Scheme",
				"Check Domain",
				"Check Base Path",
				"Check Exclude Patterns",
				"Check Normalization",
				"Add to Queue",
			},
			reason: "Should show all filtering steps in the flowchart",
		},
		{
			name: "Flowchart decisions",
			mustExist: []string{
				"Invalid syntax? → Skip",
				"Not HTTP/HTTPS? → Skip",
				"Different host? → Skip",
			},
			reason: "Should show decision points in flowchart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_References verifies implementation references are provided
func TestURLFilteringGuide_References(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "References section",
			mustExist: []string{
				"## References",
			},
			reason: "Should have a references section",
		},
		{
			name: "Implementation file references",
			mustExist: []string{
				"**URL Filtering Implementation**: `internal/crawlers/engine_crawler.go:338-371`",
				"**Path Validation**: `internal/utils/url.go:141-164`",
				"**Pattern Matching**: `internal/utils/url.go:251-264`",
				"**Normalization**: `internal/utils/url.go:266-284`",
				"**Scheme Validation**: `internal/utils/url.go:242-249`",
			},
			reason: "Should reference all relevant implementation files with line numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_TestingGuidance verifies testing guidance is provided
func TestURLFilteringGuide_TestingGuidance(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		mustExist []string
		reason    string
	}{
		{
			name: "Testing section",
			mustExist: []string{
				"## Testing URL Filtering",
			},
			reason: "Should have a testing section",
		},
		{
			name: "Verbose mode testing",
			mustExist: []string{
				"### Verbose Mode",
				"`--verbose`",
				"`grep -E \"skip|Skip\"`",
			},
			reason: "Should explain verbose mode testing",
		},
		{
			name: "Dry run testing",
			mustExist: []string{
				"### Dry Run with Max Pages",
				"`--max-pages 5`",
			},
			reason: "Should explain dry run testing with max pages",
		},
		{
			name: "Single page testing",
			mustExist: []string{
				"### Single Page Test",
				"`--max-pages 1`",
			},
			reason: "Should explain single page testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, required := range tt.mustExist {
				assert.Contains(t, docStr, required, tt.reason)
			}
		})
	}
}

// TestURLFilteringGuide_DocumentationQuality checks overall documentation quality
func TestURLFilteringGuide_DocumentationQuality(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	tests := []struct {
		name      string
		condition func() bool
		reason    string
	}{
		{
			name: "Has overview section",
			condition: func() bool {
				return strings.Contains(docStr, "## Overview") &&
					strings.Contains(docStr, "multi-layer URL filtering")
			},
			reason: "Should have an overview explaining the filtering system",
		},
		{
			name: "Has filtering rules section",
			condition: func() bool {
				return strings.Contains(docStr, "## Filtering Rules")
			},
			reason: "Should have a filtering rules section",
		},
		{
			name: "Uses code blocks for examples",
			condition: func() bool {
				return strings.Count(docStr, "```go") >= 3 &&
					strings.Count(docStr, "```yaml") >= 3
			},
			reason: "Should provide multiple code examples in Go and YAML",
		},
		{
			name: "Uses tables for reference",
			condition: func() bool {
				return strings.Count(docStr, "|") > 100
			},
			reason: "Should use tables for organized reference information",
		},
		{
			name: "Provides concrete examples",
			condition: func() bool {
				return strings.Contains(docStr, "✅") && strings.Contains(docStr, "❌")
			},
			reason: "Should use checkmarks and X marks for visual examples",
		},
		{
			name: "Includes warnings",
			condition: func() bool {
				return strings.Contains(docStr, "**Warning**") ||
					strings.Contains(docStr, "**Note**")
			},
			reason: "Should include warnings and notes for important caveats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.condition(), tt.reason)
		})
	}
}

// TestURLFilteringGuide_AllFilteringLayersCovered verifies all filtering layers are documented
func TestURLFilteringGuide_AllFilteringLayersCovered(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "guides", "url-filtering.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// Verify all layers mentioned in overview are documented
	filteringLayers := []struct {
		layer    string
		section  string
		implRef  string
		examples bool
	}{
		{
			layer:    "Domain Restriction",
			section:  "### 1. Domain Restriction",
			implRef:  "engine_crawler.go",
			examples: true,
		},
		{
			layer:    "Path Restriction",
			section:  "### 2. Path Restriction",
			implRef:  "url.go:141-164",
			examples: true,
		},
		{
			layer:    "Scheme Validation",
			section:  "### 3. Scheme Validation",
			implRef:  "url.go:242-249",
			examples: true,
		},
		{
			layer:    "Exclude Patterns",
			section:  "### 4. Exclude Patterns",
			implRef:  "url.go:251-264",
			examples: true,
		},
		{
			layer:    "Normalization",
			section:  "URL Normalization",
			implRef:  "url.go:266-284",
			examples: true,
		},
	}

	for _, layer := range filteringLayers {
		t.Run(layer.layer+" documented", func(t *testing.T) {
			assert.Contains(t, docStr, layer.section, "Should document %s layer", layer.layer)
			assert.Contains(t, docStr, layer.implRef, "Should reference implementation for %s", layer.layer)

			if layer.examples {
				// Check for examples (crawled/skipped or YAML/Go code)
				hasExamples := strings.Contains(docStr, "✅") ||
					strings.Contains(docStr, "```yaml") ||
					strings.Contains(docStr, "```go")
				assert.True(t, hasExamples, "Should provide examples for %s", layer.layer)
			}
		})
	}
}
