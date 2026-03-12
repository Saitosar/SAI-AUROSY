# Adapter Development Guide

Step-by-step guide for creating a SAI-AUROSY robot adapter using the template.

## 1. Overview

Adapters bridge vendor-specific robot runtimes (ROS1/ROS2, AimRT, SDK) to the platform's unified Telemetry Bus (NATS). Each adapter:

1. **Publishes telemetry** to `telemetry.robots.{robot_id}` when robot state changes
2. **Subscribes to commands** on `commands.robots.{robot_id}` and executes them on the robot

See [Adapter Layer](../architecture/adapter-layer.md) for the architecture overview.

## 2. Prerequisites

- Python 3.8+
- `nats-py`: `pip install nats-py`
- NATS running (e.g. `docker run -p 4222:4222 nats:2-alpine`)
- Control Plane running (for robot registration and command routing)
- Robot SDK or ROS (for real robot integration)

## 3. Copy the Template

```bash
cp -r pkg/adapters/template pkg/adapters/myrobot
cd pkg/adapters/myrobot
```

Replace `myrobot` with your vendor name (e.g. `agibot`, `unitree`).

## 4. Configure Environment

Set these variables before running the adapter:

| Variable | Description | Example |
|----------|-------------|---------|
| `ROBOT_ID` | Robot ID (must match Fleet Registry) | `myrobot-001` |
| `NATS_URL` | NATS connection URL | `nats://localhost:4222` |
| `TELEMETRY_INTERVAL` | Seconds between telemetry publishes (mock mode) | `1.0` |

Add vendor-specific variables as needed (e.g. `AGIBOT_MOCK=1`, `ROS_DISTRO=humble`).

## 5. Register the Robot

Before the adapter can receive commands, register the robot in Fleet Registry:

```bash
curl -X POST http://localhost:8080/v1/robots \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <your-api-key>" \
  -d '{
    "id": "myrobot-001",
    "vendor": "myrobot",
    "model": "ModelX",
    "adapter_endpoint": "nats://localhost:4222",
    "tenant_id": "default",
    "capabilities": ["walk", "stand", "safe_stop", "release_control", "cmd_vel", "zero_mode"]
  }'
```

Use the correct `robot_id` prefix for your vendor. See [Adapter Layer — robot_id Prefix](../architecture/adapter-layer.md#robot_id-prefix-convention).

## 6. Implement Commands

Replace the placeholder logic in `handle_command()` with your robot SDK or ROS calls.

| Command | Description | Replace with |
|---------|-------------|--------------|
| `safe_stop` | Emergency stop; robot to idle | Robot emergency stop API |
| `release_control` | Release to operator | Release control API |
| `zero_mode` | Joints to zero position | Zero-mode command |
| `stand_mode` | Standing pose | Stand command |
| `walk_mode` | Walking mode | Walk command |
| `cmd_vel` | Velocity command | ROS `/cmd_vel` or SDK velocity API |
| `navigate_to` | Navigation to target (Mall Assistant) | Payload: `target_coordinates`, `store_name`, `destination_node_id`, `route`, `estimated_distance` |
| `speak` | TTS output (Mall Assistant) | Payload: `{"text": "..."}`; audio via `audio.robots.{id}.output` |

### Example: `handle_command` (template)

```python
async def handle_command(self, msg):
    data = json.loads(msg.data.decode())
    cmd = data.get("command", "")
    payload = data.get("payload")
    if isinstance(payload, str) and payload:
        try:
            payload = json.loads(payload)
        except json.JSONDecodeError:
            payload = {}

    if cmd == "safe_stop":
        self.last_mode_command = "idle"
        # REPLACE: call your robot's emergency stop
        # e.g. self.robot_sdk.emergency_stop()

    elif cmd == "cmd_vel":
        self.last_mode_command = "walk"
        payload = payload or {}
        linear_x = float(payload.get("linear_x", 0))
        linear_y = float(payload.get("linear_y", 0))
        angular_z = float(payload.get("angular_z", 0))
        # REPLACE: send velocity to robot
        # e.g. self.cmd_vel_pub.publish(Twist(linear=..., angular=...))
```

### cmd_vel Payload

- `linear_x`, `linear_y`: m/s (typically -1.5 to 1.5)
- `angular_z`: rad/s (typically -2.0 to 2.0)

## 7. Implement Telemetry

Replace `publish_telemetry()` to use real sensor data when available.

### Option A: Mock Mode (template default)

The template publishes mock telemetry on an interval. Use for development and testing.

### Option B: Real Robot Data

1. Subscribe to your robot's state (ROS topics, SDK callbacks)
2. Normalize to the [Telemetry JSON format](robot-adapter-contract.md#telemetry-json-format)
3. Publish to `telemetry.robots.{robot_id}`

### Telemetry Format (required fields)

```json
{
  "robot_id": "myrobot-001",
  "timestamp": "2025-03-09T12:00:00.000Z",
  "online": true,
  "actuator_status": "enabled",
  "current_task": "idle",
  "mock_mode": false
}
```

Optional: `imu`, `joint_states`. See [RobotAdapter Contract — Telemetry](robot-adapter-contract.md#telemetry-json-format).

### Example: `make_telemetry` (from template)

```python
def make_telemetry(robot_id: str, online: bool = True, actuator_status: str = "enabled",
                   current_task: str = "idle", mock_mode: bool = True) -> dict:
    """Build normalized Telemetry dict per robot-adapter-contract."""
    return {
        "robot_id": robot_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "online": online,
        "actuator_status": actuator_status,
        "current_task": current_task,
        "mock_mode": mock_mode,
    }
```

Add `imu` and `joint_states` when your robot provides them.

## 8. Test

1. Start NATS and Control Plane:
   ```bash
   docker compose up -d nats control-plane
   ```

2. Register the robot (Step 5).

3. Run the adapter:
   ```bash
   export ROBOT_ID=myrobot-001
   export NATS_URL=nats://localhost:4222
   python adapter.py
   ```

4. Open Operator Console at http://localhost:3000 — robot should appear with telemetry.

5. Send a command via API:
   ```bash
   curl -X POST http://localhost:8080/v1/robots/myrobot-001/command \
     -H "Content-Type: application/json" \
     -H "X-API-Key: <key>" \
     -d '{"command": "safe_stop"}'
   ```
   (Control Plane base path is `/v1`; when using Operator Console proxy, use `/api/v1`.)

6. Verify the adapter logs the command and the robot responds (or mock logs it).

## 9. Capabilities

Declare capabilities in the Fleet Registry so tasks are assigned only to robots that support them.

| Capability | Related Commands |
|------------|------------------|
| `walk` | walk_mode, cmd_vel |
| `stand` | stand_mode |
| `safe_stop` | safe_stop |
| `release_control` | release_control |
| `cmd_vel` | cmd_vel |
| `zero_mode` | zero_mode |
| `patrol` | Scenario-level |
| `navigation` | navigate_to (requires position, target_position, distance_to_target in telemetry) |
| `speech` | speak; audio.robots.{id}.input, audio.robots.{id}.output |

See [RobotAdapter Contract — Capabilities](robot-adapter-contract.md#capabilities).

## 10. References

- [RobotAdapter Contract](robot-adapter-contract.md) — NATS topics, JSON formats, commands
- [Template adapter](../../pkg/adapters/template/README.md) — minimal example
- [Adapter Layer](../architecture/adapter-layer.md) — robot_id prefix, interface
- [AGIBOT](../vendors/agibot.md) — AGIBOT integration
- [Unitree](../vendors/unitree.md) — Unitree integration
- [ROS](../vendors/ros.md) — ROS adapter
