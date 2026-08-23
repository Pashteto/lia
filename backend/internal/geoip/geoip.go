// Package geoip resolves a visitor's IP to a supported city slug using an
// offline MaxMind GeoLite2-City database (env GEOIP_MMDB_PATH; the file is
// bind-mounted on the box, NOT baked into the image). No file → the feature
// is silently off and /cities carries no suggestion.
//
// This product includes GeoLite2 data created by MaxMind, available from
// https://www.maxmind.com.
package geoip

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"

	"github.com/Pashteto/lia/internal/models"
)

// GeoNames ids / ISO-3166-2 subdivision codes for the supported metros. The
// oblast around each city maps to the city — a visitor from Пушкин or Химки
// wants that feed, and a wrong guess is one header click away from fixed.
const (
	geoNameMoscow = 524901
	geoNameSPb    = 498817
)

var subdivisionSlugs = map[string]string{
	"MOW": models.CityMSK, // Москва (город)
	"MOS": models.CityMSK, // Московская область
	"SPE": models.CitySPB, // Санкт-Петербург (город)
	"LEN": models.CitySPB, // Ленинградская область
}

// SlugForLocation maps a GeoLite2 lookup result to a supported city slug.
// Empty string = no supported city (the frontend keeps its default).
// Exposed for tests; the mapping itself never touches the mmdb.
func SlugForLocation(countryISO string, cityGeoNameID uint, subdivisionISOs []string) string {
	if countryISO != "RU" {
		return ""
	}
	switch cityGeoNameID {
	case geoNameMoscow:
		return models.CityMSK
	case geoNameSPb:
		return models.CitySPB
	}
	for _, iso := range subdivisionISOs {
		if slug, ok := subdivisionSlugs[iso]; ok {
			return slug
		}
	}
	return ""
}

// Resolver answers "which supported city is this IP in?".
type Resolver struct {
	db *geoip2.Reader
}

// Open loads the GeoLite2-City database at path.
func Open(path string) (*Resolver, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open geoip db %q: %w", path, err)
	}
	return &Resolver{db: db}, nil
}

// Close releases the underlying mmap.
func (r *Resolver) Close() error { return r.db.Close() }

// CitySlug returns the supported city slug for ip, or "" when the IP is
// unparsable, unknown, or outside the supported metros. Never errors — a
// geo miss must not break /cities.
func (r *Resolver) CitySlug(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	rec, err := r.db.City(parsed)
	if err != nil || rec == nil {
		return ""
	}
	isos := make([]string, 0, len(rec.Subdivisions))
	for _, s := range rec.Subdivisions {
		isos = append(isos, s.IsoCode)
	}
	return SlugForLocation(rec.Country.IsoCode, rec.City.GeoNameID, isos)
}
