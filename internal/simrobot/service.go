package simrobot

import (
	"context"
	"fmt"
	"sync"

	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// SimRobotService manages simulated robots.
type SimRobotService struct {
	mu       sync.RWMutex
	bus      *telemetry.Bus
	registry registry.Store
	robots   map[string]*SimRobot
}

// NewSimRobotService creates a new SimRobotService.
func NewSimRobotService(bus *telemetry.Bus, reg registry.Store) *SimRobotService {
	return &SimRobotService{
		bus:      bus,
		registry: reg,
		robots:   make(map[string]*SimRobot),
	}
}

// CreateRobot creates a simulated robot and registers it in the fleet.
func (s *SimRobotService) CreateRobot(opts CreateRobotOpts) (*SimRobot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	robotID := opts.RobotID
	if robotID == "" {
		robotID = RobotIDPrefix + "001"
	}
	if _, ok := s.robots[robotID]; ok {
		return nil, fmt.Errorf("simrobot %s already exists", robotID)
	}

	robot, err := NewSimRobot(opts, s.bus, s.registry)
	if err != nil {
		return nil, err
	}
	s.robots[robotID] = robot
	return robot, nil
}

// Start starts the simulation for a robot.
func (s *SimRobotService) Start(ctx context.Context, robotID string) error {
	s.mu.RLock()
	robot, ok := s.robots[robotID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("simrobot %s not found", robotID)
	}
	return robot.Start(ctx)
}

// Stop stops the simulation for a robot.
func (s *SimRobotService) Stop(robotID string) error {
	s.mu.RLock()
	robot, ok := s.robots[robotID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("simrobot %s not found", robotID)
	}
	robot.Stop()
	return nil
}

// Reset resets the robot state to idle.
func (s *SimRobotService) Reset(robotID string) error {
	s.mu.RLock()
	robot, ok := s.robots[robotID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("simrobot %s not found", robotID)
	}
	robot.Reset()
	return nil
}

// InjectFailure injects a failure scenario for the robot.
func (s *SimRobotService) InjectFailure(robotID string, cfg *FailureConfig) error {
	s.mu.RLock()
	robot, ok := s.robots[robotID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("simrobot %s not found", robotID)
	}
	robot.SetFailureConfig(cfg)
	return nil
}

// GetState returns the current state of the robot.
func (s *SimRobotService) GetState(robotID string) (*SimState, error) {
	s.mu.RLock()
	robot, ok := s.robots[robotID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("simrobot %s not found", robotID)
	}
	state := robot.State()
	return &state, nil
}

// GetRobot returns the SimRobot by ID.
func (s *SimRobotService) GetRobot(robotID string) *SimRobot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.robots[robotID]
}

// ListRobots returns all simulated robot IDs.
func (s *SimRobotService) ListRobots() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.robots))
	for id := range s.robots {
		ids = append(ids, id)
	}
	return ids
}
