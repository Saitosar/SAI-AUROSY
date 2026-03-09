# Telemetry Retention

## Overview

Telemetry samples (`telemetry_samples`) grow unbounded. This document describes the retention policy: TTL, aggregation, and background cleanup.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TELEMETRY_RETENTION_DAYS` | 30 | Delete raw samples older than N days |
| `TELEMETRY_AGGREGATE_BEFORE_DELETE` | true | Aggregate to hourly buckets before deleting |

## Behavior

1. **Aggregation**: Raw samples older than the retention cutoff are aggregated into `telemetry_aggregates` (hourly buckets: `online_count`, `error_count` per robot per hour).
2. **Deletion**: Raw samples older than the cutoff are deleted.
3. **Background job**: Runs every 6 hours.

## RobotSummary Fallback

When querying analytics for a time range, `RobotSummary` first reads from `telemetry_samples`. If the range has no raw data (e.g. beyond retention), it falls back to `telemetry_aggregates` for uptime and error counts.

## Schema

```sql
CREATE TABLE telemetry_aggregates (
    id TEXT PRIMARY KEY,
    robot_id TEXT NOT NULL,
    bucket_start TIMESTAMP NOT NULL,
    bucket_type TEXT NOT NULL,  -- 'hour'
    online_count INTEGER NOT NULL,
    error_count INTEGER NOT NULL,
    tenant_id TEXT
);
```

## Related

- [Phase 2.4 Enterprise Analytics](phase-2.4-enterprise-analytics.md)
- [Platform Architecture](../architecture/platform-architecture.md)
