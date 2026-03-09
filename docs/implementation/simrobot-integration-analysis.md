# Step 1: Simulated Robot Harness — Integration Analysis

## HAL (Hardware Abstraction Layer)

**Location:** `pkg/hal/`

| Component | Purpose |
|-----------|---------|
| `hal.Telemetry` | Normalized telemetry: `RobotID`, `Timestamp`, `Online`, `ActuatorStatus`, `CurrentTask`, `Position` (string "x,y,z"), `TargetPosition`, `DistanceToTarget` (*float64) |
| `hal.Command` | `RobotID`, `Command`, `Payload` (JSON), `Timestamp`, `OperatorID` |
| `hal.Robot` | Fleet registry entry: `ID`, `Vendor`, `Model`, `TenantID`, `Capabilities` |
| `RobotAdapter` | Interface for real adapters (Connect, SubscribeTelemetry, SendCommand, Disconnect) — simulator does NOT implement this |

**Key constraint:** Simulator must produce `hal.Telemetry` and consume `hal.Command` without changing HAL.

---

## Telemetry Bus

**Location:** `pkg/telemetry/bus.go`

| Topic | Direction | Usage |
|-------|-----------|-------|
| `telemetry.robots.{robot_id}` | Adapter → Platform | Simulator publishes here |
| `commands.robots.{robot_id}` | Platform → Adapter | Simulator subscribes here |

**Methods used:**
- `bus.SubscribeCommands(robotID, handler)` — simulator receives commands
- `bus.PublishTelemetry(t *hal.Telemetry)` — simulator publishes state

---

## Task Runner

**Location:** `pkg/control-plane/tasks/runner.go`

- Polls `taskStore.List(StatusPending)` every 2s
- For `mall_assistant`: delegates to `MallAssistantRunner.Run()`
- For `navigate_to_store` / `navigation`: delegates to `ExecutionEngine.ExecuteTaskEntry()`
- Requires robot in `registry.Get(robotID)` to run tasks

**Integration point:** Simulated robot must be in Fleet Registry (`reg.Add()`).

---

## Execution Engine

**Location:** `internal/robot/execution_engine.go`

- `ExecuteTaskEntry()` handles `navigate_to_store` and `navigation` (return-to-base)
- Delegates to `TaskExecutor.ExecuteTask()` which uses `NavigationExecutor.Execute()`
- `NavigationExecutor` sends `walk_mode` + `navigate_to` via `bus.PublishCommand()`
- Subscribes to `bus.SubscribeTelemetry(robotID)` and waits for `distance_to_target < 1.0` or timeout
- On failure: sends `safe_stop` via bus

**Critical:** Simulator must publish telemetry with `DistanceToTarget` decreasing over time; arrival when `*DistanceToTarget < 1.0`.

---

## Mall Assistant Flow

**Location:** `pkg/control-plane/mallassistant/handler.go`

1. `Run()` starts scenario, speaks greeting
2. Waits for visitor request via `requestRegistry`
3. `processRequest()` → Cognitive Gateway → intent `find_store`, store name
4. `mallService.FindStoreNode()`, `GetBasePoint()`, `CalculateRoute()`
5. Creates `navigate_to_store` task with `target_coordinates`, `destination_node_id`, `route`, `estimated_distance`
6. `waitForNavigation()` polls task status
7. On completion: speaks arrival, creates return task with `target_coordinates` = base
8. Return task uses scenario `navigation` with base coordinates

**Payload format:** `target_coordinates` is `"x,y,0"` (e.g. `"15.00,5.00,0"`) per `mall.Coordinates.String()`.

---

## Navigation Executor

**Location:** `internal/robot/navigation_executor.go`

- Sends `walk_mode` then `navigate_to` with `target_coordinates`, `store_name`, `destination_node_id`
- Subscribes to telemetry; success when `t.DistanceToTarget != nil && *t.DistanceToTarget < 1.0`
- Failure when `!t.Online` or timeout
- Uses `arbiter.SafetyAllow()` before publishing (navigate_to requires non-empty `target_coordinates`)

---

## Fleet Registry

**Location:** `pkg/control-plane/registry/store.go`

- `Store.Add(r *hal.Robot)` — register robot
- `Store.Get(id)` — Task Runner checks robot exists before running
- Robot needs `Capabilities` for scenario assignment (e.g. `navigation`, `speech`)

---

## Summary: Minimal Touch Points

| Component | Simulator Action |
|-----------|------------------|
| Fleet Registry | `reg.Add(&hal.Robot{...})` with `sim-` prefix |
| Telemetry Bus | `SubscribeCommands(robotID, handler)`; `PublishTelemetry(t)` |
| HAL types | Consume `hal.Command`; produce `hal.Telemetry` |
| No changes | Task Runner, Execution Engine, Mall Assistant, Navigation Executor |
