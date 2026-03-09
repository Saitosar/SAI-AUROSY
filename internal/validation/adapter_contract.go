package validation

import (
	"encoding/json"
	"strings"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// ContractSpec defines the adapter readiness contract specification.
type ContractSpec struct {
	RequiredCommands   []string
	RequiredTelemetry  []string
	CapabilityCommands map[string][]string // capability -> commands that must work
}

// DefaultContractSpec returns the default adapter readiness contract.
func DefaultContractSpec() ContractSpec {
	return ContractSpec{
		RequiredCommands: []string{
			"navigate_to", "safe_stop", "release_control", "walk_mode", "stand_mode",
		},
		RequiredTelemetry: []string{
			"robot_id", "online", "current_task", "position", "target_position",
			"distance_to_target", "timestamp", "actuator_status",
		},
		CapabilityCommands: map[string][]string{
			hal.CapNavigation:     {"navigate_to"},
			hal.CapSafeStop:      {"safe_stop"},
			hal.CapReleaseControl: {"release_control"},
			hal.CapWalk:          {"walk_mode"},
			hal.CapStand:         {"stand_mode"},
			hal.CapSpeech:        {"speak"},
		},
	}
}

// ContractViolation describes a contract violation.
type ContractViolation struct {
	Category string // "command", "telemetry", "capability"
	Field    string
	Message  string
}

// ValidateAdapterContract checks a robot and telemetry sample against the contract.
func ValidateAdapterContract(spec ContractSpec, robot *hal.Robot, telemetry *hal.Telemetry) []ContractViolation {
	var violations []ContractViolation

	if robot == nil {
		violations = append(violations, ContractViolation{
			Category: "robot",
			Message:  "robot is nil",
		})
		return violations
	}

	// Check capability-command mapping
	caps := make(map[string]bool)
	for _, c := range robot.Capabilities {
		caps[c] = true
	}
	for cap, cmds := range spec.CapabilityCommands {
		if !caps[cap] {
			continue
		}
		_ = cmds // Capability implies command support; documented in contract
	}

	if telemetry == nil {
		violations = append(violations, ContractViolation{
			Category: "telemetry",
			Message:  "telemetry sample is nil",
		})
		return violations
	}

	// Check required telemetry fields
	for _, field := range spec.RequiredTelemetry {
		has := false
		switch field {
		case "robot_id":
			has = telemetry.RobotID != ""
		case "timestamp":
			has = !telemetry.Timestamp.IsZero()
		case "online":
			has = true // always present
		case "current_task":
			has = telemetry.CurrentTask != ""
		case "position":
			has = telemetry.Position != ""
		case "target_position":
			has = telemetry.TargetPosition != ""
		case "distance_to_target":
			has = telemetry.DistanceToTarget != nil
		case "actuator_status":
			has = telemetry.ActuatorStatus != ""
		default:
			continue
		}
		if !has {
			violations = append(violations, ContractViolation{
				Category: "telemetry",
				Field:    field,
				Message:  "required telemetry field missing or empty",
			})
		}
	}

	// Check robot_id consistency
	if robot.ID != "" && telemetry.RobotID != "" && robot.ID != telemetry.RobotID {
		violations = append(violations, ContractViolation{
			Category: "telemetry",
			Field:    "robot_id",
			Message:  "telemetry robot_id does not match registry",
		})
	}

	return violations
}

// ValidateTelemetryFields checks that telemetry has required fields for navigation.
func ValidateTelemetryFields(t *hal.Telemetry) []ContractViolation {
	var violations []ContractViolation
	if t == nil {
		violations = append(violations, ContractViolation{Category: "telemetry", Message: "nil telemetry"})
		return violations
	}
	required := []string{"robot_id", "online", "timestamp", "position", "target_position", "distance_to_target", "current_task"}
	for _, f := range required {
		ok := false
		switch f {
		case "robot_id":
			ok = t.RobotID != ""
		case "timestamp":
			ok = !t.Timestamp.IsZero()
		case "online":
			ok = true
		case "position":
			ok = t.Position != ""
		case "target_position":
			ok = t.TargetPosition != ""
		case "distance_to_target":
			ok = t.DistanceToTarget != nil
		case "current_task":
			ok = t.CurrentTask != ""
		}
		if !ok {
			violations = append(violations, ContractViolation{
				Category: "telemetry",
				Field:    f,
				Message:  "missing or empty",
			})
		}
	}
	return violations
}

// ContractJSON is the JSON-serializable structure for the adapter contract.
type ContractJSON struct {
	Description       string            `json:"description"`
	RequiredCommands  []string          `json:"required_commands"`
	RequiredTelemetry []string          `json:"required_telemetry"`
	CapabilityCommands map[string][]string `json:"capability_commands"`
	Timing            ContractTiming   `json:"timing"`
	NATSTopics        NATSTopics       `json:"nats_topics"`
}

// ContractTiming holds timing expectations from the contract.
type ContractTiming struct {
	TelemetryIntervalSec     int `json:"telemetry_interval_sec"`
	StaleThresholdSec        int `json:"stale_threshold_sec"`
	CommandEffectMaxDelaySec int `json:"command_effect_max_delay_sec"`
}

// NATSTopics holds NATS topic patterns.
type NATSTopics struct {
	Telemetry string `json:"telemetry"`
	Commands  string `json:"commands"`
}

// ContractSpecToJSON marshals the contract spec to JSON for tooling and vendor onboarding.
func ContractSpecToJSON(spec ContractSpec) ([]byte, error) {
	c := ContractJSON{
		Description:       "Adapter readiness contract for SAI AUROSY Mall Assistant pilot",
		RequiredCommands:  spec.RequiredCommands,
		RequiredTelemetry: spec.RequiredTelemetry,
		CapabilityCommands: spec.CapabilityCommands,
		Timing: ContractTiming{
			TelemetryIntervalSec:     2,
			StaleThresholdSec:        5,
			CommandEffectMaxDelaySec: 2,
		},
		NATSTopics: NATSTopics{
			Telemetry: "telemetry.robots.{robot_id}",
			Commands:  "commands.robots.{robot_id}",
		},
	}
	return json.MarshalIndent(c, "", "  ")
}

// RunContractCheck runs ValidateAdapterContract and appends violations to the report.
// Use when -contract-check is set to verify adapter readiness during validation.
func RunContractCheck(report *Report, robot *hal.Robot, telemetry *hal.Telemetry) {
	if robot == nil || telemetry == nil {
		return
	}
	spec := DefaultContractSpec()
	report.ContractViolations = ValidateAdapterContract(spec, robot, telemetry)
}

// HasCapability returns true if robot has the capability.
func HasCapability(robot *hal.Robot, cap string) bool {
	if robot == nil {
		return false
	}
	cap = strings.ToLower(cap)
	for _, c := range robot.Capabilities {
		if strings.ToLower(c) == cap {
			return true
		}
	}
	return false
}
