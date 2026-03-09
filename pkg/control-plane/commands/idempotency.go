package commands

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Store is the interface for command idempotency.
type Store interface {
	// Reserve atomically claims the idempotency key. Returns true if reserved (first use), false if key already exists.
	Reserve(ctx context.Context, key, robotID string) (reserved bool, err error)
	// Cleanup removes keys older than the given duration.
	Cleanup(ctx context.Context, olderThan time.Duration) error
}

// SQLStore is a persistent idempotency store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed idempotency store.
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

// Reserve atomically claims the idempotency key.
func (s *SQLStore) Reserve(ctx context.Context, key, robotID string) (bool, error) {
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO command_idempotency (idempotency_key, robot_id, created_at) VALUES (?, ?, ?)"),
		key, robotID, time.Now())
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Cleanup removes keys older than the given duration.
func (s *SQLStore) Cleanup(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.ExecContext(ctx,
		s.ph("DELETE FROM command_idempotency WHERE created_at < ?"),
		cutoff)
	return err
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // PostgreSQL unique_violation
	}
	return strings.Contains(err.Error(), "UNIQUE constraint") ||
		strings.Contains(err.Error(), "unique constraint")
}

// MemoryStore is an in-memory idempotency store for development.
type MemoryStore struct {
	mu   sync.RWMutex
	keys map[string]memoryEntry
}

type memoryEntry struct {
	robotID   string
	createdAt time.Time
}

// NewMemoryStore creates an in-memory idempotency store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{keys: make(map[string]memoryEntry)}
}

// Reserve atomically claims the idempotency key.
func (m *MemoryStore) Reserve(ctx context.Context, key, robotID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.keys[key]; ok {
		return false, nil
	}
	m.keys[key] = memoryEntry{robotID: robotID, createdAt: time.Now()}
	return true, nil
}

// Cleanup removes keys older than the given duration.
func (m *MemoryStore) Cleanup(ctx context.Context, olderThan time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	for k, v := range m.keys {
		if v.createdAt.Before(cutoff) {
			delete(m.keys, k)
		}
	}
	return nil
}

