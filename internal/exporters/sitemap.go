package exporters

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync"
)

type sitemapURLEntry struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

type SitemapExporter struct {
	mu   sync.Mutex
	w    io.Writer
	urls []sitemapURLEntry
}

func NewSitemapExporter(w io.Writer) *SitemapExporter {
	return &SitemapExporter{w: w}
}

func (e *SitemapExporter) WriteRecord(record PageRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.urls = append(e.urls, sitemapURLEntry{
		Loc:     record.URL,
		LastMod: record.CrawledAt.UTC().Format("2006-01-02"),
	})
	return nil
}

func (e *SitemapExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	type urlset struct {
		XMLName xml.Name          `xml:"urlset"`
		XMLNS   string            `xml:"xmlns,attr"`
		URLs    []sitemapURLEntry `xml:"url"`
	}

	set := urlset{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  e.urls,
	}

	if _, err := fmt.Fprint(e.w, xml.Header); err != nil {
		return fmt.Errorf("write xml header: %w", err)
	}

	enc := xml.NewEncoder(e.w)
	enc.Indent("", "  ")
	if err := enc.Encode(set); err != nil {
		return fmt.Errorf("encode sitemap: %w", err)
	}
	return nil
}
