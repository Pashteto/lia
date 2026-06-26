# Runbook — page back-navigation + persistent TabBar (frontend redeploy, 2026-06-26)

Adds a way to leave secondary pages (the reported bug: `/events/mine` had no
back affordance) and makes the mobile bottom nav persistent. **Frontend-only** —
backend, GateGuard, DB, and the `lia_uploads` volume are untouched. No migration.

Live target: `https://lia.pashteto.com` on **vds-ru215** (`193.32.188.7`,
`ssh vdska2`, root). Hand-managed Docker (documented demo exception).

Branch: `feat/event-edit-and-draft-visibility` (shared with concurrent RSVP /
moderation / liquid-glass work — deploy is image-from-current-tree, not a clean
feature branch). Relevant commits: `aee829b` (mine back link), `1be101a`
(map/admin/me/* back links), `0f3cd05` (TabBar → layout). The publish-draft
button (`7857d7b`) and the event-edit/draft-visibility **backend** are already
live as of the full-stack deploy (`2026-06-26-rsvp-moderation-fullstack-deploy.md`),
so this redeploy needs **no** backend change.

## What changed (code, frontend only)

- **Back links** `‹ События` → `/` added to the pages that had no nav out of
  them: `app/events/mine/page.tsx`, `app/map/page.tsx`, `app/admin/page.tsx`,
  `app/me/practices/page.tsx`, `app/me/applications/page.tsx` (both the signed-in
  and signed-out states on the `me/*` and `mine` pages). Matches the existing
  pattern on the event detail page. `/search` already had "К событиям".
- **Persistent TabBar**: `components/ui/TabBar.tsx` lifted out of `app/page.tsx`
  into `app/layout.tsx` so the mobile bottom nav shows on every screen. TabBar
  now self-hides on routes with their own bottom chrome / nav (the create form,
  an event detail page's sticky CTA, and the whole `/admin` section) and stays
  mobile-only (`sm:hidden`). `max-sm:pb-28` clearance added to the list pages it
  overlays (`mine`, `me/practices`, `me/applications`, `map`).
- Desktop is unchanged by the TabBar move (it is `sm:hidden`); desktop secondary
  pages rely on the new `‹ События` back links. Home keeps `GlassNav`.

## Pre-build verification (done on Mac, 2026-06-26)

- `npx tsc --noEmit` → clean. `eslint` on changed files → clean.
- `npm run build` (the production build the image runs) → exit 0; all 12 routes
  compile and prerender.

## Deploy procedure (frontend-only, build-on-Mac → ship)

Same pattern as `2026-06-26-liquid-glass-frontend-redeploy.md`. The box can't
reliably pull large layers, so build amd64 on the Mac and ship via
`docker save | ssh | load`. `ssh vdska2` is **password-auth** (interactive) —
run these yourself / from a session where you can enter the password.

```bash
# 0. preserve the current live image for rollback
ssh vdska2 'docker tag $(docker images lia-frontend:latest -q) lia-frontend:rollback'

# 1. build on the Mac (from repo root)
cd frontend
docker build --platform linux/amd64 \
  --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64 .

# 2. ship
docker save lia-frontend:amd64 | gzip | ssh vdska2 'gunzip | docker load'

# 3. cutover (retag + recreate the container)
ssh vdska2 'docker tag lia-frontend:amd64 lia-frontend:latest && \
  docker rm -f lia-frontend; \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

## Verify

```bash
curl -sI https://lia.pashteto.com | head -1                                              # → 200
curl -s -o /dev/null -w '%{http_code}\n' https://api.lia.pashteto.com/api/v1/events?status=published  # → 200 (untouched)
```

Visual:
- Open `https://lia.pashteto.com/events/mine` (desktop) → a `‹ События` link sits
  above "Мои события" and returns to the feed. Same on `/map`, `/admin`,
  `/me/practices`, `/me/applications`.
- Mobile width (≤640px / `sm`): the bottom TabBar is present on the feed, search,
  map, mine, me/* — and **absent** on an event detail page, the create form, and
  `/admin` (those have their own chrome).

## Rollback

```bash
ssh vdska2 'docker tag lia-frontend:rollback lia-frontend:latest && \
  docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

## Status — DEPLOYED + verified live (2026-06-26)

- Frontend image rebuilt (amd64) and shipped (`docker save | ssh vdska2 | load`),
  container recreated. Rollback tag `lia-frontend:rollback` captured first.
- Verified: `https://lia.pashteto.com` → 200; `api…/events?status=published` → 200
  (untouched); `‹ События` back link present in server-rendered `/map` HTML
  (confirms the new build is live).
- No DB migration, no backend image, no GateGuard change involved.
- **Backend NOT redeployed** this round: the working tree carries an *untracked*
  migration `000015_organizers.up.sql` (a concurrent organizers feature, not
  committed); shipping the backend would apply an unreviewed migration + in-flight
  code to prod. Backend left at its today-latest state (schema_migrations = 14:
  event-edit + RSVP + moderation). Revisit once organizers is committed/ready.
