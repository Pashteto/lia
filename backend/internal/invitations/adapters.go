package invitations

import (
	"context"

	"github.com/gofrs/uuid"
)

// eventsAdapter adapts events.Service to EventPort.
type eventsAdapter struct {
	getByID func(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
	details func(ctx context.Context, id string) (EventDetails, error)
}

// NewEventPort builds an EventPort over a getByID closure (title/organizer for
// invite/preview) and a details closure (title/start/organizer-name for the
// "my invitations" list), both typically wrapping events.Service.GetByID.
func NewEventPort(
	getByID func(ctx context.Context, id string) (string, uuid.UUID, error),
	details func(ctx context.Context, id string) (EventDetails, error),
) EventPort {
	return eventsAdapter{getByID: getByID, details: details}
}

func (a eventsAdapter) GetByID(ctx context.Context, id string) (string, uuid.UUID, error) {
	return a.getByID(ctx, id)
}

func (a eventsAdapter) Details(ctx context.Context, id string) (EventDetails, error) {
	return a.details(ctx, id)
}

// rsvpAdapter adapts rsvp.Service to RSVPPort.
type rsvpAdapter struct {
	signUp func(ctx context.Context, eventID, userID uuid.UUID, answer string) error
}

// NewRSVPPort builds an RSVPPort over a signUp closure (typically wrapping
// rsvp.Service.SignUp and discarding the returned *models.Rsvp).
func NewRSVPPort(signUp func(ctx context.Context, eventID, userID uuid.UUID, answer string) error) RSVPPort {
	return rsvpAdapter{signUp: signUp}
}

func (a rsvpAdapter) SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) error {
	return a.signUp(ctx, eventID, userID, answer)
}
