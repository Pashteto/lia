package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/http/formatter"
	venuesops "github.com/Pashteto/lia/internal/http/server/operations/venues"
	venuesdomain "github.com/Pashteto/lia/internal/venues"
	"github.com/Pashteto/lia/pkg/logger"
)

// UpdateVenue handler updates a venue's fields and/or coordinates.
type UpdateVenue struct {
	venues venuesdomain.Service
}

// NewUpdateVenue creates an UpdateVenue handler.
func NewUpdateVenue(svc venuesdomain.Service) *UpdateVenue { return &UpdateVenue{venues: svc} }

// Handle PATCH /venues/{id}.
func (h *UpdateVenue) Handle(params venuesops.UpdateVenueParams) middleware.Responder {
	id, err := uuid.FromString(params.ID.String())
	if err != nil {
		return venuesops.NewUpdateVenueBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}
	in := params.Body
	var name, address, metro, district string
	var lat, lon *float64
	if in != nil {
		if in.Name != nil {
			name = *in.Name
		}
		address = in.Address
		metro = in.Metro
		district = in.District
		// Always forward coords as pointers so the service can apply ValidateCoords.
		// Zero values (0.0, 0.0) are a valid coordinate pair and will be persisted.
		lat = &in.Lat
		lon = &in.Lon
	}
	updated, err := h.venues.Update(params.HTTPRequest.Context(), id, name, address, metro, district, lat, lon)
	if err != nil {
		logger.Log().Errorf("update venue: %s", err.Error())
		if errors.Is(err, venuesdomain.ErrInvalidInput) {
			return venuesops.NewUpdateVenueBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return venuesops.NewUpdateVenueServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}
	return venuesops.NewUpdateVenueOK().WithPayload(formatter.VenueToAPI(updated))
}
