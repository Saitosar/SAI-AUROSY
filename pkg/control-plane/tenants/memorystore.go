package tenants

import (
	"errors"
	"sync"
)

// MemoryStore is an in-memory tenant store.
type MemoryStore struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

// NewMemoryStore creates a new in-memory tenant store with the default tenant.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		tenants: make(map[string]*Tenant),
	}
	s.tenants["default"] = &Tenant{ID: "default", Name: "Default"}
	return s
}

// List returns all tenants.
func (s *MemoryStore) List() ([]Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, *t)
	}
	return out, nil
}

// Get returns a tenant by ID.
func (s *MemoryStore) Get(id string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

// Create adds a new tenant.
func (s *MemoryStore) Create(t *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[t.ID]; ok {
		return errors.New("tenant already exists")
	}
	cp := *t
	s.tenants[t.ID] = &cp
	return nil
}

// Update updates an existing tenant.
func (s *MemoryStore) Update(t *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[t.ID]; !ok {
		return ErrNotFound
	}
	cp := *t
	s.tenants[t.ID] = &cp
	return nil
}

// Delete removes a tenant by ID.
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[id]; !ok {
		return ErrNotFound
	}
	delete(s.tenants, id)
	return nil
}
