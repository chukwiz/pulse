package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/chukwiz/pulse/internal/domain"
	"github.com/chukwiz/pulse/internal/middleware"
)

const (
	maxBodyBytes     = domain.MaxDataBytes + 8<<10
	defaultListLimit = 50
	maxListLimit     = 100
)

var reqBodyTooLarge = fmt.Sprintf("request body too large: data must be at most %d KB", domain.MaxDataBytes/1024)

type EventStore interface {
	Save(ctx context.Context, e domain.Event) (domain.Event, bool, error)
	Get(ctx context.Context, id string) (domain.Event, error)
	List(ctx context.Context, eventType, cursor string, limit int) ([]domain.Event, string, error)
}

type createEventRequest struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type createEventResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type eventResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

type eventsResponse struct {
	Data       []eventResponse `json:"data"`
	NextCursor string          `json:"next_cursor"`
}

type Handler struct {
	store  EventStore
	logger *slog.Logger
}

func New(store EventStore, logger *slog.Logger) *Handler {
	return &Handler{
		store:  store,
		logger: logger,
	}
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(w, http.StatusRequestEntityTooLarge, reqBodyTooLarge, codePayloadTooLarge)
			return
		}

		writeError(w, http.StatusBadRequest, "request body must be a JSON object", codeInvalidJSON)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required", codeMissingIdempotencyKey)
		return
	}

	apiKeyID := middleware.APIKeyIDFrom(r.Context())

	event, err := domain.NewEvent(req.Type, idempotencyKey, apiKeyID, req.Data)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidEvent) {
			writeError(w, http.StatusBadRequest, err.Error(), codeInvalidEvent)
			return
		}
		h.internalError(w, r, "build event", err)
		return
	}

	stored, created, err := h.store.Save(r.Context(), event)
	if err != nil {
		h.internalError(w, r, "save event", err)
		return
	}

	if !created {
		h.logger.Info("idempotent replay", "event_id", stored.ID)
	}
	writeJSON(w, http.StatusAccepted, createEventResponse{
		ID:     stored.ID,
		Status: "queued",
	})

}

func (h *Handler) getEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	event, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error(), codeEventNotFound)
			return
		}
		h.internalError(w, r, "get event", err)
		return
	}

	writeJSON(w, http.StatusOK, toEventResponse(event))
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	eventType := queryParams.Get("type")
	cursor := queryParams.Get("cursor")

	limit, err := parseLimit(queryParams.Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), codeInvalidQueryParameter)
		return
	}

	events, next, err := h.store.List(r.Context(), eventType, cursor, limit)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCursor) {
			writeError(w, http.StatusBadRequest, err.Error(), codeInvalidCursor)
			return
		}
		h.internalError(w, r, "list events", err)
		return
	}

	eventsResponseData := make([]eventResponse, 0, len(events))

	for _, event := range events {
		eventsResponseData = append(eventsResponseData, toEventResponse(event))
	}

	listResponse := eventsResponse{
		Data:       eventsResponseData,
		NextCursor: next,
	}

	writeJSON(w, http.StatusOK, listResponse)
}

func toEventResponse(e domain.Event) eventResponse {
	return eventResponse{
		ID:        e.ID,
		Type:      e.Type,
		Data:      e.Data,
		CreatedAt: e.CreatedAt,
	}
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return defaultListLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 0, fmt.Errorf("limit must be an integer between 1 and %d", maxListLimit)
	}

	return min(limit, maxListLimit), nil
}
