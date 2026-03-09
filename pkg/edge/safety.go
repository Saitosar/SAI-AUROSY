package edge

import (
	"encoding/json"

	"github.com/sai-aurosy/platform/pkg/hal"
)

const (
	linearVelMin, linearVelMax   = -1.5, 1.5  // m/s
	angularVelMin, angularVelMax = -2.0, 2.0   // rad/s
)

// SafetyAllow checks if a command is allowed by the Safety Supervisor.
// Mirrors pkg/control-plane/arbiter/safety.go for edge-local validation.
func SafetyAllow(cmd *hal.Command) bool {
	switch cmd.Command {
	case "safe_stop":
		return true
	case "release_control":
		return true
	case "zero_mode", "stand_mode", "walk_mode":
		return true
	case "cmd_vel":
		return validateCmdVelPayload(cmd.Payload)
	default:
		return false
	}
}

func validateCmdVelPayload(payload json.RawMessage) bool {
	if len(payload) == 0 {
		return false
	}
	var p struct {
		LinearX  float64 `json:"linear_x"`
		LinearY  float64 `json:"linear_y"`
		AngularZ float64 `json:"angular_z"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return false
	}
	if p.LinearX < linearVelMin || p.LinearX > linearVelMax {
		return false
	}
	if p.LinearY < linearVelMin || p.LinearY > linearVelMax {
		return false
	}
	if p.AngularZ < angularVelMin || p.AngularZ > angularVelMax {
		return false
	}
	return true
}
