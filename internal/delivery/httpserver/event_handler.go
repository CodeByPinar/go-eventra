package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainevent "eventra/internal/domain/event"
	usecaseevent "eventra/internal/usecase/event"

	"github.com/google/uuid"
)

type EventHandler struct {
	eventService *usecaseevent.Service
}

func NewEventHandler(eventService *usecaseevent.Service) *EventHandler {
	return &EventHandler{eventService: eventService}
}

type createEventRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	EventDate        string   `json:"event_date"`
	Location         string   `json:"location"`
	Visibility       string   `json:"visibility"`
	ParticipantLimit *int     `json:"participant_limit"`
	Category         string   `json:"category"`
	Tags             []string `json:"tags"`
}

type updateEventRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	EventDate        string   `json:"event_date"`
	Location         string   `json:"location"`
	Visibility       string   `json:"visibility"`
	ParticipantLimit *int     `json:"participant_limit"`
	Category         string   `json:"category"`
	Tags             []string `json:"tags"`
}

type eventResponse struct {
	ID               string   `json:"id"`
	OrganizerID      string   `json:"organizer_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	EventDate        string   `json:"event_date"`
	Location         string   `json:"location"`
	Visibility       string   `json:"visibility"`
	ParticipantLimit *int     `json:"participant_limit,omitempty"`
	Category         string   `json:"category,omitempty"`
	Tags             []string `json:"tags"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := getAuthClaims(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	organizerID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid auth context"})
		return
	}

	var req createEventRequest
	if err = decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	eventDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EventDate))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "event_date must be RFC3339"})
		return
	}

	created, err := h.eventService.Create(r.Context(), usecaseevent.CreateInput{
		OrganizerID:      organizerID,
		Title:            req.Title,
		Description:      req.Description,
		EventDate:        eventDate,
		Location:         req.Location,
		Visibility:       req.Visibility,
		ParticipantLimit: req.ParticipantLimit,
		Category:         req.Category,
		Tags:             req.Tags,
	})
	if err != nil {
		if errors.Is(err, usecaseevent.ErrInvalidInput) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create event"})
		return
	}

	respondJSON(w, http.StatusCreated, toEventResponse(created))
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()

	var requesterUserID *uuid.UUID
	if claims, ok := getAuthClaims(r.Context()); ok {
		if parsed, err := uuid.Parse(claims.UserID); err == nil {
			requesterUserID = &parsed
		}
	}

	fromDate, err := parseOptionalRFC3339(queryValues.Get("from_date"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "from_date must be RFC3339"})
		return
	}
	toDate, err := parseOptionalRFC3339(queryValues.Get("to_date"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "to_date must be RFC3339"})
		return
	}

	limit := parseIntDefault(queryValues.Get("limit"), 20)
	offset := parseIntDefault(queryValues.Get("offset"), 0)

	events, err := h.eventService.List(r.Context(), usecaseevent.ListInput{
		RequesterUserID: requesterUserID,
		Query:           queryValues.Get("q"),
		Category:        queryValues.Get("category"),
		Tag:             queryValues.Get("tag"),
		Location:        queryValues.Get("location"),
		Visibility:      queryValues.Get("visibility"),
		FromDate:        fromDate,
		ToDate:          toDate,
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		if errors.Is(err, usecaseevent.ErrInvalidInput) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list events"})
		return
	}

	response := make([]eventResponse, 0, len(events))
	for _, e := range events {
		response = append(response, toEventResponse(e))
	}

	respondJSON(w, http.StatusOK, map[string]any{"items": response, "count": len(response)})
}

func (h *EventHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	var requesterUserID *uuid.UUID
	if claims, ok := getAuthClaims(r.Context()); ok {
		if parsed, parseErr := uuid.Parse(claims.UserID); parseErr == nil {
			requesterUserID = &parsed
		}
	}

	e, err := h.eventService.GetByID(r.Context(), eventID, requesterUserID)
	if err != nil {
		if errors.Is(err, domainevent.ErrNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, usecaseevent.ErrForbidden) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not get event"})
		return
	}

	respondJSON(w, http.StatusOK, toEventResponse(e))
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := getAuthClaims(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	requesterUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid auth context"})
		return
	}

	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	var req updateEventRequest
	if err = decodeStrictJSONBody(w, r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	eventDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EventDate))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "event_date must be RFC3339"})
		return
	}

	updated, err := h.eventService.Update(r.Context(), usecaseevent.UpdateInput{
		ID:               eventID,
		RequesterUserID:  requesterUserID,
		Title:            req.Title,
		Description:      req.Description,
		EventDate:        eventDate,
		Location:         req.Location,
		Visibility:       req.Visibility,
		ParticipantLimit: req.ParticipantLimit,
		Category:         req.Category,
		Tags:             req.Tags,
	})
	if err != nil {
		if errors.Is(err, usecaseevent.ErrInvalidInput) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, domainevent.ErrNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, usecaseevent.ErrForbidden) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update event"})
		return
	}

	respondJSON(w, http.StatusOK, toEventResponse(updated))
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := getAuthClaims(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	requesterUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid auth context"})
		return
	}

	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	err = h.eventService.Delete(r.Context(), eventID, requesterUserID)
	if err != nil {
		if errors.Is(err, domainevent.ErrNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, usecaseevent.ErrForbidden) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not delete event"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toEventResponse(e domainevent.Event) eventResponse {
	return eventResponse{
		ID:               e.ID.String(),
		OrganizerID:      e.OrganizerID.String(),
		Title:            e.Title,
		Description:      e.Description,
		EventDate:        e.EventDate.UTC().Format(time.RFC3339),
		Location:         e.Location,
		Visibility:       string(e.Visibility),
		ParticipantLimit: e.ParticipantLimit,
		Category:         e.Category,
		Tags:             e.Tags,
		CreatedAt:        e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func parseIntDefault(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
