package registry

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sai-aurosy/platform/pkg/hal"
	_ "modernc.org/sqlite"
)

// SQLStore is a persistent fleet registry store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed registry store.
// driver: "sqlite" or "pgx" (PostgreSQL); dsn: connection string.
func NewSQLStore(driver, dsn string) (*SQLStore, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	migrateDriver := driver
	if driver == "pgx" {
		migrateDriver = "postgres"
	}
	if err := Migrate(db, migrateDriver); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLStore{db: db, driver: driver}, nil
}

// NewSQLStoreFromDB creates a SQL store from an existing connection.
func NewSQLStoreFromDB(db *sql.DB, driver string) (*SQLStore, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	migrateDriver := driver
	if driver == "pgx" {
		migrateDriver = "postgres"
	}
	if err := Migrate(db, migrateDriver); err != nil {
		return nil, err
	}
	return &SQLStore{db: db, driver: driver}, nil
}

// DB returns the underlying database connection (for API key store, etc).
func (s *SQLStore) DB() *sql.DB {
	return s.db
}

// ph converts ? placeholders to $1,$2 for PostgreSQL.
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

// Close closes the database connection.
func (s *SQLStore) Close() error {
	return s.db.Close()
}

// Add adds or updates a robot.
func (s *SQLStore) Add(r *hal.Robot) {
	now := time.Now()
	r.UpdatedAt = now
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	capsJSON, _ := json.Marshal(r.Capabilities)
	if r.Capabilities == nil {
		capsJSON = []byte("[]")
	}
	edgeID := ""
	if r.EdgeID != "" {
		edgeID = r.EdgeID
	}
	if s.driver == "pgx" {
		s.db.Exec(s.ph("INSERT INTO robots (id, vendor, model, adapter_endpoint, tenant_id, edge_id, capabilities, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO UPDATE SET vendor=?, model=?, adapter_endpoint=?, tenant_id=?, edge_id=?, capabilities=?, updated_at=?"),
			r.ID, r.Vendor, r.Model, r.AdapterEndpoint, r.TenantID, edgeID, string(capsJSON), r.CreatedAt, r.UpdatedAt,
			r.Vendor, r.Model, r.AdapterEndpoint, r.TenantID, edgeID, string(capsJSON), r.UpdatedAt,
		)
	} else {
		s.db.Exec("INSERT OR REPLACE INTO robots (id, vendor, model, adapter_endpoint, tenant_id, edge_id, capabilities, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			r.ID, r.Vendor, r.Model, r.AdapterEndpoint, r.TenantID, edgeID, string(capsJSON), r.CreatedAt, r.UpdatedAt,
		)
	}
}

// Get returns a robot by ID.
func (s *SQLStore) Get(id string) *hal.Robot {
	var r hal.Robot
	var capsJSON sql.NullString
	err := s.db.QueryRow(
		s.ph("SELECT id, vendor, model, adapter_endpoint, tenant_id, COALESCE(edge_id,''), capabilities, created_at, updated_at FROM robots WHERE id=?"),
		id,
	).Scan(&r.ID, &r.Vendor, &r.Model, &r.AdapterEndpoint, &r.TenantID, &r.EdgeID, &capsJSON, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}
	if capsJSON.Valid && capsJSON.String != "" {
		_ = json.Unmarshal([]byte(capsJSON.String), &r.Capabilities)
	}
	return &r
}


// List returns all robots.
func (s *SQLStore) List() []hal.Robot {
	rows, err := s.db.Query("SELECT id, vendor, model, adapter_endpoint, tenant_id, COALESCE(edge_id,''), capabilities, created_at, updated_at FROM robots ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []hal.Robot
	for rows.Next() {
		var r hal.Robot
		var capsJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.Vendor, &r.Model, &r.AdapterEndpoint, &r.TenantID, &r.EdgeID, &capsJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		if capsJSON.Valid && capsJSON.String != "" {
			_ = json.Unmarshal([]byte(capsJSON.String), &r.Capabilities)
		}
		out = append(out, r)
	}
	return out
}

// Delete removes a robot.
func (s *SQLStore) Delete(id string) bool {
	res, err := s.db.Exec(s.ph("DELETE FROM robots WHERE id=?"), id)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}
