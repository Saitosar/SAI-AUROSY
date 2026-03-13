# Robotics Terms

## Glossary

| Term | Definition |
|------|-------------|
| **Fleet** | Collection of robots managed by the platform. Each robot is registered in Fleet Registry with vendor, model, capabilities, and tenant. |
| **Adapter** | Software bridge between vendor-specific robot runtime (ROS, AimRT, SDK) and the platform's Telemetry Bus (NATS). Publishes telemetry, subscribes to commands. |
| **Capability** | Declared robot ability (walk, stand, safe_stop, navigation, speech, etc.). Tasks are assigned only to robots with required capabilities. |
| **Telemetry** | Normalized robot state (online, actuator_status, current_task, position, etc.) published to `telemetry.robots.{robot_id}`. |
| **Command** | Instruction sent to robot via `commands.robots.{robot_id}` (e.g. safe_stop, cmd_vel, navigate_to). |

## Common Terms

### Fleet

A fleet is the set of robots managed by SAI AUROSY. Robots are registered in Fleet Registry with a unique ID, vendor, model, capabilities, and tenant. The platform routes commands and aggregates telemetry per robot.

### Orchestration

Orchestration coordinates distributed operations across the fleet: workflow runs, scenario execution, task assignment. The Task Runner delegates tasks to robots based on capabilities and availability.

### Adapter

An adapter bridges a vendor-specific robot runtime (ROS1/ROS2, AimRT, SDK) to the platform's unified NATS bus. It publishes telemetry to `telemetry.robots.{robot_id}` and subscribes to commands on `commands.robots.{robot_id}`. See [Adapter Layer](../architecture/adapter-layer.md) and [RobotAdapter Contract](../adapters/robot-adapter-contract.md).

## Related Documents

- [Platform Overview](../product/platform-overview.md)
- [Adapter Layer](../architecture/adapter-layer.md)
