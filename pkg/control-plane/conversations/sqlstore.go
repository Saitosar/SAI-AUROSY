package conversations

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// SQLStore is a persistent conversation store using SQLite or PostgreSQL.
type SQLStore struct {
	db     *sql.DB
	driver string
}

// NewSQLStore creates a new SQL-backed conversation store.
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

// List returns all conversations.
func (s *SQLStore) List(ctx context.Context) ([]Conversation, error) {
	return s.listWithTenant(ctx, "")
}

// ListByTenant returns conversations visible to the tenant.
func (s *SQLStore) ListByTenant(ctx context.Context, tenantID string) ([]Conversation, error) {
	return s.listWithTenant(ctx, tenantID)
}

func (s *SQLStore) listWithTenant(ctx context.Context, tenantID string) ([]Conversation, error) {
	q := "SELECT id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id FROM conversations"
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
	var out []Conversation
	for rows.Next() {
		var c Conversation
		var langsJSON string
		var desc, providerURL, tid sql.NullString
		if err := rows.Scan(&c.ID, &c.Intent, &c.Name, &desc, &c.ResponseTemplate, &providerURL, &langsJSON, &tid); err != nil {
			continue
		}
		if desc.Valid {
			c.Description = desc.String
		}
		if providerURL.Valid {
			c.ResponseProviderURL = providerURL.String
		}
		if tid.Valid {
			c.TenantID = tid.String
		}
		_ = json.Unmarshal([]byte(langsJSON), &c.SupportedLanguages)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns a conversation by ID.
func (s *SQLStore) Get(ctx context.Context, id string) (*Conversation, error) {
	return s.getByTenant(ctx, id, "")
}

// GetByTenant returns a conversation by ID if visible to the tenant.
func (s *SQLStore) GetByTenant(ctx context.Context, id, tenantID string) (*Conversation, error) {
	return s.getByTenant(ctx, id, tenantID)
}

func (s *SQLStore) getByTenant(ctx context.Context, id, tenantID string) (*Conversation, error) {
	var c Conversation
	var langsJSON string
	var desc, providerURL, tid sql.NullString
	var err error
	if tenantID != "" {
		err = s.db.QueryRowContext(ctx, s.ph("SELECT id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id FROM conversations WHERE id=? AND (tenant_id IS NULL OR tenant_id = ?)"), id, tenantID).
			Scan(&c.ID, &c.Intent, &c.Name, &desc, &c.ResponseTemplate, &providerURL, &langsJSON, &tid)
	} else {
		err = s.db.QueryRowContext(ctx, s.ph("SELECT id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id FROM conversations WHERE id=?"), id).
			Scan(&c.ID, &c.Intent, &c.Name, &desc, &c.ResponseTemplate, &providerURL, &langsJSON, &tid)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		c.Description = desc.String
	}
	if providerURL.Valid {
		c.ResponseProviderURL = providerURL.String
	}
	if tid.Valid {
		c.TenantID = tid.String
	}
	_ = json.Unmarshal([]byte(langsJSON), &c.SupportedLanguages)
	return &c, nil
}

// GetByIntent returns a conversation by intent. Tenant-specific takes precedence over shared.
func (s *SQLStore) GetByIntent(ctx context.Context, intent, tenantID string) (*Conversation, error) {
	// First try tenant-specific
	if tenantID != "" {
		var c Conversation
		var langsJSON string
		var desc, providerURL, tid sql.NullString
		err := s.db.QueryRowContext(ctx, s.ph("SELECT id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id FROM conversations WHERE intent=? AND tenant_id=?"), intent, tenantID).
			Scan(&c.ID, &c.Intent, &c.Name, &desc, &c.ResponseTemplate, &providerURL, &langsJSON, &tid)
		if err == nil {
			if desc.Valid {
				c.Description = desc.String
			}
			if providerURL.Valid {
				c.ResponseProviderURL = providerURL.String
			}
			if tid.Valid {
				c.TenantID = tid.String
			}
			_ = json.Unmarshal([]byte(langsJSON), &c.SupportedLanguages)
			return &c, nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
	}
	// Fallback to shared (tenant_id IS NULL)
	var c Conversation
	var langsJSON string
	var desc, providerURL, tid sql.NullString
	err := s.db.QueryRowContext(ctx, s.ph("SELECT id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id FROM conversations WHERE intent=? AND tenant_id IS NULL"), intent).
		Scan(&c.ID, &c.Intent, &c.Name, &desc, &c.ResponseTemplate, &providerURL, &langsJSON, &tid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		c.Description = desc.String
	}
	if providerURL.Valid {
		c.ResponseProviderURL = providerURL.String
	}
	if tid.Valid {
		c.TenantID = tid.String
	}
	_ = json.Unmarshal([]byte(langsJSON), &c.SupportedLanguages)
	return &c, nil
}

// Create adds a new conversation.
func (s *SQLStore) Create(ctx context.Context, c *Conversation) error {
	langsJSON, _ := json.Marshal(c.SupportedLanguages)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, s.ph("INSERT INTO conversations (id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"),
		c.ID, c.Intent, c.Name, nullIfEmpty(c.Description), c.ResponseTemplate, nullIfEmpty(c.ResponseProviderURL), string(langsJSON), nullIfEmpty(c.TenantID), now, now)
	return err
}

// Update updates an existing conversation.
func (s *SQLStore) Update(ctx context.Context, c *Conversation) error {
	langsJSON, _ := json.Marshal(c.SupportedLanguages)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, s.ph("UPDATE conversations SET intent=?, name=?, description=?, response_template=?, response_provider_url=?, supported_languages=?, tenant_id=?, updated_at=? WHERE id=?"),
		c.Intent, c.Name, nullIfEmpty(c.Description), c.ResponseTemplate, nullIfEmpty(c.ResponseProviderURL), string(langsJSON), nullIfEmpty(c.TenantID), now, c.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a conversation by ID.
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, s.ph("DELETE FROM conversations WHERE id=?"), id)
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
