package auth

import (
	"net/http"
	"os"
	"strings"
)

// Role constants.
const (
	RoleOperator      = "operator"
	RoleAdministrator = "administrator"
	RoleViewer        = "viewer"
	RoleSystem        = "system"
)

// RequireRole returns a middleware that checks the user has one of the required roles.
// When claims is nil: if ALLOW_UNSAFE_NO_AUTH=true (dev only), passes through; otherwise returns 401.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				if os.Getenv("ALLOW_UNSAFE_NO_AUTH") == "true" {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "missing authorization", http.StatusUnauthorized)
				return
			}
			userRoles := claims.GetRoles()
			if !hasAnyRole(userRoles, roles) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasAnyRole(userRoles []string, required []string) bool {
	for _, r := range required {
		for _, ur := range userRoles {
			if strings.EqualFold(ur, r) {
				return true
			}
		}
	}
	return false
}
