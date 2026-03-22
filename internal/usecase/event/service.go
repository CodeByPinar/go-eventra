package event

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainevent "eventra/internal/domain/event"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domainevent.Event, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domainevent.Event{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.EventDate.IsZero() {
		return domainevent.Event{}, fmt.Errorf("%w: event_date is required", ErrInvalidInput)
	}

	location := strings.TrimSpace(input.Location)
	if location == "" {
		return domainevent.Event{}, fmt.Errorf("%w: location is required", ErrInvalidInput)
	}

	visibility := strings.ToLower(strings.TrimSpace(input.Visibility))
	if visibility == "" {
		visibility = string(domainevent.VisibilityPublic)
	}
	if visibility != string(domainevent.VisibilityPublic) && visibility != string(domainevent.VisibilityPrivate) {
		return domainevent.Event{}, fmt.Errorf("%w: visibility must be public or private", ErrInvalidInput)
	}

	if input.ParticipantLimit != nil && *input.ParticipantLimit < 0 {
		return domainevent.Event{}, fmt.Errorf("%w: participant_limit must be >= 0", ErrInvalidInput)
	}

	newEvent := domainevent.Event{
		ID:               uuid.New(),
		OrganizerID:      input.OrganizerID,
		Title:            title,
		Description:      strings.TrimSpace(input.Description),
		EventDate:        input.EventDate.UTC(),
		Location:         location,
		Visibility:       domainevent.Visibility(visibility),
		ParticipantLimit: input.ParticipantLimit,
		Category:         strings.TrimSpace(input.Category),
		Tags:             normalizeTags(input.Tags),
	}

	created, err := s.repo.Create(ctx, newEvent)
	if err != nil {
		return domainevent.Event{}, fmt.Errorf("create event: %w", err)
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, eventID uuid.UUID, requesterUserID *uuid.UUID) (domainevent.Event, error) {
	e, err := s.repo.GetByID(ctx, eventID, requesterUserID)
	if err != nil {
		return domainevent.Event{}, err
	}

	if e.Visibility == domainevent.VisibilityPrivate && (requesterUserID == nil || *requesterUserID != e.OrganizerID) {
		return domainevent.Event{}, ErrForbidden
	}

	return e, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domainevent.Event, error) {
	filter := ListFilter{
		RequesterUserID: input.RequesterUserID,
		Query:           strings.TrimSpace(input.Query),
		Category:        strings.TrimSpace(input.Category),
		Tag:             strings.TrimSpace(input.Tag),
		Location:        strings.TrimSpace(input.Location),
		Visibility:      strings.ToLower(strings.TrimSpace(input.Visibility)),
		FromDate:        input.FromDate,
		ToDate:          input.ToDate,
		Limit:           input.Limit,
		Offset:          input.Offset,
	}

	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	if filter.Visibility != "" && filter.Visibility != string(domainevent.VisibilityPublic) && filter.Visibility != string(domainevent.VisibilityPrivate) {
		return nil, fmt.Errorf("%w: visibility must be public or private", ErrInvalidInput)
	}
	if filter.Visibility == string(domainevent.VisibilityPrivate) && filter.RequesterUserID == nil {
		return []domainevent.Event{}, nil
	}
	if filter.FromDate != nil && filter.ToDate != nil && filter.ToDate.Before(*filter.FromDate) {
		return nil, fmt.Errorf("%w: to_date must be after from_date", ErrInvalidInput)
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domainevent.Event, error) {
	existing, err := s.repo.GetByIDAny(ctx, input.ID)
	if err != nil {
		return domainevent.Event{}, err
	}

	if existing.OrganizerID != input.RequesterUserID {
		return domainevent.Event{}, ErrForbidden
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domainevent.Event{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.EventDate.IsZero() {
		return domainevent.Event{}, fmt.Errorf("%w: event_date is required", ErrInvalidInput)
	}

	location := strings.TrimSpace(input.Location)
	if location == "" {
		return domainevent.Event{}, fmt.Errorf("%w: location is required", ErrInvalidInput)
	}

	visibility := strings.ToLower(strings.TrimSpace(input.Visibility))
	if visibility == "" {
		visibility = string(domainevent.VisibilityPublic)
	}
	if visibility != string(domainevent.VisibilityPublic) && visibility != string(domainevent.VisibilityPrivate) {
		return domainevent.Event{}, fmt.Errorf("%w: visibility must be public or private", ErrInvalidInput)
	}

	if input.ParticipantLimit != nil && *input.ParticipantLimit < 0 {
		return domainevent.Event{}, fmt.Errorf("%w: participant_limit must be >= 0", ErrInvalidInput)
	}

	existing.Title = title
	existing.Description = strings.TrimSpace(input.Description)
	existing.EventDate = input.EventDate.UTC()
	existing.Location = location
	existing.Visibility = domainevent.Visibility(visibility)
	existing.ParticipantLimit = input.ParticipantLimit
	existing.Category = strings.TrimSpace(input.Category)
	existing.Tags = normalizeTags(input.Tags)

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return domainevent.Event{}, fmt.Errorf("update event: %w", err)
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, eventID, requesterUserID uuid.UUID) error {
	existing, err := s.repo.GetByIDAny(ctx, eventID)
	if err != nil {
		return err
	}

	if existing.OrganizerID != requesterUserID {
		return ErrForbidden
	}

	if err = s.repo.Delete(ctx, eventID); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	return nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		t := strings.ToLower(strings.TrimSpace(tag))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		normalized = append(normalized, t)
	}

	return normalized
}
