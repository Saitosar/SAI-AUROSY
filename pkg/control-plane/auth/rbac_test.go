package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireRole_NoClaims_PassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireRole(RoleOperator)(next)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRole_WithValidRole_PassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireRole(RoleOperator, RoleAdministrator)(next)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyClaims, &Claims{Roles: []string{"operator"}})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRole_WithInvalidRole_Forbidden(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireRole(RoleAdministrator)(next)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyClaims, &Claims{Roles: []string{"operator"}})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestRequireRole_RoleClaim_PassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireRole(RoleOperator)(next)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyClaims, &Claims{Role: "operator"})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
