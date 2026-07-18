package exporters

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONLExporter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	exp := NewJSONLExporter(&buf)

	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	err := exp.WriteRecord(PageRecord{
		URL:         "https://example.com",
		Title:       "Example",
		StatusCode:  200,
		ContentType: "text/html",
		LinksFound:  5,
		CrawledAt:   ts,
	})
	require.NoError(t, err)
	require.NoError(t, exp.Close())

	var got PageRecord
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got.URL)
	assert.Equal(t, "Example", got.Title)
	assert.Equal(t, 200, got.StatusCode)
	assert.Equal(t, 5, got.LinksFound)
}

func TestCSVExporter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	exp := NewCSVExporter(&buf)

	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	err := exp.WriteRecord(PageRecord{
		URL:         "https://example.com",
		Title:       "Example",
		StatusCode:  200,
		ContentType: "text/html",
		LinksFound:  5,
		CrawledAt:   ts,
	})
	require.NoError(t, err)
	require.NoError(t, exp.Close())

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "url,title,status_code,content_type,links_found,crawled_at", lines[0])
	assert.Contains(t, lines[1], "https://example.com")
	assert.Contains(t, lines[1], "2026-03-18T12:00:00Z")
}

func TestCSVExporterUsesConfiguredExtractionColumns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	exp := NewCSVExporter(&buf, "summary", "heading")
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	require.NoError(t, exp.WriteRecord(PageRecord{
		URL:       "https://example.com/first",
		CrawledAt: ts,
	}))
	require.NoError(t, exp.WriteRecord(PageRecord{
		URL:       "https://example.com/second",
		CrawledAt: ts,
		Extracted: map[string]string{
			"heading": "Second page heading",
			"summary": "Second page summary",
		},
	}))
	require.NoError(t, exp.Close())

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"url", "title", "status_code", "content_type", "links_found", "crawled_at", "heading", "summary"}, rows[0])
	assert.Equal(t, "", rows[1][6])
	assert.Equal(t, "Second page heading", rows[2][6])
	assert.Equal(t, "Second page summary", rows[2][7])
}

func TestSitemapExporter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	exp := NewSitemapExporter(&buf)

	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	err := exp.WriteRecord(PageRecord{
		URL:       "https://example.com",
		CrawledAt: ts,
	})
	require.NoError(t, err)
	require.NoError(t, exp.Close())

	output := buf.String()
	assert.Contains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, output, `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`)
	assert.Contains(t, output, `<loc>https://example.com</loc>`)
	assert.Contains(t, output, `<lastmod>2026-03-18</lastmod>`)
}

func TestNewExporterInvalidFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	_, err := New("invalid", &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}
