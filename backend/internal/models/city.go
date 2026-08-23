package models

import "slices"

// City slugs. Hardcoded whitelist — the city list is code, only availability
// (app_settings "cities.spb_available") is runtime. Latin lowercase slugs,
// NOT the display codes («МСК») the frontend renders.
// See docs/superpowers/specs/2026-08-23-spb-city-launch-design.md.
const (
	CityMSK = "msk"
	CitySPB = "spb"
	// DefaultCity is assumed when a request carries no city (legacy clients).
	DefaultCity = CityMSK
)

// Cities lists every known city slug in display order.
var Cities = []string{CityMSK, CitySPB}

// ValidCity reports whether s is a known city slug.
func ValidCity(s string) bool {
	return slices.Contains(Cities, s)
}
