# Clickable Dashboard Tiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: EXECUTED & DEPLOYED LIVE 2026-08-12** (all tasks complete, final review clean; runbook `docs/superpowers/runbooks/2026-08-12-clickable-tiles-deploy.md`)

**Goal:** Stat tiles on the organizer cabinet, admin overview, and /me profile become links with a visible affordance («КАПШЕН →» + hover-invert), enabled by a `?status=` URL filter on «Мои события».

**Architecture:** One primitive change (`Cell` gets optional `href` rendering a next/link with arrow suffix), one enabler (`MyEventsBrowse` reads/writes `?status=` — exact mirror of `MeProfile`'s `?tab=` pattern), then pure wiring on three dashboards. Ad-hoc admin tiles (dark theme, not Cell-based) get the same affordance inline with a `group`/`group-hover` pattern.

**Tech Stack:** Next.js App Router (next/link, useSearchParams + Suspense), Tailwind utilities `swiss-focus`/`hover-invert` (defined in `frontend/app/globals.css:160-176`), vitest + `renderToStaticMarkup` for component tests.

**Spec:** `docs/superpowers/specs/2026-08-12-clickable-dashboard-tiles-design.md` — read it first.

## Global Constraints

- The affordance language is absolute: **every tile that navigates shows ` →` appended to its caption; every tile without a target has no arrow and stays a plain div.** No arrow on list-preview rows (they are rows, not KPI tiles — hover + cursor only).
- Inert tiles (exact list): organizer «Всего записей», admin «Событий всего». Everything else in the spec's tables navigates.
- Filter param name is `status`, values `published | pending_review | draft | cancelled`; `all` is represented by NO param (router.replace to bare `/events/mine`); invalid/absent value → `all`.
- All frontend commands run from `frontend/`: `npx vitest run`, `npx tsc --noEmit` (11 pre-existing tsc errors live in 3 unrelated test files — only NEW errors matter).
- Commit after every task; append `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` to every commit message.

---

### Task 1: `Cell` gets an optional `href`

**Files:**
- Modify: `frontend/components/ui/Cell.tsx`
- Test: `frontend/components/__tests__/cell-href.test.tsx` (create)

**Interfaces:**
- Produces: `CellProps` gains `href?: string`. With `href`, `Cell` renders a `next/link` `<Link>` carrying the same layout classes plus `swiss-focus hover-invert cursor-pointer`, and the caption renders as `{caption} →`. Without `href`, output is byte-identical to today (div, no arrow). Tasks 3 and 5 pass `href` to existing `Cell` call sites.

- [x] **Step 1: Write the failing test**

```tsx
// frontend/components/__tests__/cell-href.test.tsx
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Cell } from "@/components/ui/Cell";

describe("Cell href affordance", () => {
  it("renders a link with arrow-suffixed caption when href is given", () => {
    const html = renderToStaticMarkup(
      <Cell caption="Черновики" value="01" mono href="/events/mine?status=draft" />,
    );
    expect(html).toContain("<a ");
    expect(html).toContain('href="/events/mine?status=draft"');
    expect(html).toContain("Черновики →");
    expect(html).toContain("hover-invert");
    expect(html).toContain("swiss-focus");
    expect(html).toContain("cursor-pointer");
  });

  it("stays an inert div without arrow when href is absent", () => {
    const html = renderToStaticMarkup(<Cell caption="Всего записей" value="01" mono />);
    expect(html).not.toContain("<a ");
    expect(html).not.toContain("→");
    expect(html).not.toContain("hover-invert");
  });

  it("keeps the invert styling on a linked cell", () => {
    const html = renderToStaticMarkup(
      <Cell caption="На модерации" value="00" invert href="/events/mine?status=pending_review" />,
    );
    expect(html).toContain("bg-on-surface");
    expect(html).toContain("На модерации →");
  });
});
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run components/__tests__/cell-href.test.tsx`
Expected: FAIL — `href` prop unknown / no `<a>` rendered.

- [x] **Step 3: Implement**

Rewrite `frontend/components/ui/Cell.tsx`'s `Cell` (keep `CellStrip` untouched):

```tsx
import Link from "next/link";
import { cn } from "@/lib/cn";
import type { ReactNode } from "react";

export interface CellProps {
  caption: ReactNode;
  value: ReactNode;
  /** Numeric value → JetBrains Mono (handoff: ALL numbers in mono). */
  mono?: boolean;
  /** Roomy 16/20 padding instead of dense 10/14. */
  roomy?: boolean;
  /** Ink-filled emphasis cell (e.g. O1 «На модерации»). */
  invert?: boolean;
  /** Navigation target. Present ⇒ the cell renders as a link with a « →»
   * caption suffix and hover inversion — the product-wide affordance for
   * "this tile is a button" (spec 2026-08-12-clickable-dashboard-tiles). */
  href?: string;
  valueClassName?: string;
  className?: string;
}

export function Cell({ caption, value, mono, roomy, invert, href, valueClassName, className }: CellProps) {
  const inner = (
    <>
      <span className={cn("cap", invert && "text-text-dim-dark-2")}>
        {caption}
        {href ? " →" : null}
      </span>
      <span className={cn("text-[12px] font-bold leading-[1.25]", mono && "font-mono", valueClassName)}>
        {value}
      </span>
    </>
  );
  const shared = cn(
    "flex min-w-0 flex-col gap-[4px]",
    roomy ? "px-[20px] py-[16px]" : "px-[14px] py-[10px]",
    invert && "bg-on-surface text-surface",
    className,
  );
  if (href) {
    return (
      <Link href={href} className={cn(shared, "swiss-focus hover-invert cursor-pointer")}>
        {inner}
      </Link>
    );
  }
  return <div className={shared}>{inner}</div>;
}
```

Note: `hover-invert` (globals.css:169) swaps to the themed `--hover-fill`/`--hover-text` vars, which already handle the inverted cell correctly in both contexts — no bespoke hover for `invert` cells.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run components/__tests__/cell-href.test.tsx && npx vitest run && npx tsc --noEmit`
Expected: new tests PASS; full suite still green; no new tsc errors.

- [x] **Step 5: Commit**

```bash
git add frontend/components/ui/Cell.tsx frontend/components/__tests__/cell-href.test.tsx
git commit -m "feat(ui): Cell href — arrow-caption link affordance for dashboard tiles"
```

---

### Task 2: `?status=` URL filter on «Мои события»

**Files:**
- Modify: `frontend/components/MyEventsBrowse.tsx` (filter state ~line 260, chips ~line 330)
- Modify: `frontend/app/events/mine/page.tsx` (Suspense boundary)
- Test: `frontend/components/__tests__/my-events-status-param.test.ts` (create)

**Interfaces:**
- Consumes: nothing new.
- Produces: `/events/mine?status=<Filter>` initializes the page filter; chip clicks keep the URL in sync (`all` → bare `/events/mine`). Exported helper `parseStatusParam(raw: string | null): Filter` (from `MyEventsBrowse.tsx`) — pure, testable. Task 3 links to these URLs.

- [x] **Step 1: Write the failing test**

```ts
// frontend/components/__tests__/my-events-status-param.test.ts
import { describe, expect, it } from "vitest";

import { parseStatusParam } from "@/components/MyEventsBrowse";

describe("parseStatusParam", () => {
  it("accepts every real filter value", () => {
    expect(parseStatusParam("published")).toBe("published");
    expect(parseStatusParam("pending_review")).toBe("pending_review");
    expect(parseStatusParam("draft")).toBe("draft");
    expect(parseStatusParam("cancelled")).toBe("cancelled");
  });
  it("falls back to all for absent or garbage values", () => {
    expect(parseStatusParam(null)).toBe("all");
    expect(parseStatusParam("")).toBe("all");
    expect(parseStatusParam("hacker")).toBe("all");
    expect(parseStatusParam("ALL")).toBe("all");
  });
});
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run components/__tests__/my-events-status-param.test.ts`
Expected: FAIL — `parseStatusParam` is not exported.

- [x] **Step 3: Implement**

In `MyEventsBrowse.tsx` (read the file first; `Filter` type and `FILTERS` are near line 27):

1. Export the parser next to the `Filter` type:

```ts
/** Maps the ?status= query value to a Filter; anything unknown → "all". */
export function parseStatusParam(raw: string | null): Filter {
  const valid: Filter[] = ["all", "published", "pending_review", "draft", "cancelled"];
  return valid.includes(raw as Filter) ? (raw as Filter) : "all";
}
```

2. In the component: add `useSearchParams` (import from `next/navigation`, alongside the existing `useRouter`), initialize from it, and sync chip clicks to the URL. Mirror `MeProfile.tsx:136-137` exactly:

```ts
const searchParams = useSearchParams();
const [filter, setFilter] = useState<Filter>(() => parseStatusParam(searchParams.get("status")));
```

and in the chip `onClick` (~line 337), replace `onClick={() => setFilter(f.key)}` with:

```ts
onClick={() => {
  setFilter(f.key);
  router.replace(f.key === "all" ? "/events/mine" : `/events/mine?status=${f.key}`);
}}
```

3. `app/events/mine/page.tsx`: wrap `<MyEventsBrowse />` in `<Suspense fallback={null}>` (import `Suspense` from `react`) — `useSearchParams` requires it in the App Router; mirror the comment style of `app/me/page.tsx`.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run && npx tsc --noEmit && npm run build`
Expected: all green (the `npm run build` here is load-bearing: it catches a missing Suspense boundary, which vitest cannot).

- [x] **Step 5: Commit**

```bash
git add frontend/components/MyEventsBrowse.tsx frontend/app/events/mine/page.tsx frontend/components/__tests__/my-events-status-param.test.ts
git commit -m "feat(my-events): ?status= URL filter — drafts get a shareable address"
```

---

### Task 3: Organizer cabinet tiles → links

**Files:**
- Modify: `frontend/components/OrganizerHub.tsx` (desktop strip ~line 139, mobile strip ~line 169)

**Interfaces:**
- Consumes: `Cell.href` (Task 1), `/events/mine?status=…` (Task 2).

- [x] **Step 1: Wire the tiles**

Desktop `CellStrip cols={4}`: add `href` to three cells — «Опубликовано» → `/events/mine?status=published`, «На модерации» → `/events/mine?status=pending_review`, «Черновики» → `/events/mine?status=draft`. «Всего записей» gets NO href (stays inert, no arrow). Mobile `CellStrip cols={3}`: same three hrefs on «Опубл.» / «Модер.» / «Черн.». No other changes.

- [x] **Step 2: Verify**

Run: `cd frontend && npx vitest run && npx tsc --noEmit`
Expected: green. Then start the dev server (`npx next dev -p 13000`) against any backend (mock is fine) and eyeball `/organizer`: three tiles carry ` →` and invert on hover; «Всего записей» doesn't. Stop the server.

- [x] **Step 3: Commit**

```bash
git add frontend/components/OrganizerHub.tsx
git commit -m "feat(organizer): cabinet tiles link to filtered Мои события"
```

---

### Task 4: Admin overview tiles and preview rows → links

**Files:**
- Modify: `frontend/components/AdminOverview.tsx` (desktop tiles ~lines 100-125, queue rows ~135-155, org rows ~171-187, mobile duty tiles ~205-220)

**Interfaces:**
- Consumes: nothing from earlier tasks (these are ad-hoc dark-theme divs, not `Cell`). Targets: `/admin/moderation/events`, `/admin/organizers`, `/admin/users`, `/admin/organizers?filter=pending`, `/admin/organizers/{id}`.

- [x] **Step 1: Read the component fully**, note the exact class strings on each tile (dark theme: `border-paper`, `bg-signal`, `cap`, mono values).

- [x] **Step 2: Convert the three navigable KPI tiles to links**

Pattern for a normal dark tile (arrow in cap + paper-inversion hover via `group`):

```tsx
<Link
  href="/admin/organizers"
  className="group swiss-focus cursor-pointer border-r border-paper p-[14px] transition-colors duration-[120ms] hover:bg-paper"
>
  <div className="cap group-hover:text-ink">Организаторов →</div>
  <div className="font-mono text-[26px] font-bold leading-none group-hover:text-ink">
    {tileCount(overview.organizers_total)}
  </div>
</Link>
```

- «Ждут модерации» (bg-signal tile) → `/admin/moderation/events`; keep `bg-signal`, hover to paper with red value: `hover:bg-paper`, cap `group-hover:text-signal`, value `text-white group-hover:text-signal`; caption becomes `Ждут модерации →`.
- «Пользователей» → `/admin/users` (same normal pattern, it has no `border-r`).
- «Событий всего» stays a div, no arrow.

- [x] **Step 3: Convert the preview rows**

- «Очередь модерации» rows (~135-155): wrap each row's existing content in a `<Link href="/admin/moderation/events" className="… swiss-focus cursor-pointer transition-colors hover:bg-paper group">` preserving the row's current layout classes; add `group-hover:text-ink` on text spans that carry explicit light colors. NO arrow (rows, not tiles).
- «Заявки на верификацию» rows (~171-187): same, but each row links to `/admin/organizers/${org.id}` (read the row's data shape for the actual id field — the detail route `app/admin/organizers/[id]` exists).

- [x] **Step 4: Mobile duty tiles** (~205-220): «Модерация» → `/admin/moderation/events`, «Верификация» → `/admin/organizers?filter=pending`, both with caption ` →` and the same group-hover pattern.

- [x] **Step 5: Verify**

Run: `cd frontend && npx vitest run && npx tsc --noEmit`
Expected: green, no new errors. (Visual pass happens in Task 5.)

- [x] **Step 6: Commit**

```bash
git add frontend/components/AdminOverview.tsx
git commit -m "feat(admin): overview tiles and preview rows navigate to their sections"
```

---

### Task 5: /me profile tiles → tab links + full verification

**Files:**
- Modify: `frontend/components/MeProfile.tsx` (desktop cells ~line 218-227, mobile strip ~line 229-231)

**Interfaces:**
- Consumes: `Cell.href` (Task 1); existing tab URLs `/me?tab=past|follows|applications`.

- [x] **Step 1: Wire the tiles**

Desktop: «Посещено» → `href="/me?tab=past"`, «Подписки» → `href="/me?tab=follows"`. Mobile strip: «Посещено» → `/me?tab=past`, «Заявки» → `/me?tab=applications`, «Подписки» → `/me?tab=follows`. No other changes.

- [x] **Step 2: Full verification**

Run: `cd frontend && npx vitest run && npx tsc --noEmit && npm run build`
Expected: all green, no new tsc errors, build succeeds.

- [x] **Step 3: Visual pass**

Start dev server (`npx next dev -p 13000`), eyeball: `/organizer` (three arrows + hover-invert, «Всего записей» inert), `/me` (tiles navigate to tabs), `/events/mine?status=draft` opens with «Черновики» chip active. Admin pages need a staff login — verify AdminOverview markup by the component tests/tsc only and flag the live admin check for the deploy QA. Stop the server.

- [x] **Step 4: Commit**

```bash
git add frontend/components/MeProfile.tsx
git commit -m "feat(me): profile stat tiles navigate to their tabs"
```
