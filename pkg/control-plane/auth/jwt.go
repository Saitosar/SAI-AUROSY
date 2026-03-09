package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sai-aurosy/platform/pkg/secrets"
)

var errInvalidClaims = errors.New("invalid claims")

type contextKey string

const (
	ContextKeyClaims contextKey = "jwt_claims"
)

// Claims holds JWT claims for authorization.
// TenantID is populated from the "tenant_id" claim in the JWT payload when present,
// restricting the operator's scope to that tenant.
type Claims struct {
	jwt.RegisteredClaims
	Roles    []string `json:"roles,omitempty"`
	Role     string   `json:"role,omitempty"`
	TenantID string   `json:"tenant_id,omitempty"`
}

// GetRoles returns roles from claims (supports both "roles" array and "role" string).
func (c *Claims) GetRoles() []string {
	if len(c.Roles) > 0 {
		return c.Roles
	}
	if c.Role != "" {
		return []string{c.Role}
	}
	return nil
}

// Middleware returns a middleware that validates JWT or API key and sets claims in context.
// If JWT_SECRET and JWT_PUBLIC_KEY are both unset and apiKeyStore is nil, auth is skipped.
func Middleware(next http.Handler) http.Handler {
	return MiddlewareWithAPIKeys(next, nil)
}

// OAuthTokenValidator validates OAuth access tokens.
type OAuthTokenValidator interface {
	ValidateAccessToken(ctx context.Context, accessToken string) (*Claims, error)
}

// MiddlewareWithAPIKeys returns a middleware that validates JWT or API key.
func MiddlewareWithAPIKeys(next http.Handler, apiKeyStore APIKeyStore) http.Handler {
	return MiddlewareWithAPIKeysAndOAuth(next, apiKeyStore, nil)
}

// AuthConfigured returns true if any auth mechanism is configured.
func AuthConfigured(apiKeyStore APIKeyStore, oauthValidator OAuthTokenValidator) bool {
	ctx := context.Background()
	p := secrets.Default(ctx)
	return secrets.GetSecretOrEnv(ctx, p, "JWT_SECRET") != "" ||
		secrets.GetSecretOrEnv(ctx, p, "JWT_PUBLIC_KEY") != "" ||
		apiKeyStore != nil ||
		oauthValidator != nil
}

// MiddlewareWithAPIKeysAndOAuth returns a middleware that validates JWT, API key, or OAuth token.
// When no auth is configured: if ALLOW_UNSAFE_NO_AUTH=true, passes through; otherwise returns 401.
func MiddlewareWithAPIKeysAndOAuth(next http.Handler, apiKeyStore APIKeyStore, oauthValidator OAuthTokenValidator) http.Handler {
	ctx := context.Background()
	p := secrets.Default(ctx)
	secret := secrets.GetSecretOrEnv(ctx, p, "JWT_SECRET")
	publicKeyPEM := secrets.GetSecretOrEnv(ctx, p, "JWT_PUBLIC_KEY")
	issuer := secrets.GetSecretOrEnv(ctx, p, "JWT_ISSUER")
	audience := secrets.GetSecretOrEnv(ctx, p, "JWT_AUDIENCE")
	allowUnsafeNoAuth := os.Getenv("ALLOW_UNSAFE_NO_AUTH") == "true"

	authConfigured := secret != "" || publicKeyPEM != "" || apiKeyStore != nil || oauthValidator != nil
	if !authConfigured {
		if allowUnsafeNoAuth {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" && r.Method == "GET" {
			apiKey = r.URL.Query().Get("api_key")
		}

		var claims *Claims

		if strings.HasPrefix(auth, "Bearer ") {
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			var err error
			if secret != "" || publicKeyPEM != "" {
				claims, err = parseJWT(tokenStr, secret, publicKeyPEM, issuer, audience)
			}
			if (err != nil || claims == nil) && oauthValidator != nil {
				claims, err = oauthValidator.ValidateAccessToken(r.Context(), tokenStr)
			}
			if err != nil || claims == nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
		} else if apiKey != "" && apiKeyStore != nil {
			var err error
			claims, err = apiKeyStore.Validate(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if claims == nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseJWT(tokenStr, secret, publicKeyPEM, issuer, audience string) (*Claims, error) {
	var claims Claims
	var token *jwt.Token
	var err error

	if publicKeyPEM != "" {
		token, err = jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
		})
	} else {
		token, err = jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
	}

	if err != nil || !token.Valid {
		return nil, err
	}

	if issuer != "" && claims.Issuer != issuer {
		return nil, errInvalidClaims
	}
	if audience != "" {
		found := false
		for _, a := range claims.Audience {
			if a == audience {
				found = true
				break
			}
		}
		if !found {
			return nil, errInvalidClaims
		}
	}

	return &claims, nil
}

// GetClaims extracts claims from request context.
func GetClaims(ctx context.Context) *Claims {
	c, _ := ctx.Value(ContextKeyClaims).(*Claims)
	return c
}

// Enabled returns true if JWT auth is configured.
func Enabled() bool {
	ctx := context.Background()
	p := secrets.Default(ctx)
	return secrets.GetSecretOrEnv(ctx, p, "JWT_SECRET") != "" ||
		secrets.GetSecretOrEnv(ctx, p, "JWT_PUBLIC_KEY") != ""
}
