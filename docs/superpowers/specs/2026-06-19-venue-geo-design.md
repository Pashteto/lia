# Venue Geo — Design

_Date: 2026-06-19. Follow-on to the venue-normalization spec
(`2026-06-14-venue-normalization-design.md`), which deferred everything geo._

Adds coordinates to venues and a distance-based "events nearby" experience.
Venue **identity** (table, module, `GET`/`POST /venues`, typeahead) already
shipped; this spec adds the geo half that was explicitly out of scope there.

## Locked decisions

- **Coordinate source: manual.** `lat`/`lon` are stored on `venues`. The
  **backend never calls a geocoder** — no API key, no server-side egress, no
  org external-API / data-handling sign-off required.
- **Address search: OSM Nominatim, from the browser.** On the venue form the
  creator can type an address; suggestions come from OSM Nominatim called
  **client-side**, and selecting one drops a map pin and fills `lat`/`lon`. The
  backend only ever receives `lat`/`lon` — for it this is still "manual" coords.
  The pin is draggable for correction.
- **Map library: Leaflet + OpenStreetMap tiles.** No API key. Used for both the
  venue pin-picker and the display maps.
- **Nearby query: nearest-first, no user radius.** `GET /events/nearby?lat&lon&limit`
  returns events ordered by distance ascending, with a **server-side 50 km cap**
  (so "nearby" never returns an event hundreds of km away) and a `limit`. Events
  with no venue or no coordinates are excluded.
- **UI in two phases, one spec.** Phase 1: venue pin-picker + event-detail map +
  Discovery "near me" sorted list. Phase 2: a dedicated map-browse screen with
  event pins.

## Out of scope

- Any server-side geocoding or address normalization (Yandex / 2GIS / DaData).
  Reintroducing these requires the deferred external-API + org data-handling
  decision and would be its own spec.
- Pin clustering on the map screen (MVP caps pin count instead).
- Editing coordinates of existing venues via a dedicated admin flow beyond the
  `PATCH /venues` endpoint defined below.

## Data model (migration 000009)

`venues` gains:

| column | type | notes |
|---|---|---|
| `lat` | double precision | NULL = no coordinates |
| `lon` | double precision | NULL = no coordinates |
| `geog` | geography(Point,4326) | **generated**, `STORED`, derived from `lat`/`lon` |

```sql
ALTER TABLE venues
  ADD COLUMN lat double precision,
  ADD COLUMN lon double precision,
  ADD COLUMN geog geography(Point,4326)
    GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED;
CREATE INDEX venues_geog_gist ON venues USING gist (geog);
```

- `geog` is `NULL` automatically when `lat`/`lon` are `NULL` (ST_MakePoint of a
  NULL is NULL). No trigger needed; no recompute on the app side.
- **No backfill** — existing venues simply have no coordinates until edited.
- PostGIS is already enabled (migration `000003`).

Down: drop index → drop `geog` → drop `lon` → drop `lat`.

> Nullable floats avoid the go-pg + gofrs NULL-uuid gotcha (which is specific to
> uuid columns). In Go these are `*float64` (go-pg scans SQL NULL into a nil
> pointer fine).

## Backend

### Models
- `internal/models/venue.go`: `Venue` += `Lat *float64`, `Lon *float64` with
  go-pg tags. `geog` is DB-managed and **not** mapped as a writable field.
- The nearby event response carries a per-event `distance_m float64`.

### Venue service / HTTP
- `POST /venues` and a new `PATCH /venues/{id}` accept optional `lat`/`lon`.
  Validation: if either is present, **both** must be present; `lat ∈ [-90, 90]`,
  `lon ∈ [-180, 180]`; otherwise `ErrInvalidInput`.
- `GET /venues` and `GET /venues?q=` include `lat`/`lon` in responses.

### Events nearby
- `GET /api/v1/events/nearby?lat=&lon=&limit=` (`NearbyEvents` handler).
  - Service validates `lat`/`lon` presence + ranges (`ErrInvalidInput`), clamps
    `limit` to a sane default/max (e.g. default 20, max 100).
  - Repository `Nearby(lat, lon, limit)`:
    ```sql
    SELECT e.*, ST_Distance(v.geog, ref) AS distance_m
    FROM events e JOIN venues v ON v.id = e.venue_id
    WHERE v.geog IS NOT NULL AND ST_DWithin(v.geog, ref, 50000)
    ORDER BY v.geog <-> ref
    LIMIT $limit;            -- ref = ST_SetSRID(ST_MakePoint($lon,$lat),4326)::geography
    ```
    Nested `venue` and `categories` are loaded the same way the existing list
    endpoint does.
  - Composes with the existing category filter as an additional optional WHERE
    clause (same `category_id` semantics as the list endpoint).

## Frontend

- **Leaflet** added via npm, imported **dynamically / client-only** (no SSR;
  Leaflet touches `window`). Tiles from `tile.openstreetmap.org`; OSM attribution
  rendered on every map.
- **Venue form (extend the "create new" path in `VenuePicker`):** a mini-map +
  an address search box. Typing queries OSM Nominatim **from the browser**
  (debounced, ≤ 1 req/s, descriptive request headers, attribution shown). Picking
  a result drops a pin and fills `lat`/`lon`; the marker is draggable for manual
  adjustment. Coordinates are optional — a venue can still be created without them.
- **Event-detail:** an interactive Leaflet mini-map showing the venue pin when
  the venue has coordinates; otherwise the existing text-only venue block.
- **Discovery (Phase 1):** a **"рядом со мной"** control → `navigator.geolocation`
  → `GET /events/nearby` → the existing event-card list, sorted by distance, each
  card showing a distance badge ("≈ 1.2 км"). Graceful fallback when the user
  denies geolocation or it is unavailable (keep the normal list, show a hint).
- **Map screen (Phase 2):** a dedicated route rendering a map with event pins;
  center = geolocation or a draggable point; search-this-area over the visible
  viewport. **No clustering** in MVP; cap the number of pins and `log` / surface
  a note when results are truncated (no silent cap).

### Frontend types
- `lib/types.ts`: `Venue` += `lat?: number`, `lon?: number`; the nearby response
  event type += `distance_m?: number`. `CreateEventInput` / venue-create input
  += optional `lat?`, `lon?`.

## Data-handling note (org flag)

Browser-side external egress is introduced: typed addresses → OSM Nominatim, and
map tiles → the OSM tile server. The data is **public venue addresses (not PII)**;
risk is low. There is **no server-side egress, no API keys, and no stored
secrets**. This egress is an accepted, documented part of the design. Nominatim's
usage policy (rate limit, attribution, identifying request) is honored client-side.

## Testing

- **Backend:** `Nearby` ordering / 50 km cap / exclusion of venues without
  coordinates; `lat`/`lon` range validation on create + patch; `go build/vet/test
  ./...`; CI-equivalent `golangci-lint` (v1) exits 0.
- **Frontend:** `pnpm lint` + `pnpm build` clean; Playwright — create a venue via
  the pin-picker (address search → pin → save), event-detail map renders, and
  "рядом со мной" with mocked `navigator.geolocation` returns a distance-sorted
  list.

## Phasing for the implementation plan

1. **Phase 1** — migration 000009; venue model + service + HTTP (`lat`/`lon`,
   `PATCH`); `GET /events/nearby`; Leaflet dep; venue pin-picker; event-detail
   map; Discovery "near me". Ships a complete, useful slice.
2. **Phase 2** — dedicated map-browse screen with event pins + viewport search.
