# Backend Scaffold — Implementation Plan (executed)

**Goal:** Stand up the Lia backend as a Go **modular monolith** built from the `go-microservice-template`, with the `events` domain wired end-to-end (migration → model → repository → service → HTTP) and the remaining domains as documented skeletons.

**Status:** ✅ Done. `go build/vet/test ./...` pass; `docker compose up` runs PostGIS + migrations + app and the events endpoints work end-to-end.

> **Follow-on since this plan** (see [`../../HANDOFF.md`](../../HANDOFF.md)): enriched the `events` model with `category` / `venue_name` / `venue_metro` (migration 000005) so the discovery UI can show category + venue. CI-equivalent `golangci-lint` (v1) wired into the verification loop.

**Decisions:** Foundation + one worked example (`events`). Module `github.com/Pashteto/lia` (binary `lia`). Reuse the template's stack (chi-less go-swagger HTTP, go-pg, golang-migrate, protobuf).

---

## Task 1: Import the template
- `rsync -a --exclude='.git' --exclude='.idea' --exclude='data' /…/go-microservice-template/ backend/`.

## Task 2: Rename to the Lia module
- `make rename NEW_NAME=github.com/Pashteto/lia` → go.mod module, binary `lia`, `cmd/lia.go`, `GITVER_PKG`, swagger struct `LiaAPIAPI`, Dockerfile entrypoint.
- **Fix needed:** the template `scripts/rename.sh` regex rejected dots/uppercase; relaxed it to `^[A-Za-z0-9][A-Za-z0-9._-]*(/[A-Za-z0-9._-]+)*$` (valid Go module paths).
- `make generate-all` (protobuf + go-swagger server, the template's generated code is not committed) then `go build ./...`.

## Task 3: PostGIS + local compose
- `docker-compose.yml`: image → `postgis/postgis:16-3.4`, DB `lia_dev`; added a `pg_isready` healthcheck with `start_period: 15s` (covers initdb's temp-server bounce), `migrate` depends on `service_healthy`, `app` depends on `migrate` `service_completed_successfully`. Removed obsolete `version:` key. `dev`/`dev` creds flagged local-only (prod secrets via AWS Secrets Manager).

## Task 4: Migrations
- `000003_enable_postgis` (`CREATE EXTENSION postgis`), `000004_events_table` (status enum, indexes, `update_updated_at` trigger). `organizer_id`/`venue_id` are `NOT NULL DEFAULT` zero-UUID — see Task 5 note.

## Task 5: `events` module (self-contained, end-to-end)
- `internal/models/event.go` + `event_status.go`: go-pg model with enum hooks + `Validate`.
- `internal/events/repository.go` + `service.go` (+ `service_test.go`): `Repository` over the shared `*pg.DB` (via new `repository.Module.DB()` getter); `Service` with Create/GetByID/List.
- `api/swagger.yaml`: added `Event`/`EventInput` + public `GET /events`, `GET /events/{id}`, `POST /events`; `make generate-api`.
- `internal/http/formatter/event.go`, `internal/http/handlers/events.go`; registered in `internal/http/module.go` via a new `SetEventsService` setter (so existing `NewModule` tests are untouched); wired in `internal/application.go` when the DB is enabled.
- **Design note — UUIDs:** go-pg + gofrs cannot scan SQL `NULL` into a uuid field, and `RETURNING *` reading a NULL uuid errored. Resolved by treating "unset" organizer/venue as the **zero UUID** (`NOT NULL DEFAULT`, non-pointer fields, `use_zero`) and dropping `Returning("*")` in `Create` (ID + timestamps set in `BeforeInsert`). Real FK refs arrive with the organizers/venues modules.

## Task 6: Skeleton modules
- `internal/{organizers,venues,users,rsvp,search,notifications,ai}/doc.go` — package stubs with responsibility + roadmap, pointing at the `events` pattern.

## Task 7: README + Dockerfile
- Rewrote `backend/README.md` (run, commands, module roadmap, scaffolding notes).
- **Fix needed:** rename left the Dockerfile copying the binary to `/microservice-template` while entrypoint was `/lia`; corrected to `/lia`. Dropped `COPY .git ./.git` (no `.git` in the monorepo subdir → empty git ldflags, tolerated).

## Verification (performed)
- `go build ./...`, `go vet ./...`, `go test ./...` → all pass (incl. `internal/events`).
- `docker compose up -d --build` (clean one-shot): all 4 migrations apply, app healthy.
- API: `GET /events` → `[]`; `POST /events` ×2 → 201; `GET /events` → both, ordered by `starts_at`; `?status=published` → only published; `GET /events/{id}` → 200; `GET /events/{bad}` → 422; missing → 404; missing-title POST → 422.

## Out of scope / next
- Implement organizers, venues, users (domain), rsvp, search, notifications, ai modules.
- Auth (email magic link / OTP) instead of `HTTP_MOCK_AUTH`; real organizer/venue FKs; `.ics` export; S3 image uploads.
- Add `version:` field to `.golangci.yml` (installed lint version requires it).
