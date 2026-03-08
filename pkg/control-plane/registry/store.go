package registry

import (
	"sync"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// Store is an in-memory fleet registry store.
type Store struct {
	mu     sync.RWMutex
	robots map[string]*hal.Robot
}

// NewStore creates a new registry store.
func NewStore() *Store {
	return &Store{
		robots: make(map[string]*hal.Robot),
	}
}

// Add adds or updates a robot.
func (s *Store) Add(r *hal.Robot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	r.UpdatedAt = now
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	s.robots[r.ID] = r
}

// Get returns a robot by ID.
func (s *Store) Get(id string) *hal.Robot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.robots[id]
	if !ok {
		return nil
	}
	cp := *r
	return &cp
}

// List returns all robots.
func (s *Store) List() []hal.Robot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]hal.Robot, 0, len(s.robots))
	for _, r := range s.robots {
		out = append(out, *r)
	}
	return out
}

// Delete removes a robot.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.robots[id]; ok {
		delete(s.robots, id)
		return true
	}
	return false
}
