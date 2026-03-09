package conversations

import (
	"context"
	"sync"
)

// MemoryStore is an in-memory conversation store.
type MemoryStore struct {
	mu           sync.RWMutex
	byID         map[string]Conversation
	byIntent     map[string]string // intent -> id
}

// NewMemoryStore creates a new in-memory conversation store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID:     make(map[string]Conversation),
		byIntent: make(map[string]string),
	}
}

// List returns all conversations.
func (s *MemoryStore) List(ctx context.Context) ([]Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Conversation, 0, len(s.byID))
	for _, c := range s.byID {
		out = append(out, c)
	}
	return out, nil
}

// ListByTenant returns conversations. MemoryStore has no tenant_id; all are shared.
func (s *MemoryStore) ListByTenant(ctx context.Context, tenantID string) ([]Conversation, error) {
	_ = tenantID
	return s.List(ctx)
}

// Get returns a conversation by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	if !ok {
		return nil, nil
	}
	cp := c
	return &cp, nil
}

// GetByTenant returns a conversation by ID. MemoryStore has no tenant_id.
func (s *MemoryStore) GetByTenant(ctx context.Context, id, tenantID string) (*Conversation, error) {
	_ = tenantID
	return s.Get(ctx, id)
}

// GetByIntent returns a conversation by intent. First checks tenant-specific, then shared.
func (s *MemoryStore) GetByIntent(ctx context.Context, intent, tenantID string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Linear scan: prefer tenant-specific, then shared
	var tenantMatch, sharedMatch *Conversation
	for _, c := range s.byID {
		if c.Intent != intent {
			continue
		}
		if c.TenantID == tenantID && tenantID != "" {
			cp := c
			tenantMatch = &cp
			break
		}
		if c.TenantID == "" {
			cp := c
			sharedMatch = &cp
		}
	}
	if tenantMatch != nil {
		return tenantMatch, nil
	}
	return sharedMatch, nil
}

// Create adds a new conversation.
func (s *MemoryStore) Create(ctx context.Context, c *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[c.ID]; ok {
		return ErrAlreadyExists
	}
	s.byID[c.ID] = *c
	s.byIntent[c.Intent] = c.ID
	return nil
}

// Update updates an existing conversation.
func (s *MemoryStore) Update(ctx context.Context, c *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.byID[c.ID]
	if !ok {
		return ErrNotFound
	}
	if old.Intent != c.Intent {
		delete(s.byIntent, old.Intent)
	}
	s.byID[c.ID] = *c
	s.byIntent[c.Intent] = c.ID
	return nil
}

// Delete removes a conversation by ID.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.byID, id)
	delete(s.byIntent, c.Intent)
	return nil
}
