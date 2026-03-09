package edges

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/pkg/hal"
)

// MemoryStore is an in-memory edge store for when DB is not configured.
type MemoryStore struct {
	mu     sync.RWMutex
	edges  map[string]*Edge
	cmds   []pendingCommand
}

type pendingCommand struct {
	id       string
	edgeID   string
	robotID  string
	cmd      hal.Command
	createdAt time.Time
	ackedAt  *time.Time
}

// NewMemoryStore creates a new in-memory edge store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		edges: make(map[string]*Edge),
		cmds:  nil,
	}
}

// UpsertEdge creates or updates an edge.
func (s *MemoryStore) UpsertEdge(ctx context.Context, e *Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	e.UpdatedAt = now
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	s.edges[e.ID] = e
	return nil
}

// GetEdge returns an edge by ID.
func (s *MemoryStore) GetEdge(ctx context.Context, id string) (*Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.edges[id]
	if !ok {
		return nil, nil
	}
	cp := *e
	return &cp, nil
}

// ListEdges returns all edges.
func (s *MemoryStore) ListEdges(ctx context.Context) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Edge, 0, len(s.edges))
	for _, e := range s.edges {
		out = append(out, *e)
	}
	return out, nil
}

// EnqueueCommand adds a command to the queue.
func (s *MemoryStore) EnqueueCommand(ctx context.Context, edgeID, robotID string, cmd *hal.Command) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cmds = append(s.cmds, pendingCommand{
		id:        uuid.New().String(),
		edgeID:    edgeID,
		robotID:   robotID,
		cmd:       *cmd,
		createdAt: time.Now(),
	})
	return nil
}

// FetchAndAckPendingCommands returns and acks pending commands.
func (s *MemoryStore) FetchAndAckPendingCommands(ctx context.Context, edgeID string) ([]hal.Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var out []hal.Command
	for i := range s.cmds {
		if s.cmds[i].edgeID == edgeID && s.cmds[i].ackedAt == nil {
			s.cmds[i].ackedAt = &now
			out = append(out, s.cmds[i].cmd)
		}
	}
	return out, nil
}
