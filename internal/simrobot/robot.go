package simrobot

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

const (
	defaultTickInterval = 500 * time.Millisecond
	defaultSpeed        = 1.0 // m/s
	defaultBatteryLevel = 100.0
)

// SimRobot represents a simulated robot that subscribes to commands and publishes telemetry.
type SimRobot struct {
	mu sync.RWMutex

	id       string
	tenantID string
	state    SimState

	bus      *telemetry.Bus
	registry registry.Store

	cmdSub    *nats.Subscription
	ticker    *time.Ticker
	cancel    context.CancelFunc
	running   bool

	failureConfig *FailureConfig
}

// CreateRobotOpts configures a new simulated robot.
type CreateRobotOpts struct {
	RobotID   string
	TenantID  string
	RobotType string
}

// NewSimRobot creates a simulated robot and registers it in the fleet registry.
func NewSimRobot(opts CreateRobotOpts, bus *telemetry.Bus, reg registry.Store) (*SimRobot, error) {
	robotID := opts.RobotID
	if robotID == "" {
		robotID = RobotIDPrefix + "001"
	}
	tenantID := opts.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	robotType := opts.RobotType
	if robotType == "" {
		robotType = "simulated"
	}

	caps := []string{
		hal.CapWalk, hal.CapStand, hal.CapSafeStop, hal.CapReleaseControl,
		hal.CapCmdVel, hal.CapPatrol, hal.CapNavigation, hal.CapSpeech,
	}

	now := time.Now()
	reg.Add(&hal.Robot{
		ID:              robotID,
		Vendor:          "simulated",
		Model:           robotType,
		AdapterEndpoint: "internal",
		TenantID:        tenantID,
		Capabilities:    caps,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	r := &SimRobot{
		id:       robotID,
		tenantID: tenantID,
		state: SimState{
			RobotID:        robotID,
			TenantID:       tenantID,
			RobotType:      robotType,
			Capabilities:   caps,
			Online:         true,
			Mode:           ModeIdle,
			Position:       Position{X: 0, Y: 0},
			TargetPosition: Position{},
			Speed:          defaultSpeed,
			BatteryLevel:   defaultBatteryLevel,
			TickCount:      0,
			UpdatedAt:      now,
		},
		bus:      bus,
		registry: reg,
	}
	return r, nil
}

// ID returns the robot ID.
func (r *SimRobot) ID() string {
	return r.id
}

// Start subscribes to commands and starts the telemetry tick loop.
func (r *SimRobot) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil
	}

	sub, err := r.bus.SubscribeCommands(r.id, func(cmd *hal.Command) {
		HandleCommand(r, cmd)
	})
	if err != nil {
		r.mu.Unlock()
		return err
	}

	r.cmdSub = sub
	r.running = true
	tickCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.mu.Unlock()

	go r.runTickLoop(tickCtx)
	slog.Info("simrobot started", "robot_id", r.id)
	return nil
}

// Stop unsubscribes from commands and stops the tick loop.
func (r *SimRobot) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.ticker != nil {
		r.ticker.Stop()
		r.ticker = nil
	}
	if r.cmdSub != nil {
		_ = r.cmdSub.Unsubscribe()
		r.cmdSub = nil
	}
	r.running = false
	slog.Info("simrobot stopped", "robot_id", r.id)
}

// Reset resets the robot state to idle at base position.
func (r *SimRobot) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.Mode = ModeIdle
	r.state.Position = Position{X: 0, Y: 0}
	r.state.TargetPosition = Position{}
	r.state.RouteNodes = nil
	r.state.RouteIndex = 0
	r.state.DistanceToTarget = 0
	r.state.Online = true
	r.state.CurrentTaskID = ""
	r.state.CurrentScenario = ""
	r.state.LastCommand = ""
	r.state.TickCount = 0
	r.state.UpdatedAt = time.Now()
	r.failureConfig = nil
}

// runTickLoop runs the simulation tick loop.
func (r *SimRobot) runTickLoop(ctx context.Context) {
	r.ticker = time.NewTicker(defaultTickInterval)
	defer r.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.ticker.C:
			Tick(r)
		}
	}
}

// State returns a copy of the current state.
func (r *SimRobot) State() SimState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.copyState()
}

func (r *SimRobot) copyState() SimState {
	s := r.state
	if len(r.state.RouteNodes) > 0 {
		s.RouteNodes = make([]string, len(r.state.RouteNodes))
		copy(s.RouteNodes, r.state.RouteNodes)
	}
	return s
}

// SetFailureConfig sets the failure injection config for the robot.
func (r *SimRobot) SetFailureConfig(cfg *FailureConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failureConfig = cfg
}

// ClearFailureConfig clears any failure injection config.
func (r *SimRobot) ClearFailureConfig() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failureConfig = nil
}
