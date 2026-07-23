// Package geocode is the backend proxy client for the Yandex Geocoder HTTP API.
// It is the first outbound HTTP client in the backend — there is no prior
// pattern to mirror.
package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultEndpoint = "https://geocode-maps.yandex.ru/1.x/"

// defaultPlacesEndpoint is the Yandex Places (Search) API v1 base URL.
const defaultPlacesEndpoint = "https://search-maps.yandex.ru/v1/"

// Moscow viewport bias. Presence launches in Moscow, so we hint the geocoder to
// rank results near the city center first. `ll` is the center (lon,lat) and
// `spn` the span (Δlon,Δlat) covering greater Moscow. We deliberately omit
// `rspn=1` (hard restrict) so a venue genuinely outside Moscow still resolves —
// this is a soft ranking bias, not a filter.
const (
	moscowLL  = "37.617700,55.755800"
	moscowSpn = "0.7,0.5"
)

// Result is one geocoded address, in [lat, lon] terms for frontend consumption.
type Result struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Label string  `json:"label"`
}

// Client calls the Yandex Geocoder HTTP API.
type Client struct {
	apiKey         string
	endpoint       string
	placesKey      string
	placesEndpoint string
	http           *http.Client
}

// NewClient builds a geocoder client bound to apiKey (YANDEX_GEOCODER_KEY).
// The Places (Search) API is disabled until WithPlacesKey is called.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:         apiKey,
		endpoint:       defaultEndpoint,
		placesEndpoint: defaultPlacesEndpoint,
		http:           &http.Client{Timeout: 5 * time.Second},
	}
}

// WithPlacesKey enables the Places (Search) API with its own key (YANDEX_PLACES_KEY)
// and returns c for chaining.
func (c *Client) WithPlacesKey(k string) *Client {
	c.placesKey = k
	return c
}

// yandexResponse mirrors the subset of the Yandex Geocoder 1.x JSON we read.
type yandexResponse struct {
	Response struct {
		GeoObjectCollection struct {
			FeatureMember []struct {
				GeoObject struct {
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Text string `json:"text"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
					Point struct {
						Pos string `json:"pos"` // "lon lat"
					} `json:"Point"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

// Geocode returns up to 5 matches for q. A blank query yields an empty slice
// without an HTTP call.
func (c *Client) Geocode(ctx context.Context, q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []Result{}, nil
	}
	if c.apiKey == "" {
		return nil, errors.New("geocode: api key not configured")
	}
	params := url.Values{
		"apikey":  {c.apiKey},
		"geocode": {q},
		"format":  {"json"},
		"lang":    {"ru_RU"},
		"results": {"5"},
		"ll":      {moscowLL},
		"spn":     {moscowSpn},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocode: upstream status %d", res.StatusCode)
	}
	var yr yandexResponse
	if err := json.NewDecoder(res.Body).Decode(&yr); err != nil {
		return nil, err
	}
	members := yr.Response.GeoObjectCollection.FeatureMember
	out := make([]Result, 0, len(members))
	for _, m := range members {
		lon, lat, ok := parsePos(m.GeoObject.Point.Pos)
		if !ok {
			continue
		}
		out = append(out, Result{
			Lat:   lat,
			Lon:   lon,
			Label: m.GeoObject.MetaDataProperty.GeocoderMetaData.Text,
		})
	}
	return out, nil
}

// placesResponse mirrors the subset of the Yandex Search API v1 GeoJSON we read.
type placesResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat]
		} `json:"geometry"`
		Properties struct {
			CompanyMetaData struct {
				Name    string `json:"name"`
				Address string `json:"address"`
			} `json:"CompanyMetaData"`
		} `json:"properties"`
	} `json:"features"`
}

// SearchPlaces resolves a venue/organization NAME (e.g. "Дом Радио") to points,
// biased to Moscow. Blank query → empty slice, no HTTP call.
func (c *Client) SearchPlaces(ctx context.Context, q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []Result{}, nil
	}
	if c.placesKey == "" {
		return nil, errors.New("places: api key not configured")
	}
	params := url.Values{
		"apikey":  {c.placesKey},
		"text":    {q},
		"type":    {"biz"},
		"lang":    {"ru_RU"},
		"results": {"5"},
		"ll":      {moscowLL},
		"spn":     {moscowSpn},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.placesEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("places: upstream status %d", res.StatusCode)
	}
	var pr placesResponse
	if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(pr.Features))
	for _, f := range pr.Features {
		if len(f.Geometry.Coordinates) != 2 {
			continue
		}
		label := f.Properties.CompanyMetaData.Name
		if a := f.Properties.CompanyMetaData.Address; a != "" {
			label = label + " · " + a
		}
		out = append(out, Result{
			Lon:   f.Geometry.Coordinates[0],
			Lat:   f.Geometry.Coordinates[1],
			Label: label,
		})
	}
	return out, nil
}

// parsePos parses a Yandex "lon lat" string into (lon, lat).
func parsePos(pos string) (lon, lat float64, ok bool) {
	parts := strings.Fields(pos)
	if len(parts) != 2 {
		return 0, 0, false
	}
	lon, err1 := strconv.ParseFloat(parts[0], 64)
	lat, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return lon, lat, true
}
