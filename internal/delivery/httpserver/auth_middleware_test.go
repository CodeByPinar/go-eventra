package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"eventra/pkg/security"
)

type stubTokenValidator struct {
	validateFn func(string) (security.Claims, error)
}

func (s stubTokenValidator) ValidateToken(rawToken string) (security.Claims, error) {
	return s.validateFn(rawToken)
}

func TestRequireAuthMissingHeader(t *testing.T) {
	validator := stubTokenValidator{validateFn: func(string) (security.Claims, error) { return security.Claims{}, nil }}
	h := RequireAuth(validator, nil, func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthInvalidToken(t *testing.T) {
	validator := stubTokenValidator{validateFn: func(string) (security.Claims, error) {
		return security.Claims{}, errors.New("invalid")
	}}
	h := RequireAuth(validator, nil, func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthSuccessAndMeEndpoint(t *testing.T) {
	validator := stubTokenValidator{validateFn: func(rawToken string) (security.Claims, error) {
		if rawToken != "good-token" {
			t.Fatalf("unexpected token: %s", rawToken)
		}
		return security.Claims{UserID: "user-1", Email: "alice@example.com"}, nil
	}}

	handler := &AuthHandler{}
	protected := RequireAuth(validator, nil, handler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["user_id"] != "user-1" || payload["email"] != "alice@example.com" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
