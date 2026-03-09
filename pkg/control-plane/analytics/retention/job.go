package retention

import (
	"context"
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"

)

// Config holds retention job configuration.
type Config struct {
	RetentionDays           int  // Delete raw samples older than this
	AggregateBeforeDelete   bool // If true, aggregate to hourly buckets before deleting
	AggregationIntervalHour int  // Aggregate into N-hour buckets (default 1)
}

// DefaultConfig returns default retention config.
func DefaultConfig() Config {
	return Config{
		RetentionDays:           30,
		AggregateBeforeDelete:   true,
		AggregationIntervalHour: 1,
	}
}

// Run executes the retention job: aggregate old data, then delete raw samples beyond TTL.
func Run(ctx context.Context, db *sql.DB, driver string, cfg Config) error {
	if driver == "postgres" {
		driver = "pgx"
	}
	cutoff := time.Now().AddDate(0, 0, -cfg.RetentionDays)

	if cfg.AggregateBeforeDelete {
		if err := aggregateHours(ctx, db, driver, cutoff); err != nil {
			return err
		}
	}

	return deleteOldSamples(ctx, db, driver, cutoff)
}

func ph(q string, driver string) string {
	if driver != "pgx" {
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

// aggregateHours aggregates telemetry_samples into hourly buckets for data before cutoff.
func aggregateHours(ctx context.Context, db *sql.DB, driver string, cutoff time.Time) error {
	if driver == "pgx" {
		ins := `
		INSERT INTO telemetry_aggregates (id, robot_id, bucket_start, bucket_type, online_count, error_count, tenant_id)
		SELECT gen_random_uuid()::text, robot_id, date_trunc('hour', timestamp)::timestamp, 'hour',
			SUM(CASE WHEN online = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN actuator_status = 'error' THEN 1 ELSE 0 END),
			NULL
		FROM telemetry_samples
		WHERE timestamp < $1
		GROUP BY robot_id, date_trunc('hour', timestamp)
		ON CONFLICT (robot_id, bucket_start, bucket_type) DO NOTHING
		`
		_, err := db.ExecContext(ctx, ins, cutoff)
		return err
	}
	// SQLite: INSERT OR IGNORE to skip already-aggregated buckets
	ins := `
		INSERT OR IGNORE INTO telemetry_aggregates (id, robot_id, bucket_start, bucket_type, online_count, error_count, tenant_id)
		SELECT lower(hex(randomblob(16))), robot_id, strftime('%Y-%m-%d %H:00:00', timestamp), 'hour',
			SUM(CASE WHEN online = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN actuator_status = 'error' THEN 1 ELSE 0 END),
			NULL
		FROM telemetry_samples
		WHERE timestamp < ?
		GROUP BY robot_id, strftime('%Y-%m-%d %H:00:00', timestamp)
	`
	_, err := db.ExecContext(ctx, ins, cutoff)
	return err
}

func deleteOldSamples(ctx context.Context, db *sql.DB, driver string, cutoff time.Time) error {
	q := "DELETE FROM telemetry_samples WHERE timestamp < ?"
	if driver == "pgx" {
		q = "DELETE FROM telemetry_samples WHERE timestamp < $1"
	}
	res, err := db.ExecContext(ctx, q, cutoff)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("[retention] deleted %d telemetry samples older than %s", n, cutoff.Format(time.RFC3339))
	}
	return nil
}
