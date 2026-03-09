package orchestration

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
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
func (r *Runner) Run(workflowID, operatorID string) (*RunResult, error) {
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
			robotID = r.resolveSelector(step.RobotSelector, usedRobots)
		}
		if robotID == "" {
			robotID = r.pickAvailableRobot(step.ScenarioID, usedRobots)
		}
		if robotID == "" {
			log.Printf("[orchestration] no available robot for step %d of workflow %s", i, workflowID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		scenario, ok := r.scenarioCatalog.Get(step.ScenarioID)
		if !ok {
			log.Printf("[orchestration] scenario %s not found", step.ScenarioID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		robot := r.registry.Get(robotID)
		if !hal.HasCapability(robot, scenario.RequiredCapabilities) {
			log.Printf("[orchestration] robot %s lacks capabilities for scenario %s", robotID, step.ScenarioID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		hasRunning, _ := r.taskStore.HasRunningForRobot(robotID)
		if hasRunning {
			log.Printf("[orchestration] robot %s already has running task", robotID)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		payload := step.Payload
		if len(payload) == 0 && step.ZoneID != "" {
			payload, _ = json.Marshal(map[string]any{"zone_id": step.ZoneID})
		}
		t := &tasks.Task{
			ID:         uuid.New().String(),
			RobotID:    robotID,
			Type:       "scenario",
			ScenarioID: step.ScenarioID,
			Payload:    payload,
			Status:     tasks.StatusPending,
			OperatorID: operatorID,
		}
		if err := r.taskStore.Create(t); err != nil {
			log.Printf("[orchestration] failed to create task: %v", err)
			_ = r.runStore.UpdateStatus(run.ID, WorkflowRunFailed)
			return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
		}
		_ = r.runStore.AddTask(run.ID, t.ID, i)
		taskIDs = append(taskIDs, t.ID)
		usedRobots[robotID] = true
	}
	return &RunResult{WorkflowRunID: run.ID, TaskIDs: taskIDs}, nil
}

func (r *Runner) resolveSelector(sel *RobotSelector, used map[string]bool) string {
	robots := r.registry.List()
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

func (r *Runner) pickAvailableRobot(scenarioID string, used map[string]bool) string {
	scenario, ok := r.scenarioCatalog.Get(scenarioID)
	if !ok {
		return ""
	}
	robots := r.registry.List()
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
