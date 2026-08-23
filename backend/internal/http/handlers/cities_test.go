package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	citiesops "github.com/Pashteto/lia/internal/http/server/operations/cities"
	"github.com/Pashteto/lia/internal/models"
)

type fakeCitySettings struct {
	spb bool
	err error
}

func (f *fakeCitySettings) Bool(context.Context, string) (bool, error) {
	return f.spb, f.err
}

func listCitiesParams() citiesops.ListCitiesParams {
	return citiesops.ListCitiesParams{
		HTTPRequest: httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil),
	}
}

// handleCities runs the handler and returns availability keyed by city code.
func handleCities(t *testing.T, h *ListCities) map[string]bool {
	t.Helper()
	resp, ok := h.Handle(listCitiesParams()).(*citiesops.ListCitiesOK)
	if !ok {
		t.Fatal("expected ListCitiesOK")
	}
	out := make(map[string]bool, len(resp.Payload))
	for _, c := range resp.Payload {
		out[*c.Code] = *c.Available
	}
	return out
}

func TestListCities_SPBClosedByDefault(t *testing.T) {
	got := handleCities(t, NewListCities(&fakeCitySettings{spb: false}))
	if !got[models.CityMSK] || got[models.CitySPB] {
		t.Fatalf("want msk=true spb=false, got %v", got)
	}
}

func TestListCities_SPBOpensWithTheSetting(t *testing.T) {
	got := handleCities(t, NewListCities(&fakeCitySettings{spb: true}))
	if !got[models.CityMSK] || !got[models.CitySPB] {
		t.Fatalf("want msk=true spb=true, got %v", got)
	}
}

func TestListCities_SettingsErrorDegradesToClosed(t *testing.T) {
	got := handleCities(t, NewListCities(&fakeCitySettings{spb: true, err: errors.New("db down")}))
	if !got[models.CityMSK] || got[models.CitySPB] {
		t.Fatalf("want msk=true spb=false on settings error, got %v", got)
	}
}

func TestListCities_NilSettingsMeansMSKOnly(t *testing.T) {
	got := handleCities(t, NewListCities(nil))
	if !got[models.CityMSK] || got[models.CitySPB] {
		t.Fatalf("want msk=true spb=false with nil settings, got %v", got)
	}
}
