package edge

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Agent runs the edge sync loop: heartbeat to cloud, relay commands to local NATS.
type Agent struct {
	cfg       *Config
	bus       *telemetry.Bus
	httpClient *http.Client
}

// NewAgent creates a new edge agent.
func NewAgent(cfg *Config, bus *telemetry.Bus) *Agent {
	return &Agent{
		cfg: cfg,
		bus: bus,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Run starts the heartbeat loop. Blocks until ctx is done.
func (a *Agent) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.cfg.Heartbeat) * time.Second)
	defer ticker.Stop()

	// Initial heartbeat
	if err := a.Sync(ctx); err != nil {
		log.Printf("[edge] initial heartbeat: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.Sync(ctx); err != nil {
				log.Printf("[edge] heartbeat: %v", err)
			}
		}
	}
}
