package audit

import (
	"context"
	"time"
)

// Entry represents a single audit log entry.
type Entry struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Details    string    `json:"details,omitempty"`
	TenantID   string    `json:"tenant_id,omitempty"`
}

// ListFilters filters audit log entries.
type ListFilters struct {
	RobotID  string
	TenantID string
	Actor    string
	Action   string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

// Store is the interface for audit log persistence.
type Store interface {
	Append(ctx context.Context, e *Entry) error
	List(ctx context.Context, f ListFilters) ([]*Entry, error)
}
