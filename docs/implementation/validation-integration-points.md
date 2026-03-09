# Validation Integration Points

This document summarizes the integration points between the SAI AUROSY validation layer and the platform components. It serves as a reference for understanding how validation exercises the system and what touch points exist.

## Overview

The validation layer runs end-to-end scenarios against the platform using the Simulated Robot Harness. It does not modify HAL, Execution Engine, Task Runner, or Operator Console internals. Validation orchestrates existing components and asserts outcomes.

## Integration Points Summary

| Integration Point | Location | Validation Touch |
|-------------------|----------|------------------|
| **HAL** | `pkg/hal/` (types, adapter interface) | SimRobot produces `hal.Telemetry`, consumes `hal.Command`; contract validates telemetry fields |
| **Execution Engine** | `internal/robot/` (execution_engine, navigation_executor) | Receives commands via bus; arrival detection uses `distance_to_target < 1.0`; validation asserts final state |
| **Simulated Robot Harness** | `internal/simrobot/` | Validation uses `SimRobotService` for reset, failure injection, state; subscribes to telemetry |
| **Mall Assistant** | `pkg/control-plane/mallassistant/handler.go` | Validation triggers via `StartMallAssistant` + `SubmitVisitorRequest`; asserts event sequence |
| **Task Runner** | `pkg/control-plane/tasks/runner.go` | Runs in validation; delegates mall_assistant and navigation tasks |
| **Telemetry Bus** | `pkg/telemetry/bus.go` | NATS `telemetry.robots.{id}`, `commands.robots.{id}`; validation subscribes for telemetry collection |
| **Operator Console** | `pkg/operator-console/` | Not used by validation; validation runs headless; commands flow via API/bus |

## Detailed Integration Points

### HAL (Hardware Abstraction Layer)

**Location:** `pkg/hal/`

| Component | Purpose |
|-----------|---------|
| `hal.Telemetry` | Normalized telemetry: RobotID, Timestamp, Online, ActuatorStatus, CurrentTask, Position, TargetPosition, DistanceToTarget |
| `hal.Command` | RobotID, Command, Payload, Timestamp, OperatorID |
| `hal.Robot` | Fleet registry entry with Capabilities |
| `RobotAdapter` | Interface for real adapters; SimRobot does NOT implement this |

**Validation touch:** SimRobot publishes `hal.Telemetry` and consumes `hal.Command` via the Telemetry Bus. The adapter contract (`ValidateAdapterContract`) checks that telemetry samples have required fields per the readiness contract.

### Execution Engine

**Location:** `internal/robot/` (execution_engine.go, navigation_executor.go, task_executor.go)

- Handles `navigate_to_store` and `navigation` (return-to-base) tasks
- Publishes `walk_mode` and `navigate_to` via `bus.PublishCommand()`
- Subscribes to `bus.SubscribeTelemetry(robotID)` and waits for `distance_to_target < 1.0` or timeout
- On success/failure: updates state, broadcasts events, updates task status

**Validation touch:** Validation asserts `final_robot_state` via `RobotStateProvider.GetRobotState(robotID)` which reads from the Execution Engine's state manager. Validation does not modify the engine.

### Simulated Robot Harness

**Location:** `internal/simrobot/`

- `SimRobotService`: CreateRobot, Start, Stop, Reset, InjectFailure, GetState
- Subscribes to `commands.robots.{robot_id}`; publishes to `telemetry.robots.{robot_id}`
- Registers robots in Fleet Registry with prefix `sim-`

**Validation touch:** Validation uses `SimRobotService` for setup (Reset, InjectFailure), state inspection (GetState), and telemetry collection. The validation context wraps SimRobotService via `SimRobotService` interface.

### Mall Assistant Scenario

**Location:** `pkg/control-plane/mallassistant/handler.go`

- Implements `MallAssistantRunner`; invoked by Task Runner for `mall_assistant` tasks
- Flow: greeting → wait for visitor request → resolve store via Cognitive Gateway → create navigate_to_store task → wait for completion → create return task
- Broadcasts events: `visitor_interaction_started`, `mall_store_resolved`, `navigation_started`, `navigation_completed`, `visitor_interaction_finished`

**Validation touch:** Validation triggers via `MallAssistantTrigger.StartMallAssistant()` and `SubmitVisitorRequest()`. Asserts `event_sequence` and `task_status` for mall_assistant and navigation tasks.

### Task Runner

**Location:** `pkg/control-plane/tasks/runner.go`

- Polls `taskStore.List(StatusPending)` every 2 seconds
- Delegates `mall_assistant` → MallAssistantRunner; `navigate_to_store`/`navigation` → ExecutionEngine
- Requires robot in Fleet Registry

**Validation touch:** Task Runner runs in validation process. Validation creates tasks via TaskStore; Task Runner picks them up and delegates. No changes to Task Runner.

### Telemetry Bus

**Location:** `pkg/telemetry/bus.go`

| Topic | Direction | Usage |
|-------|-----------|-------|
| `telemetry.robots.{robot_id}` | Adapter → Platform | SimRobot publishes; validation subscribes for collection |
| `commands.robots.{robot_id}` | Platform → Adapter | Execution Engine, Task Runner publish; SimRobot subscribes |

**Validation touch:** Validation subscribes to `telemetry.robots.sim-001` via `bus.SubscribeTelemetry()` and collects samples in `TelemetryCollector` for assertion evaluation.

### Operator Console

**Location:** `pkg/operator-console/`

- Web UI for robots, tasks, telemetry, Mall Assistant controls
- Commands via `POST /api/v1/robots/{id}/command`

**Validation touch:** Not used by validation. Validation runs headless. Commands flow through the same bus when validation triggers scenarios (e.g. via MallAssistantTrigger, not via HTTP).

## Validation Flow

```
Validation CLI
    │
    ├── SimRobotService (reset, inject failure, start)
    ├── TaskRunner (run loop)
    ├── MallAssistantTrigger (start scenario, submit visitor request)
    ├── EventCollector (subscribe to EventBroadcaster)
    ├── TelemetryCollector (subscribe to bus.SubscribeTelemetry)
    ├── RobotStateProvider (Execution Engine state)
    └── TaskStore (create tasks, list for assertions)
```

## Related Documents

- [Validation Layer](../architecture/validation-layer.md)
- [Simulated Robot Harness](../architecture/simulated-robot-harness.md)
- [SimRobot Integration Analysis](simrobot-integration-analysis.md)
- [Adapter Readiness Contract](../contracts/adapter-readiness-contract.md)
