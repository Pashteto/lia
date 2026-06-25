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

// UpdateEvent handler applies a partial update to an event owned by the caller.
type UpdateEvent struct {
	events eventsdomain.Service
}

// NewUpdateEvent creates an UpdateEvent handler.
func NewUpdateEvent(svc eventsdomain.Service) *UpdateEvent {
	return &UpdateEvent{events: svc}
}

// Handle PATCH /events/{id}.
func (h *UpdateEvent) Handle(params eventsops.UpdateEventParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return eventsops.NewUpdateEventUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	ownerID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return eventsops.NewUpdateEventUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	id, err := uuid.FromString(params.ID.String())
	if err != nil {
		return eventsops.NewUpdateEventBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}

	p := formatter.EventPatchToUpdateParams(params.Body)

	updated, err := h.events.Update(params.HTTPRequest.Context(), id, ownerID, p)
	if err != nil {
		logger.Log().Errorf("update event %s: %s", id, err.Error())
		switch {
		case errors.Is(err, eventsdomain.ErrInvalidInput):
			return eventsops.NewUpdateEventBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, eventsdomain.ErrNotFound):
			return eventsops.NewUpdateEventNotFound().
				WithPayload(DefaultError(http.StatusNotFound, err, nil))
		case errors.Is(err, eventsdomain.ErrNotEditable):
			return eventsops.NewUpdateEventConflict().
				WithPayload(DefaultError(http.StatusConflict, err, nil))
		default:
			return eventsops.NewUpdateEventServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	return eventsops.NewUpdateEventOK().WithPayload(formatter.EventToAPI(updated))
}
