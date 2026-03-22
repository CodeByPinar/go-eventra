package event

import (
	"context"
	"time"

	domainevent "eventra/internal/domain/event"

	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, domainevent.Event) (domainevent.Event, error)
	GetByIDAny(context.Context, uuid.UUID) (domainevent.Event, error)
	GetByID(context.Context, uuid.UUID, *uuid.UUID) (domainevent.Event, error)
	Update(context.Context, domainevent.Event) (domainevent.Event, error)
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]domainevent.Event, error)
}

type CreateInput struct {
	OrganizerID      uuid.UUID
	Title            string
	Description      string
	EventDate        time.Time
	Location         string
	Visibility       string
	ParticipantLimit *int
	Category         string
	Tags             []string
}

type ListInput struct {
	RequesterUserID *uuid.UUID
	Query           string
	Category        string
	Tag             string
	Location        string
	Visibility      string
	FromDate        *time.Time
	ToDate          *time.Time
	Limit           int
	Offset          int
}

type UpdateInput struct {
	ID               uuid.UUID
	RequesterUserID  uuid.UUID
	Title            string
	Description      string
	EventDate        time.Time
	Location         string
	Visibility       string
	ParticipantLimit *int
	Category         string
	Tags             []string
}

type ListFilter struct {
	RequesterUserID *uuid.UUID
	Query           string
	Category        string
	Tag             string
	Location        string
	Visibility      string
	FromDate        *time.Time
	ToDate          *time.Time
	Limit           int
	Offset          int
}
