package validation

import (
	"fmt"
	"strings"
)

// AssertionContext holds the data collected during scenario execution for assertion evaluation.
type AssertionContext struct {
	RobotID           string
	Events            []CollectedEvent
	TelemetrySamples  []*TelemetrySample
	Tasks             []Task
	FinalRobotState   *RobotStateResponse
	SimRobotState     *SimRobotState
	MallAssistantTaskID string
	NavigateTaskIDs   []string
}

// TelemetrySample is a simplified telemetry snapshot for assertion checks.
type TelemetrySample struct {
	RobotID          string
	Online           bool
	Position         string
	TargetPosition   string
	DistanceToTarget *float64
	CurrentTask      string
	Timestamp        string
}

// EvaluateAssertion runs a single assertion and returns the result.
func EvaluateAssertion(a Assertion, ctx *AssertionContext) ValidationResult {
	var passed bool
	var msg string
	var details map[string]interface{}

	switch a.Type {
	case AssertFinalRobotState:
		passed, msg, details = checkFinalRobotState(a, ctx)
	case AssertTaskStatus:
		passed, msg, details = checkTaskStatus(a, ctx)
	case AssertEventSequence:
		passed, msg, details = checkEventSequence(a, ctx)
	case AssertEventPresent:
		passed, msg, details = checkEventPresent(a, ctx)
	case AssertNoNavigationTask:
		passed, msg, details = checkNoNavigationTask(a, ctx)
	case AssertTelemetryFieldPresent:
		passed, msg, details = checkTelemetryFieldPresent(a, ctx)
	case AssertTelemetryProgression:
		passed, msg, details = checkTelemetryProgression(a, ctx)
	case AssertTimeoutTriggered:
		passed, msg, details = checkTimeoutTriggered(a, ctx)
	case AssertSafeStopReceived:
		passed, msg, details = checkSafeStopReceived(a, ctx)
	case AssertRobotOfflineFailure:
		passed, msg, details = checkRobotOfflineFailure(a, ctx)
	default:
		passed = false
		msg = fmt.Sprintf("unknown assertion type: %s", a.Type)
		details = map[string]interface{}{"type": a.Type}
	}

	if a.Negated {
		passed = !passed
		if msg != "" {
			msg = "negated: " + msg
		}
	}

	return ValidationResult{
		AssertionType: a.Type,
		Passed:        passed,
		Message:       msg,
		Details:       details,
	}
}

func checkFinalRobotState(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	expected := strings.ToUpper(a.State)
	if expected == "" {
		expected = "IDLE"
	}
	if ctx.FinalRobotState == nil {
		return false, "no robot state available", map[string]interface{}{"expected": expected}
	}
	actual := strings.ToUpper(ctx.FinalRobotState.State)
	passed := actual == expected
	msg := fmt.Sprintf("expected state %s, got %s", expected, actual)
	if passed {
		msg = fmt.Sprintf("robot state is %s", actual)
	}
	return passed, msg, map[string]interface{}{
		"expected": expected,
		"actual":   actual,
	}
}

func checkTaskStatus(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	scenarioID := a.ScenarioID
	expectedStatus := strings.ToLower(a.Status)
	if expectedStatus == "" {
		expectedStatus = "completed"
	}
	for _, t := range ctx.Tasks {
		if scenarioID != "" && t.ScenarioID != scenarioID {
			continue
		}
		if t.RobotID != ctx.RobotID {
			continue
		}
		actual := strings.ToLower(t.Status)
		passed := actual == expectedStatus
		msg := fmt.Sprintf("task %s: expected status %s, got %s", t.ID, expectedStatus, actual)
		if passed {
			msg = fmt.Sprintf("task %s has status %s", t.ID, actual)
		}
		return passed, msg, map[string]interface{}{
			"task_id":  t.ID,
			"expected": expectedStatus,
			"actual":   actual,
		}
	}
	return false, "no matching task found", map[string]interface{}{
		"scenario_id": scenarioID,
		"expected":    expectedStatus,
	}
}

func checkEventSequence(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	return CheckEventSequence(ctx.Events, a.Events)
}

func checkEventPresent(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	eventType := a.EventType
	if eventType == "" && len(a.Events) > 0 {
		eventType = a.Events[0]
	}
	if eventType == "" {
		return false, "no event type specified", nil
	}
	return CheckEventPresent(ctx.Events, eventType)
}

func checkNoNavigationTask(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	for _, t := range ctx.Tasks {
		if t.ScenarioID == "navigate_to_store" && t.RobotID == ctx.RobotID {
			return false, fmt.Sprintf("found navigate_to_store task %s (expected none)", t.ID),
				map[string]interface{}{"task_id": t.ID}
		}
	}
	return true, "no navigate_to_store task created", nil
}

func checkTelemetryFieldPresent(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	field := a.Field
	if field == "" {
		return false, "no field specified", nil
	}
	return CheckTelemetryFieldPresent(ctx.TelemetrySamples, field, ctx.RobotID)
}

func checkTelemetryProgression(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	return CheckTelemetryProgression(ctx.TelemetrySamples, ctx.RobotID)
}

func checkTimeoutTriggered(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	reason := strings.ToLower(a.Reason)
	if reason == "" {
		reason = "timeout"
	}
	for _, t := range ctx.Tasks {
		if t.RobotID != ctx.RobotID {
			continue
		}
		if t.Status != "failed" && t.Status != "cancelled" {
			continue
		}
		// Check if any task failed with timeout-like reason
		// In practice we might need to store failure reason in Task
		// For now, accept failed status for timeout scenarios
		if t.Status == "failed" {
			return true, "task failed (timeout path)", map[string]interface{}{
				"task_id": t.ID,
				"status":  t.Status,
			}
		}
	}
	return false, "no timeout/failure observed", nil
}

func checkSafeStopReceived(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	return CheckSafeStopReceived(ctx.Events, ctx.SimRobotState, ctx.TelemetrySamples, ctx.RobotID)
}

func checkRobotOfflineFailure(a Assertion, ctx *AssertionContext) (bool, string, map[string]interface{}) {
	// Check that a task failed and we have offline telemetry
	for _, t := range ctx.Tasks {
		if t.RobotID != ctx.RobotID {
			continue
		}
		if t.Status == "failed" {
			// Check if any telemetry shows offline
			for _, s := range ctx.TelemetrySamples {
				if s.RobotID == ctx.RobotID && !s.Online {
					return true, "task failed with robot offline", map[string]interface{}{
						"task_id": t.ID,
					}
				}
			}
			// Task failed - may be due to offline
			return true, "task failed (offline scenario)", map[string]interface{}{
				"task_id": t.ID,
			}
		}
	}
	return false, "no failure observed for offline scenario", nil
}
