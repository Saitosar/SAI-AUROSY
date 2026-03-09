package robot

import (
	"time"
)

// Event type constants for robot lifecycle.
const (
	EventRobotStateChanged     = "robot_state_changed"
	EventNavigationStarted    = "navigation_started"
	EventNavigationProgress   = "navigation_progress"
	EventNavigationCompleted  = "navigation_completed"
	EventNavigationFailed     = "navigation_failed"
	EventRobotReturningToBase = "robot_returning_to_base"
	EventRobotIdle            = "robot_idle"
)

// StateChangedPayload builds payload for robot_state_changed.
func StateChangedPayload(robotID string, state, previousState RobotState, taskID string) map[string]any {
	return map[string]any{
		"robot_id":       robotID,
		"state":          string(state),
		"previous_state": string(previousState),
		"task_id":        taskID,
		"timestamp":      time.Now().UTC(),
	}
}

// NavigationStartedPayload builds payload for navigation_started.
func NavigationStartedPayload(robotID, taskID, destination, targetStore string) map[string]any {
	return map[string]any{
		"robot_id":      robotID,
		"task_id":       taskID,
		"destination":   destination,
		"target_store":  targetStore,
		"timestamp":     time.Now().UTC(),
	}
}

// NavigationProgressPayload builds payload for navigation_progress.
func NavigationProgressPayload(robotID, taskID string, distanceRemaining float64) map[string]any {
	return map[string]any{
		"robot_id":            robotID,
		"task_id":             taskID,
		"distance_remaining":  distanceRemaining,
		"timestamp":          time.Now().UTC(),
	}
}

// NavigationCompletedPayload builds payload for navigation_completed.
func NavigationCompletedPayload(robotID, taskID, destination string) map[string]any {
	return map[string]any{
		"robot_id":     robotID,
		"task_id":      taskID,
		"destination":  destination,
		"timestamp":    time.Now().UTC(),
	}
}

// NavigationFailedPayload builds payload for navigation_failed.
func NavigationFailedPayload(robotID, taskID, reason string) map[string]any {
	return map[string]any{
		"robot_id":  robotID,
		"task_id":   taskID,
		"reason":    reason,
		"timestamp": time.Now().UTC(),
	}
}

// RobotReturningToBasePayload builds payload for robot_returning_to_base.
func RobotReturningToBasePayload(robotID, taskID string) map[string]any {
	return map[string]any{
		"robot_id":  robotID,
		"task_id":   taskID,
		"timestamp": time.Now().UTC(),
	}
}

// RobotIdlePayload builds payload for robot_idle.
func RobotIdlePayload(robotID string) map[string]any {
	return map[string]any{
		"robot_id":  robotID,
		"timestamp": time.Now().UTC(),
	}
}
