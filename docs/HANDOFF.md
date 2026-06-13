# Lia — Handoff

_Last updated: 2026-06-13. PR #1 (scaffold) is **merged** to `main` (`a23dcb7`); base branch is now `main`._

Where the project stands after the frontend + backend scaffold and the first feature slices.

## What exists

- **Docs**: tech-stack brief (`docs/event_discovery_mvp_technical_stack.md`), Apple-HIG design system (`design/DESIGN.md` + `design/screens/*.html`), scaffold plans (`docs/superpowers/plans/`).
- **Frontend** (`frontend/`): Next.js App Router + TS + Tailwind v4 + pnpm. Apple-HIG tokens in `app/globals.css`. Built screens: **Discovery** (live API data), **event-detail** (`GET /events/{id}`), **create-event** (RHF + Zod, `POST /events`). AI-search is a stub.
- **Backend** (`backend/`): Go modular monolith `github.com/Pashteto/lia` from `go-microservice-template`. PostgreSQL + PostGIS via docker-compose. **`events`** domain wired end-to-end (model → repository → service → swagger HTTP). Other domains (`organizers`, `venues`, `users`, `rsvp`, `search`, `notifications`, `ai`) are `doc.go` skeletons.

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

## What's next (not started)

1. **AI-search screen** + `internal/ai` module. Per the tech-stack doc the assistant is **search-only over real events** (no event hallucination). **Provider needs sign-off** before wiring — GigaChat / YandexGPT are the documented defaults; OpenAI/Anthropic only if legally/payment-wise permitted for this project (and per org data-handling rules).
2. **Auth + RSVP**. The detail "Записаться" button is a stub. Needs the `rsvp` module and replacing `HTTP_MOCK_AUTH=true` with real auth (email magic-link / OTP) — a security-surface change; review deliberately (touches access/audit controls).
3. **Normalize category/venue** into dedicated `categories` / `venues` modules (currently denormalized text columns on `events`). `venues` unlocks the PostGIS "events nearby" work.
4. **Images** — S3 upload + cover URLs on events (model has no cover field yet).

## Known gotchas (don't re-discover these)

- **Template codegen**: after `make rename`, run `make generate-all` (go-swagger server + protobuf) before `go build` — that code is gitignored and regenerated in CI.
- **`rename.sh` regex**: the template's rejected dots/uppercase; relaxed to accept full Go module paths. Dockerfile binary-copy path and the `COPY .git` line were also fixed for the monorepo-subdir layout.
- **go-pg + gofrs UUID**: cannot scan SQL `NULL` into a uuid field. "Unset" organizer/venue is the **zero UUID** (`NOT NULL DEFAULT`), and `events.Create` avoids `RETURNING *`.
- **golangci-lint**: CI installs **v1** (the `.golangci.yml` is v1 format) — do **not** migrate it to v2. Locally, install v1 to lint as CI does.
- **Local Docker**: Docker Desktop was unstable in dev; Postgres stayed up while the app container died. Workaround: run the app binary on the host (`go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env) against the containerized Postgres.

## Verification done

- Frontend: `pnpm lint` + `pnpm build` clean; Discovery/detail SSR checked; create-event flow verified end-to-end with Playwright (fill → submit → redirect → detail).
- Backend: `go build/vet/test ./...` pass; CI-equivalent `golangci-lint` (v1) exits 0; `docker compose up` + live API exercised (create/list/get/filter/validation).
