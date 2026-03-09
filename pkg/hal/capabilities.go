package hal

// Standard capability constants for robots.
// Used for filtering tasks by robot capabilities.
const (
	CapWalk          = "walk"
	CapStand         = "stand"
	CapSafeStop      = "safe_stop"
	CapReleaseControl = "release_control"
	CapCmdVel        = "cmd_vel"
	CapZeroMode      = "zero_mode"
	CapPatrol        = "patrol"
	CapNavigation    = "navigation"
	CapSpeech        = "speech"
)

// HasCapability returns true if the robot has all capabilities in required.
func HasCapability(robot *Robot, required []string) bool {
	if robot == nil || len(required) == 0 {
		return true
	}
	capSet := make(map[string]bool)
	for _, c := range robot.Capabilities {
		capSet[c] = true
	}
	for _, r := range required {
		if !capSet[r] {
			return false
		}
	}
	return true
}
