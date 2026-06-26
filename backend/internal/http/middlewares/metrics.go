package middlewares

import (
	"regexp"
	"strings"
)

// uuidRe matches a canonical UUID (any case).
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// numericRe matches an all-digits segment.
var numericRe = regexp.MustCompile(`^[0-9]+$`)

// normalizeRoute collapses high-cardinality path segments (UUIDs, numeric ids)
// to ":id" so the Prometheus `route` label set is bounded by route shape, not
// by the number of entities. Used for the metric label only — never for routing.
func normalizeRoute(path string) string {
	if path == "" {
		return path
	}
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if s == "" {
			continue
		}
		if uuidRe.MatchString(s) || numericRe.MatchString(s) {
			segs[i] = ":id"
		}
	}
	return strings.Join(segs, "/")
}
