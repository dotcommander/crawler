package katanaengine

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/projectdiscovery/katana/pkg/output"
)

// PageResult is the page-level payload delivered to OnPage callbacks:
// a successfully crawled, newly visited page with its response body.
type PageResult struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        []byte
	IsPDF       bool
	// Depth is katana's hop count for the discovered link; the seed
	// response starts at depth 1.
	Depth int
}

// onResult maps a katana result onto reporter and visited-store updates.
// It is invoked from katana's worker goroutines.
func (e *KatanaEngine) onResult(r output.Result) {
	if r.Request == nil {
		return
	}
	raw := r.Request.URL
	if raw == "" {
		return
	}
	// Discovery-level notices carry no response — most notably katana
	// emits depth-exceeded URLs via Output(request, nil, ErrMaxDepth-
	// Reached) without visiting them. Consuming them here would mark the
	// URL visited and suppress its real result if it is (or becomes)
	// crawlable at a valid depth, so they are dropped entirely.
	if r.Response == nil {
		return
	}
	normalized := normalizeURL(raw)

	if !e.reservePageSlot() {
		return // page limit reached; run cancellation already triggered
	}

	if !e.robotsAllowed(normalized) {
		e.reporter.Log("WARN", fmt.Sprintf("Robots-disallowed, ignoring result: %s", normalized))
		return
	}

	alreadyVisited, err := e.store.MarkVisited(normalized)
	if err != nil {
		e.reporter.Log("ERROR", fmt.Sprintf("Failed to persist visit for %s: %v", normalized, err))
		e.countFailure()
		return
	}
	if alreadyVisited {
		return
	}

	statusCode, contentType, body := decodeResponse(r)
	failed := r.Error != "" || statusCode >= 400

	e.mu.Lock()
	if failed {
		e.pagesFailed++
	} else {
		e.pagesVisited++
		e.bytesDownloaded += int64(len(body))
		if strings.Contains(strings.ToLower(contentType), "application/pdf") {
			e.pdfsDownloaded++
		}
	}
	snapshot := e.statsLocked()
	e.mu.Unlock()

	e.reporter.UpdateWorker(1, "crawled", normalized)
	e.reporter.UpdateStats(snapshot)

	if statusCode > 0 {
		if err := e.store.RecordResult(normalized, statusCode); err != nil {
			e.reporter.Log("ERROR", fmt.Sprintf("Failed to record result for %s: %v", normalized, err))
		}
	}
	if r.Error != "" {
		e.reporter.Log("ERROR", fmt.Sprintf("%s: %s", normalized, r.Error))
		return
	}
	if e.settings.onPage != nil {
		e.settings.onPage(PageResult{
			URL:         normalized,
			StatusCode:  statusCode,
			ContentType: contentType,
			Body:        []byte(body),
			IsPDF:       strings.Contains(strings.ToLower(contentType), "application/pdf"),
			Depth:       r.Request.Depth,
		})
	}
}

// reservePageSlot enforces cfg.MaxPages across concurrent results and
// reports whether this page may proceed. Hitting the limit cancels the
// run exactly once.
func (e *KatanaEngine) reservePageSlot() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.MaxPages > 0 && e.pagesVisited+e.pagesFailed >= int64(e.cfg.MaxPages) {
		if !e.limitHit {
			e.limitHit = true
			e.reporter.Log("INFO", fmt.Sprintf("Page limit reached (%d), stopping", e.cfg.MaxPages))
			if e.cancel != nil {
				e.cancel()
			}
		}
		return false
	}
	return true
}

func (e *KatanaEngine) robotsAllowed(normalized string) bool {
	e.mu.Lock()
	robots := e.robots
	e.mu.Unlock()
	if robots == nil {
		return true
	}
	return robots.IsAllowed(pathOf(normalized), e.cfg.UserAgent)
}

func (e *KatanaEngine) countFailure() {
	e.mu.Lock()
	e.pagesFailed++
	e.mu.Unlock()
}

// decodeResponse extracts the fields crawler's reporter and session
// store need from a katana result. Title and full body capture for the
// storage layer land with the factory-wiring phase. Missing responses
// yield zero values.
func decodeResponse(r output.Result) (statusCode int, contentType, body string) {
	if r.Response == nil {
		return 0, "", ""
	}
	resp := r.Response
	statusCode = resp.StatusCode
	body = resp.Body
	if resp.Headers != nil {
		for k, v := range resp.Headers {
			if strings.EqualFold(k, "Content-Type") {
				contentType = v
				break
			}
		}
	}
	return statusCode, contentType, body
}

func pathOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "/"
	}
	if u.Path == "" {
		return "/"
	}
	return u.Path
}
