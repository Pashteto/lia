# Runbook — Liquid Glass refresh + sun/moon theme switch (frontend redeploy, 2026-06-26)

Deploys branch `feat/liquid-glass-refresh`: an iOS-26 "Liquid Glass" pass over
the existing Apple-HIG frontend + a sliding sun/moon theme switch (replaces the
text "Светлая"/"Тёмная" toggle). **Frontend-only** — backend, GateGuard, DB, and
the `lia_uploads` volume are untouched.

Live target: `https://lia.pashteto.com` on **vds-ru215** (`193.32.188.7`,
`ssh vdska2`, root). Hand-managed Docker (documented demo exception).

Spec: [`../specs/2026-06-25-liquid-glass-refresh-design.md`](../specs/2026-06-25-liquid-glass-refresh-design.md).

## What changed (code)

- **New** `frontend/components/ui/ThemeSwitch.tsx` — iOS pill switch; knob carries
  the current mode's emoji (☀️/🌙) and slides over the matching edge; track tint
  shifts warm-day ↔ deep-night. Same `useSyncExternalStore` + `applyTheme` logic
  as the old `ThemeToggle` (removed). `role="switch"`, `aria-checked`,
  `prefers-reduced-motion` respected.
- `app/globals.css` — Liquid Glass material (blur 20→30px, lighter alpha, inset
  top highlight + soft shadow), larger continuous radii (cards 18→22, controls
  12→14, sheets 20→24), new `--shadow-glass` token.
- `GlassNav` → detached floating rounded glass bar; `TabBar` → floating glass
  capsule above the safe-area.
- `Button`/`EventCard`/`FilterChip` — press-scale + hover-lift, gated on
  reduced-motion. `Button` also `whitespace-nowrap`.
- **Header overflow fix** (follow-up commit `9b7c795`): the signed-in nav (switch
  + Создать событие + Мои события + email + Выйти) overflowed `max-w-3xl`, jamming
  the switch against the title and wrapping the button. Widened `GlassNav` +
  `DiscoveryFeed` to `max-w-5xl` (kept aligned), added gap + `shrink-0` title, and
  `lg:grid-cols-3` so the wider container doesn't oversize covers.

## Deploy procedure (frontend-only, build-on-Mac + ship)

The box can't reliably pull large layers over the tunnel, so the image is built
on the Mac (`--platform linux/amd64`) and shipped via `docker save | ssh | load`
(same pattern as the passwords runbook; only the frontend image is rebuilt here).

```bash
# 0. (optional but recommended) preserve the current live image for rollback
ssh vdska2 'docker tag $(docker images lia-frontend:latest -q) lia-frontend:rollback'

# 1. build on the Mac
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
curl -sI https://lia.pashteto.com | head -1                    # → 200
curl -s -o /dev/null -w '%{http_code}\n' https://api.lia.pashteto.com/api/v1/events?status=published  # → 200 (untouched)
```

Visual: load `https://lia.pashteto.com` and toggle the switch (slides, persists
across reload). The **signed-in** header must be checked — that's where the
overflow bug was. To screenshot the crowded nav without a real login, pre-seed
the session in a headless browser:

```js
localStorage.setItem("lia.auth.token", "fake.jwt.token");      // any non-null token
localStorage.setItem("lia.auth.email", "you@example.com");     // isAuthed === email !== null
```

(`lia.auth.token` / `lia.auth.email` are the keys in `lib/auth.ts`; `isAuthed` is
just `email !== null`, so a fake token renders the authed header — for visual
checks only, no API call is made on render.)

## Rollback

```bash
ssh vdska2 'docker tag lia-frontend:rollback lia-frontend:latest && \
  docker rm -f lia-frontend && \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

## Verified live (2026-06-26)

- `https://lia.pashteto.com` → 200; API still 200.
- New image deployed (`lia-frontend:latest`), container `Up`, local `127.0.0.1:3001` → 200.
- Sun/moon switch flips `<html>` class, persists to `localStorage.theme`, survives
  reload (Playwright interaction test). Signed-in header renders with correct
  spacing (no switch/title collision, no button wrap).

## Follow-ups / not done

- Branch `feat/liquid-glass-refresh` is **not merged** and has **no PR** — deployed
  from the branch image directly.
- **Image-only deploy**: `frontend/` source was not rsynced to `/opt/lia/frontend`
  on the box, so the on-box reference source is behind the running image.
- The theme switch lives only in the discovery header (its sole `GlassNav` call
  site). Other screens' nav bars don't expose it.
