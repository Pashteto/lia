package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	venuesops "github.com/Pashteto/lia/internal/http/server/operations/venues"
	"github.com/Pashteto/lia/internal/models"
	venuesdomain "github.com/Pashteto/lia/internal/venues"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListVenues handler searches venues.
type ListVenues struct {
	venues venuesdomain.Service
}

// NewListVenues creates a ListVenues handler.
func NewListVenues(svc venuesdomain.Service) *ListVenues {
	return &ListVenues{venues: svc}
}

// Handle GET /venues.
func (h *ListVenues) Handle(params venuesops.ListVenuesParams) middleware.Responder {
	q := ""
	if params.Q != nil {
		q = *params.Q
	}
	limit := 0
	if params.Limit != nil {
		limit = int(*params.Limit)
	}

	list, err := h.venues.Search(params.HTTPRequest.Context(), q, limit)
	if err != nil {
		logger.Log().Errorf("search venues: %s", err.Error())
		return venuesops.NewListVenuesServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Venue, 0, len(list))
	for _, v := range list {
		payload = append(payload, formatter.VenueToAPI(v))
	}
	return venuesops.NewListVenuesOK().WithPayload(payload)
}

// CreateVenue handler creates (find-or-create) a venue.
type CreateVenue struct {
	venues venuesdomain.Service
}

// NewCreateVenue creates a CreateVenue handler.
func NewCreateVenue(svc venuesdomain.Service) *CreateVenue {
	return &CreateVenue{venues: svc}
}

// Handle POST /venues.
func (h *CreateVenue) Handle(params venuesops.CreateVenueParams) middleware.Responder {
	in := params.Body
	domain := &models.Venue{}
	if in != nil {
		if in.Name != nil {
			domain.Name = *in.Name
		}
		domain.Address = in.Address
		domain.Metro = in.Metro
		domain.District = in.District
	}

	created, err := h.venues.Create(params.HTTPRequest.Context(), domain)
	if err != nil {
		logger.Log().Errorf("create venue: %s", err.Error())
		if errors.Is(err, venuesdomain.ErrInvalidInput) {
			return venuesops.NewCreateVenueBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return venuesops.NewCreateVenueServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	return venuesops.NewCreateVenueCreated().WithPayload(formatter.VenueToAPI(created))
}
