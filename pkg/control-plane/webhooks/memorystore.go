package webhooks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory webhook store for use when no database is configured.
type MemoryStore struct {
	mu       sync.RWMutex
	webhooks map[string]*Webhook
}

// NewMemoryStore creates a new in-memory webhook store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{webhooks: make(map[string]*Webhook)}
}

// Create adds a new webhook.
func (s *MemoryStore) Create(ctx context.Context, w *Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	s.webhooks[w.ID] = cloneWebhook(w)
	return nil
}

// Get returns a webhook by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if w, ok := s.webhooks[id]; ok {
		return cloneWebhook(w), nil
	}
	return nil, nil
}

// List returns all webhooks.
func (s *MemoryStore) List(ctx context.Context) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Webhook, 0, len(s.webhooks))
	for _, w := range s.webhooks {
		list = append(list, cloneWebhook(w))
	}
	return list, nil
}

// ListByEvent returns webhooks subscribed to the given event.
func (s *MemoryStore) ListByEvent(ctx context.Context, event string) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*Webhook
	for _, w := range s.webhooks {
		if !w.Enabled {
			continue
		}
		for _, e := range w.Events {
			if e == event {
				list = append(list, cloneWebhook(w))
				break
			}
		}
	}
	return list, nil
}

// Update updates a webhook.
func (s *MemoryStore) Update(ctx context.Context, w *Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.webhooks[w.ID]; !ok {
		return nil
	}
	w.UpdatedAt = time.Now()
	s.webhooks[w.ID] = cloneWebhook(w)
	return nil
}

// Delete removes a webhook.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.webhooks, id)
	return nil
}

func cloneWebhook(w *Webhook) *Webhook {
	events := make([]string, len(w.Events))
	copy(events, w.Events)
	return &Webhook{
		ID:        w.ID,
		URL:       w.URL,
		Events:    events,
		Secret:    w.Secret,
		Enabled:   w.Enabled,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
