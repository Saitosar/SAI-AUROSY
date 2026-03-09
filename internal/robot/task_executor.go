package robot

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
)

const (
	defaultNavigateTimeout = 30 * time.Second // matches mall handler waitForNavigation
	defaultReturnTimeout  = 60 * time.Second
)

// TaskExecutor routes task types to the appropriate execution logic.
type TaskExecutor struct {
	navExecutor *NavigationExecutor
}

// NewTaskExecutor creates a new TaskExecutor.
func NewTaskExecutor(navExecutor *NavigationExecutor) *TaskExecutor {
	return &TaskExecutor{
		navExecutor: navExecutor,
	}
}

// ExecuteTask runs the task and returns the result.
// Caller is responsible for updating task status and state transitions.
func (e *TaskExecutor) ExecuteTask(ctx context.Context, task *tasks.Task) TaskExecutionResult {
	switch task.ScenarioID {
	case "navigate_to_store":
		return e.executeNavigateToStore(ctx, task)
	case "navigation":
		return e.executeNavigation(ctx, task)
	default:
		return TaskExecutionResult{Handled: false, Reason: "unsupported scenario"}
	}
}

// TaskExecutionResult holds the outcome of task execution.
type TaskExecutionResult struct {
	Handled   bool
	Success   bool
	Timeout   bool
	Cancelled bool
	Reason    string
}

type navPayload struct {
	TargetCoordinates  string  `json:"target_coordinates"`
	StoreName          string  `json:"store_name"`
	TargetStore        string  `json:"target_store"`
	DestinationNodeID  string  `json:"destination_node_id"`
	EstimatedDistance  float64 `json:"estimated_distance"`
}

func (e *TaskExecutor) executeNavigateToStore(ctx context.Context, task *tasks.Task) TaskExecutionResult {
	var p navPayload
	if len(task.Payload) > 0 {
		_ = json.Unmarshal(task.Payload, &p)
	}
	if p.TargetCoordinates == "" {
		return TaskExecutionResult{Handled: true, Success: false, Reason: "missing target_coordinates"}
	}
	storeName := p.StoreName
	if storeName == "" {
		storeName = p.TargetStore
	}
	req := NavigationRequest{
		RobotID:      task.RobotID,
		TaskID:       task.ID,
		TargetCoords: p.TargetCoordinates,
		StoreName:    storeName,
		DestNodeID:   p.DestinationNodeID,
		OperatorID:   task.OperatorID,
		IsReturn:     false,
		Timeout:      defaultNavigateTimeout,
	}
	res := e.navExecutor.Execute(ctx, req)
	return TaskExecutionResult{
		Handled:   true,
		Success:   res.Success,
		Timeout:   res.Timeout,
		Cancelled: res.Cancelled,
		Reason:    res.Reason,
	}
}

func (e *TaskExecutor) executeNavigation(ctx context.Context, task *tasks.Task) TaskExecutionResult {
	var p struct {
		TargetCoordinates string `json:"target_coordinates"`
	}
	if len(task.Payload) > 0 {
		_ = json.Unmarshal(task.Payload, &p)
	}
	if p.TargetCoordinates == "" {
		return TaskExecutionResult{Handled: true, Success: false, Reason: "missing target_coordinates"}
	}
	req := NavigationRequest{
		RobotID:      task.RobotID,
		TaskID:       task.ID,
		TargetCoords: p.TargetCoordinates,
		OperatorID:   task.OperatorID,
		IsReturn:     true,
		Timeout:      defaultReturnTimeout,
	}
	res := e.navExecutor.Execute(ctx, req)
	return TaskExecutionResult{
		Handled:   true,
		Success:   res.Success,
		Timeout:   res.Timeout,
		Cancelled: res.Cancelled,
		Reason:    res.Reason,
	}
}
