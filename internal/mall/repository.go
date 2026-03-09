package mall

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

var (
	ErrMallNotFound  = errors.New("mall not found")
	ErrPathNotFound  = errors.New("path not found")
)

// Repository provides access to mall map data.
type Repository interface {
	GetMap(ctx context.Context, mallID string) (*MallMap, error)
}

// MemoryRepository holds mall maps in memory. Loads from JSON file on creation.
type MemoryRepository struct {
	mu   sync.RWMutex
	maps map[string]*MallMap
}

// NewMemoryRepository creates a repository and loads the default mall from path.
// If path is empty or file cannot be read, a minimal default map is used.
func NewMemoryRepository(jsonPath string) *MemoryRepository {
	r := &MemoryRepository{
		maps: make(map[string]*MallMap),
	}
	if jsonPath != "" {
		if data, err := os.ReadFile(jsonPath); err == nil {
			var m MallMap
			if json.Unmarshal(data, &m) == nil && m.ID != "" {
				r.maps[m.ID] = &m
			}
		}
	}
	if len(r.maps) == 0 {
		r.maps["default"] = defaultMallMap()
	}
	return r
}

// GetMap returns the mall map for the given ID.
func (r *MemoryRepository) GetMap(ctx context.Context, mallID string) (*MallMap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.maps[mallID]
	if !ok || m == nil {
		return nil, ErrMallNotFound
	}
	return m, nil
}

// defaultMallMap returns a minimal fallback map when no JSON is loaded.
func defaultMallMap() *MallMap {
	return &MallMap{
		ID:    "default",
		Name:  "Default Mall",
		BasePoint: "node-standby",
		Floors: []Floor{{ID: "floor-1", Name: "Ground Floor", Level: 0}},
		Nodes: []NavNode{
			{ID: "node-standby", Name: "Standby", FloorID: "floor-1", Zone: "A", Type: "standby", Coordinates: Coordinates{X: 0, Y: 0}},
			{ID: "node-entrance", Name: "Entrance", FloorID: "floor-1", Zone: "A", Type: "entrance", Coordinates: Coordinates{X: 0, Y: 5}},
		},
		Edges: []NavEdge{
			{From: "node-standby", To: "node-entrance", Distance: 5, Bidirectional: true},
		},
		Stores: []StoreLocation{},
	}
}
