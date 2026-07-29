# Frontend

Web client for Presence.Tarski — **Swiss Grid** design system
(`docs/Redesign/5/design_handoff_presence_swiss_grid/`).

Stack: **Next.js App Router + TypeScript + Tailwind v4 + pnpm**.
Fonts: Golos Text / Manrope / JetBrains Mono (Cyrillic-complete substitutes for the handoff's Archivo / Space Grotesk).

## Getting started

```bash
pnpm install
pnpm dev      # http://localhost:3000
pnpm build
pnpm lint
pnpm test
```

## Layout

| Path | Description |
|------|-------------|
| `app/globals.css` | Swiss Grid tokens + type utilities (`cap`, `lbl`, `kick`, `swiss-focus`, `hover-invert`). |
| `app/layout.tsx` | Root layout: `next/font` Golos/Manrope/JB Mono, `TabBarGate`. |
| `components/ui/` | Primitives: `Button`, `Chip`, `Field`, `EventModule`, `AppHeader`, `BottomTabBar`, … |
| `lib/` | API client, formatters, Swiss helpers. |

## Notes

- Zero border radius, zero shadows; 1px hairlines; mono numerals; signal red only for needs-attention.
- Admin is `data-surface="ink"`; A2–A4 desktop-only below 900px.
- Env: `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_YANDEX_MAPS_KEY` (both required at build for prod).
