package audit

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLStore is a persistent audit store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed audit store.
// driver: "sqlite" or "pgx" (PostgreSQL); db: existing connection (migrations must already be run).
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

// Append adds an audit entry.
func (s *SQLStore) Append(ctx context.Context, e *Entry) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO audit_log (id, actor, action, resource, resource_id, timestamp, details, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		e.ID, e.Actor, e.Action, e.Resource, nullIfEmpty(e.ResourceID), e.Timestamp, nullIfEmpty(e.Details), nullIfEmpty(e.TenantID))
	return err
}

// List returns audit entries matching the filters.
func (s *SQLStore) List(ctx context.Context, f ListFilters) ([]*Entry, error) {
	var args []interface{}
	var conds []string

	if f.RobotID != "" {
		conds = append(conds, "resource = 'robot' AND resource_id = ?")
		args = append(args, f.RobotID)
	}
	if f.Actor != "" {
		conds = append(conds, "actor = ?")
		args = append(args, f.Actor)
	}
	if f.Action != "" {
		conds = append(conds, "action = ?")
		args = append(args, f.Action)
	}
	if f.From != nil {
		conds = append(conds, "timestamp >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conds = append(conds, "timestamp <= ?")
		args = append(args, *f.To)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	limit := 100
	if f.Limit > 0 {
		limit = f.Limit
	}
	offset := 0
	if f.Offset > 0 {
		offset = f.Offset
	}

	args = append(args, limit, offset)
	q := "SELECT id, actor, action, resource, resource_id, timestamp, details, tenant_id FROM audit_log" + where + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	rows, err := s.db.QueryContext(ctx, s.ph(q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Entry
	for rows.Next() {
		var e Entry
		var resourceID, details, tenantID sql.NullString
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &resourceID, &e.Timestamp, &details, &tenantID); err != nil {
			return nil, err
		}
		if resourceID.Valid {
			e.ResourceID = resourceID.String
		}
		if details.Valid {
			e.Details = details.String
		}
		if tenantID.Valid {
			e.TenantID = tenantID.String
		}
		list = append(list, &e)
	}
	return list, rows.Err()
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
