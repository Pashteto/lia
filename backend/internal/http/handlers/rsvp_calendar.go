package handlers

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	rsvpops "github.com/Pashteto/lia/internal/http/server/operations/rsvp"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
)

// EventCalendar handles GET /events/{id}/calendar.ics (public, no auth required).
type EventCalendar struct{ rsvp rsvpdomain.Service }

// NewEventCalendar constructs an EventCalendar handler.
func NewEventCalendar(svc rsvpdomain.Service) *EventCalendar { return &EventCalendar{rsvp: svc} }

// Handle returns the event as a downloadable .ics file.
func (h *EventCalendar) Handle(params rsvpops.EventCalendarParams) middleware.Responder {
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewEventCalendarNotFound().
			WithPayload(DefaultError(http.StatusNotFound, err, nil))
	}
	data, err := h.rsvp.CalendarICS(params.HTTPRequest.Context(), eventID)
	if err != nil {
		if errors.Is(err, rsvpdomain.ErrNotFound) {
			return rsvpops.NewEventCalendarNotFound().
				WithPayload(DefaultError(http.StatusNotFound, err, nil))
		}
		return rsvpops.NewEventCalendarUnprocessableEntity().
			WithPayload(DefaultError(http.StatusUnprocessableEntity, err, nil))
	}
	body := bytes.NewReader(data)
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="event.ics"`)
		w.WriteHeader(http.StatusOK)
		_, _ = body.WriteTo(w)
	})
}
