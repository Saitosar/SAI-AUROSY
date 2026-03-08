# Adapter Layer

## Overview

The adapter layer abstracts vendor-specific APIs (AimRT, ROS2, SDK) and exposes a unified interface for the SAI-AUROSY Control Plane. Each robot vendor has an adapter that implements the `RobotAdapter` interface.

## Supported Vendors

- [Agibot](../vendors/agibot.md)
- [Unitree](../vendors/unitree.md)

## Adapter Interface

All adapters implement the `RobotAdapter` interface:

| Method | Description |
|--------|-------------|
| `Connect(ctx)` | Establishes connection to the robot runtime (AimRT/ROS2) |
| `SubscribeTelemetry(callback)` | Registers callback for telemetry; adapter invokes it on state updates |
| `SendCommand(ctx, cmd)` | Sends command to robot. Commands: `safe_stop`, `release_control` |
| `Disconnect()` | Closes connection |

### Commands

| Command | Description |
|---------|-------------|
| `safe_stop` | Emergency stop; robot transitions to idle (no torque) |
| `release_control` | Release platform control; operator takes over via joystick |

### Telemetry Callback

The callback receives normalized `Telemetry` with: `robot_id`, `timestamp`, `online`, `actuator_status`, `imu`, `current_task`.

## Adding New Vendors

1. Implement `RobotAdapter` in `pkg/adapters/<vendor>/`
2. Map vendor protocols to normalized `Telemetry` and `Command`
3. Register adapter in the Command Arbiter for the vendor's `robot_id` prefix
4. Document protocols in `docs/vendors/<vendor>.md`

## Related Documents

- [Platform Architecture](platform-architecture.md)
- [Multi-Robot Architecture](multi-robot-architecture.md)
- [Agibot](../vendors/agibot.md)
- [Unitree](../vendors/unitree.md)
