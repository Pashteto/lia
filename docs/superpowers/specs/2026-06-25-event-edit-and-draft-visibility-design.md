# Event edit endpoint + draft visibility lockdown — Design

**Date:** 2026-06-25
**Status:** Approved (pre-implementation)
**Area:** `backend/` events module + HTTP API

## Problem

Two access-control gaps in the events feature:

1. **No edit endpoint.** There is no `PATCH`/`PUT /events/{id}`. Owners cannot
   edit an event at all — draft or published.
2. **Drafts are not access-protected.** `GET /events/{id}` is unauthenticated
   and performs no status or ownership check (`internal/events/service.go`
   `GetByID`), so anyone with a UUID can read any draft. `GET /events` honors an
   arbitrary `?status=` filter with no auth, so `GET /events?status=draft`
   returns *everyone's* drafts to an anonymous caller
   (`internal/events/repository.go` `List`).

`GET /events/mine` (JWT, filtered by `organizer_id`) already correctly lets an
owner see their own events in any status — that behavior is preserved.

## Goals

- Owners can edit their own non-published events and publish them.
- Non-owners (incl. anonymous) cannot see or edit non-published events.
- Keep the public discovery surface (list, nearby, get-by-id for published)
  working without authentication.

## Non-goals

- **Moderation.** There is no moderator actor in the system (only an
  `AdminEmails` allowlist used by `auth.IsAdmin`, with no moderation endpoints).
  The `pending_review` ("on moderation") and `rejected` statuses are therefore
  **omitted from the owner flow**. They remain in the DB enum and Go enum as
  legacy values but are not settable via the API nor produced by the owner flow.
- **Admin bypass** of draft visibility — kept strictly owner-only for now.
- **Cancelling an already-published event** — `published` is a locked status, so
  this is not possible via `PATCH`. A status-only transition endpoint can be a
  follow-up.
- Removing `pending_review`/`rejected` from the DB/Go enum (a migration with no
  upside right now).

## Lifecycle (after this change)

Owner-facing lifecycle collapses to:

```
draft  ──(PATCH status=published)──▶  published   (terminal: locked for edits)
  │
  └────(PATCH status=cancelled)────▶  cancelled    (terminal: locked for edits)
```

- **Create default:** `draft` (was `published`). New events start hidden; the
  owner explicitly publishes.
- **Editable statuses:** only `draft`. `published` and `cancelled` are locked
  (→ 409). Legacy `pending_review`/`rejected` rows, if any exist, are also not
  editable.
- **Publish:** `PATCH status=published` on a draft sets `published_at = now`.

## Design

### Part A — Lock down read access to non-published events

**`GET /events/{id}` — optional auth + status gate.**
Endpoint stays public in the swagger spec (`security: []`) so published events
remain readable anonymously. The handler gains optional identity resolution:

1. Read the `Authorization` header from `params.HTTPRequest`. If present, resolve
   the caller via the injected `auth.CheckAuth` func. Any failure (missing,
   malformed, expired token) → treat as **anonymous** (do not 401 — published
   reads must still work).
2. Load the event.
3. If `status == published` → return it to anyone.
4. Otherwise (draft / cancelled / legacy pending_review / rejected) → return it
   **only if** the caller is the owner (`principal.UUID == event.OrganizerID`);
   else **404 Not Found** (do not leak existence).

Wiring: inject `m.auth.CheckAuth` into the `GetEventByID` handler, mirroring how
`uploads.NewHandler(m.storage, m.files, m.auth.CheckAuth)` already receives the
check-auth func in `internal/http/module.go`.

**`GET /events` (public list) — published-only.**
Today the list honors an arbitrary `?status=`. Fix: the public list always
returns `published`.

- Remove the `status` query parameter from `get /events` in `api/swagger.yaml`.
- The `ListEvents` handler / events service public path hardcodes
  `ListFilter{Status: "published"}`.
- Owners' own drafts remain available through the unchanged `GET /events/mine`.
- `GET /events/nearby` is already published-only (no change).

### Part B — `PATCH /events/{id}` (owner edit + publish)

**Spec (`api/swagger.yaml`):**

- New `patch` under `/events/{id}`, `security: [jwt: []]`.
- Body: new `EventPatch` definition — all fields optional.
- Responses: `200` (updated `Event`), `400` (validation / invalid status),
  `401` (unauthenticated), `404` (not found or not owner), `409` (locked
  status), `503` (db disabled).

**`EventPatch` fields (all optional):**
`title, description, format, price_type, price_min, price_max,
external_ticket_url, venue_id, cover_file_id, category_ids, starts_at, ends_at,
status`.

- `status` enum restricted to `{draft, published, cancelled}`.
- Partial-update convention mirrors the existing venue PATCH
  (`internal/venues/service.go` `Update`): a provided non-zero value overwrites;
  omitted/zero preserves the current value.
- `category_ids`: absent/nil → preserve; non-empty → replace the set.

**Authorization & status rules (enforced in the events service):**

1. Unauthenticated → **401** (swagger security; handler also guards nil
   principal).
2. Event not found **or** caller is not the owner → **404** (consistent with the
   read path; does not leak existence).
3. Current status not `draft` (i.e. `published`, `cancelled`, or legacy
   `pending_review`/`rejected`) → **409 Conflict** (locked).
4. If `status` is supplied in the body, it must be one of
   `{draft, published, cancelled}` (else 400). Transition to `published` sets
   `published_at = now` if currently unset.
5. Apply provided fields, re-validate categories/venue if changed, run
   `event.Validate()`, persist → **200** with the reloaded event.

## Components / code changes

- **`api/swagger.yaml`**
  - Add `EventPatch` definition.
  - Add `patch` operation under `/events/{id}` (`operationId: updateEvent`).
  - Remove the `status` query param from `get /events`.
  - Restrict `EventInput.status` enum to `{draft, published, cancelled}`.
  - Run `make generate-api` to regenerate the server stubs.

- **`internal/events/repository.go`**
  - Add `Update(event *models.Event) error` to the `Repository` interface and
    `pgRepository`. Mirror `venues` `Update`: update the event columns
    (`title, description, venue_id, cover_file_id, status, format, price_type,
    price_min, price_max, external_ticket_url, starts_at, ends_at,
    published_at, updated_at`) `WherePK()`, return `pg.ErrNoRows` on zero rows.
  - When categories changed, re-sync `event_categories` inside a transaction
    (delete + re-insert, like `Create`). Reload associations afterward (reuse
    `loadCategories/loadVenues/loadCover/loadOrganizers`).

- **`internal/events/service.go`**
  - Add `Update(ctx, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error)`.
    Define `UpdateParams` (pointer fields) in the events package so the service
    stays decoupled from `apimodels`.
  - New sentinel `ErrNotEditable` (→ 409). Reuse `ErrNotFound` for
    not-found/not-owner (→ 404) and `ErrInvalidInput` (→ 400).
  - Change the public `List` path to filter `Status: "published"`.

- **`internal/http/formatter/event.go`**
  - Add a mapper `EventPatchToUpdateParams(*apimodels.EventPatch) events.UpdateParams`.
  - Change `EventFromAPIInput` default status from `published` to `draft`.

- **`internal/http/handlers/events.go`**
  - New `UpdateEvent` handler: `Handle(params, principal)` → validate principal,
    map body, call `service.Update`, map domain errors to 400/404/409/503.
  - `GetEventByID`: accept an injected `checkAuth func(string) (*apimodels.User, error)`;
    resolve optional caller; enforce the status/ownership gate (404 for
    non-owner on non-published).

- **`internal/http/module.go`**
  - `api.EventsUpdateEventHandler = handlers.NewUpdateEvent(m.events)`.
  - Pass `m.auth.CheckAuth` to `handlers.NewGetEventByID(m.events, m.auth.CheckAuth)`.

## Testing (TDD)

- **Service** (`internal/events`, fake repo):
  - Non-owner update → `ErrNotFound`.
  - Update of a `published` event → `ErrNotEditable`.
  - Partial update applies only provided fields; omitted fields preserved.
  - `status=published` on a draft sets `PublishedAt`.
  - Invalid target status (`pending_review`/`rejected`) → `ErrInvalidInput`.
  - Public `List` returns only published.
- **GetByID visibility:**
  - Anonymous request for a draft → 404.
  - Owner request for own draft → 200.
  - Anonymous request for a published event → 200.
- **Handler** (where existing handler tests live): error-code mapping for the new
  `UpdateEvent` handler (401/404/409/400/200).

## Risks / consequences

- **Create default flip (`published` → `draft`)** is a frontend-visible change:
  clients calling `POST /events` and expecting immediate visibility now get a
  hidden draft until they `PATCH status=published`. The frontend create flow must
  send `status: published` or add a publish step. This is intended per the
  create → edit → publish flow.
- Removing the `status` query param from `get /events`: the frontend currently
  sends `status=published`; an unknown query param is ignored by the generated
  mux, so the existing frontend call keeps working.
