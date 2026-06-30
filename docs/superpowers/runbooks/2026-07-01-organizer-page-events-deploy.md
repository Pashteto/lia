# Organizer Page Events + Clickable Host — Deploy

_2026-07-01. Full deploy to the live demo (`lia.pashteto.com` / `api.lia.pashteto.com` on
vds-ru215). Backend + frontend both build-on-Mac → `save | ssh | load` → cutover. Shipped
from `main` @ `8b17c79` (merge of `feat/organizer-page-events`; **NO DB migration**,
schema stays 018)._

## What shipped

- **No DB migration** (schema `18`). Additive read filter only.
- **New `backend-app` image** (`linux/amd64`, FROM scratch, 9.04 MB; id `305ffc1efdb6`,
  tag `amd64-orgpage`): public `GET /events` gains an `organizer_id` query param (a
  public `organizers.id` profile id). The handler resolves it via `organizers.GetByID`
  to the **verified** organizer's owner-user-id and threads it into
  `ListFilter.OrganizerIDs` (published-only). Unknown/unverified → empty `200 []`; a real
  lookup error → `503`; a malformed (non-uuid) id → `422` at the go-swagger binding layer
  (the handler's own 400 guard sits behind binding and is effectively unreachable —
  documented; the frontend degrades any non-2xx to `[]`).
- **New `lia-frontend` image** (216 MB; id `2349eeadf69c`, tag `amd64-orgpage`): the
  event-detail «Ведущий» row links to `/organizers/{profile_id}` when verified (whole
  row, not just the badge; badge no longer a nested link); the public organizer page
  `/organizers/[id]` lists **upcoming** (primary) + **past** (secondary, most-recent-first,
  display-capped ~10) published events via `fetchEventsByOrganizer` (one call, split
  client-side by `startsAt`).

## Procedure (as executed)

1. **(Skipped) DB backup** — no migration; image-swap is reversible via rollback tags.
2. **Build backend** on Mac (build context carries the gitignored generated swagger, incl.
   the regenerated `list_events_parameters.go` with `OrganizerID`):
   `cd backend && docker build --platform linux/amd64 -t backend-app:amd64-orgpage .`
3. **Ship + cutover backend**: `docker save … | gzip | ssh vdska2 'gunzip | docker load'`;
   `docker tag backend-app:latest backend-app:rollback-preorgpage` (preserves
   `2e1e8e8748b9` = `amd64-orghub`); `docker tag backend-app:amd64-orgpage backend-app:latest`;
   recreate with **ALL THREE** compose files:
   `cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d`
   (migrate ran, no new migration, exited; `backend-app-1` recreated).
4. **Build frontend** on Mac:
   `cd frontend && docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64-orgpage .`
   (verified `next build` compiled `/organizers/[id]`).
5. **Ship + cutover frontend**: `save | ssh | load`;
   `docker tag lia-frontend:latest lia-frontend:rollback-preorgpage` (preserves
   `7173f5ffd635`); `docker tag lia-frontend:amd64-orgpage lia-frontend:latest`;
   `docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`.
6. **Cleanup**: `docker image prune -f` + `docker builder prune -f`; trimmed stale tags. Disk 59%.

## Verification (prod, live)

- Backend health `GET /api/v1/health` → **200**.
- `GET /events?organizer_id=a125ade8-b12d-4a00-ba88-7466290fdd90` (Музей «Гараж», verified)
  → **6 published events** returned — the filter works end-to-end.
- `GET /events?organizer_id=00000000-…-0` (valid-shaped, unknown) → **200 `[]`** (no leak).
- `GET /events?organizer_id=not-a-uuid` (malformed) → **422** (binding-layer format reject).
- Routes: `/` → 200, `/organizers/{id}` → 200.

### NOT prod-verified by curl (client-rendered page)

`/organizers/[id]` is a `"use client"` component that renders its content **after** its
browser-side `useEffect` fetches resolve — so the raw SSR HTML (what `curl` sees) is empty
and does not contain the event lists or section headings. The **data path is confirmed**
(backend returns 6 events for the org) and the route is healthy, but the **visual render of
the upcoming/past lists and the clickable host row require a real browser** (JS execution)
to confirm — recommend a manual browser check on `https://lia.pashteto.com/organizers/a125ade8-b12d-4a00-ba88-7466290fdd90`.

## Rollback

- **Backend**: `docker tag backend-app:rollback-preorgpage backend-app:latest` then recreate
  with the 3 compose files. `2e1e8e8748b9`.
- **Frontend**: `docker tag lia-frontend:rollback-preorgpage lia-frontend:latest && docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`. `7373…` → `7173f5ffd635`.
- **DB**: nothing to roll back (no migration).

## Notes

- Merged to `main` (`8b17c79`, `--no-ff`). **`main` ahead of `origin/main`, not yet pushed**
  (includes this + prior unpushed work — `git push origin main` when ready).
- Spec/plan: `docs/superpowers/specs/2026-07-01-organizer-page-events-design.md`,
  `docs/superpowers/plans/2026-07-01-organizer-page-events.md`.
- **GATEGUARD_AUTH_SECRET rotation still pending** (carried over).
