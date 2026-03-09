package simrobot

// Predefined failure scenarios for testing.
var (
	// ScenarioNavigationTimeout fails after 10 ticks (never arrives).
	ScenarioNavigationTimeout = &FailureConfig{
		Type:       FailureNavigationTimeout,
		AfterTicks: 10,
	}

	// ScenarioOfflineMidRoute goes offline after 5 ticks.
	ScenarioOfflineMidRoute = &FailureConfig{
		Type:       FailureOfflineMidRoute,
		AfterTicks: 5,
	}

	// ScenarioSafeStopMidRoute triggers safe_stop after 5 ticks.
	ScenarioSafeStopMidRoute = &FailureConfig{
		Type:       FailureSafeStopMidRoute,
		AfterTicks: 5,
	}

	// ScenarioBatteryLow sets battery to 5% after 3 ticks.
	ScenarioBatteryLow = &FailureConfig{
		Type:         FailureBatteryLow,
		AfterTicks:   3,
		BatteryLevel: 5,
	}

	// ScenarioOfflineAfterNavigate goes offline after receiving navigate_to.
	ScenarioOfflineAfterNavigate = &FailureConfig{
		Type:         FailureOfflineMidRoute,
		AfterCommand: "navigate_to",
		AfterTicks:   1,
	}

	// ScenarioDelayedArrival slows down to 10% speed for 20 ticks after 2 ticks.
	ScenarioDelayedArrival = &FailureConfig{
		Type:           FailureDelayedArrival,
		AfterTicks:     2,
		SlowdownFactor: 0.1,
		DurationTicks:  20,
	}
)
