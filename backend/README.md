# Backend

API and services for the Lia event-discovery MVP.

Stack: **Go modular monolith** (built from the `go-microservice-template`) with PostgreSQL + PostGIS, as described in [`../docs/event_discovery_mvp_technical_stack.md`](../docs/event_discovery_mvp_technical_stack.md). Module path: `github.com/Pashteto/lia`, binary `lia`.

> **One service, many modules — not microservices.** Despite the template name, this runs as a single Go service with clearly separated domain modules under `internal/`. Split a module into its own service only if real load or team structure demands it.

## Run locally

```bash
docker compose up -d --build      # PostGIS + migrations + app
curl localhost:8080/api/v1/health
curl localhost:8080/api/v1/events
docker compose down               # to reset the DB: also rm -rf data/
```

The app serves REST on `:8080` (base path `/api/v1/`) and gRPC on `:9090`. Local compose credentials (`dev`/`dev`) are **non-production**; production secrets come from AWS Secrets Manager via the deploy pipeline — never commit real secrets or put them in compose.

## Common commands

```bash
make build                 # build the lia binary (ldflags versioning)
make generate-api          # regenerate go-swagger server from api/swagger.yaml
make proto-generate-all    # regenerate protobuf (protoc)
make migrate-create NAME=x # new migration pair under db/migrations
go test ./...              # unit tests
```

## Domain modules (`internal/`)

| Module | Status | Notes |
|--------|--------|-------|
| `events` | ✅ **worked example** | End-to-end: migration → `models.Event` → `events.Repository` (go-pg) → `events.Service` → swagger HTTP handlers (`GET /events`, `GET /events/{id}`, `POST /events`). Wired in `application.go` via `http.Module.SetEventsService`. |
| `organizers`, `venues`, `users`, `rsvp`, `search`, `notifications`, `ai` | 📦 skeleton | `doc.go` package stubs marking responsibility + roadmap. Implement following the `events` pattern. |

The template's central `repository`/`service` layers carry the original `users` example (fetched via the gRPC client). New domains follow the self-contained `events` pattern: a `Repository` over the shared `*pg.DB` (exposed by `repository.Module.DB()`), a `Service`, and handlers registered in `internal/http/module.go`.

## How this was scaffolded from the template

1. Copied the template into `backend/` (excluding `.git`).
2. `make rename NEW_NAME=github.com/Pashteto/lia` → module path, binary, entrypoint `cmd/lia.go`, swagger API struct.
   - The template's `scripts/rename.sh` regex was relaxed to accept full Go module paths (host + uppercase + dots).
   - Fixed the Dockerfile binary copy destination (`/lia`) and dropped the `COPY .git` line (no `.git` in the monorepo subdir).
3. Swapped compose Postgres → `postgis/postgis:16-3.4`, DB `lia_dev`, added a healthcheck + ordered startup.
4. Added migrations `000003_enable_postgis`, `000004_events_table`; built the `events` module and skeleton `doc.go` stubs.

> Code is in English; product/API content is Russian where user-facing.
