# Category Normalization — Design

_Date: 2026-06-13. Status: approved design, ready for implementation plan._

Normalize the denormalized `events.category` text column into a dedicated,
curated **categories** taxonomy with a **many-to-many** relationship to events,
end-to-end (backend + API + frontend). Venues normalization is explicitly **out
of scope** — it carries the PostGIS/geo dimension and gets its own later spec.

## Background

Today `events` carries denormalized text columns (migration `000005`):
`category`, `venue_name`, `venue_metro`. The frontend (`frontend/lib/api.ts`,
`frontend/lib/types.ts`) already models `category` as a structured
`{slug, label}` object and maps it client-side from the flat field, so the UI is
already shaped for normalization. This spec replaces the single `category` text
field with a real taxonomy and a join table; it leaves `venue_name`/`venue_metro`
untouched for the venues spec.

The current create form (`frontend/components/CreateEventForm.tsx`) is a
**free-text input** (placeholder "Медиации, Мастер-классы, Лекции…"), so existing
`events.category` rows hold free-text **Russian labels** (or empty), not slugs.
The backfill therefore matches on **label**, not slug (see Migrations).

## Decisions (locked)

- **Scope**: categories only; venues deferred to a separate spec.
- **Taxonomy**: curated, admin-seeded; **many categories per event** (join table).
- **Layers**: full-stack (backend + `GET /categories` + frontend mapping, picker, rendering).
- **Write contract**: events reference categories by **`category_ids: [uuid]`** (the create form already has the IDs from the category list, giving clean FK validation).
- **Management**: **seed-only** for now — categories are created/edited via seed migration. No admin CRUD endpoints (a later, auth-gated concern).

## Data model

**`categories`**

| column | type | notes |
|---|---|---|
| `id` | uuid | pk |
| `slug` | text | unique, not null — stable filter key |
| `label` | text | not null — Russian display label |
| `sort_order` | int | not null default 0 — UI ordering |
| `created_at` | timestamptz | not null default now |
| `updated_at` | timestamptz | not null default now, update trigger |

No `icon` column (YAGNI; add when the UI needs it).

**`event_categories`** (join)

| column | type | notes |
|---|---|---|
| `event_id` | uuid | FK → `events(id)` ON DELETE CASCADE |
| `category_id` | uuid | FK → `categories(id)` ON DELETE RESTRICT |

`PRIMARY KEY (event_id, category_id)`. Secondary index on `category_id` for
"events in category" reverse lookups.

An event may have **zero or more** categories (stays optional, matching the
current `category?`).

## Migrations

Numbered up/down pairs, following the existing `db/migrations` convention.

- **`000006_categories_table`** — create `categories` (with the `update_updated_at_column()` trigger reused from `000001`); seed the curated set (below). Down: drop table.
- **`000007_event_categories`** — create `event_categories` join + `category_id` index; **backfill** by matching each non-empty `events.category` text against `categories.label` (case-insensitive, trimmed) and inserting a link. Non-empty values that match no seeded label are left **uncategorized** (no link) — acceptable at current demo-scale data volume, and called out rather than silently assumed. Then **drop** the `events.category` column and its `event_category_idx`.
  Down: re-add `events.category` (text not null default ''), repopulate it from the first linked category's **label** per event, drop the join table.

### Seed taxonomy

Slugs that existing rows map to cleanly, plus a small curated extension for a
richer demo (sort_order in parentheses):

| slug | label | sort |
|---|---|---|
| `lecture` | Лекции | 10 |
| `workshop` | Мастер-классы | 20 |
| `mediation` | Медиации | 30 |
| `concert` | Концерты | 40 |
| `exhibition` | Выставки | 50 |
| `performance` | Спектакли | 60 |
| `film` | Кино | 70 |
| `festival` | Фестивали | 80 |

Trivially extended/trimmed via a later seed migration.

## Backend

### `internal/categories` (new module, mirrors `internal/events`)

- **Model** `internal/models/category.go`: `Category{ ID, Slug, Label, SortOrder, CreatedAt, UpdatedAt }` with go-pg tags.
- **Repository**: `List() ([]*Category, error)` (ordered by `sort_order`), `GetByIDs(ids []uuid.UUID) ([]*Category, error)` for validation.
- **Service**: `List()`; `Validate(ids []uuid.UUID) error` — returns a validation error naming any id that doesn't resolve.
- **HTTP**: `GET /api/v1/categories` → `[{id, slug, label}]` sorted by `sort_order`.

### `internal/events` changes

The `event_categories` links are part of the **event aggregate**, so the events
module owns them:

- `Event` domain model gains `Categories []*models.Category` (loaded, `pg:"-"` or via relation).
- `events.Create` accepts category IDs, **validates** them via the injected categories service, then in **one transaction** inserts the event and the join rows.
- `GetByID` / `List` load each event's categories via the join (avoid N+1 — use a single join/`Relation` or a batched second query keyed by event id).
- Wiring: categories service injected into the events service in `internal/application.go`, following the existing `SetEventsService` pattern.

## API contract

Edit `backend/api/swagger.yaml`, then regenerate with `make generate-all`
(generated server/models are gitignored — see the codegen gotcha in HANDOFF).

- **Event** response model: remove flat `category` string; add
  `categories: [{ id (uuid), slug (string), label (string) }]`.
- **EventInput**: remove flat `category`; add `category_ids: [uuid]` (write).
- New **`GET /categories`** operation → array of `{id, slug, label}`.

## Frontend (full-stack)

- `lib/types.ts`: add `id` to `EventCategory`; `LiaEvent.category` →
  `categories: EventCategory[]`; update `ApiEvent` (`categories: {id,slug,label}[]`)
  and the create-input type (`category_ids: string[]`).
- `lib/api.ts`: map the `categories` array on events; add `getCategories()`.
- `lib/mock-events.ts`: convert each mock event's `category` to a `categories`
  array so the offline/mock fallback still renders.
- **Create form**: fetch categories via TanStack Query, render a **multi-select
  picker**, submit `category_ids`.
- **Discovery**: kicker shows the **first** category label. **Detail**: render
  **all** categories as chips.

## Error handling

- Unknown `category_id` on create → **400 validation error**, consistent with the
  existing `newValidationError` pattern in `internal/models/event.go`.
- Empty `category_ids` is valid (event with no categories).
- A seeded category in use cannot be deleted (`ON DELETE RESTRICT`); deletion
  isn't exposed anyway (seed-only).

## Testing

- **Backend**: categories repo + service tests (list ordering; `Validate`
  rejects unknown id); events tests for create-with-categories (links written,
  loaded back) and unknown-id rejection; formatter test for the `categories`
  array. `go build/vet/test ./...` pass; `golangci-lint` **v1** (per the gotcha —
  do not migrate to v2) exits 0.
- **Frontend**: `pnpm lint` + `pnpm build` clean; Playwright end-to-end
  create → list → detail exercising **multiple** categories on one event.

## Out of scope

- Venues normalization and PostGIS "events nearby" (separate spec).
- Admin CRUD for categories (seed-only for now).
- Category icons.
- Tags (`event_tags` in the tech-stack doc) — distinct from categories.
