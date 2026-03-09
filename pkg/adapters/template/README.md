# Adapter Template

Minimal SAI-AUROSY robot adapter template. Use this as a starting point for implementing adapters for new robot vendors.

## Quick Start

```bash
pip install nats-py
export ROBOT_ID=my-robot-001
export NATS_URL=nats://localhost:4222
python adapter.py
```

With NATS and Control Plane running, the adapter will publish mock telemetry and log received commands.

## Customization

1. **Copy** this directory to `pkg/adapters/<vendor>/` (e.g. `pkg/adapters/myrobot/`).

2. **Replace** the placeholder logic in `handle_command()`:
   - `safe_stop` → call your robot's emergency stop API
   - `cmd_vel` → send velocity to your robot (ROS `/cmd_vel`, SDK, etc.)
   - Other commands as needed

3. **Replace** `publish_telemetry()` if you have real sensor data:
   - Subscribe to your robot's state (ROS topics, SDK callbacks)
   - Normalize to the [Telemetry format](../../../docs/adapters/robot-adapter-contract.md#telemetry-json-format)
   - Publish to `telemetry.robots.{robot_id}`

4. **Add** your robot runtime connection in `connect_nats()` or a separate `connect_robot()`:
   - ROS1/ROS2, vendor SDK, REST API, etc.

## Contract

See [RobotAdapter Contract](../../../docs/adapters/robot-adapter-contract.md) for:

- NATS topics: `telemetry.robots.{robot_id}`, `commands.robots.{robot_id}`
- Telemetry and Command JSON formats
- Supported commands and capabilities

## robot_id Prefix

Register your robot in Fleet Registry with an ID that follows the prefix convention for your vendor (e.g. `myrobot-001`). See [Adapter Layer](../../../docs/architecture/adapter-layer.md).
