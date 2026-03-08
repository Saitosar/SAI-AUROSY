#!/usr/bin/env python3
"""
AGIBOT X1 Adapter - bridges ROS2 (X1 Infer) and SAI-AUROSY Telemetry Bus (NATS).

ROS2: subscribes to /joint_states, /imu/data; publishes to /start_control, /zero_mode, /stand_mode, /walk_mode, /cmd_vel
NATS: publishes telemetry.robots.{robot_id}; subscribes to commands.robots.{robot_id}

Usage:
  AGIBOT_MOCK=1 python adapter.py   # Mock mode (no ROS2)
  python adapter.py                 # ROS2 mode (requires X1 Infer running)
"""

import asyncio
import json
import os
import threading
import time
from datetime import datetime, timezone

try:
    import rclpy
    from rclpy.node import Node
    from geometry_msgs.msg import Twist
    from sensor_msgs.msg import JointState, Imu
    from std_msgs.msg import Empty
    HAS_ROS2 = True
except ImportError:
    HAS_ROS2 = False

try:
    import nats
    HAS_NATS = True
except ImportError:
    HAS_NATS = False


ROBOT_ID = os.environ.get("ROBOT_ID", "x1-001")
NATS_URL = os.environ.get("NATS_URL", "nats://localhost:4222")
MOCK_MODE = os.environ.get("AGIBOT_MOCK", "0") == "1"


def make_telemetry(robot_id: str, online: bool, actuator_status: str = "enabled",
                   imu: dict = None, joint_states: list = None, current_task: str = "idle") -> dict:
    t = {
        "robot_id": robot_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "online": online,
        "actuator_status": actuator_status,
        "imu": imu,
        "current_task": current_task,
    }
    if joint_states:
        t["joint_states"] = joint_states
    return t


class AgibotAdapter:
    def __init__(self):
        self.robot_id = ROBOT_ID
        self.nats_url = NATS_URL
        self.nats_nc = None
        self.last_joint_time = 0.0
        self.last_imu = None
        self.safe_stop_pub = None
        self.zero_mode_pub = None
        self.stand_mode_pub = None
        self.walk_mode_pub = None
        self.cmd_vel_pub = None
        self.last_joint_states = []
        self.last_mode_command = "idle"
        self._lock = threading.Lock()

    async def connect_nats(self):
        if not HAS_NATS:
            raise RuntimeError("nats-py not installed. pip install nats-py")
        self.nats_nc = await nats.connect(self.nats_url)
        print(f"[agibot] Connected to NATS at {self.nats_url}")

    async def publish_telemetry(self, online: bool = True, actuator_status: str = "enabled"):
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
        )
        await self.nats_nc.publish(f"telemetry.robots.{self.robot_id}", json.dumps(t).encode())

    def on_joint_states(self, msg):
        states = []
        names = list(msg.name) if msg.name else []
        positions = list(msg.position) if msg.position else []
        velocities = list(msg.velocity) if msg.velocity else []
        efforts = list(msg.effort) if msg.effort else []
        n = len(names)
        for i in range(n):
            states.append({
                "name": names[i] if i < len(names) else "",
                "position": float(positions[i]) if i < len(positions) else 0.0,
                "velocity": float(velocities[i]) if i < len(velocities) else 0.0,
                "effort": float(efforts[i]) if i < len(efforts) else 0.0,
            })
        with self._lock:
            self.last_joint_time = time.time()
            self.last_joint_states = states

    def on_imu(self, msg):
        with self._lock:
            self.last_imu = {
                "orientation": {
                    "x": msg.orientation.x, "y": msg.orientation.y,
                    "z": msg.orientation.z, "w": msg.orientation.w,
                },
                "angular_velocity": {
                    "x": msg.angular_velocity.x,
                    "y": msg.angular_velocity.y,
                    "z": msg.angular_velocity.z,
                },
            }

    def publish_safe_stop(self):
        if self.safe_stop_pub:
            self.safe_stop_pub.publish(Empty())
            print(f"[{self.robot_id}] safe_stop -> /start_control")
        else:
            print(f"[{self.robot_id}] safe_stop (mock)")

    def publish_zero_mode(self):
        if self.zero_mode_pub:
            self.zero_mode_pub.publish(Empty())
            print(f"[{self.robot_id}] zero_mode -> /zero_mode")
        else:
            print(f"[{self.robot_id}] zero_mode (mock)")

    def publish_stand_mode(self):
        if self.stand_mode_pub:
            self.stand_mode_pub.publish(Empty())
            print(f"[{self.robot_id}] stand_mode -> /stand_mode")
        else:
            print(f"[{self.robot_id}] stand_mode (mock)")

    def publish_walk_mode(self):
        if self.walk_mode_pub:
            self.walk_mode_pub.publish(Empty())
            print(f"[{self.robot_id}] walk_mode -> /walk_mode")
        else:
            print(f"[{self.robot_id}] walk_mode (mock)")

    def publish_cmd_vel(self, linear_x: float, linear_y: float, angular_z: float):
        if self.cmd_vel_pub and HAS_ROS2:
            twist = Twist()
            twist.linear.x = float(linear_x)
            twist.linear.y = float(linear_y)
            twist.linear.z = 0.0
            twist.angular.x = 0.0
            twist.angular.y = 0.0
            twist.angular.z = float(angular_z)
            self.cmd_vel_pub.publish(twist)
            print(f"[{self.robot_id}] cmd_vel -> /cmd_vel linear_x={linear_x} linear_y={linear_y} angular_z={angular_z}")
        else:
            print(f"[{self.robot_id}] cmd_vel (mock) linear_x={linear_x} linear_y={linear_y} angular_z={angular_z}")

    def _set_last_mode(self, mode: str):
        with self._lock:
            self.last_mode_command = mode

    async def handle_command(self, msg):
        data = json.loads(msg.data.decode())
        cmd = data.get("command")
        if cmd == "safe_stop":
            self._set_last_mode("idle")
            self.publish_safe_stop()
        elif cmd == "release_control":
            print(f"[{self.robot_id}] release_control (platform stops sending; operator uses joystick)")
        elif cmd == "zero_mode":
            self._set_last_mode("zero")
            self.publish_zero_mode()
        elif cmd == "stand_mode":
            self._set_last_mode("stand")
            self.publish_stand_mode()
        elif cmd == "walk_mode":
            self._set_last_mode("walk")
            self.publish_walk_mode()
        elif cmd == "cmd_vel":
            with self._lock:
                if self.last_mode_command == "idle":
                    self.last_mode_command = "walk"
            payload = data.get("payload") or {}
            linear_x = float(payload.get("linear_x", 0))
            linear_y = float(payload.get("linear_y", 0))
            angular_z = float(payload.get("angular_z", 0))
            self.publish_cmd_vel(linear_x, linear_y, angular_z)

    async def run_mock(self):
        await self.connect_nats()

        async def cmd_cb(msg):
            await self.handle_command(msg)

        await self.nats_nc.subscribe(
            f"commands.robots.{self.robot_id}",
            cb=cmd_cb,
        )
        print(f"[{self.robot_id}] Mock mode: publishing telemetry every 1s")
        while True:
            await self.publish_telemetry(online=True, actuator_status="enabled")
            await asyncio.sleep(1)


if HAS_ROS2:
    class AgibotROS2Node(Node):
        def __init__(self, adapter: AgibotAdapter):
            super().__init__("agibot_adapter")
            self.adapter = adapter
            self.create_subscription(JointState, "/joint_states", adapter.on_joint_states, 10)
            self.create_subscription(Imu, "/imu/data", adapter.on_imu, 10)
            adapter.safe_stop_pub = self.create_publisher(Empty, "/start_control", 10)
            adapter.zero_mode_pub = self.create_publisher(Empty, "/zero_mode", 10)
            adapter.stand_mode_pub = self.create_publisher(Empty, "/stand_mode", 10)
            adapter.walk_mode_pub = self.create_publisher(Empty, "/walk_mode", 10)
            adapter.cmd_vel_pub = self.create_publisher(Twist, "/cmd_vel", 10)
            self.get_logger().info(f"AGIBOT adapter ROS2, robot_id={adapter.robot_id}")


async def run_nats_loop(adapter: AgibotAdapter):
    await adapter.connect_nats()
    async def cmd_cb(msg):
        await adapter.handle_command(msg)

    await adapter.nats_nc.subscribe(
        f"commands.robots.{adapter.robot_id}",
        cb=cmd_cb,
    )
    while True:
        with adapter._lock:
            online = (time.time() - adapter.last_joint_time) < 2.0
        await adapter.publish_telemetry(online=online, actuator_status="enabled" if online else "unknown")
        await asyncio.sleep(0.5)


def run_ros2_spin(adapter: AgibotAdapter):
    rclpy.init()
    node = AgibotROS2Node(adapter)
    rclpy.spin(node)


async def main():
    adapter = AgibotAdapter()
    if MOCK_MODE or not HAS_ROS2:
        await adapter.run_mock()
    else:
        t = threading.Thread(target=run_ros2_spin, args=(adapter,), daemon=True)
        t.start()
        await asyncio.sleep(2)
        await run_nats_loop(adapter)


if __name__ == "__main__":
    asyncio.run(main())
