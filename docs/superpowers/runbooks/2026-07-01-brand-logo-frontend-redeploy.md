# Brand Logo Wordmark — Frontend Redeploy

_2026-07-01. Frontend-only redeploy to the live demo (`lia.pashteto.com` on vds-ru215).
Build-on-Mac → `save | ssh | load` → cutover. **No backend, no DB migration.** Shipped from
the working tree (uncommitted at deploy time)._

## What shipped

- **New `lia-frontend` image** (`linux/amd64`; id `539d6dacdda2`, tag `amd64-logo`): the home-page
  brand wordmark in `GlassNav` is now a `<BrandLogo>` component — `Presence.` in the bold system
  sans + **Tarski** rendered as inline SVG reusing the *exact* serif glyph paths from the related
  Tarski project logo (`anton-gorokhovatsky.github.io/tarski`, `assets/logo.svg`), recolored to
  `currentColor` so it tracks light/dark. **No webfont added** — respects the design system's
  system-font-only rule.
- Files: new `frontend/components/ui/BrandLogo.tsx`; `GlassNav` `title` prop widened
  `string` → `React.ReactNode`; `app/page.tsx` home nav passes `<BrandLogo />`.

## Procedure (as executed)

1. **Build**: `cd frontend && docker build --platform linux/amd64 --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64-logo .` (next build compiled all routes incl. `/`).
2. **Ship**: `docker save lia-frontend:amd64-logo | gzip | ssh vdska2 'gunzip | docker load'`.
3. **Cutover**: `docker tag lia-frontend:latest lia-frontend:rollback-prelogo` (preserves `7173f5ffd635` = `amd64-orghub`), `docker tag lia-frontend:amd64-logo lia-frontend:latest`, `docker rm -f lia-frontend`, `docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`.
4. **Cleanup**: `docker image prune -f` + `docker builder prune -f`; trimmed `rollback-*` tags to the most recent 3 (prelogo/preorghub/predatefilter). Disk 59% (7.8 GB free).

## Verification (prod, live)

- `GET https://lia.pashteto.com/` → **200**.
- Home HTML carries `aria-label="Presence.tarski"` and the serif mark `viewBox="0 0 102 44"` (wordmark live).
- `<title>` unchanged (`Presence.Tarski — События`).

## Rollback

`docker tag lia-frontend:rollback-prelogo lia-frontend:latest && docker rm -f lia-frontend && docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest`. `7173f5ffd635`.

## Notes

- Deployed from the **working tree** — the BrandLogo change is **not yet committed**
  (`BrandLogo.tsx`, `GlassNav.tsx`, `app/page.tsx`). Commit/push when ready.
