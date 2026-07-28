# Build brief — Presence Swiss Grid

You are implementing the design specified in `README.md` (read it fully before writing code).

## Ground rules
- `README.md` is the specification. `Presence Swiss Grid - Full System.dc.html` is a **visual
  reference only** — open it to check a layout, never copy its markup, its `<x-dc>`/`<sc-for>`
  tags, or `support.js`.
- Use `tokens.css` / `tokens.ts`. Never hardcode a hex value or a font size that isn't in the
  scale. If a value seems missing, it is in the README's scale table — use the nearest step.
- Build the six shared components first (README → *Shared components*). Do not start screens
  until they render correctly in isolation.
- Zero border radius, zero shadows, 1px hairline rules, uppercase tracking preserved, all
  numerals in JetBrains Mono. These are the identity of the system; a violation reads instantly.

## Suggested stack (if none exists)
Next.js App Router + TypeScript + CSS Modules or Tailwind configured **only** from
`tokens.ts` (disable the default palette, spacing, and radius scales — the design has no
radii and a bespoke spacing set). react-leaflet for maps.

## Definition of done per screen
- Matches the reference at both breakpoints (desktop ≥1024, mobile ≤430).
- Empty, loading, error, and auth-gated states implemented per README → *States*.
- Keyboard navigable, visible 2px square focus outline, tap targets ≥44px on touch.
- Content is real data from the API listed in README → *Data & state per screen*, not the mock
  arrays.
- Russian copy matches the reference unless the content team supplied a replacement.

## Order
Follow README → *Build order*. Ship the public core (U1, U2, U7, U8) before anything else.
