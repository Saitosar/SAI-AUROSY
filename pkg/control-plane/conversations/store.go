package conversations

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("conversation not found")
	ErrAlreadyExists = errors.New("conversation already exists")
)

// Store is the conversation store interface.
type Store interface {
	List(ctx context.Context) ([]Conversation, error)
	Get(ctx context.Context, id string) (*Conversation, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Conversation, error)
	GetByTenant(ctx context.Context, id, tenantID string) (*Conversation, error)
	GetByIntent(ctx context.Context, intent, tenantID string) (*Conversation, error)
	Create(ctx context.Context, c *Conversation) error
	Update(ctx context.Context, c *Conversation) error
	Delete(ctx context.Context, id string) error
}
