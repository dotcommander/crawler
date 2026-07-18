package session

import "sync"

// MemoryStore is an in-memory VisitedStore backed by sync.Map.
type MemoryStore struct {
	visited sync.Map
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) MarkVisited(url string) bool {
	_, loaded := m.visited.LoadOrStore(url, true)
	return loaded
}

func (m *MemoryStore) RecordResult(_ string, _ int) error {
	return nil
}

func (m *MemoryStore) IsVisited(url string) bool {
	_, ok := m.visited.Load(url)
	return ok
}

func (m *MemoryStore) Close() error {
	return nil
}
