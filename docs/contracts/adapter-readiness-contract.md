# Adapter Readiness Contract

This document defines the formal contract that a real robot adapter (AGIBOT, UNITREE, ROS, or custom) must satisfy before being considered pilot-ready for the SAI AUROSY platform.

## Purpose

- Reduce ambiguity for robot vendors and integration partners
- Enable deterministic validation before live deployment
- Ensure Mall Assistant and other scenarios work correctly with real hardware

---

## 1. Mandatory Commands

The adapter MUST support and correctly handle the following commands when the corresponding capability is declared:

| Command | Required When | Description |
|---------|---------------|-------------|
| `navigate_to` | `navigation` capability | Start navigation toward target coordinates. Payload: `{"target_coordinates": "x,y,z", "store_name": "...", "destination_node_id": "..."}` |
| `safe_stop` | `safe_stop` capability | Emergency stop; robot transitions to idle (no torque). Highest priority. |
| `release_control` | `release_control` capability | Release platform control; operator takes over via joystick. |
| `walk_mode` | `walk` capability | Enter walking mode. |
| `stand_mode` | `stand` capability | Enter standing pose. |
| `speak` | `speech` capability | TTS output. Payload: `{"text": "..."}`. Audio may be sent via `audio.robots.{id}.output`. |

### Command Handling Rules

- Adapter MUST NOT silently ignore safety-related commands (`safe_stop`, `release_control`).
- Adapter MUST acknowledge or execute `safe_stop` with highest priority and lowest latency.
- Unknown commands MAY be logged and ignored; known-but-failed commands SHOULD be reported (e.g. via telemetry or error event).

---

## 2. Mandatory Telemetry Fields

Telemetry MUST be published to `telemetry.robots.{robot_id}` with the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `robot_id` | string | Yes | Must match robot ID in Fleet Registry. |
| `timestamp` | string (ISO8601) | Yes | UTC timestamp of the reading. |
| `online` | boolean | Yes | Robot is connected and responsive. |
| `current_task` | string | Yes | Heuristic: `idle`, `stand`, `walk` (from last mode). |
| `position` | string | Yes* | Format: `"x,y,z"` in meters. Required for arrival detection. |
| `target_position` | string | Yes* | Format: `"x,y,z"`. Required when navigating. |
| `distance_to_target` | number | Yes* | Meters to target. Required for arrival detection. Nil if not navigating. |
| `actuator_status` | string | Yes | One of: `enabled`, `disabled`, `error`, `calibration`. |

\* Required for Mall Assistant / navigation scenarios. Adapters without position sensing cannot support navigation.

### Optional but Recommended

| Field | Type | Description |
|-------|------|-------------|
| `battery` | number | Battery level 0-100. |
| `mock_mode` | boolean | True if adapter runs without real robot. |

---

## 3. Capability Truth Requirements

- Declared capabilities MUST reflect real behavior.
- If `speech=true`, the `speak` command (or audio output) MUST work.
- If `navigation=true`, `navigate_to` MUST be executable and telemetry MUST include `position`, `target_position`, `distance_to_target`.
- If `release_control` is declared, manual override path MUST exist.
- Do NOT declare capabilities the robot cannot fulfill.

---

## 4. Failure Signaling Requirements

The adapter MUST signal the following conditions via telemetry or events:

| Condition | Signal |
|-----------|--------|
| Robot offline | `online: false` in telemetry |
| Command rejected | Log; optionally include in telemetry or error event |
| Emergency stop / safe stop | `current_task: "idle"` or mode change; `actuator_status` if applicable |
| Timeout / stalled navigation | `distance_to_target` not decreasing; platform will timeout |
| Low battery | `battery` field when available |

---

## 5. Control and Safety Behavior

- **Operator/manual control MUST override automation.** When operator takes control (e.g. via `release_control`), platform automation must not override operator commands.
- **Safety commands MUST override non-safety commands.** `safe_stop` takes precedence over `navigate_to`, `cmd_vel`, etc.
- **Adapter MUST NOT silently ignore safety-related commands.** Reject with error if unable to execute, but do not ignore.

---

## 6. Timing Expectations

| Expectation | Value |
|-------------|-------|
| Telemetry publishing interval | ≤ 2 seconds when robot is active |
| Stale telemetry threshold | Platform may consider robot offline if no telemetry for > 5 seconds |
| Command effect observation | Adapter should reflect command effect in next telemetry (within 2 seconds) |
| Command acknowledgement | Adapter may acknowledge via telemetry (`current_task`, `actuator_status`) or implicit execution |

---

## 7. NATS Topics

| Topic | Direction | Description |
|-------|-----------|-------------|
| `telemetry.robots.{robot_id}` | Adapter → Platform | Telemetry updates |
| `commands.robots.{robot_id}` | Platform → Adapter | Commands |
| `audio.robots.{robot_id}.input` | Adapter → Platform | Microphone input (speech) |
| `audio.robots.{robot_id}.output` | Platform → Adapter | TTS output (speech) |

---

## Related Documents

- [Adapter Validation Checklist](adapter-validation-checklist.md)
- [RobotAdapter Contract](../../adapters/robot-adapter-contract.md)
- [Adapter Layer](../../architecture/adapter-layer.md)
- [Simulated Robot Harness](../../architecture/simulated-robot-harness.md)
