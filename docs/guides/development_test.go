package guides

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const developmentGuidePath = "development.md"

func TestDevelopmentGuide_Exists(t *testing.T) {
	// Given: The development guide path
	guidePath := developmentGuidePath

	// When: Checking if the file exists
	info, err := os.Stat(guidePath)

	// Then: File should exist and be readable
	require.NoError(t, err, "development.md should exist")
	assert.False(t, info.IsDir(), "development.md should be a file, not a directory")
}

func TestDevelopment_GoVersionRequirement(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Scanning for Go version requirements
	// Then: Required Go version should be documented
	assert.Contains(t, guideContent, "Go 1.24.2",
		"should specify the minimum Go version requirement")
	assert.Contains(t, guideContent, "golang.org",
		"should provide a link to download Go")
	assert.Contains(t, guideContent, "go version",
		"should show how to verify Go installation")
}

func TestDevelopment_RequiredTools(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Scanning for required tools
	// Then: Required tools should be listed
	assert.Contains(t, guideContent, "Git",
		"should list Git as a prerequisite")
}

func TestDevelopment_StandardGoCommands(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for standard Go commands
	// Then: All essential commands should be documented
	assert.Contains(t, guideContent, "go build",
		"should document go build command")
	assert.Contains(t, guideContent, "go test",
		"should document go test command")
	assert.Contains(t, guideContent, "go fmt",
		"should document go fmt command")
	assert.Contains(t, guideContent, "go vet",
		"should document go vet command")
}

func TestDevelopment_PlaywrightInstallation(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for Playwright documentation
	// Then: Playwright installation should be explained for optional testing
	assert.Contains(t, guideContent, "Playwright",
		"should mention Playwright for JavaScript crawling")

	// Should be marked as optional
	assert.True(t,
		strings.Contains(guideContent, "Optional") && strings.Index(guideContent, "Optional") < strings.Index(guideContent, "Playwright"),
		"Playwright should be marked as optional")

	// Should include installation instructions
	assert.Contains(t, guideContent, "playwright install",
		"should show how to install Playwright browsers")
}

func TestDevelopment_DevWorkflowSections(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Scanning for workflow sections
	// Then: Essential sections should be present
	assert.Contains(t, guideContent, "## Getting Started",
		"should have a Getting Started section")
	assert.Contains(t, guideContent, "## Development Commands",
		"should have a Development Commands section")
	assert.Contains(t, guideContent, "## Project Structure",
		"should have a Project Structure section")
	assert.Contains(t, guideContent, "## Contributing",
		"should have a Contributing section")
}

func TestDevelopment_CoverageInfo(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for test coverage information
	// Then: Current coverage stats should be mentioned
	assert.Contains(t, guideContent, "Test Coverage",
		"should include test coverage section")
	assert.Contains(t, guideContent, "53.7%",
		"should document URL utils coverage")
	assert.Contains(t, guideContent, "70.9%",
		"should document Config coverage")
	assert.Contains(t, guideContent, "go test -cover",
		"should show how to run tests with coverage")
}

func TestDevelopment_BuildCommandExamples(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Checking for build examples
	// Then: Build command should be documented with clear examples
	assert.Contains(t, guideContent, "go build -o crawler",
		"should show the build command with output binary")
	assert.Contains(t, guideContent, "go build ./...",
		"should show how to build all packages")
}

func TestDevelopment_Troubleshooting(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for troubleshooting section
	// Then: Common issues should be addressed
	assert.Contains(t, guideContent, "## Troubleshooting",
		"should have a Troubleshooting section")

	// Should include common issues
	troubleshootingSection := extractSection(guideContent, "## Troubleshooting")
	assert.Contains(t, troubleshootingSection, "Build Errors",
		"should address build errors")
	assert.Contains(t, troubleshootingSection, "Test Failures",
		"should address test failures")
	assert.Contains(t, troubleshootingSection, "Playwright Issues",
		"should address Playwright issues")
}

func TestDevelopment_DevWorkflow(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for development workflow
	// Then: Step-by-step workflow should be documented
	workflowSection := extractSection(guideContent, "## Development Workflow")

	assert.Contains(t, workflowSection, "1.",
		"should have numbered steps")
	assert.Contains(t, workflowSection, "Make Changes",
		"should describe making changes")
	assert.Contains(t, workflowSection, "go test",
		"should include testing in workflow")
	assert.Contains(t, workflowSection, "go fmt",
		"should include formatting in workflow")
}

func TestDevelopment_FilePath(t *testing.T) {
	// Given: The expected file location
	expectedPath := developmentGuidePath

	// When: Checking if file exists at correct location
	_, err := os.Stat(expectedPath)

	// Then: File should exist at the expected path
	require.NoError(t, err, "development.md should exist at docs/guides/")
}

// Helper function to extract a section from markdown content
func extractSection(content, sectionHeader string) string {
	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		return ""
	}

	sectionStart := idx
	remainingContent := content[sectionStart+len(sectionHeader):]

	// Find the next header (## ) that's not on the same line
	lines := strings.Split(remainingContent, "\n")
	var result []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") && len(strings.TrimSpace(line)) > 3 {
			// Found the next section header
			break
		}
		result = append(result, line)
	}

	return content[sectionStart : sectionStart+len(sectionHeader)+len(strings.Join(result, "\n"))]
}

func TestDevelopment_CloneInstructions(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for setup instructions
	// Then: Clone and setup steps should be clear
	assert.Contains(t, guideContent, "git clone",
		"should show how to clone the repository")
	assert.Contains(t, guideContent, "go mod download",
		"should show how to download dependencies")
}

func TestDevelopment_TestingExamples(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for test command examples
	// Then: Various test commands should be shown
	assert.Contains(t, guideContent, "go test ./...",
		"should show how to run all tests")
	assert.Contains(t, guideContent, "-cover",
		"should show coverage flag")
	assert.Contains(t, guideContent, "-run",
		"should show how to run specific tests")
}

func TestDevelopment_CodeQualityCommands(t *testing.T) {
	// Given: The development guide
	content, err := os.ReadFile(developmentGuidePath)
	require.NoError(t, err, "should be able to read development.md")

	guideContent := string(content)

	// When: Looking for code quality section
	// Then: Formatting and vetting commands should be documented
	qualitySection := extractSection(guideContent, "### Code Quality")

	assert.Contains(t, qualitySection, "go fmt",
		"should include go fmt for formatting")
	assert.Contains(t, qualitySection, "go vet",
		"should include go vet for static analysis")
}
