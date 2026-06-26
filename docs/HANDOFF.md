# Lia — Handoff

_Last updated: 2026-06-26. **All work below is on LOCAL `main`, UNPUSHED** (`origin/main` is behind; deployed to prod from locally-built images, not a pushed ref/PR). Live on https://lia.pashteto.com: scaffold, normalization, venue geo, Presence.Tarski rename, GateGuard auth, storage/quota/cleanup, password auth, Liquid Glass UI, **RSVP feature**, **event-edit/draft + publish-from-mine**, and the **moderation/admin foundation** (sub-projects 0+1). The full-stack deploy (RSVP + moderation, prod DB migrated 011→014) shipped 2026-06-26 — runbook `docs/superpowers/runbooks/2026-06-26-rsvp-moderation-fullstack-deploy.md`. **Lots of admin work remains** — see the roadmap: `docs/superpowers/plans/2026-06-26-admin-suite-roadmap.md`._

> **Operational must-knows (2026-06-26):**
> 1. **Recreating the backend `app` container REQUIRES all three compose files** incl. `-f docker-compose.gateguard.yml` — it alone sets `HTTP_GATEKEEPER_ADDRESS=gateguard:9090` + `HTTP_MOCK_AUTH`. Omit it → register/login + token validation 503 and mock-auth flips on. (Cost a live incident; fixed.)
> 2. **Rotate `GATEGUARD_AUTH_SECRET`** — exposed in a session transcript 2026-06-26.
> 3. Live admin: `poulissimo@gmail.com` (role set in the `gateguard` DB). Promotion is manual SQL (no UI yet).

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

- **Password auth + events-user-data + LIVE deploy** (branch `feat/passwords-gateguard-and-events-user-data`, 2026-06-25, verified live). Runbook: [`superpowers/runbooks/2026-06-25-passwords-deploy.md`](superpowers/runbooks/2026-06-25-passwords-deploy.md). Plan: [`superpowers/plans/2026-06-25-passwords-and-me-suite.md`](superpowers/plans/2026-06-25-passwords-and-me-suite.md).
  - **GateGuard vendored** into the repo at `/gateguard` (own Go module) and extended: `password_hash` + email-verification columns (migration `000011`), bcrypt `internal/pkg/password`, `SignUpWithPassword`/`SignInWithPassword`/`RequestEmailVerification`/`VerifyEmail` RPCs + service + handlers. Email verification is a **STUB** (token persisted, send only logged — replace before real prod; wire GateGuard's SMTP notificator). Stack upgraded to latest: **grpc v1.81, protobuf v1.36, Go 1.26** (proto regen with protoc-gen-go-grpc v1.6.x).
  - **Lia password endpoints**: `POST /auth/register` (409 on exists) + `POST /auth/login` (401 on bad creds) → `Signer.SignUpPassword`/`SignInPassword` → GateGuard over gRPC. Vendored Lia gateguard proto regenerated to match (force-committed pb.go).
  - **Events user data**: event responses carry an `organizer` object (uuid, name, avatar_url — **never email**, public surface) via batched `loadOrganizers`; `GET /events/mine` (jwt, all statuses incl. drafts) via `ListByOrganizer`. New events default to `status=published` (formatter) so an omitted status can't create an invisible draft.
  - **Frontend**: login modal gains password + register/login toggle; `/events/mine` "Мои события" page (drafts badged) + header nav link; organizer shown on cards; `next/image` allows the API host (cover fix).
  - **Verified live** on `https://lia.pashteto.com`: register→200, login→200, wrong-pw→401, authed create→201 (published), `/events/mine` returns it with organizer name.
  - **Deploy technique (important)**: the box can't pull `golang:1.25`/`1.26` over the AmneziaWG tunnel (large layers reset), so the `linux/amd64` images are **built on the Mac and shipped via `docker save|gzip|ssh|docker load`** + retagged to the compose names — not built on the box. `deploy/vpn-build-all.sh` (VPN-up→build-on-box→VPN-down, supersedes `vpn-install-deps.sh`) exists but is unreliable for GateGuard due to that pull.

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

- **Moderation/admin foundation (sub-projects 0+1)** (2026-06-26, LIVE). RBAC via GateGuard's existing `admin` role (Approach 1 — no GateGuard reship): `gatekeeper.go` stops dropping the role, `Auth.Authenticate` returns the domain user with `.Role` synced to `users.role` (migration `000014`) every request. Plain `net/http` admin handler (`internal/http/admin`) mounted ahead of the swagger mux with a `requireStaff` gate (401 anon / 403 «Недостаточно прав» / `admin` through). Post-moderation `internal/moderation`: take-down (`published→rejected`, reason req'd) / reinstate, each writing `event_status_history` + `audit_log` in one tx; `GET /auth/me`, `/admin/overview`, `/admin/moderation/events`, `…/takedown`, `…/reinstate`. Frontend: gated `/admin` shell + overview + moderation queue + «Админ» nav link + «Снято модератором» badge. Built subagent-driven (per-task + final whole-branch review, READY TO MERGE). Spec/plan: [`superpowers/specs/2026-06-26-moderation-admin-foundation-design.md`](superpowers/specs/2026-06-26-moderation-admin-foundation-design.md), [`superpowers/plans/2026-06-26-moderation-admin-foundation.md`](superpowers/plans/2026-06-26-moderation-admin-foundation.md). **Remaining admin work** (org verification, complaints, featured, taxonomy admin, moderator/admin split, deferred follow-ups): [`superpowers/plans/2026-06-26-admin-suite-roadmap.md`](superpowers/plans/2026-06-26-admin-suite-roadmap.md).
- **RSVP + event-edit/draft** (2026-06-26, LIVE; built by a parallel session). `rsvp` domain (sign-up/application/external modes, capacity, `.ics`), `event_rsvps` (migration `000013`) + event signup fields (`000012`), `/me/practices` + `/me/applications` + organizer accept/decline panel, event-detail signup CTA, `PATCH /events/{id}`, create-defaults-to-**draft** + publish-from-`/events/mine`, Discovery locked to `published`. **Hotfix (2026-06-26):** event create defaulted `signup_mode` to `''` (`use_zero`) violating `events_signup_mode_check` from `000012` → `POST /events` 503; fixed to default `open` in the create formatter (`internal/http/formatter/event.go`).
- **Liquid Glass refresh** (2026-06-26, LIVE, branch `feat/liquid-glass-refresh`). iOS-26 glass chrome + sliding sun/moon theme switch (replaces `ThemeToggle`). Runbook `superpowers/runbooks/2026-06-26-liquid-glass-frontend-redeploy.md`.
- **Auth slice — GateGuard, Phases A+B** (landed as part of `feat/dark-mode-theme`; auth final fix + Phase C shipped in `feat/presence-tarski-storage-quota` above). Lia is a resource server validating GateGuard-issued JWTs. `CheckAuth` calls `GateguardService.CheckAuth` over gRPC via `TokenValidator` seam (`gatekeeper.go`), maps `User→Claims`, JIT-provisions a local user by email. Gatekeeper proto vendored to `protocols/gateguard` (generated client force-committed). Spec/plan: [`superpowers/specs/2026-06-25-auth-gatekeeper-design.md`](superpowers/specs/2026-06-25-auth-gatekeeper-design.md).
- **Dark mode + UI fixes** (branch `feat/dark-mode-theme`, deployed). Light/dark **theme toggle** (`ThemeToggle` via `useSyncExternalStore` + pre-hydration script). Bug fixes: event-detail Leaflet marker-icon crash; Turbopack prod 500; duplicate "Рядом" filter chip; date hydration mismatch pinned to `Europe/Moscow`.
- **Venue geo** (merged to `main`). Migration `000009`; `PATCH /venues/{id}` (preserve-on-omit coords); `GET /events/nearby` (PostGIS `ST_DWithin`/`<->`, 50 km cap, per-event `distance_m`); Leaflet map wrapper; Nominatim address search; venue pin-picker; Discovery "рядом со мной"; `/map` screen. Spec/plan: [`superpowers/specs/2026-06-19-venue-geo-design.md`](superpowers/specs/2026-06-19-venue-geo-design.md), [`superpowers/plans/2026-06-19-venue-geo.md`](superpowers/plans/2026-06-19-venue-geo.md).
- **Venue normalization** (merged). Spec/plan: [`superpowers/specs/2026-06-14-venue-normalization-design.md`](superpowers/specs/2026-06-14-venue-normalization-design.md), [`superpowers/plans/2026-06-14-venue-normalization.md`](superpowers/plans/2026-06-14-venue-normalization.md).
- **Category normalization** (merged). Spec/plan: [`superpowers/specs/2026-06-13-category-normalization-design.md`](superpowers/specs/2026-06-13-category-normalization-design.md).

## What's next

1. **Admin suite — the bulk of remaining work.** Foundation (RBAC + event moderation) is done+live; still unbuilt: **organizer entity + verification** (heaviest — no `organizers` table exists yet), **complaints**, **featured curation**, **taxonomy admin**, the **moderator/admin role split** (Approach 2 — needs a GateGuard reship), and small deferred follow-ups (thread take-down reason into `/events/mine`, in-app role-promotion UI, audit-log viewer, queue pagination, CI integration tests). Full decomposition, dependencies, and recommended order: [`superpowers/plans/2026-06-26-admin-suite-roadmap.md`](superpowers/plans/2026-06-26-admin-suite-roadmap.md). Each sub-project gets its own brainstorm → spec → plan.
2. **Push `main` / open PRs** — everything since the passwords deploy is on local `main`, unpushed. Decide branch/PR strategy and push.
3. **Rotate `GATEGUARD_AUTH_SECRET`** (exposed in a 2026-06-26 transcript).
4. **AI-search screen** + `internal/ai` module. Search-only over real events (no hallucination). **Provider needs sign-off** — GigaChat / YandexGPT defaults; OpenAI/Anthropic only if legally/payment-wise permitted (org data-handling rules). See [[lia-ai-provider-constraint]].

## Known gotchas (don't re-discover these)

- **Template codegen**: after `make rename`, run `make generate-all` (go-swagger server + protobuf) before `go build` — that code is gitignored and regenerated in CI.
- **`rename.sh` regex**: the template's rejected dots/uppercase; relaxed to accept full Go module paths. Dockerfile binary-copy path and the `COPY .git` line were also fixed for the monorepo-subdir layout.
- **go-pg + gofrs UUID**: cannot scan SQL `NULL` into a uuid field. "Unset" organizer/venue is the **zero UUID** (`NOT NULL DEFAULT`), and `events.Create` avoids `RETURNING *`. (Nullable non-uuid columns like `venues.lat`/`lon` are fine as `*float64`.)
- **go-pg raw `Query` skips hooks**: `events.repository.Nearby` scans events via raw SQL (for the PostGIS `distance_m`), so `AfterSelect` (which maps `StatusSQL`→`Status`) must be called manually per row. Prefer the model-based load when you don't need raw SQL — `loadVenues` uses it so nested-venue columns can't silently drift.
- **swagger nullable fields**: declare optional `number` fields (e.g. coords, `distance_m`) with `x-nullable: true` so go-swagger generates `*float64` (+ `omitempty`) — otherwise unset values serialize as `0` and a partial `PATCH` zeroes stored data. Create vs update use distinct bodies (`VenueInput` requires `name`; `VenueUpdateInput` doesn't).
- **golangci-lint**: CI installs **v1** (the `.golangci.yml` is v1 format) — do **not** migrate it to v2. Locally, install v1 to lint as CI does.
- **Prod `app` recreate needs ALL THREE compose files**: `-f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml`. The gateguard file is the ONLY place the `app` service gets `HTTP_GATEKEEPER_ADDRESS=gateguard:9090` + `HTTP_MOCK_AUTH` (viper `AutomaticEnv`, `cmd/root/root.go`; no in-code env binding for the gatekeeper address — default `localhost:9091`). Recreate without it → register/login + token validation 503 (signer dials `127.0.0.1:9091`) and `HTTP_MOCK_AUTH` reverts to base dev default `true`. Caused a live incident 2026-06-26.
- **go-pg `use_zero` defeats DB column defaults**: a field tagged `pg:"col,use_zero"` writes the Go zero value (e.g. `""`) on INSERT instead of omitting the column, so a `DEFAULT 'x'` never applies. Bit `events.signup_mode` (`NOT NULL DEFAULT 'open'` + `CHECK in ('open','application','external')`) — empty create → check violation → 503. Fix: default the value in the create formatter (don't rely on the DB default for `use_zero` columns), or drop `use_zero`.
- **Local Docker**: Docker Desktop was unstable in dev; Postgres stayed up while the app container died. Workaround: run the app binary on the host (`go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env) against the containerized Postgres.
- **Box IPv6 / github in Docker builds**: vds-ru215's IPv6 is broken → `curl: (28) SSL connection timeout` in Dockerfiles that curl github. Force IPv4: patch `curl -4 --retry 5`. `go mod download` is fine (proxy.golang.org, not github).
- **Auth / GateGuard**: Lia validates tokens via GateGuard `CheckAuth` gRPC — token is opaque to Lia (no key sharing). The `gatekeeper.go` `TokenValidator` is the only seam. GateGuard reuses Lia's Postgres (`gateguard` DB). Vendored `*.pb.go` is force-committed (overrides gitignore) — CI proto-regen may need a lint exclusion for `protocols/gateguard`.
- **FROM scratch + timezone**: the prod image is `FROM scratch`; the `time` package has no embedded tz data. The quota logic uses `Europe/Moscow` — must keep `_ "time/tzdata"` in the binary (currently in `cmd/lia.go` and `cmd/cleanup/cleanup.go`). **Do not remove it.**
- **GateGuard grpc Unimplemented embed**: protoc-gen-go-grpc **v1.6+** generates a `testEmbeddedByValue` assertion — the gRPC handler (`GateguardHandlers`) MUST embed `UnimplementedGateguardServiceServer` **by value**, not pointer, or `RegisterGateguardServiceServer` nil-panics at startup. `go build` compiles either way; caught only at runtime.
- **GateGuard Go toolchain pin**: after the grpc v1.81 upgrade the deps require **Go ≥ 1.25** (Dockerfile base `golang:1.26`). The box can't pull `golang:1.25/1.26` over the tunnel (large layers `connection reset`) — build the amd64 images on a well-connected machine and `docker save|ssh|docker load` to the box (see runbook). `protoc-gen-go-grpc` must be **v1.6.x** to match grpc v1.81 (the old v1.2.0/Version7 panics).
- **genproto module split**: a `go get -u ./...` can leave an "ambiguous import … googleapis/rpc/status" (old monolith `google.golang.org/genproto` + new split `.../genproto/googleapis/rpc` both provide it). Fix: `go get google.golang.org/genproto@latest google.golang.org/genproto/googleapis/rpc@latest && go mod tidy`.
- **minio-go version**: stay on **v7.0.77** — it is the highest version compatible with `go 1.24.0` in the current `go.mod`. Later minio-go releases require newer `go` directives; bumping would break the build.

## Verification done

- Frontend: `pnpm lint` + `pnpm build` clean; auth modal hydration-safe; create-event + cover upload flow verified live.
- Backend: `go build/vet/test ./...` pass; `golangci-lint` (v1) exits 0; `docker compose up` + live API exercised: auth (demo-login→JWT, 401 anon, 201 authed), upload→201, serve→200 image/png, quota (10×201, 11th→429), cleanup job (ran on startup, 0 orphans).
- Frontend nav follow-up (2026-06-26): every secondary page now has a way back —
  `‹ События` back links on `/events/mine`, `/map`, `/admin`, `/me/practices`,
  `/me/applications` (`/search` already had one), and the mobile `TabBar` was
  lifted into `app/layout.tsx` (persistent, `sm:hidden`, hidden on create/detail/
  admin, `max-sm:pb-28` clearance). `tsc`/`eslint`/`next build` clean.
  **DEPLOYED live 2026-06-26** (frontend-only, no migration; verified 200 +
  back link in `/map` HTML) —
  `docs/superpowers/runbooks/2026-06-26-nav-back-buttons-frontend-redeploy.md`.
  Backend rebuild from this branch was attempted + rolled back: the branch tree
  lacks `/admin/*` so the image regressed the moderation API → reverted to the
  full-stack `backend-app:rollback`. DB did advance **14→16** (`015_organizers`,
  `016_app_settings`; additive tables, no serving code in the running image yet —
  harmless). **Deploy the backend from `main`, not this branch.** Pre-migration
  dump: `/opt/lia/backup-pre-organizers-20260626-1425.sql.gz`.
