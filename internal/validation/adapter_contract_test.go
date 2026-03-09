package validation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

func TestValidateAdapterContract_NilRobot(t *testing.T) {
	spec := DefaultContractSpec()
	violations := ValidateAdapterContract(spec, nil, &hal.Telemetry{})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Category != "robot" {
		t.Errorf("expected category robot, got %s", violations[0].Category)
	}
}

func TestValidateAdapterContract_NilTelemetry(t *testing.T) {
	spec := DefaultContractSpec()
	robot := &hal.Robot{ID: "sim-001", Capabilities: []string{hal.CapNavigation}}
	violations := ValidateAdapterContract(spec, robot, nil)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Category != "telemetry" {
		t.Errorf("expected category telemetry, got %s", violations[0].Category)
	}
}

func TestValidateAdapterContract_Valid(t *testing.T) {
	spec := DefaultContractSpec()
	robot := &hal.Robot{
		ID:           "sim-001",
		Capabilities: []string{hal.CapNavigation, hal.CapSafeStop},
	}
	telemetry := &hal.Telemetry{
		RobotID:          "sim-001",
		Timestamp:        time.Now(),
		Online:           true,
		CurrentTask:      "idle",
		Position:         "0,0,0",
		TargetPosition:   "10,0,0",
		DistanceToTarget: ptrFloat64(5.0),
		ActuatorStatus:   "enabled",
	}
	violations := ValidateAdapterContract(spec, robot, telemetry)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d: %v", len(violations), violations)
	}
}

func TestValidateAdapterContract_MissingTelemetryFields(t *testing.T) {
	spec := DefaultContractSpec()
	robot := &hal.Robot{ID: "sim-001", Capabilities: []string{hal.CapNavigation}}
	telemetry := &hal.Telemetry{
		RobotID: "sim-001",
		Online:  true,
		// missing: timestamp, current_task, position, target_position, distance_to_target, actuator_status
	}
	violations := ValidateAdapterContract(spec, robot, telemetry)
	if len(violations) < 4 {
		t.Errorf("expected at least 4 violations for missing fields, got %d", len(violations))
	}
}

func TestValidateAdapterContract_RobotIDMismatch(t *testing.T) {
	spec := DefaultContractSpec()
	robot := &hal.Robot{ID: "sim-001"}
	telemetry := &hal.Telemetry{
		RobotID:          "sim-002",
		Timestamp:        time.Now(),
		Online:           true,
		CurrentTask:      "idle",
		Position:         "0,0,0",
		TargetPosition:   "0,0,0",
		DistanceToTarget: ptrFloat64(0),
		ActuatorStatus:   "enabled",
	}
	violations := ValidateAdapterContract(spec, robot, telemetry)
	found := false
	for _, v := range violations {
		if v.Field == "robot_id" && v.Message != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected robot_id mismatch violation")
	}
}

func TestValidateTelemetryFields_Valid(t *testing.T) {
	tel := &hal.Telemetry{
		RobotID:          "sim-001",
		Timestamp:        time.Now(),
		Online:           true,
		CurrentTask:      "idle",
		Position:         "0,0,0",
		TargetPosition:   "10,0,0",
		DistanceToTarget: ptrFloat64(5.0),
	}
	violations := ValidateTelemetryFields(tel)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestValidateTelemetryFields_Nil(t *testing.T) {
	violations := ValidateTelemetryFields(nil)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestContractSpecToJSON(t *testing.T) {
	spec := DefaultContractSpec()
	data, err := ContractSpecToJSON(spec)
	if err != nil {
		t.Fatalf("ContractSpecToJSON: %v", err)
	}
	var c ContractJSON
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(c.RequiredCommands) == 0 {
		t.Error("expected required_commands")
	}
	if len(c.RequiredTelemetry) == 0 {
		t.Error("expected required_telemetry")
	}
	if c.NATSTopics.Telemetry != "telemetry.robots.{robot_id}" {
		t.Errorf("unexpected telemetry topic: %s", c.NATSTopics.Telemetry)
	}
}

func TestRunContractCheck(t *testing.T) {
	report := &Report{ScenarioName: "test"}
	robot := &hal.Robot{ID: "sim-001", Capabilities: []string{hal.CapNavigation}}
	telemetry := &hal.Telemetry{
		RobotID:          "sim-001",
		Timestamp:        time.Now(),
		Online:           true,
		CurrentTask:      "idle",
		Position:         "0,0,0",
		TargetPosition:   "0,0,0",
		DistanceToTarget: ptrFloat64(0),
		ActuatorStatus:   "enabled",
	}
	RunContractCheck(report, robot, telemetry)
	if len(report.ContractViolations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.ContractViolations))
	}
}

func TestRunContractCheck_NilRobotSkips(t *testing.T) {
	report := &Report{}
	RunContractCheck(report, nil, &hal.Telemetry{RobotID: "x"})
	if len(report.ContractViolations) != 0 {
		t.Error("expected no violations when robot is nil (check skipped)")
	}
}

func ptrFloat64(v float64) *float64 {
	return &v
}
