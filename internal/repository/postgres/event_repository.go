package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainevent "eventra/internal/domain/event"
	usecaseevent "eventra/internal/usecase/event"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, e domainevent.Event) (domainevent.Event, error) {
	query := `
		INSERT INTO events (id, organizer_id, title, description, event_date, location, visibility, participant_limit, category, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10)
		RETURNING id, organizer_id, title, description, event_date, location, visibility, participant_limit, COALESCE(category, ''), tags, created_at, updated_at
	`

	if err := r.db.QueryRow(
		ctx,
		query,
		e.ID,
		e.OrganizerID,
		e.Title,
		e.Description,
		e.EventDate,
		e.Location,
		string(e.Visibility),
		e.ParticipantLimit,
		e.Category,
		e.Tags,
	).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.Title,
		&e.Description,
		&e.EventDate,
		&e.Location,
		&e.Visibility,
		&e.ParticipantLimit,
		&e.Category,
		&e.Tags,
		&e.CreatedAt,
		&e.UpdatedAt,
	); err != nil {
		return domainevent.Event{}, fmt.Errorf("insert event: %w", err)
	}

	return e, nil
}

func (r *EventRepository) GetByIDAny(ctx context.Context, eventID uuid.UUID) (domainevent.Event, error) {
	query := `
		SELECT id, organizer_id, title, description, event_date, location, visibility, participant_limit, COALESCE(category, ''), tags, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var e domainevent.Event
	err := r.db.QueryRow(ctx, query, eventID).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.Title,
		&e.Description,
		&e.EventDate,
		&e.Location,
		&e.Visibility,
		&e.ParticipantLimit,
		&e.Category,
		&e.Tags,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainevent.Event{}, domainevent.ErrNotFound
	}
	if err != nil {
		return domainevent.Event{}, fmt.Errorf("get event by id any: %w", err)
	}

	return e, nil
}

func (r *EventRepository) GetByID(ctx context.Context, eventID uuid.UUID, requesterUserID *uuid.UUID) (domainevent.Event, error) {
	args := []any{eventID}
	query := `
		SELECT id, organizer_id, title, description, event_date, location, visibility, participant_limit, COALESCE(category, ''), tags, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	if requesterUserID == nil {
		query += ` AND visibility = 'public'`
	} else {
		args = append(args, *requesterUserID)
		query += fmt.Sprintf(" AND (visibility = 'public' OR organizer_id = $%d)", len(args))
	}

	var e domainevent.Event
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.Title,
		&e.Description,
		&e.EventDate,
		&e.Location,
		&e.Visibility,
		&e.ParticipantLimit,
		&e.Category,
		&e.Tags,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainevent.Event{}, domainevent.ErrNotFound
	}
	if err != nil {
		return domainevent.Event{}, fmt.Errorf("get event by id: %w", err)
	}

	return e, nil
}

func (r *EventRepository) List(ctx context.Context, filter usecaseevent.ListFilter) ([]domainevent.Event, error) {
	conditions := []string{}
	args := []any{}

	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.RequesterUserID == nil {
		conditions = append(conditions, "visibility = 'public'")
	} else {
		conditions = append(conditions, fmt.Sprintf("(visibility = 'public' OR organizer_id = %s)", arg(*filter.RequesterUserID)))
	}

	if filter.Visibility != "" {
		conditions = append(conditions, fmt.Sprintf("visibility = %s", arg(filter.Visibility)))
	}
	if filter.Query != "" {
		qArg := arg("%" + filter.Query + "%")
		conditions = append(conditions, fmt.Sprintf("(title ILIKE %s OR description ILIKE %s)", qArg, qArg))
	}
	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = %s", arg(filter.Category)))
	}
	if filter.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("%s = ANY(tags)", arg(strings.ToLower(filter.Tag))))
	}
	if filter.Location != "" {
		conditions = append(conditions, fmt.Sprintf("location ILIKE %s", arg("%"+filter.Location+"%")))
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("event_date >= %s", arg(*filter.FromDate)))
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("event_date <= %s", arg(*filter.ToDate)))
	}

	query := `
		SELECT id, organizer_id, title, description, event_date, location, visibility, participant_limit, COALESCE(category, ''), tags, created_at, updated_at
		FROM events
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY event_date ASC LIMIT %s OFFSET %s", arg(filter.Limit), arg(filter.Offset))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := make([]domainevent.Event, 0)
	for rows.Next() {
		var e domainevent.Event
		if err = rows.Scan(
			&e.ID,
			&e.OrganizerID,
			&e.Title,
			&e.Description,
			&e.EventDate,
			&e.Location,
			&e.Visibility,
			&e.ParticipantLimit,
			&e.Category,
			&e.Tags,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}
		events = append(events, e)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event rows: %w", err)
	}

	return events, nil
}

func (r *EventRepository) Update(ctx context.Context, e domainevent.Event) (domainevent.Event, error) {
	query := `
		UPDATE events
		SET title = $2,
		    description = $3,
		    event_date = $4,
		    location = $5,
		    visibility = $6,
		    participant_limit = $7,
		    category = NULLIF($8, ''),
		    tags = $9,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, organizer_id, title, description, event_date, location, visibility, participant_limit, COALESCE(category, ''), tags, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		e.ID,
		e.Title,
		e.Description,
		e.EventDate,
		e.Location,
		string(e.Visibility),
		e.ParticipantLimit,
		e.Category,
		e.Tags,
	).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.Title,
		&e.Description,
		&e.EventDate,
		&e.Location,
		&e.Visibility,
		&e.ParticipantLimit,
		&e.Category,
		&e.Tags,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainevent.Event{}, domainevent.ErrNotFound
	}
	if err != nil {
		return domainevent.Event{}, fmt.Errorf("update event: %w", err)
	}

	return e, nil
}

func (r *EventRepository) Delete(ctx context.Context, eventID uuid.UUID) error {
	query := `DELETE FROM events WHERE id = $1`
	res, err := r.db.Exec(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domainevent.ErrNotFound
	}

	return nil
}
