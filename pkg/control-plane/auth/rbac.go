package auth

import (
	"net/http"
	"strings"
)

// Role constants.
const (
	RoleOperator     = "operator"
	RoleAdministrator = "administrator"
	RoleSystem       = "system"
)

// RequireRole returns a middleware that checks the user has one of the required roles.
// If no claims in context (auth not configured), the check is skipped.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
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
