package oauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLClientStore is a SQL-backed OAuth client store.
type SQLClientStore struct {
	db     *sql.DB
	driver string
}

// NewSQLClientStore creates a new SQL client store.
func NewSQLClientStore(db *sql.DB, driver string) *SQLClientStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLClientStore{db: db, driver: driver}
}

func (s *SQLClientStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	n := 1
	var b strings.Builder
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

// GetByClientID returns a client by client_id.
func (s *SQLClientStore) GetByClientID(ctx context.Context, clientID string) (*Client, error) {
	var c Client
	var redirectURIs, scopes string
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT id, client_id, client_secret_hash, redirect_uris, scopes, tenant_id, created_at, updated_at FROM oauth_clients WHERE client_id = ?"),
		clientID,
	).Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &redirectURIs, &scopes, &c.TenantID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.RedirectURIs = strings.Split(redirectURIs, ",")
	c.Scopes = strings.Split(scopes, " ")
	for i, u := range c.RedirectURIs {
		c.RedirectURIs[i] = strings.TrimSpace(u)
	}
	for i, sc := range c.Scopes {
		c.Scopes[i] = strings.TrimSpace(sc)
	}
	return &c, nil
}

// List returns OAuth clients. If tenantID is empty, returns all; otherwise filters by tenant.
func (s *SQLClientStore) List(ctx context.Context, tenantID string) ([]*Client, error) {
	var rows *sql.Rows
	var err error
	if tenantID == "" {
		rows, err = s.db.QueryContext(ctx,
			s.ph("SELECT id, client_id, redirect_uris, scopes, tenant_id, created_at, updated_at FROM oauth_clients ORDER BY created_at DESC"))
	} else {
		rows, err = s.db.QueryContext(ctx,
			s.ph("SELECT id, client_id, redirect_uris, scopes, tenant_id, created_at, updated_at FROM oauth_clients WHERE tenant_id = ? ORDER BY created_at DESC"),
			tenantID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Client
	for rows.Next() {
		var c Client
		var redirectURIs, scopes string
		if err := rows.Scan(&c.ID, &c.ClientID, &redirectURIs, &scopes, &c.TenantID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		c.RedirectURIs = strings.Split(redirectURIs, ",")
		c.Scopes = strings.Split(scopes, " ")
		for i, u := range c.RedirectURIs {
			c.RedirectURIs[i] = strings.TrimSpace(u)
		}
		for i, sc := range c.Scopes {
			c.Scopes[i] = strings.TrimSpace(sc)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// Create creates a new OAuth client.
func (s *SQLClientStore) Create(ctx context.Context, c *Client) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	redirectURIs := strings.Join(c.RedirectURIs, ",")
	scopes := strings.Join(c.Scopes, " ")
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO oauth_clients (id, client_id, client_secret_hash, redirect_uris, scopes, tenant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		c.ID, c.ClientID, c.ClientSecretHash, redirectURIs, scopes, c.TenantID, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// Update updates an OAuth client's redirect_uris and scopes.
func (s *SQLClientStore) Update(ctx context.Context, c *Client) error {
	redirectURIs := strings.Join(c.RedirectURIs, ",")
	scopes := strings.Join(c.Scopes, " ")
	_, err := s.db.ExecContext(ctx,
		s.ph("UPDATE oauth_clients SET redirect_uris = ?, scopes = ?, tenant_id = ?, updated_at = ? WHERE client_id = ?"),
		redirectURIs, scopes, c.TenantID, time.Now(), c.ClientID,
	)
	return err
}

// Delete removes an OAuth client by client_id.
func (s *SQLClientStore) Delete(ctx context.Context, clientID string) error {
	_, err := s.db.ExecContext(ctx, s.ph("DELETE FROM oauth_clients WHERE client_id = ?"), clientID)
	return err
}

// SQLCodeStore is a SQL-backed authorization code store.
type SQLCodeStore struct {
	db     *sql.DB
	driver string
}

// NewSQLCodeStore creates a new SQL code store.
func NewSQLCodeStore(db *sql.DB, driver string) *SQLCodeStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLCodeStore{db: db, driver: driver}
}

func (s *SQLCodeStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	n := 1
	var b strings.Builder
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

// Create stores an authorization code.
func (s *SQLCodeStore) Create(ctx context.Context, code *AuthCode) error {
	if code.ID == "" {
		code.ID = uuid.New().String()
	}
	code.CreatedAt = time.Now()
	scopes := strings.Join(code.Scopes, " ")
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO oauth_codes (id, code_hash, client_id, tenant_id, scopes, redirect_uri, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		code.ID, code.CodeHash, code.ClientID, code.TenantID, scopes, code.RedirectURI, code.ExpiresAt, code.CreatedAt,
	)
	return err
}

// GetByCodeHash returns a code by its hash.
func (s *SQLCodeStore) GetByCodeHash(ctx context.Context, codeHash string) (*AuthCode, error) {
	var code AuthCode
	var scopes string
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT id, code_hash, client_id, tenant_id, scopes, redirect_uri, expires_at, created_at FROM oauth_codes WHERE code_hash = ?"),
		codeHash,
	).Scan(&code.ID, &code.CodeHash, &code.ClientID, &code.TenantID, &scopes, &code.RedirectURI, &code.ExpiresAt, &code.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	code.Scopes = strings.Split(scopes, " ")
	for i, sc := range code.Scopes {
		code.Scopes[i] = strings.TrimSpace(sc)
	}
	return &code, nil
}

// Delete removes an authorization code.
func (s *SQLCodeStore) Delete(ctx context.Context, codeHash string) error {
	_, err := s.db.ExecContext(ctx, s.ph("DELETE FROM oauth_codes WHERE code_hash = ?"), codeHash)
	return err
}

// SQLTokenStore is a SQL-backed token store.
type SQLTokenStore struct {
	db     *sql.DB
	driver string
}

// NewSQLTokenStore creates a new SQL token store.
func NewSQLTokenStore(db *sql.DB, driver string) *SQLTokenStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLTokenStore{db: db, driver: driver}
}

func (s *SQLTokenStore) ph(q string) string {
	if s.driver != "pgx" {
		return q
	}
	n := 1
	var b strings.Builder
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

// Create stores a token.
func (s *SQLTokenStore) Create(ctx context.Context, t *Token) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	scopes := strings.Join(t.Scopes, " ")
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO oauth_tokens (id, access_token_hash, refresh_token_hash, client_id, tenant_id, scopes, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"),
		t.ID, t.AccessTokenHash, nullIfEmpty(t.RefreshTokenHash), t.ClientID, t.TenantID, scopes, t.ExpiresAt, t.CreatedAt,
	)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetByAccessTokenHash returns a token by access token hash.
func (s *SQLTokenStore) GetByAccessTokenHash(ctx context.Context, hash string) (*Token, error) {
	return s.getByHash(ctx, "access_token_hash", hash)
}

// GetByRefreshTokenHash returns a token by refresh token hash.
func (s *SQLTokenStore) GetByRefreshTokenHash(ctx context.Context, hash string) (*Token, error) {
	return s.getByHash(ctx, "refresh_token_hash", hash)
}

func (s *SQLTokenStore) getByHash(ctx context.Context, col, hash string) (*Token, error) {
	var t Token
	var scopes string
	var refreshHash sql.NullString
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT id, access_token_hash, refresh_token_hash, client_id, tenant_id, scopes, expires_at, created_at FROM oauth_tokens WHERE "+col+" = ?"),
		hash,
	).Scan(&t.ID, &t.AccessTokenHash, &refreshHash, &t.ClientID, &t.TenantID, &scopes, &t.ExpiresAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if refreshHash.Valid {
		t.RefreshTokenHash = refreshHash.String
	}
	t.Scopes = strings.Split(scopes, " ")
	for i, sc := range t.Scopes {
		t.Scopes[i] = strings.TrimSpace(sc)
	}
	return &t, nil
}

// DeleteByAccessToken removes a token by access token hash.
func (s *SQLTokenStore) DeleteByAccessToken(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, s.ph("DELETE FROM oauth_tokens WHERE access_token_hash = ?"), hash)
	return err
}

// HashSHA256 returns SHA256 hex of the input.
func HashSHA256(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
