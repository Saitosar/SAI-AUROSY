package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// MemoryStore is an in-memory scenario store with built-in defaults.
type MemoryStore struct {
	mu        sync.RWMutex
	scenarios map[string]Scenario
}

// NewMemoryStore creates a new in-memory scenario store with built-in scenarios.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		scenarios: make(map[string]Scenario),
	}
	// Seed built-in scenarios (same as catalog registerDefaults)
	s.scenarios["standby"] = Scenario{
		ID:                  "standby",
		Name:                "Ожидание",
		Description:         "Стоячая поза",
		RequiredCapabilities: []string{hal.CapStand},
		Steps:               []ScenarioStep{{Command: "stand_mode", DurationSec: 0}},
	}
	patrolPayload1, _ := json.Marshal(map[string]float64{"linear_x": 0.3, "linear_y": 0, "angular_z": 0})
	patrolPayload2, _ := json.Marshal(map[string]float64{"linear_x": 0, "linear_y": 0, "angular_z": 0})
	s.scenarios["patrol"] = Scenario{
		ID:                  "patrol",
		Name:                "Патруль",
		Description:         "walk_mode + cmd_vel N сек",
		RequiredCapabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol},
		Steps: []ScenarioStep{
			{Command: "walk_mode", DurationSec: 0},
			{Command: "cmd_vel", Payload: patrolPayload1, DurationSec: -1},
			{Command: "cmd_vel", Payload: patrolPayload2, DurationSec: 0},
		},
	}
	s.scenarios["navigation"] = Scenario{
		ID:                  "navigation",
		Name:                "Навигация",
		Description:         "walk_mode + движение по параметрам",
		RequiredCapabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapNavigation},
		Steps: []ScenarioStep{
			{Command: "walk_mode", DurationSec: 0},
			{Command: "cmd_vel", Payload: nil, DurationSec: -1},
		},
	}
	return s
}

// List returns all scenarios.
func (s *MemoryStore) List(ctx context.Context) ([]Scenario, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Scenario, 0, len(s.scenarios))
	for _, sc := range s.scenarios {
		out = append(out, sc)
	}
	return out, nil
}

// ListByTenant returns scenarios. MemoryStore has no tenant_id; all scenarios are shared.
func (s *MemoryStore) ListByTenant(ctx context.Context, tenantID string) ([]Scenario, error) {
	_ = tenantID
	return s.List(ctx)
}

// Get returns a scenario by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Scenario, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.scenarios[id]
	if !ok {
		return nil, nil
	}
	cp := sc
	return &cp, nil
}

// GetByTenant returns a scenario by ID. MemoryStore has no tenant_id; all scenarios are shared.
func (s *MemoryStore) GetByTenant(ctx context.Context, id, tenantID string) (*Scenario, error) {
	_ = tenantID
	return s.Get(ctx, id)
}

// Create adds a new scenario.
func (s *MemoryStore) Create(ctx context.Context, sc *Scenario) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scenarios[sc.ID]; ok {
		return errors.New("scenario already exists")
	}
	s.scenarios[sc.ID] = *sc
	return nil
}

// Update updates an existing scenario.
func (s *MemoryStore) Update(ctx context.Context, sc *Scenario) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scenarios[sc.ID]; !ok {
		return ErrNotFound
	}
	s.scenarios[sc.ID] = *sc
	return nil
}

// Delete removes a scenario by ID.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scenarios[id]; !ok {
		return ErrNotFound
	}
	delete(s.scenarios, id)
	return nil
}
