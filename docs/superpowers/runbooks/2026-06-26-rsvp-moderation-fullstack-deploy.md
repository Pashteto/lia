# Runbook — full-stack deploy: RSVP + moderation/admin (2026-06-26)

Deploys everything that accumulated on local `main` since the last live cutover:
event-edit/draft-visibility, the **RSVP** feature, publish-draft, and the
**moderation/admin foundation** (RBAC via GateGuard `admin` role, `/admin`
queue, take-down/reinstate, audit log). Backend + frontend images rebuilt,
**three DB migrations applied (000012 → 000014)**. GateGuard untouched (moderation
reuses its existing `admin` role — Approach 1, no reship).

Live target: `https://lia.pashteto.com` / `https://api.lia.pashteto.com` on
**vds-ru215** (`193.32.188.7`, `ssh vdska2`, root). Hand-managed Docker (demo
exception). Prod was at migration **011** before this deploy (RSVP had only ever
been merged to local `main`, never deployed).

## Pre-deploy state captured
- Rollback tags: `backend-app:rollback` (= old `c429304`), `lia-frontend:rollback` (= old `9ace229`).
- DB backup: `/opt/lia/backup-pre-moderation-20260626-1000.sql.gz` (pre-migration dump).

## Procedure (build-on-Mac → ship → migrate → cutover)

```bash
# --- BACKEND ---
# Mac: regenerate gitignored swagger code, then build amd64 (make build = go build only)
cd backend && make generate-api && go build ./...        # sanity
docker build --platform linux/amd64 -t backend-app:amd64 .
docker save backend-app:amd64 | gzip | ssh vdska2 'gunzip | docker load'
# ship the new migration files (the compose `migrate` service reads the box's ./db/migrations)
rsync -az backend/db/migrations/ vdska2:/opt/lia/backend/db/migrations/

# Box: migrate 11 -> 14, verify, then swap the app image (no on-box build)
ssh vdska2
cd /opt/lia/backend
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml run --rm migrate   # 12,13,14
docker exec backend-postgres-1 sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT version,dirty FROM schema_migrations;"'  # -> 14|f
docker tag backend-app:amd64 backend-app:latest
# IMPORTANT: include -f docker-compose.gateguard.yml — it sets the app's
# HTTP_GATEKEEPER_ADDRESS=gateguard:9090 and HTTP_MOCK_AUTH. Recreating `app`
# WITHOUT it drops those env vars → app falls back to the localhost:9091 default
# → register/login (and token validation) fail with 503 / mock-auth turns on.
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d --no-build app

# --- FRONTEND ---
cd frontend
docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64 .
docker save lia-frontend:amd64 | gzip | ssh vdska2 'gunzip | docker load'
ssh vdska2 'docker tag lia-frontend:amd64 lia-frontend:latest && docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

## Verify (all passed 2026-06-26)
```bash
curl -s -o /dev/null -w '%{http_code}\n' https://lia.pashteto.com                         # 200
curl -s -o /dev/null -w '%{http_code}\n' https://api.lia.pashteto.com/api/v1/events?status=published  # 200
curl -s -o /dev/null -w '%{http_code}\n' https://lia.pashteto.com/admin                   # 200 (client gate)
# backend new surface (localhost on box): /auth/me, /api/v1/admin/* all 401 anon (gate active)
```
schema_migrations = 14 (clean); `users.role`, `event_rsvps`, `event_status_history`, `audit_log` present.

## REMAINING — seed an admin (moderation UI is gated until then)
Roles live in **GateGuard** (`gateguard` DB on the same Postgres). To make a real
account staff, set its role to `admin` there; on the user's next request Lia's
`Authenticate` syncs `users.role` and the «Админ» nav + `/admin` unlock.
Pick the email, then (schema-confirm the gateguard `users.role` column first):
```sql
-- inside backend-postgres-1, gateguard DB:
UPDATE users SET role = 'admin' WHERE email = '<ADMIN_EMAIL>';
```

## Rollback
```bash
# backend (ALWAYS include all three -f files — see note above)
ssh vdska2 'cd /opt/lia/backend && docker tag backend-app:rollback backend-app:latest && \
  docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d --no-build app'
# frontend
ssh vdska2 'docker tag lia-frontend:rollback lia-frontend:latest && docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
# DB (only if a migration must be undone): migrate down to 11, or restore the pre-migration dump.
```

## Notes / not-done
- Image-only deploy: backend source on the box is image-built; only `db/migrations` was rsynced. Frontend source not rsynced (image only).
- Moderation integration tests (`//go:build integration`) were NOT run against prod; covered in CI with a migrated test DB.
- All work remains on **local `main`, unpushed** (origin/main behind) — deployed from local images, not from a pushed ref / PR.
- Compliance (ISO 27001 / Vanta): additive schema migrations (change-management — logged here); pre-migration DB dump retained; admin-role grant is a privilege escalation (least-privilege — grant only the intended account).
