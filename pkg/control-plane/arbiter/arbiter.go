package arbiter

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/audit"
	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Arbiter routes commands to the appropriate adapter via the Telemetry Bus.
type Arbiter struct {
	bus        *telemetry.Bus
	registry   registry.Store
	auditStore audit.Store
}

// NewArbiter creates a new Command Arbiter. auditStore is optional.
func NewArbiter(bus *telemetry.Bus, reg registry.Store, auditStore audit.Store) *Arbiter {
	return &Arbiter{bus: bus, registry: reg, auditStore: auditStore}
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
	if a.auditStore == nil {
		return
	}
	details, _ := json.Marshal(map[string]any{"command": cmd.Command, "allowed": allowed})
	_ = a.auditStore.Append(context.Background(), &audit.Entry{
		Actor:      nullIfEmpty(cmd.OperatorID),
		Action:     "command",
		Resource:   "robot",
		ResourceID: cmd.RobotID,
		Timestamp:  time.Now(),
		Details:    string(details),
	})
}

func nullIfEmpty(s string) string {
	if s == "" {
		return "system"
	}
	return s
}
