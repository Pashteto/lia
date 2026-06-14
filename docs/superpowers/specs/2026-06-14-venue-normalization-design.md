# Venue Normalization — Design

_Date: 2026-06-14. Status: approved design, ready for implementation plan._

Normalize the denormalized `events.venue_name` / `events.venue_metro` text columns
into a dedicated **venues** entity that events reference by `venue_id`, end-to-end
(backend + go-swagger API + Next.js frontend). This covers venue **identity only**.

## Out of scope (deferred to a separate "venue geo" spec)

- Coordinates (lat/lon) and the PostGIS point/GIST index.
- "Events nearby" geo query (`GET /events/nearby`) and any map UI.
- Geocoder / address-normalization integrations (Yandex Maps / 2GIS / DaData).

PostGIS stays enabled (migration `000003`) but is unused by this spec.

## Background

`events` currently carries: `venue_id uuid NOT NULL DEFAULT <zero>` (a **loose
reference** — zero UUID means "no venue", mirroring `organizer_id`), plus
denormalized `venue_name` and `venue_metro` text columns (migration `000005`).
The create form free-texts venue name + metro. The frontend already models
`venue` as a structured `{id, name, metro?, address?}` object (`lib/types.ts`)
and maps it from the flat fields, so the UI is already shaped for normalization.

This is the direct parallel of the merged **category normalization** (spec
`2026-06-13`), minus the many-to-many (venue is one-per-event) and minus geo.

Tech-stack §13 `Venue` = `{id, name, address, lat, lon, metro, district}`; this
spec implements all of those **except lat/lon** (the geo spec adds those).

## Decisions (locked)

- **Scope**: venue identity only; geo deferred.
- **Cardinality**: one venue per event (the existing `venue_id` FK-style reference).
- **Attachment model**: **pick-or-create via a searchable typeahead**. `GET /venues?q=`
  powers search; "create new" calls `POST /venues` (find-or-create by normalized
  name) → returns id; the event is created with `venue_id`. (Approach A: event
  create takes only `venue_id`; venue creation is a separate, reusable endpoint.)
- **Reference style**: `events.venue_id` stays `uuid NOT NULL DEFAULT <zero>` as a
  **loose reference with no DB foreign-key constraint** — consistent with
  `organizer_id`, and required by the go-pg + gofrs NULL-uuid gotcha (a strict FK
  can't coexist with the zero-UUID "unset" sentinel; making it nullable would break
  scanning the event row). Validity of a non-zero `venue_id` is enforced in the
  service layer instead.
- **Layers**: full-stack (backend + `GET`/`POST /venues` + frontend typeahead + display).

## Data model

**`venues`**

| column | type | notes |
|---|---|---|
| `id` | uuid | pk |
| `name` | text | not null |
| `address` | text | not null default '' |
| `metro` | text | not null default '' |
| `district` | text | not null default '' |
| `created_at` | timestamptz | not null default now |
| `updated_at` | timestamptz | not null default now, update trigger |

Btree index on `lower(name)` for case-insensitive search + find-or-create lookup.
No unique constraint (two real venues may share a name; the picker + find-or-create
by normalized name keep accidental dupes down without forbidding legitimate ones).

`events.venue_id` is unchanged in type (`uuid NOT NULL DEFAULT <zero>`); the
denormalized `venue_name` / `venue_metro` columns are **dropped**.

## Migration (000008)

Up:
1. Create `venues` (with the `update_updated_at_column()` trigger from `000001`)
   and the `lower(name)` index.
2. **Backfill**: for each distinct normalized (`lower(btrim(venue_name))`) non-empty
   `venue_name` on `events`, insert a venue (`name` = a representative original
   casing, `metro` = the corresponding `venue_metro`); then `UPDATE events` to set
   `venue_id` to the matching venue (join on `lower(btrim(venue_name)) = lower(name)`).
3. Drop `events.venue_name` and `events.venue_metro`.

Down:
1. Re-add `venue_name text NOT NULL DEFAULT ''`, `venue_metro text NOT NULL DEFAULT ''`.
2. Repopulate them from the joined venue (`UPDATE events ... FROM venues WHERE
   venues.id = events.venue_id`), leaving zero-UUID rows blank.
3. Drop `venues`.

(No FK constraint is added on `events.venue_id`, so no constraint ordering concerns.)

## Backend

### `internal/venues` (new module, mirrors `internal/categories`)

- **Model** `internal/models/venue.go`: `Venue{ ID, Name, Address, Metro, District, CreatedAt, UpdatedAt }` with go-pg tags + a `BeforeInsert` (UUID + timestamps).
- **Repository**:
  - `Search(q string, limit int) ([]*Venue, error)` — `ILIKE '%q%'` on name (empty `q` → first N by name), ordered by name.
  - `GetByID(id uuid.UUID) (*Venue, error)`.
  - `GetByIDs(ids []uuid.UUID) ([]*Venue, error)` — for batch event loading.
  - `FindOrCreateByName(v *Venue) (*Venue, error)` — returns an existing venue whose `lower(name)` matches, else inserts.
- **Service**: `Search`, `GetByID`, `Create` (validates `name` non-empty → `ErrInvalidInput`, normalizes, delegates to `FindOrCreateByName`); `Validate(id uuid.UUID) (*Venue, error)` — for a non-zero id, returns `ErrInvalidInput` if it doesn't exist (zero id → nil, no error).
- **HTTP**: `GET /api/v1/venues?q=&limit=` (`ListVenues`), `POST /api/v1/venues` (`CreateVenue`).

### `internal/events` changes

- `Event` domain model: replace `VenueName`/`VenueMetro` (`pg:"venue_name"/"venue_metro"`) with a non-persisted `Venue *models.Venue` (`pg:"-"`) loaded read-side. `VenueID uuid.UUID` stays.
- `events.Create`: validate `event.VenueID` via the injected venue validator (non-zero must exist → else `ErrInvalidInput`), then insert (no new join table — venue is a single column).
- `GetByID`/`List`: load each event's venue via `venues` `GetByIDs` for the non-zero `venue_id`s (single batched query, no N+1), attach as `event.Venue`.
- Wiring: a venue validator/loader injected into the events service in `internal/application.go`, mirroring the categories wiring (local interface in the events package for testability).

## API contract (`api/swagger.yaml` → regenerate with `make generate-api`)

- **`Venue`** definition: `{ id (uuid, required), name (string, required), address, metro, district }`.
- **Event** response: remove `venue_name` and `venue_metro`; add `venue: $ref Venue` (populated only when `venue_id` is set). `venue_id` stays.
- **EventInput**: remove `venue_name`/`venue_metro`; keep `venue_id` (already present).
- **`VenueInput`**: `{ name (required), address, metro, district }`.
- New paths: `GET /venues` (params `q`, `limit`; → `[Venue]`); `POST /venues` (body `VenueInput`; → `Venue`, 400 on empty name).

Generated server/models under `internal/http/server` + `internal/http/models` are
gitignored and regenerated; never hand-edit them (codegen gotcha).

## Frontend (full-stack)

- `lib/types.ts`: `Venue` += `district?`; `ApiEvent` replaces `venue_name`/`venue_metro`/`venue_id` flat fields with a nested `venue?: { id, name, address?, metro?, district? }` (keep `venue_id?` for the write path if needed); `CreateEventInput` uses `venue_id?: string` (drop `venue_name`/`venue_metro`).
- `lib/api.ts`: map `event.venue`; add `searchVenues(q)` (GET /venues) and `createVenue(input)` (POST /venues).
- **Create form** (`components/CreateEventForm.tsx`): replace the free-text "Место" + "Метро" inputs with a **venue typeahead** — debounced search via `searchVenues`, pick a result (sets `venueId`), or "create «…»" which calls `createVenue` then sets `venueId`. Submit `venue_id`.
- Detail (`app/events/[id]/page.tsx`) and card (`components/ui/EventCard.tsx`) already render `venue.name`/`venue.metro` — keep; detail may additionally show `venue.address`.
- `lib/mock-events.ts`: venues are already `{id, name, metro}` objects — align to the new shape (add `district?` where natural); mock fallback keeps rendering venues offline (the deployed demo has no backend, so the typeahead shows an offline/empty state — graceful, like the category picker).

## Error handling

- `POST /venues` with empty/blank name → **400** (service `ErrInvalidInput`).
- Event create with a non-zero `venue_id` that doesn't exist → **400**.
- Absent / zero `venue_id` → event has no venue (`venue` omitted in the response).
- `GET /venues` with no `q` → returns the first `limit` venues by name (default limit, e.g. 20).

## Testing

- **Backend**: venues repo + service tests (search filters by name; `FindOrCreateByName` reuses vs inserts; `Create` rejects empty name; `Validate` rejects unknown non-zero id, accepts zero id). Events tests: create with a valid `venue_id` attaches it; unknown `venue_id` → `ErrInvalidInput`; `GetByID`/`List` load the venue. Formatter test for the `venue` mapping. `go build/vet/test ./...`; `golangci-lint` **v1** exits 0; live API (`docker compose up` + curl: create venue, search, create event with venue_id, read back venue, unknown venue_id → 400).
- **Frontend**: `pnpm lint` + `pnpm build` + `tsc --noEmit` clean; Playwright/SSR e2e: search-and-pick an existing venue on create, and create-a-new venue inline, then confirm the detail page shows the venue.

## Notes for the follow-on geo spec (not built here)

When geo lands: add `lat`/`lon` (+ a PostGIS `geography(Point)` column and GIST
index) to `venues`, a coordinate source (manual vs geocoder — needs an
external-API + org data-handling decision), `GET /events/nearby`, and the map UI.
