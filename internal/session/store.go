package session

import "io"

// VisitedStore tracks which URLs have been visited during a crawl session.
type VisitedStore interface {
	// MarkVisited atomically marks a URL as visited.
	// Returns true if the URL was already visited.
	MarkVisited(url string) (alreadyVisited bool)

	// RecordResult updates the status code for a previously visited URL.
	RecordResult(url string, statusCode int) error

	// IsVisited checks whether a URL has been visited.
	IsVisited(url string) bool

	io.Closer
}
