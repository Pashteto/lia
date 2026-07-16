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

// Result is one geocoded address, in [lat, lon] terms for frontend consumption.
type Result struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Label string  `json:"label"`
}

// Client calls the Yandex Geocoder HTTP API.
type Client struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

// NewClient builds a geocoder client bound to apiKey (YANDEX_GEOCODER_KEY).
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
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
