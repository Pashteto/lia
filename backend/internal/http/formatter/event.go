package formatter

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	apiModels "github.com/Pashteto/lia/internal/http/models"
	domainModels "github.com/Pashteto/lia/internal/models"
)

// EventToAPI converts a domain Event to its API representation.
func EventToAPI(event *domainModels.Event) *apiModels.Event {
	if event == nil {
		return nil
	}

	status := event.Status.String()
	out := &apiModels.Event{
		ID:                strfmt.UUID(event.ID.String()),
		Title:             &event.Title,
		Description:       event.Description,
		Status:            &status,
		Format:            event.Format,
		PriceType:         event.PriceType,
		ExternalTicketURL: event.ExternalURL,
		CreatedAt:         strfmt.DateTime(event.CreatedAt),
		UpdatedAt:         strfmt.DateTime(event.UpdatedAt),
	}

	if event.OrganizerID != uuid.Nil {
		out.OrganizerID = strfmt.UUID(event.OrganizerID.String())
	}
	if event.VenueID != uuid.Nil {
		out.VenueID = strfmt.UUID(event.VenueID.String())
	}
	if event.PriceMin != nil {
		out.PriceMin = *event.PriceMin
	}
	if event.PriceMax != nil {
		out.PriceMax = *event.PriceMax
	}

	startsAt := strfmt.DateTime(event.StartsAt)
	out.StartsAt = &startsAt
	if event.EndsAt != nil {
		out.EndsAt = strfmt.DateTime(*event.EndsAt)
	}
	if event.PublishedAt != nil {
		out.PublishedAt = strfmt.DateTime(*event.PublishedAt)
	}

	return out
}

// EventFromAPIInput converts an API EventInput into a domain Event.
// Applies the same defaults as the database (draft / offline / free).
func EventFromAPIInput(in *apiModels.EventInput) (*domainModels.Event, error) {
	if in == nil {
		return nil, fmt.Errorf("event input is required")
	}

	event := &domainModels.Event{
		Description:       in.Description,
		Format:            defaultStr(in.Format, "offline"),
		PriceType:         defaultStr(in.PriceType, "free"),
		ExternalURL:       in.ExternalTicketURL,
	}

	if in.Title != nil {
		event.Title = *in.Title
	}

	status := defaultStr(in.Status, "draft")
	parsedStatus, err := domainModels.EventStatusFromString(status)
	if err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	event.Status = parsedStatus

	if id, ok := parseOptionalUUID(in.OrganizerID); ok {
		event.OrganizerID = id
	}
	if id, ok := parseOptionalUUID(in.VenueID); ok {
		event.VenueID = id
	}

	if in.PriceMin != 0 {
		v := in.PriceMin
		event.PriceMin = &v
	}
	if in.PriceMax != 0 {
		v := in.PriceMax
		event.PriceMax = &v
	}

	if in.StartsAt != nil {
		event.StartsAt = time.Time(*in.StartsAt)
	}
	if !time.Time(in.EndsAt).IsZero() {
		ends := time.Time(in.EndsAt)
		event.EndsAt = &ends
	}

	return event, nil
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func parseOptionalUUID(v strfmt.UUID) (uuid.UUID, bool) {
	if v.String() == "" {
		return uuid.Nil, false
	}
	parsed, err := uuid.FromString(v.String())
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}
