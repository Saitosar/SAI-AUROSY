package scenarios

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// SQLStore is a persistent scenario store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed scenario store.
func NewSQLStore(db *sql.DB, driver string) *SQLStore {
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

// List returns all scenarios.
func (s *SQLStore) List(ctx context.Context) ([]Scenario, error) {
	return s.listWithTenant(ctx, "")
}

// ListByTenant returns scenarios visible to the tenant: shared (tenant_id IS NULL) or tenant-specific.
func (s *SQLStore) ListByTenant(ctx context.Context, tenantID string) ([]Scenario, error) {
	return s.listWithTenant(ctx, tenantID)
}

func (s *SQLStore) listWithTenant(ctx context.Context, tenantID string) ([]Scenario, error) {
	q := "SELECT id, name, description, steps, required_capabilities FROM scenarios"
	var args []interface{}
	if tenantID != "" {
		q += " WHERE tenant_id IS NULL OR tenant_id = ?"
		args = append(args, tenantID)
	}
	q += " ORDER BY id"
	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = s.db.QueryContext(ctx, s.ph(q), args...)
	} else {
		rows, err = s.db.QueryContext(ctx, q)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scenario
	for rows.Next() {
		var sc Scenario
		var stepsJSON, capsJSON string
		var desc sql.NullString
		if err := rows.Scan(&sc.ID, &sc.Name, &desc, &stepsJSON, &capsJSON); err != nil {
			continue
		}
		if desc.Valid {
			sc.Description = desc.String
		}
		_ = json.Unmarshal([]byte(stepsJSON), &sc.Steps)
		_ = json.Unmarshal([]byte(capsJSON), &sc.RequiredCapabilities)
		out = append(out, sc)
	}
	return out, rows.Err()
}

// Get returns a scenario by ID.
func (s *SQLStore) Get(ctx context.Context, id string) (*Scenario, error) {
	return s.getByTenant(ctx, id, "")
}

// GetByTenant returns a scenario by ID if it is shared or belongs to the tenant.
func (s *SQLStore) GetByTenant(ctx context.Context, id, tenantID string) (*Scenario, error) {
	return s.getByTenant(ctx, id, tenantID)
}

func (s *SQLStore) getByTenant(ctx context.Context, id, tenantID string) (*Scenario, error) {
	var sc Scenario
	var stepsJSON, capsJSON string
	var desc sql.NullString
	var err error
	if tenantID != "" {
		err = s.db.QueryRowContext(ctx, s.ph("SELECT id, name, description, steps, required_capabilities FROM scenarios WHERE id=? AND (tenant_id IS NULL OR tenant_id = ?)"), id, tenantID).
			Scan(&sc.ID, &sc.Name, &desc, &stepsJSON, &capsJSON)
	} else {
		err = s.db.QueryRowContext(ctx, s.ph("SELECT id, name, description, steps, required_capabilities FROM scenarios WHERE id=?"), id).
			Scan(&sc.ID, &sc.Name, &desc, &stepsJSON, &capsJSON)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		sc.Description = desc.String
	}
	_ = json.Unmarshal([]byte(stepsJSON), &sc.Steps)
	_ = json.Unmarshal([]byte(capsJSON), &sc.RequiredCapabilities)
	return &sc, nil
}

// Create adds a new scenario.
func (s *SQLStore) Create(ctx context.Context, sc *Scenario) error {
	stepsJSON, _ := json.Marshal(sc.Steps)
	capsJSON, _ := json.Marshal(sc.RequiredCapabilities)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, s.ph("INSERT INTO scenarios (id, name, description, steps, required_capabilities, tenant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		sc.ID, sc.Name, nullIfEmpty(sc.Description), string(stepsJSON), string(capsJSON), nil, now, now)
	return err
}

// Update updates an existing scenario.
func (s *SQLStore) Update(ctx context.Context, sc *Scenario) error {
	stepsJSON, _ := json.Marshal(sc.Steps)
	capsJSON, _ := json.Marshal(sc.RequiredCapabilities)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, s.ph("UPDATE scenarios SET name=?, description=?, steps=?, required_capabilities=?, updated_at=? WHERE id=?"),
		sc.Name, nullIfEmpty(sc.Description), string(stepsJSON), string(capsJSON), now, sc.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a scenario by ID.
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, s.ph("DELETE FROM scenarios WHERE id=?"), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
