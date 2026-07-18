package crawlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
)

func TestCollyEngine_CrawlPage_PreVisitCancellation(t *testing.T) {
	t.Parallel()

	engine, err := NewCollyEngine(&config.CrawlerConfig{
		UserAgent:    "TestBot",
		DefaultDelay: 0,
	})
	if err != nil {
		t.Fatalf("NewCollyEngine: %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before any network work

	// Use an unroutable address; with a cancelled ctx no request should be
	// attempted, so this must return promptly with a cancellation error.
	item := &QueueItem{URL: "http://127.0.0.1:0/never"}

	resultCh := make(chan *CrawlResult, 1)
	go func() {
		res, _ := engine.CrawlPage(ctx, item)
		resultCh <- res
	}()

	select {
	case res := <-resultCh:
		if res.Success {
			t.Fatal("expected failure for cancelled context, got success")
		}
		if res.Error == nil {
			t.Fatal("expected a cancellation error, got nil")
		}
		if !errors.Is(res.Error, context.Canceled) {
			t.Fatalf("expected context.Canceled in error chain, got %v", res.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CrawlPage did not return promptly on pre-cancelled context")
	}
}

func TestCollyEngine_CrawlPage_SkipsNonHTMLNonPDFBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("binary-junk-payload"))
	}))
	t.Cleanup(srv.Close)

	engine, err := NewCollyEngine(&config.CrawlerConfig{
		UserAgent:    "TestBot",
		DefaultDelay: 0,
	})
	if err != nil {
		t.Fatalf("NewCollyEngine: %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })

	res, err := engine.CrawlPage(context.Background(), &QueueItem{URL: srv.URL})
	if err != nil {
		t.Fatalf("CrawlPage returned error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("unexpected crawl error: %v", res.Error)
	}
	if !res.Success {
		t.Fatal("expected non-HTML response to be marked successful")
	}
	// Body must NOT be stored for non-HTML/non-PDF content types.
	if res.Content != nil {
		t.Fatalf("expected Content to be nil for application/octet-stream, got %d bytes", len(res.Content))
	}
	// ContentLength still reflects the actual response size.
	if res.ContentLength == 0 {
		t.Fatal("expected ContentLength to be set even when body not stored")
	}
}

func TestDownloadPDFWithHTTPStoresContent(t *testing.T) {
	t.Parallel()

	const body = "%PDF-1.7 test payload"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "TestBot/1.0" {
			t.Errorf("User-Agent = %q, want TestBot/1.0", got)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	result := &CrawlResult{URL: srv.URL + "/file.pdf", IsPDF: true}
	res, err := downloadPDFWithHTTP(context.Background(), &config.CrawlerConfig{
		UserAgent: "TestBot/1.0",
	}, srv.URL+"/file.pdf", result)
	if err != nil {
		t.Fatalf("downloadPDFWithHTTP returned error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("unexpected result error: %v", res.Error)
	}
	if !res.Success {
		t.Fatal("expected PDF download success")
	}
	if string(res.Content) != body {
		t.Fatalf("Content = %q, want %q", string(res.Content), body)
	}
	if res.ContentLength != int64(len(body)) {
		t.Fatalf("ContentLength = %d, want %d", res.ContentLength, len(body))
	}
	if res.ContentType != "application/pdf" {
		t.Fatalf("ContentType = %q, want application/pdf", res.ContentType)
	}
}

func TestCollyEngine_GetOrCreateCollector_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	engine, err := NewCollyEngine(&config.CrawlerConfig{UserAgent: "TestBot", DefaultDelay: 0})
	if err != nil {
		t.Fatalf("NewCollyEngine: %v", err)
	}

	domains := []string{"a.example.com", "a.example.com", "b.example.com", "c.example.com", "a.example.com"}
	var wg sync.WaitGroup
	for _, d := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			_ = engine.getOrCreateCollector(domain)
		}(d)
	}
	wg.Wait()

	if err := engine.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
