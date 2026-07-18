package utils

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// NormalizeURL normalizes a URL by:
// - Removing fragments
// - Removing trailing slash (except for root)
// - Removing default ports
// - Removing Apache directory listing sort parameters
func NormalizeURL(u *url.URL) string {
	// Create a copy to avoid modifying the original
	normalized := *u

	// Remove fragment
	normalized.Fragment = ""

	// Remove trailing slash unless it's root
	if normalized.Path != "/" && strings.HasSuffix(normalized.Path, "/") {
		normalized.Path = strings.TrimSuffix(normalized.Path, "/")
	}

	// Remove default ports
	if (normalized.Scheme == "http" && normalized.Port() == "80") ||
		(normalized.Scheme == "https" && normalized.Port() == "443") {
		normalized.Host = normalized.Hostname()
	}

	// Remove Apache directory listing sort parameters
	// These include: C (column), O (order), N (name), M (modified), S (size), D (descending), A (ascending)
	// Handle both & and ; as separators (Apache uses ; in directory listings)
	rawQuery := normalized.RawQuery
	if rawQuery != "" {
		// Replace semicolons with ampersands for proper parsing
		rawQuery = strings.ReplaceAll(rawQuery, ";", "&")
		normalized.RawQuery = rawQuery

		query := normalized.Query()
		if query.Has("C") || query.Has("O") {
			query.Del("C")
			query.Del("O")
			normalized.RawQuery = query.Encode()
		}
	}

	return normalized.String()
}

// NormalizeURLString normalizes a URL string, returning the original string if parsing fails
func NormalizeURLString(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	return NormalizeURL(u)
}

// IsApacheSortLink checks if a URL is an Apache directory listing sort link
func IsApacheSortLink(u *url.URL) bool {
	// Apache directory listing sort links have query parameters like ?C=N;O=D
	// Handle both & and ; as separators
	rawQuery := u.RawQuery
	if rawQuery != "" {
		// Replace semicolons with ampersands for proper parsing
		rawQuery = strings.ReplaceAll(rawQuery, ";", "&")
		u2 := *u
		u2.RawQuery = rawQuery
		query := u2.Query()
		return query.Has("C") || query.Has("O")
	}
	return false
}

// IsWithinBasePath reports whether targetURL's path is within the directory of
// baseURL. The base directory is the longest prefix of baseURL.Path ending in
// "/" (the directory containing the seed resource), or "/" for a root/empty
// path. This replaces an earlier dot-heuristic that called filepath.Dir on a
// URL path and collapsed seeds like "/v1.2" to base path "//", silently
// excluding every discovered link.
func IsWithinBasePath(targetURL, baseURL *url.URL) bool {
	baseDir := baseDirectory(baseURL.Path)

	targetPath := targetURL.Path
	if targetPath == "" {
		targetPath = "/"
	}

	return strings.HasPrefix(targetPath, baseDir)
}

// baseDirectory returns the longest prefix of p ending in "/", or "/" for
// root/empty paths. Examples: "/docs/" -> "/docs/", "/docs/index.html" ->
// "/docs/", "/v1.2" -> "/", "" -> "/", "/" -> "/".
func baseDirectory(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	if strings.HasSuffix(p, "/") {
		return p
	}
	if idx := strings.LastIndex(p, "/"); idx > 0 {
		return p[:idx+1]
	}
	return "/"
}

// ValidateAndParseURL performs comprehensive URL validation and parsing
func ValidateAndParseURL(urlStr string) (*url.URL, error) {
	// Parse the URL first to get detailed information
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.New("invalid URL format")
	}

	// Custom security validations first (more specific errors)
	if err := validateURLSecurity(u); err != nil {
		return nil, err
	}

	return u, nil
}

// validatePathSecurity checks for path traversal and other path-based attacks
func validatePathSecurity(path string) error {
	// Check for suspicious characters
	if strings.Contains(path, "\\") {
		return errors.New("backslash not allowed in URL path")
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return errors.New("directory traversal detected in path")
	}

	return nil
}

// validateURLSecurity performs security-focused URL validation including encoded paths
func validateURLSecurity(u *url.URL) error {
	// Ensure scheme is HTTP or HTTPS only
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only HTTP and HTTPS schemes allowed")
	}

	// Validate hostname (prevent manipulation attempts)
	if u.Hostname() == "" {
		return errors.New("hostname cannot be empty")
	}

	// Check for suspicious patterns in hostname
	hostname := u.Hostname()
	if strings.Contains(hostname, "..") || strings.Contains(hostname, "\\") {
		return errors.New("suspicious characters in hostname")
	}

	// Check raw path for encoded attacks first
	if u.RawPath != "" {
		lowerRawPath := strings.ToLower(u.RawPath)
		if strings.Contains(lowerRawPath, "%2e%2e") {
			return errors.New("encoded directory traversal detected")
		}
	}

	// Validate path for directory traversal attempts
	if err := validatePathSecurity(u.Path); err != nil {
		return err
	}

	return nil
}

// GenerateFilePath generates the file path for saving a URL's content with enhanced security
func GenerateFilePath(outputDir string, urlStr string) string {
	u, err := ValidateAndParseURL(urlStr)
	if err != nil {
		return filepath.Join(outputDir, "error.html")
	}

	// Remove query parameters and fragments for file path
	urlPath := u.Path

	// Handle root and empty paths
	if urlPath == "" || urlPath == "/" {
		urlPath = "/index.html"
	} else if !strings.Contains(path.Base(urlPath), ".") {
		// If no extension, assume it's a directory and add index.html
		urlPath = path.Join(urlPath, "index.html")
	}

	// Enhanced path cleaning and validation
	urlPath = filepath.Clean(urlPath)
	urlPath = strings.TrimPrefix(urlPath, "/")

	// Additional security: ensure no traversal in cleaned path
	if strings.Contains(urlPath, "..") {
		return filepath.Join(outputDir, "sanitized.html")
	}

	// Combine output directory, hostname, and path
	hostname := u.Hostname()
	// Sanitize hostname for filesystem
	hostname = strings.ReplaceAll(hostname, ":", "_")
	hostname = strings.ReplaceAll(hostname, "/", "_")

	return filepath.Join(outputDir, hostname, urlPath)
}

// ShouldSkipURL checks if a URL should be skipped based on its scheme
func ShouldSkipURL(u *url.URL) bool {
	// Skip non-HTTP(S) schemes (mailto:, tel:, ftp:, javascript:, etc.)
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return true
	}
	return false
}

// MatchesExcludePatterns reports whether urlStr matches any exclude pattern.
// Each pattern is matched three ways to support common authoring styles:
//   - path.Match against the full URL path (e.g. "/private/*")
//   - path.Match against the last path segment, so bare globs like "*.pdf"
//     match "/a/b.pdf" (path.Match's "*" does not cross "/", so a full-path
//     match alone would miss nested files)
//   - substring containment in the path (e.g. "logout")
func MatchesExcludePatterns(urlStr string, patterns []string) bool {
	full := urlStr
	last := urlStr
	if u, err := url.Parse(urlStr); err == nil {
		full = u.EscapedPath()
		if b := path.Base(full); b != "." && b != "/" {
			last = b
		}
	}
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, full); matched {
			return true
		}
		if matched, _ := path.Match(pattern, last); matched {
			return true
		}
		if strings.Contains(full, pattern) {
			return true
		}
	}
	return false
}
