package formatter

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
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
		// Only emit `verified` when true so omitempty keeps non-verified
		// organizer payloads byte-identical to before (presence ⇒ verified).
		if event.Organizer.Verified {
			v := true
			org.Verified = &v
		}
		if event.Organizer.ProfileID != uuid.Nil {
			pid := event.Organizer.ProfileID.String()
			org.ProfileID = &pid
		}
		out.Organizer = org
	}

	out.SignupMode = event.SignupMode
	out.CuratorQuestion = event.CuratorQuestion
	out.ExternalRegistrationURL = event.ExternalRegistrationURL
	if event.Capacity != nil {
		c := int64(*event.Capacity)
		out.Capacity = &c
	}
	if event.SeatsRemaining != nil {
		s := int64(*event.SeatsRemaining)
		out.SeatsRemaining = &s
	}
	out.MyRsvpStatus = event.MyRsvpStatus

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
// Applies sensible defaults (draft / offline / free): a new event with no
// explicit status defaults to draft so clients can review it before publishing.
// Clients can explicitly pass "published" to make it immediately visible.
func EventFromAPIInput(in *apiModels.EventInput) (*domainModels.Event, error) {
	if in == nil {
		return nil, fmt.Errorf("event input is required")
	}

	event := &domainModels.Event{
		Description:             in.Description,
		Format:                  defaultStr(in.Format, "offline"),
		PriceType:               defaultStr(in.PriceType, "free"),
		ExternalURL:             in.ExternalTicketURL,
		SignupMode:              defaultStr(in.SignupMode, "open"),
		CuratorQuestion:         in.CuratorQuestion,
		ExternalRegistrationURL: in.ExternalRegistrationURL,
	}
	if in.Capacity != nil {
		c := int(*in.Capacity)
		event.Capacity = &c
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

// EventPatchToUpdateParams converts an API EventPatch into the domain
// UpdateParams. A zero/empty API value maps to a nil pointer ("preserve");
// category_ids maps to nil when absent (preserve) and to a slice when present
// (replace). Clearing a field to empty is not supported via PATCH.
func EventPatchToUpdateParams(in *apiModels.EventPatch) eventsdomain.UpdateParams {
	var p eventsdomain.UpdateParams
	if in == nil {
		return p
	}
	if in.Title != "" {
		v := in.Title
		p.Title = &v
	}
	if in.Description != "" {
		v := in.Description
		p.Description = &v
	}
	if in.Format != "" {
		v := in.Format
		p.Format = &v
	}
	if in.PriceType != "" {
		v := in.PriceType
		p.PriceType = &v
	}
	if in.PriceMin != 0 {
		v := in.PriceMin
		p.PriceMin = &v
	}
	if in.PriceMax != 0 {
		v := in.PriceMax
		p.PriceMax = &v
	}
	if in.ExternalTicketURL != "" {
		v := in.ExternalTicketURL
		p.ExternalURL = &v
	}
	if id, ok := parseOptionalUUID(in.VenueID); ok {
		p.VenueID = &id
	}
	if in.CoverFileID != nil {
		if id, ok := parseOptionalUUID(*in.CoverFileID); ok {
			p.CoverFileID = &id
		}
	}
	if !time.Time(in.StartsAt).IsZero() {
		t := time.Time(in.StartsAt)
		p.StartsAt = &t
	}
	if !time.Time(in.EndsAt).IsZero() {
		t := time.Time(in.EndsAt)
		p.EndsAt = &t
	}
	if in.Status != "" {
		v := in.Status
		p.Status = &v
	}
	if len(in.CategoryIds) > 0 {
		ids := make([]uuid.UUID, 0, len(in.CategoryIds))
		for _, raw := range in.CategoryIds {
			if parsed, err := uuid.FromString(raw.String()); err == nil {
				ids = append(ids, parsed)
			}
		}
		p.CategoryIDs = ids
	}
	if in.SignupMode != "" {
		v := in.SignupMode
		p.SignupMode = &v
	}
	if in.CuratorQuestion != "" {
		v := in.CuratorQuestion
		p.CuratorQuestion = &v
	}
	if in.ExternalRegistrationURL != "" {
		v := in.ExternalRegistrationURL
		p.ExternalRegistrationURL = &v
	}
	// capacity is set at create; PATCH support deferred
	return p
}
