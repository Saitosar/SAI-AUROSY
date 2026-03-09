package tasks

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/coordinator"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func mustConnectBus(t *testing.T) *telemetry.Bus {
	bus, err := telemetry.NewBus("nats://localhost:4222")
	if err != nil {
		t.Skipf("NATS unavailable, skipping: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

func TestRunner_UnknownScenario_TaskFailed(t *testing.T) {
	taskStore := NewMemoryStore()
	taskStore.Create(&Task{
		ID:         "t1",
		RobotID:    "r1",
		ScenarioID: "nonexistent",
		Status:     StatusPending,
	})
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Capabilities: []string{hal.CapStand}})
	bus := mustConnectBus(t)
	scenarioCatalog := scenarios.NewCatalog()

	runner := NewRunner(taskStore, scenarioCatalog, reg, bus, RunnerConfig{})
	runner.RunOnce(context.Background())

	task, _ := taskStore.Get("t1")
	if task == nil || task.Status != StatusFailed {
		t.Errorf("expected task failed, got %v", task)
	}
}

func TestRunner_RobotLacksCapabilities_TaskFailed(t *testing.T) {
	taskStore := NewMemoryStore()
	taskStore.Create(&Task{
		ID:         "t1",
		RobotID:    "r1",
		ScenarioID: "standby",
		Status:     StatusPending,
	})
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Capabilities: []string{}}) // no stand capability
	bus := mustConnectBus(t)
	scenarioCatalog := scenarios.NewCatalog()

	runner := NewRunner(taskStore, scenarioCatalog, reg, bus, RunnerConfig{})
	runner.RunOnce(context.Background())

	task, _ := taskStore.Get("t1")
	if task == nil || task.Status != StatusFailed {
		t.Errorf("expected task failed (robot lacks capabilities), got %v", task)
	}
}

func TestRunner_ZoneOccupied_TaskFailed(t *testing.T) {
	coordStore := coordinator.NewStore()
	coordStore.RegisterZone(&coordinator.Zone{ID: "A", Name: "A"})
	coordStore.AcquireZone("other-robot", "A") // zone A already taken
	coord := coordinator.NewCoordinatorWithStore(coordStore)

	taskStore := NewMemoryStore()
	payload, _ := json.Marshal(map[string]any{"zone_id": "A", "duration_sec": 5})
	taskStore.Create(&Task{
		ID:         "t1",
		RobotID:    "r1",
		ScenarioID: "patrol",
		Payload:    payload,
		Status:     StatusPending,
	})
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Capabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol}})
	bus := mustConnectBus(t)
	scenarioCatalog := scenarios.NewCatalog()

	runner := NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, coord, RunnerConfig{})
	runner.RunOnce(context.Background())

	task, _ := taskStore.Get("t1")
	if task == nil || task.Status != StatusFailed {
		t.Errorf("expected task failed (zone occupied), got %v", task)
	}
}

func TestRunner_StandbyTask_CompletesAndPublishesCommands(t *testing.T) {
	taskStore := NewMemoryStore()
	taskStore.Create(&Task{
		ID:         "t1",
		RobotID:    "r1",
		ScenarioID: "standby",
		Status:     StatusPending,
	})
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Capabilities: []string{hal.CapStand}})
	bus := mustConnectBus(t)
	scenarioCatalog := scenarios.NewCatalog()

	var commandsMu sync.Mutex
	var commands []*hal.Command
	sub, err := bus.SubscribeCommands("r1", func(cmd *hal.Command) {
		commandsMu.Lock()
		commands = append(commands, cmd)
		commandsMu.Unlock()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	runner := NewRunner(taskStore, scenarioCatalog, reg, bus, RunnerConfig{PollInterval: time.Second})
	runner.RunOnce(context.Background())

	// Give NATS time to deliver
	time.Sleep(100 * time.Millisecond)

	task, _ := taskStore.Get("t1")
	if task == nil || task.Status != StatusCompleted {
		t.Errorf("expected task completed, got %v", task)
	}
	commandsMu.Lock()
	n := len(commands)
	commandsMu.Unlock()
	if n < 1 {
		t.Errorf("expected at least 1 command published, got %d", n)
	}
}

func TestRunner_TaskCancelled_SafeStopPublished(t *testing.T) {
	// Use patrol with short duration so runner blocks in cmd_vel step, giving us time to cancel
	payload, _ := json.Marshal(map[string]any{"zone_id": "", "duration_sec": 5})
	taskStore := NewMemoryStore()
	taskStore.Create(&Task{
		ID:         "t1",
		RobotID:    "r1",
		ScenarioID: "patrol",
		Payload:    payload,
		Status:     StatusPending,
	})
	reg := registry.NewMemoryStore()
	reg.Add(&hal.Robot{ID: "r1", Vendor: "test", Capabilities: []string{hal.CapWalk, hal.CapCmdVel, hal.CapPatrol}})
	bus := mustConnectBus(t)
	scenarioCatalog := scenarios.NewCatalog()

	var safeStopReceived bool
	sub, err := bus.SubscribeCommands("r1", func(cmd *hal.Command) {
		if cmd.Command == "safe_stop" {
			safeStopReceived = true
		}
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	runner := NewRunner(taskStore, scenarioCatalog, reg, bus, RunnerConfig{PollInterval: 100 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())

	go runner.Run(ctx)

	// Wait for task to be Running (runner picked it and is in cmd_vel step)
	for i := 0; i < 100; i++ {
		task, _ := taskStore.Get("t1")
		if task != nil && task.Status == StatusRunning {
			// Cancel and let runner detect it in its poll loop
			_ = taskStore.UpdateStatus("t1", StatusCancelled)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(800 * time.Millisecond) // runner polls every 500ms inside duration loop
	cancel()
	time.Sleep(200 * time.Millisecond)

	task, _ := taskStore.Get("t1")
	if task == nil || task.Status != StatusCancelled {
		t.Errorf("expected task cancelled, got %v", task)
	}
	if !safeStopReceived {
		t.Error("expected safe_stop command to be published")
	}
}
