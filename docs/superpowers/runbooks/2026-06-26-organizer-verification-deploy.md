# Runbook — deploy: organizer entity + verification (admin-suite #2) — 2026-06-26

Deploys admin-suite **sub-project #2** (organizer entity + verification), merged to local
`main` (FF `7e595fe`). Backend + frontend images rebuilt **from the reviewed `main`** and
cut over. **DB migrations `000015`+`000016` were already applied** to prod (a concurrent
session had migrated `schema_migrations` to **16** + created the tables/seed), so the
`migrate` step was **skipped** this round.

Live target: `https://lia.pashteto.com` / `https://api.lia.pashteto.com` on **vds-ru215**
(`193.32.188.7`, `ssh vdska2`, root). Hand-managed Docker (demo exception). See
[[lia-demo-deployment]] for box/auth details.

## What shipped
- Migrations: `000015_organizers` (`organizers` + `organizer_verification_history`), `000016_app_settings` — **already on prod** (verify `schema_migrations = 16`, tables present, `app_settings` seed `organizers.auto_verify_all = {"enabled": false}`).
- Backend: `internal/organizers` + `internal/settings` domains; user/admin/public endpoints; derived `verified`/`profile_id` on event payloads (additive swagger `Organizer` fields).
- Frontend: `/me/organizer`, `/admin/moderation/organizers`, `/admin/organizers`, `/admin/settings`, public `/organizers/[id]`, `VerifiedBadge` on event cards/detail.

## Pre-deploy state captured
- Rollback tags pinned to the **then-live** images: `backend-app:rollback-orgverify` (`9df3e10`), `lia-frontend:rollback-orgverify` (`439ced`). (The generic `backend-app:rollback` / `lia-frontend:rollback` from the prior deploy still exist too.)
- Pre-deploy DB dump: `/opt/lia/backup-pre-orgverify-20260626-1549.sql.gz`.
- **Do NOT trust** the stray `backend-app:amd64` the concurrent session left on the box — rebuild from `main`.

## Procedure (build-on-Mac → ship → cutover; migrate already done)

```bash
# --- BACKEND ---
# Mac: regenerate the gitignored swagger model (adds Organizer.verified/profile_id), sanity build
cd backend && make generate-api && go build ./...      # NOT make generate-all — its proto step
                                                        # fails on a missing ./userservice dir
docker build --platform linux/amd64 -t backend-app:amd64-orgverify .
docker save backend-app:amd64-orgverify | gzip | ssh vdska2 'gunzip | docker load'
rsync -az backend/db/migrations/ vdska2:/opt/lia/backend/db/migrations/   # keep box in sync (015/016 already applied)

# Box: DB already at 16 — skip `migrate`. Tag + recreate app with ALL THREE compose files.
ssh vdska2 'cd /opt/lia/backend && docker tag backend-app:amd64-orgverify backend-app:latest && \
  docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d --no-build app'
# (gateguard compose file is MANDATORY — it sets HTTP_GATEKEEPER_ADDRESS + HTTP_MOCK_AUTH;
#  omit it → register/login 503 + mock-auth flips on. See lia-demo-deployment.)

# --- FRONTEND ---
cd frontend
docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64-orgverify .
docker save lia-frontend:amd64-orgverify | gzip | ssh vdska2 'gunzip | docker load'
ssh vdska2 'docker tag lia-frontend:amd64-orgverify lia-frontend:latest && docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

## Verify (all PASSED 2026-06-26 unless noted)
```bash
B=https://api.lia.pashteto.com; F=https://lia.pashteto.com
curl -s -o /dev/null -w '%{http_code}\n' "$B/api/v1/events?status=published"       # 200
curl -s -o /dev/null -w '%{http_code}\n' "$B/api/v1/me/organizer"                  # 401 (new route live; 404 = old image)
curl -s -o /dev/null -w '%{http_code}\n' "$B/api/v1/admin/moderation/organizers"   # 401
curl -s -o /dev/null -w '%{http_code}\n' "$B/api/v1/admin/settings"                # 401
curl -s -o /dev/null -w '%{http_code}\n' "$B/api/v1/organizers/<rand-uuid>"        # 404 (no leak)
for p in / /me/organizer /admin /admin/settings /admin/organizers /organizers/<id>; do
  curl -s -o /dev/null -w "$p %{http_code}\n" "$F$p"; done                          # all 200
```
**Functional (user-facing), verified end-to-end:** `POST /api/v1/auth/register` → 200 + token
(NB the real path is **`/api/v1/auth/register`**, basePath `/api/v1/` — NOT `/auth/register`),
then `PUT /api/v1/me/organizer` → 200 draft, `POST …/submit` → `pending` (global auto-verify
off), `GET …` → pending; public `GET /api/v1/organizers/{pending-id}` → 404 (no leak).

## NOT prod-verified (needs the real admin account)
Admin verify→verified, public page 200 post-verify, the global auto-verify toggle, and the ✓
badge on events are gated behind admin auth and were **not** exercised on prod (bootstrapping a
throwaway admin via a prod gateguard-DB write was blocked as an unrequested privilege
escalation). To finish: log in as `poulissimo@gmail.com` (live admin) and verify the leftover
pending smoke org in `/admin/moderation/organizers`, OR grant a test account `admin` in the
`gateguard` DB and clean up after. A harmless leftover pending smoke org `975022d5-…` (owner
`orgverify-smoke-*@presence.test`) is invisible while pending.

## Rollback
```bash
# backend (ALWAYS all three -f files)
ssh vdska2 'cd /opt/lia/backend && docker tag backend-app:rollback-orgverify backend-app:latest && \
  docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d --no-build app'
# frontend
ssh vdska2 'docker tag lia-frontend:rollback-orgverify lia-frontend:latest && docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
# DB: migrations 015/016 are additive (new tables only); to undo, `migrate` down to 14 or restore the dump.
```

## Notes / not-done
- Image-only deploy (only `db/migrations` rsynced; source not synced to the box).
- Organizer `//go:build integration` repo tests were NOT run against prod (no test DB) — wire into CI.
- All on **local `main`, unpushed** (deployed from local images, not a pushed ref/PR).
- **`GATEGUARD_AUTH_SECRET` rotation still pending** (exposed in a 2026-06-26 transcript).
- Compliance (ISO 27001 / Vanta): additive schema (change-management — logged here); pre-deploy dump retained; auto-verify toggles are audited (`settings.update` / `organizer.set_auto_verify` in `audit_log`).
