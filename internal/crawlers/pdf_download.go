package crawlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dotcommander/crawler/internal/config"
)

const maxPDFDownloadSize = 50 * 1024 * 1024

func downloadPDFWithHTTP(ctx context.Context, cfg *config.CrawlerConfig, pdfURL string, result *CrawlResult) (*CrawlResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("create PDF request: %w", err)
		return result, nil
	}

	userAgent := cfg.UserAgent
	if cfg.Mobile && cfg.MobileUserAgent != "" {
		userAgent = cfg.MobileUserAgent
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("download PDF %s: %w", pdfURL, err)
		return result, nil
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("download PDF %s: HTTP %d", pdfURL, resp.StatusCode)
		return result, nil
	}

	limited := io.LimitReader(resp.Body, maxPDFDownloadSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		result.Error = fmt.Errorf("read PDF body: %w", err)
		return result, nil
	}
	if len(body) > maxPDFDownloadSize {
		result.Error = fmt.Errorf("PDF exceeds maximum size of %d bytes", maxPDFDownloadSize)
		return result, nil
	}

	result.Content = body
	result.ContentLength = int64(len(body))
	result.IsPDF = true
	result.Success = true
	return result, nil
}
