package edges

import (
	"context"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// Edge represents an edge node.
type Edge struct {
	ID            string    `json:"id"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	ConfigJSON   string    `json:"config_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Store is the edge store interface.
type Store interface {
	UpsertEdge(ctx context.Context, e *Edge) error
	GetEdge(ctx context.Context, id string) (*Edge, error)
	ListEdges(ctx context.Context) ([]Edge, error)
	EnqueueCommand(ctx context.Context, edgeID, robotID string, cmd *hal.Command) error
	FetchAndAckPendingCommands(ctx context.Context, edgeID string) ([]hal.Command, error)
}
