# Liquid Glass refresh + sun/moon theme switch — Design

_Date: 2026-06-25. Branch base: `feat/passwords-gateguard-and-events-user-data`. Frontend: Next.js App Router + TS + Tailwind v4. Source of truth for the design language remains `design/DESIGN.md`; this spec layers an iOS-26 "Liquid Glass" pass on top of the existing Apple-HIG implementation._

## Goal

Two user-requested changes:

1. Replace the text theme toggle ("Светлая"/"Тёмная") with a **sliding sun/moon pill switch**.
2. Push the already-Apple-HIG frontend toward the **iOS 26 "Liquid Glass"** look — stronger translucency, larger continuous radii, floating chrome, micro-interactions — **at the token + primitive level** so every screen benefits without rebuilding layouts.

Non-goals: no new screens, no copy changes, no accent/color-system overhaul (systemBlue stays), no font change, no structural layout rewrites. Glass stays on **chrome only** (nav/tab/sheet) per HIG — content cards stay solid.

## Part 1 — Theme switch (`components/ui/ThemeSwitch.tsx`)

New component replacing `components/ThemeToggle.tsx` at its single call site (`app/page.tsx`, inside `GlassNav` actions). Other screens that import `ThemeToggle` (if any) switch to `ThemeSwitch`.

- Built on the iOS-switch mechanics already in `components/ui/Switch.tsx` (capsule track, sliding knob, transform transition).
- Track ~`56×30px`; knob `26px` sliding left↔right with a spring/ease transition.
- **☀️ pinned at the left edge, 🌙 at the right edge** of the track (dimmed); the knob rides over the active side.
- Light → knob left (over sun); dark → knob right (over moon). Track tint shifts warm-light ↔ deep-dark so the control reads the mode at a glance.
- **Logic unchanged**: keep `useSyncExternalStore` + `subscribe`/`getSnapshot`/`getServerSnapshot` + `applyTheme` exactly as in `ThemeToggle.tsx` (move it into the new file or import it). `role="switch"`, `aria-checked={dark}`, `aria-label="Переключить тему"` preserved. SSR default = light (matches server markup) — no hydration mismatch.
- Respects `prefers-reduced-motion` (no slide animation when set).

## Part 2 — Liquid Glass refresh

### 2.1 `app/globals.css` — material + tokens

- **Glass material** (`@utility glass`): blur `20→30px`, keep `saturate(180%)` (light); lower bg alpha (`--glass` → ~`.6`). Add a **1px inset top highlight** (`box-shadow: inset 0 1px 0 rgba(255,255,255,.5)` light / lower in dark) + a soft outer shadow so chrome reads as a lit translucent slab. Dark keeps blur-only (no saturate) per DESIGN.md; highlight alpha reduced.
- New token `--shadow-glass` for the floating chrome elevation.
- **Radii bump**: `--radius-card 18→22`, `--radius-control 12→14`, `--radius-sheet 20→24`. `--radius-card-sm`, `--radius-fact` nudged proportionally. Capsule unchanged (980).
- Refined card shadows (`--shadow-card`, `--shadow-card-subtle`) for cleaner depth layering under translucent chrome.

### 2.2 Floating chrome

- **`GlassNav`**: detached rounded glass bar — inset from the viewport edges with `--radius-*` corners and `--shadow-glass`, instead of a full-width bar with a hard bottom hairline. Sticky behaviour preserved.
- **`TabBar`** (mobile): floating glass **capsule** sitting above the safe-area inset, the signature iOS 26 look, instead of a full-width bottom bar with a top hairline. Active-tab accent unchanged.

### 2.3 Micro-interactions

- `Button`, `EventCard`, `FilterChip`: `active:scale-[0.97]` press feedback + smoother hover tints; cards lift (shadow-card-subtle → shadow-card) on hover (already partly present — make consistent).
- All transitions gated behind `prefers-reduced-motion: no-preference`.

## Verification

- `pnpm lint` + `pnpm build` clean.
- Run the app locally; capture light + dark screenshots of Discovery, event-detail, create-event **before and after** to confirm the refresh reads as a visible, intentional improvement (not just different). Toggle verified: slides, persists to localStorage, survives reload, no hydration warning.

## Files touched

- New: `components/ui/ThemeSwitch.tsx`
- Edit: `app/globals.css`, `components/ui/GlassNav.tsx`, `components/ui/TabBar.tsx`, `components/ui/Button.tsx`, `components/ui/EventCard.tsx`, `components/ui/FilterChip.tsx`, `app/page.tsx` (swap toggle), any other `ThemeToggle` import sites.
- Remove (or keep as thin re-export): `components/ThemeToggle.tsx`.
