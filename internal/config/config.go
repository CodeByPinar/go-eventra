package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port                       string
	DBURL                      string
	JWTSecret                  string
	JWTExpiration              time.Duration
	RefreshTokenExpiration     time.Duration
	CORSAllowedOrigins         []string
	SecurityAlertWebhookURL    string
	SecurityAlertWebhookFormat string
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return Config{}, errors.New("DB_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	jwtExpiration := 24 * time.Hour
	if raw := os.Getenv("JWT_EXPIRATION"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid JWT_EXPIRATION: %w", err)
		}
		jwtExpiration = parsed
	}

	refreshExpiration := 7 * 24 * time.Hour
	if raw := os.Getenv("REFRESH_TOKEN_EXPIRATION"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid REFRESH_TOKEN_EXPIRATION: %w", err)
		}
		refreshExpiration = parsed
	}

	allowedOrigins := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); raw != "" {
		parts := strings.Split(raw, ",")
		parsed := make([]string, 0, len(parts))
		for _, part := range parts {
			origin := strings.TrimSpace(part)
			if origin != "" {
				parsed = append(parsed, origin)
			}
		}
		if len(parsed) > 0 {
			allowedOrigins = parsed
		}
	}

	return Config{
		Port:                       port,
		DBURL:                      dbURL,
		JWTSecret:                  jwtSecret,
		JWTExpiration:              jwtExpiration,
		RefreshTokenExpiration:     refreshExpiration,
		CORSAllowedOrigins:         allowedOrigins,
		SecurityAlertWebhookURL:    strings.TrimSpace(os.Getenv("SECURITY_ALERT_WEBHOOK_URL")),
		SecurityAlertWebhookFormat: strings.ToLower(strings.TrimSpace(os.Getenv("SECURITY_ALERT_WEBHOOK_FORMAT"))),
	}, nil
}
