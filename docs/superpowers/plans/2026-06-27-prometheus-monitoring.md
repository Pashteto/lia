# Simple Prometheus Monitoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Instrument the Lia Go backend with Prometheus HTTP + runtime metrics, scrape them (plus host metrics via node_exporter) from a lightweight Prometheus on the same VDS, with hard-capped TSDB size and hard-capped container logs so nothing grows unboundedly on the 1.9 GB box.

**Architecture:** A new `Metrics()` middleware in the existing `alice` chain records RED metrics labelled by a cardinality-bounded normalized route; `/metrics` is mounted ahead of the go-swagger mux in the existing pre-swagger dispatcher and blocked at nginx. A new additive `docker-compose.monitoring.yml` overlay runs Prometheus (loopback-only, retention-capped) + node_exporter (internal-network-only) on the existing compose project network. Per-service `logging:` blocks cap all container logs.

**Tech Stack:** Go 1.24, go-swagger generated mux, `justinas/alice` middleware, `github.com/prometheus/client_golang`, `prom/prometheus` + `prom/node-exporter` containers, docker compose, nginx.

## Global Constraints

- Go version floor: **`go 1.24.0`** (already in `go.mod`); do not bump.
- Module path: **`github.com/Pashteto/lia`** — all internal imports use this prefix.
- New Go dependency: **`github.com/prometheus/client_golang`** only. No other new deps.
- Metric label `route` MUST come from `normalizeRoute(...)`, **never** the raw URL path (cardinality safeguard).
- The `Metrics()` middleware MUST NOT record `/metrics` or `/health` requests.
- Prometheus UI published on **`127.0.0.1:9091:9090`** only (loopback). node_exporter has **no host port**.
- TSDB bounded by **both** `--storage.tsdb.retention.time=15d` **and** `--storage.tsdb.retention.size=512MB`.
- Container logs: `logging:` driver `json-file`, `max-size: "10m"`, `max-file: "3"` on every service touched.
- Container RAM ceilings: prometheus `mem_limit: 256m`, node_exporter `mem_limit: 64m`.
- Reuse the existing `responseWriter`/`newResponseWriter` in `internal/http/middlewares/logger.go` — do not define a second status-capturing wrapper.
- Bringing up `app` in prod ALWAYS uses the full 3 compose files (`docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml`); the monitoring overlay is a 4th file added on top.

---

### Task 1: Route normalizer (cardinality safeguard)

Pure function, no new dependency. Collapses UUIDs and numeric path segments so the `route` metric label stays bounded.

**Files:**
- Create: `backend/internal/http/middlewares/metrics.go`
- Test: `backend/internal/http/middlewares/metrics_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func normalizeRoute(path string) string` (unexported, same `middlewares` package) — returns the path with each UUID or all-numeric segment replaced by `:id`.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/http/middlewares/metrics_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/middlewares/ -run TestNormalizeRoute -v`
Expected: FAIL — build error `undefined: normalizeRoute`.

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/http/middlewares/metrics.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/middlewares/ -run TestNormalizeRoute -v`
Expected: PASS (all sub-tests).

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/http/middlewares/metrics.go internal/http/middlewares/metrics_test.go
git commit -m "feat(metrics): route normalizer to bound Prometheus label cardinality"
```

---

### Task 2: HTTP metrics middleware

Add the `prometheus/client_golang` dependency and the `Metrics()` middleware that records RED metrics. Reuses the existing `responseWriter` from `logger.go`.

**Files:**
- Modify: `backend/internal/http/middlewares/metrics.go` (add the metrics struct + middleware)
- Modify: `backend/internal/http/middlewares/metrics_test.go` (add middleware test)
- Modify: `backend/go.mod`, `backend/go.sum` (via `go get`)

**Interfaces:**
- Consumes: `normalizeRoute(string) string` (Task 1); `responseWriter`, `newResponseWriter(http.ResponseWriter) *responseWriter` (existing, `logger.go`).
- Produces:
  - `func Metrics() func(http.Handler) http.Handler` — alice-compatible middleware on the default Prometheus registry; used by `module.go` (Task 3).
  - `func newHTTPMetrics(reg prometheus.Registerer) *httpMetrics` and `func (m *httpMetrics) middleware() func(http.Handler) http.Handler` — used in tests with an isolated registry.
  - Metric names: `http_requests_total{method,route,status}` (counter), `http_request_duration_seconds{method,route,status}` (histogram), `http_requests_in_flight` (gauge).

- [ ] **Step 1: Add the dependency**

Run:
```bash
cd backend && go get github.com/prometheus/client_golang@latest && go mod tidy
```
Expected: `go.mod` gains a `github.com/prometheus/client_golang vX.Y.Z` require line; `go.sum` updated.

- [ ] **Step 2: Write the failing test**

Append to `backend/internal/http/middlewares/metrics_test.go` (add imports at the top of the file: `net/http`, `net/http/httptest`, `github.com/prometheus/client_golang/prometheus`, `github.com/prometheus/client_golang/prometheus/testutil`):

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/middlewares/ -run TestMetricsMiddleware -v`
Expected: FAIL — `undefined: newHTTPMetrics`.

- [ ] **Step 4: Write minimal implementation**

Add to `backend/internal/http/middlewares/metrics.go` (add imports `net/http`, `strconv`, `time`, `github.com/prometheus/client_golang/prometheus`):

```go
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./internal/http/middlewares/ -v`
Expected: PASS (`TestNormalizeRoute`, `TestMetricsMiddlewareRecords`, `TestMetricsMiddlewareSkipsMetricsAndHealth`).

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/http/middlewares/metrics.go internal/http/middlewares/metrics_test.go go.mod go.sum
git commit -m "feat(metrics): http RED metrics middleware (client_golang)"
```

---

### Task 3: Mount /metrics and wire the middleware into the server

Expose `/metrics` via `promhttp` in the existing pre-swagger dispatcher and add `Metrics()` to the alice chain.

**Files:**
- Modify: `backend/internal/http/module.go` (add import; add `/metrics` branch at the dispatcher ~line 336–341; add `middlewares.Metrics()` to the chain at ~line 345)

**Interfaces:**
- Consumes: `middlewares.Metrics()` (Task 2); `promhttp.Handler()` from `github.com/prometheus/client_golang/prometheus/promhttp`.
- Produces: HTTP route `GET /metrics` serving Prometheus text exposition; instrumented request handling.

- [ ] **Step 1: Add the promhttp import**

In `backend/internal/http/module.go`, add to the import block (with the other third-party imports near `github.com/justinas/alice`):

```go
	"github.com/prometheus/client_golang/prometheus/promhttp"
```

- [ ] **Step 2: Add the `/metrics` dispatcher branch**

In `backend/internal/http/module.go`, in the dispatcher func, immediately **before** the final `base.ServeHTTP(w, r)` (currently line 341), add:

```go
		if p == "/metrics" {
			promhttp.Handler().ServeHTTP(w, r)
			return
		}
```

- [ ] **Step 3: Add the metrics middleware to the chain**

In `backend/internal/http/module.go`, change the alice chain (currently lines 345–350) to insert `middlewares.Metrics()` right after `middlewares.Logger()`:

```go
	handler := alice.New(
		middlewares.Recovery(),
		middlewares.Logger(),
		middlewares.Metrics(),
		middlewares.Cors(m.config.CORS),
		middlewares.RateLimit(m.config.RateLimit),
	).Then(router)
```

- [ ] **Step 4: Build and verify the binary, then smoke-test /metrics**

Run:
```bash
cd backend && go build ./... && go vet ./internal/http/...
```
Expected: no errors.

Then run the server locally and curl the endpoint (uses the dev compose defaults; `HTTP_PORT=8080`):
```bash
cd backend && (HTTP_ENABLED=true HTTP_HOST=127.0.0.1 HTTP_PORT=8080 DATABASE_ENABLED=false GRPC_ENABLED=false HTTP_MOCK_AUTH=true go run ./cmd/lia.go serve &) ; sleep 4 ; \
  curl -s 127.0.0.1:8080/metrics | grep -E "^http_requests_in_flight|^go_goroutines" ; \
  curl -s -o /dev/null -w "events status=%{http_code}\n" 127.0.0.1:8080/api/v1/events ; \
  curl -s 127.0.0.1:8080/metrics | grep "http_requests_total" | head ; \
  pkill -f "go run ./cmd/lia.go" || true
```
Expected: `/metrics` returns text including `go_goroutines` and `http_requests_in_flight`; after hitting `/api/v1/events` a `http_requests_total{...route="/api/v1/events"...}` line appears.

> If DATABASE_ENABLED=false is not supported by the app's bootstrap, fall back to the dev docker stack: `docker compose up -d` then curl `127.0.0.1:8080/metrics`. See [`../../memory/lia-dev-gotchas.md`](../../memory/lia-dev-gotchas.md) for the flaky-Docker host-run workaround.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/http/module.go
git commit -m "feat(metrics): expose /metrics and instrument the http chain"
```

---

### Task 4: Monitoring stack overlay + Prometheus config + alert rules

Add the additive compose overlay and the Prometheus config/rules. No app code change.

**Files:**
- Create: `backend/docker-compose.monitoring.yml`
- Create: `backend/monitoring/prometheus.yml`
- Create: `backend/monitoring/alerts.yml`

**Interfaces:**
- Consumes: the `app` service on the default compose network (`app:8080/metrics` from Task 3).
- Produces: `prometheus` (loopback `127.0.0.1:9091`) and `node_exporter` (network-internal `node_exporter:9100`) services; the `lia_prometheus` named volume.

- [ ] **Step 1: Create the Prometheus scrape config**

Create `backend/monitoring/prometheus.yml`:

```yaml
global:
  scrape_interval: 30s
  evaluation_interval: 30s

rule_files:
  - /etc/prometheus/alerts.yml

scrape_configs:
  - job_name: lia-app
    metrics_path: /metrics
    static_configs:
      - targets: ['app:8080']

  - job_name: node
    static_configs:
      - targets: ['node_exporter:9100']

  - job_name: prometheus
    static_configs:
      - targets: ['localhost:9090']
```

- [ ] **Step 2: Create the alert rules**

Create `backend/monitoring/alerts.yml`:

```yaml
groups:
  - name: lia
    rules:
      - alert: AppDown
        expr: up{job="lia-app"} == 0
        for: 2m
        labels: { severity: critical }
        annotations:
          summary: "Lia app is down (scrape failing for 2m)"

      - alert: High5xxRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
            / clamp_min(sum(rate(http_requests_total[5m])), 1) > 0.05
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "5xx error ratio above 5% over 5m"

      - alert: HighRequestLatencyP99
        expr: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) > 1
        for: 10m
        labels: { severity: warning }
        annotations:
          summary: "p99 HTTP latency above 1s for 10m"

      - alert: HostLowDisk
        expr: |
          node_filesystem_avail_bytes{mountpoint="/",fstype!~"tmpfs|overlay"}
            / node_filesystem_size_bytes{mountpoint="/",fstype!~"tmpfs|overlay"} < 0.10
        for: 5m
        labels: { severity: critical }
        annotations:
          summary: "Host root filesystem below 10% free"

      - alert: HostHighMemory
        expr: |
          node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes < 0.10
        for: 10m
        labels: { severity: warning }
        annotations:
          summary: "Host available memory below 10% for 10m"
```

> Note: node_exporter runs with `--path.rootfs=/host`, which **strips** the `/host` prefix — so the host root fs is reported as `mountpoint="/"` (NOT `/host`). Confirmed on vds-ru215 at deploy time (2026-06-27): root is `/` ext4. The Task 6 verify step asserts this series is non-empty.

- [ ] **Step 3: Create the monitoring compose overlay**

Create `backend/docker-compose.monitoring.yml`:

```yaml
# Additive monitoring overlay. Apply ALONGSIDE the 3-file app stack:
#
#   docker compose --env-file .env.prod \
#     -f docker-compose.yml -f docker-compose.prod.yml \
#     -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d
#
# Prometheus UI is loopback-only (127.0.0.1:9091); reach it via an SSH tunnel:
#   ssh -L 9091:localhost:9091 vdska2   then open http://localhost:9091
# node_exporter has NO host port — only Prometheus reaches it over the compose network.
services:
  prometheus:
    image: prom/prometheus:v2.54.1
    restart: unless-stopped
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
      - "--storage.tsdb.retention.time=15d"
      - "--storage.tsdb.retention.size=512MB"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./monitoring/alerts.yml:/etc/prometheus/alerts.yml:ro
      - lia_prometheus:/prometheus
    ports:
      - "127.0.0.1:9091:9090"
    mem_limit: 256m
    mem_reservation: 128m
    logging:
      driver: json-file
      options: { max-size: "10m", max-file: "3" }

  node_exporter:
    image: prom/node-exporter:v1.8.2
    restart: unless-stopped
    command:
      - "--path.rootfs=/host"
    pid: host
    volumes:
      - "/:/host:ro,rslave"
    mem_limit: 64m
    logging:
      driver: json-file
      options: { max-size: "10m", max-file: "3" }

volumes:
  lia_prometheus:
```

- [ ] **Step 4: Validate compose config locally**

Run (validates merge + syntax without starting anything; uses dev base, no `.env.prod` needed locally):
```bash
cd backend && docker compose -f docker-compose.yml -f docker-compose.monitoring.yml config >/dev/null && echo "compose OK"
```
Expected: `compose OK` (no YAML/merge errors). Confirm `prometheus`, `node_exporter`, and the `lia_prometheus` volume appear:
```bash
cd backend && docker compose -f docker-compose.yml -f docker-compose.monitoring.yml config | grep -E "prometheus:|node_exporter:|lia_prometheus|127.0.0.1:9091"
```

- [ ] **Step 5: Commit**

```bash
cd backend && git add docker-compose.monitoring.yml monitoring/prometheus.yml monitoring/alerts.yml
git commit -m "feat(monitoring): prometheus + node_exporter overlay with capped retention"
```

---

### Task 5: Log rotation on existing services

Cap logs on the existing containers via compose `logging:` blocks (IaC, no daemon restart).

**Files:**
- Modify: `backend/docker-compose.yml` (`app`, `postgres`)
- Modify: `backend/docker-compose.gateguard.yml` (`gateguard`, `gateguard-redis`)

**Interfaces:**
- Consumes: nothing.
- Produces: each touched service capped at ≤30 MB of json-file logs (10m × 3).

- [ ] **Step 1: Add logging blocks in the base compose**

In `backend/docker-compose.yml`, add the following block (same indentation as the other keys, e.g. after `restart: unless-stopped`) to BOTH the `postgres` and `app` services:

```yaml
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

- [ ] **Step 2: Add logging blocks in the gateguard compose**

In `backend/docker-compose.gateguard.yml`, add the same block to BOTH the `gateguard` and `gateguard-redis` services:

```yaml
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

- [ ] **Step 3: Validate compose config**

Run:
```bash
cd backend && docker compose -f docker-compose.yml -f docker-compose.gateguard.yml config | grep -A3 "logging:" | head -40
```
Expected: `logging:` blocks render under `app`, `postgres`, `gateguard`, `gateguard-redis` with `max-size: "10m"` / `max-file: "3"`.

- [ ] **Step 4: Commit**

```bash
cd backend && git add docker-compose.yml docker-compose.gateguard.yml
git commit -m "chore(ops): cap container logs (json-file 10m x3) on app/postgres/gateguard"
```

---

### Task 6: nginx /metrics block + deploy runbook

Block `/metrics` from the public API vhost and document the additive deploy.

**Files:**
- Create: `docs/superpowers/runbooks/2026-06-27-prometheus-monitoring-deploy.md`
- (On the box at deploy time, not in-repo: `/etc/nginx/sites-available/lia`)

**Interfaces:**
- Consumes: all prior tasks (the built image with `/metrics`, the monitoring overlay, the log-rotation blocks).
- Produces: a step-by-step deploy runbook; the public `api.lia.pashteto.com/metrics` returns 403.

- [ ] **Step 1: Write the runbook**

Create `docs/superpowers/runbooks/2026-06-27-prometheus-monitoring-deploy.md`:

````markdown
# Runbook — Deploy simple Prometheus monitoring (vds-ru215)

Spec: `../specs/2026-06-27-prometheus-monitoring-design.md`
Plan: `../plans/2026-06-27-prometheus-monitoring.md`

Additive: adds `/metrics` to the backend image + a Prometheus/node_exporter overlay,
caps logs on existing services, and denies `/metrics` at nginx. Loopback-only UI.

## 1. Build & ship the new backend image (Mac → box)
Same pattern as prior deploys (build on Mac, `docker save | ssh | docker load`; IPv6 broken so build pulls use `curl -4`). Keep a rollback tag:
```bash
# on the box, before loading the new image
docker tag <current-app-image> lia-backend:rollback-premonitoring
```
Build, save, copy, load the new `app` image per the standard deploy runbook.

## 2. Copy monitoring configs to the box
```bash
scp -r backend/monitoring vdska2:/opt/lia/backend/monitoring
scp backend/docker-compose.monitoring.yml vdska2:/opt/lia/backend/
# also re-sync the edited docker-compose.yml + docker-compose.gateguard.yml (log blocks)
scp backend/docker-compose.yml backend/docker-compose.gateguard.yml vdska2:/opt/lia/backend/
```

## 3. Recreate app + capped-log services + start monitoring (one up -d)
On the box, in `/opt/lia/backend`, use ALL FOUR compose files (the 3-file app stack is
mandatory or app reverts to mock auth — known gotcha):
```bash
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d
```
Blast radius: brief (seconds) restart of `app`/`postgres`/`gateguard` as the log-rotation
recreate applies. Confirm app health after: `curl -s localhost:9080/health`.

## 4. Block /metrics at nginx
Edit `/etc/nginx/sites-available/lia`, inside the `api.lia.pashteto.com` server block add:
```nginx
location = /metrics { deny all; return 403; }
```
Then:
```bash
nginx -t && systemctl reload nginx
```

## 5. Verify
```bash
# /metrics blocked publicly:
curl -s -o /dev/null -w "%{http_code}\n" https://api.lia.pashteto.com/metrics    # 403
# /metrics reachable on loopback:
curl -s localhost:9080/metrics | grep -c http_requests_total                      # >=1
# Prometheus targets (tunnel from laptop):
#   ssh -L 9091:localhost:9091 vdska2 ; open http://localhost:9091/targets
# all three jobs (lia-app, node, prometheus) show state=up; /alerts lists 5 rules.
```
If `HostLowDisk` never has data, check node_exporter's root mountpoint label
(`node_filesystem_avail_bytes`) and adjust `alerts.yml` to match. On vds-ru215 the
root fs is `mountpoint="/"` (ext4) because `--path.rootfs=/host` strips the prefix —
the shipped `alerts.yml` uses `/` accordingly.

## 6. One-time: trim pre-rotation logs (if already large)
The `logging:` caps only apply to logs written after recreate. If existing logs are big:
```bash
docker ps --format '{{.Names}}' | xargs -I{} sh -c 'truncate -s 0 $(docker inspect --format="{{.LogPath}}" {}) 2>/dev/null || true'
```

## Rollback
```bash
docker compose ... -f docker-compose.monitoring.yml down            # stop monitoring only
# revert app image:
docker tag lia-backend:rollback-premonitoring <app-image-ref> && docker compose ... up -d app
# remove nginx block + reload.
```
````

- [ ] **Step 2: Verify the runbook references resolve**

Run:
```bash
ls docs/superpowers/specs/2026-06-27-prometheus-monitoring-design.md docs/superpowers/plans/2026-06-27-prometheus-monitoring.md
```
Expected: both paths exist (the runbook's relative links resolve).

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/runbooks/2026-06-27-prometheus-monitoring-deploy.md
git commit -m "docs(runbook): deploy simple Prometheus monitoring to vds-ru215"
```

---

## Final verification (after all tasks)

- [ ] `cd backend && go build ./... && go test ./internal/http/middlewares/ -v` — all pass.
- [ ] `cd backend && docker compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml config >/dev/null` — merges cleanly (needs the `.env.prod` vars or run with the dev subset).
- [ ] `git log --oneline -8` shows the six task commits.
- [ ] Local smoke: `/metrics` serves Go + http_* metrics; hitting an API route increments `http_requests_total` with a normalized `route` label.
```
