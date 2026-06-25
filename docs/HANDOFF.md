# Lia — Handoff

_Last updated: 2026-06-25. Merged to `main`: scaffold, deploy artifacts, category + venue normalization, venue geo, and the Option-A live deploy. **On branch `feat/dark-mode-theme` (awaiting merge):** dark-mode toggle + UI bug fixes (deployed), and the auth slice (GateGuard) — backend done + GateGuard running on the box; demo-login UI still pending (see What's next)._

Where the project stands after the frontend + backend scaffold and the first feature slices.

## What exists

- **Docs**: tech-stack brief (`docs/event_discovery_mvp_technical_stack.md`), Apple-HIG design system (`design/DESIGN.md` + `design/screens/*.html`), scaffold plans (`docs/superpowers/plans/`).
- **Frontend** (`frontend/`): Next.js App Router + TS + Tailwind v4 + pnpm. Apple-HIG tokens in `app/globals.css`. Built screens: **Discovery** (live API data, category filter, geolocation "рядом со мной"), **event-detail** (`GET /events/{id}`, category chips + venue + Leaflet map), **create-event** (RHF + Zod, `POST /events`, category multi-select + venue pick-or-create typeahead), **`/map`** browse. **Light/dark theme toggle** (`useSyncExternalStore`, pre-hydration script, `.light`/`.dark` on `<html>`). AI-search is a stub; the "Войти" button is a disabled stub until the auth demo-login lands.
- **Backend** (`backend/`): Go modular monolith `github.com/Pashteto/lia` from `go-microservice-template`. PostgreSQL + PostGIS via docker-compose. Wired end-to-end (model → repository → service → swagger HTTP): **`events`**, **`categories`** (`GET /categories`; many-to-many via `event_categories`), **`venues`** (`GET`/`POST`/`PATCH /venues`, `GET /events/nearby`; events reference a loose `venue_id`, embed nested `venue`), and **auth** (`internal/http/auth`): `CheckAuth` validates a bearer token via **GateGuard** gRPC (`gatekeeper.go`, `TokenValidator` seam), JIT-provisions a local `users` row by email, and `POST /events` requires `jwt` (organizer = principal). `MockAuth` still bypasses it in the running demo. Remaining domains (`organizers`, `rsvp`, `search`, `notifications`, `ai`) are `doc.go` skeletons.

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

Live at **`https://lia.pashteto.com`** / **`https://api.lia.pashteto.com`** on **vds-ru215** (`193.32.188.7`, `ssh vdska2`) with the **real backend** since 2026-06-23 (no longer mock). Hand-managed Docker + nginx + certbot — a documented exception to the Terraform/Secrets-Manager standard, demo scope only. Detailed ops live in deploy memory + the runbooks below, not duplicated here.

- **Frontend**: image `lia-frontend` built on the box with `NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com`, runs `127.0.0.1:3001`. Update = rsync `frontend/` → `/opt/lia/frontend`, rebuild, `docker rm -f && docker run`.
- **Backend**: `docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d` in `/opt/lia/backend`. PostGIS (internal) + migrate + `backend-app` → `127.0.0.1:9080`. Creds in `.env.prod` (chmod 600, git-ignored).
- **⚠️ Auth currently ENFORCED but demo-login broken (2026-06-25):** prod runs `HTTP_MOCK_AUTH=false` (deployed with the auth backend) → `POST /events` requires a token (401 without). But `POST /auth/demo-login` **503s** because GateGuard's `SignInOAuth` panics — so **"create event" on the live demo is broken** (no token obtainable). Triage + **one-line revert to `MockAuth=true`** + next steps: [`superpowers/runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md`](superpowers/runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md).
- **GateGuard (auth, Phase B done)**: self-hosted copy of gateway.fm's GateGuard runs on the box (`backend-gateguard-1` + `backend-gateguard-redis-1`, gRPC `gateguard:9090`, internal-only) on Lia's `backend_default` network, **reusing Lia's Postgres** (separate `gateguard` DB). Compose `backend/docker-compose.gateguard.yml`. Lia points at it (`HTTP_GATEKEEPER_ADDRESS`) but does **not** call it yet (`MockAuth=true`). Runbook: [`superpowers/runbooks/2026-06-25-gateguard-deploy.md`](superpowers/runbooks/2026-06-25-gateguard-deploy.md).
- **Runbooks**: live cutover [`superpowers/runbooks/2026-06-23-vds-ru215-deploy.md`](superpowers/runbooks/2026-06-23-vds-ru215-deploy.md); GateGuard [`superpowers/runbooks/2026-06-25-gateguard-deploy.md`](superpowers/runbooks/2026-06-25-gateguard-deploy.md).
- The box is **flaky** (1.9 GB RAM, intermittent SSH, broken IPv6) — run long builds detached + poll. The old **oracle-1** demo (frontend-only mock) is orphaned (DNS repointed); tear down later.

**Recently done:**

- **Auth slice — GateGuard, Phases A+B** (branch `feat/dark-mode-theme`, awaiting merge). Lia is a resource server validating GateGuard-issued JWTs. **A (backend, done, TDD'd):** `internal/http/auth` — `CheckAuth` calls `GateguardService.CheckAuth` over gRPC via a `TokenValidator` seam (`gatekeeper.go`), maps `User→Claims`, JIT-provisions a local user by email; `POST /events` now requires `jwt` (organizer = principal, client-supplied `organizer_id` ignored). Gatekeeper proto vendored to `protocols/gateguard` (generated client force-committed). **B (deploy, done):** self-hosted GateGuard runs on vds-ru215 (gRPC `:9090`, reuses Lia's Postgres `gateguard` DB + a Redis). **`MockAuth=true` stays on** → demo unchanged until Phase C. Login decided = **demo-login (no Google)** via `SignInOAuth`. Spec/plan/runbook: [`superpowers/specs/2026-06-25-auth-gatekeeper-design.md`](superpowers/specs/2026-06-25-auth-gatekeeper-design.md), [`superpowers/plans/2026-06-25-auth-gatekeeper.md`](superpowers/plans/2026-06-25-auth-gatekeeper.md), [`superpowers/runbooks/2026-06-25-gateguard-deploy.md`](superpowers/runbooks/2026-06-25-gateguard-deploy.md).
- **Dark mode + UI fixes** (branch `feat/dark-mode-theme`, **deployed**). Light/dark **theme toggle** (`ThemeToggle` via `useSyncExternalStore` + pre-hydration script). Bug fixes, all browser-verified live: event-detail crash (Leaflet marker-icon `iconUrl` undefined under Turbopack prod → every detail page 500'd); dead "Войти" → disabled stub; duplicate "Рядом" filter chip removed; date hydration mismatch (#418) fixed by pinning `Europe/Moscow` in `formatEventDate`.
- **Venue geo** (merged to `main`). The deferred geo half of venues: `venues.lat`/`lon` (`*float64`, nullable) + a PostGIS generated `geog geography(Point,4326)` column + GIST index (migration `000009`); `PATCH /venues/{id}` (dedicated `VenueUpdateInput`, **preserve-on-omit** coord semantics); `GET /events/nearby?lat&lon&limit` (PostGIS `ST_DWithin`/`<->`, nearest-first, **50 km cap**, published-only, excludes coordless venues; per-event `distance_m`). Frontend: a Leaflet (`+ OSM tiles`) map wrapper, **browser-side OSM Nominatim** address search (backend never geocodes — no keys/server egress), a venue **pin-picker** modal, an **event-detail venue map**, a Discovery **"рядом со мной"** distance-sorted list, and a **`/map`** browse screen (pins capped at 100). **Data-handling:** browser-only egress (addresses → Nominatim, tiles → OSM); public venue data, no PII, no secrets. Verified: `go build/vet/test`; live API end-to-end (coord create/PATCH-preserve/nearby ordering + 50 km cap + no `distance_m` leak); `pnpm lint`/`build`; SSR-safety of all maps. **Not automation-verified:** pixel-level Leaflet canvas render (no Playwright in the build env — data flow + SSR confirmed). Spec/plan: [`superpowers/specs/2026-06-19-venue-geo-design.md`](superpowers/specs/2026-06-19-venue-geo-design.md), [`superpowers/plans/2026-06-19-venue-geo.md`](superpowers/plans/2026-06-19-venue-geo.md).
- **Venue normalization** (merged). The denormalized `events.venue_name`/`venue_metro` are now a dedicated **`venues`** entity: `venues` table + backfill (migration 000008), an `internal/venues` module with search + find-or-create, `GET`/`POST /venues`, events reference a loose `venue_id` (zero UUID = none, no FK) and embed a nested `venue`; frontend has a pick-or-create **typeahead** (`VenuePicker`) and renders venue name/metro. Identity only — **geo deferred** (see What's next). Verified end-to-end (live API + frontend SSR). Spec/plan: [`superpowers/specs/2026-06-14-venue-normalization-design.md`](superpowers/specs/2026-06-14-venue-normalization-design.md), [`superpowers/plans/2026-06-14-venue-normalization.md`](superpowers/plans/2026-06-14-venue-normalization.md).
- **Category normalization** (merged). Curated, many-to-many **categories** taxonomy: seeded `categories` table + `event_categories` join (migrations 000006/000007), `internal/categories` module, `GET /categories`, events embed `categories[]` / accept `category_ids`; frontend multi-select picker + chips. Spec/plan: [`superpowers/specs/2026-06-13-category-normalization-design.md`](superpowers/specs/2026-06-13-category-normalization-design.md). The frontend demo was **redeployed 2026-06-13** with the multi-category build.

## What's next

1. **Auth — Phase C (demo-login + flip)**. Backend: a demo-login entrypoint that calls `GateguardService.SignInOAuth(User{email,name})` → JWT → httpOnly cookie on the lia domain (decide: plain HTTP handler vs swagger endpoint). Frontend: "Войти" form, attach `Bearer` to API calls, gate `/events/new`, sign-out. Then **flip `HTTP_MOCK_AUTH=false`**, redeploy `app`, verify `POST /events` (401 anon, 201 after login). ⚠️ Access-control change → document for ISO 27001 / Vanta. Real Google OAuth + presto front is the later upgrade (needs Google creds — out of demo scope).
2. **RSVP** — the detail "Записаться" button is a stub; needs the `rsvp` module (sits behind auth).
3. **Images** — disk+nginx (demo) or S3 upload + cover URLs on events (model has no cover field yet). **Deferred behind auth** (upload must not be an open write surface).
4. **AI-search screen** + `internal/ai` module. Search-only over real events (no hallucination). **Provider needs sign-off** — GigaChat / YandexGPT defaults; OpenAI/Anthropic only if legally/payment-wise permitted (org data-handling rules).

## Known gotchas (don't re-discover these)

- **Template codegen**: after `make rename`, run `make generate-all` (go-swagger server + protobuf) before `go build` — that code is gitignored and regenerated in CI.
- **`rename.sh` regex**: the template's rejected dots/uppercase; relaxed to accept full Go module paths. Dockerfile binary-copy path and the `COPY .git` line were also fixed for the monorepo-subdir layout.
- **go-pg + gofrs UUID**: cannot scan SQL `NULL` into a uuid field. "Unset" organizer/venue is the **zero UUID** (`NOT NULL DEFAULT`), and `events.Create` avoids `RETURNING *`. (Nullable non-uuid columns like `venues.lat`/`lon` are fine as `*float64`.)
- **go-pg raw `Query` skips hooks**: `events.repository.Nearby` scans events via raw SQL (for the PostGIS `distance_m`), so `AfterSelect` (which maps `StatusSQL`→`Status`) must be called manually per row, else `status` reads as the zero value. Prefer the model-based load (`db.Model(&x).Where(...).Select()`) when you don't need raw SQL — `loadVenues` uses it so nested-venue columns (incl. `lat`/`lon`) can't silently drift.
- **swagger nullable fields**: declare optional `number` fields (e.g. coords, `distance_m`) with `x-nullable: true` so go-swagger generates `*float64` (+ `omitempty`) — otherwise unset values serialize as `0` and a partial `PATCH` zeroes stored data. Create vs update use distinct bodies (`VenueInput` requires `name`; `VenueUpdateInput` doesn't).
- **golangci-lint**: CI installs **v1** (the `.golangci.yml` is v1 format) — do **not** migrate it to v2. Locally, install v1 to lint as CI does.
- **Local Docker**: Docker Desktop was unstable in dev; Postgres stayed up while the app container died. Workaround: run the app binary on the host (`go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env) against the containerized Postgres.
- **Box IPv6 / github in Docker builds**: vds-ru215's IPv6 is broken, so a Dockerfile that `curl`s github (e.g. GateGuard downloading protoc) hangs → `curl: (28) SSL connection timeout`. Force IPv4: `sed -i "s#curl -OL#curl -4 --retry 5 ...#g" Dockerfile`. `go mod download` is fine (goes via `proxy.golang.org`, not github).
- **Auth / GateGuard**: Lia validates tokens via GateGuard `CheckAuth` gRPC — the token is opaque to Lia (no key sharing). The `gatekeeper.go` `TokenValidator` is the only seam. GateGuard reuses Lia's Postgres (separate `gateguard` DB) on the box. Vendored GateGuard `*.pb.go` is force-committed (overrides the `*.pb.go` gitignore) since those protos don't match Lia's buf-lint conventions — CI proto-regen may need a lint exclusion for `protocols/gateguard`. Generated locally with `protoc` (buf not required).

## Verification done

- Frontend: `pnpm lint` + `pnpm build` clean; Discovery/detail SSR checked; create-event flow verified end-to-end with Playwright (fill → submit → redirect → detail).
- Backend: `go build/vet/test ./...` pass; CI-equivalent `golangci-lint` (v1) exits 0; `docker compose up` + live API exercised (create/list/get/filter/validation).
