package hal

import "context"

// RobotAdapter is the interface that all robot adapters must implement.
// It abstracts vendor-specific APIs (AimRT, ROS2, SDK) behind a unified contract.
type RobotAdapter interface {
	// Connect establishes connection to the robot runtime (AimRT, ROS2, etc.).
	Connect(ctx context.Context) error

	// SubscribeTelemetry registers a callback for telemetry updates from the robot.
	// The adapter should call the callback whenever new state is received.
	SubscribeTelemetry(callback func(*Telemetry)) error

	// SendCommand sends a command to the robot.
	// Supported commands: safe_stop, release_control
	SendCommand(ctx context.Context, cmd *Command) error

	// Disconnect closes the connection to the robot.
	Disconnect() error
}
