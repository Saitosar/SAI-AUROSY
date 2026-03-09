"""ROS2-specific bridge for the generic ROS adapter."""

import threading
from typing import Callable, Optional

try:
    import rclpy
    from rclpy.node import Node
    from geometry_msgs.msg import Twist
    from sensor_msgs.msg import JointState, Imu
    from std_msgs.msg import Empty
    HAS_ROS2 = True
except ImportError:
    HAS_ROS2 = False
    Node = object
    Twist = None
    JointState = None
    Imu = None
    Empty = None


class ROS2Bridge(Node if HAS_ROS2 else object):
    """ROS2 node that subscribes to telemetry topics and publishes commands."""

    def __init__(self, config: dict, on_joint_states: Callable, on_imu: Callable):
        if not HAS_ROS2:
            raise RuntimeError("rclpy not installed. pip install rclpy")
        super().__init__("sai_aurosy_ros_adapter")
        self.config = config
        self.on_joint_states = on_joint_states
        self.on_imu = on_imu
        topics = config["topics"]

        self.create_subscription(
            JointState,
            topics["joint_states"],
            self._joint_cb,
            10,
        )
        self.create_subscription(
            Imu,
            topics["imu"],
            self._imu_cb,
            10,
        )

        self.cmd_vel_pub = self.create_publisher(Twist, topics["cmd_vel"], 10)
        self.safe_stop_pub = self.create_publisher(Empty, topics["safe_stop"], 10)
        self.stand_pub = self.create_publisher(Empty, topics["stand"], 10)
        self.walk_pub = self.create_publisher(Empty, topics["walk"], 10)
        self.zero_pub = self.create_publisher(Empty, topics["zero"], 10)

    def _joint_cb(self, msg):
        self.on_joint_states(msg)

    def _imu_cb(self, msg):
        self.on_imu(msg)

    def publish_cmd_vel(self, linear_x: float, linear_y: float, angular_z: float):
        twist = Twist()
        twist.linear.x = float(linear_x)
        twist.linear.y = float(linear_y)
        twist.linear.z = 0.0
        twist.angular.x = 0.0
        twist.angular.y = 0.0
        twist.angular.z = float(angular_z)
        self.cmd_vel_pub.publish(twist)

    def publish_safe_stop(self):
        self.safe_stop_pub.publish(Empty())

    def publish_stand(self):
        self.stand_pub.publish(Empty())

    def publish_walk(self):
        self.walk_pub.publish(Empty())

    def publish_zero(self):
        self.zero_pub.publish(Empty())


def run_ros2_spin(bridge: "ROS2Bridge"):
    """Run rclpy.spin in main thread."""
    rclpy.init()
    rclpy.spin(bridge)
