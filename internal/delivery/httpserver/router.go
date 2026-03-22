package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"eventra/pkg/security"
)

func NewRouter(authHandler *AuthHandler, eventHandler *EventHandler, jwtManager *security.JWTManager, blacklist tokenBlacklistChecker, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()
	rateLimiter := newInMemoryRateLimiter(20, time.Minute)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("POST /api/v1/auth/register", authRateLimitMiddleware(rateLimiter, http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/v1/auth/login", authRateLimitMiddleware(rateLimiter, http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /api/v1/auth/refresh", authRateLimitMiddleware(rateLimiter, http.HandlerFunc(authHandler.Refresh)))
	mux.Handle("POST /api/v1/auth/logout", authRateLimitMiddleware(rateLimiter, http.HandlerFunc(authHandler.Logout)))
	mux.HandleFunc("GET /api/v1/auth/me", RequireAuth(jwtManager, blacklist, authHandler.Me))
	mux.HandleFunc("GET /api/v1/events", OptionalAuth(jwtManager, blacklist, eventHandler.List))
	mux.HandleFunc("GET /api/v1/events/{id}", OptionalAuth(jwtManager, blacklist, eventHandler.GetByID))
	mux.HandleFunc("POST /api/v1/events", RequireAuth(jwtManager, blacklist, eventHandler.Create))
	mux.HandleFunc("PUT /api/v1/events/{id}", RequireAuth(jwtManager, blacklist, eventHandler.Update))
	mux.HandleFunc("DELETE /api/v1/events/{id}", RequireAuth(jwtManager, blacklist, eventHandler.Delete))

	return chainMiddleware(
		mux,
		recoverMiddleware,
		requestLogger,
		securityHeadersMiddleware,
		bodyLimitMiddleware,
		corsMiddleware(allowedOrigins),
	)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s in %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
