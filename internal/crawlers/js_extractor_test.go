package crawlers

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSEndpoints(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/app/")
	require.NoError(t, err)

	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name: "fetch calls",
			html: `<html><script>
				fetch("/api/users")
				fetch("/api/subscribe.php")
			</script></html>`,
			expected: []string{
				"https://example.com/api/users",
				"https://example.com/api/subscribe.php",
			},
		},
		{
			name: "full URLs",
			html: `<html><script>
				var url = "https://api.example.com/v2/data";
			</script></html>`,
			expected: []string{
				"https://api.example.com/v2/data",
			},
		},
		{
			name: "mixed quotes",
			html: `<html><script>
				axios.get('/api/items')
				const endpoint = "/api/search"
			</script></html>`,
			expected: []string{
				"https://example.com/api/items",
				"https://example.com/api/search",
			},
		},
		{
			name: "skips static assets",
			html: `<html><script>
				var css = "/styles/main.css"
				var js = "/bundle.js"
				var api = "/api/data"
			</script></html>`,
			expected: []string{
				"https://example.com/api/data",
			},
		},
		{
			name:     "no script tags",
			html:     `<html><body><a href="/page">link</a></body></html>`,
			expected: nil,
		},
		{
			name:     "empty script",
			html:     `<html><script></script></html>`,
			expected: nil,
		},
		{
			name:     "empty content",
			html:     "",
			expected: nil,
		},
		{
			name: "deduplicates",
			html: `<html><script>
				fetch("/api/users")
				fetch("/api/users")
			</script></html>`,
			expected: []string{
				"https://example.com/api/users",
			},
		},
		{
			name: "multiple script blocks",
			html: `<html>
				<script>var a = "/api/one"</script>
				<script>var b = "/api/two"</script>
			</html>`,
			expected: []string{
				"https://example.com/api/one",
				"https://example.com/api/two",
			},
		},
		{
			name:     "skips short paths",
			html:     `<html><script>var x = "/a"</script></html>`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSEndpoints([]byte(tt.html), baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipJSEndpoint(t *testing.T) {
	assert.True(t, shouldSkipJSEndpoint("/a"))
	assert.True(t, shouldSkipJSEndpoint("/style.css"))
	assert.True(t, shouldSkipJSEndpoint("/app.js"))
	assert.True(t, shouldSkipJSEndpoint("//"))
	assert.True(t, shouldSkipJSEndpoint("data:text"))
	assert.False(t, shouldSkipJSEndpoint("/api/users"))
	assert.False(t, shouldSkipJSEndpoint("/submit.php"))
}

func TestResolveEndpoint(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/app/")
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/api/data", resolveEndpoint("/api/data", baseURL))
	assert.Equal(t, "https://other.com/path", resolveEndpoint("https://other.com/path", baseURL))
	assert.Equal(t, "", resolveEndpoint("relative/path", baseURL))
	assert.Equal(t, "", resolveEndpoint("", baseURL))
}

func TestExtractScriptSrcURLs(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("https://example.com/app/")
	require.NoError(t, err)

	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name:     "single script src",
			html:     `<html><script src="/js/app.js"></script></html>`,
			expected: []string{"https://example.com/js/app.js"},
		},
		{
			name:     "multiple script srcs",
			html:     `<html><script src="/js/vendor.js"></script><script src="/js/app.js"></script></html>`,
			expected: []string{"https://example.com/js/vendor.js", "https://example.com/js/app.js"},
		},
		{
			name:     "absolute URL",
			html:     `<html><script src="https://cdn.example.com/lib.js"></script></html>`,
			expected: []string{"https://cdn.example.com/lib.js"},
		},
		{
			name:     "deduplicates",
			html:     `<html><script src="/js/app.js"></script><script src="/js/app.js"></script></html>`,
			expected: []string{"https://example.com/js/app.js"},
		},
		{
			name:     "inline script ignored",
			html:     `<html><script>var x = 1;</script></html>`,
			expected: nil,
		},
		{
			name:     "empty content",
			html:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractScriptSrcURLs([]byte(tt.html), baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSFromRawSource(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("https://example.com/")
	require.NoError(t, err)

	tests := []struct {
		name     string
		js       string
		expected []string
	}{
		{
			name:     "fetch call",
			js:       `fetch("/api/users")`,
			expected: []string{"https://example.com/api/users"},
		},
		{
			name:     "multiple endpoints",
			js:       `fetch("/api/users"); fetch("/api/posts");`,
			expected: []string{"https://example.com/api/users", "https://example.com/api/posts"},
		},
		{
			name:     "empty content",
			js:       "",
			expected: nil,
		},
		{
			name:     "deduplicates",
			js:       `fetch("/api/users"); fetch("/api/users");`,
			expected: []string{"https://example.com/api/users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractJSFromRawSource([]byte(tt.js), baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}
