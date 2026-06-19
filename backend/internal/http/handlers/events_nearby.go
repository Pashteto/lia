package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	"github.com/Pashteto/lia/pkg/logger"
)

// NearbyEvents handler returns published events nearest to the given coordinates.
type NearbyEvents struct {
	events eventsdomain.Service
}

// NewNearbyEvents creates a NearbyEvents handler.
func NewNearbyEvents(svc eventsdomain.Service) *NearbyEvents { return &NearbyEvents{events: svc} }

// Handle GET /events/nearby.
func (h *NearbyEvents) Handle(params eventsops.NearbyEventsParams) middleware.Responder {
	lat, lon := params.Lat, params.Lon
	limit := 0
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	res, err := h.events.Nearby(params.HTTPRequest.Context(), &lat, &lon, limit)
	if err != nil {
		logger.Log().Errorf("nearby events: %s", err.Error())
		if errors.Is(err, eventsdomain.ErrInvalidInput) {
			return eventsops.NewNearbyEventsBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return eventsops.NewNearbyEventsServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}
	payload := make([]*apimodels.Event, 0, len(res))
	for _, r := range res {
		payload = append(payload, formatter.EventToAPIWithDistance(r.Event, r.DistanceM))
	}
	return eventsops.NewNearbyEventsOK().WithPayload(payload)
}
