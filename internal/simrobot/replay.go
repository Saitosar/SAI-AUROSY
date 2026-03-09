package simrobot

import (
	"encoding/json"
	"os"
	"time"
)

// LoadReplayScript loads a replay script from a JSON file.
func LoadReplayScript(path string) (*ReplayScript, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var script ReplayScript
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, err
	}
	return &script, nil
}

// RunReplay executes a replay script against a robot, publishing telemetry from each tick.
// The robot must be stopped (no tick loop running). This is used for deterministic testing.
// Each tick is published at approximately the default tick interval.
func RunReplay(robot *SimRobot, script *ReplayScript, tickInterval time.Duration) {
	if script == nil || len(script.Ticks) == 0 {
		return
	}
	if tickInterval <= 0 {
		tickInterval = 500 * time.Millisecond
	}

	for i := range script.Ticks {
		tick := &script.Ticks[i]
		robot.mu.Lock()
		robot.state.Position = tick.Position
		robot.state.DistanceToTarget = tick.DistanceToTarget
		robot.state.Online = tick.Online
		robot.state.TickCount = i + 1
		robot.state.UpdatedAt = time.Now()
		robot.mu.Unlock()

		t := buildTelemetryFromState(robot.State())
		_ = robot.bus.PublishTelemetry(t)

		if i < len(script.Ticks)-1 {
			time.Sleep(tickInterval)
		}
	}
}
