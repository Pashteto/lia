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
| `app/page.tsx` | **Discovery** screen. |
| `app/events/[id]/page.tsx` | **Event detail** screen (fetches `GET /events/{id}`). |
| `app/events/new/page.tsx` | **Create event** form (`POST /events`). |
| `app/search/page.tsx` | AI-search — stub. |
| `app/providers.tsx` | TanStack Query provider. |
| `components/ui/` | Design-system primitives: `Button`, `EventCard`, `FilterChip`, `SearchField`, `GlassNav`, `TabBar`, `Kicker`, `Segmented`, `Switch`. |
| `components/` | Composed pieces: `DiscoveryFeed`, `CreateEventForm`, `VenuePicker` (venue search/create typeahead), `ComingSoon`, `ThemeToggle`. |
| `lib/` | `types.ts` (domain + API types), `api.ts` (backend client + mappers), `mock-events.ts` (fallback data), `format.ts`, `cn.ts`. |

## Screen status

| Screen | Source | Status |
|--------|--------|--------|
| Главная / Discovery | `design/screens/discovery.html` | ✅ built (`app/page.tsx`) — live API data |
| Детали события | `design/screens/event-detail.html` | ✅ built (`app/events/[id]`) |
| Создание события | `design/screens/create-event.html` | ✅ built (`app/events/new`) — RHF + Zod, `POST /events`, category multi-select + venue pick-or-create typeahead |
| AI-поиск | `design/screens/ai-search.html` | ⏳ stub (needs the `ai` backend module) |

## Notes

- **No SF Pro webfont** — system font stack only (Apple license; see `DESIGN.md` Decisions). SF renders on Apple devices, OS font elsewhere.
- Dark mode: `prefers-color-scheme` plus an explicit `.dark` class (the in-nav theme toggle flips it for review).
- Discovery fetches published events from the **Go backend** (`GET /api/v1/events?status=published`) server-side for SSR (`lib/api.ts`), with TanStack Query owning client-side refetch (`app/providers.tsx`). If the backend is unreachable it **falls back to mock data** (`lib/mock-events.ts`) so the page still renders. Set the API base via `NEXT_PUBLIC_API_URL` (see `.env.example`; defaults to `http://localhost:8080`).
- Events carry **`categories[]`** (curated taxonomy — multi-select chip picker on create, chips on detail, filter on Discovery) and a **`venue`** object (pick-or-create typeahead via `GET`/`POST /venues`). Both are normalized backend entities. Cover images and organizer profiles are not in the model yet, so those degrade gracefully. Mock cover images come from Unsplash — see `next.config.ts` `images.remotePatterns`; swap for the real S3/CDN host later.
- Forms use **React Hook Form + Zod**; data mutations use **TanStack Query** (`createEvent` in `lib/api.ts`).
- UI copy is **Russian**; identifiers/components stay **English**.
