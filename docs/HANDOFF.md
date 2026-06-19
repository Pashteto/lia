# Lia — Handoff

_Last updated: 2026-06-19. Merged to `main`: scaffold (PR #1), deploy artifacts (PR #2), category normalization, and venue normalization. On branch `feat/venue-geo` (awaiting merge): venue geo._

Where the project stands after the frontend + backend scaffold and the first feature slices.

## What exists

- **Docs**: tech-stack brief (`docs/event_discovery_mvp_technical_stack.md`), Apple-HIG design system (`design/DESIGN.md` + `design/screens/*.html`), scaffold plans (`docs/superpowers/plans/`).
- **Frontend** (`frontend/`): Next.js App Router + TS + Tailwind v4 + pnpm. Apple-HIG tokens in `app/globals.css`. Built screens: **Discovery** (live API data, category filter), **event-detail** (`GET /events/{id}`, category chips + venue), **create-event** (RHF + Zod, `POST /events`, category multi-select + venue pick-or-create typeahead). AI-search is a stub.
- **Backend** (`backend/`): Go modular monolith `github.com/Pashteto/lia` from `go-microservice-template`. PostgreSQL + PostGIS via docker-compose. Wired end-to-end (model → repository → service → swagger HTTP): **`events`**, **`categories`** (`GET /categories`; many-to-many via `event_categories`), and **`venues`** (`GET`/`POST /venues`; events reference a loose `venue_id`, embed nested `venue`). Remaining domains (`organizers`, `users`, `rsvp`, `search`, `notifications`, `ai`) are `doc.go` skeletons.

A full **create → list → detail** loop works against the real API (verified with Playwright).

## Run it

```bash
# Backend: PostGIS + migrations + app
cd backend && docker compose up -d --build
curl localhost:8080/api/v1/health

# Frontend (points at the backend via NEXT_PUBLIC_API_URL, default :8080)
cd frontend && pnpm install && pnpm dev   # http://localhost:3000
```

Frontend falls back to mock data (`lib/mock-events.ts`) when the backend is unreachable, so it renders standalone too.

## Deploy

- **Frontend demo**: `https://lia.pashteto.com` (host `oracle-1`, hand-managed nginx + docker — no Terraform). Frontend-only, serves mock data; **no backend is deployed anywhere**. `NEXT_PUBLIC_API_URL` is baked to a dead port so SSR/client fall back to mocks. Deploy image is `frontend/Dockerfile` (+ `.dockerignore`), now committed. Update = rsync `frontend/` to the box, rebuild, `docker rm -f && docker run` the `lia-frontend` container (`127.0.0.1:3001`).
- The box is **shared** with another project; specific vhosts/containers there are off-limits. Deploy/runbook detail lives outside the repo (deploy memory / ops notes), not in git.

**Recently done:**

- **Venue geo** (branch `feat/venue-geo`, awaiting merge). The deferred geo half of venues: `venues.lat`/`lon` (`*float64`, nullable) + a PostGIS generated `geog geography(Point,4326)` column + GIST index (migration `000009`); `PATCH /venues/{id}` (dedicated `VenueUpdateInput`, **preserve-on-omit** coord semantics); `GET /events/nearby?lat&lon&limit` (PostGIS `ST_DWithin`/`<->`, nearest-first, **50 km cap**, published-only, excludes coordless venues; per-event `distance_m`). Frontend: a Leaflet (`+ OSM tiles`) map wrapper, **browser-side OSM Nominatim** address search (backend never geocodes — no keys/server egress), a venue **pin-picker** modal, an **event-detail venue map**, a Discovery **"рядом со мной"** distance-sorted list, and a **`/map`** browse screen (pins capped at 100). **Data-handling:** browser-only egress (addresses → Nominatim, tiles → OSM); public venue data, no PII, no secrets. Verified: `go build/vet/test`; live API end-to-end (coord create/PATCH-preserve/nearby ordering + 50 km cap + no `distance_m` leak); `pnpm lint`/`build`; SSR-safety of all maps. **Not automation-verified:** pixel-level Leaflet canvas render (no Playwright in the build env — data flow + SSR confirmed). Spec/plan: [`superpowers/specs/2026-06-19-venue-geo-design.md`](superpowers/specs/2026-06-19-venue-geo-design.md), [`superpowers/plans/2026-06-19-venue-geo.md`](superpowers/plans/2026-06-19-venue-geo.md).
- **Venue normalization** (merged). The denormalized `events.venue_name`/`venue_metro` are now a dedicated **`venues`** entity: `venues` table + backfill (migration 000008), an `internal/venues` module with search + find-or-create, `GET`/`POST /venues`, events reference a loose `venue_id` (zero UUID = none, no FK) and embed a nested `venue`; frontend has a pick-or-create **typeahead** (`VenuePicker`) and renders venue name/metro. Identity only — **geo deferred** (see What's next). Verified end-to-end (live API + frontend SSR). Spec/plan: [`superpowers/specs/2026-06-14-venue-normalization-design.md`](superpowers/specs/2026-06-14-venue-normalization-design.md), [`superpowers/plans/2026-06-14-venue-normalization.md`](superpowers/plans/2026-06-14-venue-normalization.md).
- **Category normalization** (merged). Curated, many-to-many **categories** taxonomy: seeded `categories` table + `event_categories` join (migrations 000006/000007), `internal/categories` module, `GET /categories`, events embed `categories[]` / accept `category_ids`; frontend multi-select picker + chips. Spec/plan: [`superpowers/specs/2026-06-13-category-normalization-design.md`](superpowers/specs/2026-06-13-category-normalization-design.md). The frontend demo was **redeployed 2026-06-13** with the multi-category build.

## What's next

1. **AI-search screen** + `internal/ai` module. Per the tech-stack doc the assistant is **search-only over real events** (no event hallucination). **Provider needs sign-off** before wiring — GigaChat / YandexGPT are the documented defaults; OpenAI/Anthropic only if legally/payment-wise permitted for this project (and per org data-handling rules).
2. **Auth + RSVP**. The detail "Записаться" button is a stub. Needs the `rsvp` module and replacing `HTTP_MOCK_AUTH=true` with real auth (email magic-link / OTP) — a security-surface change; review deliberately (touches access/audit controls).
3. **Images** — S3 upload + cover URLs on events (model has no cover field yet).

## Known gotchas (don't re-discover these)

- **Template codegen**: after `make rename`, run `make generate-all` (go-swagger server + protobuf) before `go build` — that code is gitignored and regenerated in CI.
- **`rename.sh` regex**: the template's rejected dots/uppercase; relaxed to accept full Go module paths. Dockerfile binary-copy path and the `COPY .git` line were also fixed for the monorepo-subdir layout.
- **go-pg + gofrs UUID**: cannot scan SQL `NULL` into a uuid field. "Unset" organizer/venue is the **zero UUID** (`NOT NULL DEFAULT`), and `events.Create` avoids `RETURNING *`. (Nullable non-uuid columns like `venues.lat`/`lon` are fine as `*float64`.)
- **go-pg raw `Query` skips hooks**: `events.repository.Nearby` scans events via raw SQL (for the PostGIS `distance_m`), so `AfterSelect` (which maps `StatusSQL`→`Status`) must be called manually per row, else `status` reads as the zero value. Prefer the model-based load (`db.Model(&x).Where(...).Select()`) when you don't need raw SQL — `loadVenues` uses it so nested-venue columns (incl. `lat`/`lon`) can't silently drift.
- **swagger nullable fields**: declare optional `number` fields (e.g. coords, `distance_m`) with `x-nullable: true` so go-swagger generates `*float64` (+ `omitempty`) — otherwise unset values serialize as `0` and a partial `PATCH` zeroes stored data. Create vs update use distinct bodies (`VenueInput` requires `name`; `VenueUpdateInput` doesn't).
- **golangci-lint**: CI installs **v1** (the `.golangci.yml` is v1 format) — do **not** migrate it to v2. Locally, install v1 to lint as CI does.
- **Local Docker**: Docker Desktop was unstable in dev; Postgres stayed up while the app container died. Workaround: run the app binary on the host (`go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env) against the containerized Postgres.

## Verification done

- Frontend: `pnpm lint` + `pnpm build` clean; Discovery/detail SSR checked; create-event flow verified end-to-end with Playwright (fill → submit → redirect → detail).
- Backend: `go build/vet/test ./...` pass; CI-equivalent `golangci-lint` (v1) exits 0; `docker compose up` + live API exercised (create/list/get/filter/validation).
