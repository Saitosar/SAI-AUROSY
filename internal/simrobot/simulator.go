package simrobot

import (
	"math"
	"time"
)

const (
	defaultArrivalThresholdM = 0.5
)

// Tick advances the simulation by one step: applies failure injection, updates movement, publishes telemetry.
func Tick(robot *SimRobot) {
	robot.mu.Lock()
	defer robot.mu.Unlock()

	state := &robot.state
	state.TickCount++
	state.UpdatedAt = time.Now()

	// Check failure injection before movement
	if robot.failureConfig != nil {
		if applied := applyFailure(robot); applied {
			publishTelemetry(robot)
			return
		}
	}

	// Skip movement if not navigating or in safe_stop
	if state.Mode != ModeNavigating && state.Mode != ModeReturning {
		publishTelemetry(robot)
		return
	}

	// Move toward target
	dist := distance(state.Position, state.TargetPosition)
	speed := state.Speed
	if speed <= 0 {
		speed = defaultSpeed
	}

	// Tick interval is 500ms, so we move speed * 0.5 meters per tick
	step := speed * 0.5
	if step > dist {
		step = dist
	}

	if dist > defaultArrivalThresholdM {
		// Move toward target
		dx := state.TargetPosition.X - state.Position.X
		dy := state.TargetPosition.Y - state.Position.Y
		norm := math.Sqrt(dx*dx + dy*dy)
		if norm > 0 {
			state.Position.X += (dx / norm) * step
			state.Position.Y += (dy / norm) * step
		}
		state.DistanceToTarget = distance(state.Position, state.TargetPosition)
	} else {
		// Arrived
		state.Position = state.TargetPosition
		state.DistanceToTarget = 0
		state.Mode = ModeArrived
	}

	publishTelemetry(robot)
}

// publishTelemetry builds and publishes telemetry. Caller must hold robot.mu.
func publishTelemetry(robot *SimRobot) {
	t := buildTelemetryFromState(robot.state)
	_ = robot.bus.PublishTelemetry(t)
}
