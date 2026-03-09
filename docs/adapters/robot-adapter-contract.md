# RobotAdapter Contract

This document defines the contract that all SAI-AUROSY robot adapters must implement. Adapters bridge vendor-specific robot runtimes (ROS1/ROS2, AimRT, SDK) to the platform's unified Telemetry Bus (NATS).

## Overview

An adapter:

1. **Publishes telemetry** to `telemetry.robots.{robot_id}` whenever the robot state changes
2. **Subscribes to commands** on `commands.robots.{robot_id}` and executes them on the robot
3. Uses the normalized JSON formats defined below

## RobotAdapter Interface (Go Reference)

The platform defines the interface in `pkg/hal/adapter.go`:

| Method | Description |
|--------|-------------|
| `Connect(ctx)` | Establishes connection to the robot runtime (AimRT, ROS2, SDK, etc.) |
| `SubscribeTelemetry(callback)` | Registers callback for telemetry; adapter invokes it on state updates |
| `SendCommand(ctx, cmd)` | Sends command to robot |
| `Disconnect()` | Closes connection |

Python adapters implement the same semantics via NATS pub/sub; they do not use the Go interface directly.

## NATS Contract

### Topics

| Topic | Direction | Description |
|-------|-----------|-------------|
| `telemetry.robots.{robot_id}` | Adapter → Platform | Telemetry updates. Adapter publishes; Control Plane subscribes. |
| `commands.robots.{robot_id}` | Platform → Adapter | Commands. Control Plane publishes; adapter subscribes. |
| `audio.robots.{robot_id}.input` | Adapter → Platform | Raw audio from robot microphone (Speech Layer). |
| `audio.robots.{robot_id}.output` | Platform → Adapter | TTS audio for robot speaker (Speech Layer). |

### Wildcard Subscriptions

- `telemetry.robots.>` — all robots' telemetry
- `commands.robots.>` — all robots' commands (used by Command Arbiter)

## Telemetry JSON Format

Published to `telemetry.robots.{robot_id}`.

```json
{
  "robot_id": "x1-001",
  "timestamp": "2025-03-09T12:00:00.000Z",
  "online": true,
  "actuator_status": "enabled",
  "mock_mode": false,
  "imu": {
    "orientation": { "x": 0, "y": 0, "z": 0, "w": 1 },
    "angular_velocity": { "x": 0, "y": 0, "z": 0 }
  },
  "joint_states": [
    { "name": "joint_1", "position": 0.0, "velocity": 0.0, "effort": 0.0 }
  ],
  "current_task": "idle"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `robot_id` | string | Yes | Must match the robot ID in Fleet Registry |
| `timestamp` | string (ISO8601) | Yes | UTC timestamp of the reading |
| `online` | boolean | Yes | Robot is connected and responsive |
| `actuator_status` | string | Yes | `enabled`, `disabled`, `error`, `calibration` |
| `mock_mode` | boolean | No | True if adapter runs without real robot |
| `imu` | object | No | IMU data (orientation, angular_velocity) |
| `joint_states` | array | No | Per-joint state |
| `current_task` | string | Yes | Heuristic: `idle`, `zero`, `stand`, `walk` |

### JointStateData

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Joint name |
| `position` | number | Position (rad) |
| `velocity` | number | Velocity (rad/s) |
| `effort` | number | Effort/torque (Nm) |

### IMUData

| Field | Type | Description |
|-------|------|-------------|
| `orientation` | object | Keys: x, y, z, w (quaternion) |
| `angular_velocity` | object | Keys: x, y, z (rad/s) |

## Command JSON Format

Received from `commands.robots.{robot_id}`.

```json
{
  "robot_id": "x1-001",
  "command": "cmd_vel",
  "payload": "{\"linear_x\":0.5,\"linear_y\":0,\"angular_z\":0.1}",
  "timestamp": "2025-03-09T12:00:00.000Z",
  "operator_id": "console"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `robot_id` | string | Yes | Target robot |
| `command` | string | Yes | Command name |
| `payload` | string (JSON) | No | Command-specific payload |
| `timestamp` | string (ISO8601) | Yes | When command was issued |
| `operator_id` | string | No | Who issued the command |

## Commands

| Command | Description | Payload |
|---------|-------------|---------|
| `safe_stop` | Emergency stop; robot transitions to idle (no torque) | None |
| `release_control` | Release platform control; operator takes over via joystick | None |
| `zero_mode` | Robot joints to zero position | None |
| `stand_mode` | Standing pose | None |
| `walk_mode` | Walking mode | None |
| `cmd_vel` | Velocity command | `{"linear_x": float, "linear_y": float, "angular_z": float}` |

### cmd_vel Payload

- `linear_x`, `linear_y`: m/s, typically -1.5 to 1.5
- `angular_z`: rad/s, typically -2.0 to 2.0

## Capabilities

Robots declare capabilities in the Fleet Registry. Tasks are assigned only to robots with the required capabilities. Adapters should support the commands corresponding to their robot's capabilities.

| Capability | Related Command |
|------------|-----------------|
| `walk` | walk_mode, cmd_vel |
| `stand` | stand_mode |
| `safe_stop` | safe_stop |
| `release_control` | release_control |
| `cmd_vel` | cmd_vel |
| `zero_mode` | zero_mode |
| `patrol` | Scenario-level |
| `navigation` | Scenario-level |
| `speech` | Microphone + speaker; audio.robots.{id}.input, audio.robots.{id}.output |

## Requirements

1. **robot_id consistency**: Adapter must use the same `robot_id` for both telemetry and command subscription. The robot must exist in Fleet Registry.
2. **Telemetry frequency**: Publish at least every few seconds when online; more frequently when state changes.
3. **Command handling**: Ignore unknown commands; log and optionally report errors for known-but-failed commands.
4. **Safety**: `safe_stop` must be handled with highest priority and lowest latency.

## Related Documents

- [Adapter Layer](../architecture/adapter-layer.md)
- [Speech Layer](../architecture/speech-layer.md)
- [AGIBOT](../vendors/agibot.md)
- [Unitree](../vendors/unitree.md)
- [ROS](../vendors/ros.md)
