package oauth

import (
	"context"
	"time"
)

// Client represents an OAuth client application.
type Client struct {
	ID             string
	ClientID       string
	ClientSecretHash string
	RedirectURIs   []string
	Scopes         []string
	TenantID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AuthCode represents an authorization code for the Authorization Code flow.
type AuthCode struct {
	ID         string
	CodeHash   string
	ClientID   string
	TenantID   string
	Scopes     []string
	RedirectURI string
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// Token represents an OAuth access/refresh token.
type Token struct {
	ID              string
	AccessTokenHash  string
	RefreshTokenHash string
	ClientID        string
	TenantID        string
	Scopes          []string
	ExpiresAt       time.Time
	CreatedAt       time.Time
}

// ClientStore manages OAuth clients.
type ClientStore interface {
	GetByClientID(ctx context.Context, clientID string) (*Client, error)
	List(ctx context.Context, tenantID string) ([]*Client, error)
	Create(ctx context.Context, c *Client) error
	Update(ctx context.Context, c *Client) error
	Delete(ctx context.Context, clientID string) error
}

// CodeStore manages authorization codes.
type CodeStore interface {
	Create(ctx context.Context, code *AuthCode) error
	GetByCodeHash(ctx context.Context, codeHash string) (*AuthCode, error)
	Delete(ctx context.Context, codeHash string) error
}

// TokenStore manages OAuth tokens.
type TokenStore interface {
	Create(ctx context.Context, t *Token) error
	GetByAccessTokenHash(ctx context.Context, hash string) (*Token, error)
	GetByRefreshTokenHash(ctx context.Context, hash string) (*Token, error)
	DeleteByAccessToken(ctx context.Context, hash string) error
}
