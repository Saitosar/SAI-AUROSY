package robot

import "time"

// RobotState represents the current operational state of a robot.
type RobotState string

const (
	StateIdle              RobotState = "IDLE"
	StateGreetingVisitor  RobotState = "GREETING_VISITOR"
	StateProcessingRequest RobotState = "PROCESSING_REQUEST"
	StatePlanningRoute    RobotState = "PLANNING_ROUTE"
	StateNavigating       RobotState = "NAVIGATING_TO_STORE"
	StateArrived          RobotState = "ARRIVED_AT_DESTINATION"
	StateReturning        RobotState = "RETURNING_TO_BASE"
	StateError            RobotState = "ERROR_STATE"
)

// RobotContext holds the execution context for a robot.
type RobotContext struct {
	RobotID         string
	CurrentState    RobotState
	PreviousState  RobotState
	CurrentTaskID   string
	TargetStore     string
	DestinationNode string
	TargetCoords    string
	StatusMessage   string
	UpdatedAt       time.Time
}

// CanTransition returns true if transitioning from fromState to toState is valid.
func CanTransition(fromState, toState RobotState) bool {
	if toState == StateError {
		return true // any state can transition to ERROR_STATE
	}
	switch fromState {
	case StateIdle:
		return toState == StateGreetingVisitor || toState == StateNavigating || toState == StateReturning
	case StateGreetingVisitor:
		return toState == StateProcessingRequest
	case StateProcessingRequest:
		return toState == StatePlanningRoute
	case StatePlanningRoute:
		return toState == StateNavigating
	case StateNavigating:
		return toState == StateArrived || toState == StateIdle
	case StateArrived:
		return toState == StateReturning || toState == StateIdle
	case StateReturning:
		return toState == StateIdle
	case StateError:
		return toState == StateReturning || toState == StateIdle
	default:
		return false
	}
}

// IsExecutionState returns true if the state is managed by the Execution Engine.
func IsExecutionState(s RobotState) bool {
	switch s {
	case StateIdle, StateNavigating, StateArrived, StateReturning, StateError:
		return true
	default:
		return false
	}
}
