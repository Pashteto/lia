package formatter

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	apiModels "github.com/Pashteto/lia/internal/http/models"
	domainModels "github.com/Pashteto/lia/internal/models"
)

// VenueToAPI converts a domain Venue to its API representation.
func VenueToAPI(v *domainModels.Venue) *apiModels.Venue {
	if v == nil {
		return nil
	}
	name := v.Name
	out := &apiModels.Venue{
		ID:       strfmt.UUID(v.ID.String()),
		Name:     &name,
		Address:  v.Address,
		Metro:    v.Metro,
		District: v.District,
	}
	// Lat/Lon are *float64 in both the domain model and the generated API model.
	// Assign pointers directly so coordless venues omit the fields (omitempty).
	out.Lat = v.Lat
	out.Lon = v.Lon
	return out
}

// CategoryToAPI converts a domain Category to its API representation.
func CategoryToAPI(c *domainModels.Category) *apiModels.Category {
	if c == nil {
		return nil
	}
	id := strfmt.UUID(c.ID.String())
	return &apiModels.Category{
		ID:    &id,
		Slug:  &c.Slug,
		Label: &c.Label,
	}
}

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

	out.Categories = make([]*apiModels.Category, 0, len(event.Categories))
	for _, c := range event.Categories {
		out.Categories = append(out.Categories, CategoryToAPI(c))
	}

	out.Venue = VenueToAPI(event.Venue)

	if event.CoverURL != "" {
		out.CoverURL = &event.CoverURL
	}

	if event.Organizer != nil {
		org := &apiModels.Organizer{
			UUID: strfmt.UUID(event.Organizer.UUID.String()),
			Name: event.Organizer.Name,
		}
		if event.Organizer.AvatarURL != "" {
			org.AvatarURL = &event.Organizer.AvatarURL
		}
		out.Organizer = org
	}

	return out
}

// EventToAPIWithDistance is EventToAPI plus the nearby distance in meters.
// The generated Event.Distancem field is *float64 (x-nullable: true).
func EventToAPIWithDistance(e *domainModels.Event, distanceM float64) *apiModels.Event {
	out := EventToAPI(e)
	d := distanceM
	out.Distancem = &d
	return out
}

// EventFromAPIInput converts an API EventInput into a domain Event.
// Applies sensible defaults (published / offline / free): a new event with no
// explicit status is published so it is immediately visible in the discovery
// feed (which lists status=published). Clients can pass "draft" to hide it.
func EventFromAPIInput(in *apiModels.EventInput) (*domainModels.Event, error) {
	if in == nil {
		return nil, fmt.Errorf("event input is required")
	}

	event := &domainModels.Event{
		Description: in.Description,
		Format:      defaultStr(in.Format, "offline"),
		PriceType:   defaultStr(in.PriceType, "free"),
		ExternalURL: in.ExternalTicketURL,
	}

	if in.Title != nil {
		event.Title = *in.Title
	}

	status := defaultStr(in.Status, "published")
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
	if in.CoverFileID != nil {
		if id, ok := parseOptionalUUID(*in.CoverFileID); ok {
			event.CoverFileID = id
		}
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

	for _, raw := range in.CategoryIds {
		if parsed, err := uuid.FromString(raw.String()); err == nil {
			event.CategoryIDs = append(event.CategoryIDs, parsed)
		}
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
