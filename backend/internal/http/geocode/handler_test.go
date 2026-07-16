package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	geo "github.com/Pashteto/lia/internal/geocode"
	domain "github.com/Pashteto/lia/internal/models"
)

type fakeGeocoder struct{ results []geo.Result }

func (f fakeGeocoder) Geocode(_ context.Context, _ string) ([]geo.Result, error) {
	return f.results, nil
}

func TestGeocodeRejectsUnauthenticated(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: func(string) (*domain.User, error) { return nil, nil },
		Client:       fakeGeocoder{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/geocode?q=x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestGeocodeReturnsResultsForAuthed(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: func(string) (*domain.User, error) { return &domain.User{}, nil },
		Client:       fakeGeocoder{results: []geo.Result{{Lat: 55.7, Lon: 37.6, Label: "Москва"}}},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/geocode?q=Москва", nil)
	req.Header.Set("Authorization", "Bearer t")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var got []geo.Result
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Label != "Москва" {
		t.Fatalf("got = %+v", got)
	}
}
