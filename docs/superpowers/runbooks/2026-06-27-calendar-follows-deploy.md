# Calendar + organizer-follow — Deploy

_2026-06-27. Full deploy of the personal calendar + organizer follow/subscribe
feature to the live demo (`lia.pashteto.com` / `api.lia.pashteto.com` on vds-ru215).
Backend + frontend both build-on-Mac → `save | ssh | load` → cutover. Shipped from
branch `feat/calendar-and-follows` @ `02f45c3` (NOT merged to `main`; deployed from
local images, like prior feature deploys)._

## What shipped

- DB migration **017 → 018** (`organizer_follows` table + two indexes + composite PK).
  Additive, non-destructive.
- New `backend-app` image (`linux/amd64`, FROM scratch, 8.88 MB; id `7428be5f9def`):
  `internal/follows` domain + `internal/http/follows` handler. New endpoints
  `POST/DELETE /api/v1/me/follows/{organizerId}`, `GET /api/v1/me/follows`,
  `GET /api/v1/me/calendar?from&to`; `is_following` on public `/api/v1/organizers/{id}`.
  Events gained `ListForCalendar`/`GetEnriched`; rsvp gained `ListActiveEventsInRange`.
- New `lia-frontend` image (216 MB; id `fb0592445f2d`): `/me/calendar` page
  (month/week/day, color+legend), follow toggle on `/organizers/[id]`, nav entries
  (desktop AuthButton + mobile TabBar). Built from `feat/calendar-and-follows`, which
  is a **superset** of the live `amd64-complaints` frontend (branched off `main` after
  the complaints+draft-visibility merge), so it adds the calendar without regressing.

## Procedure (as executed)

1. **Backup** prod DB:
   `docker exec backend-postgres-1 pg_dump -U lia_prod lia_prod | gzip > /opt/lia/backup-pre-calendar-20260627-020709.sql.gz`
2. **Ship migration** (migrate service bind-mounts `./db/migrations`; NOT in the image):
   `scp db/migrations/000018_organizer_follows.{up,down}.sql vdska2:/opt/lia/backend/db/migrations/`
3. **Build backend** on Mac (box can't pull large layers over the tunnel; build context
   carries the gitignored generated swagger + `protocols/userservice/*.pb.go`, already
   present in the working tree from an earlier `make generate-api`):
   `docker build --platform linux/amd64 -t backend-app:amd64-calendar .` (from `backend/`).
4. **Ship** backend: `docker save backend-app:amd64-calendar | gzip | ssh vdska2 'gunzip | docker load'`.
5. **Cutover backend**: `docker tag backend-app:latest backend-app:rollback-precalendar`
   (preserve `7614e9f862b8`), `docker tag backend-app:amd64-calendar backend-app:latest`,
   then recreate with **ALL THREE** compose files (gateguard file mandatory — it alone sets
   `HTTP_GATEKEEPER_ADDRESS` + `HTTP_MOCK_AUTH`; omit it → auth 503s):
   `cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d`
   → `backend-migrate-1` ran `18/u organizer_follows (24ms)` and exited; `backend-app-1` recreated.
6. **Build frontend** on Mac: `docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64-calendar .` (from `frontend/`).
7. **Ship + cutover frontend**: `save | ssh | load`, then
   `docker tag lia-frontend:latest lia-frontend:rollback-precalendar` (preserve `9eb9de3f0b33`),
   `docker tag lia-frontend:amd64-calendar lia-frontend:latest`,
   `docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`.

## Verification (prod, live)

- Migration: `SELECT version,dirty FROM schema_migrations` → `18|f`. Table `organizer_follows`
  + `organizer_follows_user_idx` / `organizer_follows_organizer_idx` / `organizer_follows_pkey` present.
- Health: `GET /api/v1/health` → 200 healthy.
- Anon gating: `GET /me/calendar`, `GET /me/follows`, `POST /me/follows/{id}` → **401**;
  `GET /events?status=published` → 200 (unaffected).
- Authed (throwaway `cal-smoke-<ts>@presence.test` via register): `GET /me/calendar` → **200 `[]`**
  (full merge path exercised), `GET /me/follows` → **200 `[]`**, follow unknown/non-verified org → **404** (no leak).
- Frontend: `lia.pashteto.com/` → 200, `/me/calendar` → 200 (SSR renders «Календарь» auth-gate),
  home SSR has `/me/calendar` nav link, `/organizers/{id}` → 200. Container `Up`, Next "Ready in 540ms".

## Rollback

- Backend: `docker tag backend-app:rollback-precalendar backend-app:latest` then recreate
  (`up -d` with the 3 compose files). `7614e9f862b8`.
- Frontend: `docker tag lia-frontend:rollback-precalendar lia-frontend:latest && docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`. `9eb9de3f0b33`.
- DB: migration 018 is additive (CREATE TABLE only). Down = `migrate … down 1` (`DROP TABLE organizer_follows`),
  or restore `backup-pre-calendar-20260627-020709.sql.gz`. Rolling the app back without dropping
  the table is safe (only the new code references it).

## NOT prod-verified (needs the real admin account)

The full visual flow needs a verified organizer + a logged-in user: follow a verified org →
its published events appear on `/me/calendar` (from-followed color), an RSVP'd event appears
(attending color), an event that is both shows the combined treatment; month/week/day nav.
Anon/auth gating and the empty-state merge path ARE verified. The auto-mode classifier blocks
granting `admin` / arbitrary prod-DB writes from a throwaway account, so the verified-org path
was not exercised end-to-end.

## Cleanup TODO

- One throwaway user `cal-smoke-1782515361@presence.test` registered on prod during the smoke
  test (no follows/RSVPs, harmless — consistent with prior `*@presence.test` smoke accounts left in place).

## Notes

- Not merged to `main` / no PR; deployed from `feat/calendar-and-follows` @ `02f45c3` images.
- **GATEGUARD_AUTH_SECRET rotation still pending** (carried over from prior deploys).
