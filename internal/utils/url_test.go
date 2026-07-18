package utils

import (
	"net/url"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
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
			name:     "URL with trailing slash",
			input:    "https://example.com/path/",
			expected: "https://example.com/path",
		},
		{
			name:     "root URL with trailing slash",
			input:    "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "URL with default HTTP port",
			input:    "http://example.com:80/path",
			expected: "http://example.com/path",
		},
		{
			name:     "URL with default HTTPS port",
			input:    "https://example.com:443/path",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with Apache sort parameters",
			input:    "https://example.com/dir?C=N;O=D",
			expected: "https://example.com/dir",
		},
		{
			name:     "URL with mixed query parameters",
			input:    "https://example.com/dir?search=test&C=N;O=D&other=value",
			expected: "https://example.com/dir?other=value&search=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}
			result := NormalizeURL(u)
			if result != tt.expected {
				t.Errorf("NormalizeURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsApacheSortLink(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Apache sort link with C and O",
			input:    "https://example.com/dir?C=N;O=D",
			expected: true,
		},
		{
			name:     "Apache sort link with only C",
			input:    "https://example.com/dir?C=N",
			expected: true,
		},
		{
			name:     "Apache sort link with only O",
			input:    "https://example.com/dir?O=D",
			expected: true,
		},
		{
			name:     "Regular URL without sort parameters",
			input:    "https://example.com/dir?search=test",
			expected: false,
		},
		{
			name:     "URL without query parameters",
			input:    "https://example.com/dir",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}
			result := IsApacheSortLink(u)
			if result != tt.expected {
				t.Errorf("IsApacheSortLink() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateAndParseURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid HTTP URL",
			input:     "http://example.com/path",
			shouldErr: false,
		},
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
			name:      "directory traversal attempt",
			input:     "https://example.com/../etc/passwd",
			shouldErr: true,
			errMsg:    "directory traversal detected in path",
		},
		{
			name:      "encoded directory traversal",
			input:     "https://example.com/%2e%2e/etc/passwd",
			shouldErr: true,
			errMsg:    "encoded directory traversal detected",
		},
		{
			name:      "suspicious hostname",
			input:     "https://evil..example.com/path",
			shouldErr: true,
			errMsg:    "suspicious characters in hostname",
		},
		{
			name:      "empty hostname",
			input:     "https:///path",
			shouldErr: true,
			errMsg:    "hostname cannot be empty",
		},
		{
			name:      "malformed URL",
			input:     "not-a-url",
			shouldErr: true,
			errMsg:    "only HTTP and HTTPS schemes allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ValidateAndParseURL(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Errorf("Expected valid URL but got nil")
				}
			}
		})
	}
}

func TestIsWithinBasePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		targetURL string
		baseURL   string
		expected  bool
	}{
		{
			name:      "target within base directory",
			targetURL: "https://example.com/docs/page1.html",
			baseURL:   "https://example.com/docs/",
			expected:  true,
		},
		{
			name:      "target outside base directory",
			targetURL: "https://example.com/other/page1.html",
			baseURL:   "https://example.com/docs/",
			expected:  false,
		},
		{
			name:      "base is file, target in same directory",
			targetURL: "https://example.com/docs/page2.html",
			baseURL:   "https://example.com/docs/index.html",
			expected:  true,
		},
		{
			name:      "target at root, base at root",
			targetURL: "https://example.com/page.html",
			baseURL:   "https://example.com/",
			expected:  true,
		},
		{
			name:      "target at subdirectory of root",
			targetURL: "https://example.com/sub/page.html",
			baseURL:   "https://example.com/",
			expected:  true,
		},
		{
			name:      "versioned seed path does not collapse scope",
			targetURL: "https://example.com/v1.2/intro.html",
			baseURL:   "https://example.com/v1.2",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			targetU, err := url.Parse(tt.targetURL)
			if err != nil {
				t.Fatalf("Failed to parse target URL: %v", err)
			}
			baseU, err := url.Parse(tt.baseURL)
			if err != nil {
				t.Fatalf("Failed to parse base URL: %v", err)
			}

			result := IsWithinBasePath(targetU, baseU)
			if result != tt.expected {
				t.Errorf("IsWithinBasePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "HTTP URL should not be skipped",
			input:    "http://example.com",
			expected: false,
		},
		{
			name:     "HTTPS URL should not be skipped",
			input:    "https://example.com",
			expected: false,
		},
		{
			name:     "mailto URL should be skipped",
			input:    "mailto:test@example.com",
			expected: true,
		},
		{
			name:     "FTP URL should be skipped",
			input:    "ftp://files.example.com",
			expected: true,
		},
		{
			name:     "JavaScript URL should be skipped",
			input:    "javascript:alert('test')",
			expected: true,
		},
		{
			name:     "tel URL should be skipped",
			input:    "tel:+1234567890",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}
			result := ShouldSkipURL(u)
			if result != tt.expected {
				t.Errorf("ShouldSkipURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchesExcludePatterns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		url      string
		patterns []string
		expected bool
	}{
		{
			name:     "URL matches glob pattern",
			url:      "file.pdf",
			patterns: []string{"*.pdf"},
			expected: true,
		},
		{
			name:     "URL matches substring pattern",
			url:      "https://example.com/admin/login",
			patterns: []string{"admin"},
			expected: true,
		},
		{
			name:     "URL does not match any pattern",
			url:      "https://example.com/page.html",
			patterns: []string{"*.pdf", "admin"},
			expected: false,
		},
		{
			name:     "Empty patterns",
			url:      "https://example.com/page.html",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "Multiple patterns, one matches",
			url:      "https://example.com/logout.php",
			patterns: []string{"*.pdf", "logout", "admin"},
			expected: true,
		},
		{
			name:     "glob matches nested path segment",
			url:      "https://example.com/a/b/c/report.pdf",
			patterns: []string{"*.pdf"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := MatchesExcludePatterns(tt.url, tt.patterns)
			if result != tt.expected {
				t.Errorf("MatchesExcludePatterns() = %v, want %v", result, tt.expected)
			}
		})
	}
}
