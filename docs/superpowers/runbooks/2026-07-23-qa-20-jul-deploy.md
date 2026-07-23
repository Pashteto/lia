# QA-20-jul fixes — Deploy (as executed)

_2026-07-23. Target: `presence.tarski.ru` / `api.presence.tarski.ru` on vds-ru215
(`ssh vdska2`, 193.32.188.7). GateGuard DB migration **12 → 13**. Plan:
`docs/superpowers/plans/2026-07-23-qa-20-jul-fix-everything.md`._

**Outcome:** deployed and healthy. All routes + API 200; `/api/v1/places` mounted
(401 unauth). **One follow-up:** `YANDEX_PLACES_KEY` not yet provisioned →
venue-name search inert (degrades to address-only) until the key is added.

## What shipped (11 commits, merged to `main` `bafd694`)
Banner gates on `roleResolved` (not hydration); gateguard 000013 backfills admin
`email_verified`; SignupCTA shows status for non-published events; owner can
withdraw without verification; Google Calendar deep-link; calendar query caching;
**Yandex Places proxy `/api/v1/places`** + frontend name search; **invite-accept
verifies email** (new `MarkEmailVerified` gRPC); signup field reorder. (Task 7 —
backend calendar fan-out collapse — intentionally deferred.)

## Build model (unchanged): build amd64 on Mac → `docker save | gzip | ssh | docker load`
Mac is Apple Silicon → every build used `docker build --platform linux/amd64`.
Images: `gateguard:qa20-r1`, `lia-backend:qa20-r1`, `lia-frontend:qa20-r1` (all
verified `Architecture=amd64`). gateguard's generated `*_grpc.pb.go` is
`.gitignored` but was **already regenerated locally** (Task 10 ran protoc), so the
Mac build baked in `MarkEmailVerified` — **no on-box gateguard proto regen needed**
(this side-steps runbook-2026-07-16 trap 2, where building gateguard on the box
fails on the github TLS curl).

## Traps / notes
1. **DB creds are `DATABASE_USER`/`DATABASE_PASSWORD`** in `.env.prod` (not
   `POSTGRES_*`); Postgres superuser is `lia_prod`; both `lia_prod` and `gateguard`
   DBs live in `backend-postgres-1`. Migrate host is service alias `postgres` on
   network `backend_default`, image `migrate/migrate:v4.17.1`, migrations flat in
   `/opt/gateguard/db`.
2. **SQL string literals need single quotes** (`role='admin'`) — double quotes are
   read as column identifiers (same trap as 2026-07-16).
3. **New env var needs BOTH `.env.prod` AND compose declaration.** Added
   `YANDEX_PLACES_KEY: ${YANDEX_PLACES_KEY}` under `app.environment` in
   `docker-compose.prod.yml` (after the geocoder line). With the value absent,
   compose passes it empty → backend `SearchPlaces` errors per-request → handler
   503 → frontend `Promise.allSettled` degrades to address-only. No crash.
4. **Frontend maps footgun**: built with BOTH `--build-arg
   NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` and
   `NEXT_PUBLIC_YANDEX_MAPS_KEY` (from `frontend/.env.local`). The `YandexMap` loader
   lives in a lazily-imported (`ssr:false`) chunk, so it can't be grepped from the
   `/map` HTML — confirm maps render visually.

## Procedure (as executed)
1. Merge `qa-20-jul-fixes` → `main` (ff), tests green, branch deleted.
2. Backups: `pg_dump gateguard` + `pg_dump lia_prod` → `/opt/lia/backup-pre-qa20-*.gz`.
3. Build `gateguard:qa20-r1` (Mac amd64) → save/load → box.
4. `scp gateguard/db/000013_*.sql` → `/opt/gateguard/db/`; run
   `/opt/lia/migrate-qa20-013.sh` (sources creds from `.env.prod`). **12 → 13**,
   `dirty=f`, 1 admin verified.
5. Build `lia-backend:qa20-r1` (Mac amd64) → save/load → box. (No `make generate-api`
   — swagger spec unchanged this deploy; generated files present in context.)
6. Build `lia-frontend:qa20-r1` (Mac amd64, both build-args) → save/load → box.
7. Rollback-tag live images (`*:rollback-qa20-20260723-202249`); wire compose
   (`YANDEX_PLACES_KEY`), backup `docker-compose.prod.yml.bak-pre-qa20-*`.
8. Cutover: `docker tag gateguard:qa20-r1 gateguard:local` + `docker tag
   lia-backend:qa20-r1 backend-app:latest` → `docker compose --env-file .env.prod
   -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml
   -f docker-compose.monitoring.yml up -d --no-build --force-recreate gateguard app`.
9. Frontend swap: rollback-tag `lia-frontend-presence:latest` →
   `docker tag lia-frontend:qa20-r1 lia-frontend-presence:latest` → stop + rename old
   container `lia-frontend-presence-old-qa20-20260723-202601` → `docker run -d
   --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001
   lia-frontend-presence:latest`.
10. Verify (all 200): `/`, `/map`, `/auth/verify`, `/me/invitations`,
    `api…/events?limit=1`; `/api/v1/places` + `/api/v1/geocode` → 401 (mounted,
    auth-gated); gateguard log `GRPC on 0.0.0.0:9090`, no errors; backend env has
    `YANDEX_GEOCODER_KEY` + `YANDEX_PLACES_KEY`.
11. Prune: `docker builder prune -f` + `docker image prune -f`; trim `rollback-*`
    to newest 3/repo. Disk 70% → 66%.

## Rollback
- **Backend**: `docker tag backend-app:rollback-qa20-20260723-202249 backend-app:latest`
  → recreate `--no-build app` (4 compose files).
- **GateGuard**: `docker tag gateguard:rollback-qa20-20260723-202249 gateguard:local`
  → recreate `--force-recreate gateguard`.
- **Frontend**: `docker rm -f lia-frontend-presence && docker rename
  lia-frontend-presence-old-qa20-20260723-202601 lia-frontend-presence && docker
  start lia-frontend-presence`.
- **DB**: gateguard `migrate … down 1` (13 → 12; down is a no-op `SELECT 1;`), or
  restore `/opt/lia/backup-pre-qa20-gateguard-20260723-201312.sql.gz`.
- **Env/compose**: `.env.prod.*` + `docker-compose.prod.yml.bak-pre-qa20-*` on box.

## Remaining / follow-ups
- **Provision `YANDEX_PLACES_KEY`** in `/opt/lia/backend/.env.prod`, then
  `cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml
  -f docker-compose.prod.yml -f docker-compose.gateguard.yml -f
  docker-compose.monitoring.yml up -d --no-build app` to activate venue-name search.
- **Unproven live** (need auth + real data / a real inbox): map renders on `/map`;
  invite-accept end-to-end verifies an unverified invitee; the admin banner is gone
  after re-login (the `roleResolved` fix + backfill).
