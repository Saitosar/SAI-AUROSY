package validation

// StateCheck helpers for robot state validation.
// Primary logic is in assertions.go; this file holds shared state-check utilities.

// IsTerminalState returns true if the state indicates the robot has finished.
func IsTerminalState(state string) bool {
	switch state {
	case "IDLE", "ERROR_STATE":
		return true
	default:
		return false
	}
}

// IsFailureState returns true if the state indicates the robot has failed.
func IsFailureState(state string) bool {
	return state == "ERROR_STATE"
}
