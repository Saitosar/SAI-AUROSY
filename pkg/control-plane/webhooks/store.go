package webhooks

import (
	"context"
	"time"
)

// Webhook represents a webhook configuration.
type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Event types for webhooks.
const (
	EventRobotOnline    = "robot_online"
	EventTaskCompleted  = "task_completed"
	EventSafeStop       = "safe_stop"
)

// Store is the interface for webhook persistence.
type Store interface {
	Create(ctx context.Context, w *Webhook) error
	Get(ctx context.Context, id string) (*Webhook, error)
	List(ctx context.Context) ([]*Webhook, error)
	Update(ctx context.Context, w *Webhook) error
	Delete(ctx context.Context, id string) error
	ListByEvent(ctx context.Context, event string) ([]*Webhook, error)
}
