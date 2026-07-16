# Yandex Maps Migration — Design

**Date:** 2026-07-16
**Status:** Implemented
**Author:** brainstorming session

> **AMENDMENT 2026-07-16 — JS API v3 → v2.1.** This document specifies the map
> component on **JS API v3**. That is superseded: the implementation ships on
> **v2.1** (`api-maps.yandex.ru/2.1`). Reason: the provisioned JavaScript API key
> is rejected by v3 (`403 {"message":"Invalid api key"}`) even after its
> **HTTP Referer restriction was correctly configured** (key status "Активен",
> referers `presence.tarski.ru` + `localhost`) and given propagation time. The
> same key is served by v2.1 (HTTP 200), which renders equally
> border-compliant Russian maps — the actual goal. Root cause of the v3
> rejection is unconfirmed; a v3 key/product was not obtainable from either the
> old or new developer cabinet on this account.
>
> Consequences vs. the text below:
> - Component targets v2.1 (`ymaps` global, `ymaps.ready`, `ymaps.Map`,
>   `ymaps.Placemark`, `map.geoObjects`) instead of v3 (`ymaps3`).
> - v2.1 uses **[lat, lon] natively**, matching the component props, so the
>   `lib/coords.ts` `toLngLat` helper was **removed** as dead code.
> - Pin balloons take HTML strings, so labels/hrefs are **HTML-escaped**
>   (v3's DOM-element markers did not need this).
> - Everything else stands: the auth-gated backend geocode proxy is unchanged
>   and verified working; Leaflet/Nominatim fully removed.
>
> **Note:** the free JavaScript API tier is **100 requests/day** (geocoder:
> 1000/day) — tighter than the pricing pages suggest, and a likely reason to
> move to a paid tier for real traffic.

## Problem

The frontend renders maps with **Leaflet + OpenStreetMap tiles** (`tile.openstreetmap.org`)
and geocodes addresses via **OSM Nominatim** (`nominatim.openstreetmap.org`).

This is unacceptable for a Russian deployment (`presence.tarski.ru`):

1. The OpenStreetMap project/tiles have carried a "Support Ukraine" solidarity note.
2. More seriously, OSM renders international borders — notably **Crimea** — per
   international convention. Russian law requires maps shown to Russian users to
   depict Crimea as Russian territory. OSM tiles are therefore **legally
   non-compliant** for this deployment.

## Decision

Replace OpenStreetMap/Leaflet with **Yandex Maps**, which renders
legally-compliant borders and provides Russian-language geocoding.

- **Map rendering:** Yandex **JavaScript API v3**, keyed by
  `NEXT_PUBLIC_YANDEX_MAPS_KEY` (free "Бесплатный с ограничениями" license).
- **Geocoding:** Yandex **Geocoder HTTP API**, keyed by `YANDEX_GEOCODER_KEY`,
  called **server-side** through a new backend proxy so the geocoder key never
  reaches the browser.

Keys already provisioned in the Yandex Developer Cabinet (account
`Poulissimo@yandex.ru`, country = Russia, free license):

- JavaScript API key — goes in `NEXT_PUBLIC_YANDEX_MAPS_KEY` (frontend, public
  by nature; restrict by HTTP Referer in the cabinet).
- Geocoder API key — goes in `YANDEX_GEOCODER_KEY` (backend secret).

> Actual key values are NOT stored in this repo. They live in `.env.local`
> (frontend, gitignored) and deployment env (backend). This spec references them
> by variable name only.

## Guiding principle — minimize blast radius

The three map surfaces already sit behind a single component
(`frontend/components/map/LeafletMap.tsx`) exposing a stable interface:

```ts
interface MapPin { id: string; lat: number; lon: number; label?: string; href?: string }
// props: center, zoom, marker, draggableMarker, onMarkerMove, pins, className
```

Consumers:
- `frontend/components/VenueMap.tsx` — single static pin (event detail page).
- `frontend/components/MapBrowse.tsx` — `/map` browse screen, multi-pin, pins
  link to `/events/{id}`.
- `frontend/components/VenueGeoModal.tsx` — organizer venue picker: address
  search + draggable marker.

We preserve this interface and swap the implementation, so consumers change only
their dynamic-import target.

## Components

### 1. `frontend/components/map/YandexMap.tsx` (new)

Same props and `MapPin` interface as `LeafletMap`. Internals on Yandex JS API v3:

- Load the JS API script **once** (module-level singleton promise) with
  `NEXT_PUBLIC_YANDEX_MAPS_KEY` and `lang=ru_RU`. Guard against double-injection
  across the multiple map instances that can mount on one page.
- Render map, honor `center` / `zoom` and recenter on prop change.
- Single marker: static or draggable; on drag-end call `onMarkerMove(lat, lon)`.
- Multi-pin layer from `pins`; a pin with `href` navigates to that route on
  click (balloon with a link, or direct navigation), matching current behavior.
- Client-only (`"use client"`), consumed via `dynamic(..., { ssr: false })` as
  today.
- **Empty state:** if `NEXT_PUBLIC_YANDEX_MAPS_KEY` is absent, render a neutral
  placeholder ("Карта недоступна") instead of crashing — so local dev without a
  key still builds and runs.

### 2. `frontend/lib/geocode.ts` (replaces `lib/nominatim.ts`)

- Same return shape: `Promise<GeoResult[]>` with `{ lat, lon, label }`.
- Calls the backend proxy (`GET {API}/geocode?q=...`) instead of Nominatim.
- Sends credentials (the endpoint is auth-gated — see backend).
- Keep the existing 700 ms debounce in `VenueGeoModal` (still polite; also
  limits backend/Yandex load).

### 3. Backend geocode proxy (new)

New hand-mounted `http.Handler` package `backend/internal/http/geocode/`,
following the `internal/http/complaints/` shape (plain net/http, own
`writeJSON`/`writeErr` helpers).

- Route: `GET /geocode?q=<address>` — **auth-gated** (logged-in user required),
  reusing the same auth mechanism the other authenticated hand-mounted handlers
  use (`internal/http/auth`). Mounted via a path branch in the `router`
  dispatcher in `internal/http/module.go` (~L431, next to `/metrics`).
- Calls Yandex Geocoder HTTP API server-side with the backend's **first**
  outbound HTTP client (`http.Client{Timeout}` + `http.NewRequestWithContext`),
  passing `apikey=YANDEX_GEOCODER_KEY`, `format=json`, `lang=ru_RU`,
  `results=5`, `geocode=<q>`.
- Maps Yandex's GeoObjectCollection response to `[]GeoResult{lat, lon, label}`.
- Existing per-IP rate-limit middleware (`middlewares/rate_limit.go`) already
  wraps all routes; recommend enabling it in production to protect quota.

Config wiring (viper, explicit `BindEnv` — no `AutomaticEnv`):
- Add a field in `backend/config/scheme.go` (e.g. `GeocoderConfig{ Key string }`
  or onto `HTTPConfig`).
- `viper.BindEnv("geocoder.key", "YANDEX_GEOCODER_KEY")` in
  `backend/config/init.go`.
- Thread into the HTTP module and a `SetGeocoder(...)` injector in
  `backend/internal/application.go`, mirroring `SetInvitations`.

### 4. Removals

- `frontend/components/map/LeafletMap.tsx` — deleted.
- `frontend/lib/nominatim.ts` — deleted.
- `leaflet` + `@types/leaflet` — removed from `frontend/package.json` +
  lockfile.
- Update the three consumers' dynamic imports to `YandexMap`.
- Replace the attribution note in `VenueGeoModal.tsx`
  (currently "Поиск адресов — © OpenStreetMap / Nominatim…") with the
  Yandex-required attribution. **Yandex ToS requires visible attribution**
  ("© Яндекс" + terms link) — the JS API renders its own copyright control; the
  picker's helper text is updated to reference Yandex.

### 5. Config & docs

- `frontend/.env.example` — add `NEXT_PUBLIC_YANDEX_MAPS_KEY`.
- `backend/.env.prod.example` — add `YANDEX_GEOCODER_KEY`.
- Deployment: set both on the box; **frontend build-arg** must include
  `NEXT_PUBLIC_YANDEX_MAPS_KEY` (Next.js inlines `NEXT_PUBLIC_*` at build time —
  same gotcha class as `NEXT_PUBLIC_API_URL`).
- Update `docs/HANDOFF.md` map notes and any references calling the map
  "Leaflet".

## Data flow

- **Display (`VenueMap`, `MapBrowse`):** unchanged data path — coords come from
  `venue.lat/lon` and `/events/nearby`; only the render layer changes.
- **Geocode (`VenueGeoModal`):** address string → debounce →
  `GET /geocode?q=...` (auth) → backend → Yandex Geocoder → `GeoResult[]` →
  user picks a result or drags the marker → `updateVenue(id, {lat, lon})`
  (unchanged).

## Error handling

- Missing JS API key → placeholder, no crash.
- JS API script load failure → placeholder + console warning; page stays up.
- Geocode proxy: Yandex error / timeout → `503` with `{error}`; frontend catches
  and clears results (same as today's `.catch(() => setResults([]))`).
- Unauthenticated geocode call → `401`; the picker is only reachable by
  logged-in organizers, so this is a safety net, not a normal path.

## Testing

- `YandexMap`: render test with a mocked/stubbed `ymaps3` global — asserts it
  mounts, places a marker, and renders the placeholder when the key is absent.
- Geocode handler: unit test with a stubbed Yandex HTTP response (httptest) —
  asserts query mapping, response mapping, auth rejection, and error → 503.
- Manual verification (real Yandex key): drive `/map` (pins + links), an event
  detail page (single pin), and the venue picker (search → select → drag → save)
  in the browser; confirm compliant borders and no OSM/"Support Ukraine" text.

## Out of scope

- Reverse geocoding (coords → address); not currently used.
- Routing, distance matrix, org search (other Yandex products, not connected).
- Caching geocode responses (no cache layer exists; rate-limit suffices for now;
  can revisit if quota pressure appears).
- Migrating the `/events/nearby` PostGIS query (unchanged).

## Security notes

- Restrict **both** Yandex keys by HTTP Referer / allowed domains in the cabinet
  (`presence.tarski.ru`, `api.presence.tarski.ru`, `localhost`).
- JS API key is unavoidably browser-exposed; domain restriction is the control.
- Geocoder key stays server-side via the proxy; never shipped to the browser.
- Enable the existing per-IP rate limiter in production to cap geocoder usage.
