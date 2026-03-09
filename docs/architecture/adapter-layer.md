# Adapter Layer

## Overview

The adapter layer abstracts vendor-specific APIs (AimRT, ROS2, SDK) and exposes a unified interface for the SAI-AUROSY Control Plane. Each robot vendor has an adapter that implements the `RobotAdapter` interface.

## Supported Vendors

- [Agibot](../vendors/agibot.md)
- [Unitree](../vendors/unitree.md)
- [ROS](../vendors/ros.md) (ROS1/ROS2 generic adapter)

## robot_id Prefix Convention

Adapters subscribe to `commands.robots.{robot_id}` and publish to `telemetry.robots.{robot_id}`. Use the following prefix convention when registering robots in Fleet Registry:

| Vendor | robot_id Prefix | Adapter |
|--------|-----------------|---------|
| AGIBOT | `x1-` | `pkg/adapters/agibot/` |
| Unitree | `go2-` | `pkg/adapters/unitree/` |
| ROS (generic) | `ros-` | `pkg/adapters/ros/` |
| Custom | `{vendor}-` | Implement per [RobotAdapter Contract](../adapters/robot-adapter-contract.md) |

Examples: `x1-001`, `go2-001`, `ros-001`.

## Adapter Interface

All adapters implement the `RobotAdapter` interface:

| Method | Description |
|--------|-------------|
| `Connect(ctx)` | Establishes connection to the robot runtime (AimRT/ROS2) |
| `SubscribeTelemetry(callback)` | Registers callback for telemetry; adapter invokes it on state updates |
| `SendCommand(ctx, cmd)` | Sends command to robot. Commands: `safe_stop`, `release_control`, `zero_mode`, `stand_mode`, `walk_mode`, `cmd_vel` |
| `Disconnect()` | Closes connection |

### Commands

| Command | Description |
|---------|-------------|
| `safe_stop` | Emergency stop; robot transitions to idle (no torque) |
| `release_control` | Release platform control; operator takes over via joystick |
| `zero_mode` | Robot joints to zero position |
| `stand_mode` | Standing pose |
| `walk_mode` | Walking mode |
| `cmd_vel` | Velocity command. Payload: `{ linear_x, linear_y, angular_z }` (m/s, rad/s) |

### Telemetry Callback

The callback receives normalized `Telemetry` with: `robot_id`, `timestamp`, `online`, `actuator_status`, `imu`, `joint_states`, `current_task`.

- `joint_states` — array of `{ name, position, velocity, effort }` per joint (from `/joint_states`).
- `current_task` — heuristic from last mode command: `idle` (safe_stop), `zero` (zero_mode), `stand` (stand_mode), `walk` (walk_mode, cmd_vel).

## Capability Model

Each robot in the Fleet Registry has a `capabilities` array describing what it can do. Standard capabilities (see `pkg/hal/capabilities.go`):

| Capability | Description |
|------------|-------------|
| `walk` | Walking mode |
| `stand` | Standing pose |
| `safe_stop` | Emergency stop |
| `release_control` | Release to operator |
| `cmd_vel` | Velocity commands |
| `zero_mode` | Zero joints |
| `patrol` | Patrol scenario |
| `navigation` | Navigation scenario |

Scenarios declare `RequiredCapabilities`. Tasks are only assigned to robots that have all required capabilities. The Task Runner and API validate this before execution.

## Adding New Vendors

1. Implement `RobotAdapter` in `pkg/adapters/<vendor>/` (see [template](../../pkg/adapters/template/README.md))
2. Map vendor protocols to normalized `Telemetry` and `Command` per [RobotAdapter Contract](../adapters/robot-adapter-contract.md)
3. Use a consistent `robot_id` prefix for your vendor (e.g. `myrobot-001`)
4. Document protocols in `docs/vendors/<vendor>.md`

## Related Documents

- [RobotAdapter Contract](../adapters/robot-adapter-contract.md)
- [Platform Architecture](platform-architecture.md)
- [Multi-Robot Architecture](multi-robot-architecture.md)
- [Agibot](../vendors/agibot.md)
- [Unitree](../vendors/unitree.md)
- [ROS](../vendors/ros.md)
