# Frontend

Web client for the Lia event-discovery MVP.

Stack: **Next.js (App Router) + TypeScript + Tailwind v4 + pnpm**, implementing the Apple HIG-based design system in [`../design/DESIGN.md`](../design/DESIGN.md) (system font stack, Apple semantic colors, Liquid Glass, light + dark). Reference screens: [`../design/screens/`](../design/screens/).

## Getting started

```bash
pnpm install
pnpm dev      # http://localhost:3000
pnpm build    # production build
pnpm lint
```

## Layout

| Path | Description |
|------|-------------|
| `app/globals.css` | **Design tokens.** Apple-HIG colors (light/dark), radii, shadows, system font stack, and the `glass` Liquid-Glass utility — exposed to Tailwind via `@theme`. Sourced from `../design/DESIGN.md`. |
| `app/layout.tsx` | Root layout: `lang="ru"`, system font, grouped background. |
| `app/page.tsx` | **Discovery** screen (the built example). |
| `app/events/[id]`, `app/events/new`, `app/search` | Stub routes (placeholders). |
| `components/ui/` | Design-system primitives: `Button`, `EventCard`, `FilterChip`, `SearchField`, `GlassNav`, `TabBar`, `Kicker`. |
| `components/` | Composed pieces: `DiscoveryFeed`, `ComingSoon`, `ThemeToggle`. |
| `lib/` | `types.ts` (domain types), `mock-events.ts` (sample data), `format.ts`, `cn.ts`. |

## Screen status

| Screen | Source | Status |
|--------|--------|--------|
| Главная / Discovery | `design/screens/discovery.html` | ✅ built (`app/page.tsx`) |
| Детали события | `design/screens/event-detail.html` | ⏳ stub |
| Создание события | `design/screens/create-event.html` | ⏳ stub |
| AI-поиск | `design/screens/ai-search.html` | ⏳ stub |

## Notes

- **No SF Pro webfont** — system font stack only (Apple license; see `DESIGN.md` Decisions). SF renders on Apple devices, OS font elsewhere.
- Dark mode: `prefers-color-scheme` plus an explicit `.dark` class (the in-nav theme toggle flips it for review).
- Data is currently **mock** (`lib/mock-events.ts`). Wiring to the Go backend API (TanStack Query) is the next step. Mock cover images come from Unsplash — see `next.config.ts` `images.remotePatterns`; swap for the real S3/CDN host when integrating.
- UI copy is **Russian**; identifiers/components stay **English**.
