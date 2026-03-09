package conversations

import (
	"context"
	"fmt"
	"strings"
)

// Catalog provides access to conversations (intent-to-response).
type Catalog struct {
	store Store
}

// NewCatalog creates a catalog backed by a store.
func NewCatalog(store Store) *Catalog {
	return &Catalog{store: store}
}

// Get returns a conversation by ID.
func (c *Catalog) Get(ctx context.Context, id string) (*Conversation, error) {
	return c.GetForTenant(ctx, id, "")
}

// GetForTenant returns a conversation by ID if visible to the tenant.
func (c *Catalog) GetForTenant(ctx context.Context, id, tenantID string) (*Conversation, error) {
	if c.store == nil {
		return nil, ErrNotFound
	}
	return c.store.GetByTenant(ctx, id, tenantID)
}

// GetByIntent returns a conversation by intent name for the tenant.
// Tenant-specific conversations take precedence over shared (tenant_id IS NULL).
func (c *Catalog) GetByIntent(ctx context.Context, intent, tenantID string) (*Conversation, error) {
	if c.store == nil {
		return nil, ErrNotFound
	}
	return c.store.GetByIntent(ctx, intent, tenantID)
}

// List returns all conversations.
func (c *Catalog) List(ctx context.Context) ([]Conversation, error) {
	return c.ListForTenant(ctx, "")
}

// ListForTenant returns conversations visible to the tenant.
func (c *Catalog) ListForTenant(ctx context.Context, tenantID string) ([]Conversation, error) {
	if c.store == nil {
		return nil, nil
	}
	return c.store.ListByTenant(ctx, tenantID)
}

// Create adds a new conversation.
func (c *Catalog) Create(ctx context.Context, conv *Conversation) error {
	if c.store == nil {
		return ErrNotFound
	}
	return c.store.Create(ctx, conv)
}

// Update updates an existing conversation.
func (c *Catalog) Update(ctx context.Context, conv *Conversation) error {
	if c.store == nil {
		return ErrNotFound
	}
	return c.store.Update(ctx, conv)
}

// Delete removes a conversation by ID.
func (c *Catalog) Delete(ctx context.Context, id string) error {
	if c.store == nil {
		return ErrNotFound
	}
	return c.store.Delete(ctx, id)
}

// ResolveResponse fills the response template with intent parameters.
// Placeholders like {{brand}} are replaced with values from params.
func ResolveResponse(template string, params map[string]interface{}, language string) string {
	if template == "" {
		return ""
	}
	result := template
	for k, v := range params {
		placeholder := "{{" + k + "}}"
		var s string
		switch x := v.(type) {
		case string:
			s = x
		case float64:
			s = fmt.Sprintf("%v", x)
		case int:
			s = fmt.Sprintf("%d", x)
		default:
			s = fmt.Sprintf("%v", v)
		}
		result = strings.ReplaceAll(result, placeholder, s)
	}
	return result
}
