package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sai-aurosy/platform/pkg/hal"
)

// SQLStore is a persistent analytics store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed analytics store.
func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLStore{db: db, driver: driver}
}

func (s *SQLStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	var b strings.Builder
	n := 1
	for _, r := range q {
		if r == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// WriteTelemetry stores a telemetry sample.
func (s *SQLStore) WriteTelemetry(ctx context.Context, t *hal.Telemetry) error {
	id := uuid.New().String()
	online := 0
	if t.Online {
		online = 1
	}
	var imuJSON, jointStatesJSON string
	if t.IMU != nil {
		b, _ := json.Marshal(t.IMU)
		imuJSON = string(b)
	}
	if len(t.JointStates) > 0 {
		b, _ := json.Marshal(t.JointStates)
		jointStatesJSON = string(b)
	}
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO telemetry_samples (id, robot_id, timestamp, online, actuator_status, current_task, imu_json, joint_states_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		id, t.RobotID, t.Timestamp, online, nullIfEmpty(t.ActuatorStatus), nullIfEmpty(t.CurrentTask), nullIfEmpty(imuJSON), nullIfEmpty(jointStatesJSON))
	return err
}

// RobotSummary returns aggregated analytics for a robot.
func (s *SQLStore) RobotSummary(ctx context.Context, robotID string, from, to time.Time) (*RobotSummary, error) {
	sum := &RobotSummary{RobotID: robotID}

	// Uptime and errors: from telemetry_samples or telemetry_aggregates (for old data beyond retention)
	var onlineCount, errCount int
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT COUNT(*) FROM telemetry_samples WHERE robot_id = ? AND timestamp >= ? AND timestamp <= ? AND online = 1"),
		robotID, from, to).Scan(&onlineCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx,
		s.ph("SELECT COUNT(*) FROM telemetry_samples WHERE robot_id = ? AND timestamp >= ? AND timestamp <= ? AND actuator_status = 'error'"),
		robotID, from, to).Scan(&errCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	// Fallback to aggregates when range has no raw samples (data was downsampled)
	if onlineCount == 0 && errCount == 0 {
		var aggOnline, aggErr int64
		_ = s.db.QueryRowContext(ctx,
			s.ph("SELECT COALESCE(SUM(online_count), 0), COALESCE(SUM(error_count), 0) FROM telemetry_aggregates WHERE robot_id = ? AND bucket_start >= ? AND bucket_start <= ? AND bucket_type = 'hour'"),
			robotID, from, to).Scan(&aggOnline, &aggErr)
		onlineCount = int(aggOnline)
		errCount = int(aggErr)
	}
	sum.UptimeSec = float64(onlineCount) * 1.5
	sum.ErrorsCount = errCount

	// Commands: from audit_log
	var cmdCount int
	err = s.db.QueryRowContext(ctx,
		s.ph("SELECT COUNT(*) FROM audit_log WHERE resource = 'robot' AND resource_id = ? AND action = 'command' AND timestamp >= ? AND timestamp <= ?"),
		robotID, from, to).Scan(&cmdCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	sum.CommandsCount = cmdCount

	// Tasks completed/failed: from tasks table
	err = s.db.QueryRowContext(ctx,
		s.ph("SELECT COUNT(*) FROM tasks WHERE robot_id = ? AND status = 'completed' AND completed_at >= ? AND completed_at <= ?"),
		robotID, from, to).Scan(&sum.TasksCompleted)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx,
		s.ph("SELECT COUNT(*) FROM tasks WHERE robot_id = ? AND status = 'failed' AND completed_at >= ? AND completed_at <= ?"),
		robotID, from, to).Scan(&sum.TasksFailed)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return sum, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
