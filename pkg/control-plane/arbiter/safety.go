package arbiter

import "github.com/sai-aurosy/platform/pkg/hal"

// SafetyAllow checks if a command is allowed by the Safety Supervisor.
// safe_stop is always allowed (critical safety command).
func SafetyAllow(cmd *hal.Command) bool {
	switch cmd.Command {
	case "safe_stop":
		return true
	case "release_control":
		return true
	default:
		return false
	}
}
