# Yandex Maps (JS API v2.1) — Deploy

_2026-07-16. Deploy runbook for the OSM/Leaflet → Yandex Maps migration
(`feat/yandex-maps`). Target: `presence.tarski.ru` (API `api.presence.tarski.ru`)
on vds-ru215 (193.32.188.7). Build-on-Mac → `docker save | ssh | docker load` →
cutover, same pattern as every other feature on this box._

## What shipped

- Frontend maps (`/map`, event-detail venue map, venue pin-picker) now render via
  **Yandex Maps JS API v2.1** (not v3 — v3 rejects the provisioned key), loaded
  client-side with `NEXT_PUBLIC_YANDEX_MAPS_KEY`. Without a key (or if the build
  didn't inline one), the UI falls back to a "Карта недоступна" placeholder
  instead of failing hard.
- Backend gained a geocode proxy: `GET /api/v1/geocode` (auth-gated), config-bound
  to `YANDEX_GEOCODER_KEY`, used by the venue address-search typeahead so the
  Yandex Geocoder key never reaches the browser.
- `components/map/LeafletMap.tsx` and the OSM/Leaflet dependency are **deleted**.

## Non-obvious steps (read before deploying)

1. **Frontend build-arg is mandatory, and there are now two of them.** The
   Dockerfile only inlines env vars it explicitly declares as `ARG`/`ENV`
   *before* `RUN pnpm build` — anything passed via `--build-arg` that isn't
   declared is silently dropped, and the map key inlines as `""`. Build with
   **both**:
   ```
   docker build --platform linux/amd64 \
     --build-arg NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru \
     --build-arg NEXT_PUBLIC_YANDEX_MAPS_KEY=<JS API key from the Yandex cabinet> \
     -t lia-frontend:<tag> .
   ```
   The `NEXT_PUBLIC_API_URL` value **must** be `https://api.presence.tarski.ru` —
   older runbooks in this directory say `lia.pashteto.com`/`api.lia.pashteto.com`;
   that host is stale and using it degrades the site to mock data.
2. **Backend needs the geocoder secret in its env, then a rebuild/restart.**
   Set `YANDEX_GEOCODER_KEY=<geocoder key from the Yandex cabinet>` in
   `/opt/lia/backend/.env.prod` (or wherever the box's prod env file lives),
   then rebuild/ship the `backend-app` image as usual and recreate the
   container so the new `/api/v1/geocode` route + config binding pick it up.
   Without this, the venue address-search typeahead silently returns nothing.
3. **Post-deploy verification** (in addition to the usual route/health checks):
   - Load `/map` and confirm an actual Yandex map tile renders with the
     **"© Яндекс"** attribution badge — NOT the "Карта недоступна" placeholder.
     Placeholder rendering means the build-arg from step 1 was dropped.
   - As a logged-in organizer, open the venue create/edit flow and type into
     the address field; confirm autocomplete suggestions appear. This exercises
     the backend geocode proxy end-to-end (step 2).

## Procedure (as executed — adapt tags/paths per usual)

1. **Build frontend** on Mac with both build-args (see step 1 above); verify
   `next build` compiles clean (exit 0).
2. **Ship + cutover frontend**: `docker save | gzip | ssh vds-ru215 'gunzip | docker load'`,
   tag current `lia-frontend-presence` as a `rollback-*` tag, tag the new image
   `latest`, recreate the container on `:3002`.
3. **Set `YANDEX_GEOCODER_KEY`** in the backend's prod env file on the box.
4. **Build + ship backend** as usual (build-on-Mac → save/ssh/load), tag a
   `rollback-*` before cutover, recreate with the full compose-file set
   (gateguard file mandatory, per standing deploy convention).
5. **Verify** per the three checks above.
6. **Cleanup**: `docker builder prune -f` + `docker image prune -f`, and trim
   old `rollback-*` tags down to the last ~3 — the box's disk is 20 GB and has
   hit 90% before.

## Rollback

- **Frontend**: retag the preserved `rollback-*` image back to `latest`,
  recreate `lia-frontend-presence`.
- **Backend**: retag the preserved `rollback-*` image back to `latest`,
  recreate with the 3 compose files. Removing `YANDEX_GEOCODER_KEY` from the
  env is not required for rollback (the route just won't be exercised by an
  older frontend that doesn't call it), but restoring the prior env file is
  the cleanest revert.
- **DB**: no migration is expected for this feature; confirm against the
  actual PR before assuming so.

## Standing caveats

- The Yandex free JS API tier is capped at **100 requests/day** — watch for
  the placeholder reappearing under load; that's the quota, not a build
  regression, once step 1's build-arg has been confirmed present in the image.
- Both Yandex keys are **Referer-restricted** to `presence.tarski.ru` and
  `localhost`. A key that works locally but shows the placeholder in prod (or
  vice versa) is almost always a Referer-restriction mismatch, not a missing
  build-arg — check both before assuming the worse-case cause.
- Do not commit or paste actual key values into docs/runbooks/commits; refer
  to them by env var name and "from the Yandex cabinet".
