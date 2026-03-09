package webhooks

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLStore is a persistent webhook store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed webhook store.
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

// Create adds a new webhook.
func (s *SQLStore) Create(ctx context.Context, w *Webhook) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	events := strings.Join(w.Events, ",")
	enabled := 0
	if w.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO webhooks (id, url, events, secret, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"),
		w.ID, w.URL, events, nullIfEmpty(w.Secret), enabled, w.CreatedAt, w.UpdatedAt)
	return err
}

// Get returns a webhook by ID.
func (s *SQLStore) Get(ctx context.Context, id string) (*Webhook, error) {
	var w Webhook
	var events, secret sql.NullString
	var enabled int
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT id, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE id = ?"),
		id).Scan(&w.ID, &w.URL, &events, &secret, &enabled, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if events.Valid {
		w.Events = splitEvents(events.String)
	}
	if secret.Valid {
		w.Secret = secret.String
	}
	w.Enabled = enabled == 1
	return &w, nil
}

// List returns all webhooks.
func (s *SQLStore) List(ctx context.Context) ([]*Webhook, error) {
	rows, err := s.db.QueryContext(ctx,
		s.ph("SELECT id, url, events, secret, enabled, created_at, updated_at FROM webhooks ORDER BY created_at DESC"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanWebhooks(rows)
}

// ListByEvent returns webhooks subscribed to the given event.
func (s *SQLStore) ListByEvent(ctx context.Context, event string) ([]*Webhook, error) {
	rows, err := s.db.QueryContext(ctx,
		s.ph("SELECT id, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE enabled = 1 AND (events LIKE ? OR events = ?)"),
		"%"+event+"%", event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanWebhooks(rows)
}

// Update updates a webhook.
func (s *SQLStore) Update(ctx context.Context, w *Webhook) error {
	w.UpdatedAt = time.Now()
	events := strings.Join(w.Events, ",")
	enabled := 0
	if w.Enabled {
		enabled = 1
	}
	res, err := s.db.ExecContext(ctx,
		s.ph("UPDATE webhooks SET url = ?, events = ?, secret = ?, enabled = ?, updated_at = ? WHERE id = ?"),
		w.URL, events, nullIfEmpty(w.Secret), enabled, w.UpdatedAt, w.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a webhook.
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, s.ph("DELETE FROM webhooks WHERE id = ?"), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLStore) scanWebhooks(rows *sql.Rows) ([]*Webhook, error) {
	var list []*Webhook
	for rows.Next() {
		var w Webhook
		var events, secret sql.NullString
		var enabled int
		if err := rows.Scan(&w.ID, &w.URL, &events, &secret, &enabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		if events.Valid {
			w.Events = splitEvents(events.String)
		}
		if secret.Valid {
			w.Secret = secret.String
		}
		w.Enabled = enabled == 1
		list = append(list, &w)
	}
	return list, rows.Err()
}

func splitEvents(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
