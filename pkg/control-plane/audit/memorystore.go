package audit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory audit store for use when no database is configured.
type MemoryStore struct {
	mu    sync.RWMutex
	entries []*Entry
}

// NewMemoryStore creates a new in-memory audit store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: make([]*Entry, 0, 100)}
}

// Append adds an audit entry.
func (s *MemoryStore) Append(ctx context.Context, e *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	s.entries = append(s.entries, e)
	return nil
}

// List returns audit entries matching the filters.
func (s *MemoryStore) List(ctx context.Context, f ListFilters) ([]*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Entry
	for i := len(s.entries) - 1; i >= 0 && len(result) < 1000; i-- {
		e := s.entries[i]
		if f.RobotID != "" && (e.Resource != "robot" || e.ResourceID != f.RobotID) {
			continue
		}
		if f.Actor != "" && e.Actor != f.Actor {
			continue
		}
		if f.Action != "" && e.Action != f.Action {
			continue
		}
		if f.From != nil && e.Timestamp.Before(*f.From) {
			continue
		}
		if f.To != nil && e.Timestamp.After(*f.To) {
			continue
		}
		result = append(result, e)
	}

	limit := 100
	if f.Limit > 0 {
		limit = f.Limit
	}
	offset := f.Offset
	if offset >= len(result) {
		return []*Entry{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}
