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

# Assert HostLowDisk is not silently dead: this MUST return a non-empty result.
# A missing series means the alert can never fire (worse than a noisy one).
curl -s 'http://localhost:9091/api/v1/query?query=node_filesystem_avail_bytes%7Bmountpoint%3D%22%2Fhost%22%7D' \
  | grep -q '"result":\[{' && echo "HostLowDisk series OK" || echo "HostLowDisk series MISSING — fix mountpoint below"
```
If the HostLowDisk series is MISSING (or the alert never has data), check
node_exporter's actual root mountpoint label —
`curl -s 'http://localhost:9091/api/v1/query?query=node_filesystem_avail_bytes' | tr ',' '\n' | grep mountpoint`
— and adjust `alerts.yml` `mountpoint="/host"` (both HostLowDisk lines) to match, then reload Prometheus.

## 6. One-time: trim pre-rotation logs (if already large)
The `logging:` caps only apply to logs written after recreate. If existing logs are big:
```bash
docker ps --format '{{.Names}}' | xargs -I{} sh -c 'truncate -s 0 $(docker inspect --format="{{.LogPath}}" {}) 2>/dev/null || true'
```

## Rollback
```bash
cd /opt/lia/backend
# stop monitoring only (app/db/gateguard keep running):
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml stop prometheus node_exporter
# revert the app image to the pre-monitoring build, then recreate app:
docker tag backend-app:rollback-premonitoring backend-app:latest
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d
# remove the nginx /metrics deny: restore the backup + reload
cp /etc/nginx/sites-available/lia.bak-premonitoring-20260627 /etc/nginx/sites-available/lia && nginx -t && systemctl reload nginx
```

---

## As executed — 2026-06-27 (LIVE on vds-ru215)

Deployed from local `main` `4e19a1b`. No DB migration (prod stayed at **018**); a
pre-deploy backup was taken anyway: `/opt/lia/backup-pre-monitoring-20260627.sql.gz`.

- **Image:** built `backend-app:amd64-monitoring` (`c1e68320e552`, linux/amd64, 8.7 MB
  compressed) on the Mac → `docker save | ssh | docker load`. Retagged on box:
  `backend-app:rollback-premonitoring` = `7428be5f9def` (prior calendar build);
  `backend-app:latest` = `c1e68320e552`.
- **Monitoring images:** `prom/prometheus:v2.54.1` + `prom/node-exporter:v1.8.2`
  **pulled directly on the box** (it pulls Hub fine — only the ~1 GB golang build base is
  the tunnel problem; local `docker save` of these multi-arch images failed with a
  containerd manifest error, so pull-on-box was used).
- **Configs shipped:** `docker-compose.{yml,gateguard.yml}` (log-rotation blocks only —
  diffed against the box copies first, additions-only), new `docker-compose.monitoring.yml`,
  and `monitoring/{prometheus.yml,alerts.yml}`.
- **Recreate:** `up -d` with all 4 compose files recreated app/postgres/gateguard/redis
  (log-rotation config change) + started prometheus + node_exporter; `migrate` no-op
  (already 018). Brief blip; postgres returned healthy.
- **nginx:** added `location = /metrics { deny all; return 403; }` to the
  `api.lia.pashteto.com` block (backup `lia.bak-premonitoring-20260627`), `nginx -t` + reload.

**Verified live:** all 6 backend containers + frontend Up; `up{job=...}==1` for lia-app /
node / prometheus; 5 alert rules `health: ok`; loopback `/metrics` serves with normalized
route labels; public `https://api.lia.pashteto.com/metrics` → **403**; public API → 200;
log rotation (`json-file 10m×3`) confirmed on all recreated containers; prometheus
`mem_limit`=256 MB, retention `15d`+`512MB`, bound to `127.0.0.1:9091` only.

**Fixed at deploy time:** `HostLowDisk` used `mountpoint="/host"`, but `--path.rootfs=/host`
**strips** the prefix → root fs is `mountpoint="/"` (ext4). Series was empty (0) until
`alerts.yml` was corrected to `/` and Prometheus reloaded (`docker kill -s HUP
backend-prometheus-1`); now returns ~2.05 GB free. The repo `alerts.yml` carries `/`.

**Known follow-up (not blocking):** the middleware skips bare `/health`, but the live
health endpoint is `/api/v1/health`, so uptime pings accrue under
`http_requests_total{route="/api/v1/health"}` (one bounded series — harmless). The
important `/metrics` self-skip works. Fold `/api/v1/health` into the skip-guard on the next
backend rebuild.

**Access:** `ssh -L 9091:localhost:9091 vdska2` → open `http://localhost:9091`.
