package simrobot

// applyFailure checks the failure config and applies it if conditions are met.
// Returns true if a failure was applied (caller should skip normal movement).
// Caller must hold robot.mu.
func applyFailure(robot *SimRobot) bool {
	cfg := robot.failureConfig
	if cfg == nil {
		return false
	}

	state := &robot.state

	// Check trigger conditions
	triggered := false
	if cfg.AfterTicks > 0 && state.TickCount >= cfg.AfterTicks {
		triggered = true
	}
	if cfg.WhenDistanceLt > 0 && state.DistanceToTarget > 0 && state.DistanceToTarget < cfg.WhenDistanceLt {
		triggered = true
	}
	if cfg.AfterCommand != "" && state.LastCommand == cfg.AfterCommand && state.TickCount > 0 {
		triggered = true
	}

	if !triggered {
		return false
	}

	// Apply failure
	switch cfg.Type {
	case FailureNavigationTimeout:
		// Simulate timeout: stop moving, keep distance high so arrival never happens
		state.Mode = ModeError
		state.DistanceToTarget = 999 // never arrive
	case FailureOfflineMidRoute:
		state.Online = false
		state.Mode = ModeError
	case FailureSafeStopMidRoute:
		state.Mode = ModeSafeStop
		state.TargetPosition = Position{}
		state.RouteNodes = nil
		state.DistanceToTarget = 0
	case FailureBatteryLow:
		if cfg.BatteryLevel > 0 {
			state.BatteryLevel = cfg.BatteryLevel
		} else {
			state.BatteryLevel = 5
		}
	case FailureCommandRejected:
		// Next command would be rejected - for sim we just clear; actual rejection handled in handler
		state.Mode = ModeError
	case FailureDelayedArrival:
		if !cfg.applied && cfg.SlowdownFactor > 0 {
			state.Speed *= cfg.SlowdownFactor
			cfg.applied = true
		}
		if cfg.DurationTicks > 0 {
			cfg.DurationTicks--
			if cfg.DurationTicks <= 0 {
				robot.failureConfig = nil
			}
		} else {
			robot.failureConfig = nil
		}
		return false // continue movement, just slower
	}
	robot.failureConfig = nil // one-shot
	return true
}
