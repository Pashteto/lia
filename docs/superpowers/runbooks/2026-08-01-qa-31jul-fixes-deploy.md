# QA 31-jul fixes — Deploy (as executed)

_2026-08-01. Target: `presence.tarski.ru` / `api.presence.tarski.ru` on vds-ru215
(`ssh vdska2`, 193.32.188.7). Branch `fix/qa-31jul` (10 commits, NOT merged to
`main` — see "Concurrent session" below). Source report:
`docs/superpowers/reports/2026-07-31-qa-full-site-sweep.md`._

**Outcome:** deployed and healthy. Backend + frontend both recreated; all routes
and API endpoints 200; every fixed defect verified live (see the QA report for
per-defect evidence). Two rounds: `qa31-r1` (backend + frontend), then
`qa31-r2` (frontend only, for the modal defect found during the QA walk).

## What shipped (10 commits on `fix/qa-31jul`)

All 13 defects from the sweep, plus one found during the post-deploy walk.
P1: attendeeCount derivation, feed `from` default, honest moderation copy,
visible Places degradation. P2: short-id collisions, RU copy + `published_at`,
feedback gating, rsvp list enrichment, admin counts, complaints/settings
redesign. P3: coverless hero plate, square venue-map marker. Plus: Swiss Grid
modal layer (SquareRadio, ConfirmModal).

## Build model (unchanged): build amd64 on Mac → `docker save | gzip | ssh | docker load`

**Trap — build from an isolated worktree, not the shared checkout.** Another
session had untracked work in the tree (migration 000022). Built from
`git worktree add --detach <scratchpad> fix/qa-31jul` so nothing foreign was
baked into the image.

**Trap — a clean worktree does not build the backend.** Two generated trees are
gitignored and must be copied in from the main checkout before `docker build`:

```
backend/internal/http/server/     # go-swagger generated server
backend/internal/http/models/     # go-swagger generated models
backend/protocols/userservice/*.pb.go
```

Safe to copy verbatim only because `api/swagger.yaml` was unchanged on this
branch (verified with `diff -q`). If the spec changes, regenerate instead.

## Procedure (as executed)

1. `pg_dump` both DBs → `/opt/lia/backup-pre-qa31-{lia,gateguard}-20260801-083305.sql.gz`.
2. Build `lia-frontend:qa31-r1` (both build-args) and `lia-backend:qa31-r1`,
   both `--platform linux/amd64`, confirmed via `docker inspect`.
3. `docker save | gzip` → `scp` → `docker load` on box.
4. Rollback-tag live images `*:rollback-qa31-20260801-083834`.
5. Backend cutover: `docker tag lia-backend:qa31-r1 backend-app:latest` →
   `docker compose --env-file .env.prod -f docker-compose.yml -f
   docker-compose.prod.yml -f docker-compose.gateguard.yml -f
   docker-compose.monitoring.yml up -d --no-build --force-recreate app`.
   Confirmed in the log: `rsvp lists wired to events enrichment`.
6. Frontend swap: tag `lia-frontend-presence:latest`, stop + rename old to
   `lia-frontend-presence-old-qa31-20260801-083834`, `docker run -d --restart
   unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001`.
7. Verify: `/`, `/search`, `/map`, `/events/mine`, `/login`, `/me/calendar`,
   all admin routes → 200; unknown route → 404; `admin/overview` anon → 401.
8. **Data:** shifted the 7 curated seeded events +42 days (see below).
9. Second round `qa31-r2`: frontend only, same swap, for the modal fix.
10. Prune: `docker builder prune -f`, `docker image prune -f`, dropped 3 stale
    rollback tags and the superseded `-old-u1covers-*` container, removed
    uploaded tarballs. Disk 74–75%.

## The migrate container runs on `docker compose up app`

`backend-migrate-1` is a dependency and fires on every `up`. It mounts the
**host's** `/opt/lia/backend/db/migrations`, not the image's, so it logged
`no change` and the schema stayed at 20. Do not assume an image-baked migration
will (or won't) apply — check `schema_migrations` after every `up`.

## Data change: seeded events shifted +42 days

The `from`-default fix (#4.2) is correct, and it exposed that 8 of 9 published
events were in the past — the feed legitimately dropped to 1 event. With the
user's approval:

```sql
UPDATE events SET starts_at = starts_at + interval '42 days',
                  ends_at   = ends_at   + interval '42 days',
                  updated_at = now()
 WHERE id::text LIKE 'b0000000-%' AND status = 'published';   -- 7 rows
```

42 days preserves each event's time of day, duration and relative spacing.
`twix-mars` was deliberately **left in the past** so the junk event drops off
the feed on its own (partial §5 cleanup for free). Feed went 1 → 8 events.

## Concurrent session — read before merging

Another session was working in the same checkout and on the same prod box
during this deploy:

- It left untracked `backend/db/migrations/000022_venue_search_extensions.*`.
- **Mid-QA it applied migrations 021 and 022 to production** — prod
  `schema_migrations` went 20 → 22 between two of my screenshots, which added
  the `reading-group` category and shifted every category numeral.
- Nothing broke: the deployed backend does not call `unaccent()`/`similarity()`,
  and all routes stayed 200 (re-verified after).

`fix/qa-31jul` was therefore **never merged to `main` and `main` was never
checked out** — switching HEAD in a shared worktree is exactly the collision
`concurrent-sessions-shared-worktree` warns about. Merge when the other session
is idle.

Prod schema is now at **22**; `main` still only carries up to 021.

## Rollback

```
ssh vdska2
# frontend
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-qa31-20260801-083834 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
# backend
cd /opt/lia/backend
docker tag backend-app:rollback-qa31-20260801-083834 backend-app:latest
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build --force-recreate app
# data (only if the +42d shift must be undone)
UPDATE events SET starts_at = starts_at - interval '42 days',
                  ends_at   = ends_at   - interval '42 days'
 WHERE id::text LIKE 'b0000000-%' AND status = 'published';
```
