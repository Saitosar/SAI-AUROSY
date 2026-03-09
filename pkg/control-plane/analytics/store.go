package analytics

import (
	"context"
	"time"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// Store is the interface for telemetry storage and analytics queries.
type Store interface {
	WriteTelemetry(ctx context.Context, t *hal.Telemetry) error
	RobotSummary(ctx context.Context, robotID string, from, to time.Time) (*RobotSummary, error)
}

// RobotSummary contains aggregated analytics for a robot over a time range.
type RobotSummary struct {
	RobotID        string  `json:"robot_id"`
	UptimeSec      float64 `json:"uptime_sec"`
	CommandsCount  int     `json:"commands_count"`
	ErrorsCount    int     `json:"errors_count"`
	TasksCompleted int     `json:"tasks_completed"`
	TasksFailed    int     `json:"tasks_failed"`
}
