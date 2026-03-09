package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

// APIKeyStore validates API keys against the database.
type APIKeyStore interface {
	Validate(ctx context.Context, key string) (*Claims, error)
}

// APIKeyManager manages API keys (create, list, delete). Implemented by SQLAPIKeyStore when DB is configured.
type APIKeyManager interface {
	Create(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)
	List(ctx context.Context, tenantID string) ([]APIKeyInfo, error)
	Delete(ctx context.Context, id string, tenantID string) error
}

// CreateAPIKeyRequest is the request to create an API key.
type CreateAPIKeyRequest struct {
	Name     string
	Roles    string // comma-separated, e.g. "operator"
	TenantID string // required for operator role
}

// CreateAPIKeyResponse returns the created key. Key is the raw secret, shown only once.
type CreateAPIKeyResponse struct {
	ID       string
	Key      string
	Name     string
	Roles    string
	TenantID string
	CreatedAt time.Time
}

// APIKeyInfo is a non-sensitive view of an API key (no raw secret).
type APIKeyInfo struct {
	ID        string
	Name      string
	Roles     string
	TenantID  string
	CreatedAt time.Time
}

// SQLAPIKeyStore validates API keys using the api_keys table.
type SQLAPIKeyStore struct {
	db     *sql.DB
	driver string
}

// NewSQLAPIKeyStore creates an API key store backed by SQL.
func NewSQLAPIKeyStore(db *sql.DB, driver string) *SQLAPIKeyStore {
	if driver == "postgres" {
		driver = "pgx"
	}
	return &SQLAPIKeyStore{db: db, driver: driver}
}

func (s *SQLAPIKeyStore) ph(q string) string {
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

// Validate checks the API key and returns claims if valid.
func (s *SQLAPIKeyStore) Validate(ctx context.Context, key string) (*Claims, error) {
	if key == "" {
		return nil, nil
	}
	hash := sha256.Sum256([]byte(key))
	hashHex := hex.EncodeToString(hash[:])

	var name, roles string
	var tenantID sql.NullString
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT name, roles, tenant_id, expires_at FROM api_keys WHERE key_hash = ?"),
		hashHex,
	).Scan(&name, &roles, &tenantID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return nil, nil
	}

	roleList := strings.Split(roles, ",")
	for i, r := range roleList {
		roleList[i] = strings.TrimSpace(r)
	}

	c := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "apikey:" + name,
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Roles: roleList,
	}
	if tenantID.Valid && tenantID.String != "" {
		c.TenantID = tenantID.String
	}
	return c, nil
}

// Create generates a new API key and stores its hash. The raw key is returned only once.
func (s *SQLAPIKeyStore) Create(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if req.Name == "" || req.Roles == "" {
		return nil, errors.New("name and roles are required")
	}
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	rawKey := "sk-" + hex.EncodeToString(mustReadRandom(16))
	hash := sha256.Sum256([]byte(rawKey))
	hashHex := hex.EncodeToString(hash[:])
	id := "key-" + uuid.New().String()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		s.ph("INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"),
		id, hashHex, req.Name, req.Roles, tenantID, now)
	if err != nil {
		return nil, err
	}
	return &CreateAPIKeyResponse{
		ID:        id,
		Key:       rawKey,
		Name:      req.Name,
		Roles:     req.Roles,
		TenantID:  tenantID,
		CreatedAt: now,
	}, nil
}

// List returns API keys. If tenantID is empty (admin), returns all; otherwise filters by tenant.
func (s *SQLAPIKeyStore) List(ctx context.Context, tenantID string) ([]APIKeyInfo, error) {
	var rows *sql.Rows
	var err error
	if tenantID == "" {
		rows, err = s.db.QueryContext(ctx,
			s.ph("SELECT id, name, roles, tenant_id, created_at FROM api_keys ORDER BY created_at DESC"))
	} else {
		rows, err = s.db.QueryContext(ctx,
			s.ph("SELECT id, name, roles, tenant_id, created_at FROM api_keys WHERE tenant_id = ? ORDER BY created_at DESC"),
			tenantID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyInfo
	for rows.Next() {
		var info APIKeyInfo
		if err := rows.Scan(&info.ID, &info.Name, &info.Roles, &info.TenantID, &info.CreatedAt); err != nil {
			continue
		}
		out = append(out, info)
	}
	return out, rows.Err()
}

// Delete removes an API key by id. If tenantID is set (operator), only deletes if key belongs to that tenant.
func (s *SQLAPIKeyStore) Delete(ctx context.Context, id string, tenantID string) error {
	var res sql.Result
	var err error
	if tenantID == "" {
		res, err = s.db.ExecContext(ctx, s.ph("DELETE FROM api_keys WHERE id = ?"), id)
	} else {
		res, err = s.db.ExecContext(ctx, s.ph("DELETE FROM api_keys WHERE id = ? AND tenant_id = ?"), id, tenantID)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

func mustReadRandom(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

