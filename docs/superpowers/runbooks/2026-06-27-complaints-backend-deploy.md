# Complaints / Reports — Backend Deploy (admin-suite #3)

_2026-06-27. **Backend-only** deploy of the complaints feature to the live demo
(`api.lia.pashteto.com` on vds-ru215). Frontend deliberately NOT deployed — see
"Why backend only". Feature branch `worktree-feat-complaints` @ `8044adf` (UNMERGED)._

## What shipped

- DB migration **016 → 017** (`complaints` table + two partial indexes). Additive,
  non-destructive.
- New `backend-app` image (`linux/amd64`, FROM scratch, 8.87 MB; image id `7614e9f862b8`)
  carrying the `internal/complaints` domain + the public submit endpoint
  (`POST /api/v1/events/{id}/complaints`) + the admin inbox endpoints
  (`GET /api/v1/admin/complaints`, `GET …/events/{id}`, `POST …/events/{id}/resolve`)
  + `complaints_open` on `/admin/overview`.

## Why backend only

A concurrent session had **uncommitted** event-detail/draft-visibility work in the
main checkout (refactor of `app/events/[id]/page.tsx` + new `EventDetailView.tsx`,
plus `fetchEventWithAuth` in `lib/api.ts`). The live `lia-frontend` image was rebuilt
~1 h before this deploy and may already contain that work. Rebuilding the frontend
from this feature branch could **revert** it, so the frontend half (report modal +
`/admin/complaints` page) is held until the branch is merged and `page.tsx` is
reconciled (move `ReportButton` into the new `EventDetailView`). The backend is
additive and independent — the old frontend simply doesn't call the new endpoints yet.

## Procedure (as executed)

1. **Backup** prod DB (rollback point):
   `docker exec backend-postgres-1 pg_dump -U lia_prod lia_prod | gzip > /opt/lia/backup-pre-complaints-20260627-0037.sql.gz`
2. **Ship migrations** (migrate service bind-mounts `./db/migrations`, so files must
   be on the box — they are NOT in the image):
   `scp db/migrations/000017_complaints.{up,down}.sql vdska2:/opt/lia/backend/db/migrations/`
3. **Build** image on the Mac (box can't pull large layers over the tunnel):
   `docker build --platform linux/amd64 -t backend-app:amd64-complaints .` (from `backend/`).
   Build context must contain the generated swagger code + `protocols/userservice/*.pb.go`
   (both gitignored; present in a worktree only after `make generate-api` + copying the
   userservice pb from a checkout that has it — `make build` is plain `go build`, no codegen,
   and `.dockerignore` does not exclude generated files, so `COPY . ./` ships them).
4. **Ship** image: `docker save backend-app:amd64-complaints | gzip | ssh vdska2 'gunzip | docker load'`.
5. **Retag** on box: `docker tag backend-app:latest backend-app:rollback-precomplaints`
   (preserve `9da00af611d8`), then `docker tag backend-app:amd64-complaints backend-app:latest`.
6. **Recreate** with ALL THREE compose files (the gateguard file is mandatory — it alone
   sets `HTTP_GATEKEEPER_ADDRESS` + `HTTP_MOCK_AUTH`; omit it → auth 503s):
   `cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d`
   → `backend-migrate-1` ran `17/u complaints (65ms)` and exited; `backend-app-1` restarted.

## Verification (prod, live)

- Migration: `SELECT version,dirty FROM schema_migrations` → `17|f`. Table + indexes
  (`complaints_one_open_per_reporter`, `complaints_open_target_idx`, `complaints_pkey`) present.
- Health: `GET /api/v1/health` → `{"status":"healthy"}`.
- Functional (throwaway `*@presence.test` user against `api.lia.pashteto.com`):
  authed submit → **201**; idempotent repeat → **200**; bad category → **400**;
  unknown event → **404**; anon submit → **401**; anon `GET /admin/complaints` → **401**.

## Rollback

- App: `docker tag backend-app:rollback-precomplaints backend-app:latest` then recreate
  (`up -d` with the 3 compose files).
- DB: migration 017 is additive (CREATE TABLE only). Down = `DROP TABLE complaints`
  (`migrate … down 1`), or restore `backup-pre-complaints-20260627-0037.sql.gz`. Rolling
  the app back without dropping the table is safe (no code references it).

## NOT yet prod-verified (needs the real admin account `poulissimo@gmail.com`)

The admin-authenticated paths could not be exercised (no admin password available):
`GET /admin/complaints` returning the grouped inbox, `POST …/resolve` takedown/dismiss,
and `complaints_open` on `/admin/overview`. The routes are confirmed mounted + gated
(401 anon). Verify these with the real admin once available.

## Cleanup TODO

- One smoke-test complaint row remains on prod (event `b0000000-…0005`, reporter
  `complaint-smoke-<ts>@presence.test`, note "smoke test", status `open`). The automated
  cleanup `DELETE` was correctly blocked (prod DB write not covered by deploy approval).
  Dismiss it via the admin UI once the frontend ships (doubles as a real resolve test),
  or authorize a one-row delete.
- Frontend deploy is pending the branch merge + `page.tsx`/`EventDetailView` reconciliation.
