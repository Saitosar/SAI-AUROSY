#!/usr/bin/env python3
"""
SAI-AUROSY Adapter Template - minimal mock adapter for the Telemetry Bus (NATS).

Use this as a starting point for implementing a new robot adapter.
Copy to pkg/adapters/<vendor>/ and replace the placeholder logic with your robot SDK/ROS integration.

NATS: publishes telemetry.robots.{robot_id}; subscribes to commands.robots.{robot_id}

Usage:
  ROBOT_ID=my-robot-001 NATS_URL=nats://localhost:4222 python adapter.py

See docs/adapters/robot-adapter-contract.md for the full contract.
"""

import asyncio
import json
import os
from datetime import datetime, timezone

try:
    import nats
    HAS_NATS = True
except ImportError:
    HAS_NATS = False


ROBOT_ID = os.environ.get("ROBOT_ID", "template-001")
NATS_URL = os.environ.get("NATS_URL", "nats://localhost:4222")
TELEMETRY_INTERVAL = float(os.environ.get("TELEMETRY_INTERVAL", "1.0"))


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


class TemplateAdapter:
    """
    Minimal adapter template. Replace the command handlers with your robot SDK calls.
    """

    def __init__(self):
        self.robot_id = ROBOT_ID
        self.nats_url = NATS_URL
        self.nats_nc = None
        self.last_mode_command = "idle"

    async def connect_nats(self):
        if not HAS_NATS:
            raise RuntimeError("nats-py not installed. pip install nats-py")
        self.nats_nc = await nats.connect(self.nats_url)
        print(f"[template] Connected to NATS at {self.nats_url}")

    async def publish_telemetry(self):
        """Publish telemetry to telemetry.robots.{robot_id}."""
        if not self.nats_nc:
            return
        t = make_telemetry(
            self.robot_id,
            online=True,
            actuator_status="enabled",
            current_task=self.last_mode_command,
            mock_mode=True,
        )
        await self.nats_nc.publish(
            f"telemetry.robots.{self.robot_id}",
            json.dumps(t).encode(),
        )

    async def handle_command(self, msg):
        """Handle command from commands.robots.{robot_id}. Replace with your logic."""
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
            # Replace: call your robot's emergency stop
            print(f"[{self.robot_id}] safe_stop (replace with robot SDK call)")

        elif cmd == "release_control":
            # Replace: release control to operator
            print(f"[{self.robot_id}] release_control (replace with robot SDK call)")

        elif cmd == "zero_mode":
            self.last_mode_command = "zero"
            # Replace: send zero joints command
            print(f"[{self.robot_id}] zero_mode (replace with robot SDK call)")

        elif cmd == "stand_mode":
            self.last_mode_command = "stand"
            # Replace: send stand command
            print(f"[{self.robot_id}] stand_mode (replace with robot SDK call)")

        elif cmd == "walk_mode":
            self.last_mode_command = "walk"
            # Replace: send walk mode command
            print(f"[{self.robot_id}] walk_mode (replace with robot SDK call)")

        elif cmd == "cmd_vel":
            self.last_mode_command = "walk"
            payload = payload or {}
            linear_x = float(payload.get("linear_x", 0))
            linear_y = float(payload.get("linear_y", 0))
            angular_z = float(payload.get("angular_z", 0))
            # Replace: send velocity command to robot
            print(f"[{self.robot_id}] cmd_vel linear_x={linear_x} linear_y={linear_y} angular_z={angular_z} (replace with robot SDK call)")

        else:
            print(f"[{self.robot_id}] unknown command: {cmd}")

    async def run(self):
        """Main loop: connect, subscribe to commands, publish telemetry periodically."""
        await self.connect_nats()

        async def cmd_cb(msg):
            await self.handle_command(msg)

        await self.nats_nc.subscribe(
            f"commands.robots.{self.robot_id}",
            cb=cmd_cb,
        )
        print(f"[{self.robot_id}] Template adapter running. Publishing telemetry every {TELEMETRY_INTERVAL}s")

        while True:
            await self.publish_telemetry()
            await asyncio.sleep(TELEMETRY_INTERVAL)


async def main():
    adapter = TemplateAdapter()
    await adapter.run()


if __name__ == "__main__":
    asyncio.run(main())
