package edge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// HeartbeatRequest is sent to the cloud.
type HeartbeatRequest struct {
	EdgeID   string   `json:"edge_id"`
	Timestamp string  `json:"timestamp"`
	Robots   []string `json:"robots"`
}

// HeartbeatResponse is received from the cloud.
type HeartbeatResponse struct {
	PendingCommands []hal.Command `json:"pending_commands"`
}

// Sync performs heartbeat to cloud and relays pending commands to local NATS.
func (a *Agent) Sync(ctx context.Context) error {
	req := HeartbeatRequest{
		EdgeID:   a.cfg.EdgeID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Robots:   a.cfg.RobotIDs,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := a.cfg.CloudURL + "/v1/edges/" + a.cfg.EdgeID + "/heartbeat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.cfg.APIKey != "" {
		httpReq.Header.Set("X-API-Key", a.cfg.APIKey)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed: %d %s", resp.StatusCode, string(b))
	}

	var hbResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return err
	}

	for _, cmd := range hbResp.PendingCommands {
		if !SafetyAllow(&cmd) {
			continue
		}
		if err := a.bus.PublishCommand(&cmd); err != nil {
			// Log but continue
			continue
		}
	}

	return nil
}
