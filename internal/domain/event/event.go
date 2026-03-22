package event

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("event not found")

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type Event struct {
	ID               uuid.UUID
	OrganizerID      uuid.UUID
	Title            string
	Description      string
	EventDate        time.Time
	Location         string
	Visibility       Visibility
	ParticipantLimit *int
	Category         string
	Tags             []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
