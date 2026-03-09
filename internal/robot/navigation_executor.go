package robot

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/arbiter"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

const (
	defaultArrivalThresholdM = 1.0
	defaultNavigationTimeout = 120 * time.Second
)

// NavigationRequest holds parameters for a navigation execution.
type NavigationRequest struct {
	RobotID     string
	TaskID     string
	TargetCoords string
	StoreName  string
	DestNodeID string
	OperatorID string
	IsReturn   bool
	Timeout    time.Duration
}

// NavigationResult indicates the outcome of navigation.
type NavigationResult struct {
	Success   bool
	Timeout   bool
	Cancelled bool
	Reason    string
}

// NavigationExecutor executes navigation commands and monitors for arrival.
type NavigationExecutor struct {
	bus               *telemetry.Bus
	stateManager      *StateManager
	arrivalThresholdM float64
	defaultTimeout    time.Duration
}

// NewNavigationExecutor creates a new NavigationExecutor.
func NewNavigationExecutor(bus *telemetry.Bus, stateManager *StateManager, arrivalThresholdM float64, defaultTimeout time.Duration) *NavigationExecutor {
	if arrivalThresholdM <= 0 {
		arrivalThresholdM = defaultArrivalThresholdM
	}
	if defaultTimeout <= 0 {
		defaultTimeout = defaultNavigationTimeout
	}
	return &NavigationExecutor{
		bus:               bus,
		stateManager:      stateManager,
		arrivalThresholdM: arrivalThresholdM,
		defaultTimeout:    defaultTimeout,
	}
}

// Execute runs navigation: sends commands, monitors telemetry, returns on arrival or timeout.
func (e *NavigationExecutor) Execute(ctx context.Context, req NavigationRequest) NavigationResult {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}
	deadline := time.Now().Add(timeout)

	// Send walk_mode
	walkCmd := &hal.Command{
		RobotID:    req.RobotID,
		Command:    "walk_mode",
		Timestamp:  time.Now(),
		OperatorID: req.OperatorID,
	}
	if arbiter.SafetyAllow(walkCmd) {
		if err := e.bus.PublishCommand(walkCmd); err != nil {
			slog.Warn("navigation_executor walk_mode failed", "robot_id", req.RobotID, "error", err)
		}
	}

	// Build navigate_to payload
	navPayload := map[string]string{
		"target_coordinates": req.TargetCoords,
		"store_name":         req.StoreName,
	}
	if req.DestNodeID != "" {
		navPayload["destination_node_id"] = req.DestNodeID
	}
	payloadBytes, _ := json.Marshal(navPayload)

	navCmd := &hal.Command{
		RobotID:    req.RobotID,
		Command:    "navigate_to",
		Payload:    payloadBytes,
		Timestamp:  time.Now(),
		OperatorID: req.OperatorID,
	}
	if !arbiter.SafetyAllow(navCmd) {
		return NavigationResult{Success: false, Reason: "safety rejected navigate_to"}
	}
	if err := e.bus.PublishCommand(navCmd); err != nil {
		slog.Error("navigation_executor navigate_to failed", "robot_id", req.RobotID, "error", err)
		return NavigationResult{Success: false, Reason: "publish failed"}
	}

	// Monitor telemetry for arrival
	var mu sync.Mutex
	var result NavigationResult
	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	sub, err := e.bus.SubscribeTelemetry(req.RobotID, func(t *hal.Telemetry) {
		mu.Lock()
		if result.Success || result.Timeout || result.Cancelled || result.Reason != "" {
			mu.Unlock()
			return
		}
		if !t.Online {
			result = NavigationResult{Success: false, Reason: "robot offline"}
			mu.Unlock()
			closeDone()
			return
		}
		if t.DistanceToTarget != nil {
			if *t.DistanceToTarget >= 0 && *t.DistanceToTarget < e.arrivalThresholdM {
				result = NavigationResult{Success: true}
				mu.Unlock()
				closeDone()
				return
			}
		}
		mu.Unlock()
	})
	if err != nil {
		slog.Error("navigation_executor subscribe failed", "robot_id", req.RobotID, "error", err)
		return NavigationResult{Success: false, Reason: "subscribe failed"}
	}
	defer sub.Unsubscribe()

	// Poll for context cancel, timeout, or arrival
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			if !result.Success && !result.Timeout {
				result = NavigationResult{Cancelled: true, Reason: "context cancelled"}
			}
			mu.Unlock()
			return result
		case <-done:
			mu.Lock()
			r := result
			mu.Unlock()
			return r
		case <-ticker.C:
			mu.Lock()
			if result.Success || result.Timeout || result.Cancelled || result.Reason != "" {
				r := result
				mu.Unlock()
				return r
			}
			mu.Unlock()
			if time.Now().After(deadline) {
				return NavigationResult{Timeout: true, Reason: "navigation timeout"}
			}
		}
	}
}
