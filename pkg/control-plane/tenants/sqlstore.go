package tenants

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
)

// SQLStore is a persistent tenant store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed tenant store.
// db: existing connection (migrations must already be run, e.g. by registry).
// driver: "sqlite" or "pgx" (PostgreSQL).
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

// List returns all tenants.
func (s *SQLStore) List() ([]Tenant, error) {
	rows, err := s.db.Query("SELECT id, name, config FROM tenants ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tenant
	for rows.Next() {
		var t Tenant
		var config sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &config); err != nil {
			continue
		}
		if config.Valid && config.String != "" {
			t.Config = json.RawMessage(config.String)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Get returns a tenant by ID.
func (s *SQLStore) Get(id string) (*Tenant, error) {
	var t Tenant
	var config sql.NullString
	err := s.db.QueryRow(s.ph("SELECT id, name, config FROM tenants WHERE id=?"), id).
		Scan(&t.ID, &t.Name, &config)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if config.Valid && config.String != "" {
		t.Config = json.RawMessage(config.String)
	}
	return &t, nil
}

// Create adds a new tenant.
func (s *SQLStore) Create(t *Tenant) error {
	config := ""
	if len(t.Config) > 0 {
		config = string(t.Config)
	}
	_, err := s.db.Exec(s.ph("INSERT INTO tenants (id, name, config) VALUES (?, ?, ?)"),
		t.ID, t.Name, nullIfEmpty(config))
	return err
}

// Update updates an existing tenant.
func (s *SQLStore) Update(t *Tenant) error {
	config := ""
	if len(t.Config) > 0 {
		config = string(t.Config)
	}
	res, err := s.db.Exec(s.ph("UPDATE tenants SET name=?, config=? WHERE id=?"),
		t.Name, nullIfEmpty(config), t.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a tenant by ID.
func (s *SQLStore) Delete(id string) error {
	res, err := s.db.Exec(s.ph("DELETE FROM tenants WHERE id=?"), id)
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
