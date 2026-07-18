package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	defer s.Close()

	assert.False(t, s.MarkVisited("https://example.com"))
	assert.True(t, s.MarkVisited("https://example.com"))
	assert.True(t, s.IsVisited("https://example.com"))
	assert.False(t, s.IsVisited("https://other.com"))
	assert.NoError(t, s.RecordResult("https://example.com", 200))
}

func TestSQLiteStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	t.Run("fresh session", func(t *testing.T) {
		t.Parallel()
		s, err := NewSQLiteStore(dir, "https://example.com/fresh", false)
		require.NoError(t, err)

		assert.False(t, s.MarkVisited("https://example.com/page1"))
		assert.True(t, s.MarkVisited("https://example.com/page1"))
		assert.True(t, s.IsVisited("https://example.com/page1"))
		assert.False(t, s.IsVisited("https://example.com/page2"))

		require.NoError(t, s.RecordResult("https://example.com/page1", 200))
		require.NoError(t, s.Close())
	})

	t.Run("resume session", func(t *testing.T) {
		t.Parallel()
		resumeDir := filepath.Join(dir, "resume")

		s1, err := NewSQLiteStore(resumeDir, "https://resume.com", false)
		require.NoError(t, err)
		s1.MarkVisited("https://resume.com/a")
		require.NoError(t, s1.RecordResult("https://resume.com/a", 200))
		s1.MarkVisited("https://resume.com/b")
		require.NoError(t, s1.RecordResult("https://resume.com/b", 200))
		require.NoError(t, s1.Close())

		s2, err := NewSQLiteStore(resumeDir, "https://resume.com", true)
		require.NoError(t, err)
		defer s2.Close()

		assert.True(t, s2.IsVisited("https://resume.com/a"))
		assert.True(t, s2.IsVisited("https://resume.com/b"))
		assert.False(t, s2.IsVisited("https://resume.com/c"))
	})

	t.Run("resume re-queues in-flight visits", func(t *testing.T) {
		t.Parallel()
		partialDir := filepath.Join(dir, "partial")

		s1, err := NewSQLiteStore(partialDir, "https://partial.com", false)
		require.NoError(t, err)
		// done: marked + recorded
		s1.MarkVisited("https://partial.com/done")
		require.NoError(t, s1.RecordResult("https://partial.com/done", 200))
		// in-flight: marked but crawl killed before RecordResult
		s1.MarkVisited("https://partial.com/inflight")
		require.NoError(t, s1.Close())

		s2, err := NewSQLiteStore(partialDir, "https://partial.com", true)
		require.NoError(t, err)
		defer s2.Close()

		// done rows are treated as visited on resume
		assert.True(t, s2.IsVisited("https://partial.com/done"))
		// in-flight rows are NOT treated as visited, so they get re-crawled
		assert.False(t, s2.IsVisited("https://partial.com/inflight"))
	})

	t.Run("fresh clears previous data", func(t *testing.T) {
		t.Parallel()
		freshDir := filepath.Join(dir, "fresh-clear")

		s1, err := NewSQLiteStore(freshDir, "https://fresh.com", false)
		require.NoError(t, err)
		s1.MarkVisited("https://fresh.com/old")
		require.NoError(t, s1.Close())

		s2, err := NewSQLiteStore(freshDir, "https://fresh.com", false)
		require.NoError(t, err)
		defer s2.Close()

		assert.False(t, s2.IsVisited("https://fresh.com/old"))
	})
}

func TestGetSessionsDir(t *testing.T) {
	t.Run("custom env", func(t *testing.T) {
		t.Setenv("CRAWLER_SESSIONS_DIR", "/tmp/test-sessions")
		assert.Equal(t, "/tmp/test-sessions", GetSessionsDir())
	})

	t.Run("default has sessions suffix", func(t *testing.T) {
		t.Setenv("CRAWLER_SESSIONS_DIR", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		dir := GetSessionsDir()
		assert.Contains(t, dir, "sessions")
	})
}

func TestSQLiteStoreDBFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	s, err := NewSQLiteStore(dir, "https://example.com", false)
	require.NoError(t, err)
	defer s.Close()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	found := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".db" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a .db file in sessions dir")
}
