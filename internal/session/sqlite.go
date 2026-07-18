package session

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists visited URLs to a SQLite database.
type SQLiteStore struct {
	db      *sql.DB
	mu      sync.Mutex
	visited sync.Map // in-memory cache for fast lookups
}

// NewSQLiteStore creates a SQLite-backed visited store.
// If resume is false, the visited table is cleared on open.
func NewSQLiteStore(sessionsDir, startURL string, resume bool) (*SQLiteStore, error) {
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}

	hash := sha256.Sum256([]byte(startURL))
	dbName := fmt.Sprintf("%x.db", hash[:8])
	dbPath := filepath.Join(sessionsDir, dbName)

	db, err := sql.Open("sqlite", dbPath+"?_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("open session db: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS visited (
		url TEXT PRIMARY KEY,
		status_code INT DEFAULT 0,
		state TEXT NOT NULL DEFAULT 'pending',
		crawled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create visited table: %w", err)
	}

	// Backward-compat: add state column to DB files created before this column existed.
	// ADD COLUMN is not idempotent in SQLite, so ignore the "duplicate column" error.
	if _, err := db.Exec(`ALTER TABLE visited ADD COLUMN state TEXT NOT NULL DEFAULT 'pending'`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			db.Close()
			return nil, fmt.Errorf("add visited state column: %w", err)
		}
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_visited_state ON visited (state)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create visited state index: %w", err)
	}

	s := &SQLiteStore{db: db}

	if !resume {
		if _, err := db.Exec(`DELETE FROM visited`); err != nil {
			db.Close()
			return nil, fmt.Errorf("clear visited table: %w", err)
		}
	} else {
		rows, err := db.Query(`SELECT url FROM visited WHERE state = 'done'`)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("load visited urls: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var u string
			if err := rows.Scan(&u); err != nil {
				db.Close()
				return nil, fmt.Errorf("scan visited url: %w", err)
			}
			s.visited.Store(u, true)
		}
		if err := rows.Err(); err != nil {
			db.Close()
			return nil, fmt.Errorf("iterate visited urls: %w", err)
		}
	}

	return s, nil
}

func (s *SQLiteStore) MarkVisited(url string) bool {
	if _, loaded := s.visited.LoadOrStore(url, true); loaded {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(
		`INSERT OR IGNORE INTO visited (url, state, crawled_at) VALUES (?, 'in_flight', ?)`,
		url, time.Now().UTC(),
	)
	return false
}

func (s *SQLiteStore) RecordResult(url string, statusCode int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`UPDATE visited SET status_code = ?, state = 'done', crawled_at = ? WHERE url = ?`,
		statusCode, time.Now().UTC(), url,
	)
	if err != nil {
		return fmt.Errorf("record result for %s: %w", url, err)
	}
	return nil
}

func (s *SQLiteStore) IsVisited(url string) bool {
	_, ok := s.visited.Load(url)
	return ok
}

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
