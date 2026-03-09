package hal

import (
	"encoding/json"
	"time"
)

// Robot represents a robot in the fleet registry.
type Robot struct {
	ID               string    `json:"id"`
	Vendor           string    `json:"vendor"`
	Model            string    `json:"model"`
	AdapterEndpoint  string    `json:"adapter_endpoint"`
	TenantID         string    `json:"tenant_id"`
	EdgeID           string    `json:"edge_id,omitempty"`   // Optional: robot is managed by this edge node
	Location         string    `json:"location,omitempty"` // Optional: for fleet grouping (e.g. "Warehouse A")
	Capabilities     []string  `json:"capabilities"`      // walk, stand, safe_stop, cmd_vel, ...
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// JointStateData holds position, velocity, effort for one joint.
type JointStateData struct {
	Name     string  `json:"name"`
	Position float64 `json:"position"`
	Velocity float64 `json:"velocity"`
	Effort   float64 `json:"effort"`
}

// Telemetry is the normalized telemetry payload published to the Telemetry Bus.
type Telemetry struct {
	RobotID         string           `json:"robot_id"`
	Timestamp       time.Time        `json:"timestamp"`
	Online          bool             `json:"online"`
	ActuatorStatus  string           `json:"actuator_status"` // enabled, disabled, error, calibration
	MockMode        bool             `json:"mock_mode,omitempty"`
	IMU             *IMUData         `json:"imu,omitempty"`
	JointStates     []JointStateData  `json:"joint_states,omitempty"`
	CurrentTask     string           `json:"current_task"` // idle, zero, stand, walk
	Position         string   `json:"position,omitempty"`          // "x,y,z" for arrival detection
	TargetPosition   string   `json:"target_position,omitempty"`
	DistanceToTarget *float64 `json:"distance_to_target,omitempty"` // meters; nil if adapter does not report
}

// IMUData holds orientation and angular velocity from IMU.
type IMUData struct {
	Orientation    map[string]float64 `json:"orientation,omitempty"`
	AngularVel     map[string]float64 `json:"angular_velocity,omitempty"`
}

// Command represents a command sent to a robot.
type Command struct {
	RobotID    string          `json:"robot_id"`
	Command    string          `json:"command"` // safe_stop, release_control, cmd_vel, ...
	Payload    json.RawMessage `json:"payload,omitempty"` // for cmd_vel: {"linear_x":0.5,"linear_y":0,"angular_z":0.1}
	Timestamp  time.Time       `json:"timestamp"`
	OperatorID string          `json:"operator_id,omitempty"`
}
