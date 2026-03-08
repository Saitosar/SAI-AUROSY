#!/usr/bin/env python3
"""
AGIBOT X1 Adapter - bridges ROS2 (X1 Infer) and SAI-AUROSY Telemetry Bus (NATS).

ROS2: subscribes to /joint_states, /imu/data; publishes to /start_control for safe_stop
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
                   imu: dict = None, current_task: str = "idle") -> dict:
    return {
        "robot_id": robot_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "online": online,
        "actuator_status": actuator_status,
        "imu": imu,
        "current_task": current_task,
    }


class AgibotAdapter:
    def __init__(self):
        self.robot_id = ROBOT_ID
        self.nats_url = NATS_URL
        self.nats_nc = None
        self.last_joint_time = 0.0
        self.last_imu = None
        self.safe_stop_pub = None
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
        t = make_telemetry(
            self.robot_id,
            online=online,
            actuator_status=actuator_status,
            imu=imu,
            current_task="idle",
        )
        await self.nats_nc.publish(f"telemetry.robots.{self.robot_id}", json.dumps(t).encode())

    def on_joint_states(self, msg):
        with self._lock:
            self.last_joint_time = time.time()

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

    async def handle_command(self, msg):
        data = json.loads(msg.data.decode())
        if data.get("command") == "safe_stop":
            self.publish_safe_stop()

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


class AgibotROS2Node(Node):
    def __init__(self, adapter: AgibotAdapter):
        super().__init__("agibot_adapter")
        self.adapter = adapter
        self.create_subscription(JointState, "/joint_states", adapter.on_joint_states, 10)
        self.create_subscription(Imu, "/imu/data", adapter.on_imu, 10)
        adapter.safe_stop_pub = self.create_publisher(Empty, "/start_control", 10)
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
