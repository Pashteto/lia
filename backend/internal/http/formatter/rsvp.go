package formatter

import (
	"github.com/go-openapi/strfmt"

	apiModels "github.com/Pashteto/lia/internal/http/models"
	domainModels "github.com/Pashteto/lia/internal/models"
)

// RsvpToAPI converts a domain Rsvp to its API representation.
// When r.Event is non-nil the nested event is mapped via EventToAPI.
func RsvpToAPI(r *domainModels.Rsvp) *apiModels.Rsvp {
	if r == nil {
		return nil
	}
	out := &apiModels.Rsvp{
		ID:                strfmt.UUID(r.ID.String()),
		EventID:           strfmt.UUID(r.EventID.String()),
		UserID:            strfmt.UUID(r.UserID.String()),
		Status:            string(r.Status),
		ApplicationAnswer: r.ApplicationAnswer,
		CreatedAt:         strfmt.DateTime(r.CreatedAt),
	}
	if r.Event != nil {
		out.Event = EventToAPI(r.Event)
	}
	return out
}
