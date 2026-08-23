package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	domainmodels "github.com/Pashteto/lia/internal/models"
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

// canSeeEvent reports whether a caller may load an event's detail page.
//
// Published is public. A cancelled event stays readable for anyone who is
// already registered — cancelling must not silently delete the page out from
// under the people who planned to come; they need to land on it and read
// "отменено". Everything else (draft, moderation, rejected) is owner-only, and
// non-owners get 404 rather than 403 so existence is never leaked.
func canSeeEvent(status string, isOwner, hasRsvp bool) bool {
	if status == "published" || isOwner {
		return true
	}
	return status == "cancelled" && hasRsvp
}

// GetEventByID handler returns a single event by UUID. Non-published events are
// visible only to their owner (and, once cancelled, to registered attendees);
// everyone else gets 404 (existence not leaked).
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

	// Resolve the caller once: both the visibility gate and my_rsvp_status need
	// it, and a cancelled event's gate depends on the RSVP lookup.
	//
	// my_rsvp_status also feeds the detail page's join/apply state on reload
	// (design-review R4).
	var caller *apimodels.User
	if h.checkAuth != nil {
		if u, err := h.checkAuth(params.HTTPRequest.Header.Get("Authorization")); err == nil {
			caller = u
		}
	}
	if caller != nil && h.rsvp != nil {
		if uid, err := uuid.FromString(caller.UUID.String()); err == nil {
			if eid, err := uuid.FromString(params.ID.String()); err == nil {
				if st, err := h.rsvp.StatusForUser(params.HTTPRequest.Context(), eid, uid); err == nil {
					event.MyRsvpStatus = string(st)
				}
			}
		}
	}

	isOwner := caller != nil && caller.UUID.String() == event.OrganizerID.String()
	registered := domainmodels.RsvpStatus(event.MyRsvpStatus).IsActive()
	if !canSeeEvent(event.Status.String(), isOwner, registered) {
		return eventsops.NewGetEventByIDNotFound().
			WithPayload(DefaultError(http.StatusNotFound, errors.New("event not found"), nil))
	}

	payload := formatter.EventToAPI(event)
	payload.IsOwner = isOwner
	return eventsops.NewGetEventByIDOK().WithPayload(payload)
}

// quotaMessage renders a rejected create in Russian. The daily cap is
// configurable and can be raised per organizer, so its message is built from
// the actual number rather than hardcoded; the monthly cap keeps its
// spec-mandated wording.
func quotaMessage(err error) string {
	var q *eventsdomain.QuotaError
	if errors.As(err, &q) && q.Period == "day" {
		return fmt.Sprintf(
			"Достигнут дневной лимит: %s. Лимит обновится завтра.",
			plural(q.Limit, "событие", "события", "событий"),
		)
	}
	// NOTE: "10 событий в месяц" is intentionally hardcoded per spec.
	// Keep in sync with the EVENTS_MONTHLY_LIMIT config value.
	return "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
}

// plural renders a Russian count with the right noun form ("1 событие",
// "3 события", "5 событий").
func plural(n int, one, few, many string) string {
	mod100 := n % 100
	mod10 := n % 10
	switch {
	case mod100 >= 11 && mod100 <= 14:
		return fmt.Sprintf("%d %s", n, many)
	case mod10 == 1:
		return fmt.Sprintf("%d %s", n, one)
	case mod10 >= 2 && mod10 <= 4:
		return fmt.Sprintf("%d %s", n, few)
	default:
		return fmt.Sprintf("%d %s", n, many)
	}
}

// CreateEvent handler creates a new event.
type CreateEvent struct {
	events     eventsdomain.Service
	organizers organizersdomain.Service // optional; nil → no profile is created
}

// NewCreateEvent creates a CreateEvent handler. orgs may be nil (no-DB mode),
// in which case creating an event does not create an organizer profile.
func NewCreateEvent(svc eventsdomain.Service, orgs organizersdomain.Service) *CreateEvent {
	return &CreateEvent{events: svc, organizers: orgs}
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
			return eventsops.NewCreateEventTooManyRequests().
				WithPayload(DefaultError(http.StatusTooManyRequests, errors.New(quotaMessage(err)), nil))
		default:
			return eventsops.NewCreateEventServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	// Creating an event is what makes somebody an organizer, so the profile is
	// minted here rather than waiting for them to find the profile form. Best
	// effort: the event is already committed, and a registry row is not worth
	// failing the request the user actually made.
	if h.organizers != nil && principal != nil && event.OrganizerID != uuid.Nil {
		name := ""
		if principal.Name != nil {
			name = *principal.Name
		}
		if _, err := h.organizers.EnsureForOwner(params.HTTPRequest.Context(), event.OrganizerID, name); err != nil {
			logger.Log().Errorf("ensure organizer profile for %s: %s", event.OrganizerID, err.Error())
		}
	}

	return eventsops.NewCreateEventCreated().WithPayload(formatter.EventToAPI(event))
}

// ListMyEvents handler returns all events (any status, including drafts) created
// by the authenticated user.
type ListMyEvents struct {
	events eventsdomain.Service
	rsvp   rsvpdomain.Service // optional: pending-application counters
}

// NewListMyEvents constructs a ListMyEvents handler. rsvp may be nil — the
// list then simply omits pending_applications_count.
func NewListMyEvents(svc eventsdomain.Service, rsvp rsvpdomain.Service) *ListMyEvents {
	return &ListMyEvents{events: svc, rsvp: rsvp}
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

	// Pending-application counters for the «Заявки · N» badge. Best-effort:
	// a failed count must not break the list.
	var pending map[uuid.UUID]int
	if h.rsvp != nil {
		ids := make([]uuid.UUID, 0, len(list))
		for _, e := range list {
			ids = append(ids, e.ID)
		}
		if counts, err := h.rsvp.CountPendingApplications(params.HTTPRequest.Context(), ids); err == nil {
			pending = counts
		} else {
			logger.Log().Errorf("count pending applications: %s", err.Error())
		}
	}

	payload := make([]*apimodels.Event, 0, len(list))
	for _, e := range list {
		out := formatter.EventToAPI(e)
		out.IsOwner = true // /events/mine is owner-only by construction
		if n, ok := pending[e.ID]; ok {
			out.PendingApplicationsCount = int64(n)
		}
		payload = append(payload, out)
	}

	return eventsops.NewListMyEventsOK().WithPayload(payload)
}
