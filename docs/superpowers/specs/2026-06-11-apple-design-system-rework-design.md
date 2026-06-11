# Design Spec — Apple Design System Rework (Lia)

**Date:** 2026-06-11
**Status:** Approved (design), pending implementation plan
**Author:** brainstormed with the user (pavel.dodonov@gateway.fm)

## 1. Background & Goal

Lia is a web-first discovery platform for participatory cultural practices in Moscow (iOS to follow). The existing design direction (`design/stitch_/curatorial_minimalist/DESIGN.md`) is a bespoke "Curatorial Minimalist" system: Noto Serif + Inter, custom Material-derived color tokens, hairline-only elevation, pill buttons.

Per Anton Gorokhovatsky's proposal — *"invent less, take Apple's design system and build the app within it"* — and the supporting articles (design lives in code / "truth to materials"; "design and engineering as one"), we are **replacing the bespoke system with Apple's Human Interface Guidelines (HIG) as the foundation.**

**Goal of this task:** rework the `design/` folder to an Apple-based design system. Deliverable = a rewritten design-system doc + 4 reference screens (HTML, light + dark). **Frontend application code is out of scope** for this task.

## 2. Decisions (confirmed with user)

| Decision | Choice |
|---|---|
| How literal to Apple | **Literal** — build within HIG, invent minimally. Distinctiveness comes from content (photography, curatorial Russian voice), not custom chrome. |
| Deliverable | **Design doc + 4 reference screens.** Do not touch frontend code. |
| Primary platform | **Web-first, rendered Apple-style.** Apple HIG reproduced on web (system font stack, materials, native control patterns). iOS app later inherits the same language. |
| Theme support | **Light + dark from the start.** |
| Accent color | **systemBlue** (`#007AFF` light / `#0A84FF` dark). Single accent token, trivially swappable. |
| Web font | **System font stack only.** Renders SF on Apple devices, falls back to OS system font (Segoe/Roboto) elsewhere. Do NOT bundle SF Pro as a webfont — Apple's license restricts it (defer to Legal if a single cross-platform font is ever wanted). |
| Reference screens | Discovery, Create event (organizer), Event detail + registration, AI search assistant. |

## 3. What stays / what changes

**Stays:** Russian curatorial voice and copy ("записаться", "участники", "ведущий/медиатор", no hype words); generous whitespace and content-first hierarchy; large editorial covers.

**Changes (core):**

- **Typography** — drop Noto Serif + Inter. One system font stack `-apple-system, "SF Pro Text", "SF Pro Display", system-ui, "Segoe UI", Roboto, sans-serif`. Apple type ramp: Large Title (34) → Title 1/2/3 → Headline → Body (17) → Callout → Subhead → Footnote → Caption. SF covers Cyrillic.
- **Color** — Apple semantic system colors (see §4), defined for light and dark.
- **Elevation/materials** — Apple materials: translucent blurred **Liquid Glass** for navigation layers (nav bars, tab bars, sheets, popovers); thin separators; subtle system shadows on cards/sheets. Liquid Glass used on chrome only, not on content (per Apple guidance).
- **Shape** — continuous ("squircle") radii: ~10–12px controls, 16–20px cards/sheets.
- **Controls** — native patterns: filled/tinted/plain buttons (not pills), segmented controls, switches, steppers, grouped inset lists (Settings-style), large-title navigation, Apple-style search field, sheets/popovers.

## 4. Design tokens

### Color (light / dark)

| Token | Light | Dark |
|---|---|---|
| `bg` (base) | `#FFFFFF` | `#000000` |
| `bg-grouped` | `#F2F2F7` | `#000000` |
| `bg-secondary` (cards on grouped) | `#FFFFFF` | `#1C1C1E` |
| `bg-tertiary` | `#FFFFFF` | `#2C2C2E` |
| `label` | `#000000` | `#FFFFFF` |
| `label-secondary` | `rgba(60,60,67,.6)` | `rgba(235,235,245,.6)` |
| `label-tertiary` | `rgba(60,60,67,.3)` | `rgba(235,235,245,.3)` |
| `separator` | `rgba(60,60,67,.18)` | `rgba(84,84,88,.65)` |
| `fill` (controls) | `rgba(118,118,128,.12)` | `rgba(118,118,128,.24)` |
| `accent` | `#007AFF` | `#0A84FF` |
| `success` (switch on) | `#34C759` | `#30D158` |
| `glass` (material) | `rgba(249,249,249,.72)` + blur 20px saturate 180% | `rgba(30,30,30,.72)` + blur 20px |

### Typography (system font stack)

| Style | Size / weight | Use |
|---|---|---|
| Large Title | 34 / 700, tracking −.022em | Screen titles |
| Title 2 | 28–30 / 700 | Section / event titles |
| Headline | 17 / 600 | Card titles, emphasis |
| Body | 16–17 / 400 | Body, list rows |
| Subhead | 15 / 400–500 | Secondary |
| Footnote | 13 / 400 | Meta lines |
| Caption / Kicker | 11–12 / 600, uppercase, tracking +.02–.03em, accent color | Category kickers, section labels |

### Spacing / radius

- 4pt base grid; common steps 4 / 8 / 12 / 16 / 20 / 24.
- Radius: controls 10–12px; cards 16–18px; sheets 20px+; switches/capsule chips fully rounded (980px) — capsule filters are the one intentional "pill" (matches Apple capsule filter chips).

## 5. Component mapping (current → Apple)

| Element | Apple pattern |
|---|---|
| Primary button | Filled, accent fill, ~12px continuous radius |
| Secondary button | Tinted (accent-tinted fill) or bordered gray |
| Ghost button | Plain (accent text) |
| Event card | Content card on grouped bg: 16–18px radius, subtle system shadow, no hairline border |
| Text field | Apple field: `fill` background, rounded, accent focus ring |
| Filter chips | Capsule filters (multi) + segmented control (mutually exclusive options) |
| Lists / forms | Grouped inset list (Settings-style): section header, rows, inset separators |
| Navigation | Large-title nav bar on Liquid Glass; mobile glass tab bar (bottom) |
| Modals | Sheets (rounded-top, iOS) on mobile; popovers on desktop |
| Registration CTA | Sticky Liquid Glass bottom bar (Apple buy-bar pattern) |

## 6. Reference screens (validated in visual companion)

1. **Discovery / home** — large-title "События", Apple search field, capsule filter row, event card grid on grouped bg, glass nav (desktop) / glass tab bar (mobile).
2. **Create event (organizer)** — grouped inset form (Settings-style): sections Основное / Формат / Место и время / Участники / Публикация; segmented (format, publication status), switch (Бесплатно), stepper (capacity), cover upload; glass nav with Отмена/Сохранить.
3. **Event detail + registration** — large cover → large title → key-facts block (когда/где/участники/формат) on grouped bg → sections О встрече / Ведущий (avatar) / Место; sticky glass bottom bar with price + "Записаться".
4. **AI search / assistant** — natural-language curatorial query as a user bubble → short assistant reply → result cards each with a "почему подошло" rationale; Liquid Glass input bar.

All four exist as approved HTML mockups under `.superpowers/brainstorm/.../content/` (light + dark, desktop + mobile).

## 7. Deliverables (target state of `design/`)

- `design/DESIGN.md` — the Apple-based design system (this spec's §3–§5 as the canonical doc, replacing the curatorial-minimalist system as the active direction).
- `design/screens/` — the 4 reference screens as standalone HTML files (light + dark), promoted from the validated mockups.
- `design/README.md` — updated to point at the Apple system and screens; note that `stitch_/` is now historical reference (superseded direction), kept for archive.
- Root `README.md` / `frontend/README.md` — design direction reference updated to Apple system.

## 8. Out of scope

- Frontend application code (Next.js scaffolding, real components).
- Backend (covered separately — built from `go-microservice-template` as a single modular monolith; see `backend/README.md`).
- iOS app (later; inherits this language natively via SwiftUI).

## 9. Open considerations (non-blocking)

- Liquid Glass applied tastefully to navigation layers only, not content.
- Accent is a single token — swap to a custom brand accent later if desired without touching components.
- SF Pro licensing: system-stack approach needs no license; bundling SF is a Legal question if ever pursued.
