# Frontend Scaffold — Implementation Plan (executed)

**Goal:** Stand up the Lia web client as a Next.js (App Router) + TypeScript + Tailwind v4 + pnpm app that implements the Apple-HIG design system (`design/DESIGN.md`), with the **Discovery** reference screen built for real and the other three screens as placeholder routes.

**Status:** ✅ Done. `pnpm build` + `pnpm lint` pass; Discovery renders the expected Russian content in light/dark.

> **Follow-on since this plan** (see [`../../HANDOFF.md`](../../HANDOFF.md)): wired Discovery to the live API (SSR + TanStack Query), built the **event-detail** and **create-event** screens, and mapped the enriched category/venue fields. AI-search remains the only stub. Since then (both merged): category became a **multi-select over a curated taxonomy** (`GET /categories` picker + `categories[]` rendering), and the free-text venue inputs became a **pick-or-create typeahead** (`VenuePicker` over `GET`/`POST /venues`, submitting `venue_id`). See [`../specs/2026-06-13-category-normalization-design.md`](../specs/2026-06-13-category-normalization-design.md) and [`../specs/2026-06-14-venue-normalization-design.md`](../specs/2026-06-14-venue-normalization-design.md). **Next**: AI-search screen + `ai` module.

**Decisions:** Foundation + one worked example (Discovery). Tailwind v4 (CSS-first `@theme`). pnpm. System font stack only (no SF webfont — Apple license, per DESIGN.md).

---

## Task 1: Initialize the Next.js app
- Stash the curated `frontend/README.md`; run `pnpm create next-app@latest . --ts --tailwind --eslint --app --no-src-dir --import-alias "@/*" --use-pnpm --no-turbopack --yes`; restore the README.
- Approve native builds (`sharp`, `unrs-resolver`) via `pnpm-workspace.yaml` (`allowBuilds` / `onlyBuiltDependencies`), then `pnpm rebuild`.
- **Verify:** Next 16 + Tailwind v4 present (`@tailwindcss/postcss`, `@import "tailwindcss"`).

## Task 2: Port design tokens + base layout
- `app/globals.css`: all tokens from `design/DESIGN.md` as CSS variables with light defaults + `prefers-color-scheme` **and** `.dark` overrides, exposed via `@theme inline` (colors, radii, shadows, system font). `@utility glass` = `backdrop-filter: saturate(180%) blur(20px)` (dark drops `saturate`).
- `app/layout.tsx`: `lang="ru"`, system font, `bg-grouped` / `text-label`, no webfont import.

## Task 3: Base UI primitives (`components/ui/`)
- `Button` (filled/tinted/plain), `EventCard` (content card, 18px radius, subtle shadow, no glass), `FilterChip` (capsule), `SearchField` (fill bg), `GlassNav` (large-title chrome), `TabBar` (mobile glass bottom), `Kicker`.
- `lib/`: `types.ts` (LiaEvent/Organizer/Venue), `mock-events.ts` (Russian curatorial samples), `format.ts` (ru date/price/attendance), `cn.ts`.

## Task 4: Discovery worked example
- `app/page.tsx` + `components/DiscoveryFeed.tsx` (client: filter chips + search over mock data) + `components/ThemeToggle.tsx` (flips `.dark` for review). Large-title «События», search, capsule filters, responsive card grid, glass nav (desktop) / tab bar (mobile).
- `next.config.ts`: allow `images.unsplash.com` (mock covers).

## Task 5: Placeholder routes
- `app/events/[id]/page.tsx`, `app/events/new/page.tsx`, `app/search/page.tsx` → `components/ComingSoon.tsx`, each with a `// TODO: build from design/screens/<name>.html` pointer.

## Task 6: README
- Rewrote `frontend/README.md`: getting started, folder map, token-source note, screen-status table.

## Verification (performed)
- `pnpm lint` → clean. `pnpm build` → success; `/` prerendered static, TypeScript passes.
- `next start` smoke: `/` returns 200 and contains «События», «Память и архив», «Медиации»; `/search` returns 200.

## Out of scope / next
- Build event-detail, create-event, ai-search screens from their reference HTML.
- Wire real data via TanStack Query against the Go backend (`GET /api/v1/events`); swap the Unsplash image host for the real S3/CDN host.
- React Hook Form + Zod for the create-event form.
