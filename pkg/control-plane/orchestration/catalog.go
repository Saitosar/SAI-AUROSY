package orchestration

import "encoding/json"

// Catalog provides predefined workflows.
type Catalog struct {
	workflows map[string]Workflow
}

// NewCatalog creates a catalog with default workflows.
func NewCatalog() *Catalog {
	c := &Catalog{workflows: make(map[string]Workflow)}
	c.registerDefaults()
	return c
}

func (c *Catalog) registerDefaults() {
	// patrol_zones_ABC: 3 robots patrol zones A, B, C (requires 3 robots with patrol capability)
	c.workflows["patrol_zones_ABC"] = Workflow{
		ID:          "patrol_zones_ABC",
		Name:        "Патруль зон A, B, C",
		Description: "3 робота патрулируют зоны A, B, C",
		Steps: []WorkflowStep{
			{ScenarioID: "patrol", ZoneID: "A", Payload: mustMarshal(map[string]any{"zone_id": "A", "duration_sec": 30})},
			{ScenarioID: "patrol", ZoneID: "B", Payload: mustMarshal(map[string]any{"zone_id": "B", "duration_sec": 30})},
			{ScenarioID: "patrol", ZoneID: "C", Payload: mustMarshal(map[string]any{"zone_id": "C", "duration_sec": 30})},
		},
	}
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// Get returns a workflow by ID.
func (c *Catalog) Get(id string) (Workflow, bool) {
	w, ok := c.workflows[id]
	return w, ok
}

// List returns all workflows.
func (c *Catalog) List() []Workflow {
	out := make([]Workflow, 0, len(c.workflows))
	for _, w := range c.workflows {
		out = append(out, w)
	}
	return out
}
