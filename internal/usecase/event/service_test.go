package event

import (
	"context"
	"errors"
	"testing"
	"time"

	domainevent "eventra/internal/domain/event"

	"github.com/google/uuid"
)

type fakeRepository struct {
	createFn     func(context.Context, domainevent.Event) (domainevent.Event, error)
	getByIDAnyFn func(context.Context, uuid.UUID) (domainevent.Event, error)
	getByIDFn    func(context.Context, uuid.UUID, *uuid.UUID) (domainevent.Event, error)
	updateFn     func(context.Context, domainevent.Event) (domainevent.Event, error)
	deleteFn     func(context.Context, uuid.UUID) error
	listFn       func(context.Context, ListFilter) ([]domainevent.Event, error)
}

func (f fakeRepository) Create(ctx context.Context, e domainevent.Event) (domainevent.Event, error) {
	if f.createFn != nil {
		return f.createFn(ctx, e)
	}
	return e, nil
}

func (f fakeRepository) GetByIDAny(ctx context.Context, id uuid.UUID) (domainevent.Event, error) {
	if f.getByIDAnyFn != nil {
		return f.getByIDAnyFn(ctx, id)
	}
	return domainevent.Event{}, domainevent.ErrNotFound
}

func (f fakeRepository) GetByID(ctx context.Context, id uuid.UUID, requester *uuid.UUID) (domainevent.Event, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id, requester)
	}
	return domainevent.Event{}, domainevent.ErrNotFound
}

func (f fakeRepository) Update(ctx context.Context, e domainevent.Event) (domainevent.Event, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, e)
	}
	return e, nil
}

func (f fakeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f fakeRepository) List(ctx context.Context, filter ListFilter) ([]domainevent.Event, error) {
	if f.listFn != nil {
		return f.listFn(ctx, filter)
	}
	return []domainevent.Event{}, nil
}

func TestUpdateRejectsNonOwner(t *testing.T) {
	ownerID := uuid.New()
	requesterID := uuid.New()
	eventID := uuid.New()

	svc := NewService(fakeRepository{
		getByIDAnyFn: func(context.Context, uuid.UUID) (domainevent.Event, error) {
			return domainevent.Event{ID: eventID, OrganizerID: ownerID}, nil
		},
	})

	_, err := svc.Update(context.Background(), UpdateInput{
		ID:              eventID,
		RequesterUserID: requesterID,
		Title:           "Updated",
		EventDate:       time.Now().Add(24 * time.Hour),
		Location:        "Istanbul",
		Visibility:      "public",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateOwnerSuccess(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	now := time.Now().UTC().Add(24 * time.Hour)

	svc := NewService(fakeRepository{
		getByIDAnyFn: func(context.Context, uuid.UUID) (domainevent.Event, error) {
			return domainevent.Event{ID: eventID, OrganizerID: ownerID}, nil
		},
		updateFn: func(_ context.Context, e domainevent.Event) (domainevent.Event, error) {
			if e.Title != "Updated Event" {
				t.Fatalf("unexpected title: %s", e.Title)
			}
			if e.Visibility != domainevent.VisibilityPrivate {
				t.Fatalf("unexpected visibility: %s", e.Visibility)
			}
			if len(e.Tags) != 1 || e.Tags[0] != "tech" {
				t.Fatalf("unexpected tags: %#v", e.Tags)
			}
			return e, nil
		},
	})

	updated, err := svc.Update(context.Background(), UpdateInput{
		ID:              eventID,
		RequesterUserID: ownerID,
		Title:           " Updated Event ",
		Description:     "desc",
		EventDate:       now,
		Location:        "Ankara",
		Visibility:      "private",
		Category:        "conference",
		Tags:            []string{"Tech", "tech", ""},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != eventID {
		t.Fatalf("unexpected event id: %s", updated.ID)
	}
}

func TestDeleteRejectsNonOwner(t *testing.T) {
	ownerID := uuid.New()
	requesterID := uuid.New()
	eventID := uuid.New()

	svc := NewService(fakeRepository{
		getByIDAnyFn: func(context.Context, uuid.UUID) (domainevent.Event, error) {
			return domainevent.Event{ID: eventID, OrganizerID: ownerID}, nil
		},
	})

	err := svc.Delete(context.Background(), eventID, requesterID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteOwnerSuccess(t *testing.T) {
	ownerID := uuid.New()
	eventID := uuid.New()
	deleted := false

	svc := NewService(fakeRepository{
		getByIDAnyFn: func(context.Context, uuid.UUID) (domainevent.Event, error) {
			return domainevent.Event{ID: eventID, OrganizerID: ownerID}, nil
		},
		deleteFn: func(context.Context, uuid.UUID) error {
			deleted = true
			return nil
		},
	})

	err := svc.Delete(context.Background(), eventID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to be called")
	}
}
