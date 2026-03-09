package sdk

import (
	"context"
	"encoding/json"
)

// ListRobots returns all robots, optionally filtered by tenant.
func (c *Client) ListRobots(ctx context.Context, tenantID string) ([]Robot, error) {
	var out []Robot
	err := c.doJSON(ctx, "GET", "/robots", nil, tenantID, &out)
	return out, err
}

// GetRobot returns a robot by ID.
func (c *Client) GetRobot(ctx context.Context, id string) (*Robot, error) {
	var out Robot
	err := c.doJSON(ctx, "GET", "/robots/"+id, nil, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SendCommandRequest is the request body for sending a command.
type SendCommandRequest struct {
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// SendCommand sends a command to a robot (e.g. safe_stop).
func (c *Client) SendCommand(ctx context.Context, robotID string, req SendCommandRequest) error {
	return c.doJSON(ctx, "POST", "/robots/"+robotID+"/command", req, "", nil)
}
