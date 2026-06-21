# Lia → live (Option A): real backend + DB behind the demo

**Date:** 2026-06-20
**Status:** Approved (design)
**Scope target:** Option A — real data, mock auth

## Goal

Serve `lia.pashteto.com` from a real Go backend + PostgreSQL/PostGIS instead of
the built-in mock data. Read-only discovery (feed, event detail, `/map`, "рядом
со мной") works against real seeded events. Create-event works, attributed to the
mock `test@example.com` organizer. No real user identity yet.

This is the smallest change that makes the public demo "not mock". Auth, RSVP, AI
search, cover images, and the organizer/user/notification domains are explicitly
deferred (see Out of Scope).

## Context / starting state

- `venue-geo` is **already merged to `main`** (commit `c5397ca`). Migrations run
  through `000009_venue_geo`. No branch merge is needed.
- `backend/docker-compose.yml` **already exists** and defines a working stack:
  - `postgres` — `postgis/postgis:16-3.4`, db `lia_dev`, creds `dev/dev`.
  - `migrate` — one-shot `migrate/migrate`, applies `db/migrations/*` then exits.
  - `app` — builds `backend/Dockerfile` (scratch image), `HTTP_MOCK_AUTH=true`,
    publishes `8080` (HTTP) and `9090` (gRPC).
- The Go binary embeds its swagger spec (`internal/http/server/embedded_spec.go`,
  loaded via `loads.Analyzed(httpserver.SwaggerJSON)`), so the `scratch` runtime
  image needs **no** filesystem spec — the container runs standalone.
- The frontend bakes `NEXT_PUBLIC_API_URL` at **build time** (`lib/api.ts`), and
  every fetch helper degrades to `MOCK_EVENTS` on error. The current demo build
  points at a dead port (`http://127.0.0.1:9`) so it always shows mock.
- Host **oracle-1** (`129.146.183.89`) is hand-managed (nginx + certbot + docker,
  no Terraform). `:8080` is taken by the unrelated "dollbuilder" project. Existing
  vhosts and containers must not be touched. This box is already a documented
  exception to the org Terraform/Secrets-Manager standard — it stays a demo
  exception, not a precedent for production.

## Architecture

```
oracle-1 (129.146.183.89)
├── :3001  lia-frontend            (existing; rebuilt with new API URL)
├── :8080  dollbuilder             (DO NOT TOUCH)
├── 127.0.0.1:9080  lia-backend    ← new: app container, host-bound to 9080→8080
└── (compose-internal) lia-postgres ← new: NOT published to host
                                       persistent named volume

nginx vhosts:
├── lia.pashteto.com        → 127.0.0.1:3001   (existing)
└── api.lia.pashteto.com    → 127.0.0.1:9080   ← new vhost + certbot cert

DNS (Namecheap, BasicDNS):
└── A record  api.lia  → 129.146.183.89        ← new
```

All additions are `lia-*`-prefixed and additive. Nothing rebinds dollbuilder's
`:8080`, and Postgres is never exposed off the compose network.

## Components

### A. Production compose override — `backend/docker-compose.prod.yml`

A committed override applied on the box with
`docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d`. It adjusts
only what differs from local dev:

- **postgres**: remove the host port mapping entirely (compose-internal only).
  Use a named volume for data persistence across restarts. Credentials and db
  name come from environment (see env file below), not the committed `dev/dev`.
- **app**: bind `127.0.0.1:9080:8080`; do **not** publish gRPC (`9090`). Set:
  - `HTTP_MOCK_AUTH=true`
  - `HTTP_CORS_ALLOWED_ORIGINS=https://lia.pashteto.com` (narrow from the `*`
    default — least privilege)
  - `DATABASE_*` from the env file
  - `restart: unless-stopped`
- **migrate**: unchanged; reads the same `DATABASE_*` env.

Secrets: an **uncommitted** `backend/.env.prod` on the box supplies
`DATABASE_USER` / `DATABASE_PASSWORD` / `DATABASE_NAME` (non-default values). It is
referenced via compose `env_file` / `${VAR}` interpolation and is git-ignored. No
real credentials are committed.

### B. Seed data — `backend/db/seed/seed.sql`

Idempotent SQL, committed, run once after the stack is healthy:

```
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  exec -T postgres psql -U "$DATABASE_USER" -d "$DATABASE_NAME" < db/seed/seed.sql
```

Contents:

- **~8 real Moscow venues** with accurate `lat`/`lon` (e.g. Garage Museum, GES-2,
  Tretyakov Gallery, Pushkin Museum, Zotov Centre, …), fixed UUIDs, so the map
  shows legitimate pins and "рядом со мной" returns real distances.
- **~8–10 events**, `status='published'` with `published_at` set, fixed UUIDs,
  each referencing a seeded `venue_id`. `organizer_id` left as the zero UUID
  (loose ref; matches the scaffold convention).
- **`event_categories`** join rows that resolve the category by slug, because the
  curated category UUIDs are generated with `gen_random_uuid()` in migration
  `000006` and are therefore not knowable ahead of time:
  ```sql
  INSERT INTO event_categories (event_id, category_id)
  SELECT '<event-uuid>', id FROM categories WHERE slug = 'lecture'
  ON CONFLICT DO NOTHING;
  ```
- Every statement is idempotent (`ON CONFLICT (…) DO NOTHING`) so re-running the
  file is safe. It is **not** wired as a compose service, so container restarts
  never re-seed.

### C. Frontend re-point

Rebuild the `lia-frontend` image on the box with the real API base URL baked in:

```
NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com
```

(Build arg / build-time env, since the value is inlined by Next at build.) The
existing mock fallback in `lib/api.ts` stays untouched as a safety net — but see
the verification note about not mistaking a silent fallback for success.

## Data flow

1. Browser → `https://lia.pashteto.com` (nginx → `lia-frontend:3001`).
2. Frontend SSR + client fetches → `https://api.lia.pashteto.com/api/v1/*`
   (nginx → `lia-backend:9080` → app `:8080`).
3. App → Postgres over the compose-internal network.
4. On any backend fetch failure the frontend falls back to `MOCK_EVENTS`.

## Error handling / failure modes

- **Backend down** → frontend silently shows mock data. This is desirable for
  uptime but dangerous for verification: a broken deploy can look "fine". Mitigate
  by verifying against an event **title unique to the seed** that does not exist in
  `lib/mock-events.ts`.
- **CORS** → narrowed to the frontend origin; a misconfigured origin surfaces as
  blocked requests → mock fallback (same symptom as above).
- **Migrate failure** → `app` won't start (compose `depends_on … service_completed_
  successfully`); fix migrations before app boots.
- **Postgres data loss on redeploy** → prevented by the named volume; never bind
  to an ephemeral path.

## Verification

1. `curl -s https://api.lia.pashteto.com/api/v1/events?status=published` returns
   the seeded events as JSON.
2. `curl -s "https://api.lia.pashteto.com/api/v1/events/nearby?lat=55.75&lon=37.62&limit=50"`
   returns events with `distance_m`, nearest-first.
3. `lia.pashteto.com` discovery feed, an event detail page, and `/map` render the
   seeded content — confirmed via a seed-unique event title (proves it is *not*
   the mock fallback).
4. Existing vhosts (`amphitheater`, `api.lindentar`, etc.) and dollbuilder
   containers are still up and untouched.

## Security / org-policy notes

- Postgres is never published off the compose network; `dev/dev` defaults are
  replaced by non-default creds from an uncommitted env file. No secrets in git.
- CORS narrowed to `https://lia.pashteto.com` (from the `*` default).
- oracle-1 remains a hand-managed demo exception to the Terraform/Secrets-Manager
  standard — justified by demo scope, not extended to production.
- Mock auth (`HTTP_MOCK_AUTH=true`) is intentional for Option A and must be called
  out as a known non-production control when this is reviewed.

## Out of scope (deferred)

- Real authentication / JWT / gatekeeper integration (replacing `HTTP_MOCK_AUTH`).
- RSVP, AI search (`internal/ai` + provider sign-off), text search (`internal/search`).
- Cover images / S3.
- `internal/organizers`, `internal/users`, `internal/notifications` domains.
```
