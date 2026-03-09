package validation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// RunConfig configures the validation run.
type RunConfig struct {
	ScenarioDir    string
	OutputDir      string
	TimeoutPerStep time.Duration
	ContractCheck  bool
}

// DefaultRunConfig returns default run configuration.
func DefaultRunConfig() RunConfig {
	return RunConfig{
		ScenarioDir:    "testdata/validation",
		OutputDir:      "outputs/validation",
		TimeoutPerStep: 90 * time.Second,
	}
}

// RunScenario executes a single scenario and returns the report.
func RunScenario(ctx context.Context, vctx *ValidationContext, scenario *Scenario, cfg RunConfig) (*Report, error) {
	start := time.Now()
	report := &Report{
		ScenarioName:  scenario.Name,
		Status:        "FAIL",
		StartTime:     start,
		Results:       []ValidationResult{},
		EmittedEvents: []CollectedEvent{},
	}

	robotID := scenario.RobotID
	if robotID == "" {
		robotID = "sim-001"
	}

	// Setup
	if vctx.SimRobotService != nil {
		if err := vctx.SimRobotService.Reset(robotID); err != nil {
			report.Error = fmt.Sprintf("reset failed: %v", err)
			report.EndTime = time.Now()
			report.DurationMs = time.Since(start).Milliseconds()
			return report, nil
		}
		if scenario.FailureConfig != nil {
			fc := &SimRobotFailureConfig{
				Type:            scenario.FailureConfig.Type,
				AfterTicks:      scenario.FailureConfig.AfterTicks,
				WhenDistanceLt:  scenario.FailureConfig.WhenDistanceLt,
				AfterCommand:    scenario.FailureConfig.AfterCommand,
				BatteryLevel:    scenario.FailureConfig.BatteryLevel,
				SlowdownFactor:  scenario.FailureConfig.SlowdownFactor,
				DurationTicks:   scenario.FailureConfig.DurationTicks,
			}
			_ = vctx.SimRobotService.InjectFailure(robotID, fc)
		}
	}

	if vctx.EventCollector != nil {
		vctx.EventCollector.Clear()
	}
	if vctx.TelemetryCollector != nil {
		vctx.TelemetryCollector.Clear()
	}

	// Execute steps
	stepCtx, cancel := context.WithTimeout(ctx, cfg.TimeoutPerStep*time.Duration(len(scenario.Steps)+1))
	defer cancel()

	var mallTaskID string
	for _, step := range scenario.Steps {
		select {
		case <-stepCtx.Done():
			report.Error = "scenario timeout"
			break
		default:
		}

		switch step.Action {
		case "start_mall_assistant":
			if vctx.MallAssistantTrigger != nil {
				id, err := vctx.MallAssistantTrigger.StartMallAssistant(robotID, "default", "validation")
				if err != nil {
					report.Error = fmt.Sprintf("start_mall_assistant: %v", err)
					report.EndTime = time.Now()
					report.DurationMs = time.Since(start).Milliseconds()
					return report, nil
				}
				mallTaskID = id
			}
		case "submit_visitor_request":
			text := step.Text
			if text == "" && step.StoreName != "" {
				text = fmt.Sprintf("Where is %s?", step.StoreName)
			}
			if text == "" {
				text = "Where is Nike?"
			}
			if vctx.MallAssistantTrigger != nil {
				_, _ = vctx.MallAssistantTrigger.SubmitVisitorRequest(robotID, text)
			}
		case "wait_navigation_complete", "wait_return_complete":
			timeout := step.TimeoutSec
			if timeout <= 0 {
				timeout = 60
			}
			time.Sleep(time.Duration(timeout) * time.Second)
		case "wait":
			sec := step.TimeoutSec
			if sec <= 0 {
				sec = 5
			}
			time.Sleep(time.Duration(sec) * time.Second)
		case "inject_safe_stop":
			// Simulate operator sending safe_stop - would need bus publish
			time.Sleep(2 * time.Second)
		}
	}

	// Allow platform to settle
	time.Sleep(2 * time.Second)

	// Build assertion context
	ac := &AssertionContext{
		RobotID:          robotID,
		MallAssistantTaskID: mallTaskID,
	}

	if vctx.EventCollector != nil {
		ac.Events = vctx.EventCollector.GetCollected()
		report.EmittedEvents = ac.Events
	}
	if vctx.TelemetryCollector != nil {
		tel := vctx.TelemetryCollector.GetCollected()
		ac.TelemetrySamples = TelemetryToSamples(tel)
	}
	if vctx.TaskStore != nil {
		tasks, _ := vctx.TaskStore.List(TaskListFilters{RobotID: robotID})
		ac.Tasks = tasks
		for _, t := range tasks {
			if t.ScenarioID == "navigate_to_store" {
				ac.NavigateTaskIDs = append(ac.NavigateTaskIDs, t.ID)
			}
		}
	}
	if vctx.RobotStateProvider != nil {
		ac.FinalRobotState = vctx.RobotStateProvider.GetRobotState(robotID)
		if ac.FinalRobotState != nil {
			report.FinalRobotState = ac.FinalRobotState.State
		}
	}
	if vctx.SimRobotService != nil {
		state, _ := vctx.SimRobotService.GetState(robotID)
		ac.SimRobotState = state
	}

	// Run assertions
	passed := 0
	failed := 0
	for _, a := range scenario.Assertions {
		res := EvaluateAssertion(a, ac)
		report.Results = append(report.Results, res)
		if res.Passed {
			passed++
		} else {
			failed++
		}
	}

	report.AssertionsPassed = passed
	report.AssertionsFailed = failed
	report.EndTime = time.Now()
	report.DurationMs = time.Since(start).Milliseconds()
	if failed == 0 && report.Error == "" {
		report.Status = "PASS"
	} else {
		report.Status = "FAIL"
	}

	if cfg.ContractCheck && vctx.RobotRegistry != nil {
		robot := vctx.RobotRegistry.Get(robotID)
		tel := vctx.TelemetryCollector.GetCollected()
		var lastTel *hal.Telemetry
		if len(tel) > 0 {
			lastTel = tel[len(tel)-1]
		}
		RunContractCheck(report, robot, lastTel)
	}

	return report, nil
}

// RunAllScenarios loads and runs all scenarios from the scenario directory.
func RunAllScenarios(ctx context.Context, vctx *ValidationContext, cfg RunConfig) ([]*Report, error) {
	scenarios, err := LoadScenariosFromDir(cfg.ScenarioDir)
	if err != nil {
		return nil, fmt.Errorf("load scenarios: %w", err)
	}
	reports := make([]*Report, 0, len(scenarios))
	for _, s := range scenarios {
		r, err := RunScenario(ctx, vctx, s, cfg)
		if err != nil {
			return nil, fmt.Errorf("run %s: %w", s.Name, err)
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// EventCollectorImpl is a concrete event collector.
type EventCollectorImpl struct {
	mu      sync.Mutex
	events  []CollectedEvent
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollectorImpl {
	return &EventCollectorImpl{events: []CollectedEvent{}}
}

// Collect records an event.
func (e *EventCollectorImpl) Collect(eventType string, payload map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, CollectedEvent{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	})
}

// GetCollected returns all collected events.
func (e *EventCollectorImpl) GetCollected() []CollectedEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]CollectedEvent, len(e.events))
	copy(out, e.events)
	return out
}

// Clear clears collected events.
func (e *EventCollectorImpl) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = nil
}

// TelemetryCollectorImpl is a concrete telemetry collector.
type TelemetryCollectorImpl struct {
	mu     sync.Mutex
	samples []*hal.Telemetry
}

// NewTelemetryCollector creates a new telemetry collector.
func NewTelemetryCollector() *TelemetryCollectorImpl {
	return &TelemetryCollectorImpl{samples: []*hal.Telemetry{}}
}

// Collect records telemetry.
func (t *TelemetryCollectorImpl) Collect(tel *hal.Telemetry) {
	if tel == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := *tel
	t.samples = append(t.samples, &cp)
}

// GetCollected returns all collected telemetry.
func (t *TelemetryCollectorImpl) GetCollected() []*hal.Telemetry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*hal.Telemetry, len(t.samples))
	copy(out, t.samples)
	return out
}

// Clear clears collected telemetry.
func (t *TelemetryCollectorImpl) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.samples = nil
}
