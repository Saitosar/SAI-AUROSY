package webhooks

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeadLetterStore persists failed webhook deliveries for replay or monitoring.
type DeadLetterStore interface {
	Record(ctx context.Context, webhookID, event string, payload []byte, err error) error
}

// SQLDeadLetterStore is a persistent dead-letter store.
type SQLDeadLetterStore struct {
	db     *sql.DB
	driver string
}

// NewSQLDeadLetterStore creates a new SQL-backed dead-letter store.
func NewSQLDeadLetterStore(db *sql.DB, driver string) *SQLDeadLetterStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLDeadLetterStore{db: db, driver: driver}
}

func (s *SQLDeadLetterStore) ph(q string) string {
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

// Record stores a failed delivery.
func (s *SQLDeadLetterStore) Record(ctx context.Context, webhookID, event string, payload []byte, err error) error {
	id := uuid.New().String()
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	payloadStr := string(payload)
	_, dbErr := s.db.ExecContext(ctx,
		s.ph("INSERT INTO webhook_delivery_failures (id, webhook_id, event, payload_json, error, created_at) VALUES (?, ?, ?, ?, ?, ?)"),
		id, webhookID, event, payloadStr, errStr, time.Now())
	return dbErr
}

// MemoryDeadLetterStore is an in-memory dead-letter store for development.
type MemoryDeadLetterStore struct {
	mu   sync.Mutex
	items []Failure
}

// Failure represents a failed webhook delivery.
type Failure struct {
	ID        string
	WebhookID string
	Event     string
	Payload   []byte
	Error     string
	CreatedAt time.Time
}

// NewMemoryDeadLetterStore creates an in-memory dead-letter store.
func NewMemoryDeadLetterStore() *MemoryDeadLetterStore {
	return &MemoryDeadLetterStore{}
}

// Record stores a failed delivery.
func (m *MemoryDeadLetterStore) Record(ctx context.Context, webhookID, event string, payload []byte, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.items = append(m.items, Failure{
		ID:        uuid.New().String(),
		WebhookID: webhookID,
		Event:     event,
		Payload:   payload,
		Error:     errStr,
		CreatedAt: time.Now(),
	})
	return nil
}
