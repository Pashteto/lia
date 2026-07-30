# U1 feed covers — Deploy (as executed)

_2026-07-30. Target: `presence.tarski.ru` (`ssh vdska2`, 193.32.188.7). Frontend
only — no backend/gateguard/DB change, prod LIA DB stays at **020**. Plan:
`docs/superpowers/plans/2026-07-30-u1-feed-covers.md`, spec:
`docs/superpowers/specs/2026-07-30-u1-feed-covers-design.md`._

**Outcome:** deployed and healthy. `/`, `/search`, `/map`, `/events/mine`,
`/api/v1/events` all 200. Cover markup confirmed live (9 cards, `aspect-[5/2]`
bands, `44px` mobile grid, all 7 curated + 1 uploaded cover source resolving).

## What shipped (8 commits, merged to `main` `464fd22`, pushed)

`EventCover` refactored from `event: LiaEvent` to `src?/fallback?`; `EventModule`
gained a `cover?` prop rendering a 5:2 desktop band / 44×44 mobile thumbnail /
numeral-plate fallback on `bg-cell-blank`; wired through the feed, `/search`, and
the create-event wizard preview. Two `sizes` strings are deliberately px-only (no
`vw`) to keep the hidden desktop/mobile subtree from double-fetching images —
verified live (see below). One post-merge fix: numeral-plate text color pinned
against `hover-invert`'s inherited color swap (was invisible on hover).

## Build model (unchanged): build amd64 on Mac → `docker save | gzip | ssh | docker load`

Mac is Apple Silicon (`arm64`) → build used `docker build --platform linux/amd64`.
Image: `lia-frontend:u1covers-r1`, confirmed `Architecture=amd64` via `docker
inspect`. Both build-args present: `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`
and `NEXT_PUBLIC_YANDEX_MAPS_KEY` (sourced from `frontend/.env.local`).

## Procedure (as executed)

1. `git push origin main` (12 commits — 4 pre-existing docs commits + this
   plan's 8).
2. Build `lia-frontend:u1covers-r1` (Mac amd64, both build-args) → `docker save
   | gzip` (215M) → `scp` to `/opt/lia/` → `docker load` on box.
3. Rollback-tag live image: `lia-frontend-presence:rollback-u1covers-20260730-190130`.
4. Cutover: `docker tag lia-frontend:u1covers-r1 lia-frontend-presence:latest` →
   stop + rename old container `lia-frontend-presence-old-u1covers-20260730-190130`
   → `docker run -d --restart unless-stopped --name lia-frontend-presence -p
   127.0.0.1:3002:3001 lia-frontend-presence:latest`.
5. Verify (all 200): `/`, `/search`, `/map`, `/events/mine`,
   `api…/events?status=published`. `/design-preview` → **404, pre-existing** —
   confirmed by spinning up the just-superseded rollback image on a spare port
   and hitting the same route: also 404 there. Not caused by this deploy; not
   investigated further (out of scope for this plan).
6. Confirmed cover markup live: `curl` the feed HTML, count `aspect-[5/2]` (9)
   and `grid-cols-[44px_1fr_auto]` (9) occurrences, and every `_next/image` URL
   resolves to a real cover (6 curated categories + 1 uploaded file).
7. Prune: `docker builder prune -f` + `docker image prune -f`; trimmed
   `lia-frontend-presence` rollback tags to newest 3 (dropped
   `rollback-qa20r4-*`, `rollback-qa20r3-*` from 2026-07-24); removed the
   uploaded tarball from `/opt/lia/` and locally. Disk 75% → 74%.

## Rollback

```
ssh vdska2
docker rm -f lia-frontend-presence
docker rename lia-frontend-presence-old-u1covers-20260730-190130 lia-frontend-presence
docker start lia-frontend-presence
```

Or re-tag: `docker tag lia-frontend-presence:rollback-u1covers-20260730-190130
lia-frontend-presence:latest` then recreate the container as in step 4.

## Follow-ups (not part of this deploy)

- `/design-preview` 404 on prod — pre-existing (confirmed present before this
  deploy too), not investigated. Works locally (`next dev`, `next build`).
- `YANDEX_PLACES_KEY` still unprovisioned (long-standing, unrelated).
- Mobile (<640px) live network verification for the covers `sizes` guard was
  not possible in the browser-automation environment used during development
  (viewport resize not honored); desktop was verified live (7 requests, all
  `w=640`, zero hidden-thumbnail fetches) and the mechanism is structurally
  symmetric. Static HTML confirms the mobile grid and `sizes` string are
  correct. Worth a real-phone check when convenient.
