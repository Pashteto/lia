package middlewares

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

// httpMetrics holds the RED metric collectors for HTTP traffic.
type httpMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

// newHTTPMetrics builds and registers the HTTP collectors on reg.
func newHTTPMetrics(reg prometheus.Registerer) *httpMetrics {
	m := &httpMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, normalized route, and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, normalized route, and status.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight)
	return m
}

// middleware returns an alice-compatible middleware that records RED metrics.
// It skips /metrics and /health so scrape and health traffic do not pollute the series.
func (m *httpMetrics) middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			m.inFlight.Inc()
			defer m.inFlight.Dec()

			start := time.Now()
			wrapped := newResponseWriter(w)
			next.ServeHTTP(wrapped, r)

			route := normalizeRoute(r.URL.Path)
			status := strconv.Itoa(wrapped.statusCode)
			m.requests.WithLabelValues(r.Method, route, status).Inc()
			m.duration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
		})
	}
}

// defaultHTTPMetrics registers on the default registry so promhttp.Handler() exposes them.
var defaultHTTPMetrics = newHTTPMetrics(prometheus.DefaultRegisterer)

// Metrics is the alice-compatible HTTP metrics middleware for the server chain.
func Metrics() func(http.Handler) http.Handler {
	return defaultHTTPMetrics.middleware()
}
