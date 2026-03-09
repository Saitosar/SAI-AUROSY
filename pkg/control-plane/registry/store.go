package registry

import (
	"sync"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// Store is the fleet registry store interface.
type Store interface {
	Add(r *hal.Robot)
	Get(id string) *hal.Robot
	List() []hal.Robot
	ListByTenant(tenantID string) []hal.Robot
	Delete(id string) bool
}

// MemoryStore is an in-memory fleet registry store.
type MemoryStore struct {
	mu     sync.RWMutex
	robots map[string]*hal.Robot
}

// NewMemoryStore creates a new in-memory registry store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		robots: make(map[string]*hal.Robot),
	}
}

// Add adds or updates a robot.
func (s *MemoryStore) Add(r *hal.Robot) {
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
func (s *MemoryStore) Get(id string) *hal.Robot {
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
func (s *MemoryStore) List() []hal.Robot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]hal.Robot, 0, len(s.robots))
	for _, r := range s.robots {
		out = append(out, *r)
	}
	return out
}

// ListByTenant returns robots for the given tenant. If tenantID is empty, returns all.
func (s *MemoryStore) ListByTenant(tenantID string) []hal.Robot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]hal.Robot, 0)
	for _, r := range s.robots {
		if tenantID == "" || r.TenantID == tenantID {
			out = append(out, *r)
		}
	}
	return out
}

// Delete removes a robot.
func (s *MemoryStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.robots[id]; ok {
		delete(s.robots, id)
		return true
	}
	return false
}
