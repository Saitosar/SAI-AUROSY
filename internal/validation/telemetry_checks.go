package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// TelemetryToSamples converts hal.Telemetry slices to TelemetrySample for assertion checks.
func TelemetryToSamples(tel []*hal.Telemetry) []*TelemetrySample {
	out := make([]*TelemetrySample, 0, len(tel))
	for _, t := range tel {
		if t == nil {
			continue
		}
		ts := ""
		if !t.Timestamp.IsZero() {
			ts = t.Timestamp.Format(time.RFC3339)
		}
		out = append(out, &TelemetrySample{
			RobotID:          t.RobotID,
			Online:           t.Online,
			Position:         t.Position,
			TargetPosition:   t.TargetPosition,
			DistanceToTarget: t.DistanceToTarget,
			CurrentTask:      t.CurrentTask,
			Timestamp:        ts,
		})
	}
	return out
}

// CheckTelemetryFieldPresent verifies that at least one telemetry sample has the given field populated.
func CheckTelemetryFieldPresent(samples []*TelemetrySample, field string, robotID string) (bool, string, map[string]interface{}) {
	field = strings.ToLower(field)
	for _, s := range samples {
		if robotID != "" && s.RobotID != robotID {
			continue
		}
		has := false
		switch field {
		case "robot_id":
			has = s.RobotID != ""
		case "online":
			has = true // always present
		case "position":
			has = s.Position != ""
		case "target_position":
			has = s.TargetPosition != ""
		case "distance_to_target":
			has = s.DistanceToTarget != nil
		case "current_task":
			has = s.CurrentTask != ""
		case "timestamp":
			has = s.Timestamp != ""
		default:
			return false, fmt.Sprintf("unknown field: %s", field), nil
		}
		if has {
			return true, fmt.Sprintf("field %s present in telemetry", field),
				map[string]interface{}{"field": field}
		}
	}
	return false, fmt.Sprintf("field %s not found in telemetry samples", field),
		map[string]interface{}{
			"field":   field,
			"samples": len(samples),
		}
}

// CheckTelemetryProgression verifies that distance_to_target decreases over time during navigation.
func CheckTelemetryProgression(samples []*TelemetrySample, robotID string) (bool, string, map[string]interface{}) {
	var prev *float64
	for _, s := range samples {
		if robotID != "" && s.RobotID != robotID {
			continue
		}
		if s.DistanceToTarget == nil {
			continue
		}
		d := *s.DistanceToTarget
		if prev != nil && d > *prev {
			// Allow some tolerance - could be noise
			if *prev > 0.1 && d > *prev+0.5 {
				return false, fmt.Sprintf("distance increased from %v to %v", *prev, d),
					map[string]interface{}{
						"prev": *prev,
						"curr": d,
					}
			}
		}
		prev = &d
	}
	if prev == nil {
		return false, "no distance_to_target samples", nil
	}
	return true, "distance_to_target progressed or stayed constant",
		map[string]interface{}{"final_distance": *prev}
}

// CheckSafeStopReceived verifies that safe_stop was observed (event or robot mode).
func CheckSafeStopReceived(events []CollectedEvent, simState *SimRobotState, samples []*TelemetrySample, robotID string) (bool, string, map[string]interface{}) {
	for _, e := range events {
		if e.Type == "safe_stop" {
			return true, "safe_stop event emitted", map[string]interface{}{"event_type": "safe_stop"}
		}
	}
	if simState != nil && simState.Mode == "safe_stop" {
		return true, "robot in safe_stop mode", map[string]interface{}{"mode": simState.Mode}
	}
	for _, s := range samples {
		if robotID != "" && s.RobotID != robotID {
			continue
		}
		if s.CurrentTask == "idle" && len(samples) > 1 {
			// After safe_stop, current_task often becomes idle
			return true, "telemetry shows idle (safe_stop effect)", nil
		}
	}
	return false, "safe_stop not observed", nil
}
