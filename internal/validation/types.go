package validation

import (
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// AssertionType constants.
const (
	AssertFinalRobotState       = "final_robot_state"
	AssertTaskStatus            = "task_status"
	AssertEventSequence         = "event_sequence"
	AssertEventPresent          = "event_present"
	AssertNoNavigationTask      = "no_navigation_task_created"
	AssertTelemetryFieldPresent = "telemetry_field_present"
	AssertTelemetryProgression  = "telemetry_progression"
	AssertTimeoutTriggered      = "timeout_triggered"
	AssertSafeStopReceived      = "safe_stop_received"
	AssertRobotOfflineFailure   = "robot_offline_causes_failure"
)

// Scenario defines a validation scenario loaded from YAML.
type Scenario struct {
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	RobotID       string                 `yaml:"robot_id"`
	Preconditions []Precondition         `yaml:"preconditions"`
	Steps         []ScenarioStep         `yaml:"steps"`
	Assertions    []Assertion             `yaml:"assertions"`
	FailureConfig *FailureConfig         `yaml:"failure_config,omitempty"`
	Params        map[string]interface{}  `yaml:"params,omitempty"`
}

// Precondition defines a setup requirement.
type Precondition struct {
	RobotOnline   *bool  `yaml:"robot_online,omitempty"`
	RobotAtBase   *bool  `yaml:"robot_at_base,omitempty"`
	NoRunningTask *bool  `yaml:"no_running_task,omitempty"`
	Key           string `yaml:"key,omitempty"`
	Value         string `yaml:"value,omitempty"`
}

// ScenarioStep defines a single step in a scenario.
type ScenarioStep struct {
	Action      string                 `yaml:"action"`
	Text        string                 `yaml:"text,omitempty"`
	TimeoutSec  int                    `yaml:"timeout_sec,omitempty"`
	StoreName   string                 `yaml:"store_name,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty"`
}

// Assertion defines a validation assertion.
type Assertion struct {
	Type       string                 `yaml:"type"`
	State      string                 `yaml:"state,omitempty"`
	ScenarioID string                 `yaml:"scenario_id,omitempty"`
	Status     string                 `yaml:"status,omitempty"`
	Events     []string               `yaml:"events,omitempty"`
	EventType  string                 `yaml:"event_type,omitempty"`
	Field      string                 `yaml:"field,omitempty"`
	Negated    bool                   `yaml:"negated,omitempty"`
	Reason     string                 `yaml:"reason,omitempty"`
	Params     map[string]interface{} `yaml:"params,omitempty"`
}

// FailureConfig configures failure injection for the simrobot.
type FailureConfig struct {
	Type            string  `yaml:"type"`
	AfterTicks       int     `yaml:"after_ticks,omitempty"`
	WhenDistanceLt   float64 `yaml:"when_distance_lt,omitempty"`
	AfterCommand     string  `yaml:"after_command,omitempty"`
	BatteryLevel     float64 `yaml:"battery_level,omitempty"`
	SlowdownFactor   float64 `yaml:"slowdown_factor,omitempty"`
	DurationTicks    int     `yaml:"duration_ticks,omitempty"`
}

// ValidationResult holds the outcome of a single assertion.
type ValidationResult struct {
	AssertionType string
	Passed        bool
	Message       string
	Details       map[string]interface{}
}

// Report holds the full validation report for a scenario.
type Report struct {
	ScenarioName       string              `json:"scenario_name"`
	Status             string              `json:"status"` // PASS, FAIL
	StartTime          time.Time           `json:"start_time"`
	EndTime            time.Time           `json:"end_time"`
	DurationMs         int64               `json:"duration_ms"`
	AssertionsPassed   int                  `json:"assertions_passed"`
	AssertionsFailed   int                  `json:"assertions_failed"`
	FinalRobotState    string              `json:"final_robot_state,omitempty"`
	EmittedEvents      []CollectedEvent    `json:"emitted_events,omitempty"`
	Notes              string              `json:"notes,omitempty"`
	Results            []ValidationResult   `json:"results,omitempty"`
	Error              string              `json:"error,omitempty"`
	ContractViolations []ContractViolation `json:"contract_violations,omitempty"`
}

// CollectedEvent holds an event captured during validation.
type CollectedEvent struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// RobotRegistry provides robot lookup for contract validation.
type RobotRegistry interface {
	Get(id string) *hal.Robot
}

// ValidationContext provides dependencies for the validation runner.
type ValidationContext struct {
	SimRobotService      SimRobotService
	TaskStore            TaskStore
	EventCollector       EventCollector
	TelemetryCollector   TelemetryCollector
	MallAssistantTrigger MallAssistantTrigger
	RobotStateProvider   RobotStateProvider
	RobotRegistry        RobotRegistry
	Bus                  TelemetryBus
}

// SimRobotService interface for simrobot operations.
type SimRobotService interface {
	Reset(robotID string) error
	InjectFailure(robotID string, cfg *SimRobotFailureConfig) error
	GetState(robotID string) (*SimRobotState, error)
	GetRobot(robotID string) SimRobot
	Start(ctx interface{}, robotID string) error
	Stop(robotID string) error
}

// SimRobotFailureConfig is the failure config for simrobot (maps to simrobot.FailureConfig).
type SimRobotFailureConfig struct {
	Type            string
	AfterTicks      int
	WhenDistanceLt  float64
	AfterCommand    string
	BatteryLevel    float64
	SlowdownFactor  float64
	DurationTicks   int
}

// SimRobotState holds simrobot state (simplified view).
type SimRobotState struct {
	RobotID   string
	Online    bool
	Mode      string
	Position  string
	DistanceToTarget float64
}

// SimRobot interface for direct robot access (e.g. replay).
type SimRobot interface {
	State() SimRobotState
}

// TaskStore interface for task operations.
type TaskStore interface {
	Create(task *Task) error
	Get(id string) (*Task, error)
	List(filters TaskListFilters) ([]Task, error)
}

// Task represents a task for validation.
type Task struct {
	ID         string
	RobotID    string
	ScenarioID string
	Status     string
	Payload    []byte
}

// TaskListFilters filters for task listing.
type TaskListFilters struct {
	RobotID  string
	Status   string
}

// EventCollector captures events during validation.
type EventCollector interface {
	Collect(eventType string, payload map[string]interface{})
	GetCollected() []CollectedEvent
	Clear()
}

// TelemetryCollector captures telemetry during validation.
type TelemetryCollector interface {
	Collect(t *hal.Telemetry)
	GetCollected() []*hal.Telemetry
	Clear()
}

// MallAssistantTrigger triggers mall assistant scenario steps.
type MallAssistantTrigger interface {
	StartMallAssistant(robotID, tenantID, operatorID string) (taskID string, err error)
	SubmitVisitorRequest(robotID, text string) (bool, error)
}

// RobotStateProvider provides robot execution state.
type RobotStateProvider interface {
	GetRobotState(robotID string) *RobotStateResponse
}

// RobotStateResponse holds robot state for validation.
type RobotStateResponse struct {
	RobotID       string
	State         string
	CurrentTask   string
	Destination   string
	TargetStore   string
	StatusMessage string
}

// TelemetryBus interface for publishing and subscribing.
type TelemetryBus interface {
	SubscribeTelemetry(robotID string, callback func(*hal.Telemetry)) (interface{ Unsubscribe() error }, error)
}
