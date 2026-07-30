# Design — U1 feed covers

**Date:** 2026-07-30
**Repo:** `Pashteto/lia` · branch base: `main` @ `27a0fc2`
**Design authority:** `docs/Redesign/5/design_handoff_presence_swiss_grid/` (README + `Presence Swiss Grid - Full System.dc.html`)
**Status:** approved design, ready for an implementation plan

---

## 1. Why

The canonical Swiss Grid mock was updated (commit `27a0fc2`) with exactly one new
delta: the **U1 catalogue grid gains cover imagery**. Desktop cells get a
full-bleed cover band on top of the module; mobile rows get a square thumbnail in
place of the bare numeral slot. The mock's own note states the intent — *«Обложка
занимает верх модуля во всю ширину ячейки, без отступов и скруглений — как
приклеенная репродукция»*.

Naming: this is **U1 · Лента событий** in the canonical README. U2 is the event
detail page, which has had a cover strip since Phase 2.

Nothing in the shipped frontend renders imagery in the feed: `EventCover` is
imported only by `EventDetailView.tsx`, and `DiscoveryFeed.tsx` contains no
image reference at all.

### Live cover coverage

`GET https://api.presence.tarski.ru/api/v1/events?status=published` (2026-07-30):
9 published events — 7 seeded (`id` prefix `b0000000-0000-0000-0000-`), 1 with an
uploaded `cover_url`, **1 with no resolvable cover**. All eight curated assets
(`/covers/{festival,mediation,lecture,workshop,concert,exhibition,performance,film}.jpg`)
return 200 on prod, 100–263 KB each.

`lib/covers.ts` deliberately serves a curated category photo **only** to seeded
events, so events created by real organizers stay blank until they upload one.
That policy is unchanged by this work — which makes the blank state a first-class
design concern rather than an edge case.

---

## 2. Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | Blank cover renders a **typographic numeral plate**, not an empty band | The numeral already carries category identity in this system (Swiss rule: numerals instead of colours), so absence becomes part of the system rather than a hole. A grid of empty grey bands reads as "failed to load". The mock hints at this itself — the mobile thumbnail's placeholder *is* the numeral. |
| D2 | Cover band is **`aspect-[5/2]`**, not the mock's fixed `88px` and not the README's `3:2` | At `max-w-[1360px]` a feed column is ~453px. Fixed 88px there is 5.1:1 and keeps only ~26% of a 3:2 source's height — photos of people get cut through the heads. `3:2` (302px) turns the catalogue into an image grid where text is under a third of the module. `5:2` (181px) keeps ~65% of the source frame and leaves the module text-led. |
| D3 | Covers appear in **all three `EventModule` call sites** — feed, `/search`, and the create-event preview | One object, one card. `DiscoverBrowse` uses an identical `grid-cols-3`, so divergence would be arbitrary. The create-event rail is literally labelled «Превью в ленте»; omitting the cover there would make it lie to the organizer who just uploaded one. |
| D4 | `EventCover` stops taking `event: LiaEvent` and takes `src?: string` + `fallback?: ReactNode` | The component needs a URL, not a domain model. Cover-resolution policy stays in `lib/covers.ts`, called from the adapters. Makes the component reusable across module / detail / preview. |
| D5 | Double-download of the hidden layout is suppressed via media-conditioned, **px-only** `sizes`, not a layout rewrite | See §6 — measured against the real component; a `vw` unit anywhere in `sizes` makes the guard inert. |

---

## 3. Component API

```tsx
// components/ui/EventCover.tsx
export function EventCover({
  src,                       // resolved photo URL, or undefined
  sizes,
  aspect = "aspect-[16/9]",
  priority,
  className,
  fallback,                  // rendered when src is undefined; default = bare paper
}: {
  src?: string;
  sizes: string;
  aspect?: string;
  priority?: boolean;
  className?: string;
  fallback?: React.ReactNode;
}): JSX.Element
```

`src` present → `next/image` with `fill` + `object-cover` (unchanged behaviour).
`src` absent → `fallback` inside the same aspect box on `bg-cell-blank`, or a bare
`bg-cell-blank` box if no fallback is given.

`bg-cell-blank` resolves to `--cell-blank: #eceae4` — the exact tone the mock uses
for the empty cover slot, already the established blank-cell token in `Skeleton`,
`CalendarView` and the map's unavailable state. Not `bg-paper`: feed cells already
sit on paper (`#f2f0ec`), so a paper plate would be invisible.

```tsx
// components/ui/EventModule.tsx — one new optional prop
cover?: string;
```

The fallback plate is composed from the existing `numeral` and `category` props;
no new data enters the component.

### Call-site updates

| Site | Change |
|---|---|
| `lib/event-module.ts` | `eventToModuleProps` returns `cover: coverPhoto(event)`; `EventModuleData` gains `cover?: string` |
| `components/EventDetailView.tsx:43` | `event={event}` → `src={coverPhoto(event)}` |
| — side effect | `EventCover`'s blank state moves from `bg-paper` to `bg-cell-blank`, so a coverless **event detail** hero also changes tone. Deliberate: one blank-cover tone across the product. No fallback plate on the detail hero — a 3:1 strip with a giant numeral would overpower the page. |
| `components/CreateEventForm.tsx:863` | add `cover={coverPreviewUrl}` |
| `components/DiscoveryFeed.tsx:241` | grid gains `auto-rows-fr` |
| `components/DiscoverBrowse.tsx:177` | grid gains `auto-rows-fr` |
| `app/design-preview/page.tsx` | one of the three sample modules gets a `cover`, one stays blank to show the plate |

`auto-rows-fr` is required: without it cells in a row size independently and the
titles stop sitting on a shared line.

---

## 4. Desktop cell

```
┌───────────────────────────────────┐
│                                   │
│            [ cover ]              │  aspect-[5/2]  ·  border-b border-rule-inner
│                                   │
├───────────────────────────────────┤
│ 07                     ФЕСТИВАЛИ  │  px-[14px] py-[11px]
│ Летний фестиваль медиаискусства   │  15px / 900
│ МУЗЕЙ «ГАРАЖ»                     │  .cap
│ 25–26.07                    FREE  │  mono 11px / 700  ·  12px / 900
└───────────────────────────────────┘
```

Band: `flex-none`, `aspect-[5/2]`, hairline `border-b border-rule-inner`.
Content column: `flex flex-1 flex-col px-[14px] py-[11px]` (the mock moved the
vertical padding 12 → 11). Everything below the band is unchanged, including the
U3 «Совпало: …» line on its hairline top rule.

**Fallback plate**, inside the same 5:2 box:

```
┌───────────────────────────────────┐
│                                   │
│  07                   ФЕСТИВАЛИ   │  bg-cell-blank · flex items-center
│                                   │  justify-between px-[14px]
└───────────────────────────────────┘
```

Numeral `font-mono text-[26px] font-bold`, category `.cap`. `text-[26px]` is an
**existing** step in the codebase (13 uses) — this spec adds no new ad-hoc type
size, deliberately, given the 29-distinct-sizes debt recorded in §9.

---

## 5. Mobile row

`grid-cols-[22px_1fr_auto]` → `grid-cols-[44px_1fr_auto]`.

```
┌──────┬──────────────────────┬──────┐
│      │ Медиация в залах     │ FREE │
│  07  │ старых мастеров      │      │   44×44 · row-span-2 · border border-ink · self-start
│      │ 07 · ГМИИ · 12.07    │      │   caption col-span-2
└──────┴──────────────────────┴──────┘
```

The numeral leaves its own slot and joins the caption: `07 · ГМИИ им. Пушкина ·
12.07`. Fallback inside the 44×44 box: the numeral, `font-mono text-[10px]
font-bold`, centred on `bg-cell-blank`. The `border border-ink` applies in both
states — photo and plate — per the mock.

Micro-deviation from the mock: the caption gets `col-span-2`. In the mock it sits
in the title column only, where «07 · ГМИИ им. Пушкина · 12.07» starts wrapping
on narrow viewports.

---

## 6. Image loading

`EventModule` renders the desktop and mobile layouts as **two separate DOM
subtrees** toggled by `hidden` / `sm:flex`. `next/image` in a `display:none`
subtree still downloads, so naively each card would fetch both a band and a
thumbnail. On a 390px viewport the hidden desktop band at `sizes="33vw"` resolves
to a ~129px-wide image; across a 42-card feed that is ~400 KB paid for pixels the
phone never shows.

Suppressed by making each instance resolve to the smallest srcset candidate when
it is the hidden one:

| Instance | `sizes` |
|---|---|
| Desktop band | `(max-width: 639px) 1px, (max-width: 1023px) 340px, 460px` |
| Mobile thumbnail | `(min-width: 640px) 1px, 44px` |

**Both strings must be entirely in `px`.** Measured on the real component with
`renderToStaticMarkup` (Next 16.2.9): Next builds the srcset ladder from whether
`sizes` contains a viewport unit. With any `vw` in the string the smallest
candidate is `256w`; with px-only values the ladder starts at `32w`.

| `sizes` | srcset widths |
|---|---|
| `33vw` | 256, 384, 640, … 3840 |
| `(max-width: 639px) 1px, 33vw` | 256, 384, 640, … 3840 |
| `44px` or `1px` | **32**, 48, 64, 96, 128, 256, … |

So the `1px` guard is inert next to `33vw` — the phone picks `256w` either way.
Expressed in px it is real: the hidden band resolves to `1px` and the browser
takes `32w` (~1–2 KB) instead of `256w` (~20 KB). The desktop hints are px
approximations of the same column — 460px at the `1360px` container, 340px below
`1024px` — which land on the same ladder rung `33vw` would have chosen.

`priority` is set on **no** module cover — the feed's LCP element is the H1, and
covers load lazily by default. (The detail-page hero keeps its `priority`.)

Skeletons: `h-[140px]` in both grids → `h-[290px] max-sm:h-[66px]`, matching the
new cell height so the loading → results transition doesn't shift layout.

---

## 7. Tests

The repo has no jsdom and no Testing Library. Component tests follow the existing
pattern in `components/__tests__/map-browse-control.test.tsx`: `renderToStaticMarkup`
from `react-dom/server` plus string assertions on the markup. Verified by spike that
both `next/link` and `next/image` render under it with no mocking — a cover with a
photo emits `data-nimg="fill"`, `loading="lazy"`, the `sizes` string verbatim, and a
`/_next/image?url=…` srcset.

| File | Coverage |
|---|---|
| `lib/__tests__/event-module.test.ts` (update) | the `toEqual` assertion gains `cover`; new cases — seeded event resolves to its category photo, non-seeded event without upload resolves to `undefined` |
| `lib/__tests__/covers.test.ts` (new) | `coverUrl` wins over the curated photo; seed-prefix gating; event with no category → `undefined` |
| `components/__tests__/event-module-cover.test.tsx` (new) | with `cover` an `img` renders; without `cover` the plate renders the numeral and category; mobile row places the numeral in the caption |

---

## 8. Spec amendments (part of the work, not a follow-up)

`docs/Redesign/5/design_handoff_presence_swiss_grid/README.md`:

1. **U1 section** — data tuple `{ i, cat, t, v, d, p }` → `{ i, cat, t, v, d, p, img }`, plus a
   sentence describing the band and the numeral-plate fallback.
2. **Assets section** — «Production must serve properly sized, cropped 3:2 images»
   becomes: sources are 3:2; the feed crops to 5:2, the detail hero to 3:1
   (3:2 below `md`). Covers remain `object-fit: cover`, never letterboxed.

Without this the mock and the spec stay in contradiction and the next session
re-derives the ratio from scratch.

### Recorded deviations

| From | Deviation | Why |
|---|---|---|
| Mock: fixed `88px` band | `aspect-[5/2]` | D2 — a ratio scales with the column; 88px at 453px destroys the frame |
| README: `3:2` in the feed | `5:2` in the feed | D2 — 3:2 makes the catalogue an image grid |
| Mock: `«Обложка»` text placeholder | numeral + category plate | D1 |
| Mock: caption in the title column | caption `col-span-2` | wrapping on narrow viewports |

---

## 9. Explicitly out of scope

- **Type scale.** 347 occurrences of `text-[Npx]` across 29 distinct values,
  including near-duplicate pairs 11/11.5, 12/12.5, 13/14, 8/9. Collapsing these
  to ~7 named `type-*` utilities is its own pass.
- **AI «Подбор».** `/search` is a filter form with a `matchReason` stub; making it
  a real assisted-discovery flow is a product-sized project.
- **Formal Lighthouse / CLS budget.** More relevant after this change, still separate.
- **A1 «Пользователей» tile** — locked P6 deviation #5, needs product sign-off.
- **Calendar and map surfaces.** Neither uses `EventModule`; they keep their
  current treatment.

---

## 10. Constraints carried forward

Zero border radius, zero shadows, 1px hairlines. Mono numerals. Signal red only
for needs-attention / destructive. UI copy Russian, code and commits English.
Fonts Golos Text / Manrope / JetBrains Mono. No icon library.

Frontend-only change: no backend, no migration — prod LIA DB stays at 020.
Deploy is images-only, built `linux/amd64` on the Mac and shipped
`docker save | gzip | ssh | docker load`; the frontend build needs **both**
`NEXT_PUBLIC_API_URL` and `NEXT_PUBLIC_YANDEX_MAPS_KEY`.
