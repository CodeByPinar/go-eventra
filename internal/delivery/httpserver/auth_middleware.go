package httpserver

import (
	"context"
	"net/http"
	"strings"

	"eventra/pkg/security"
)

type contextKey string

const authClaimsContextKey contextKey = "auth_claims"

type tokenValidator interface {
	ValidateToken(rawToken string) (security.Claims, error)
}

type tokenBlacklistChecker interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

func RequireAuth(validator tokenValidator, blacklist tokenBlacklistChecker, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
			return
		}

		claims, err := validator.ValidateToken(parts[1])
		if err != nil {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		if blacklist != nil && claims.ID != "" {
			blocked, err := blacklist.IsBlacklisted(r.Context(), claims.ID)
			if err != nil {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			if blocked {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "token revoked"})
				return
			}
		}

		ctx := context.WithValue(r.Context(), authClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getAuthClaims(ctx context.Context) (security.Claims, bool) {
	claims, ok := ctx.Value(authClaimsContextKey).(security.Claims)
	return claims, ok
}
