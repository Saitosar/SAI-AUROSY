package simrobot

import (
	"fmt"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// BuildTelemetry converts the simulated robot state to hal.Telemetry for publishing.
func BuildTelemetry(robot *SimRobot) *hal.Telemetry {
	state := robot.State()
	return buildTelemetryFromState(state)
}

// buildTelemetryFromState builds hal.Telemetry from SimState.
func buildTelemetryFromState(s SimState) *hal.Telemetry {
	posStr := fmt.Sprintf("%.2f,%.2f,0", s.Position.X, s.Position.Y)
	targetStr := ""
	if s.Mode == ModeNavigating || s.Mode == ModeReturning || s.Mode == ModeArrived {
		targetStr = fmt.Sprintf("%.2f,%.2f,0", s.TargetPosition.X, s.TargetPosition.Y)
	}

	currentTask := "idle"
	switch s.Mode {
	case ModeStand:
		currentTask = "stand"
	case ModeWalk, ModeNavigating, ModeReturning:
		currentTask = "walk"
	case ModeSafeStop:
		currentTask = "idle"
	}

	actuatorStatus := "enabled"
	if !s.Online {
		actuatorStatus = "disabled"
	} else if s.Mode == ModeError {
		actuatorStatus = "error"
	}

	dist := s.DistanceToTarget
	t := &hal.Telemetry{
		RobotID:         s.RobotID,
		Timestamp:       time.Now(),
		Online:         s.Online,
		ActuatorStatus:  actuatorStatus,
		MockMode:        true,
		CurrentTask:     currentTask,
		Position:        posStr,
		TargetPosition:  targetStr,
		DistanceToTarget: &dist,
	}
	return t
}
