package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNormalizeRoute(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"static path unchanged", "/api/v1/events", "/api/v1/events"},
		{"uuid collapsed", "/api/v1/events/2f1c7707-1234-4abc-89ef-0123456789ab", "/api/v1/events/:id"},
		{"uuid mid-path collapsed", "/api/v1/events/2f1c7707-1234-4abc-89ef-0123456789ab/complaints", "/api/v1/events/:id/complaints"},
		{"numeric segment collapsed", "/api/v1/files/12345", "/api/v1/files/:id"},
		{"root unchanged", "/", "/"},
		{"health unchanged", "/health", "/health"},
		{"trailing slash preserved", "/api/v1/events/", "/api/v1/events/"},
		{"multiple ids", "/a/2f1c7707-1234-4abc-89ef-0123456789ab/b/42", "/a/:id/b/:id"},
		{"word with digits not collapsed", "/api/v1/oauth2/token", "/api/v1/oauth2/token"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := normalizeRoute(c.in); got != c.want {
				t.Fatalf("normalizeRoute(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMetricsMiddlewareRecords(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newHTTPMetrics(reg)

	h := m.middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418, distinctive
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/2f1c7707-1234-4abc-89ef-0123456789ab", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	got := testutil.ToFloat64(m.requests.WithLabelValues("GET", "/api/v1/events/:id", "418"))
	if got != 1 {
		t.Fatalf("http_requests_total{GET,/api/v1/events/:id,418} = %v, want 1", got)
	}
	if c := testutil.CollectAndCount(m.duration); c == 0 {
		t.Fatalf("expected at least one duration histogram series, got 0")
	}
}

func TestMetricsMiddlewareSkipsMetricsAndHealth(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newHTTPMetrics(reg)
	h := m.middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))

	for _, p := range []string{"/metrics", "/health"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, p, nil))
	}
	if c := testutil.CollectAndCount(m.requests); c != 0 {
		t.Fatalf("expected /metrics and /health to be skipped, but got %d request series", c)
	}
}
