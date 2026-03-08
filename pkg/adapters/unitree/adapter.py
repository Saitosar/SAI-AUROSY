#!/usr/bin/env python3
"""
Unitree Go2 Adapter - bridges ROS2 (unitree_ros2) / unitree_sdk2_python and SAI-AUROSY Telemetry Bus (NATS).

ROS2: subscribes to sportmodestate, lowstate; publishes to /api/sport/request
Alternative: unitree_sdk2_python SportClient for commands (when unitree_ros2 not built)
NATS: publishes telemetry.robots.{robot_id}; subscribes to commands.robots.{robot_id}

Usage:
  UNITREE_MOCK=1 python adapter.py   # Mock mode (no ROS2/SDK)
  python adapter.py                  # ROS2/SDK mode (requires unitree_ros2 or unitree_sdk2_python)
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
    HAS_ROS2 = True
except ImportError:
    HAS_ROS2 = False

try:
    import nats
    HAS_NATS = True
except ImportError:
    HAS_NATS = False

HAS_UNITREE_ROS2 = False
HAS_UNITREE_SDK2 = False
SportClient = None

if HAS_ROS2:
    try:
        from unitree_go.msg import SportModeState, LowState
        from unitree_api.msg import Request
        HAS_UNITREE_ROS2 = True
    except ImportError:
        pass

if not HAS_UNITREE_ROS2:
    try:
        from unitree_sdk2py.go2.sport.sport_client import SportClient as _SportClient
        from unitree_sdk2py.core.channel import ChannelFactoryInitialize
        SportClient = _SportClient
        HAS_UNITREE_SDK2 = True
    except ImportError:
        pass

ROBOT_ID = os.environ.get("ROBOT_ID", "go2-001")
NATS_URL = os.environ.get("NATS_URL", "nats://localhost:4222")
MOCK_MODE = os.environ.get("UNITREE_MOCK", "0") == "1"


def make_telemetry(robot_id: str, online: bool, actuator_status: str = "enabled",
                   imu: dict = None, joint_states: list = None, current_task: str = "idle",
                   mock_mode: bool = False) -> dict:
    t = {
        "robot_id": robot_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "online": online,
        "actuator_status": actuator_status,
        "imu": imu,
        "current_task": current_task,
        "mock_mode": mock_mode,
    }
    if joint_states:
        t["joint_states"] = joint_states
    return t


class UnitreeAdapter:
    def __init__(self):
        self.robot_id = ROBOT_ID
        self.nats_url = NATS_URL
        self.nats_nc = None
        self.last_state_time = 0.0
        self.last_imu = None
        self.sport_request_pub = None
        self.sport_client = None
        self.last_joint_states = []
        self.last_mode_command = "idle"
        self._lock = threading.Lock()

    async def connect_nats(self):
        if not HAS_NATS:
            raise RuntimeError("nats-py not installed. pip install nats-py")
        self.nats_nc = await nats.connect(self.nats_url)
        print(f"[unitree] Connected to NATS at {self.nats_url}")

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

    def on_sport_mode_state(self, msg):
        with self._lock:
            self.last_state_time = time.time()
            if hasattr(msg, "imu_state") and msg.imu_state:
                imu = msg.imu_state
                self.last_imu = {
                    "orientation": {
                        "x": getattr(imu, "quaternion", [0, 0, 0, 1])[0] if hasattr(imu, "quaternion") else 0,
                        "y": getattr(imu, "quaternion", [0, 0, 0, 1])[1] if hasattr(imu, "quaternion") else 0,
                        "z": getattr(imu, "quaternion", [0, 0, 0, 1])[2] if hasattr(imu, "quaternion") else 0,
                        "w": getattr(imu, "quaternion", [0, 0, 0, 1])[3] if hasattr(imu, "quaternion") else 1,
                    },
                    "angular_velocity": {
                        "x": getattr(imu, "gyroscope", [0, 0, 0])[0] if hasattr(imu, "gyroscope") else 0,
                        "y": getattr(imu, "gyroscope", [0, 0, 0])[1] if hasattr(imu, "gyroscope") else 0,
                        "z": getattr(imu, "gyroscope", [0, 0, 0])[2] if hasattr(imu, "gyroscope") else 0,
                    },
                }

    def on_low_state(self, msg):
        with self._lock:
            self.last_state_time = time.time()
            states = []
            if hasattr(msg, "motor_state") and msg.motor_state:
                for i, m in enumerate(msg.motor_state):
                    q = getattr(m, "q", 0.0) if hasattr(m, "q") else 0.0
                    dq = getattr(m, "dq", 0.0) if hasattr(m, "dq") else 0.0
                    tau = getattr(m, "tau_est", 0.0) if hasattr(m, "tau_est") else 0.0
                    states.append({
                        "name": f"motor_{i}",
                        "position": float(q),
                        "velocity": float(dq),
                        "effort": float(tau),
                    })
                self.last_joint_states = states
            if hasattr(msg, "imu_state") and msg.imu_state:
                imu = msg.imu_state
                self.last_imu = {
                    "orientation": {
                        "x": getattr(imu, "quaternion", [0, 0, 0, 1])[0] if hasattr(imu, "quaternion") else 0,
                        "y": getattr(imu, "quaternion", [0, 0, 0, 1])[1] if hasattr(imu, "quaternion") else 0,
                        "z": getattr(imu, "quaternion", [0, 0, 0, 1])[2] if hasattr(imu, "quaternion") else 0,
                        "w": getattr(imu, "quaternion", [0, 0, 0, 1])[3] if hasattr(imu, "quaternion") else 1,
                    },
                    "angular_velocity": {
                        "x": getattr(imu, "gyroscope", [0, 0, 0])[0] if hasattr(imu, "gyroscope") else 0,
                        "y": getattr(imu, "gyroscope", [0, 0, 0])[1] if hasattr(imu, "gyroscope") else 0,
                        "z": getattr(imu, "gyroscope", [0, 0, 0])[2] if hasattr(imu, "gyroscope") else 0,
                    },
                }

    def publish_safe_stop(self):
        if HAS_UNITREE_SDK2 and self.sport_client:
            try:
                self.sport_client.Damp()
                print(f"[{self.robot_id}] safe_stop -> Damp (SDK2)")
            except Exception as e:
                print(f"[{self.robot_id}] safe_stop error: {e}")
        elif self.sport_request_pub and HAS_UNITREE_ROS2:
            try:
                req = Request()
                req.parameter = json.dumps({})
                req.binary = []
                self._publish_sport_request(req, "Damp")
                print(f"[{self.robot_id}] safe_stop -> Damp (ROS2)")
            except Exception as e:
                print(f"[{self.robot_id}] safe_stop error: {e}")
        else:
            print(f"[{self.robot_id}] safe_stop (mock)")

    def _publish_sport_request(self, req, api_name: str):
        if self.sport_request_pub:
            self.sport_request_pub.publish(req)

    def publish_stand_mode(self):
        if HAS_UNITREE_SDK2 and self.sport_client:
            try:
                self.sport_client.BalanceStand()
                print(f"[{self.robot_id}] stand_mode -> BalanceStand (SDK2)")
            except Exception as e:
                print(f"[{self.robot_id}] stand_mode error: {e}")
        elif self.sport_request_pub and HAS_UNITREE_ROS2:
            try:
                req = Request()
                req.parameter = json.dumps({})
                req.binary = []
                self._publish_sport_request(req, "BalanceStand")
                print(f"[{self.robot_id}] stand_mode -> BalanceStand (ROS2)")
            except Exception as e:
                print(f"[{self.robot_id}] stand_mode error: {e}")
        else:
            print(f"[{self.robot_id}] stand_mode (mock)")

    def publish_walk_mode(self):
        if HAS_UNITREE_SDK2 and self.sport_client:
            try:
                self.sport_client.StopMove()
                self.sport_client.Move(0, 0, 0)
                print(f"[{self.robot_id}] walk_mode -> Move(0,0,0) (SDK2)")
            except Exception as e:
                print(f"[{self.robot_id}] walk_mode error: {e}")
        elif self.sport_request_pub and HAS_UNITREE_ROS2:
            try:
                req = Request()
                req.parameter = json.dumps({"x": 0, "y": 0, "z": 0})
                req.binary = []
                self._publish_sport_request(req, "Move")
                print(f"[{self.robot_id}] walk_mode -> Move (ROS2)")
            except Exception as e:
                print(f"[{self.robot_id}] walk_mode error: {e}")
        else:
            print(f"[{self.robot_id}] walk_mode (mock)")

    def publish_zero_mode(self):
        if HAS_UNITREE_SDK2 and self.sport_client:
            try:
                self.sport_client.StandDown()
                print(f"[{self.robot_id}] zero_mode -> StandDown (SDK2)")
            except Exception as e:
                print(f"[{self.robot_id}] zero_mode error: {e}")
        else:
            print(f"[{self.robot_id}] zero_mode (mock)")

    def publish_cmd_vel(self, linear_x: float, linear_y: float, angular_z: float):
        if HAS_UNITREE_SDK2 and self.sport_client:
            try:
                self.sport_client.Move(float(linear_x), float(linear_y), float(angular_z))
                print(f"[{self.robot_id}] cmd_vel -> Move({linear_x},{linear_y},{angular_z}) (SDK2)")
            except Exception as e:
                print(f"[{self.robot_id}] cmd_vel error: {e}")
        elif self.sport_request_pub and HAS_UNITREE_ROS2:
            try:
                req = Request()
                req.parameter = json.dumps({"x": linear_x, "y": linear_y, "z": angular_z})
                req.binary = []
                self._publish_sport_request(req, "Move")
                print(f"[{self.robot_id}] cmd_vel -> Move (ROS2)")
            except Exception as e:
                print(f"[{self.robot_id}] cmd_vel error: {e}")
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
            await self.publish_telemetry(online=True, actuator_status="enabled", mock_mode=True)
            await asyncio.sleep(1)


if HAS_ROS2 and HAS_UNITREE_ROS2:
    class UnitreeROS2Node(Node):
        def __init__(self, adapter: UnitreeAdapter):
            super().__init__("unitree_adapter")
            self.adapter = adapter
            self.create_subscription(
                SportModeState, "sportmodestate", adapter.on_sport_mode_state, 10
            )
            self.create_subscription(
                LowState, "lowstate", adapter.on_low_state, 10
            )
            adapter.sport_request_pub = self.create_publisher(Request, "/api/sport/request", 10)
            self.get_logger().info(f"Unitree adapter ROS2, robot_id={adapter.robot_id}")


async def run_nats_loop(adapter: UnitreeAdapter):
    await adapter.connect_nats()
    async def cmd_cb(msg):
        await adapter.handle_command(msg)

    await adapter.nats_nc.subscribe(
        f"commands.robots.{adapter.robot_id}",
        cb=cmd_cb,
    )
    while True:
        with adapter._lock:
            online = (time.time() - adapter.last_state_time) < 2.0
        await adapter.publish_telemetry(
            online=online,
            actuator_status="enabled" if online else "unknown",
            mock_mode=False,
        )
        await asyncio.sleep(0.5)


def run_ros2_spin(adapter: UnitreeAdapter):
    rclpy.init()
    node = UnitreeROS2Node(adapter)
    rclpy.spin(node)


def init_sdk2_client(adapter: UnitreeAdapter):
    if not HAS_UNITREE_SDK2 or not SportClient:
        return False
    try:
        ChannelFactoryInitialize(0)
        adapter.sport_client = SportClient()
        adapter.sport_client.Init()
        return True
    except Exception as e:
        print(f"[unitree] SDK2 init failed: {e}")
        return False


async def main():
    adapter = UnitreeAdapter()

    if MOCK_MODE:
        await adapter.run_mock()
        return

    if not HAS_ROS2 and not HAS_UNITREE_SDK2:
        print("[unitree] No ROS2 or unitree_sdk2_python; falling back to mock")
        await adapter.run_mock()
        return

    if HAS_UNITREE_SDK2:
        if init_sdk2_client(adapter):
            print("[unitree] Using unitree_sdk2_python for commands")

    if HAS_ROS2 and HAS_UNITREE_ROS2:
        t = threading.Thread(target=run_ros2_spin, args=(adapter,), daemon=True)
        t.start()
        await asyncio.sleep(2)
    elif not adapter.sport_client:
        print("[unitree] No ROS2 unitree packages; falling back to mock")
        await adapter.run_mock()
        return

    await run_nats_loop(adapter)


if __name__ == "__main__":
    asyncio.run(main())
