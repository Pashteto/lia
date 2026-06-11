# Lia — Design System (Apple HIG)

Active design direction. Supersedes the `stitch_/curatorial_minimalist` system, which is kept as historical reference.

---

## Philosophy

Build within Apple's Human Interface Guidelines; invent minimally. Distinctiveness comes from content — photography, event context, and the Russian curatorial voice — not from custom chrome. Design lives in code: the reference screens are the spec.

The platform is web-first, rendered Apple-style (system font stack, Apple materials, native control patterns). The iOS app will later inherit the same language natively via SwiftUI. Light and dark themes are supported from day one.

---

## What stays / what changes

**Stays:**

- Russian curatorial voice and copy ("записаться", "участники", "ведущий/медиатор", no hype words)
- Generous whitespace and content-first hierarchy
- Large editorial covers

**Changes (core):**

| Area | Change |
|---|---|
| Typography | Drop Noto Serif + Inter. One system font stack `-apple-system, "SF Pro Text", "SF Pro Display", system-ui, "Segoe UI", Roboto, sans-serif`. Apple type ramp: Large Title (34) → Title 2 → Headline → Body (17) → Subhead → Footnote → Caption. SF covers Cyrillic. |
| Color | Apple semantic system colors (see Color tokens), defined for light and dark. |
| Elevation/materials | Apple materials: translucent blurred Liquid Glass for navigation layers (nav bars, tab bars, sheets, popovers); thin separators; subtle system shadows on cards/sheets. Liquid Glass used on chrome only, not on content (per Apple guidance). |
| Shape | Continuous ("squircle") radii: ~10–12px controls, 16–18px cards, 20px+ sheets. |
| Controls | Native patterns: filled/tinted/plain buttons (not pills), segmented controls, switches, steppers, grouped inset lists (Settings-style), large-title navigation, Apple-style search field, sheets/popovers. |

---

## Color tokens

### Color (light / dark)

| Token | Light | Dark |
|---|---|---|
| `bg` (base) | `#FFFFFF` | `#000000` |
| `bg-grouped` | `#F2F2F7` | `#000000` (intentional per UIKit — grouped background is pure black in dark mode, same as `bg`) |
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

---

## Typography

System font stack:

```
-apple-system, "SF Pro Text", "SF Pro Display", system-ui, "Segoe UI", Roboto, sans-serif
```

### Type ramp

| Style | Size / weight | Use |
|---|---|---|
| Large Title | 34 / 700, tracking −.022em | Screen titles |
| Title 2 | 28–30 / 700 | Section / event titles |
| Headline | 17 / 600 | Card titles, emphasis |
| Body | 16–17 / 400 | Body, list rows |
| Subhead | 15 / 400–500 | Secondary |
| Footnote | 13 / 400 | Meta lines |
| Caption / Kicker | 11–12 / 600, uppercase, tracking +.02–.03em, accent color | Category kickers, section labels |

---

## Spacing / radius

- 4pt base grid; common steps: 4 / 8 / 12 / 16 / 20 / 24.
- Radius: controls 10–12px; cards 16–18px; sheets 20px+; switches/capsule chips fully rounded (980px) — capsule filters are the one intentional "pill" (matches Apple capsule filter chips).

---

## Materials & elevation

**Liquid Glass** is applied to chrome layers only: nav bars, tab bars, sheets, and popovers. It is not used on content cards or editorial surfaces (per Apple HIG guidance).

CSS implementation:

```css
backdrop-filter: saturate(180%) blur(20px);
```

> Pair with the `glass` background token. Dark mode omits `saturate` — use plain `blur(20px)` over `rgba(30,30,30,.72)`.

Light surface: `rgba(249,249,249,.72)` + blur 20px saturate 180%.  
Dark surface: `rgba(30,30,30,.72)` + blur 20px.

Cards and sheets use subtle system shadows (not heavy drop shadows). Hairline borders are replaced by thin separators (`separator` token).

---

## Component mapping

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

---

## Reference screens

1. [Главная / Discovery](screens/discovery.html) — large-title "События", Apple search field, capsule filter row, event card grid on grouped bg, glass nav (desktop) / glass tab bar (mobile).
2. [Создание события](screens/create-event.html) — grouped inset form (Settings-style): sections Основное / Формат / Место и время / Участники / Публикация; segmented (format, publication status), switch (Бесплатно), stepper (capacity), cover upload; glass nav with Отмена/Сохранить.
3. [Детали события + регистрация](screens/event-detail.html) — large cover → large title → key-facts block (когда/где/участники/формат) on grouped bg → sections О встрече / Ведущий (avatar) / Место; sticky glass bottom bar with price + "Записаться". (Detail/reading views use a plain white `bg` base with grouped blocks layered on top — the inverse of list screens, per Apple's `systemBackground` vs `systemGroupedBackground` convention.)
4. [AI-поиск / ассистент](screens/ai-search.html) — natural-language curatorial query as a user bubble → short assistant reply → result cards each with a "почему подошло" rationale; Liquid Glass input bar.

[Все экраны](screens/index.html)

---

## Decisions

| Decision | Choice |
|---|---|
| Accent color | **systemBlue** — `#007AFF` light / `#0A84FF` dark. Single accent token; trivially swappable to a custom brand accent without touching components. |
| Web font | **System font stack only.** Renders SF Pro on Apple devices, falls back to OS system font (Segoe UI / Roboto) elsewhere. Do NOT bundle SF Pro as a webfont — Apple's license restricts it. Defer to Legal if a single cross-platform font is ever wanted. |
| Theme support | **Light + dark from day one.** All color tokens are defined for both modes. |
| How literal to Apple | **Literal.** Build within HIG, invent minimally. Distinctiveness comes from content (photography, Russian curatorial voice), not custom chrome. |
| Primary platform | **Web-first, rendered Apple-style.** iOS app later inherits the same language natively via SwiftUI. |
