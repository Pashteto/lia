# Organizer Hub — Deploy

_2026-07-01. Full deploy of the «Организаторам» organizer-hub feature to the live demo
(`lia.pashteto.com` / `api.lia.pashteto.com` on vds-ru215). Backend + frontend both
build-on-Mac → `save | ssh | load` → cutover. Shipped from `main` @ `1cca22a` (merge of
`feat/organizer-hub`; **NO DB migration** — schema stays at 018)._

## What shipped

- **No DB migration.** Schema unchanged (still `18`). The backend change is an additive
  read-model field only.
- **New `backend-app` image** (`linux/amd64`, FROM scratch, 9.04 MB; id `2e1e8e8748b9`,
  tag `amd64-orghub`): the event-applications response (`GET /events/{id}/applications`)
  now carries an `applicant {uuid, name}` object. Enriched in `rsvp.ListApplications` via
  a batched `LoadApplicantNames` query (`SELECT uuid, name FROM users WHERE uuid IN (...)`,
  name only — **no email**), mirroring the `loadOrganizers` pattern. `RsvpToAPI` maps it.
  NOT applied to `MyApplications`/`MyPractices`. `swagger.yaml` gained the additive
  `applicant` property; generated models regenerated via `make generate-api` (kept local,
  gitignored — not committed).
- **New `lia-frontend` image** (216 MB; id `7173f5ffd635`, tag `amd64-orghub`): new
  `/organizer` hub («Организаторам») + `/organizer/applications` aggregated view (events +
  applicant count badge + named applicants with accept/decline, reusing `EventApplicationsPanel`);
  desktop header link «Мои события» → «Организаторам» (`AuthButton`); mobile tab bar 5th slot
  «Создать» → «Организаторам» (`TabBar`, new `GlyphOrganizer`); home-page standalone
  «Создать событие» button removed. Soft framing — no new permission gating.

## Procedure (as executed)

1. **(Skipped) DB backup** — no migration this deploy; image-swap is reversible via rollback
   tags. (The prod `pg_dump` was also gated by the auto-mode classifier; not needed here.)
2. **Build backend** on Mac (build context carries the gitignored generated swagger +
   `protocols/userservice/*.pb.go`, present from `make generate-api`):
   `cd backend && docker build --platform linux/amd64 -t backend-app:amd64-orghub .`
3. **Ship backend**: `docker save backend-app:amd64-orghub | gzip | ssh vdska2 'gunzip | docker load'`.
4. **Cutover backend**: `docker tag backend-app:latest backend-app:rollback-preorghub`
   (preserves `f9fa5f7a9c6d` = `amd64-datefilter`), `docker tag backend-app:amd64-orghub backend-app:latest`,
   then recreate with **ALL THREE** compose files (gateguard file mandatory):
   `cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d`
   → `backend-migrate-1` ran (no new migration, exited), `backend-app-1` recreated.
5. **Build frontend** on Mac:
   `cd frontend && docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64-orghub .`
   (verified `next build` compiled `/organizer` + `/organizer/applications` routes).
6. **Ship + cutover frontend**: `save | ssh | load`, then
   `docker tag lia-frontend:latest lia-frontend:rollback-preorghub` (preserves `a95f71f82ed4`),
   `docker tag lia-frontend:amd64-orghub lia-frontend:latest`,
   `docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`.
7. **Cleanup**: `docker image prune -f` + `docker builder prune -f`; trimmed two stale feature tags. Disk 59% (7.7 GB free).

## Verification (prod, live)

- Backend health `GET /api/v1/health` → **200**; migration `SELECT version,dirty` → **`18|f`** (unchanged).
- Anon `GET /events/{id}/applications` → **401** (organizer-gated, unaffected).
- Routes: `/` → 200, `/organizer` → 200, `/organizer/applications` → 200, `/events/new` → 200,
  `/me/calendar` → 200.
- `/organizer` SSR renders «Организаторам» heading, «Создать событие» (in-hub button),
  «Заявки участников», «Подписчики» placeholder.
- Home page: standalone «Создать событие» button **removed** (only source occurrence is the
  in-hub button in `app/organizer/page.tsx`; the string in home SSR is Next prefetch flight-data
  for the linked `/organizer` route, not a visible CTA). Mobile nav shows «Организаторам».

## NOT prod-verified (needs a verified org + a second applicant)

The full applicant-name flow end-to-end: an organizer with an application-mode event + a real
second user applying → the applicant's **name** + the count badge («Заявок: N · ожидают: M»)
showing in `/organizer/applications`, and accept/decline. Anon gating, route health, and the
hub/IA changes ARE verified. The auto-mode classifier blocks granting admin / arbitrary prod
writes from throwaway accounts, so the named-applicant visual path was not exercised live.

## Rollback

- **Backend**: `docker tag backend-app:rollback-preorghub backend-app:latest` then recreate
  (`up -d` with the 3 compose files). `f9fa5f7a9c6d`.
- **Frontend**: `docker tag lia-frontend:rollback-preorghub lia-frontend:latest && docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`. `a95f71f82ed4`.
- **DB**: nothing to roll back (no migration).

## Notes

- Merged to `main` (`1cca22a`, `--no-ff`). **`main` is ahead of `origin/main` and not yet
  pushed** (`git push origin main` when ready — includes this feature + prior unpushed commits).
- **GATEGUARD_AUTH_SECRET rotation still pending** (carried over from prior deploys).
