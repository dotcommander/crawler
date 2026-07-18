package exporters

import (
	"fmt"
	"io"
	"time"
)

// PageRecord represents a single crawled page for export
type PageRecord struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	LinksFound  int               `json:"links_found"`
	CrawledAt   time.Time         `json:"crawled_at"`
	Extracted   map[string]string `json:"extracted,omitzero"`
}

// Exporter defines the interface for structured output formats
type Exporter interface {
	WriteRecord(record PageRecord) error
	Close() error
}

// New creates an exporter for the given format writing to w. extractedKeys
// defines the configured CSV extraction columns; other formats ignore it.
func New(format string, w io.Writer, extractedKeys ...string) (Exporter, error) {
	switch format {
	case "jsonl":
		return NewJSONLExporter(w), nil
	case "csv":
		return NewCSVExporter(w, extractedKeys...), nil
	case "sitemap":
		return NewSitemapExporter(w), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}
