package tenants

import "encoding/json"

// Tenant represents a tenant in the multi-tenant platform.
type Tenant struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
}
