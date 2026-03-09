package sdk

import (
	"context"
)

// ListWebhooks returns all webhooks (administrator only).
func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var out []Webhook
	err := c.doJSON(ctx, "GET", "/webhooks", nil, "", &out)
	return out, err
}

// GetWebhook returns a webhook by ID (administrator only).
func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	var out Webhook
	err := c.doJSON(ctx, "GET", "/webhooks/"+id, nil, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateWebhook creates a new webhook (administrator only).
func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*Webhook, error) {
	var out Webhook
	err := c.doJSON(ctx, "POST", "/webhooks", req, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateWebhook updates a webhook (administrator only).
func (c *Client) UpdateWebhook(ctx context.Context, id string, req UpdateWebhookRequest) (*Webhook, error) {
	var out Webhook
	err := c.doJSON(ctx, "PUT", "/webhooks/"+id, req, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWebhook deletes a webhook (administrator only).
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.doJSON(ctx, "DELETE", "/webhooks/"+id, nil, "", nil)
}
