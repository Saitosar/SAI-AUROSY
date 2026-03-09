package cognitive

// NavigateRequest is the input for path planning / navigation.
type NavigateRequest struct {
	RobotID string  `json:"robot_id"`
	From    []float64 `json:"from"` // [x, y] or [x, y, theta]
	To      []float64 `json:"to"`
	MapID   string  `json:"map_id,omitempty"`
}

// NavigateResult is the output of path planning.
type NavigateResult struct {
	Path      [][]float64 `json:"path"`       // sequence of [x, y] waypoints
	Duration  float64     `json:"duration"`    // estimated duration in seconds
	Distance  float64     `json:"distance"`    // path length
}

// RecognizeRequest is the input for object/person recognition.
type RecognizeRequest struct {
	RobotID    string      `json:"robot_id"`
	SensorData interface{} `json:"sensor_data"` // image URL, base64, or sensor payload
}

// RecognizedObject represents a detected object.
type RecognizedObject struct {
	Class      string    `json:"class"`       // person, obstacle, etc.
	Confidence float64   `json:"confidence"`
	BBox       []float64 `json:"bbox,omitempty"` // [x, y, w, h]
	Distance   float64   `json:"distance,omitempty"`
}

// RecognizeResult is the output of recognition.
type RecognizeResult struct {
	Objects []RecognizedObject `json:"objects"`
}

// PlanRequest is the input for task/action planning.
type PlanRequest struct {
	TaskType string                 `json:"task_type"`
	Context  map[string]interface{} `json:"context"`
}

// PlanResult is the output of planning.
type PlanResult struct {
	Steps []PlanStep `json:"steps"`
}

// PlanStep is a single step in a plan.
type PlanStep struct {
	Action   string                 `json:"action"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	DurationSec int                `json:"duration_sec,omitempty"`
}
