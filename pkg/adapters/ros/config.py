"""Topic mapping and configuration for ROS adapter."""

import os


def get_config():
    """Load configuration from environment."""
    ros_distro = os.environ.get("ROS_DISTRO", "humble").lower()
    is_ros1 = ros_distro in ("noetic", "melodic", "kinetic")

    return {
        "robot_id": os.environ.get("ROBOT_ID", "ros-001"),
        "nats_url": os.environ.get("NATS_URL", "nats://localhost:4222"),
        "ros_distro": ros_distro,
        "is_ros1": is_ros1,
        "mock_mode": os.environ.get("ROS_MOCK", "0") == "1",
        "topics": {
            "joint_states": os.environ.get("TELEMETRY_JOINT_STATES", "/joint_states"),
            "imu": os.environ.get("TELEMETRY_IMU", "/imu/data"),
            "cmd_vel": os.environ.get("CMD_VEL", "/cmd_vel"),
            "safe_stop": os.environ.get("SAFE_STOP_TOPIC", "/emergency_stop"),
            "stand": os.environ.get("STAND_TOPIC", "/stand"),
            "walk": os.environ.get("WALK_TOPIC", "/walk"),
            "zero": os.environ.get("ZERO_TOPIC", "/zero"),
        },
    }
