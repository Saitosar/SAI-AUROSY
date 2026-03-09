package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// APIKeyStore validates API keys against the database.
type APIKeyStore interface {
	Validate(ctx context.Context, key string) (*Claims, error)
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
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		s.ph("SELECT name, roles, expires_at FROM api_keys WHERE key_hash = ?"),
		hashHex,
	).Scan(&name, &roles, &expiresAt)
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

	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "apikey:" + name,
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Roles: roleList,
	}, nil
}

