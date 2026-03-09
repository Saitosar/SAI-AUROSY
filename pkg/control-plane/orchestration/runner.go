package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/pkg/control-plane/observability"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"go.opentelemetry.io/otel/attribute"
)

// Runner runs workflows by creating tasks and tracking runs.
type Runner struct {
	workflowCatalog *Catalog
	runStore        RunStore
	taskStore       tasks.Store
	scenarioCatalog *scenarios.Catalog
	registry        registry.Store
}

// NewRunner creates a workflow runner.
func NewRunner(wfCatalog *Catalog, runStore RunStore, taskStore tasks.Store, scenarioCatalog *scenarios.Catalog, reg registry.Store) *Runner {
	return &Runner{
		workflowCatalog: wfCatalog,
		runStore:        runStore,
		taskStore:       taskStore,
		scenarioCatalog: scenarioCatalog,
		registry:        reg,
	}
}

// RunResult is the result of starting a workflow run.
type RunResult struct {
	WorkflowRunID string   `json:"workflow_run_id"`
	TaskIDs       []string `json:"task_ids"`
}

// ErrWorkflowNotFound is returned when the workflow does not exist.
var ErrWorkflowNotFound = errors.New("workflow not found")

// Run starts a workflow by creating tasks for each step.
// Resolves robot_selector to pick available robots with required capabilities.
// When tenantID is non-empty, only robots from that tenant are considered.
func (r *Runner) Run(ctx context.Context, workflowID, operatorID, tenantID string) (*RunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, end := observability.StartSpan(ctx, "workflow.run",
		attribute.String("workflow_id", workflowID),
		attribute.String("operator_id", operatorID),
		attribute.String("tenant_id", tenantID),
	)
	defer end()

	wf, ok := r.workflowCatalog.Get(workflowID)
	if !ok {
		return nil, ErrWorkflowNotFound
	}
	run := &WorkflowRun{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Status:     WorkflowRunRunning,
	}
	if err := r.runStore.Create(run); err != nil {
		return nil, err
	}
	var taskIDs []string
	usedRobots := make(map[string]bool)
	for i, step := range wf.Steps {
		robotID := step.RobotID
		if robotID == "" && step.RobotSelector != nil {
			robotID = r.resolveSelector(step.RobotSelector, usedRobots, tenantID)
		}
		if robotID == "" {
			robotID = r.pickAvailableRobot(step.ScenarioID, usedRobots, tenantID)
		}
		if robotID != "" && tenantID != "" {
			robot := r.registry.Get(robotID)
			if robot != nil && robot.TenantID != tenantID {
				robotID = "" // reject robot from other tenant
			}
		}
		if robotID == "" {
			slog.Warn("orchestration no available robot", "step", i, "workflow_id", workflowID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		scenario, ok := r.scenarioCatalog.Get(step.ScenarioID)
		if !ok {
			slog.Warn("orchestration scenario not found", "scenario_id", step.ScenarioID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		robot := r.registry.Get(robotID)
		if !hal.HasCapability(robot, scenario.RequiredCapabilities) {
			slog.Warn("orchestration robot lacks capabilities", "robot_id", robotID, "scenario_id", step.ScenarioID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		hasRunning, _ := r.taskStore.HasRunningForRobot(robotID)
		if hasRunning {
			slog.Warn("orchestration robot already has running task", "robot_id", robotID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		if run.TenantID == "" {
			run.TenantID = robot.TenantID
			if run.TenantID == "" {
				run.TenantID = "default"
			}
			_ = r.runStore.UpdateTenantID(run.ID, run.TenantID)
		}
		taskTenantID := robot.TenantID
		if taskTenantID == "" {
			taskTenantID = "default"
		}
		payload := step.Payload
		if len(payload) == 0 && step.ZoneID != "" {
			payload, _ = json.Marshal(map[string]any{"zone_id": step.ZoneID})
		}
		t := &tasks.Task{
			ID:         uuid.New().String(),
			RobotID:    robotID,
			TenantID:   taskTenantID,
			Type:       "scenario",
			ScenarioID: step.ScenarioID,
			Payload:    payload,
			Status:     tasks.StatusPending,
			OperatorID: operatorID,
		}
		if err := r.taskStore.Create(t); err != nil {
			slog.Error("orchestration failed to create task", "error", err)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		_ = r.runStore.AddTask(run.ID, t.ID, i)
		taskIDs = append(taskIDs, t.ID)
		usedRobots[robotID] = true
	}
	return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
}

func (r *Runner) resolveSelector(sel *RobotSelector, used map[string]bool, tenantID string) string {
	var robots []hal.Robot
	if tenantID != "" {
		robots = r.registry.ListByTenant(tenantID)
	} else {
		robots = r.registry.List()
	}
	for _, robot := range robots {
		if used[robot.ID] {
			continue
		}
		if !hal.HasCapability(&robot, sel.Capabilities) {
			continue
		}
		hasRunning, _ := r.taskStore.HasRunningForRobot(robot.ID)
		if hasRunning {
			continue
		}
		return robot.ID
	}
	return ""
}

func (r *Runner) pickAvailableRobot(scenarioID string, used map[string]bool, tenantID string) string {
	scenario, ok := r.scenarioCatalog.Get(scenarioID)
	if !ok {
		return ""
	}
	var robots []hal.Robot
	if tenantID != "" {
		robots = r.registry.ListByTenant(tenantID)
	} else {
		robots = r.registry.List()
	}
	for _, robot := range robots {
		if used[robot.ID] {
			continue
		}
		if !hal.HasCapability(&robot, scenario.RequiredCapabilities) {
			continue
		}
		hasRunning, _ := r.taskStore.HasRunningForRobot(robot.ID)
		if hasRunning {
			continue
		}
		return robot.ID
	}
	return ""
}
