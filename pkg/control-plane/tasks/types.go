package tasks

import (
	"encoding/json"
	"time"
)

// Status represents task execution status.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Task represents a robot task.
type Task struct {
	ID          string    `json:"id"`
	RobotID     string    `json:"robot_id"`
	Type        string    `json:"type"` // scenario or command
	ScenarioID  string    `json:"scenario_id,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	OperatorID  string    `json:"operator_id,omitempty"`
}
