# Lia — Handoff

_Last updated: 2026-06-25. Merged to `main`: scaffold, deploy artifacts, category + venue normalization, venue geo, and the Option-A live deploy. **On branch `feat/presence-tarski-storage-quota` (PR to main pending):** product rename (Presence.Tarski), auth complete + LIVE, swappable blob storage, event cover images, per-user monthly quota, orphaned-file cleanup._

Where the project stands after the frontend + backend scaffold and the first feature slices.

## What exists

- **Docs**: tech-stack brief (`docs/event_discovery_mvp_technical_stack.md`), Apple-HIG design system (`design/DESIGN.md` + `design/screens/*.html`), scaffold plans (`docs/superpowers/plans/`).
- **Frontend** (`frontend/`): Next.js App Router + TS + Tailwind v4 + pnpm. Apple-HIG tokens in `app/globals.css`. Built screens: **Discovery** (live API data, category filter, geolocation "рядом со мной"), **event-detail** (`GET /events/{id}`, category chips + venue + Leaflet map), **create-event** (RHF + Zod, `POST /events`, category multi-select + venue pick-or-create typeahead, cover image upload), **`/map`** browse. **Light/dark theme toggle** (`useSyncExternalStore`, pre-hydration script, `.light`/`.dark` on `<html>`). **"Войти" modal** (demo-login, hydration-safe via `useSyncExternalStore`); Bearer token attached to API calls; create-event gated behind auth. AI-search is a stub.
- **Backend** (`backend/`): Go modular monolith `github.com/Pashteto/lia` from `go-microservice-template`. PostgreSQL + PostGIS via docker-compose. Wired end-to-end: **`events`**, **`categories`**, **`venues`**, **`files`** (migration 000010: `files` table; `POST /api/v1/uploads` bearer-required, MIME allowlist, 5 MB cap; `GET /api/v1/files/{key}` public read), **`auth`** (`CheckAuth` via GateGuard gRPC + JIT user provision; `POST /auth/demo-login` → JWT; `HTTP_MOCK_AUTH=false` enforced). Event cover images (`events.cover_file_id` migration 000011; `cover_url` resolved via storage). Per-user **monthly event quota** (10/month, Europe/Moscow boundary, 429 with RU copy). Daily **orphaned-file cleanup** (`FILE_CLEANUP_ENABLED`, 24 h grace, logged) + `lia files:cleanup` CLI. Remaining domains (`organizers`, `rsvp`, `search`, `notifications`, `ai`) are `doc.go` skeletons.

A full **create → list → detail** loop (with auth + cover image) works against the real API (verified live).

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

- **Frontend**: image `lia-frontend` built on the box with `NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com`, runs `127.0.0.1:3001`. Display name is **Presence.Tarski** (live title confirmed: "Presence.Tarski — События"). Go module, DB names, container names, and the lia.pashteto.com domains are intentionally unchanged.
- **Backend**: `docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d` in `/opt/lia/backend`. PostGIS (internal) + migrate + `backend-app` → `127.0.0.1:9080`. Creds in `.env.prod` (chmod 600, git-ignored). Key env: `HTTP_MOCK_AUTH=false`, `STORAGE_BACKEND=local` (default), `STORAGE_PUBLIC_BASE=https://api.lia.pashteto.com/api/v1/files`, `EVENTS_MONTHLY_LIMIT=10`, `FILE_CLEANUP_ENABLED=true`. Uploads persisted via `lia_uploads` Docker volume → `/data/uploads` in the container.
- **GateGuard (auth, DONE)**: self-hosted GateGuard on the box (`backend-gateguard-1` + `backend-gateguard-redis-1`, gRPC `gateguard:9090`, internal-only), reusing Lia's Postgres (`gateguard` DB). Compose `backend/docker-compose.gateguard.yml`. Runbook: [`superpowers/runbooks/2026-06-25-gateguard-deploy.md`](superpowers/runbooks/2026-06-25-gateguard-deploy.md).
- **Runbooks**: live cutover [`superpowers/runbooks/2026-06-23-vds-ru215-deploy.md`](superpowers/runbooks/2026-06-23-vds-ru215-deploy.md); GateGuard [`superpowers/runbooks/2026-06-25-gateguard-deploy.md`](superpowers/runbooks/2026-06-25-gateguard-deploy.md).
- The box is **flaky** (1.9 GB RAM, intermittent SSH, broken IPv6) — run long builds detached + poll. The old **oracle-1** demo (frontend-only mock) is orphaned (DNS repointed); tear down later.

**Compliance note (ISO 27001 / Vanta):** Auth is enforced in prod (`HTTP_MOCK_AUTH=false`); GateGuard validates JWTs. The demo-login endpoint is a **non-production control** (mints a JWT for any email, no password) — must never be enabled in real prod. Uploads are authenticated, MIME-allowlisted (byte-sniff), size-capped (5 MB) — not an open write surface. `GET /files/{key}` is intentionally public read (event cover images are public content) — documented decision. The cleanup job is destructive but narrow (only unreferenced blobs >24 h, logged). S3 secrets via env only.

**Recently done:**

- **Presence.Tarski rename + storage/quota/cleanup/auth** (branch `feat/presence-tarski-storage-quota`, 26 commits, 2026-06-25, verified live). All items below deployed and working on vds-ru215:
  - **Product rename** to Presence.Tarski (display only — module/DB/domain unchanged).
  - **Auth complete.** GateGuard `SignInOAuth` panic (`index out of range [2]`) fixed with valid Status+Role in the demo-login signer. `POST /auth/demo-login`→200 JWT; anon `POST /events`→401; authed→201. Frontend demo-login modal + Bearer attach + create-event gate, hydration-safe.
  - **Swappable blob storage** (`internal/storage`): `Storage` interface; local-disk backend (default, `lia_uploads` volume); `minio-go` v7.0.77 S3 backend (config-gated — swap to Yandex Object Storage / Selectel / VK Cloud / Cloud.ru / MinIO via `STORAGE_BACKEND=s3` + `S3_*`, region `us-east-1`).
  - **Files domain**: migration 000010; `POST /api/v1/uploads` (bearer-required, image MIME allowlist, 5 MB cap); `GET /api/v1/files/{key}` (public).
  - **Event cover images**: `events.cover_file_id` + `users.avatar_file_id` (migration 000011, avatar column only — no avatar UI yet); `cover_url` resolved via storage. Frontend uploads + renders covers.
  - **Event quota**: 10/month/user, Europe/Moscow boundary, `EVENTS_MONTHLY_LIMIT` configurable → HTTP 429 «Достигнут лимит: 10 событий в месяц».
  - **Orphaned-file cleanup**: daily in-process job (`FILE_CLEANUP_ENABLED`, 24 h grace) + `lia files:cleanup` CLI.
  - Spec: [`superpowers/specs/2026-06-25-storage-quota-cleanup-design.md`](superpowers/specs/2026-06-25-storage-quota-cleanup-design.md). Plan: [`superpowers/plans/2026-06-25-storage-quota-cleanup.md`](superpowers/plans/2026-06-25-storage-quota-cleanup.md).
  - **Non-blocking follow-ups**: `s3.Delete` lacks `NoSuchKey` guard (S3 DeleteObject is idempotent anyway); `toStorageSettings` duplicated in `cmd/cleanup/cleanup.go` + `internal/application.go` (import cycle blocks extraction); S3 env-var prefix inconsistency (`S3_*` vs `STORAGE_S3_USE_SSL`) documented in `.env.prod.example`; 429 has no `Retry-After`; `uploadFile()` no runtime response-shape validation; `Storage.resolve()` symlink guard is path-string only (demo-tier). ~12 test events (`*@presence.test` owners) left on prod DB intentionally — do NOT delete them.

- **Auth slice — GateGuard, Phases A+B** (landed as part of `feat/dark-mode-theme`; auth final fix + Phase C shipped in `feat/presence-tarski-storage-quota` above). Lia is a resource server validating GateGuard-issued JWTs. `CheckAuth` calls `GateguardService.CheckAuth` over gRPC via `TokenValidator` seam (`gatekeeper.go`), maps `User→Claims`, JIT-provisions a local user by email. Gatekeeper proto vendored to `protocols/gateguard` (generated client force-committed). Spec/plan: [`superpowers/specs/2026-06-25-auth-gatekeeper-design.md`](superpowers/specs/2026-06-25-auth-gatekeeper-design.md).
- **Dark mode + UI fixes** (branch `feat/dark-mode-theme`, deployed). Light/dark **theme toggle** (`ThemeToggle` via `useSyncExternalStore` + pre-hydration script). Bug fixes: event-detail Leaflet marker-icon crash; Turbopack prod 500; duplicate "Рядом" filter chip; date hydration mismatch pinned to `Europe/Moscow`.
- **Venue geo** (merged to `main`). Migration `000009`; `PATCH /venues/{id}` (preserve-on-omit coords); `GET /events/nearby` (PostGIS `ST_DWithin`/`<->`, 50 km cap, per-event `distance_m`); Leaflet map wrapper; Nominatim address search; venue pin-picker; Discovery "рядом со мной"; `/map` screen. Spec/plan: [`superpowers/specs/2026-06-19-venue-geo-design.md`](superpowers/specs/2026-06-19-venue-geo-design.md), [`superpowers/plans/2026-06-19-venue-geo.md`](superpowers/plans/2026-06-19-venue-geo.md).
- **Venue normalization** (merged). Spec/plan: [`superpowers/specs/2026-06-14-venue-normalization-design.md`](superpowers/specs/2026-06-14-venue-normalization-design.md), [`superpowers/plans/2026-06-14-venue-normalization.md`](superpowers/plans/2026-06-14-venue-normalization.md).
- **Category normalization** (merged). Spec/plan: [`superpowers/specs/2026-06-13-category-normalization-design.md`](superpowers/specs/2026-06-13-category-normalization-design.md).

## What's next

1. **RSVP** — the detail "Записаться" button is a stub; needs the `rsvp` module (sits behind auth).
2. **AI-search screen** + `internal/ai` module. Search-only over real events (no hallucination). **Provider needs sign-off** — GigaChat / YandexGPT defaults; OpenAI/Anthropic only if legally/payment-wise permitted (org data-handling rules). See [[lia-ai-provider-constraint]].

## Known gotchas (don't re-discover these)

- **Template codegen**: after `make rename`, run `make generate-all` (go-swagger server + protobuf) before `go build` — that code is gitignored and regenerated in CI.
- **`rename.sh` regex**: the template's rejected dots/uppercase; relaxed to accept full Go module paths. Dockerfile binary-copy path and the `COPY .git` line were also fixed for the monorepo-subdir layout.
- **go-pg + gofrs UUID**: cannot scan SQL `NULL` into a uuid field. "Unset" organizer/venue is the **zero UUID** (`NOT NULL DEFAULT`), and `events.Create` avoids `RETURNING *`. (Nullable non-uuid columns like `venues.lat`/`lon` are fine as `*float64`.)
- **go-pg raw `Query` skips hooks**: `events.repository.Nearby` scans events via raw SQL (for the PostGIS `distance_m`), so `AfterSelect` (which maps `StatusSQL`→`Status`) must be called manually per row. Prefer the model-based load when you don't need raw SQL — `loadVenues` uses it so nested-venue columns can't silently drift.
- **swagger nullable fields**: declare optional `number` fields (e.g. coords, `distance_m`) with `x-nullable: true` so go-swagger generates `*float64` (+ `omitempty`) — otherwise unset values serialize as `0` and a partial `PATCH` zeroes stored data. Create vs update use distinct bodies (`VenueInput` requires `name`; `VenueUpdateInput` doesn't).
- **golangci-lint**: CI installs **v1** (the `.golangci.yml` is v1 format) — do **not** migrate it to v2. Locally, install v1 to lint as CI does.
- **Local Docker**: Docker Desktop was unstable in dev; Postgres stayed up while the app container died. Workaround: run the app binary on the host (`go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env) against the containerized Postgres.
- **Box IPv6 / github in Docker builds**: vds-ru215's IPv6 is broken → `curl: (28) SSL connection timeout` in Dockerfiles that curl github. Force IPv4: patch `curl -4 --retry 5`. `go mod download` is fine (proxy.golang.org, not github).
- **Auth / GateGuard**: Lia validates tokens via GateGuard `CheckAuth` gRPC — token is opaque to Lia (no key sharing). The `gatekeeper.go` `TokenValidator` is the only seam. GateGuard reuses Lia's Postgres (`gateguard` DB). Vendored `*.pb.go` is force-committed (overrides gitignore) — CI proto-regen may need a lint exclusion for `protocols/gateguard`.
- **FROM scratch + timezone**: the prod image is `FROM scratch`; the `time` package has no embedded tz data. The quota logic uses `Europe/Moscow` — must keep `_ "time/tzdata"` in the binary (currently in `cmd/lia.go` and `cmd/cleanup/cleanup.go`). **Do not remove it.**
- **minio-go version**: stay on **v7.0.77** — it is the highest version compatible with `go 1.24.0` in the current `go.mod`. Later minio-go releases require newer `go` directives; bumping would break the build.

## Verification done

- Frontend: `pnpm lint` + `pnpm build` clean; auth modal hydration-safe; create-event + cover upload flow verified live.
- Backend: `go build/vet/test ./...` pass; `golangci-lint` (v1) exits 0; `docker compose up` + live API exercised: auth (demo-login→JWT, 401 anon, 201 authed), upload→201, serve→200 image/png, quota (10×201, 11th→429), cleanup job (ran on startup, 0 orphans).
