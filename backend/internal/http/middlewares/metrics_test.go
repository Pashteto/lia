package middlewares

import "testing"

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
