package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sai-aurosy/platform/pkg/control-plane/auth"
)

// Default scopes for OAuth.
const (
	ScopeRobotsRead   = "robots:read"
	ScopeTasksWrite   = "tasks:write"
	ScopeWebhooksRead = "webhooks:read"
	ScopeAnalyticsRead = "analytics:read"
)

// Server handles OAuth 2.0 Authorization Code flow.
type Server struct {
	clientStore ClientStore
	codeStore   CodeStore
	tokenStore  TokenStore
	baseURL     string
	accessTTL   time.Duration
}

// NewServer creates a new OAuth server.
func NewServer(clientStore ClientStore, codeStore CodeStore, tokenStore TokenStore, baseURL string) *Server {
	return &Server{
		clientStore: clientStore,
		codeStore:   codeStore,
		tokenStore:  tokenStore,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		accessTTL:   2 * time.Hour,
	}
}

// HandleAuthorize handles GET /oauth/authorize (Authorization Code flow).
func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	if clientID == "" || redirectURI == "" || responseType != "code" {
		s.oauthError(w, "invalid_request", "client_id, redirect_uri, response_type=code required", 400)
		return
	}

	client, err := s.clientStore.GetByClientID(r.Context(), clientID)
	if err != nil || client == nil {
		s.oauthError(w, "invalid_client", "client not found", 401)
		return
	}

	validRedirect := false
	for _, u := range client.RedirectURIs {
		if u == redirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		s.oauthError(w, "invalid_request", "redirect_uri not allowed", 400)
		return
	}

	scopes := parseScopes(scope)
	if len(scopes) == 0 {
		scopes = client.Scopes
	}
	for _, sc := range scopes {
		if !containsScope(client.Scopes, sc) {
			s.oauthError(w, "invalid_scope", "scope not allowed", 400)
			return
		}
	}

	code := randomToken(32)
	codeHash := HashSHA256(code)
	authCode := &AuthCode{
		CodeHash:   codeHash,
		ClientID:   clientID,
		TenantID:   client.TenantID,
		Scopes:     scopes,
		RedirectURI: redirectURI,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}
	if err := s.codeStore.Create(r.Context(), authCode); err != nil {
		s.oauthError(w, "server_error", "failed to create code", 500)
		return
	}

	redirect, _ := url.Parse(redirectURI)
	q := redirect.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

// HandleToken handles POST /oauth/token (token exchange).
func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.oauthError(w, "invalid_request", "POST required", 405)
		return
	}

	grantType := r.FormValue("grant_type")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	refreshToken := r.FormValue("refresh_token")

	if clientID == "" || clientSecret == "" {
		s.oauthError(w, "invalid_client", "client_id and client_secret required", 401)
		return
	}

	client, err := s.clientStore.GetByClientID(r.Context(), clientID)
	if err != nil || client == nil {
		s.oauthError(w, "invalid_client", "client not found", 401)
		return
	}
	if HashSHA256(clientSecret) != client.ClientSecretHash {
		s.oauthError(w, "invalid_client", "invalid client_secret", 401)
		return
	}

	var tenantID string
	var scopes []string

	switch grantType {
	case "authorization_code":
		if code == "" || redirectURI == "" {
			s.oauthError(w, "invalid_request", "code and redirect_uri required", 400)
			return
		}
		codeHash := HashSHA256(code)
		authCode, err := s.codeStore.GetByCodeHash(r.Context(), codeHash)
		if err != nil || authCode == nil {
			s.oauthError(w, "invalid_grant", "invalid or expired code", 400)
			return
		}
		if authCode.ClientID != clientID {
			s.oauthError(w, "invalid_grant", "code client mismatch", 400)
			return
		}
		if authCode.RedirectURI != redirectURI {
			s.oauthError(w, "invalid_grant", "redirect_uri mismatch", 400)
			return
		}
		if authCode.ExpiresAt.Before(time.Now()) {
			_ = s.codeStore.Delete(r.Context(), codeHash)
			s.oauthError(w, "invalid_grant", "code expired", 400)
			return
		}
		tenantID = authCode.TenantID
		scopes = authCode.Scopes
		_ = s.codeStore.Delete(r.Context(), codeHash)
	case "refresh_token":
		if refreshToken == "" {
			s.oauthError(w, "invalid_request", "refresh_token required", 400)
			return
		}
		refreshHash := HashSHA256(refreshToken)
		tok, err := s.tokenStore.GetByRefreshTokenHash(r.Context(), refreshHash)
		if err != nil || tok == nil {
			s.oauthError(w, "invalid_grant", "invalid refresh token", 400)
			return
		}
		if tok.ClientID != clientID {
			s.oauthError(w, "invalid_grant", "token client mismatch", 400)
			return
		}
		tenantID = tok.TenantID
		scopes = tok.Scopes
	default:
		s.oauthError(w, "unsupported_grant_type", "authorization_code or refresh_token", 400)
		return
	}

	accessToken := randomToken(32)
	refreshTokenNew := randomToken(32)
	accessHash := HashSHA256(accessToken)
	refreshHash := HashSHA256(refreshTokenNew)

	token := &Token{
		AccessTokenHash:  accessHash,
		RefreshTokenHash: refreshHash,
		ClientID:         clientID,
		TenantID:         tenantID,
		Scopes:           scopes,
		ExpiresAt:        time.Now().Add(s.accessTTL),
	}
	if err := s.tokenStore.Create(r.Context(), token); err != nil {
		s.oauthError(w, "server_error", "failed to create token", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"refresh_token": refreshTokenNew,
		"scope":         strings.Join(scopes, " "),
	})
}

// HandleRevoke handles POST /oauth/revoke.
func (s *Server) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")
	if token == "" {
		w.WriteHeader(200)
		return
	}
	hash := HashSHA256(token)
	if tokenTypeHint == "refresh_token" {
		tok, _ := s.tokenStore.GetByRefreshTokenHash(r.Context(), hash)
		if tok != nil {
			_ = s.tokenStore.DeleteByAccessToken(r.Context(), tok.AccessTokenHash)
		}
	} else {
		_ = s.tokenStore.DeleteByAccessToken(r.Context(), hash)
	}
	w.WriteHeader(200)
}

// ListClients returns OAuth clients. tenantID empty = all.
func (s *Server) ListClients(ctx context.Context, tenantID string) ([]*Client, error) {
	return s.clientStore.List(ctx, tenantID)
}

// CreateClient creates an OAuth client. ClientSecretHash must be set (use HashSHA256 on raw secret).
func (s *Server) CreateClient(ctx context.Context, c *Client) error {
	return s.clientStore.Create(ctx, c)
}

// UpdateClient updates redirect_uris, scopes, tenant_id.
func (s *Server) UpdateClient(ctx context.Context, c *Client) error {
	return s.clientStore.Update(ctx, c)
}

// DeleteClient removes an OAuth client by client_id.
func (s *Server) DeleteClient(ctx context.Context, clientID string) error {
	return s.clientStore.Delete(ctx, clientID)
}

// ValidateAccessToken validates an access token and returns claims for API auth.
func (s *Server) ValidateAccessToken(ctx context.Context, accessToken string) (*auth.Claims, error) {
	if accessToken == "" {
		return nil, nil
	}
	hash := HashSHA256(accessToken)
	tok, err := s.tokenStore.GetByAccessTokenHash(ctx, hash)
	if err != nil || tok == nil {
		return nil, nil
	}
	if tok.ExpiresAt.Before(time.Now()) {
		_ = s.tokenStore.DeleteByAccessToken(ctx, hash)
		return nil, nil
	}
	return &auth.Claims{
		TenantID: tok.TenantID,
		Roles:    []string{auth.RoleOperator},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "oauth:" + tok.ClientID,
			ID:      tok.ID,
		},
	}, nil
}

func (s *Server) oauthError(w http.ResponseWriter, errCode, errDesc string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": errDesc,
	})
}

func parseScopes(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, " ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsScope(scopes []string, s string) bool {
	for _, sc := range scopes {
		if sc == s {
			return true
		}
	}
	return false
}

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
