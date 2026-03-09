package robot

import (
	"sync"
	"time"
)

// StateManager provides thread-safe per-robot context storage.
type StateManager struct {
	mu   sync.RWMutex
	ctxs map[string]*RobotContext
}

// NewStateManager creates a new StateManager.
func NewStateManager() *StateManager {
	return &StateManager{
		ctxs: make(map[string]*RobotContext),
	}
}

// Get returns the context for the robot, or nil if not found.
func (m *StateManager) Get(robotID string) *RobotContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ctx := m.ctxs[robotID]
	if ctx == nil {
		return nil
	}
	// Return a copy to avoid races
	return copyContext(ctx)
}

// Set stores or updates the context for the robot.
func (m *StateManager) Set(ctx *RobotContext) {
	if ctx == nil {
		return
	}
	ctx.UpdatedAt = time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctxs[ctx.RobotID] = copyContext(ctx)
}

// Transition updates the robot state if the transition is valid.
// Returns true if the transition was applied.
func (m *StateManager) Transition(robotID string, toState RobotState, taskID, targetStore, destNode, targetCoords, statusMsg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctx := m.ctxs[robotID]
	if ctx == nil {
		ctx = &RobotContext{
			RobotID:         robotID,
			CurrentState:    StateIdle,
			PreviousState:   StateIdle,
			CurrentTaskID:   taskID,
			TargetStore:     targetStore,
			DestinationNode: destNode,
			TargetCoords:    targetCoords,
			StatusMessage:   statusMsg,
			UpdatedAt:       time.Now(),
		}
		m.ctxs[robotID] = ctx
	}
	if !CanTransition(ctx.CurrentState, toState) {
		return false
	}
	ctx.PreviousState = ctx.CurrentState
	ctx.CurrentState = toState
	ctx.CurrentTaskID = taskID
	ctx.TargetStore = targetStore
	ctx.DestinationNode = destNode
	ctx.TargetCoords = targetCoords
	ctx.StatusMessage = statusMsg
	ctx.UpdatedAt = time.Now()
	m.ctxs[robotID] = ctx
	return true
}

// GetOrCreate returns the context for the robot, creating one with IDLE state if not found.
func (m *StateManager) GetOrCreate(robotID string) *RobotContext {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctx := m.ctxs[robotID]
	if ctx == nil {
		ctx = &RobotContext{
			RobotID:       robotID,
			CurrentState:  StateIdle,
			PreviousState: StateIdle,
			UpdatedAt:     time.Now(),
		}
		m.ctxs[robotID] = ctx
	}
	return copyContext(ctx)
}

// List returns all robot contexts (copies).
func (m *StateManager) List() []*RobotContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*RobotContext, 0, len(m.ctxs))
	for _, ctx := range m.ctxs {
		out = append(out, copyContext(ctx))
	}
	return out
}

func copyContext(c *RobotContext) *RobotContext {
	if c == nil {
		return nil
	}
	return &RobotContext{
		RobotID:         c.RobotID,
		CurrentState:    c.CurrentState,
		PreviousState:   c.PreviousState,
		CurrentTaskID:   c.CurrentTaskID,
		TargetStore:     c.TargetStore,
		DestinationNode: c.DestinationNode,
		TargetCoords:    c.TargetCoords,
		StatusMessage:   c.StatusMessage,
		UpdatedAt:       c.UpdatedAt,
	}
}
