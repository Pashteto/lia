package middlewares

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/Pashteto/lia/config"
)

// idleTTL is how long a client entry survives with no requests before the
// sweeper evicts it. Bounds map growth so a flood of distinct IPs can't leak
// memory; well above any real user's think-time so it never drops a live limiter.
const idleTTL = 15 * time.Minute

// sweepInterval is how often the eviction sweeper runs.
const sweepInterval = 5 * time.Minute

// client holds a single IP's two token buckets and its last-seen time.
type client struct {
	general  *rate.Limiter
	auth     *rate.Limiter
	lastSeen time.Time
}

// RateLimit implements per-IP token-bucket rate limiting with two tiers: a
// generous general bucket for all traffic and a strict bucket for auth
// endpoints (paths containing "/auth/"). Disabled config is a no-op passthrough.
//
// The client IP is read from X-Real-IP / X-Forwarded-For (set by our nginx),
// falling back to RemoteAddr — keying on RemoteAddr alone would collapse every
// request behind the reverse proxy onto one bucket.
func RateLimit(cfg *config.RateLimitConfig) func(http.Handler) http.Handler {
	if cfg == nil || !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	genRate := rate.Limit(cfg.RequestsPerSec)
	genBurst := cfg.Burst
	// Convert the auth allowance from per-minute to per-second for rate.Limit.
	authRate := rate.Limit(cfg.AuthRequestsPerMin / 60.0)
	authBurst := cfg.AuthBurst

	// Background sweeper: evict entries idle longer than idleTTL.
	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-idleTTL)
			mu.Lock()
			for ip, c := range clients {
				if c.lastSeen.Before(cutoff) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			mu.Lock()
			c, ok := clients[ip]
			if !ok {
				c = &client{
					general: rate.NewLimiter(genRate, genBurst),
					auth:    rate.NewLimiter(authRate, authBurst),
				}
				clients[ip] = c
			}
			c.lastSeen = time.Now()
			limiter := c.general
			if isAuthPath(r.URL.Path) {
				limiter = c.auth
			}
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"code":429,"message":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAuthPath reports whether a request path is an auth endpoint that should use
// the strict bucket. Matches on "/auth/" so it holds under both the /api/v1
// prefix (login, register) and the root-level /auth/verify-email path.
func isAuthPath(path string) bool {
	return strings.Contains(path, "/auth/")
}

// clientIP extracts the real client IP, preferring the reverse-proxy headers our
// own nginx sets, then falling back to the connection's RemoteAddr.
func clientIP(r *http.Request) string {
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// The left-most entry is the original client.
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
