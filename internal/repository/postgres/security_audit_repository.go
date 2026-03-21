package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"eventra/internal/usecase/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SecurityAuditRepository struct {
	db            *pgxpool.Pool
	webhookURL    string
	webhookFormat string
	httpClient    *http.Client
}

func NewSecurityAuditRepository(db *pgxpool.Pool, webhookURL, webhookFormat string) *SecurityAuditRepository {
	format := strings.TrimSpace(strings.ToLower(webhookFormat))
	if format == "" {
		format = "json"
	}

	return &SecurityAuditRepository{
		db:            db,
		webhookURL:    strings.TrimSpace(webhookURL),
		webhookFormat: format,
		httpClient:    &http.Client{Timeout: 4 * time.Second},
	}
}

func (r *SecurityAuditRepository) LogEvent(ctx context.Context, event auth.AuditEvent) error {
	var userID *uuid.UUID
	if event.UserID != nil && *event.UserID != uuid.Nil {
		id := *event.UserID
		userID = &id
	}

	metadata := []byte("{}")
	if len(event.Metadata) > 0 {
		payload, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshal audit metadata: %w", err)
		}
		metadata = payload
	}

	query := `
		INSERT INTO security_audit_logs (id, event_type, severity, user_id, ip, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
	`

	if _, err := r.db.Exec(ctx, query, uuid.New(), event.EventType, string(event.Severity), userID, event.IP, event.UserAgent, string(metadata), event.OccurredAt); err != nil {
		return fmt.Errorf("insert security audit log: %w", err)
	}

	if event.Severity == auth.AuditSeverityHigh {
		log.Printf("SECURITY ALERT: %s user=%v ip=%s", event.EventType, userID, event.IP)
		_ = r.sendExternalAlert(ctx, event, userID)
	}

	return nil
}

func (r *SecurityAuditRepository) sendExternalAlert(ctx context.Context, event auth.AuditEvent, userID *uuid.UUID) error {
	if r.webhookURL == "" {
		return nil
	}

	payload := map[string]any{
		"event_type":  event.EventType,
		"severity":    string(event.Severity),
		"ip":          event.IP,
		"user_agent":  event.UserAgent,
		"occurred_at": event.OccurredAt,
		"metadata":    event.Metadata,
	}
	userIDText := ""
	if userID != nil {
		userIDText = userID.String()
		payload["user_id"] = userIDText
	}

	var body []byte
	var err error
	if r.webhookFormat == "slack" {
		text := fmt.Sprintf("[SECURITY ALERT] %s severity=%s user=%s ip=%s", event.EventType, event.Severity, userIDText, event.IP)
		body, err = json.Marshal(map[string]string{"text": text})
	} else {
		body, err = json.Marshal(payload)
	}
	if err != nil {
		return fmt.Errorf("marshal alert payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.webhookURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create alert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send alert webhook: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("alert webhook status: %d", res.StatusCode)
	}

	return nil
}
