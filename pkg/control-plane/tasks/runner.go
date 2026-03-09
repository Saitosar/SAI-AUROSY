package tasks

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/observability"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Runner executes pending tasks by expanding scenarios and sending commands via the bus.
type Runner struct {
	taskStore       Store
	scenarioCatalog *scenarios.Catalog
	registry        registry.Store
	bus             *telemetry.Bus
	coordinator     *coordinator.Coordinator
	pollInterval    time.Duration
	onTaskCompleted func(taskID, robotID, status string)
	onTaskStarted   func(taskID, robotID, scenarioID, zoneID string)
	onZoneAcquired  func(robotID, zoneID, taskID string)
	onZoneReleased  func(robotID, zoneID, taskID string)
}

// RunnerConfig configures the Task Runner.
type RunnerConfig struct {
	PollInterval    time.Duration
	OnTaskCompleted func(taskID, robotID, status string) // optional; called when task completes, fails, or is cancelled
	OnTaskStarted   func(taskID, robotID, scenarioID, zoneID string)
	OnZoneAcquired  func(robotID, zoneID, taskID string)
	OnZoneReleased  func(robotID, zoneID, taskID string)
}

// NewRunner creates a new Task Runner.
func NewRunner(taskStore Store, scenarioCatalog *scenarios.Catalog, reg registry.Store, bus *telemetry.Bus, cfg RunnerConfig) *Runner {
	return NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, nil, cfg)
}

// NewRunnerWithCoordinator creates a Task Runner with optional zone coordinator.
func NewRunnerWithCoordinator(taskStore Store, scenarioCatalog *scenarios.Catalog, reg registry.Store, bus *telemetry.Bus, coord *coordinator.Coordinator, cfg RunnerConfig) *Runner {
	interval := 2 * time.Second
	if cfg.PollInterval > 0 {
		interval = cfg.PollInterval
	}
	return &Runner{
		taskStore:       taskStore,
		scenarioCatalog: scenarioCatalog,
		registry:        reg,
		bus:             bus,
		coordinator:     coord,
		pollInterval:    interval,
		onTaskCompleted: cfg.OnTaskCompleted,
		onTaskStarted:   cfg.OnTaskStarted,
		onZoneAcquired:  cfg.OnZoneAcquired,
		onZoneReleased:  cfg.OnZoneReleased,
	}
}

// Run starts the task runner loop. Blocks until ctx is done.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOne(ctx)
		}
	}
}

// RunOnce runs a single iteration of the task runner (for testing).
func (r *Runner) RunOnce(ctx context.Context) {
	r.runOne(ctx)
}

func (r *Runner) runOne(ctx context.Context) {
	pending, err := r.taskStore.List(ListFilters{Status: StatusPending})
	if err != nil || len(pending) == 0 {
		return
	}
	task := pending[0]
	ctx, end := observability.StartSpan(ctx, "task.run",
		attribute.String("task_id", task.ID),
		attribute.String("robot_id", task.RobotID),
		attribute.String("scenario_id", task.ScenarioID),
	)
	defer end()
	hasRunning, err := r.taskStore.HasRunningForRobot(task.RobotID)
	if err != nil || hasRunning {
		return
	}
	if r.registry.Get(task.RobotID) == nil {
		return
	}
	scenario, ok := r.scenarioCatalog.Get(task.ScenarioID)
	if !ok {
		_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusFailed, time.Now())
		r.emitTaskCompleted(task.ID, task.RobotID, string(StatusFailed))
		return
	}
	robot := r.registry.Get(task.RobotID)
	if !hal.HasCapability(robot, scenario.RequiredCapabilities) {
		slog.Warn("task-runner robot lacks capabilities", "robot_id", task.RobotID, "scenario_id", task.ScenarioID)
		_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusFailed, time.Now())
		r.emitTaskCompleted(task.ID, task.RobotID, string(StatusFailed))
		return
	}
	// Execute steps
	var taskPayload struct {
		DurationSec int     `json:"duration_sec"`
		LinearX     float64 `json:"linear_x"`
		LinearY     float64 `json:"linear_y"`
		AngularZ    float64 `json:"angular_z"`
		ZoneID      string  `json:"zone_id"`
	}
	if len(task.Payload) > 0 {
		_ = json.Unmarshal(task.Payload, &taskPayload)
	}
	if taskPayload.DurationSec <= 0 {
		taskPayload.DurationSec = 30
	}

	if err := r.taskStore.UpdateStatus(task.ID, StatusRunning); err != nil {
		return
	}
	task.Status = StatusRunning
	if r.onTaskStarted != nil {
		r.onTaskStarted(task.ID, task.RobotID, task.ScenarioID, taskPayload.ZoneID)
	}

	// Zone coordination: acquire zone for patrol/navigation if zone_id in payload
	if r.coordinator != nil && taskPayload.ZoneID != "" && (task.ScenarioID == "patrol" || task.ScenarioID == "navigation") {
		if !r.coordinator.AcquireZone(task.RobotID, taskPayload.ZoneID) {
			slog.Warn("task-runner zone occupied", "zone_id", taskPayload.ZoneID, "task_id", task.ID)
			_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusFailed, time.Now())
			r.emitTaskCompleted(task.ID, task.RobotID, string(StatusFailed))
			return
		}
		if r.onZoneAcquired != nil {
			r.onZoneAcquired(task.RobotID, taskPayload.ZoneID, task.ID)
		}
		defer func() {
			if r.onZoneReleased != nil {
				r.onZoneReleased(task.RobotID, taskPayload.ZoneID, task.ID)
			}
			r.coordinator.ReleaseZone(task.RobotID, taskPayload.ZoneID)
		}()
	}

	for i, step := range scenario.Steps {
		// Check context cancellation (e.g. graceful shutdown) before each step
		select {
		case <-ctx.Done():
			now := time.Now()
			_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusCancelled, now)
			r.emitTaskCompleted(task.ID, task.RobotID, string(StatusCancelled))
			r.sendSafeStop(task.RobotID, task.OperatorID)
			return
		default:
		}
		// Check cancel before each step
		t, _ := r.taskStore.Get(task.ID)
		if t != nil && t.Status == StatusCancelled {
			now := time.Now()
			_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusCancelled, now)
			r.emitTaskCompleted(task.ID, task.RobotID, string(StatusCancelled))
			r.sendSafeStop(task.RobotID, task.OperatorID)
			return
		}

		payload := step.Payload
		durationSec := step.DurationSec
		if step.DurationSec == -1 {
			durationSec = taskPayload.DurationSec
		}
		if len(step.Payload) == 0 && step.Command == "cmd_vel" {
			payload = mustMarshal(map[string]float64{
				"linear_x":  taskPayload.LinearX,
				"linear_y":  taskPayload.LinearY,
				"angular_z": taskPayload.AngularZ,
			})
			if len(task.Payload) > 0 {
				payload = task.Payload
			}
		}

		cmd := &hal.Command{
			RobotID:    task.RobotID,
			Command:    step.Command,
			Payload:    payload,
			Timestamp:  time.Now(),
			OperatorID: task.OperatorID,
		}
		if !arbiter.SafetyAllow(cmd) {
			slog.Warn("task-runner safety rejected step", "step", i+1, "task_id", task.ID)
			_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusFailed, time.Now())
			r.emitTaskCompleted(task.ID, task.RobotID, string(StatusFailed))
			return
		}
		if err := r.bus.PublishCommand(cmd); err != nil {
			slog.Error("task-runner failed to publish command", "error", err)
			_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusFailed, time.Now())
			r.emitTaskCompleted(task.ID, task.RobotID, string(StatusFailed))
			return
		}

		if durationSec > 0 {
			deadline := time.Now().Add(time.Duration(durationSec) * time.Second)
			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					now := time.Now()
					_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusCancelled, now)
					r.emitTaskCompleted(task.ID, task.RobotID, string(StatusCancelled))
					r.sendSafeStop(task.RobotID, task.OperatorID)
					return
				default:
					t, _ := r.taskStore.Get(task.ID)
					if t != nil && t.Status == StatusCancelled {
						now := time.Now()
						_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusCancelled, now)
						r.emitTaskCompleted(task.ID, task.RobotID, string(StatusCancelled))
						r.sendSafeStop(task.RobotID, task.OperatorID)
						return
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
	}

	now := time.Now()
	_ = r.taskStore.UpdateStatusAndCompletedAt(task.ID, StatusCompleted, now)
	r.emitTaskCompleted(task.ID, task.RobotID, string(StatusCompleted))
	slog.Info("task-runner task completed", "task_id", task.ID)
}

func (r *Runner) emitTaskCompleted(taskID, robotID, status string) {
	if r.onTaskCompleted != nil {
		r.onTaskCompleted(taskID, robotID, status)
	}
}

func (r *Runner) sendSafeStop(robotID, operatorID string) {
	cmd := &hal.Command{
		RobotID:    robotID,
		Command:    "safe_stop",
		Timestamp:  time.Now(),
		OperatorID: operatorID,
	}
	if arbiter.SafetyAllow(cmd) {
		_ = r.bus.PublishCommand(cmd)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
