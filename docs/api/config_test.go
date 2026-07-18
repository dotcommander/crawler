package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigAPI_AllFieldsDocumented verifies that all config.Config fields are listed in the API doc
func TestConfigAPI_AllFieldsDocumented(t *testing.T) {
	// Read the config API documentation
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "config.md should exist")
	docStr := string(content)

	// All fields from config.CrawlerConfig struct
	expectedFields := []string{
		"StartURL",
		"OutputDir",
		"CacheDir",
		"MaxDepth",
		"Concurrency",
		"DefaultDelay",
		"MaxRetries",
		"Force",
		"DomainDelays",
		"ExcludePatterns",
		"UserAgent",
		"Headers",
		"Mobile",
		"MaxPages",
		"WaitStrategy",
		"ExtraWaitTime",
	}

	// Check each field is documented
	for _, field := range expectedFields {
		// Look for field in markdown table format
		found := strings.Contains(docStr, "`"+field+"`")
		assert.True(t, found, "Field %s should be documented in config.md", field)
	}
}

// TestConfigAPI_FieldTypesDocumented verifies that each field has its type documented
func TestConfigAPI_FieldTypesDocumented(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// Field type mappings from the actual struct
	fieldTypes := map[string]string{
		"StartURL":        "string",
		"OutputDir":       "string",
		"CacheDir":        "string",
		"MaxDepth":        "int",
		"Concurrency":     "int",
		"DefaultDelay":    "time.Duration",
		"MaxRetries":      "int",
		"Force":           "bool",
		"DomainDelays":    "map[string]time.Duration",
		"ExcludePatterns": "[]string",
		"UserAgent":       "string",
		"Headers":         "map[string]string",
		"Mobile":          "bool",
		"MaxPages":        "int",
		"WaitStrategy":    "string",
		"ExtraWaitTime":   "time.Duration",
	}

	// Check each field's type is documented
	for field, expectedType := range fieldTypes {
		// Look for the field followed by its type in the table
		found := strings.Contains(docStr, "`"+field+"`") && strings.Contains(docStr, "`"+expectedType+"`")
		assert.True(t, found, "Field %s should have type %s documented", field, expectedType)
	}
}

// TestConfigAPI_DefaultsSpecified verifies that default values are specified for each field
func TestConfigAPI_DefaultsSpecified(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// Expected default values from viper_config.go setDefaults()
	expectedDefaults := map[string]string{
		"MaxDepth":      "3",
		"Concurrency":   "5",
		"DefaultDelay":  "1s",
		"MaxRetries":    "2",
		"Force":         "false",
		"Mobile":        "false",
		"MaxPages":      "0",
		"WaitStrategy":  `"networkidle"`,
		"ExtraWaitTime": "500ms",
	}

	// Check that defaults are documented
	for field, defaultValue := range expectedDefaults {
		found := strings.Contains(docStr, defaultValue)
		assert.True(t, found, "Default value %s for field %s should be documented", defaultValue, field)
	}

	// Verify the Default column exists and has values
	assert.Contains(t, docStr, "| Default |", "Table should have a Default column")
	assert.Contains(t, docStr, "`~/.config/crawler/storage`", "OutputDir default should be documented")
}

// TestConfigAPI_CLIFlagMapping verifies that field descriptions map to CLI flag behavior
func TestConfigAPI_CLIFlagMapping(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// CLI flag mappings from cmd/root.go
	cliMappings := map[string]string{
		"MaxDepth":    "--depth",
		"Concurrency": "--concurrency",
		"MaxRetries":  "--max-retries",
		"Force":       "--force",
		"Mobile":      "--mobile",
		"MaxPages":    "--max-pages",
		"OutputDir":   "--output",
	}

	// Verify CLI flags are documented in the table
	for _, flag := range cliMappings {
		found := strings.Contains(docStr, "`"+flag+"`")
		assert.True(t, found, "CLI flag %s should be documented", flag)
	}

	// Verify the CLI Flag column exists
	assert.Contains(t, docStr, "| CLI Flag |", "Table should have a CLI Flag column")
}

// TestConfigAPI_EnvVarMapping verifies that field descriptions map to environment variables
func TestConfigAPI_EnvVarMapping(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// Expected environment variable mappings
	envMappings := map[string]string{
		"CRAWLER_DEPTH":       "MaxDepth",
		"CRAWLER_CONCURRENCY": "Concurrency",
		"CRAWLER_DELAY":       "DefaultDelay",
		"CRAWLER_FORCE":       "Force",
		"CRAWLER_MOBILE":      "Mobile",
		"CRAWLER_MAXPAGES":    "MaxPages",
	}

	// Verify environment variables are documented
	for envVar := range envMappings {
		found := strings.Contains(docStr, "`"+envVar+"`")
		assert.True(t, found, "Environment variable %s should be documented", envVar)
	}

	// Verify the Env Var column exists
	assert.Contains(t, docStr, "| Env Var |", "Table should have an Env Var column")
}

// TestConfigAPI_ConfigKeyMapping verifies that field descriptions map to YAML config keys
func TestConfigAPI_ConfigKeyMapping(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "api", "config.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)
	docStr := string(content)

	// Config key mappings from viper_config.go
	configKeys := map[string]string{
		"depth":          "MaxDepth",
		"concurrency":    "Concurrency",
		"delay":          "DefaultDelay",
		"maxRetries":     "MaxRetries",
		"mobile":         "Mobile",
		"maxPages":       "MaxPages",
		"waitStrategy":   "WaitStrategy",
		"extraWaitTime":  "ExtraWaitTime",
		"ignorePatterns": "ExcludePatterns",
		"domainDelays":   "DomainDelays",
		"userAgent":      "UserAgent",
		"headers":        "Headers",
		"force":          "Force",
	}

	// Verify config keys are documented
	for configKey := range configKeys {
		found := strings.Contains(docStr, "`"+configKey+"`")
		assert.True(t, found, "Config key %s should be documented", configKey)
	}

	// Verify the Config Key column exists
	assert.Contains(t, docStr, "| Config Key |", "Table should have a Config Key column")
}
