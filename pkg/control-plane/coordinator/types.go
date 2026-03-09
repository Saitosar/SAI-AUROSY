package coordinator

import "time"

// Zone represents a named zone that can be occupied by a robot.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZoneLock represents an exclusive lock on a zone held by a robot.
type ZoneLock struct {
	ZoneID   string    `json:"zone_id"`
	RobotID  string    `json:"robot_id"`
	Acquired time.Time `json:"acquired_at"`
}

// ZoneStatus represents the current status of a zone.
type ZoneStatus struct {
	ZoneID   string  `json:"zone_id"`
	RobotID  *string `json:"robot_id,omitempty"` // nil if available
	Occupied bool    `json:"occupied"`
}
