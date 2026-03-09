package orchestration

import (
	"encoding/json"
	"time"
)

// WorkflowStep defines a single step in a workflow (one task per robot).
type WorkflowStep struct {
	RobotID       string          `json:"robot_id,omitempty"`       // concrete robot
	RobotSelector *RobotSelector   `json:"robot_selector,omitempty"` // selector when robot_id not set
	ScenarioID    string          `json:"scenario_id"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	ZoneID        string          `json:"zone_id,omitempty"`
}

// RobotSelector selects a robot by capabilities and preferences.
type RobotSelector struct {
	Capabilities   []string `json:"capabilities,omitempty"`
	ZonePreference string   `json:"zone_preference,omitempty"`
}

// Workflow represents a multi-robot workflow definition.
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []WorkflowStep `json:"steps"`
}

// WorkflowRunStatus is the status of a workflow run.
type WorkflowRunStatus string

const (
	WorkflowRunPending   WorkflowRunStatus = "pending"
	WorkflowRunRunning   WorkflowRunStatus = "running"
	WorkflowRunCompleted WorkflowRunStatus = "completed"
	WorkflowRunFailed    WorkflowRunStatus = "failed"
	WorkflowRunCancelled WorkflowRunStatus = "cancelled"
)

// WorkflowRunTask links a workflow run to a task.
type WorkflowRunTask struct {
	TaskID string `json:"task_id"`
	StepIndex int  `json:"step_index"`
}

// WorkflowRun represents an instance of a workflow execution.
type WorkflowRun struct {
	ID         string            `json:"id"`
	WorkflowID string            `json:"workflow_id"`
	Status     WorkflowRunStatus  `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Tasks      []WorkflowRunTask `json:"tasks,omitempty"`
}
