package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	usecaseevent "eventra/internal/usecase/event"
	"eventra/pkg/security"
)

func newTestRouter() http.Handler {
	handler := &AuthHandler{}
	eventHandler := NewEventHandler(usecaseevent.NewService(nil))
	jwtManager := security.NewJWTManager("test-secret", time.Hour)
	return NewRouter(handler, eventHandler, jwtManager, nil, []string{"http://localhost:5173"})
}

func TestCORSBypassBlocked(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", rec.Code)
	}
}

func TestMalformedJSONRejected(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed json, got %d", rec.Code)
	}
}

func TestRateLimitSuiteAuthAbuse(t *testing.T) {
	router := newTestRouter()

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:5173")
		req.RemoteAddr = "198.51.100.50:12345"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	req.RemoteAddr = "198.51.100.50:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when rate limit exceeded, got %d", rec.Code)
	}
}
