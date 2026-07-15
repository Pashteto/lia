package invitations

import (
	"context"

	"github.com/gofrs/uuid"
)

// eventsAdapter adapts events.Service to EventPort.
type eventsAdapter struct {
	getByID func(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
}

// NewEventPort builds an EventPort over a getByID closure (typically wrapping
// events.Service.GetByID and unpacking the title/organizer fields).
func NewEventPort(getByID func(ctx context.Context, id string) (string, uuid.UUID, error)) EventPort {
	return eventsAdapter{getByID: getByID}
}

func (a eventsAdapter) GetByID(ctx context.Context, id string) (string, uuid.UUID, error) {
	return a.getByID(ctx, id)
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
