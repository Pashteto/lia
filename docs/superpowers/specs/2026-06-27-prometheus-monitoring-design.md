# Simple Prometheus Monitoring — Design

_Date: 2026-06-27. Status: **design, awaiting review**. Scope: instrument the Lia backend with Prometheus metrics, scrape it (plus host metrics) from a lightweight Prometheus running on the same VDS, with bounded disk + bounded logs so nothing grows unboundedly on the RAM-constrained box._

## Goal

Add **simple, low-footprint observability** to the live Lia deployment (`lia.pashteto.com` on vds-ru215, a single ~1.9 GB hand-managed Docker + nginx box) so we can answer: is the app up, how much traffic / error rate / latency is it serving, and is the host running out of CPU / RAM / disk.

Explicitly **not** a full observability platform. No local Grafana, no Alertmanager/notifications, no distributed tracing.

## Constraints (what shapes every decision)

1. **Single small VDS, ~1.9 GB RAM, historically flaky** (swap added, IPv6 broken). Every added container must have a memory ceiling and must not be able to OOM the `app`.
2. **Disk is finite** — Prometheus TSDB and container logs must be **hard-capped**, not just time-bounded.
3. **Hand-managed Docker + nginx**, not Terraform (this box is the documented IaC exception). Changes land as additive compose files + an nginx snippet + a runbook.
4. **No new public attack surface.** `/metrics` and the Prometheus UI must not be reachable from the internet.
5. **Additive only** — bringing up the monitoring stack must not require recreating or risking the `app`/`postgres`/`gateguard` containers beyond a controlled, optional log-rotation recreate.

## Decisions (resolved during brainstorming)

| Decision | Choice |
|---|---|
| Stack footprint | App `/metrics` + `node_exporter` (host) + lightweight Prometheus. **No local Grafana.** |
| Dashboards | Prometheus' own UI / PromQL now; Grafana Cloud (free, remote) optional later. |
| UI access | **SSH tunnel only.** Prometheus bound to `127.0.0.1`; no public vhost. |
| Alerting | Alert **rules** defined (visible on Prometheus `/alerts`); **no Alertmanager / no notifications** yet. |
| Path-label cardinality | **Path normalizer** (regex collapsing IDs to `:id`) — no coupling to go-swagger route context. |
| TSDB growth | Bounded by **both** `retention.time=15d` and `retention.size=512MB` (size cap = hard guarantee). |
| Log growth | Per-service `logging:` blocks (`json-file`, `max-size=10m`, `max-file=3`) on monitoring **and** existing services. |

## Current state (from infra survey, 2026-06-27)

- Go `1.24.0`, module `github.com/Pashteto/lia`, built from go-microservice-template as a modular monolith.
- HTTP server uses the **go-swagger generated mux**. Middleware is an `alice` chain at `internal/http/module.go:345` (`Recovery → Logger → Cors → RateLimit → router`).
- `Logger()` middleware (`internal/http/middlewares/logger.go:38`) already wraps the `ResponseWriter` and captures method, URI, status, and duration — the same signals the metrics middleware needs.
- A **custom-path dispatcher** already sits in front of the swagger mux (`internal/http/module.go:320-342`): it branches on path prefix to route to the admin/organizer/complaints/uploads plain-`net/http` handlers before falling through to `base.ServeHTTP`. **This is exactly where `/metrics` mounts.**
- `GET /health` exists (`internal/http/handlers/health.go:24`) but returns hardcoded `"healthy"` (no real dependency check).
- **No** prometheus client, `/metrics`, `expvar`, or `pprof` today.
- Prod: nginx fronts `api.lia.pashteto.com → 127.0.0.1:9080` (the `app`) and `lia.pashteto.com → 127.0.0.1:3001` (frontend). `app` is reached on the host only at `127.0.0.1:9080`; inside the compose network it is `app:8080`. gRPC disabled in prod. Compose is the 3-file stack: `docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml`.

## Architecture

```
                          ssh -L 9091:localhost:9091 vdska2
   developer laptop  ─────────────────────────────────────────►  browser localhost:9091
                                                                        │
   ┌───────────────────────────── vds-ru215 (docker compose project "backend") ───────────┐
   │                                                                     │ 127.0.0.1:9091  │
   │   ┌──────────┐    scrape app:8080/metrics      ┌────────────────────▼──────────────┐ │
   │   │   app    │◄────────────────────────────────│  prometheus                       │ │
   │   │  :8080   │                                  │   retention 15d / 512MB           │ │
   │   │ /metrics │    scrape node_exporter:9100     │   rules: alerts.yml               │ │
   │   └──────────┘◄──────────────┐                  │   vol: lia_prometheus             │ │
   │                              │                  └───────────────────────────────────┘ │
   │   ┌──────────┐   ┌───────────▼─────┐                                                   │
   │   │ postgres │   │ node_exporter   │  (host CPU/RAM/disk, internal network only)       │
   │   └──────────┘   └─────────────────┘                                                   │
   │   ┌──────────┐                                                                          │
   │   │gateguard │   nginx: api vhost has `location /metrics { deny all; }`                 │
   │   └──────────┘                                                                          │
   └──────────────────────────────────────────────────────────────────────────────────────┘
```

All monitoring services join the existing `backend` compose project's default network, so Prometheus reaches `app:8080` and `node_exporter:9100` by service name. Nothing but the Prometheus UI is published to the host, and that only on loopback.

## Components

### 1. App instrumentation — `internal/http/middlewares/metrics.go` (new)

Depends on `github.com/prometheus/client_golang` (added to `go.mod`; standard choice, BSD-3).

- Register the **default Go + process collectors** (goroutines, GC pauses, heap, open FDs, CPU seconds) — free runtime visibility, no extra code.
- HTTP metrics on the default registry:
  - `http_requests_total{method,route,status}` — counter.
  - `http_request_duration_seconds{method,route,status}` — histogram (default buckets, suitable for sub-second web latencies).
  - `http_requests_in_flight` — gauge.
- `Metrics()` returns an `alice`-compatible `func(http.Handler) http.Handler`. It wraps the response writer to capture the status code (mirroring `logger.go`), measures wall-clock duration, and records the three metrics.
- **Self-instrumentation guard:** the middleware records nothing for `/metrics` and `/health` (skip by exact path) so scrape traffic and health pings don't pollute the series.
- **`route` label** comes from the **path normalizer** (component 2), never the raw URL path — this is the cardinality safeguard.

Wiring: insert into the `alice` chain at `internal/http/module.go:345`, immediately after `Logger()`:
```go
handler := alice.New(
    middlewares.Recovery(),
    middlewares.Logger(),
    middlewares.Metrics(),   // ← new
    middlewares.Cors(m.config.CORS),
    middlewares.RateLimit(m.config.RateLimit),
).Then(router)
```

### 2. Path normalizer — `normalizeRoute(path string) string` (in `metrics.go`)

Pure function, independently unit-tested. Collapses high-cardinality identifiers so the `route` label set stays bounded:
- UUIDs (`[0-9a-f]{8}-...`) → `:id`
- bare numeric path segments → `:id`
- everything else passes through unchanged.

Result: `/api/v1/events/2f1c.../complaints` → `/api/v1/events/:id/complaints`. Cardinality is bounded by the number of route shapes, not the number of entities. This is **Option A** from brainstorming; we deliberately do **not** recover the go-swagger matched-route template (Option B) to avoid coupling to generated middleware ordering.

### 3. `/metrics` endpoint — `internal/http/module.go` dispatcher

Add one branch to the existing pre-swagger dispatcher (`module.go:320-342`), before `base.ServeHTTP`:
```go
if p == "/metrics" {
    promhttp.Handler().ServeHTTP(w, r)
    return
}
```
This mirrors the existing uploads/admin/organizer/complaints hoist pattern — no swagger-spec edit, no generated-code change. The endpoint is reachable as `app:8080/metrics` inside the compose network and `127.0.0.1:9080/metrics` on the host loopback.

### 4. nginx — block `/metrics` publicly

Because the `api.lia.pashteto.com` vhost proxies **all** paths to `127.0.0.1:9080`, `/metrics` would otherwise be world-readable. Add to that server block:
```nginx
location = /metrics { deny all; return 403; }
```
Applied via the hand-edited `/etc/nginx/sites-available/lia` + `nginx -t && systemctl reload nginx` (captured in the runbook). The committed copy of the vhost in the repo (if any) is updated to match.

### 5. Monitoring stack — `backend/docker-compose.monitoring.yml` (new, additive overlay)

A 4th, optional compose file (consistent with the existing multi-file pattern). Brought up alongside the 3-file app stack; never required to run the app.

**`prometheus`** (`prom/prometheus`, pinned by digest):
- Mounts `./monitoring/prometheus.yml` (ro) and `./monitoring/alerts.yml` (ro).
- Named volume `lia_prometheus` → `/prometheus`.
- Flags: `--config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=15d --storage.tsdb.retention.size=512MB`.
- Published **`127.0.0.1:9091:9090`** (loopback only — SSH-tunnel access).
- `mem_limit: 256m` (+ `mem_reservation`) so it can never starve the `app`.
- `restart: unless-stopped`.
- `logging:` json-file `max-size=10m`, `max-file=3`.

**`node_exporter`** (`prom/node-exporter`, pinned by digest):
- Read-only host mounts (`/proc`, `/sys`, `/` ro) per the standard node_exporter compose recipe; `--path.rootfs=/host`.
- **No host port** — only reachable on the compose network as `node_exporter:9100`.
- `mem_limit: 64m`, `restart: unless-stopped`, same `logging:` block.

### 6. Prometheus config — `backend/monitoring/prometheus.yml` (new)

```yaml
global:
  scrape_interval: 30s        # gentle on a small box
  evaluation_interval: 30s
rule_files:
  - /etc/prometheus/alerts.yml
scrape_configs:
  - job_name: lia-app
    metrics_path: /metrics
    static_configs: [{ targets: ['app:8080'] }]
  - job_name: node
    static_configs: [{ targets: ['node_exporter:9100'] }]
  - job_name: prometheus
    static_configs: [{ targets: ['localhost:9090'] }]
```

### 7. Alert rules — `backend/monitoring/alerts.yml` (new)

Rules only; surfaced on Prometheus `/alerts` (no routing/notifications):
- **AppDown** — `up{job="lia-app"} == 0` for 2m.
- **High5xxRate** — 5xx ratio of `http_requests_total` > 5% over 5m.
- **HighRequestLatencyP99** — `histogram_quantile(0.99, http_request_duration_seconds)` > 1s for 10m.
- **HostLowDisk** — node_exporter root filesystem free < 10%.
- **HostHighMemory** — node_exporter available memory < 10% for 10m.

### 8. Log rotation on existing services

Add a `logging:` block (`json-file`, `max-size=10m`, `max-file=3` → ≤30 MB/container) to **`app`, `postgres`, `gateguard`, `gateguard-redis`** in their compose files (`docker-compose.yml` / `docker-compose.prod.yml` / `docker-compose.gateguard.yml`). Applied with a controlled `up -d` recreate of those services.

> **Why compose blocks, not `/etc/docker/daemon.json`:** the daemon-wide default would require `systemctl restart docker`, which restarts every container at once — a real outage risk on this flaky box — and is an un-versioned host edit. Per-service compose blocks are IaC-tracked and only recreate the targeted containers. **Blast radius of the recreate:** brief (seconds) restart of `app`/`postgres`/`gateguard` during `up -d`; schedule with the deploy, not ad hoc. Existing pre-rotation log files are not retroactively trimmed — note in the runbook to `truncate`/`docker logs` rotate manually once if they're already large.

## Memory & disk budget

| Item | RAM ceiling | Disk |
|---|---|---|
| prometheus | `mem_limit 256m` | ≤512 MB TSDB (`lia_prometheus`) |
| node_exporter | `mem_limit 64m` | negligible |
| per-container logs | — | ≤30 MB each (10m×3) |

Worst case added RAM ≈ 320 MB ceiling (typical steady-state ~120–150 MB). On a 1.9 GB box with swap this is acceptable but deliberately capped so a runaway Prometheus cannot take down the app.

## Testing (TDD)

1. **`normalizeRoute` table test** — UUIDs and numeric segments collapse to `:id`; static paths unchanged; `/metrics` and `/health` handled; proves bounded cardinality.
2. **Middleware test** — drive a request through `Metrics()(handler)` and assert via `prometheus/testutil` that `http_requests_total` increments with the expected `{method,route,status}` and the histogram observes one sample; assert `/metrics`/`/health` are skipped.
3. **Manual / runbook verification** — `curl 127.0.0.1:9080/metrics` returns Prometheus text; `curl -s api.lia.pashteto.com/metrics` returns 403; Prometheus `/targets` shows all three jobs `up`; `/alerts` lists the five rules.

## Deployment (additive)

Detailed in a runbook `docs/superpowers/runbooks/2026-06-27-prometheus-monitoring-deploy.md`. Outline:
1. Build the new backend image (with `/metrics`) — Mac build → `docker save | ssh | docker load` per the existing deploy pattern; keep a rollback tag.
2. Copy `monitoring/` configs to `/opt/lia/backend/monitoring/`.
3. Recreate `app` with the **full 3-file** stack (so it doesn't revert to mock auth — known gotcha) + the controlled log-rotation recreate of `postgres`/`gateguard`.
4. `docker compose ... -f docker-compose.monitoring.yml up -d` to start prometheus + node_exporter.
5. nginx: add `location = /metrics { deny all; }`, `nginx -t && systemctl reload nginx`.
6. Verify per the testing checklist above.

## Out of scope (future increments, each its own slice)

- Local Grafana (RAM); use Grafana Cloud remote scraping/dashboards if desired.
- Alertmanager + notifications (email/Telegram/Slack).
- DB connection-pool metrics (go-pg stats) and a real-DB `/health` check — the hardcoded `/health` stays; Prometheus `up` already signals process liveness.
- pprof / tracing.

## Audit / compliance note

This adds a **detective control** (metrics + alert rules) and is net-positive for ISO 27001 monitoring expectations. No control is disabled. The Prometheus UI is loopback-only behind SSH (no new public surface, no new credential). `/metrics` carries operational metrics only — no secrets, no PII — and is denied at nginx. No production data leaves the box.
