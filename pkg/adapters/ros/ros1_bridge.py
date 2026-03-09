"""ROS1-specific bridge for the generic ROS adapter."""

try:
    import rospy
    from geometry_msgs.msg import Twist
    from sensor_msgs.msg import JointState, Imu
    from std_msgs.msg import Empty
    HAS_ROS1 = True
except ImportError:
    HAS_ROS1 = False
    Twist = None
    JointState = None
    Imu = None
    Empty = None


class ROS1Bridge:
    """ROS1 node that subscribes to telemetry topics and publishes commands."""

    def __init__(self, config: dict, on_joint_states: callable, on_imu: callable):
        if not HAS_ROS1:
            raise RuntimeError("rospy not installed. Install ROS1 (e.g. melodic, noetic)")
        self.config = config
        self.on_joint_states = on_joint_states
        self.on_imu = on_imu
        topics = config["topics"]

        rospy.init_node("sai_aurosy_ros_adapter", anonymous=True)
        rospy.Subscriber(topics["joint_states"], JointState, on_joint_states, queue_size=10)
        rospy.Subscriber(topics["imu"], Imu, on_imu, queue_size=10)

        self.cmd_vel_pub = rospy.Publisher(topics["cmd_vel"], Twist, queue_size=10)
        self.safe_stop_pub = rospy.Publisher(topics["safe_stop"], Empty, queue_size=10)
        self.stand_pub = rospy.Publisher(topics["stand"], Empty, queue_size=10)
        self.walk_pub = rospy.Publisher(topics["walk"], Empty, queue_size=10)
        self.zero_pub = rospy.Publisher(topics["zero"], Empty, queue_size=10)

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


def run_ros1_spin(bridge: "ROS1Bridge"):
    """Run rospy.spin."""
    rospy.spin()
