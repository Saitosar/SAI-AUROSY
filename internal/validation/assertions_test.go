package validation

import (
	"testing"
	"time"
)

func TestEvaluateAssertion_FinalRobotState(t *testing.T) {
	ac := &AssertionContext{
		FinalRobotState: &RobotStateResponse{State: "IDLE"},
	}
	res := EvaluateAssertion(Assertion{Type: AssertFinalRobotState, State: "IDLE"}, ac)
	if !res.Passed {
		t.Errorf("expected pass, got %s", res.Message)
	}
}

func TestEvaluateAssertion_FinalRobotStateMismatch(t *testing.T) {
	ac := &AssertionContext{
		FinalRobotState: &RobotStateResponse{State: "NAVIGATING"},
	}
	res := EvaluateAssertion(Assertion{Type: AssertFinalRobotState, State: "IDLE"}, ac)
	if res.Passed {
		t.Error("expected fail for state mismatch")
	}
}

func TestEvaluateAssertion_TaskStatus(t *testing.T) {
	ac := &AssertionContext{
		RobotID: "sim-001",
		Tasks: []Task{
			{ID: "t1", RobotID: "sim-001", ScenarioID: "mall_assistant", Status: "completed"},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertTaskStatus, ScenarioID: "mall_assistant", Status: "completed"}, ac)
	if !res.Passed {
		t.Errorf("expected pass, got %s", res.Message)
	}
}

func TestEvaluateAssertion_EventSequence(t *testing.T) {
	ac := &AssertionContext{
		Events: []CollectedEvent{
			{Type: "a", Timestamp: time.Now()},
			{Type: "b", Timestamp: time.Now()},
			{Type: "c", Timestamp: time.Now()},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertEventSequence, Events: []string{"a", "b", "c"}}, ac)
	if !res.Passed {
		t.Errorf("expected pass, got %s", res.Message)
	}
}

func TestEvaluateAssertion_EventSequenceMissing(t *testing.T) {
	ac := &AssertionContext{
		Events: []CollectedEvent{
			{Type: "a", Timestamp: time.Now()},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertEventSequence, Events: []string{"a", "b", "c"}}, ac)
	if res.Passed {
		t.Error("expected fail for missing event")
	}
}

func TestEvaluateAssertion_EventPresent(t *testing.T) {
	ac := &AssertionContext{
		Events: []CollectedEvent{
			{Type: "navigation_started", Timestamp: time.Now()},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertEventPresent, EventType: "navigation_started"}, ac)
	if !res.Passed {
		t.Errorf("expected pass, got %s", res.Message)
	}
}

func TestEvaluateAssertion_NoNavigationTask(t *testing.T) {
	ac := &AssertionContext{
		RobotID: "sim-001",
		Tasks: []Task{
			{ID: "t1", RobotID: "sim-001", ScenarioID: "mall_assistant", Status: "completed"},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertNoNavigationTask}, ac)
	if !res.Passed {
		t.Errorf("expected pass (no navigate_to_store task), got %s", res.Message)
	}
}

func TestEvaluateAssertion_NoNavigationTask_Found(t *testing.T) {
	ac := &AssertionContext{
		RobotID: "sim-001",
		Tasks: []Task{
			{ID: "t1", RobotID: "sim-001", ScenarioID: "navigate_to_store", Status: "completed"},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertNoNavigationTask}, ac)
	if res.Passed {
		t.Error("expected fail when navigate_to_store task exists")
	}
}

func TestEvaluateAssertion_TelemetryFieldPresent(t *testing.T) {
	d := 5.0
	ac := &AssertionContext{
		RobotID: "sim-001",
		TelemetrySamples: []*TelemetrySample{
			{RobotID: "sim-001", DistanceToTarget: &d},
		},
	}
	res := EvaluateAssertion(Assertion{Type: AssertTelemetryFieldPresent, Field: "distance_to_target"}, ac)
	if !res.Passed {
		t.Errorf("expected pass, got %s", res.Message)
	}
}

func TestEvaluateAssertion_UnknownType(t *testing.T) {
	ac := &AssertionContext{}
	res := EvaluateAssertion(Assertion{Type: "unknown_type"}, ac)
	if res.Passed {
		t.Error("expected fail for unknown assertion type")
	}
}
