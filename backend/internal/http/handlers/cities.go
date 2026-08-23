package handlers

import (
	"context"

	"github.com/go-openapi/runtime/middleware"

	apimodels "github.com/Pashteto/lia/internal/http/models"
	citiesops "github.com/Pashteto/lia/internal/http/server/operations/cities"
	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/settings"
	"github.com/Pashteto/lia/pkg/logger"
)

// CitySettings is the slice of the settings service ListCities needs. Kept as
// a local interface so the handler stays testable with a fake.
type CitySettings interface {
	Bool(ctx context.Context, key string) (bool, error)
}

// ListCities handler returns known cities and their availability. msk is
// always available; spb is gated by app_settings cities.spb_available so the
// СПб launch is a settings flip, not a deploy. Degrades to spb=false on a
// settings read failure — availability may lag, the endpoint never 500s.
type ListCities struct {
	settings CitySettings
}

// NewListCities creates a ListCities handler. settings may be nil (no-DB
// mode) — every non-default city then reads as unavailable.
func NewListCities(svc CitySettings) *ListCities {
	return &ListCities{settings: svc}
}

// Handle GET /cities.
func (h *ListCities) Handle(params citiesops.ListCitiesParams) middleware.Responder {
	spb := false
	if h.settings != nil {
		if v, err := h.settings.Bool(params.HTTPRequest.Context(), settings.KeyCitySPBAvailable); err == nil {
			spb = v
		} else {
			logger.Log().Errorf("read %s: %s", settings.KeyCitySPBAvailable, err.Error())
		}
	}
	availability := map[string]bool{
		models.CityMSK: true,
		models.CitySPB: spb,
	}

	payload := make([]*apimodels.City, 0, len(models.Cities))
	for _, c := range models.Cities {
		code := c
		available := availability[c]
		payload = append(payload, &apimodels.City{Code: &code, Available: &available})
	}
	return citiesops.NewListCitiesOK().WithPayload(payload)
}
