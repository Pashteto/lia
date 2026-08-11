package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pashteto/lia/config"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func do(h http.Handler, path, ip string) int {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if ip != "" {
		req.Header.Set("X-Real-IP", ip)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimit_DisabledIsPassthrough(t *testing.T) {
	h := RateLimit(&config.RateLimitConfig{Enabled: false})(okHandler())
	for i := 0; i < 100; i++ {
		if code := do(h, "/api/v1/auth/login", "1.2.3.4"); code != http.StatusOK {
			t.Fatalf("disabled limiter blocked request %d: %d", i, code)
		}
	}
}

func TestRateLimit_AuthBucketIsStrict(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		RequestsPerSec:     1000,
		Burst:              1000,
		AuthRequestsPerMin: 5,
		AuthBurst:          3, // only 3 immediate auth attempts
	}
	h := RateLimit(cfg)(okHandler())

	// First 3 auth requests pass, the 4th is throttled.
	for i := 0; i < 3; i++ {
		if code := do(h, "/api/v1/auth/login", "9.9.9.9"); code != http.StatusOK {
			t.Fatalf("auth request %d should pass, got %d", i, code)
		}
	}
	if code := do(h, "/api/v1/auth/login", "9.9.9.9"); code != http.StatusTooManyRequests {
		t.Fatalf("4th auth request should be 429, got %d", code)
	}
}

func TestRateLimit_GeneralUnaffectedByAuthBurst(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		RequestsPerSec:     100,
		Burst:              100,
		AuthRequestsPerMin: 5,
		AuthBurst:          1,
	}
	h := RateLimit(cfg)(okHandler())

	// Exhaust the auth bucket for this IP.
	_ = do(h, "/api/v1/auth/login", "8.8.8.8")
	if code := do(h, "/api/v1/auth/login", "8.8.8.8"); code != http.StatusTooManyRequests {
		t.Fatalf("auth bucket should be exhausted, got %d", code)
	}
	// General endpoints for the same IP still pass — separate bucket.
	if code := do(h, "/api/v1/events", "8.8.8.8"); code != http.StatusOK {
		t.Fatalf("general request should pass despite auth throttle, got %d", code)
	}
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		RequestsPerSec:     100,
		Burst:              100,
		AuthRequestsPerMin: 5,
		AuthBurst:          1,
	}
	h := RateLimit(cfg)(okHandler())

	// IP A exhausts its auth bucket.
	_ = do(h, "/api/v1/auth/login", "10.0.0.1")
	if code := do(h, "/api/v1/auth/login", "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("IP A should be throttled, got %d", code)
	}
	// IP B is unaffected.
	if code := do(h, "/api/v1/auth/login", "10.0.0.2"); code != http.StatusOK {
		t.Fatalf("IP B should not be throttled by IP A, got %d", code)
	}
}

func TestClientIP_PrefersRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5555" // the proxy
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 127.0.0.1")
	req.Header.Set("X-Real-IP", "203.0.113.7")
	if ip := clientIP(req); ip != "203.0.113.7" {
		t.Fatalf("expected real client IP, got %q", ip)
	}
}

func TestClientIP_FallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.4:4321"
	if ip := clientIP(req); ip != "198.51.100.4" {
		t.Fatalf("expected host from RemoteAddr, got %q", ip)
	}
}
