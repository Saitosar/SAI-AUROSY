package simrobot

import "time"

// RobotIDPrefix is the prefix for simulated robot IDs in the fleet registry.
const RobotIDPrefix = "sim-"

// SimMode represents the operational mode of a simulated robot.
type SimMode string

const (
	ModeIdle           SimMode = "idle"
	ModeNavigating    SimMode = "navigating"
	ModeArrived       SimMode = "arrived"
	ModeReturning     SimMode = "returning_to_base"
	ModeSafeStop      SimMode = "safe_stop"
	ModeError         SimMode = "error"
	ModeStand         SimMode = "stand"
	ModeWalk          SimMode = "walk"
)

// FailureType identifies a configurable failure scenario.
type FailureType string

const (
	FailureNavigationTimeout FailureType = "navigation_timeout"
	FailureOfflineMidRoute   FailureType = "offline_mid_route"
	FailureSafeStopMidRoute  FailureType = "safe_stop_mid_route"
	FailureBatteryLow        FailureType = "battery_low"
	FailureCommandRejected   FailureType = "command_rejected"
	FailureDelayedArrival    FailureType = "delayed_arrival"
)

// FailureConfig specifies when and how a failure should be triggered.
type FailureConfig struct {
	Type FailureType `json:"type"`

	// AfterTicks triggers after N telemetry ticks.
	AfterTicks int `json:"after_ticks,omitempty"`

	// WhenDistanceLt triggers when distance_to_target < X meters.
	WhenDistanceLt float64 `json:"when_distance_lt,omitempty"`

	// AfterCommand triggers after receiving a specific command (e.g. "navigate_to").
	AfterCommand string `json:"after_command,omitempty"`

	// BatteryLevel sets battery to this value (0-100) for battery_low.
	BatteryLevel float64 `json:"battery_level,omitempty"`

	// SlowdownFactor multiplies speed for delayed_arrival (e.g. 0.1 = 10% speed).
	SlowdownFactor float64 `json:"slowdown_factor,omitempty"`

	// DurationTicks applies slowdown or effect for N ticks.
	DurationTicks int `json:"duration_ticks,omitempty"`

	// applied is set when delayed_arrival has been applied (internal use).
	applied bool
}

// Position holds X, Y coordinates for the simulated robot.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// SimState holds the full runtime state of a simulated robot.
type SimState struct {
	RobotID         string    `json:"robot_id"`
	TenantID        string    `json:"tenant_id"`
	RobotType       string    `json:"robot_type"`
	Capabilities    []string  `json:"capabilities"`

	Online          bool      `json:"online"`
	Mode            SimMode   `json:"mode"`
	Position        Position  `json:"position"`
	TargetPosition  Position  `json:"target_position"`
	RouteNodes      []string  `json:"route_nodes,omitempty"`
	RouteIndex      int       `json:"route_index"`
	DistanceToTarget float64  `json:"distance_to_target"`
	Speed           float64   `json:"speed"` // m/s simulated

	BatteryLevel    float64   `json:"battery_level"`
	CurrentTaskID   string    `json:"current_task_id"`
	CurrentScenario string    `json:"current_scenario"`
	LastCommand     string    `json:"last_command"`
	LastSpokenText  string    `json:"last_spoken_text"`

	TickCount       int       `json:"tick_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReplayTick represents a single tick in a replay script.
type ReplayTick struct {
	Position        Position `json:"position"`
	DistanceToTarget float64 `json:"distance_to_target"`
	Online          bool     `json:"online"`
}

// ReplayScript defines a deterministic replay scenario.
type ReplayScript struct {
	Scenario string       `json:"scenario"`
	Ticks    []ReplayTick `json:"ticks"`
}
