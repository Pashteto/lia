package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeocodeParsesFeatureMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("geocode"); got != "Москва" {
			t.Errorf("geocode param = %q, want Москва", got)
		}
		if got := r.URL.Query().Get("apikey"); got != "test-key" {
			t.Errorf("apikey = %q, want test-key", got)
		}
		if got := r.URL.Query().Get("ll"); got != "37.617700,55.755800" {
			t.Errorf("ll = %q, want Moscow center", got)
		}
		if got := r.URL.Query().Get("spn"); got != "0.7,0.5" {
			t.Errorf("spn = %q, want Moscow span", got)
		}
		if got := r.URL.Query().Get("rspn"); got != "" {
			t.Errorf("rspn = %q, want empty (soft bias, not hard restrict)", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"GeoObjectCollection":{"featureMember":[
			{"GeoObject":{"metaDataProperty":{"GeocoderMetaData":{"text":"Россия, Москва"}},"Point":{"pos":"37.617635 55.755814"}}}
		]}}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.endpoint = srv.URL

	got, err := c.Geocode(context.Background(), "Москва")
	if err != nil {
		t.Fatalf("Geocode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Label != "Россия, Москва" {
		t.Errorf("label = %q", got[0].Label)
	}
	if got[0].Lat != 55.755814 || got[0].Lon != 37.617635 {
		t.Errorf("coords = %v,%v want 55.755814,37.617635", got[0].Lat, got[0].Lon)
	}
}

func TestGeocodeBlankQuerySkipsRequest(t *testing.T) {
	c := NewClient("k")
	got, err := c.Geocode(context.Background(), "   ")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
