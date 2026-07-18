package exporters

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
)

type CSVExporter struct {
	mu            sync.Mutex
	w             *csv.Writer
	headerWritten bool
	extractedKeys []string
}

func NewCSVExporter(w io.Writer, extractedKeys ...string) *CSVExporter {
	keys := append([]string(nil), extractedKeys...)
	sort.Strings(keys)
	return &CSVExporter{
		w:             csv.NewWriter(w),
		extractedKeys: keys,
	}
}

func (e *CSVExporter) WriteRecord(record PageRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.headerWritten {
		header := []string{"url", "title", "status_code", "content_type", "links_found", "crawled_at"}
		header = append(header, e.extractedKeys...)
		if err := e.w.Write(header); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		e.headerWritten = true
	}

	row := []string{
		record.URL,
		record.Title,
		strconv.Itoa(record.StatusCode),
		record.ContentType,
		strconv.Itoa(record.LinksFound),
		record.CrawledAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	for _, k := range e.extractedKeys {
		row = append(row, record.Extracted[k])
	}
	if err := e.w.Write(row); err != nil {
		return fmt.Errorf("write csv row: %w", err)
	}
	return nil
}

func (e *CSVExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.w.Flush()
	return e.w.Error()
}
