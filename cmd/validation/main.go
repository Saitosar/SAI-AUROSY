package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/internal/mall"
	"github.com/sai-aurosy/platform/internal/robot"
	"github.com/sai-aurosy/platform/internal/simrobot"
	"github.com/sai-aurosy/platform/internal/validation"
	"github.com/sai-aurosy/platform/pkg/control-plane/cognitive"
	"github.com/sai-aurosy/platform/pkg/control-plane/events"
	"github.com/sai-aurosy/platform/pkg/control-plane/mallassistant"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/control-plane/scenarios"
	"github.com/sai-aurosy/platform/pkg/control-plane/tasks"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func main() {
	scenarioName := flag.String("scenario", "", "Run single scenario (name); empty = all")
	scenarioDir := flag.String("scenario-dir", "testdata/validation", "Scenario YAML directory")
	outputDir := flag.String("output-dir", "outputs/validation", "Output directory for reports")
	contractCheck := flag.Bool("contract-check", false, "Run adapter contract validation after each scenario")
	outputContract := flag.String("output-contract", "", "Write adapter contract JSON to path and exit (e.g. outputs/validation/adapter_contract.json)")
	flag.Parse()

	if *outputContract != "" {
		spec := validation.DefaultContractSpec()
		data, err := validation.ContractSpecToJSON(spec)
		if err != nil {
			log.Fatalf("contract spec to JSON: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(*outputContract), 0755); err != nil {
			log.Fatalf("create output dir: %v", err)
		}
		if err := os.WriteFile(*outputContract, data, 0644); err != nil {
			log.Fatalf("write contract: %v", err)
		}
		log.Printf("wrote adapter contract to %s", *outputContract)
		os.Exit(0)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	bus, err := telemetry.NewBus(natsURL)
	if err != nil {
		log.Fatalf("NATS connection failed: %v (ensure NATS is running, e.g. docker compose up -d)", err)
	}
	defer bus.Close()

	reg := registry.NewMemoryStore()
	taskStore := tasks.NewMemoryStore()
	scenarioCatalog := scenarios.NewCatalog()

	eventBroadcaster := events.NewBroadcaster()
	eventCollector := validation.NewEventCollector()
	telemetryCollector := validation.NewTelemetryCollector()

	eventSub := eventBroadcaster.Subscribe()
	go func() {
		for e := range eventSub {
			eventCollector.Collect(e.Type, e.Data)
		}
	}()
	defer eventBroadcaster.Unsubscribe(eventSub)

	cogCfg, err := cognitive.LoadConfig()
	if err != nil {
		log.Fatalf("cognitive config: %v", err)
	}
	cogGateway, err := cognitive.NewGateway(*cogCfg)
	if err != nil {
		log.Fatalf("cognitive gateway: %v", err)
	}

	mallRequestRegistry := mallassistant.NewVisitorRequestRegistry()
	mallMapPath := resolveMallMapPath()
	mallRepo := mall.NewMemoryRepository(mallMapPath)
	mallService := mall.NewService(mallRepo)

	mallAssistantHandler := mallassistant.NewHandler(bus, cogGateway, taskStore, eventBroadcaster, mallRequestRegistry, mallassistant.HandlerConfig{
		MallService: mallService,
		OnTaskCompleted: func(taskID, robotID, status string) {
			eventBroadcaster.Broadcast("task_completed", map[string]any{"task_id": taskID, "robot_id": robotID, "status": status})
		},
	})

	stateManager := robot.NewStateManager()
	navExecutor := robot.NewNavigationExecutor(bus, stateManager, 1.0, 60*time.Second)
	taskExecutor := robot.NewTaskExecutor(navExecutor)
	executionEngine := robot.NewExecutionEngine(stateManager, taskExecutor, bus, robot.ExecutionEngineConfig{
		EventBroadcaster:    eventBroadcaster,
		TaskStore:          taskStore,
		TimeoutAsCompletion: true,
		OnTaskCompleted: func(taskID, robotID, status string) {
			eventBroadcaster.Broadcast("task_completed", map[string]any{"task_id": taskID, "robot_id": robotID, "status": status})
		},
	})

	taskRunner := tasks.NewRunnerWithCoordinator(taskStore, scenarioCatalog, reg, bus, nil, tasks.RunnerConfig{
		MallAssistantRunner: mallAssistantHandler,
		ExecutionEngine:     executionEngine,
		OnTaskCompleted: func(taskID, robotID, status string) {
			eventBroadcaster.Broadcast("task_completed", map[string]any{"task_id": taskID, "robot_id": robotID, "status": status})
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskRunner.Run(ctx)

	simRobotService := simrobot.NewSimRobotService(bus, reg)
	if _, err := simRobotService.CreateRobot(simrobot.CreateRobotOpts{
		RobotID:   "sim-001",
		TenantID:  "default",
		RobotType: "simulated",
	}); err != nil {
		log.Printf("WARNING: simrobot create: %v", err)
	}
	if err := simRobotService.Start(ctx, "sim-001"); err != nil {
		log.Fatalf("simrobot start: %v", err)
	}

	teleSub, err := bus.SubscribeTelemetry("sim-001", func(t *hal.Telemetry) {
		telemetryCollector.Collect(t)
	})
	if err != nil {
		log.Printf("WARNING: telemetry subscribe: %v", err)
	} else {
		defer teleSub.Unsubscribe()
	}

	vctx := buildValidationContext(simRobotService, taskStore, eventCollector, telemetryCollector, mallAssistantHandler, executionEngine, reg)
	cfg := validation.RunConfig{
		ScenarioDir:    *scenarioDir,
		OutputDir:      *outputDir,
		TimeoutPerStep: 90 * time.Second,
		ContractCheck:  *contractCheck,
	}

	if *scenarioName != "" {
		scenario, err := validation.LoadScenario(filepath.Join(*scenarioDir, *scenarioName+".yaml"))
		if err != nil {
			log.Fatalf("load scenario: %v", err)
		}
		report, err := validation.RunScenario(ctx, vctx, scenario, cfg)
		if err != nil {
			log.Fatalf("run scenario: %v", err)
		}
		printReport(report)
		if _, err := validation.WriteReportJSON(report, *outputDir); err != nil {
			log.Printf("write report: %v", err)
		}
		if _, err := validation.WriteReportMarkdown(report, *outputDir); err != nil {
			log.Printf("write markdown: %v", err)
		}
		os.Exit(exitCode(report))
	}

	reports, err := validation.RunAllScenarios(ctx, vctx, cfg)
	if err != nil {
		log.Fatalf("run scenarios: %v", err)
	}
	for _, r := range reports {
		printReport(r)
		if _, err := validation.WriteReportJSON(r, *outputDir); err != nil {
			log.Printf("write %s: %v", r.ScenarioName, err)
		}
	}
	if _, err := validation.WriteSummaryMarkdown(reports, *outputDir); err != nil {
		log.Printf("write summary: %v", err)
	}
	failed := 0
	for _, r := range reports {
		if r.Status == "FAIL" || len(r.ContractViolations) > 0 {
			failed++
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func buildValidationContext(
	sim *simrobot.SimRobotService,
	ts tasks.Store,
	ev *validation.EventCollectorImpl,
	tel *validation.TelemetryCollectorImpl,
	handler *mallassistant.Handler,
	engine *robot.RobotExecutionEngine,
	reg *registry.MemoryStore,
) *validation.ValidationContext {
	return &validation.ValidationContext{
		SimRobotService:      &simRobotAdapter{sim: sim},
		TaskStore:            &taskStoreAdapter{store: ts},
		EventCollector:       ev,
		TelemetryCollector:   tel,
		MallAssistantTrigger: &mallTriggerAdapter{handler: handler, taskStore: ts},
		RobotStateProvider:   &stateAdapter{engine: engine},
		RobotRegistry:        reg,
		Bus:                  nil,
	}
}

func printReport(r *validation.Report) {
	log.Printf("[%s] %s - %d passed, %d failed (%d ms)", r.Status, r.ScenarioName, r.AssertionsPassed, r.AssertionsFailed, r.DurationMs)
	if r.Error != "" {
		log.Printf("  Error: %s", r.Error)
	}
	for _, res := range r.Results {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		log.Printf("  %s %s: %s", status, res.AssertionType, res.Message)
	}
	for _, v := range r.ContractViolations {
		log.Printf("  CONTRACT %s: %s %s", v.Category, v.Field, v.Message)
	}
}

func exitCode(r *validation.Report) int {
	if r.Status != "PASS" {
		return 1
	}
	if len(r.ContractViolations) > 0 {
		return 1
	}
	return 0
}

type simRobotAdapter struct {
	sim *simrobot.SimRobotService
}

func (a *simRobotAdapter) Reset(robotID string) error {
	return a.sim.Reset(robotID)
}

func (a *simRobotAdapter) InjectFailure(robotID string, cfg *validation.SimRobotFailureConfig) error {
	if cfg == nil {
		return nil
	}
	fc := &simrobot.FailureConfig{
		Type:           simrobot.FailureType(cfg.Type),
		AfterTicks:      cfg.AfterTicks,
		WhenDistanceLt:  cfg.WhenDistanceLt,
		AfterCommand:    cfg.AfterCommand,
		BatteryLevel:    cfg.BatteryLevel,
		SlowdownFactor:  cfg.SlowdownFactor,
		DurationTicks:   cfg.DurationTicks,
	}
	return a.sim.InjectFailure(robotID, fc)
}

func (a *simRobotAdapter) GetState(robotID string) (*validation.SimRobotState, error) {
	s, err := a.sim.GetState(robotID)
	if err != nil || s == nil {
		return nil, err
	}
	return &validation.SimRobotState{
		RobotID:          s.RobotID,
		Online:           s.Online,
		Mode:             string(s.Mode),
		Position:         formatPosition(s.Position),
		DistanceToTarget: s.DistanceToTarget,
	}, nil
}

func (a *simRobotAdapter) GetRobot(robotID string) validation.SimRobot {
	r := a.sim.GetRobot(robotID)
	if r == nil {
		return nil
	}
	return &simRobotWrapper{robot: r}
}

func (a *simRobotAdapter) Start(ctx interface{}, robotID string) error {
	if c, ok := ctx.(context.Context); ok {
		return a.sim.Start(c, robotID)
	}
	return a.sim.Start(context.Background(), robotID)
}

func (a *simRobotAdapter) Stop(robotID string) error {
	return a.sim.Stop(robotID)
}

func formatPosition(p simrobot.Position) string {
	return fmtPosition(p.X, p.Y)
}

func fmtPosition(x, y float64) string {
	return fmt.Sprintf("%.2f,%.2f,0", x, y)
}

type simRobotWrapper struct {
	robot *simrobot.SimRobot
}

func (w *simRobotWrapper) State() validation.SimRobotState {
	s := w.robot.State()
	return validation.SimRobotState{
		RobotID:          s.RobotID,
		Online:           s.Online,
		Mode:             string(s.Mode),
		Position:         formatPosition(s.Position),
		DistanceToTarget: s.DistanceToTarget,
	}
}

type taskStoreAdapter struct {
	store tasks.Store
}

func (a *taskStoreAdapter) Create(task *validation.Task) error {
	t := &tasks.Task{
		ID:         task.ID,
		RobotID:    task.RobotID,
		ScenarioID: task.ScenarioID,
		Status:     tasks.Status(task.Status),
		Payload:    task.Payload,
	}
	return a.store.Create(t)
}

func (a *taskStoreAdapter) Get(id string) (*validation.Task, error) {
	t, err := a.store.Get(id)
	if err != nil || t == nil {
		return nil, err
	}
	return &validation.Task{
		ID:         t.ID,
		RobotID:    t.RobotID,
		ScenarioID: t.ScenarioID,
		Status:     string(t.Status),
		Payload:    t.Payload,
	}, nil
}

func (a *taskStoreAdapter) List(filters validation.TaskListFilters) ([]validation.Task, error) {
	lf := tasks.ListFilters{RobotID: filters.RobotID, Status: tasks.Status(filters.Status)}
	list, err := a.store.List(lf)
	if err != nil {
		return nil, err
	}
	out := make([]validation.Task, 0, len(list))
	for _, t := range list {
		out = append(out, validation.Task{
			ID:         t.ID,
			RobotID:    t.RobotID,
			ScenarioID: t.ScenarioID,
			Status:     string(t.Status),
			Payload:    t.Payload,
		})
	}
	return out, nil
}

type mallTriggerAdapter struct {
	handler   *mallassistant.Handler
	taskStore tasks.Store
}

func (a *mallTriggerAdapter) StartMallAssistant(robotID, tenantID, operatorID string) (string, error) {
	t := &tasks.Task{
		ID:         mustUUID(),
		RobotID:    robotID,
		TenantID:   tenantID,
		Type:       "scenario",
		ScenarioID: "mall_assistant",
		Status:     tasks.StatusPending,
		OperatorID: operatorID,
	}
	if err := a.taskStore.Create(t); err != nil {
		return "", err
	}
	return t.ID, nil
}

func (a *mallTriggerAdapter) SubmitVisitorRequest(robotID, text string) (bool, error) {
	return a.handler.SubmitVisitorRequest(robotID, text), nil
}

func mustUUID() string {
	return uuid.New().String()
}

// resolveMallMapPath returns the first path that exists. Tries project root and relative paths.
func resolveMallMapPath() string {
	candidates := []string{
		"scenarios/data/mall_map.json",
		filepath.Join("..", "..", "scenarios/data/mall_map.json"),
		filepath.Join("..", "scenarios/data/mall_map.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if p != candidates[0] {
				log.Printf("mall map: using fallback path %s", p)
			}
			return p
		}
	}
	// Return default; NewMemoryRepository falls back to minimal map if file missing
	return candidates[0]
}

type stateAdapter struct {
	engine *robot.RobotExecutionEngine
}

func (a *stateAdapter) GetRobotState(robotID string) *validation.RobotStateResponse {
	r := a.engine.GetRobotState(robotID)
	if r == nil {
		return nil
	}
	return &validation.RobotStateResponse{
		RobotID:       r.RobotID,
		State:         r.State,
		CurrentTask:   r.CurrentTask,
		Destination:   r.Destination,
		TargetStore:   r.TargetStore,
		StatusMessage: r.StatusMessage,
	}
}

