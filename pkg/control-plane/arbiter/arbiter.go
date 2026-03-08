package arbiter

import (
	"context"
	"log"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Arbiter routes commands to the appropriate adapter via the Telemetry Bus.
type Arbiter struct {
	bus      *telemetry.Bus
	registry *registry.Store
	auditLog []AuditEntry
}

// AuditEntry records a command for audit trail.
type AuditEntry struct {
	RobotID    string    `json:"robot_id"`
	Command    string    `json:"command"`
	OperatorID string    `json:"operator_id"`
	Timestamp  time.Time `json:"timestamp"`
	Allowed    bool      `json:"allowed"`
}

// NewArbiter creates a new Command Arbiter.
func NewArbiter(bus *telemetry.Bus, reg *registry.Store) *Arbiter {
	return &Arbiter{bus: bus, registry: reg, auditLog: make([]AuditEntry, 0, 100)}
}

// Run subscribes to commands and routes them. Blocks until ctx is done.
func (a *Arbiter) Run(ctx context.Context) error {
	sub, err := a.bus.SubscribeAllCommands(a.handleCommand)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return nil
}

func (a *Arbiter) handleCommand(cmd *hal.Command) {
	robot := a.registry.Get(cmd.RobotID)
	if robot == nil {
		log.Printf("[arbiter] robot %s not in registry, rejecting", cmd.RobotID)
		a.audit(cmd, false)
		return
	}
	if !SafetyAllow(cmd) {
		log.Printf("[arbiter] safety rejected command %s for %s", cmd.Command, cmd.RobotID)
		a.audit(cmd, false)
		return
	}
	if err := a.bus.PublishCommand(cmd); err != nil {
		log.Printf("[arbiter] failed to publish command: %v", err)
		a.audit(cmd, false)
		return
	}
	log.Printf("[arbiter] routed %s -> %s", cmd.Command, cmd.RobotID)
	a.audit(cmd, true)
}

func (a *Arbiter) audit(cmd *hal.Command, allowed bool) {
	a.auditLog = append(a.auditLog, AuditEntry{
		RobotID:    cmd.RobotID,
		Command:    cmd.Command,
		OperatorID: cmd.OperatorID,
		Timestamp:  time.Now(),
		Allowed:    allowed,
	})
}
