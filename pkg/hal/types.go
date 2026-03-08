package hal

import "time"

// Robot represents a robot in the fleet registry.
type Robot struct {
	ID               string    `json:"id"`
	Vendor           string    `json:"vendor"`
	Model            string    `json:"model"`
	AdapterEndpoint  string    `json:"adapter_endpoint"`
	TenantID         string    `json:"tenant_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Telemetry is the normalized telemetry payload published to the Telemetry Bus.
type Telemetry struct {
	RobotID         string    `json:"robot_id"`
	Timestamp       time.Time `json:"timestamp"`
	Online          bool      `json:"online"`
	ActuatorStatus  string    `json:"actuator_status"` // enabled, disabled, error, calibration
	IMU             *IMUData  `json:"imu,omitempty"`
	CurrentTask     string    `json:"current_task"` // idle, standing, walking
}

// IMUData holds orientation and angular velocity from IMU.
type IMUData struct {
	Orientation    map[string]float64 `json:"orientation,omitempty"`
	AngularVel     map[string]float64 `json:"angular_velocity,omitempty"`
}

// Command represents a command sent to a robot.
type Command struct {
	RobotID   string    `json:"robot_id"`
	Command   string    `json:"command"` // safe_stop, release_control
	Timestamp time.Time `json:"timestamp"`
	OperatorID string   `json:"operator_id,omitempty"`
}
