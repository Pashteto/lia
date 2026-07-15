package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	organizersdomain "github.com/Pashteto/lia/internal/organizers"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListEvents handler returns events, optionally filtered by status / organizer.
type ListEvents struct {
	events     eventsdomain.Service
	organizers organizersdomain.Service
}

// NewListEvents creates a ListEvents handler.
func NewListEvents(svc eventsdomain.Service, orgs organizersdomain.Service) *ListEvents {
	return &ListEvents{events: svc, organizers: orgs}
}

// Handle GET /events.
func (h *ListEvents) Handle(params eventsops.ListEventsParams) middleware.Responder {
	var from, to *time.Time
	if params.From != nil {
		t := time.Time(*params.From)
		from = &t
	}
	if params.To != nil {
		t := time.Time(*params.To)
		to = &t
	}

	// organizer_id (a public organizers.id profile id) restricts to that
	// verified organizer's events. Unknown / unverified / malformed id, or no
	// organizers service (no-DB mode) → empty list, no error, no leak.
	var organizerOwner *uuid.UUID
	if params.OrganizerID != nil {
		if h.organizers == nil {
			return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
		}
		// params.OrganizerID is already validated as a uuid by the go-swagger
		// binding layer (a malformed value is rejected with 400 before this
		// handler runs), so this parse effectively never fails. Be honest about
		// the contract anyway: a bad uuid is a 400, not an empty list.
		profileID, perr := uuid.FromString(params.OrganizerID.String())
		if perr != nil {
			return eventsops.NewListEventsBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, perr, nil))
		}
		org, oerr := h.organizers.GetByID(params.HTTPRequest.Context(), profileID)
		if oerr != nil {
			if errors.Is(oerr, organizersdomain.ErrNotFound) {
				// Unknown profile id — no leak, just an empty list.
				return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
			}
			// A real lookup failure (DB down, timeout) must not masquerade as "no events".
			logger.Log().Errorf("resolve organizer %s: %s", profileID, oerr.Error())
			return eventsops.NewListEventsServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, oerr, nil))
		}
		if org == nil || org.VerificationStatus != "verified" {
			// Exists-but-unverified or nil → empty list (no leak of non-verified profiles).
			return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
		}
		organizerOwner = &org.OwnerUserID
	}

	list, err := h.events.List(params.HTTPRequest.Context(), "published", from, to, organizerOwner)
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

// GetEventByID handler returns a single event by UUID. Non-published events are
// visible only to their owner; everyone else gets 404 (existence not leaked).
type GetEventByID struct {
	events    eventsdomain.Service
	rsvp      rsvpdomain.Service // optional; nil → my_rsvp_status stays ""
	checkAuth func(string) (*apimodels.User, error)
}

// NewGetEventByID creates a GetEventByID handler. checkAuth resolves the caller
// from the Authorization header; it may be nil (treated as always-anonymous).
// rsvp may be nil (no-DB mode), in which case my_rsvp_status is left empty.
func NewGetEventByID(
	svc eventsdomain.Service,
	rsvp rsvpdomain.Service,
	checkAuth func(string) (*apimodels.User, error),
) *GetEventByID {
	return &GetEventByID{events: svc, rsvp: rsvp, checkAuth: checkAuth}
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

	// Non-published events are visible only to the owner.
	if event.Status.String() != "published" && !h.callerOwns(params, event.OrganizerID.String()) {
		return eventsops.NewGetEventByIDNotFound().
			WithPayload(DefaultError(http.StatusNotFound, errors.New("event not found"), nil))
	}

	// Populate my_rsvp_status for the authenticated caller so the detail page
	// renders the correct join/apply state on reload (design-review R4).
	if h.rsvp != nil && h.checkAuth != nil {
		if u, err := h.checkAuth(params.HTTPRequest.Header.Get("Authorization")); err == nil && u != nil {
			if uid, err := uuid.FromString(u.UUID.String()); err == nil {
				if eid, err := uuid.FromString(params.ID.String()); err == nil {
					if st, err := h.rsvp.StatusForUser(params.HTTPRequest.Context(), eid, uid); err == nil {
						event.MyRsvpStatus = string(st)
					}
				}
			}
		}
	}

	return eventsops.NewGetEventByIDOK().WithPayload(formatter.EventToAPI(event))
}

// callerOwns reports whether the (optional) authenticated caller owns the event.
// Any auth failure is treated as anonymous (not the owner).
func (h *GetEventByID) callerOwns(params eventsops.GetEventByIDParams, organizerID string) bool {
	if h.checkAuth == nil {
		return false
	}
	u, err := h.checkAuth(params.HTTPRequest.Header.Get("Authorization"))
	if err != nil || u == nil {
		return false
	}
	return u.UUID.String() == organizerID
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
	if !IsVerified(principal) {
		return UnverifiedResponder()
	}

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
		switch {
		case errors.Is(err, eventsdomain.ErrInvalidInput):
			return eventsops.NewCreateEventBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, eventsdomain.ErrQuotaExceeded):
			// NOTE: "10 событий в месяц" is intentionally hardcoded per spec.
			// Keep in sync with the EVENTS_MONTHLY_LIMIT config value.
			return eventsops.NewCreateEventTooManyRequests().
				WithPayload(DefaultError(http.StatusTooManyRequests, errors.New("Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."), nil))
		default:
			return eventsops.NewCreateEventServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	return eventsops.NewCreateEventCreated().WithPayload(formatter.EventToAPI(event))
}

// ListMyEvents handler returns all events (any status, including drafts) created
// by the authenticated user.
type ListMyEvents struct {
	events eventsdomain.Service
}

// NewListMyEvents constructs a ListMyEvents handler.
func NewListMyEvents(svc eventsdomain.Service) *ListMyEvents {
	return &ListMyEvents{events: svc}
}

// Handle returns the caller's own events.
func (h *ListMyEvents) Handle(params eventsops.ListMyEventsParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return eventsops.NewListMyEventsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	id, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return eventsops.NewListMyEventsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}

	list, err := h.events.ListByOrganizer(params.HTTPRequest.Context(), id)
	if err != nil {
		logger.Log().Errorf("list my events: %s", err.Error())
		return eventsops.NewListMyEventsServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Event, 0, len(list))
	for _, e := range list {
		payload = append(payload, formatter.EventToAPI(e))
	}

	return eventsops.NewListMyEventsOK().WithPayload(payload)
}
