# U1 Feed Covers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render event cover imagery in the U1 catalogue grid — a `5:2` band on desktop cells, a `44×44` thumbnail on mobile rows, and a typographic numeral plate when an event has no photo.

**Architecture:** `EventCover` is refactored from taking a domain object (`event: LiaEvent`) to taking a plain `src?: string` plus an optional `fallback` node, so cover-resolution policy stays in `lib/covers.ts` and the component becomes reusable. `EventModule` gains one optional `cover?: string` prop and composes its no-photo plate from the `numeral`/`category` props it already receives. `eventToModuleProps` resolves the URL once, so the feed, `/search` and the create-event preview all get covers from a single adapter.

**Tech Stack:** Next 16.2.9 (App Router), React 19.2.4, Tailwind CSS v4 (`@theme inline` tokens in `app/globals.css`), Vitest 4, `react-dom/server` for component tests.

**Spec:** `docs/superpowers/specs/2026-07-30-u1-feed-covers-design.md` — read it before starting. It records the four approved deviations from the mock/README and the reasoning behind the `sizes` strings.

## Global Constraints

- **Working directory for every command:** `frontend/` (the repo root is one level up). All paths in this plan are relative to `frontend/` unless prefixed with `docs/`.
- **Tests:** `npx vitest run <path>` for one file, `npx vitest run` for all. There is **no jsdom and no Testing Library** — component tests use `renderToStaticMarkup` from `react-dom/server` and assert on markup strings, following `components/__tests__/map-browse-control.test.tsx`.
- **Lint:** `npx eslint <paths>`.
- **Swiss Grid, non-negotiable:** zero border radius, zero shadows, 1px hairlines, mono numerals, signal red reserved for needs-attention/destructive. No icon library.
- **Colours come from tokens only** — `bg-cell-blank`, `border-rule-inner`, `border-ink`, `bg-paper`. Never a raw hex in a component.
- **Do not introduce a new `text-[Npx]` value.** The codebase already carries 347 occurrences across 29 distinct sizes; this work reuses existing steps (`text-[26px]`, `text-[10px]`) only.
- **UI copy Russian, code and commit messages English.**
- **Frontend only.** No backend change, no migration. Prod LIA DB stays at 020.
- **Commit after every task.** Do not push; do not merge to `main`.

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `components/ui/EventCover.tsx` | Render a cover box at a given aspect: photo or fallback node | Modify — API becomes `src`/`fallback` |
| `components/ui/EventModule.tsx` | The Swiss feed card, desktop cell + mobile row | Modify — cover band, thumbnail, numeral plate |
| `lib/event-module.ts` | Adapt `LiaEvent` → `EventModuleProps` | Modify — resolve `cover` |
| `components/EventDetailView.tsx` | Event detail page | Modify — one call site |
| `components/CreateEventForm.tsx` | Organizer wizard incl. «Превью в ленте» | Modify — pass the uploaded cover |
| `components/DiscoveryFeed.tsx` | U1 feed | Modify — `auto-rows-fr`, skeleton height |
| `components/DiscoverBrowse.tsx` | U3 `/search` | Modify — `auto-rows-fr`, skeleton height |
| `app/design-preview/page.tsx` | Primitive gallery | Modify — show both cover states |
| `lib/__tests__/covers.test.ts` | Cover-resolution policy | Create |
| `lib/__tests__/event-module.test.ts` | Adapter | Modify |
| `components/__tests__/event-module-cover.test.tsx` | Card markup, both states, both layouts | Create |
| `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` | Design authority | Modify — U1 tuple + Assets ratios |

---

### Task 1: Pin down the cover-resolution policy with tests

`lib/covers.ts` already implements the policy but has no test. Lock it before anything depends on it.

**Files:**
- Test: `lib/__tests__/covers.test.ts` (create)
- Read only: `lib/covers.ts`

**Interfaces:**
- Consumes: `coverPhoto(event: LiaEvent): string | undefined` from `lib/covers.ts`
- Produces: nothing — pure characterization tests

- [ ] **Step 1: Write the failing test**

Create `lib/__tests__/covers.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { coverPhoto } from "../covers";
import type { LiaEvent } from "../types";

const SEED_ID = "b0000000-0000-0000-0000-000000000007";
const REAL_ID = "b4e0d8a3-9a8a-4f10-9d6a-1c2b3d4e5f60";

function event(over: Partial<LiaEvent> = {}): LiaEvent {
  return {
    id: SEED_ID,
    title: "Летний фестиваль медиаискусства",
    categories: [{ id: "c1", slug: "festival", label: "Фестивали" }],
    format: "offline",
    status: "published",
    startsAt: "2026-07-25T15:00:00Z",
    priceType: "free",
    ...over,
  } as LiaEvent;
}

describe("coverPhoto", () => {
  it("an uploaded cover always wins", () => {
    expect(
      coverPhoto(event({ id: REAL_ID, coverUrl: "https://api.example/f/abc" })),
    ).toBe("https://api.example/f/abc");
  });

  it("an uploaded cover wins even on a seeded event", () => {
    expect(coverPhoto(event({ coverUrl: "https://api.example/f/xyz" }))).toBe(
      "https://api.example/f/xyz",
    );
  });

  it("a seeded event falls back to its category photo", () => {
    expect(coverPhoto(event())).toBe("/covers/festival.jpg");
    expect(
      coverPhoto(
        event({ categories: [{ id: "c2", slug: "mediation", label: "Медиации" }] }),
      ),
    ).toBe("/covers/mediation.jpg");
  });

  it("a non-seeded event without an upload has no photo", () => {
    expect(coverPhoto(event({ id: REAL_ID }))).toBeUndefined();
  });

  it("a seeded event with no category has no photo", () => {
    expect(coverPhoto(event({ categories: [] }))).toBeUndefined();
  });

  it("a seeded event in an unmapped category has no photo", () => {
    expect(
      coverPhoto(event({ categories: [{ id: "c9", slug: "reading", label: "Читательские" }] })),
    ).toBeUndefined();
  });
});
```

- [ ] **Step 2: Run the test**

Run: `npx vitest run lib/__tests__/covers.test.ts`
Expected: PASS — this characterizes existing behaviour. If any case fails, **stop and report**: the spec's cover-coverage numbers were derived from this policy and a mismatch means the spec is wrong, not the test.

- [ ] **Step 3: Commit**

```bash
git add lib/__tests__/covers.test.ts
git commit -m "test(frontend): characterize cover resolution policy"
```

---

### Task 2: Refactor `EventCover` to take `src` and a `fallback`

**Files:**
- Modify: `components/ui/EventCover.tsx`
- Modify: `components/EventDetailView.tsx:43-48` (the only current call site)
- Test: `components/__tests__/event-cover.test.tsx` (create)

**Interfaces:**
- Consumes: `coverPhoto` from `lib/covers.ts`
- Produces:
  ```tsx
  EventCover(props: {
    src?: string;
    sizes: string;
    aspect?: string;      // default "aspect-[16/9]"
    priority?: boolean;
    className?: string;
    fallback?: React.ReactNode;
  }): JSX.Element
  ```
  Task 3 renders `EventCover` with `src`, `aspect`, `sizes` and `fallback`.

- [ ] **Step 1: Write the failing test**

Create `components/__tests__/event-cover.test.tsx`:

```tsx
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EventCover } from "@/components/ui/EventCover";

describe("EventCover", () => {
  it("renders a lazy next/image at the requested aspect when src is given", () => {
    const html = renderToStaticMarkup(
      <EventCover src="/covers/festival.jpg" aspect="aspect-[5/2]" sizes="460px" />,
    );
    expect(html).toContain("aspect-[5/2]");
    expect(html).toContain('data-nimg="fill"');
    expect(html).toContain('loading="lazy"');
    expect(html).toContain('sizes="460px"');
    expect(html).toContain("%2Fcovers%2Ffestival.jpg");
  });

  it("renders the fallback node on cell-blank when src is missing", () => {
    const html = renderToStaticMarkup(
      <EventCover aspect="aspect-[5/2]" sizes="460px" fallback={<span>07</span>} />,
    );
    expect(html).toContain("bg-cell-blank");
    expect(html).toContain("<span>07</span>");
    expect(html).not.toContain("<img");
  });

  it("renders a bare cell-blank box when neither src nor fallback is given", () => {
    const html = renderToStaticMarkup(<EventCover sizes="460px" />);
    expect(html).toContain("bg-cell-blank");
    expect(html).not.toContain("<img");
  });

  it("keeps the photo box on paper so a transparent PNG reads as paper", () => {
    const html = renderToStaticMarkup(<EventCover src="/covers/film.jpg" sizes="460px" />);
    expect(html).toContain("bg-paper");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run components/__tests__/event-cover.test.tsx`
Expected: FAIL — the current component requires an `event` prop and never emits `bg-cell-blank`.

- [ ] **Step 3: Rewrite the component**

Replace the whole body of `components/ui/EventCover.tsx`:

```tsx
import { cn } from "@/lib/cn";
import Image from "next/image";
import type { ReactNode } from "react";

/**
 * Event cover — plain photograph, edge-to-edge. No gradients, no decorative
 * glyphs (Swiss Grid P7.3).
 *
 * Takes a resolved URL rather than a LiaEvent: cover-resolution policy lives in
 * `lib/covers.ts` and is applied by the callers' adapters, which keeps this
 * component reusable across the feed card, the detail hero and the wizard
 * preview. Missing photo → `fallback` on the blank-cell tone.
 */
export function EventCover({
  src,
  sizes,
  priority,
  className,
  aspect = "aspect-[16/9]",
  fallback,
}: {
  src?: string;
  sizes: string;
  priority?: boolean;
  className?: string;
  /** Tailwind aspect-ratio utility, e.g. "aspect-[5/2]" (feed) or "aspect-[3/1]" (detail). */
  aspect?: string;
  /** Rendered instead of the photo when `src` is undefined. */
  fallback?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "relative w-full overflow-hidden",
        src ? "bg-paper" : "bg-cell-blank",
        aspect,
        className,
      )}
    >
      {src ? (
        <Image
          src={src}
          alt=""
          fill
          sizes={sizes}
          priority={priority}
          className="object-cover"
        />
      ) : (
        fallback
      )}
    </div>
  );
}
```

- [ ] **Step 4: Update the detail-page call site**

In `components/EventDetailView.tsx`, add the import and change the call:

```tsx
import { coverPhoto } from "@/lib/covers";
```

```tsx
        <EventCover
          src={coverPhoto(event)}
          aspect="aspect-[3/1] max-md:aspect-[3/2]"
          sizes="(max-width: 768px) 100vw, 1360px"
          priority
        />
```

No `fallback` here on purpose: a `3:1` strip carrying a giant numeral would overpower the page (spec §3).

- [ ] **Step 5: Run the tests**

Run: `npx vitest run components/__tests__/event-cover.test.tsx`
Expected: PASS (4 tests)

Run: `npx vitest run`
Expected: PASS — whole suite green. `EventCover` had one call site, so nothing else should move.

- [ ] **Step 6: Typecheck and lint**

Run: `npx tsc --noEmit`
Expected: no errors. A leftover `event={...}` anywhere shows up here.

Run: `npx eslint components/ui/EventCover.tsx components/EventDetailView.tsx components/__tests__/event-cover.test.tsx`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add components/ui/EventCover.tsx components/EventDetailView.tsx components/__tests__/event-cover.test.tsx
git commit -m "refactor(frontend): EventCover takes a resolved src and a fallback node"
```

---

### Task 3: Cover band, thumbnail and numeral plate in `EventModule`

**Files:**
- Modify: `components/ui/EventModule.tsx`
- Test: `components/__tests__/event-module-cover.test.tsx` (create)

**Interfaces:**
- Consumes: `EventCover({ src, sizes, aspect, fallback })` from Task 2
- Produces: `EventModuleProps` gains `cover?: string`. Tasks 4–7 pass it.

The two `sizes` strings are **px-only on purpose** — see spec §6. A `vw` unit anywhere in the string makes Next's srcset ladder start at `256w`, which defeats the guard that stops the hidden layout from downloading a real image.

- [ ] **Step 1: Write the failing test**

Create `components/__tests__/event-module-cover.test.tsx`:

```tsx
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EventModule } from "@/components/ui/EventModule";

const BASE = {
  numeral: "07",
  category: "Фестивали",
  title: "Летний фестиваль медиаискусства",
  venue: "Музей «Гараж»",
  date: "25–26.07",
  price: "FREE",
  href: "/events/evt-1",
} as const;

describe("EventModule covers", () => {
  it("renders the desktop band at 5:2 and the mobile thumbnail when a cover is given", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html).toContain("aspect-[5/2]");
    expect(html).toContain('sizes="(max-width: 639px) 1px, (max-width: 1023px) 340px, 460px"');
    expect(html).toContain('sizes="(min-width: 640px) 1px, 44px"');
    expect(html.match(/%2Fcovers%2Ffestival.jpg/g)?.length).toBeGreaterThanOrEqual(2);
  });

  it("keeps every sizes string free of viewport units so the srcset ladder starts at 32w", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    for (const m of html.matchAll(/sizes="([^"]+)"/g)) {
      expect(m[1]).not.toMatch(/\d(vw|vh|vmin|vmax)/);
    }
    expect(html).toContain("w=32&");
  });

  it("renders the numeral plate instead of a photo when no cover is given", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} />);
    expect(html).toContain("bg-cell-blank");
    expect(html).not.toContain("<img");
    // numeral appears in the desktop plate, the mobile plate and the mobile caption
    expect(html.match(/07/g)?.length).toBeGreaterThanOrEqual(3);
    expect(html).toContain("Фестивали");
  });

  it("moves the numeral into the mobile caption and keeps the 44px column", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html).toContain("grid-cols-[44px_1fr_auto]");
    expect(html).not.toContain("grid-cols-[22px_1fr_auto]");
    expect(html).toContain("07 · Музей «Гараж» · 25–26.07");
  });

  it("still renders the U3 match reason under the content column", () => {
    const html = renderToStaticMarkup(
      <EventModule {...BASE} cover="/covers/festival.jpg" matchReason="маленькая группа" />,
    );
    expect(html).toContain("Совпало: маленькая группа");
  });

  it("wraps everything in exactly one anchor", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html.match(/<a /g)).toHaveLength(1);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run components/__tests__/event-module-cover.test.tsx`
Expected: FAIL — `cover` is not a prop, there is no `aspect-[5/2]`, and the mobile grid is still `22px`.

- [ ] **Step 3: Implement**

Replace the whole body of `components/ui/EventModule.tsx`:

```tsx
import Link from "next/link";
import { cn } from "@/lib/cn";
import { EventCover } from "@/components/ui/EventCover";

/** Cover `sizes` are px-only by design. Next builds the srcset ladder from
 * whether `sizes` carries a viewport unit: with any `vw` the smallest candidate
 * is 256w, with px-only values it starts at 32w. Since the desktop and mobile
 * layouts are separate DOM subtrees toggled by `hidden`/`sm:flex`, and a hidden
 * next/image still downloads, each string resolves to `1px` in the breakpoint
 * where its subtree is invisible — so the browser fetches a 32w placeholder
 * instead of a real image. */
const BAND_SIZES = "(max-width: 639px) 1px, (max-width: 1023px) 340px, 460px";
const THUMB_SIZES = "(min-width: 640px) 1px, 44px";

export interface EventModuleProps {
  numeral: string; // "01" — from categoryNumeral()
  category: string; // "Фестивали"
  title: string;
  venue: string;
  date: string; // preformatted, renders in mono
  price: string; // from priceLabel() — "FREE" | "800 ₽"
  href: string;
  /** Resolved cover URL from coverPhoto(); undefined → numeral plate. */
  cover?: string;
  /** U3 AI variant: match reason on a hairline top rule. */
  matchReason?: string;
  className?: string;
}

/** Swiss Grid feed card.
 * Desktop: 5:2 cover band (or numeral plate) → numeral/category row, 15px/900
 * title, venue caption, footer date(mono 700)+price.
 * Mobile: grid 44px 1fr auto — 44x44 cover / title / price, then the caption
 * carrying numeral · venue · date.
 * Single <Link> wrapper — never nest an <a> inside (React #418 fix). */
export function EventModule({
  numeral, category, title, venue, date, price, href, cover, matchReason, className,
}: EventModuleProps) {
  return (
    <Link
      href={href}
      className={cn("block h-full min-w-0 swiss-focus hover-invert", className)}
    >
      {/* Desktop — handoff U1 catalogue cell */}
      <div className="hidden h-full flex-col sm:flex">
        <EventCover
          src={cover}
          sizes={BAND_SIZES}
          aspect="aspect-[5/2]"
          className="flex-none border-b border-rule-inner"
          fallback={
            <div className="flex h-full items-center justify-between px-[14px]">
              <span className="font-mono text-[26px] font-bold tracking-[-0.02em]">
                {numeral}
              </span>
              <span className="cap">{category}</span>
            </div>
          }
        />
        <div className="flex flex-1 flex-col px-[14px] py-[11px]">
          <div className="flex items-baseline justify-between">
            <span className="font-mono text-[11px] font-bold">{numeral}</span>
            <span className="cap">{category}</span>
          </div>
          <h3 className="mt-[7px] text-[15px] font-black leading-[1.02] tracking-[-0.02em]">
            {title}
          </h3>
          <p className="cap mt-[6px]">{venue}</p>
          <div className="mt-auto flex items-baseline justify-between pt-[9px]">
            <span className="font-mono text-[11px] font-bold">{date}</span>
            <span className="text-[12px] font-black">{price}</span>
          </div>
          {matchReason ? (
            <p className="mt-[8px] border-t border-rule-inner pt-[8px] text-[10px]">
              Совпало: {matchReason}
            </p>
          ) : null}
        </div>
      </div>

      {/* Mobile — handoff U1 row: 44px 1fr auto */}
      <div className="grid grid-cols-[44px_1fr_auto] items-baseline gap-x-[10px] gap-y-[2px] px-[14px] py-[9px] sm:hidden">
        <EventCover
          src={cover}
          sizes={THUMB_SIZES}
          aspect="aspect-square"
          className="row-span-2 self-start border border-ink"
          fallback={
            <div className="flex h-full items-center justify-center">
              <span className="font-mono text-[10px] font-bold">{numeral}</span>
            </div>
          }
        />
        <h3 className="text-[12.5px] font-bold leading-[1.1] tracking-normal">
          {title}
        </h3>
        <span className="text-[10px] font-black">{price}</span>
        <span className="cap col-span-2">
          {numeral} · {venue} · {date}
        </span>
        {matchReason ? (
          <p className="col-span-3 mt-[6px] border-t border-rule-inner pt-[6px] text-[10px]">
            Совпало: {matchReason}
          </p>
        ) : null}
      </div>
    </Link>
  );
}
```

Notes on what changed and why, so a reviewer can check intent:
- The desktop wrapper loses `px-[14px] py-[12px]`; padding moves to the inner content column at `py-[11px]` so the band can sit flush to the cell edges. Title margin `8px → 7px`, footer padding `10px → 9px`, per the mock.
- The mobile grid drops the two `<span aria-hidden />` spacers: the cover now occupies column 1 across both rows via `row-span-2`, and the caption spans the remaining two columns.
- `aspect-square` gives the mobile thumbnail its `44×44` box from the `44px` grid column.

- [ ] **Step 4: Run the tests**

Run: `npx vitest run components/__tests__/event-module-cover.test.tsx`
Expected: PASS (6 tests)

Run: `npx vitest run`
Expected: PASS — `lib/__tests__/event-module.test.ts` still passes; it asserts the adapter, which has not changed yet.

- [ ] **Step 5: Lint**

Run: `npx eslint components/ui/EventModule.tsx components/__tests__/event-module-cover.test.tsx`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add components/ui/EventModule.tsx components/__tests__/event-module-cover.test.tsx
git commit -m "feat(frontend): cover band, thumbnail and numeral plate in EventModule"
```

---

### Task 4: Resolve the cover in the adapter

**Files:**
- Modify: `lib/event-module.ts`
- Modify: `lib/__tests__/event-module.test.ts`

**Interfaces:**
- Consumes: `coverPhoto` (Task 1 characterized it), `EventModuleProps.cover` (Task 3)
- Produces: `EventModuleData` gains `cover?: string`. Tasks 5 and 6 spread this object straight into `EventModule`, so they need no cover logic of their own.

- [ ] **Step 1: Write the failing test**

In `lib/__tests__/event-module.test.ts`, the first case asserts the whole object with `toEqual` and will break. Replace that `it` block and append two new ones:

```ts
  it("maps every module field", () => {
    expect(eventToModuleProps(EVENT, CATS)).toEqual({
      numeral: "02",
      category: "Медиации",
      title: "Медиация по выставке «Свет»",
      venue: "Онлайн",
      date: "12.07 · 16:00",
      price: "FREE",
      href: "/events/evt-1",
      cover: undefined,
    });
  });

  it("a seeded event carries its curated category photo", () => {
    const seeded = {
      ...EVENT,
      id: "b0000000-0000-0000-0000-000000000002",
    } as LiaEvent;
    expect(eventToModuleProps(seeded, CATS).cover).toBe("/covers/mediation.jpg");
  });

  it("an uploaded cover wins over the curated photo", () => {
    const uploaded = {
      ...EVENT,
      id: "b0000000-0000-0000-0000-000000000002",
      coverUrl: "https://api.example/f/abc",
    } as LiaEvent;
    expect(eventToModuleProps(uploaded, CATS).cover).toBe("https://api.example/f/abc");
  });
```

`EVENT.id` is `"evt-1"` — not a seed prefix and no `coverUrl` — so the first case expects `cover: undefined`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run lib/__tests__/event-module.test.ts`
Expected: FAIL — `cover` is not in the returned object, so `toEqual` reports a missing key and the two new cases get `undefined`.

- [ ] **Step 3: Implement**

In `lib/event-module.ts` add the import, the interface field, and the resolution:

```ts
import { categoryNumeral } from "./category-numerals";
import { coverPhoto } from "./covers";
import { formatModuleDate } from "./format";
import { priceLabel } from "./price-label";
import type { LiaEvent } from "./types";

export interface EventModuleData {
  numeral: string;
  category: string;
  title: string;
  venue: string;
  date: string;
  price: string;
  href: string;
  /** Resolved cover URL, or undefined → the module renders its numeral plate. */
  cover?: string;
}
```

and inside the returned object, after `href`:

```ts
    cover: coverPhoto(event),
```

- [ ] **Step 4: Run the tests**

Run: `npx vitest run lib/__tests__/event-module.test.ts`
Expected: PASS (6 tests)

Run: `npx vitest run`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/event-module.ts lib/__tests__/event-module.test.ts
git commit -m "feat(frontend): resolve event cover in the module adapter"
```

---

### Task 5: Feed and search grids — equal rows and honest skeletons

Both grids size their rows independently today, so with a cover band the titles in a row stop sitting on a shared line. Both also use a `140px` skeleton that no longer matches a cell.

**Files:**
- Modify: `components/DiscoveryFeed.tsx` (grid at line ~241, skeleton at line ~229)
- Modify: `components/DiscoverBrowse.tsx` (grid at line ~177, skeleton at line ~169)

**Interfaces:**
- Consumes: `eventToModuleProps` returning `cover` (Task 4) — both files already spread its result, so no prop wiring is needed here
- Produces: nothing

- [ ] **Step 1: Add `auto-rows-fr` to the feed results grid**

In `components/DiscoveryFeed.tsx`, the results grid:

```tsx
        <div className="grid auto-rows-fr grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
```

- [ ] **Step 2: Fix the feed skeleton height**

Same file, the pending branch:

```tsx
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1">
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} className="h-[290px] max-sm:h-[66px]" />
          ))}
        </div>
```

`290px ≈ 181px` band at a `453px` column plus `~110px` of content; `66px` is the mobile row with its `44px` thumbnail and padding.

- [ ] **Step 3: Apply both changes to search**

In `components/DiscoverBrowse.tsx`, the results grid gains `auto-rows-fr` in the same position:

```tsx
        <div className="grid auto-rows-fr grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
```

and the loading branch:

```tsx
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} className="h-[290px] max-sm:h-[66px]" />
          ))}
        </div>
```

- [ ] **Step 4: Verify nothing regressed**

Run: `npx vitest run`
Expected: PASS.

Run: `npx tsc --noEmit`
Expected: no errors.

Run: `npx eslint components/DiscoveryFeed.tsx components/DiscoverBrowse.tsx`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add components/DiscoveryFeed.tsx components/DiscoverBrowse.tsx
git commit -m "feat(frontend): equal-height cover rows and matching skeletons in feed and search"
```

---

### Task 6: Make «Превью в ленте» tell the truth

The wizard rail builds `EventModule` props by hand and never passed a cover, so an organizer who has just uploaded one would not see it in a block labelled «Превью в ленте».

**Files:**
- Modify: `components/CreateEventForm.tsx:863` (the `<EventModule …>` in the right rail)
- Modify: `app/design-preview/page.tsx` (the `EventModule` gallery section, lines ~59-76)

**Interfaces:**
- Consumes: `EventModuleProps.cover` (Task 3); `coverPreviewUrl` state already exists in `CreateEventForm` at line ~270 and is set by `handleCoverFile`
- Produces: nothing

- [ ] **Step 1: Pass the uploaded cover into the preview**

In `components/CreateEventForm.tsx`, add the prop to the rail's module:

```tsx
              <EventModule
                numeral={firstCat ? categoryNumeral(firstCat.slug, categories) : "—"}
                category={firstCat?.label ?? "—"}
                title={title?.trim() || "Без названия"}
                venue={previewVenue}
                date={previewDate}
                price={previewPrice}
                cover={coverPreviewUrl}
                href={mode === "edit" && eventId ? `/events/${eventId}` : "#"}
              />
```

- [ ] **Step 2: Show both cover states in the primitive gallery**

In `app/design-preview/page.tsx`, the `EventModule` section renders three samples. Give the first two covers and leave the third bare, so a reviewer sees a photo card, a photo card with a match reason, and the plate side by side. Add the `cover` prop to the first two only:

```tsx
          <EventModule
            numeral={categoryNumeral("festival", CATS)}
            category="Фестивали" title="Летний фестиваль медиаискусства"
            venue="Музей «Гараж»" date="12.07 · 16:00" price={priceLabel(0)} href="/"
            cover="/covers/festival.jpg"
          />
          <EventModule
            numeral={categoryNumeral("lecture", CATS)}
            category="Лекции" title="Разговор о новой вещественности"
            venue="ГМИИ им. Пушкина" date="15.07 · 19:00" price={priceLabel(800)} href="/"
            matchReason="тихое, вечером, вдвоём"
            cover="/covers/lecture.jpg"
          />
```

Leave the third (`mediation`) sample exactly as it is — it is the plate case.

- [ ] **Step 3: Verify**

Run: `npx vitest run`
Expected: PASS.

Run: `npx tsc --noEmit`
Expected: no errors — this catches a typo in `coverPreviewUrl`.

Run: `npx eslint components/CreateEventForm.tsx app/design-preview/page.tsx`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add components/CreateEventForm.tsx app/design-preview/page.tsx
git commit -m "feat(frontend): show the uploaded cover in the feed preview and design gallery"
```

---

### Task 7: Verify in the running app

Static tests cannot show a crop, a hairline or a row of plates. Look at it.

**Files:** none — verification only.

**Interfaces:**
- Consumes: everything from Tasks 2–6
- Produces: the evidence quoted in the final report

- [ ] **Step 1: Build**

Run: `npx next build`
Expected: build succeeds. This is the gate that catches a `next/image` misuse the tests miss, e.g. a remote host absent from `remotePatterns`.

- [ ] **Step 2: Start the dev server against the live API**

Run: `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru npx next dev -p 3010`

The live API has 9 published events — 7 seeded (curated photos), 1 with an uploaded cover, and 1 with none — so a single feed shows both the photo and the plate states.

- [ ] **Step 3: Check the four surfaces**

At `http://localhost:3010`:

1. `/` — desktop ≥1024px: bands are `5:2`, all titles in a row sit on one line, exactly one cell shows a numeral plate. Below `640px`: `44×44` thumbnails with a 1px ink border, the caption reads `NN · площадка · дата`.
2. `/search` — run a query; the cards match the feed and «Совпало: …» still sits on its hairline.
3. `/events/new` (as an organizer) — upload a cover on the Основное step; it appears in the rail preview cropped to `5:2`.
4. `/design-preview` — the `EventModule` section shows a photo card and a plate card side by side.

- [ ] **Step 4: Confirm the hidden layout is not downloading real images**

In DevTools Network, filter `_next/image`. At a desktop width every request should carry `w=1080` or similar **and** there should be a matching set of `w=32` requests (the hidden mobile thumbnails). Below `640px` the relationship inverts. Seeing two full-size requests per card means a `sizes` string picked up a `vw` unit — re-read spec §6.

- [ ] **Step 5: Report**

Write down, with the actual numbers observed: the `_next/image` width requested per card at each breakpoint, and a note on whether the plate reads as intentional next to real photos. If the plate looks wrong at the real column width, **stop and report** rather than redesigning it — the plate treatment was approved from an ASCII mockup and is the one thing here most likely to need a second look on real pixels.

- [ ] **Step 6: Stop the dev server**

---

### Task 8: Bring the design authority back in sync

**Files:**
- Modify: `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` (U1 section around line 199, Assets section around line 478)

**Interfaces:**
- Consumes: the four recorded deviations from spec §8
- Produces: nothing

- [ ] **Step 1: Update the U1 data tuple and add the cover rules**

In the U1 section, change the `**Data:**` line to:

```markdown
**Data:** `{ i, cat, t, v, d, p, img }` — index numeral, category, title, venue, date, price
string, cover URL.
**Covers:** desktop cells open on a full-bleed cover band, `aspect-ratio 5/2`, hairline
`border-bottom`, content padding `11px 14px` below it. Mobile rows carry a `44×44` cover in a
`1px` ink box as grid column one (`44px 1fr auto`, `row-span 2`), and the numeral moves into the
caption (`NN · площадка · дата`). No photo → a blank-cell (`#eceae4`) plate: the category
numeral in mono `26px/700` with the category name in caption caps on the desktop band, and the
numeral alone in the mobile box. The mock's fixed `88px` band and its `«Обложка»` text
placeholder are superseded — see `docs/superpowers/specs/2026-07-30-u1-feed-covers-design.md`.
```

- [ ] **Step 2: Correct the Assets ratio rule**

Replace the `**Event covers**` bullet:

```markdown
- **Event covers** — real photographs from `https://presence.tarski.ru/covers/*.jpg`
  (`festival.jpg`, `mediation.jpg` used in the mocks). Sources are 3:2. Crops per surface:
  U1 feed `5:2`, U2 detail hero `3:1` (`3:2` below `md`), moderation record `120px` thumb.
  Covers are always `object-fit: cover`, never letterboxed.
```

- [ ] **Step 3: Verify the mock still opens**

Run from the repo root: `cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099`
Open `http://localhost:8099/Presence%20Swiss%20Grid%20-%20Full%20System.dc.html`, confirm the U1 grid renders with covers, then stop the server. This only checks the mock is intact — the README edit cannot break it, but a stale checkout would show up here.

- [ ] **Step 4: Commit**

```bash
git add docs/Redesign/5/design_handoff_presence_swiss_grid/README.md
git commit -m "docs(design): record U1 cover band and per-surface crop ratios"
```

---

## Done when

- `npx vitest run` green, `npx tsc --noEmit` clean, `npx eslint` clean on every touched file.
- `npx next build` succeeds.
- Task 7 verification recorded, including the observed `_next/image` widths per breakpoint.
- README U1 and Assets sections match the shipped ratios.
- Eight commits on a feature branch; `main` untouched, nothing pushed.

## Not in this plan

Type-scale consolidation (347 `text-[Npx]` occurrences / 29 distinct values), AI «Подбор», a formal Lighthouse/CLS budget, the A1 «Пользователей» tile, and calendar/map surfaces — spec §9.

## Deployment

Frontend-only, images-only; prod LIA DB stays at 020. Build `linux/amd64` on the Mac with **both** `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` and `NEXT_PUBLIC_YANDEX_MAPS_KEY`, then `docker save | gzip | ssh | docker load`. Tag a rollback image before recreating. Deployment happens only on explicit request — it is not part of this plan.
