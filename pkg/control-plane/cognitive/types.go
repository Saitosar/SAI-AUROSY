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
	Action      string                 `json:"action"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	DurationSec int                    `json:"duration_sec,omitempty"`
}

// TranscribeRequest is the input for speech-to-text (STT).
type TranscribeRequest struct {
	RobotID     string `json:"robot_id"`
	AudioBase64 string `json:"audio_base64"` // base64-encoded audio (e.g. PCM 16kHz mono)
	Language    string `json:"language,omitempty"` // hint: uz, en, ru, az, ar; empty = auto-detect
}

// TranscribeResult is the output of STT.
type TranscribeResult struct {
	Text       string  `json:"text"`
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// SynthesizeRequest is the input for text-to-speech (TTS).
type SynthesizeRequest struct {
	RobotID  string `json:"robot_id"`
	Text     string `json:"text"`
	Language string `json:"language"` // uz, en, ru, az, ar
}

// SynthesizeResult is the output of TTS.
type SynthesizeResult struct {
	AudioBase64 string `json:"audio_base64"` // base64-encoded audio
}

// UnderstandIntentRequest is the input for intent extraction.
type UnderstandIntentRequest struct {
	RobotID   string `json:"robot_id"`
	Text      string `json:"text"`
	Language  string `json:"language,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"` // optional conversation context
}

// IntentResult is the output of intent understanding.
type IntentResult struct {
	Intent     string                 `json:"intent"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Confidence float64                `json:"confidence"`
}

// TranslateRequest is the input for text translation.
type TranslateRequest struct {
	RobotID         string `json:"robot_id"`
	Text            string `json:"text"`
	TargetLanguage  string `json:"target_language"` // uz, en, ru, az, ar
}

// TranslateResult is the output of translation.
type TranslateResult struct {
	Text string `json:"text"`
}
