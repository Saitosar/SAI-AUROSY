package sdk

import (
	"encoding/json"
	"time"
)

// Robot represents a robot in the fleet.
type Robot struct {
	ID               string    `json:"id"`
	Vendor           string    `json:"vendor"`
	Model            string    `json:"model"`
	AdapterEndpoint  string    `json:"adapter_endpoint"`
	TenantID         string    `json:"tenant_id"`
	EdgeID           string    `json:"edge_id,omitempty"`
	Capabilities     []string  `json:"capabilities"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TaskStatus is the task execution status.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents a robot task.
type Task struct {
	ID          string          `json:"id"`
	RobotID     string          `json:"robot_id"`
	TenantID    string          `json:"tenant_id,omitempty"`
	Type        string          `json:"type"`
	ScenarioID  string          `json:"scenario_id,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Status      TaskStatus      `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	OperatorID  string          `json:"operator_id,omitempty"`
}

// CreateTaskRequest is the request body for creating a task.
type CreateTaskRequest struct {
	RobotID    string          `json:"robot_id"`
	ScenarioID string          `json:"scenario_id"`
	Type       string          `json:"type,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	OperatorID string          `json:"operator_id,omitempty"`
}

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

// CreateWebhookRequest is the request body for creating a webhook.
type CreateWebhookRequest struct {
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Secret  string   `json:"secret,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}

// UpdateWebhookRequest is the request body for updating a webhook.
type UpdateWebhookRequest struct {
	URL     string   `json:"url,omitempty"`
	Events  []string `json:"events,omitempty"`
	Secret  string   `json:"secret,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}
