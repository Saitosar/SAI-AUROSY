package orchestration

import (
	"context"
	"testing"

	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
)

func TestRun_CreatesTasksForWorkflow(t *testing.T) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", Capabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol}})
	reg.Add(&hal.Robot{ID: "r2", Vendor: "test", Model: "X", Capabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol}})
	reg.Add(&hal.Robot{ID: "r3", Vendor: "test", Model: "X", Capabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol}})

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "patrol_zones_ABC", "op1", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.WorkflowRunID == "" {
		t.Error("expected workflow_run_id")
	}
	if len(result.TaskIDs) != 3 {
		t.Errorf("expected 3 task IDs, got %d", len(result.TaskIDs))
	}
	run, _ := runStore.Get(result.WorkflowRunID)
	if run == nil || run.Status != WorkflowRunRunning {
		t.Errorf("run status: expected running, got %v", run)
	}
	for _, tid := range result.TaskIDs {
		task, _ := taskStore.Get(tid)
		if task == nil {
			t.Errorf("task %s not found", tid)
		}
		if task.Status != tasks.StatusPending {
			t.Errorf("task %s status: expected pending, got %s", tid, task.Status)
		}
	}
}

func TestRun_ErrWorkflowNotFound(t *testing.T) {
	reg := registry.NewMemoryStore()
	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	_, err := runner.Run(context.Background(), "nonexistent", "op1", "")
	if err != ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestRun_NoAvailableRobot_WorkflowFailed(t *testing.T) {
	reg := registry.NewMemoryStore()
	// No robots with patrol capability
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", Capabilities: []string{hal.CapStand}})

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "patrol_zones_ABC", "op1", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.WorkflowRunID == "" {
		t.Error("expected workflow_run_id")
	}
	if len(result.TaskIDs) != 0 {
		t.Errorf("expected 0 tasks when no robot available, got %d", len(result.TaskIDs))
	}
	run, _ := runStore.Get(result.WorkflowRunID)
	if run == nil || run.Status != WorkflowRunFailed {
		t.Errorf("run status: expected failed, got %v", run)
	}
}

func TestRun_UnknownScenario_WorkflowFailed(t *testing.T) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", Capabilities: []string{hal.CapStand}})

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	wfCatalog.Register(Workflow{
		ID:   "bad_scenario",
		Name: "Bad",
		Steps: []WorkflowStep{
			{RobotID: "r1", ScenarioID: "nonexistent_scenario"},
		},
	})
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "bad_scenario", "op1", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	run, _ := runStore.Get(result.WorkflowRunID)
	if run == nil || run.Status != WorkflowRunFailed {
		t.Errorf("run status: expected failed for unknown scenario, got %v", run)
	}
}

func TestRun_TenantIsolation(t *testing.T) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", TenantID: "t1", Capabilities: []string{hal.CapStand}})
	reg.Add(&hal.Robot{ID: "r2", Vendor: "test", Model: "X", TenantID: "t2", Capabilities: []string{hal.CapStand}})

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	wfCatalog.Register(Workflow{
		ID:   "single_standby",
		Name: "Single",
		Steps: []WorkflowStep{
			{ScenarioID: "standby"},
		},
	})
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "single_standby", "op1", "t1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.TaskIDs) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.TaskIDs))
	}
	task, _ := taskStore.Get(result.TaskIDs[0])
	if task == nil || task.RobotID != "r1" {
		t.Errorf("expected robot r1 (tenant t1), got %v", task)
	}
	// t2 has no robots with standby - actually t2 has r2 with CapStand. Standby requires CapStand. So r2 should work for t2.
	result2, _ := runner.Run(context.Background(), "single_standby", "op1", "t2")
	if len(result2.TaskIDs) != 1 {
		t.Fatalf("t2: expected 1 task, got %d", len(result2.TaskIDs))
	}
	task2, _ := taskStore.Get(result2.TaskIDs[0])
	if task2.RobotID != "r2" {
		t.Errorf("t2: expected robot r2, got %s", task2.RobotID)
	}
}

func TestRun_RobotWithRunningTaskSkipped(t *testing.T) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", Capabilities: []string{hal.CapStand}})

	taskStore := tasks.NewMemoryStore()
	// Pre-create a running task for r1
	taskStore.Create(&tasks.Task{ID: "running-1", RobotID: "r1", ScenarioID: "standby", Status: tasks.StatusRunning})

	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	wfCatalog.Register(Workflow{
		ID:   "single_standby",
		Name: "Single",
		Steps: []WorkflowStep{
			{ScenarioID: "standby"},
		},
	})
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "single_standby", "op1", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.TaskIDs) != 0 {
		t.Errorf("expected 0 tasks (r1 has running task), got %d", len(result.TaskIDs))
	}
	run, _ := runStore.Get(result.WorkflowRunID)
	if run.Status != WorkflowRunFailed {
		t.Errorf("run status: expected failed, got %s", run.Status)
	}
}

func TestRun_RobotSelectorResolves(t *testing.T) {
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Model: "X", Capabilities: []string{hal.CapStand, hal.CapWalk}})

	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()
	wfCatalog := NewCatalog()
	wfCatalog.Register(Workflow{
		ID:   "selector_workflow",
		Name: "Selector",
		Steps: []WorkflowStep{
			{
				RobotSelector: &RobotSelector{Capabilities: []string{hal.CapStand}},
				ScenarioID:   "standby",
			},
		},
	})
	runStore := NewMemoryRunStore()

	runner := NewRunner(wfCatalog, runStore, taskStore, scenarioCatalog, reg)
	result, err := runner.Run(context.Background(), "selector_workflow", "op1", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.TaskIDs) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.TaskIDs))
	}
	task, _ := taskStore.Get(result.TaskIDs[0])
	if task.RobotID != "r1" {
		t.Errorf("expected robot r1 from selector, got %s", task.RobotID)
	}
}
