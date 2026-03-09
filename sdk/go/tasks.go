package sdk

import (
	"context"
	"net/url"
)

// ListTasks returns tasks, optionally filtered by tenant, robot, or status.
func (c *Client) ListTasks(ctx context.Context, opts *ListTasksOptions) ([]Task, error) {
	path := "/tasks"
	if opts != nil {
		q := url.Values{}
		if opts.TenantID != "" {
			q.Set("tenant_id", opts.TenantID)
		}
		if opts.RobotID != "" {
			q.Set("robot_id", opts.RobotID)
		}
		if opts.Status != "" {
			q.Set("status", string(opts.Status))
		}
		if s := q.Encode(); s != "" {
			path += "?" + s
		}
	}
	var out []Task
	tenantID := ""
	if opts != nil {
		tenantID = opts.TenantID
	}
	err := c.doJSON(ctx, "GET", path, nil, tenantID, &out)
	return out, err
}

// ListTasksOptions are options for listing tasks.
type ListTasksOptions struct {
	TenantID string
	RobotID  string
	Status   TaskStatus
}

// GetTask returns a task by ID.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	var out Task
	err := c.doJSON(ctx, "GET", "/tasks/"+id, nil, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var out Task
	err := c.doJSON(ctx, "POST", "/tasks", req, "", &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelTask cancels a task.
func (c *Client) CancelTask(ctx context.Context, id string) error {
	return c.doJSON(ctx, "POST", "/tasks/"+id+"/cancel", nil, "", nil)
}
