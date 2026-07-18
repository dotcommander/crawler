package crawlers

import (
	"net/url"
	"regexp"
	"strings"
)

var jsEndpointPatterns = []*regexp.Regexp{
	// Full URLs in string literals
	regexp.MustCompile("(?:\"|'|`)(" + `https?://[^\s"'` + "`" + `\]>}{]+` + ")(?:\"|'|`)"),
	// Relative paths starting with / in string literals (min 2 path chars)
	regexp.MustCompile("(?:\"|'|`)(" + `/[a-zA-Z0-9_][a-zA-Z0-9_./-]+` + ")(?:\"|'|`)"),
}

var scriptTagPattern = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)

var scriptSrcPattern = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*["']([^"']+)["']`)

// ExtractJSEndpoints extracts URL endpoints from HTML content by parsing
// inline <script> tags and applying regex patterns to find paths and URLs.
func ExtractJSEndpoints(htmlContent []byte, baseURL *url.URL) []string {
	if len(htmlContent) == 0 || baseURL == nil {
		return nil
	}

	matches := scriptTagPattern.FindAllStringSubmatch(string(htmlContent), -1)
	var jsBlocks []string
	for _, match := range matches {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			jsBlocks = append(jsBlocks, match[1])
		}
	}

	if len(jsBlocks) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var endpoints []string

	for _, block := range jsBlocks {
		for _, pattern := range jsEndpointPatterns {
			found := pattern.FindAllStringSubmatch(block, -1)
			for _, m := range found {
				if len(m) < 2 {
					continue
				}
				raw := strings.TrimSpace(m[1])
				if raw == "" || shouldSkipJSEndpoint(raw) {
					continue
				}

				resolved := resolveEndpoint(raw, baseURL)
				if resolved == "" {
					continue
				}

				if _, exists := seen[resolved]; exists {
					continue
				}
				seen[resolved] = struct{}{}
				endpoints = append(endpoints, resolved)
			}
		}
	}

	return endpoints
}

// ExtractScriptSrcURLs extracts URLs from <script src="..."> tags in HTML content.
func ExtractScriptSrcURLs(htmlContent []byte, baseURL *url.URL) []string {
	if len(htmlContent) == 0 || baseURL == nil {
		return nil
	}

	matches := scriptSrcPattern.FindAllSubmatch(htmlContent, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var urls []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(string(match[1]))
		if raw == "" {
			continue
		}

		resolved := resolveEndpoint(raw, baseURL)
		if resolved == "" {
			continue
		}

		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		urls = append(urls, resolved)
	}

	return urls
}

// ExtractJSFromRawSource extracts URL endpoints from raw JavaScript source code
// (not wrapped in <script> tags). Used for parsing fetched external .js files.
func ExtractJSFromRawSource(jsContent []byte, baseURL *url.URL) []string {
	if len(jsContent) == 0 || baseURL == nil {
		return nil
	}

	content := string(jsContent)
	seen := make(map[string]struct{})
	var endpoints []string

	for _, pattern := range jsEndpointPatterns {
		found := pattern.FindAllStringSubmatch(content, -1)
		for _, m := range found {
			if len(m) < 2 {
				continue
			}
			raw := strings.TrimSpace(m[1])
			if raw == "" || shouldSkipJSEndpoint(raw) {
				continue
			}

			resolved := resolveEndpoint(raw, baseURL)
			if resolved == "" {
				continue
			}

			if _, exists := seen[resolved]; exists {
				continue
			}
			seen[resolved] = struct{}{}
			endpoints = append(endpoints, resolved)
		}
	}

	return endpoints
}

func shouldSkipJSEndpoint(endpoint string) bool {
	if len(endpoint) < 3 {
		return true
	}

	lower := strings.ToLower(endpoint)
	skipSuffixes := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".map"}
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	if strings.HasPrefix(endpoint, "//") && !strings.Contains(endpoint[2:], "/") {
		return true
	}

	if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "blob:") {
		return true
	}

	if endpoint == "//" || endpoint == "/*" || endpoint == "/**" {
		return true
	}

	return false
}

func resolveEndpoint(raw string, baseURL *url.URL) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return parsed.String()
	}

	if strings.HasPrefix(raw, "/") {
		ref, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return baseURL.ResolveReference(ref).String()
	}

	return ""
}
