# ROS

## Overview

The ROS adapter is a generic adapter for robots running ROS1 or ROS2. It bridges standard ROS topics to the SAI-AUROSY Telemetry Bus (NATS), enabling any ROS-based robot to integrate with the platform.

Supports:

- **ROS1:** noetic, melodic, kinetic
- **ROS2:** humble, iron, jazzy, rolling

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ROBOT_ID` | `ros-001` | Robot ID (use `ros-` prefix per convention) |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `ROS_DISTRO` | `humble` | ROS distribution: `noetic` (ROS1) or `humble`/`iron` (ROS2) |
| `ROS_MOCK` | `0` | Set to `1` for mock mode (no ROS, NATS only) |

### Topic Mapping

All topics are configurable via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TELEMETRY_JOINT_STATES` | `/joint_states` | ROS topic for joint states |
| `TELEMETRY_IMU` | `/imu/data` | ROS topic for IMU data |
| `CMD_VEL` | `/cmd_vel` | ROS topic for velocity commands (geometry_msgs/Twist) |
| `SAFE_STOP_TOPIC` | `/emergency_stop` | ROS topic for emergency stop (std_msgs/Empty) |
| `STAND_TOPIC` | `/stand` | ROS topic for stand command |
| `WALK_TOPIC` | `/walk` | ROS topic for walk mode |
| `ZERO_TOPIC` | `/zero` | ROS topic for zero joints |

## Command Mapping

| SAI-AUROSY Command | ROS Topic | Message Type |
|--------------------|-----------|--------------|
| `safe_stop` | `SAFE_STOP_TOPIC` | std_msgs/Empty |
| `cmd_vel` | `CMD_VEL` | geometry_msgs/Twist |
| `stand_mode` | `STAND_TOPIC` | std_msgs/Empty |
| `walk_mode` | `WALK_TOPIC` | std_msgs/Empty |
| `zero_mode` | `ZERO_TOPIC` | std_msgs/Empty |
| `release_control` | — | No ROS equivalent; platform stops sending |

## Telemetry Mapping

| SAI-AUROSY Field | ROS Topic | Message Type |
|------------------|-----------|--------------|
| `joint_states` | `TELEMETRY_JOINT_STATES` | sensor_msgs/JointState |
| `imu` | `TELEMETRY_IMU` | sensor_msgs/Imu |

## Usage

### Mock Mode (no ROS)

```bash
pip install nats-py
export ROBOT_ID=ros-001
export NATS_URL=nats://localhost:4222
export ROS_MOCK=1
python pkg/adapters/ros/adapter.py
```

### ROS2 (Humble)

```bash
source /opt/ros/humble/setup.bash
pip install nats-py rclpy
export ROBOT_ID=ros-001
export NATS_URL=nats://localhost:4222
export ROS_DISTRO=humble
python pkg/adapters/ros/adapter.py
```

### ROS1 (Noetic)

```bash
source /opt/ros/noetic/setup.bash
pip install nats-py
export ROBOT_ID=ros-001
export NATS_URL=nats://localhost:4222
export ROS_DISTRO=noetic
python pkg/adapters/ros/adapter.py
```

### Custom Topics

```bash
export TELEMETRY_JOINT_STATES=/my_robot/joint_states
export TELEMETRY_IMU=/my_robot/imu
export CMD_VEL=/my_robot/cmd_vel
export SAFE_STOP_TOPIC=/my_robot/emergency_stop
python pkg/adapters/ros/adapter.py
```

## Fleet Registry

Register the robot with `ros-` prefix:

```json
{
  "id": "ros-001",
  "vendor": "ros",
  "model": "Generic",
  "adapter_endpoint": "nats://localhost:4222",
  "capabilities": ["walk", "stand", "safe_stop", "release_control", "cmd_vel", "zero_mode"]
}
```

## Limitations

- **Topic availability:** Commands are published only if the ROS topic exists. If your robot uses different topic names (e.g. `/cmd_vel` vs `/mobile_base_controller/cmd_vel`), configure via env.
- **Message types:** The adapter expects standard ROS message types (JointState, Imu, Twist, Empty). Custom message types require adapter modification.
- **Single robot:** One adapter process serves one `robot_id`. For multiple ROS robots, run multiple adapter instances with different `ROBOT_ID`.

## Related Documents

- [Adapter Layer](../architecture/adapter-layer.md)
- [RobotAdapter Contract](../adapters/robot-adapter-contract.md)
