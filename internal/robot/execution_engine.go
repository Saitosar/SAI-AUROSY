package robot

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// EventBroadcaster emits lifecycle events. Nil-safe.
type EventBroadcaster interface {
	Broadcast(eventType string, data map[string]any)
}

// RobotStateProvider provides robot execution state for the API.
type RobotStateProvider interface {
	GetRobotState(robotID string) *RobotStateResponse
}

// ExecutionEngineConfig configures the Robot Execution Engine.
type ExecutionEngineConfig struct {
	EventBroadcaster        EventBroadcaster
	TaskStore               tasks.Store
	OnTaskCompleted         func(taskID, robotID, status string)
	TimeoutAsCompletion     bool // when true, navigation timeout is treated as success (backward compat)
}

// RobotExecutionEngine is the central runtime controller for robot tasks.
type RobotExecutionEngine struct {
	stateManager   *StateManager
	taskExecutor   *TaskExecutor
	bus            *telemetry.Bus
	eventBroadcast EventBroadcaster
	taskStore      tasks.Store
	onTaskCompleted func(taskID, robotID, status string)
	timeoutAsCompletion bool
}

// NewExecutionEngine creates a new Robot Execution Engine.
func NewExecutionEngine(
	stateManager *StateManager,
	taskExecutor *TaskExecutor,
	bus *telemetry.Bus,
	cfg ExecutionEngineConfig,
) *RobotExecutionEngine {
	return &RobotExecutionEngine{
		stateManager:       stateManager,
		taskExecutor:       taskExecutor,
		bus:                bus,
		eventBroadcast:     cfg.EventBroadcaster,
		taskStore:         cfg.TaskStore,
		onTaskCompleted:    cfg.OnTaskCompleted,
		timeoutAsCompletion: cfg.TimeoutAsCompletion,
	}
}

// ExecuteTask runs a task. Blocks until completion, cancellation, or failure.
// Returns true if the task was handled by the engine.
func (e *RobotExecutionEngine) ExecuteTask(ctx context.Context, task *tasks.Task) bool {
	result := e.taskExecutor.ExecuteTask(ctx, task)
	if !result.Handled {
		return false
	}

	robotID := task.RobotID
	taskID := task.ID

	// Parse payload for display
	var payload struct {
		TargetCoordinates string `json:"target_coordinates"`
		StoreName          string `json:"store_name"`
		TargetStore        string `json:"target_store"`
	}
	_ = json.Unmarshal(task.Payload, &payload)
	targetStore := payload.StoreName
	if targetStore == "" {
		targetStore = payload.TargetStore
	}
	dest := payload.TargetCoordinates

	switch task.ScenarioID {
	case "navigate_to_store":
		return e.handleNavigateToStoreResult(ctx, robotID, taskID, targetStore, dest, result)
	case "navigation":
		return e.handleNavigationResult(ctx, robotID, taskID, result)
	default:
		return false
	}
}

func (e *RobotExecutionEngine) handleNavigateToStoreResult(ctx context.Context, robotID, taskID, targetStore, dest string, result TaskExecutionResult) bool {
	if result.Success || (result.Timeout && e.timeoutAsCompletion) {
		e.stateManager.Transition(robotID, StateArrived, taskID, targetStore, "", dest, "arrived")
		e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateArrived, StateNavigating, taskID))
		e.broadcast(EventNavigationCompleted, NavigationCompletedPayload(robotID, taskID, dest))
		e.emitTaskCompleted(taskID, robotID, string(tasks.StatusCompleted))
		now := time.Now()
		if e.taskStore != nil {
			_ = e.taskStore.UpdateStatusAndCompletedAt(taskID, tasks.StatusCompleted, now)
		}
		return true
	}
	// Failure
	e.stateManager.Transition(robotID, StateError, taskID, targetStore, "", dest, result.Reason)
	e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateError, StateNavigating, taskID))
	e.broadcast(EventNavigationFailed, NavigationFailedPayload(robotID, taskID, result.Reason))
	e.sendSafeStop(robotID, "")
	status := tasks.StatusFailed
	if result.Cancelled {
		status = tasks.StatusCancelled
	}
	e.emitTaskCompleted(taskID, robotID, string(status))
	now := time.Now()
	if e.taskStore != nil {
		_ = e.taskStore.UpdateStatusAndCompletedAt(taskID, status, now)
	}
	return true
}

func (e *RobotExecutionEngine) handleNavigationResult(ctx context.Context, robotID, taskID string, result TaskExecutionResult) bool {
	if result.Success || (result.Timeout && e.timeoutAsCompletion) {
		e.stateManager.Transition(robotID, StateIdle, taskID, "", "", "", "idle")
		e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateIdle, StateReturning, taskID))
		e.broadcast(EventRobotIdle, RobotIdlePayload(robotID))
		e.emitTaskCompleted(taskID, robotID, string(tasks.StatusCompleted))
		now := time.Now()
		if e.taskStore != nil {
			_ = e.taskStore.UpdateStatusAndCompletedAt(taskID, tasks.StatusCompleted, now)
		}
		return true
	}
	// Failure
	e.stateManager.Transition(robotID, StateError, taskID, "", "", "", result.Reason)
	e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateError, StateReturning, taskID))
	e.broadcast(EventNavigationFailed, NavigationFailedPayload(robotID, taskID, result.Reason))
	e.sendSafeStop(robotID, "")
	status := tasks.StatusFailed
	if result.Cancelled {
		status = tasks.StatusCancelled
	}
	e.emitTaskCompleted(taskID, robotID, string(status))
	now := time.Now()
	if e.taskStore != nil {
		_ = e.taskStore.UpdateStatusAndCompletedAt(taskID, status, now)
	}
	return true
}

// ExecuteNavigateToStore runs a navigate_to_store task. Called from Runner.
// Transitions to NAVIGATING_TO_STORE, executes, handles result.
func (e *RobotExecutionEngine) ExecuteNavigateToStore(ctx context.Context, task *tasks.Task) bool {
	robotID := task.RobotID
	taskID := task.ID

	var payload struct {
		TargetCoordinates string `json:"target_coordinates"`
		StoreName         string `json:"store_name"`
		TargetStore       string `json:"target_store"`
		DestinationNodeID string `json:"destination_node_id"`
	}
	_ = json.Unmarshal(task.Payload, &payload)
	targetStore := payload.StoreName
	if targetStore == "" {
		targetStore = payload.TargetStore
	}

	e.stateManager.Transition(robotID, StateNavigating, taskID, targetStore, payload.DestinationNodeID, payload.TargetCoordinates, "navigating")
	e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateNavigating, StateIdle, taskID))
	e.broadcast(EventNavigationStarted, NavigationStartedPayload(robotID, taskID, payload.TargetCoordinates, targetStore))

	result := e.taskExecutor.ExecuteTask(ctx, task)
	if !result.Handled {
		return false
	}
	return e.handleNavigateToStoreResult(ctx, robotID, taskID, targetStore, payload.TargetCoordinates, result)
}

// ExecuteReturnToBase runs a navigation (return) task. Called from Runner.
func (e *RobotExecutionEngine) ExecuteReturnToBase(ctx context.Context, task *tasks.Task) bool {
	robotID := task.RobotID
	taskID := task.ID

	e.stateManager.Transition(robotID, StateReturning, taskID, "", "", "", "returning to base")
	e.broadcast(EventRobotStateChanged, StateChangedPayload(robotID, StateReturning, StateArrived, taskID))
	e.broadcast(EventRobotReturningToBase, RobotReturningToBasePayload(robotID, taskID))

	result := e.taskExecutor.ExecuteTask(ctx, task)
	if !result.Handled {
		return false
	}
	return e.handleNavigationResult(ctx, robotID, taskID, result)
}

// ExecuteTaskEntry dispatches to the appropriate handler based on scenario.
func (e *RobotExecutionEngine) ExecuteTaskEntry(ctx context.Context, task *tasks.Task) bool {
	switch task.ScenarioID {
	case "navigate_to_store":
		return e.ExecuteNavigateToStore(ctx, task)
	case "navigation":
		return e.ExecuteReturnToBase(ctx, task)
	default:
		return false
	}
}

func (e *RobotExecutionEngine) broadcast(eventType string, data map[string]any) {
	if e.eventBroadcast != nil {
		e.eventBroadcast.Broadcast(eventType, data)
	}
}

func (e *RobotExecutionEngine) emitTaskCompleted(taskID, robotID, status string) {
	if e.onTaskCompleted != nil {
		e.onTaskCompleted(taskID, robotID, status)
	}
}

func (e *RobotExecutionEngine) sendSafeStop(robotID, operatorID string) {
	cmd := &hal.Command{
		RobotID:    robotID,
		Command:    "safe_stop",
		Timestamp:  time.Now(),
		OperatorID: operatorID,
	}
	if arbiter.SafetyAllow(cmd) {
		if err := e.bus.PublishCommand(cmd); err != nil {
			slog.Warn("execution_engine safe_stop failed", "robot_id", robotID, "error", err)
		}
	}
}

// StateManager returns the state manager for API access.
func (e *RobotExecutionEngine) StateManager() *StateManager {
	return e.stateManager
}

// RobotStateResponse is the API response for GET /v1/robots/{id}/state.
type RobotStateResponse struct {
	RobotID       string `json:"robot_id"`
	State         string `json:"state"`
	CurrentTask   string `json:"current_task"`
	Destination   string `json:"destination"`
	TargetStore   string `json:"target_store"`
	StatusMessage string `json:"status_message"`
}

// GetRobotState returns the current execution state for the robot.
func (e *RobotExecutionEngine) GetRobotState(robotID string) *RobotStateResponse {
	ctx := e.stateManager.Get(robotID)
	if ctx == nil {
		return &RobotStateResponse{
			RobotID:       robotID,
			State:         string(StateIdle),
			StatusMessage: "idle",
		}
	}
	return &RobotStateResponse{
		RobotID:       ctx.RobotID,
		State:         string(ctx.CurrentState),
		CurrentTask:   ctx.CurrentTaskID,
		Destination:   ctx.TargetCoords,
		TargetStore:   ctx.TargetStore,
		StatusMessage: ctx.StatusMessage,
	}
}
