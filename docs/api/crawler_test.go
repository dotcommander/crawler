package api_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestAPIDocExists verifies that docs/api/crawler.md exists
func TestAPIDocExists(t *testing.T) {
	path := "crawler.md"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("API documentation file does not exist: %s", path)
	}
}

// TestAPIDocDocumentsInterface verifies the doc documents the api.Crawler
// interface methods
func TestAPIDocDocumentsInterface(t *testing.T) {
	content, err := os.ReadFile("crawler.md")
	if err != nil {
		t.Fatalf("Failed to read API doc: %v", err)
	}

	text := string(content)

	// Check that interface methods are documented
	requiredMethods := []string{
		"Start()",
		"Cancel()",
		"Close()",
	}

	for _, method := range requiredMethods {
		if !strings.Contains(text, method) {
			t.Errorf("API doc missing documentation for method: %s", method)
		}
	}
}

// TestAPIDocMethodParameters verifies that Start(), Cancel(), Close() have
// parameter and return documentation
func TestAPIDocMethodParameters(t *testing.T) {
	content, err := os.ReadFile("crawler.md")
	if err != nil {
		t.Fatalf("Failed to read API doc: %v", err)
	}

	text := string(content)

	// Test Start() has return docs
	startSection := extractMethodSection(text, "Start()")
	if startSection == "" {
		t.Fatal("Could not find Start() method section")
	}

	if !strings.Contains(startSection, "Returns") {
		t.Error("Start() method missing return documentation")
	}

	if !strings.Contains(startSection, "error") {
		t.Error("Start() method missing 'error' return type documentation")
	}

	// Test Cancel() has return docs (even if it's "None")
	cancelSection := extractMethodSection(text, "Cancel()")
	if cancelSection == "" {
		t.Fatal("Could not find Cancel() method section")
	}

	if !strings.Contains(cancelSection, "Returns") {
		t.Error("Cancel() method missing return documentation")
	}

	// Test Close() has return docs
	closeSection := extractMethodSection(text, "Close()")
	if closeSection == "" {
		t.Fatal("Could not find Close() method section")
	}

	if !strings.Contains(closeSection, "Returns") {
		t.Error("Close() method missing return documentation")
	}
}

// TestAPIDocHasCodeExample verifies that a code snippet shows programmatic
// usage
func TestAPIDocHasCodeExample(t *testing.T) {
	content, err := os.ReadFile("crawler.md")
	if err != nil {
		t.Fatalf("Failed to read API doc: %v", err)
	}

	text := string(content)

	// Check for code block
	codeBlockPattern := regexp.MustCompile("```go" + `\n([\s\S]+?)` + "```")
	matches := codeBlockPattern.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		t.Fatal("API doc missing code examples (no ```go blocks found)")
	}

	// Verify at least one example shows programmatic usage
	hasProgrammaticExample := false
	requiredPatterns := []string{
		"crawler.Start()",
		"defer crawler.Close()",
	}

	for _, match := range matches {
		if len(match) > 1 {
			code := match[1]
			allPatternsFound := true
			for _, pattern := range requiredPatterns {
				if !strings.Contains(code, pattern) {
					allPatternsFound = false
					break
				}
			}
			if allPatternsFound {
				hasProgrammaticExample = true
				break
			}
		}
	}

	if !hasProgrammaticExample {
		t.Error("API doc missing programmatic usage example with Start() " +
			"and defer Close()")
	}
}

// TestAPIDocLineLength verifies documentation stays within 80 char limit
func TestAPIDocLineLength(t *testing.T) {
	content, err := os.ReadFile("crawler.md")
	if err != nil {
		t.Fatalf("Failed to read API doc: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	maxLineLength := 80

	for i, line := range lines {
		// Skip code blocks and URLs
		if strings.HasPrefix(strings.TrimSpace(line), "http") {
			continue
		}

		if len(line) > maxLineLength {
			t.Errorf("Line %d exceeds %d characters (length: %d): %s",
				i+1, maxLineLength, len(line),
				truncateString(line, 50))
		}
	}
}

// extractMethodSection extracts the documentation section for a given method
func extractMethodSection(text, methodName string) string {
	// Find the method heading (e.g., "### `Start() error`")
	// Extract from the method heading until the next heading or end of file
	lines := strings.Split(text, "\n")
	var result []string
	capturing := false

	methodHeading := "### `" + methodName

	for _, line := range lines {
		if capturing {
			// Stop at next heading
			if strings.HasPrefix(line, "###") {
				break
			}
			result = append(result, line)
		} else if strings.Contains(line, methodHeading) {
			capturing = true
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
