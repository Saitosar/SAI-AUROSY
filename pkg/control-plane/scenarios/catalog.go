package scenarios

import (
	"encoding/json"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// ScenarioStep represents a single step in a scenario.
type ScenarioStep struct {
	Command     string          `json:"command"`      // walk_mode, stand_mode, cmd_vel
	Payload     json.RawMessage `json:"payload"`       // optional; for cmd_vel: {linear_x, linear_y, angular_z}
	DurationSec int             `json:"duration_sec"` // seconds to hold/execute; 0 = instant
}

// Scenario represents a predefined scenario.
type Scenario struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	Steps                []ScenarioStep `json:"steps"`
	RequiredCapabilities  []string       `json:"required_capabilities"`
}

// Catalog provides access to predefined scenarios.
type Catalog struct {
	scenarios map[string]Scenario
}

// NewCatalog creates a catalog with predefined scenarios.
func NewCatalog() *Catalog {
	c := &Catalog{
		scenarios: make(map[string]Scenario),
	}
	c.registerDefaults()
	return c
}

func (c *Catalog) registerDefaults() {
	// standby: stand_mode only
	c.scenarios["standby"] = Scenario{
		ID:                  "standby",
		Name:                "Ожидание",
		Description:         "Стоячая поза",
		RequiredCapabilities: []string{hal.CapStand},
		Steps: []ScenarioStep{
			{Command: "stand_mode", DurationSec: 0},
		},
	}

	// patrol: walk_mode -> cmd_vel(0.3) for duration -> cmd_vel(0) to stop
	// duration_sec comes from task payload, default 30
	c.scenarios["patrol"] = Scenario{
		ID:                  "patrol",
		Name:                "Патруль",
		Description:         "walk_mode + cmd_vel N сек",
		RequiredCapabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol},
		Steps: []ScenarioStep{
			{Command: "walk_mode", DurationSec: 0},
			{
				Command:     "cmd_vel",
				Payload:     mustMarshal(map[string]float64{"linear_x": 0.3, "linear_y": 0, "angular_z": 0}),
				DurationSec: -1, // from task payload
			},
			{
				Command:     "cmd_vel",
				Payload:     mustMarshal(map[string]float64{"linear_x": 0, "linear_y": 0, "angular_z": 0}),
				DurationSec: 0,
			},
		},
	}

	// navigation: walk_mode -> cmd_vel from task payload for duration from task
	c.scenarios["navigation"] = Scenario{
		ID:                  "navigation",
		Name:                "Навигация",
		Description:         "walk_mode + движение по параметрам",
		RequiredCapabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapNavigation},
		Steps: []ScenarioStep{
			{Command: "walk_mode", DurationSec: 0},
			{
				Command:     "cmd_vel",
				Payload:     nil, // from task payload
				DurationSec: -1,  // from task payload
			},
		},
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// Get returns a scenario by ID.
func (c *Catalog) Get(id string) (Scenario, bool) {
	s, ok := c.scenarios[id]
	return s, ok
}

// List returns all scenarios.
func (c *Catalog) List() []Scenario {
	out := make([]Scenario, 0, len(c.scenarios))
	for _, s := range c.scenarios {
		out = append(out, s)
	}
	return out
}
