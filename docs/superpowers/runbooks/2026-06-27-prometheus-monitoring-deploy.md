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
(`node_filesystem_avail_bytes`) and adjust `alerts.yml` `mountpoint="/host"` to match.

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
