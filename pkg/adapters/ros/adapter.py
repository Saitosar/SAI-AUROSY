#!/usr/bin/env python3
"""
SAI-AUROSY ROS Adapter - generic adapter for ROS1/ROS2 robots.

Bridges ROS topics to the SAI-AUROSY Telemetry Bus (NATS).
Supports both ROS1 (noetic, melodic) and ROS2 (humble, iron, etc.) via ROS_DISTRO.

Usage:
  ROS_MOCK=1 python adapter.py                    # Mock mode (no ROS)
  ROS_DISTRO=noetic python adapter.py              # ROS1
  ROS_DISTRO=humble python adapter.py              # ROS2 (default)

Environment:
  ROBOT_ID, NATS_URL, ROS_DISTRO, ROS_MOCK
  TELEMETRY_JOINT_STATES, TELEMETRY_IMU, CMD_VEL, SAFE_STOP_TOPIC, STAND_TOPIC, WALK_TOPIC, ZERO_TOPIC
"""

import asyncio
import json
import os
import threading
import time
from datetime import datetime, timezone

try:
    import nats
    HAS_NATS = True
except ImportError:
    HAS_NATS = False

try:
    from .config import get_config
except ImportError:
    from config import get_config


def make_telemetry(robot_id: str, online: bool, actuator_status: str = "enabled",
                   imu: dict = None, joint_states: list = None, current_task: str = "idle",
                   mock_mode: bool = False) -> dict:
    """Build normalized Telemetry dict per robot-adapter-contract."""
    t = {
        "robot_id": robot_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "online": online,
        "actuator_status": actuator_status,
        "current_task": current_task,
        "mock_mode": mock_mode,
    }
    if imu:
        t["imu"] = imu
    if joint_states:
        t["joint_states"] = joint_states
    return t


def joint_states_from_ros(msg) -> list:
    """Extract joint_states from ROS JointState message."""
    states = []
    names = list(msg.name) if hasattr(msg, "name") and msg.name else []
    positions = list(msg.position) if hasattr(msg, "position") and msg.position else []
    velocities = list(msg.velocity) if hasattr(msg, "velocity") and msg.velocity else []
    efforts = list(msg.effort) if hasattr(msg, "effort") and msg.effort else []
    n = len(names)
    for i in range(n):
        states.append({
            "name": names[i] if i < len(names) else "",
            "position": float(positions[i]) if i < len(positions) else 0.0,
            "velocity": float(velocities[i]) if i < len(velocities) else 0.0,
            "effort": float(efforts[i]) if i < len(efforts) else 0.0,
        })
    return states


def imu_from_ros(msg) -> dict:
    """Extract IMU dict from ROS Imu message."""
    return {
        "orientation": {
            "x": getattr(msg.orientation, "x", 0),
            "y": getattr(msg.orientation, "y", 0),
            "z": getattr(msg.orientation, "z", 0),
            "w": getattr(msg.orientation, "w", 1),
        },
        "angular_velocity": {
            "x": getattr(msg.angular_velocity, "x", 0),
            "y": getattr(msg.angular_velocity, "y", 0),
            "z": getattr(msg.angular_velocity, "z", 0),
        },
    }


class ROSAdapter:
    """Generic ROS1/ROS2 adapter for SAI-AUROSY."""

    def __init__(self, config: dict = None):
        self.config = config or get_config()
        self.robot_id = self.config["robot_id"]
        self.nats_url = self.config["nats_url"]
        self.nats_nc = None
        self.ros_bridge = None
        self._lock = threading.Lock()
        self.last_joint_time = 0.0
        self.last_imu = None
        self.last_joint_states = []
        self.last_mode_command = "idle"

    async def connect_nats(self):
        if not HAS_NATS:
            raise RuntimeError("nats-py not installed. pip install nats-py")
        self.nats_nc = await nats.connect(self.nats_url)
        print(f"[ros] Connected to NATS at {self.nats_url}")

    def on_joint_states(self, msg):
        with self._lock:
            self.last_joint_time = time.time()
            self.last_joint_states = joint_states_from_ros(msg)

    def on_imu(self, msg):
        with self._lock:
            self.last_imu = imu_from_ros(msg)

    def _set_last_mode(self, mode: str):
        with self._lock:
            self.last_mode_command = mode

    async def publish_telemetry(self, online: bool = True, actuator_status: str = "enabled",
                                mock_mode: bool = False):
        if not self.nats_nc:
            return
        with self._lock:
            imu = self.last_imu
            joint_states = list(self.last_joint_states)
            current_task = self.last_mode_command
        t = make_telemetry(
            self.robot_id,
            online=online,
            actuator_status=actuator_status,
            imu=imu,
            joint_states=joint_states if joint_states else None,
            current_task=current_task,
            mock_mode=mock_mode,
        )
        await self.nats_nc.publish(f"telemetry.robots.{self.robot_id}", json.dumps(t).encode())

    async def handle_command(self, msg):
        data = json.loads(msg.data.decode())
        cmd = data.get("command", "")
        payload = data.get("payload")
        if isinstance(payload, str) and payload:
            try:
                payload = json.loads(payload)
            except json.JSONDecodeError:
                payload = {}
        payload = payload or {}

        if cmd == "safe_stop":
            self._set_last_mode("idle")
            if self.ros_bridge:
                self.ros_bridge.publish_safe_stop()
                print(f"[{self.robot_id}] safe_stop -> {self.config['topics']['safe_stop']}")
            else:
                print(f"[{self.robot_id}] safe_stop (mock)")

        elif cmd == "release_control":
            print(f"[{self.robot_id}] release_control (platform stops sending)")

        elif cmd == "zero_mode":
            self._set_last_mode("zero")
            if self.ros_bridge:
                self.ros_bridge.publish_zero()
                print(f"[{self.robot_id}] zero_mode -> {self.config['topics']['zero']}")
            else:
                print(f"[{self.robot_id}] zero_mode (mock)")

        elif cmd == "stand_mode":
            self._set_last_mode("stand")
            if self.ros_bridge:
                self.ros_bridge.publish_stand()
                print(f"[{self.robot_id}] stand_mode -> {self.config['topics']['stand']}")
            else:
                print(f"[{self.robot_id}] stand_mode (mock)")

        elif cmd == "walk_mode":
            self._set_last_mode("walk")
            if self.ros_bridge:
                self.ros_bridge.publish_walk()
                print(f"[{self.robot_id}] walk_mode -> {self.config['topics']['walk']}")
            else:
                print(f"[{self.robot_id}] walk_mode (mock)")

        elif cmd == "cmd_vel":
            self._set_last_mode("walk")
            linear_x = float(payload.get("linear_x", 0))
            linear_y = float(payload.get("linear_y", 0))
            angular_z = float(payload.get("angular_z", 0))
            if self.ros_bridge:
                self.ros_bridge.publish_cmd_vel(linear_x, linear_y, angular_z)
                print(f"[{self.robot_id}] cmd_vel -> {self.config['topics']['cmd_vel']}")
            else:
                print(f"[{self.robot_id}] cmd_vel (mock) linear_x={linear_x} linear_y={linear_y} angular_z={angular_z}")

        else:
            print(f"[{self.robot_id}] unknown command: {cmd}")

    async def run_mock(self):
        """Mock mode: no ROS, just NATS telemetry and command logging."""
        await self.connect_nats()

        async def cmd_cb(m):
            await self.handle_command(m)

        await self.nats_nc.subscribe(f"commands.robots.{self.robot_id}", cb=cmd_cb)
        print(f"[{self.robot_id}] ROS adapter (mock): publishing telemetry every 1s")
        while True:
            await self.publish_telemetry(online=True, actuator_status="enabled", mock_mode=True)
            await asyncio.sleep(1)

    async def run_with_ros(self):
        """Run with ROS bridge and NATS loop."""
        if self.config["is_ros1"]:
            try:
                from .ros1_bridge import ROS1Bridge
            except ImportError:
                from ros1_bridge import ROS1Bridge
            self.ros_bridge = ROS1Bridge(self.config, self.on_joint_states, self.on_imu)
            ros_thread = threading.Thread(
                target=lambda: __import__("rospy").spin(),
                daemon=True,
            )
        else:
            try:
                from .ros2_bridge import ROS2Bridge
            except ImportError:
                from ros2_bridge import ROS2Bridge
            import rclpy
            rclpy.init()
            self.ros_bridge = ROS2Bridge(self.config, self.on_joint_states, self.on_imu)
            ros_thread = threading.Thread(
                target=lambda: rclpy.spin(self.ros_bridge),
                daemon=True,
            )
        ros_thread.start()

        await asyncio.sleep(2)
        await self.connect_nats()

        async def cmd_cb(m):
            await self.handle_command(m)

        await self.nats_nc.subscribe(f"commands.robots.{self.robot_id}", cb=cmd_cb)
        print(f"[{self.robot_id}] ROS adapter ({self.config['ros_distro']}): telemetry + commands")

        while True:
            with self._lock:
                online = (time.time() - self.last_joint_time) < 2.0
            await self.publish_telemetry(
                online=online,
                actuator_status="enabled" if online else "unknown",
                mock_mode=False,
            )
            await asyncio.sleep(0.5)


async def main():
    config = get_config()
    adapter = ROSAdapter(config)

    if config["mock_mode"]:
        await adapter.run_mock()
    else:
        await adapter.run_with_ros()


if __name__ == "__main__":
    asyncio.run(main())
