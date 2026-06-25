package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListEvents handler returns events, optionally filtered by status.
type ListEvents struct {
	events eventsdomain.Service
}

// NewListEvents creates a ListEvents handler.
func NewListEvents(svc eventsdomain.Service) *ListEvents {
	return &ListEvents{events: svc}
}

// Handle GET /events.
func (h *ListEvents) Handle(params eventsops.ListEventsParams) middleware.Responder {
	status := ""
	if params.Status != nil {
		status = *params.Status
	}

	list, err := h.events.List(params.HTTPRequest.Context(), status)
	if err != nil {
		logger.Log().Errorf("list events: %s", err.Error())
		if errors.Is(err, eventsdomain.ErrInvalidInput) {
			return eventsops.NewListEventsBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return eventsops.NewListEventsServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Event, 0, len(list))
	for _, e := range list {
		payload = append(payload, formatter.EventToAPI(e))
	}

	return eventsops.NewListEventsOK().WithPayload(payload)
}

// GetEventByID handler returns a single event by UUID.
type GetEventByID struct {
	events eventsdomain.Service
}

// NewGetEventByID creates a GetEventByID handler.
func NewGetEventByID(svc eventsdomain.Service) *GetEventByID {
	return &GetEventByID{events: svc}
}

// Handle GET /events/{id}.
func (h *GetEventByID) Handle(params eventsops.GetEventByIDParams) middleware.Responder {
	event, err := h.events.GetByID(params.HTTPRequest.Context(), params.ID.String())
	if err != nil {
		logger.Log().Errorf("get event %s: %s", params.ID.String(), err.Error())
		switch {
		case errors.Is(err, eventsdomain.ErrInvalidInput):
			return eventsops.NewGetEventByIDBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, eventsdomain.ErrNotFound):
			return eventsops.NewGetEventByIDNotFound().
				WithPayload(DefaultError(http.StatusNotFound, err, nil))
		default:
			return eventsops.NewGetEventByIDServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	return eventsops.NewGetEventByIDOK().WithPayload(formatter.EventToAPI(event))
}

// CreateEvent handler creates a new event.
type CreateEvent struct {
	events eventsdomain.Service
}

// NewCreateEvent creates a CreateEvent handler.
func NewCreateEvent(svc eventsdomain.Service) *CreateEvent {
	return &CreateEvent{events: svc}
}

// Handle POST /events.
func (h *CreateEvent) Handle(params eventsops.CreateEventParams, principal *apimodels.User) middleware.Responder {
	event, err := formatter.EventFromAPIInput(params.Body)
	if err != nil {
		return eventsops.NewCreateEventBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}

	// The organizer is the authenticated user — never trust a client-supplied
	// organizer_id from the request body.
	if principal != nil {
		if id, err := uuid.FromString(principal.UUID.String()); err == nil {
			event.OrganizerID = id
		}
	}

	if err := h.events.Create(params.HTTPRequest.Context(), event); err != nil {
		logger.Log().Errorf("create event: %s", err.Error())
		if errors.Is(err, eventsdomain.ErrInvalidInput) {
			return eventsops.NewCreateEventBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return eventsops.NewCreateEventServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	return eventsops.NewCreateEventCreated().WithPayload(formatter.EventToAPI(event))
}
