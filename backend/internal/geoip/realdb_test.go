package geoip

import (
	"os"
	"testing"
)

// TestRealDB is a local smoke over an actual GeoLite2-City file; skipped
// unless MMDB points at one (the 62MB database is not in the repo).
func TestRealDB(t *testing.T) {
	path := os.Getenv("MMDB")
	if path == "" {
		t.Skip("no MMDB env")
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	for ip, want := range map[string]string{
		"95.220.0.1":   "msk", // Moscow (MOW)
		"31.134.191.1": "spb", // St Petersburg (SPE)
		"85.140.0.1":   "",    // Kazan — unsupported metro
		"8.8.8.8":      "",    // US
		"not-an-ip":    "",
	} {
		if got := r.CitySlug(ip); got != want {
			t.Errorf("%s: got %q want %q", ip, got, want)
		}
	}
}
