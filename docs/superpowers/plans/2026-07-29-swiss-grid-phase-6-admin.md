# Swiss Grid Phase 6 — Admin inverted suite (A1–A3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the three admin ink screens — **A1 Обзор**, **A2 Модерация событий**, **A3 Организаторы и верификация** — to Swiss Grid fidelity under `data-surface="ink"`, on existing routes and admin APIs, without reopening locked P1–P5 deviations.

**Architecture:** Admin layout owns auth gate + ink surface + `AppHeader admin` + `ADMIN_NAV`. Thin page shells mount client bodies (`AdminOverview`, `AdminModeration`, `AdminOrganizers`). New logic lives in pure Vitest-covered helpers (`admin-relative`, `admin-id`, `admin-reject-reasons`, `admin-test-heuristic`, `admin-org-status`, `admin-queue`). Screens reuse Phase 1 primitives (`Cell`/`CellStrip`, `Chip`, `Button` inverted/destructive/dark-ghost, `StatusChip`, `Skeleton`, `EmptyState`). Ship against existing admin APIs (overview, moderation list + takedown/reinstate, organizer list/search/verify/reject/revoke); backend gaps become deliberate stubs (same class as P3–P5).

**Tech Stack:** Next.js 16 App Router / React 19 / TypeScript, Tailwind v4 (Swiss Grid tokens + `data-surface="ink"`), Vitest 4 (node-only pure helpers), existing Go admin endpoints. **No new backend in this phase.** Keep `useEffect`+tick fetch pattern on admin pages (master plan risk #10 — TanStack Query migration out of scope).

**Spec:** `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` § Part 3 Admin + Admin inverted rules. Pixel: `Presence Swiss Grid - Full System.dc.html` badges `A1 · Обзор` (~846), `A2 · Модерация событий` (~897), `A3 · Организаторы и верификация` (~952). **A4 (~985) is Phase 7 — out of scope.** Handoff: `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-5-HANDOFF.md`. Master: `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` Phase 6 + Decision checkpoint 1 (admin ink = surface, not theme).

**Prerequisite:** `main` tip includes Phase 5 merge (`8dce925` or later). Verify before starting:

```bash
git log --oneline -5
# must include Phase 5 organizer work (e.g. 8dce925 fidelity pass or earlier O1–O5 commits)
test -f frontend/components/ui/AppHeader.tsx
grep -q 'ADMIN_NAV' frontend/components/ui/AppHeader.tsx
grep -q 'data-surface="ink"' frontend/app/globals.css
grep -q 'inverted' frontend/components/ui/Button.tsx
grep -q 'dark-active' frontend/components/ui/Chip.tsx
test -f frontend/lib/relative-time.ts
```

**Branch / worktree (at execution time only):** `redesign/swiss-grid-p6` via `superpowers:using-git-worktrees`, branched from `main` @ ≥ Phase 5 tip. Working directory for all `pnpm` commands: `frontend/`. **Do not implement until this plan is reviewed.**

**Screen order:** A1 → A2 → A3 → secondary stubs → deploy+verify (master plan).

---

## Global Constraints

Everything from the master plan (`2026-07-28-swiss-grid-redesign-master-plan.md`) applies. This phase trips over:

- **Admin = `data-surface="ink"`** on the admin layout root: `#111` bg, `#F2F0EC` text, structural rules `#F2F0EC` (`border-on-surface` / `border-paper`), inner rules `#3A3733` (`border-rule-inner`), table heads `#1C1A18` (`bg-surface-head` / `bg-admin-head`), secondary `#8A857C`, body `#CFCABF` (`text-text-dim` / `text-text-dim-dark`).
- **Zero border radius, zero shadows.** 1px hairlines only.
- **Categories are numerals** via existing helpers when a category name appears on the A2 record (not colour chips).
- **All numbers in JetBrains Mono** (`font-mono`): tile heroes, queue ages, IDs, seat/price facts, complaint counts. Unknown → `—`.
- **`signal` red only for needs-attention:** A1 «Ждут модерации» tile fill + `#FFD9D6` caption; A1/A2/A3 test-data titles/names; A2 `ОТКЛОНИТЬ`; A3 signal filter chip + complaint count when > 0; A3 «На проверке» / «Тестовый» chips.
- Uppercase tracking preserved (`.cap` 0.13em, chips 0.12em, buttons 0.07em, nav 0.14em).
- Hover on admin fills `#1C1A18` (`hover-invert` under ink surface); focus `swiss-focus` (paper outline on ink); transitions 120ms linear on background/color. Loading = `Skeleton` — **no spinners**, no «Загрузка…» prose.
- Empty / gated / error = `EmptyState` patterns adapted to ink (layout already role-gates).
- **Tap targets ≥44px on touch.** Dense mock paddings grow; type sizes stay.
- Content max 1360px; Moscow formatters; React #418.
- UI copy Russian; code/commits English.
- Fonts stay **Golos Text / Manrope / JetBrains Mono** (locked P1). Manrope has no 900 — titles use `font-black` on `--font-ui`.
- **A2–A3 desktop-only** (min-width notice below 900px). **A1 also has mobile duty mode.**
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Deliberate deviations for Phase 6 (pre-decided — do not reopen P1–P5; do not "fix" these without product sign-off)

1. **Routes stay mapped** (master plan): A1 `/admin`, A2 `/admin/moderation/events`, A3 `/admin/organizers`. Handoff `/admin/moderation` is conceptual — keep the existing events path. Absorb verify/reject UI from `/admin/moderation/organizers` into A3; that old page becomes a **redirect** to `/admin/organizers?filter=pending`.
2. **No pending_review event moderation API.** Backend `GET /admin/moderation/events` only accepts `published` | `rejected`; writes are `takedown` / `reinstate`. A2 is a **visual conveyor over that API**, not a pre-publish approval desk:
   - Chip **«Ждут»** → `listModerationEvents("published")` (staff review / takedown queue).
   - Chip **«Все»** → published ∪ rejected (client merge, rejected after published; de-dupe by id).
   - **ОТКЛОНИТЬ** → `takedownEvent(id, concatenateReasons(selectedChips))` — ≥1 chip required.
   - **ОДОБРИТЬ** → if selected status is `rejected`, `reinstateEvent(id)`; if `published`, no write (already live) — still **advance to next** queue item.
   - **НА ДОРАБОТКУ** → `takedownEvent(id, "На доработку: " + concatenateReasons(...))` with ≥1 chip (same reason gate as reject). No separate revision endpoint.
3. **A1 tile «Ждут модерации»** has no overview field. Derive count as `listModerationEvents("published").length` (same operational queue as A2 «Ждут»). Do **not** map `events_removed` into that tile.
4. **A1 tile «Организаторов»** — overview only returns `organizers_pending`, not total. Show **`—`** for the total tile value (do not invent). Verification queue panel still lists real `listModerationOrganizers("pending")` with count in the panel caption.
5. **A1 tile «Пользователей»** — no users API (A4 / Phase 7). Show **`—`**.
6. **A1 «Сигналы»** — no test-data hygiene API. Derive: if `complaints_open > 0`, copy `N событий с жалобами` (link to `/admin/complaints`); else if any published-queue title matches `isLikelyTestContent`, copy `N события с тестовыми данными`; else muted stub `.cap` `Сигналов нет`. Never invent rows.
7. **Queue age** — `AdminEvent` has `starts_at`, not submitted_at. Use `starts_at` with `formatRelativeCompactRu` as the age column (honest enough for v1). Full event detail for the A2 record pane comes from `fetchEventWithAuth` (cover, description, price, capacity, format).
8. **Admin short IDs** — mock `EV-1042`. Display `adminShortId(uuid)` → `EV-` + first 4 hex chars of UUID (uppercase). Full UUID stays in `title` / `data-id` for a11y.
9. **Organizer email / event count / complaint count** — not on `AdminOrganizer` JSON. Caption under name = `website_url` if present, else `—`. Событий / Жалоб columns = `—` (mono). Do not fetch every org’s events list in A3 (N+1).
10. **Test organizers / events** — client heuristic `isLikelyTestContent(name|title|website)` (`/QA|тест|test|\.test\b|bla bla/i`). Match → red title/name + optional «Тестовый» chip override when status would otherwise be verified. **УДАЛИТЬ** primary when heuristic hits: no delete API → open ··· panel with **Отозвать** (`revokeOrganizer`) as the destructive stand-in; do not fake delete.
11. **A3 filter «С жалобами»** — no per-org complaint count. Chip stays in the UI but selecting it shows `EmptyState` caption `Фильтр по жалобам появится позже` (or filters to `[]` with that empty). Counts on other chips = lengths of the corresponding `listModerationOrganizers` / search results.
12. **ADMIN_NAV «Пользователи»** stays for header fidelity. `/admin/users` is a **Phase-7 stub page** (`EmptyState`: `Раздел появится позже`) — **not** A4 registry/hygiene.
13. **`/admin/settings` and `/admin/complaints`** stay reachable by URL (and A1 signal link for complaints). Restyle to sit on ink surface (no glass/radius); not in the A1–A3 fidelity contract.
14. **Desktop-only gate (A2/A3):** below `max-[899px]` show a full-bleed notice `Админ-инструменты — с экрана от 900px` + link `К обзору` → `/admin`. A1 duty mode renders instead of that notice on `/admin`.
15. **Layout loading:** replace «Загрузка…» with ink `Skeleton` cells; auth redirect behaviour unchanged.
16. **AppHeader admin mobile:** mock A1 mobile wordmark is `ADMIN`. Keep `PRESENCE / ADMIN` on `sm+`; on `max-sm` when `admin`, show `ADMIN` only (small AppHeader tweak). `mobileCaption` per screen (`ОБЗОР` / `МОДЕРАЦИЯ` / `ОРГАНИЗАТОРЫ`).
17. **No TanStack Query migration** for admin fetches (master plan risk #10).

**Locked from earlier phases (do not reopen):** favorites/♡ deferred; home `/`; U4 personal calendar; Yandex not OSM; fonts Golos/Manrope/JB Mono; U3 non-AI `/search`; O1 activity stub; O4 bulk = sequential decide; O5 no telegram field; etc.

**Out of scope:** **A4** Users + content hygiene (Phase 7 + backend). Do not build registry table or `СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ`.

---

## Design fidelity contract (HTML badges checked 2026-07-29)

`Presence Swiss Grid - Full System.dc.html` is pixel authority; README § A1–A3 is behaviour authority. Serve locally:

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
# open Presence Swiss Grid - Full System.dc.html — search badges «A1» «A2» «A3»
```

| Screen | Desktop (key) | Mobile (key) |
|---|---|---|
| **A1** `:846-895` | Ink frame; header `PRESENCE / ADMIN` + nav (Обзор active, paper underline); **4-tile** strip `repeat(4,1fr)`, pad 14px, cap `#8A857C`, nums **26px mono**; **Ждут модерации** tile `background:#E2231A`, cap `#FFD9D6`, num white; body `1fr 1fr`: left queue (cap `Очередь модерации`, rows 11.5px/700 + mono age `#8A857C`, inner `#3A3733`), paper CTA `ОТКРЫТЬ ОЧЕРЕДЬ`; right `Заявки на верификацию · N` + **Сигналы** footer red 11px/700 | **Duty mode:** wordmark `ADMIN` + cap `ОБЗОР`; **2 tiles** (Модерация red / Верификация), nums 20px; 3 queue rows `age · organizer` + title (test title red); paper CTA |
| **A2** `:897-950` | Working surface ~`250px 1fr`; left chips `Ждут · N` **dark-active** / `Все` **dark-muted**; rows mono id 9.5px + age, title 11.5px/700, org cap; **selected row paper fill + ink text**; test title `#E2231A`; right: cover 120px + meta (`EV-… · подано …`), title 20px, org·category cap; 4-cell fact strip (Дата/Цена/Мест/Формат) inner `#3A3733`; description `#CFCABF`; reason chips dark-muted (toggleable); action bar `ОДОБРИТЬ` inverted / `ОТКЛОНИТЬ` destructive / `НА ДОРАБОТКУ` dark-ghost | **Desktop-only notice** &lt;900px (no mobile composition in mock) |
| **A3** `:952-983` | Filter chips row + `Поиск ⌕` right; table head `44px 1fr 110px 90px 90px 170px` on `#1C1A18`; rows inner `#3A3733`; mono id; name 12px/700 (+ email/website cap); status chip; mono counts; actions paper primary + `···` dark-ghost; test name red | **Desktop-only notice** &lt;900px |

**Interaction contract (A2):** rejecting / «на доработку» requires ≥1 selected reason chip; on any decision (including approve/advance), select the next queue item without navigating away; if queue empties, show ink `EmptyState`.

---

## File Structure

```
frontend/
  lib/admin-relative.ts                    CREATE — compact RU relative (2 ч / 1 д)
  lib/admin-id.ts                          CREATE — EV-XXXX short id
  lib/admin-reject-reasons.ts              CREATE — chip catalogue + concatenate
  lib/admin-test-heuristic.ts              CREATE — isLikelyTestContent
  lib/admin-org-status.ts                  CREATE — verification_status → RU label
  lib/admin-queue.ts                       CREATE — nextIndex after decision
  lib/__tests__/admin-relative.test.ts     CREATE
  lib/__tests__/admin-id.test.ts           CREATE
  lib/__tests__/admin-reject-reasons.test.ts CREATE
  lib/__tests__/admin-test-heuristic.test.ts CREATE
  lib/__tests__/admin-org-status.test.ts   CREATE
  lib/__tests__/admin-queue.test.ts        CREATE

  components/ui/AppHeader.tsx              MODIFY — admin max-sm wordmark ADMIN
  components/ui/TabBarGate.tsx             VERIFY — /admin already hidden (no change if grepped)

  app/admin/layout.tsx                     REWRITE — ink surface, AppHeader, auth gate, Skeleton
  components/AdminDesktopOnly.tsx          CREATE — &lt;900px notice for A2/A3
  components/AdminOverview.tsx             CREATE — A1 body (desktop + duty)
  app/admin/page.tsx                       REPLACE — thin shell → AdminOverview

  components/AdminModeration.tsx           CREATE — A2 queue/record body
  app/admin/moderation/events/page.tsx     REPLACE — thin shell + DesktopOnly

  components/AdminOrganizers.tsx           CREATE — A3 table + absorb verify/reject
  app/admin/organizers/page.tsx            REPLACE — thin shell + DesktopOnly
  app/admin/moderation/organizers/page.tsx REPLACE — redirect → /admin/organizers?filter=pending

  app/admin/users/page.tsx                 CREATE — Phase-7 stub EmptyState only
  app/admin/settings/page.tsx              MODIFY — ink-safe restyle (no glass)
  app/admin/complaints/page.tsx            MODIFY — ink-safe restyle (no glass)
```

**Not touched:** backend Go handlers, user/organizer public screens, P1–P5 locked deviations, A4 hygiene UI.

**Data reuse:**

| Need | Source |
|---|---|
| Overview counts | `getAdminOverview()` — `events_total`, `events_published`, `events_removed`, `organizers_pending?`, `complaints_open?` |
| Event queue | `listModerationEvents("published"\|"rejected")` |
| Takedown / reinstate | `takedownEvent(id, reason)`, `reinstateEvent(id)` |
| Event record detail | `fetchEventWithAuth(id)` → `LiaEvent` |
| Org pending / verified / … | `listModerationOrganizers(status)` |
| Org search | `searchOrganizers(q)` |
| Org detail / history | `getAdminOrganizer(id)` |
| Verify / reject / revoke / auto | `verifyOrganizer`, `rejectOrganizer`, `revokeOrganizer`, `setOrganizerAutoVerify` |
| Complaints signal | `complaints_open` from overview; page `listComplaints` |
| Compact age | new `formatRelativeCompactRu` |
| Pad counts | `padCount` from `lib/org-seats.ts` |
| Status chips | `statusChipVariant` + `adminOrgStatusLabel` |
| Category numeral / label | `categoryNumeral` / categories list if A2 shows category |

---

### Task 1: Pure helpers (TDD)

**Files:**
- Create: `frontend/lib/admin-relative.ts`
- Create: `frontend/lib/admin-id.ts`
- Create: `frontend/lib/admin-reject-reasons.ts`
- Create: `frontend/lib/admin-test-heuristic.ts`
- Create: `frontend/lib/admin-org-status.ts`
- Create: `frontend/lib/admin-queue.ts`
- Test: matching files under `frontend/lib/__tests__/`

**Interfaces:**
- Produces:

```ts
// admin-relative.ts
/** Compact RU relative for admin queues. Moscow day boundaries. */
export function formatRelativeCompactRu(iso: string, now?: Date): string;
// <60s → "сейчас"; <60m → "N мин"; <24h → "N ч"; <7d → "N д"; else formatShortDate(iso)

// admin-id.ts
/** Display id for moderation rows: EV- + first 4 hex of UUID (no dashes), uppercased. */
export function adminShortId(id: string): string;
// "a1b2c3d4-...." → "EV-A1B2"; empty/invalid → "EV-————"

// admin-reject-reasons.ts
export const REJECT_REASON_CHIPS = [
  "Тестовые данные",
  "Нет описания",
  "Обложка низкого качества",
  "Дубликат",
] as const;
export type RejectReason = (typeof REJECT_REASON_CHIPS)[number];

/** Join selected reasons with "; ". Empty → "". */
export function concatenateReasons(reasons: readonly string[]): string;

/** Prefix for НА ДОРАБОТКУ takedown body. */
export function revisionReason(reasons: readonly string[]): string;
// "На доработку: " + concatenateReasons(reasons)

// admin-test-heuristic.ts
export function isLikelyTestContent(...parts: Array<string | null | undefined>): boolean;
// true if any part matches /QA|тест|test|\.test\b|bla\s*bla/i

// admin-org-status.ts
export function adminOrgStatusLabel(
  status: string,
  opts?: { test?: boolean },
): string;
// pending → "На проверке"; verified → "Верифицирован"; rejected → "Отклонён";
// draft → "Черновик"; opts.test → "Тестовый" (overrides)

// admin-queue.ts
/** Index to select after acting on `current`. If last, clamp to newLast; if empty → -1. */
export function nextQueueIndex(current: number, lengthAfterRemoval: number): number;
// after removing current from list of oldLen, newLen = oldLen-1;
// prefer same index (former next); if current was last → newLen-1; if newLen===0 → -1
```

- Consumes: `formatShortDate` from `lib/format.ts` (relative only).

- [ ] **Step 1: Write the failing tests**

```ts
// lib/__tests__/admin-relative.test.ts
import { describe, expect, it } from "vitest";
import { formatRelativeCompactRu } from "../admin-relative";

describe("formatRelativeCompactRu", () => {
  const now = new Date("2026-07-12T15:00:00+03:00");
  it("formats hours compact", () => {
    expect(formatRelativeCompactRu("2026-07-12T13:00:00+03:00", now)).toBe("2 ч");
  });
  it("formats days compact", () => {
    expect(formatRelativeCompactRu("2026-07-10T15:00:00+03:00", now)).toBe("2 д");
  });
});
```

```ts
// lib/__tests__/admin-id.test.ts
import { describe, expect, it } from "vitest";
import { adminShortId } from "../admin-id";

describe("adminShortId", () => {
  it("prefixes EV- and uses first 4 hex", () => {
    expect(adminShortId("a1b2c3d4-5678-4012-8000-000000000001")).toBe("EV-A1B2");
  });
});
```

```ts
// lib/__tests__/admin-reject-reasons.test.ts
import { describe, expect, it } from "vitest";
import {
  REJECT_REASON_CHIPS,
  concatenateReasons,
  revisionReason,
} from "../admin-reject-reasons";

describe("reject reasons", () => {
  it("lists four handoff chips", () => {
    expect([...REJECT_REASON_CHIPS]).toEqual([
      "Тестовые данные",
      "Нет описания",
      "Обложка низкого качества",
      "Дубликат",
    ]);
  });
  it("concatenates with semicolon", () => {
    expect(concatenateReasons(["Тестовые данные", "Дубликат"])).toBe(
      "Тестовые данные; Дубликат",
    );
  });
  it("builds revision prefix", () => {
    expect(revisionReason(["Нет описания"])).toBe("На доработку: Нет описания");
  });
});
```

```ts
// lib/__tests__/admin-test-heuristic.test.ts
import { describe, expect, it } from "vitest";
import { isLikelyTestContent } from "../admin-test-heuristic";

describe("isLikelyTestContent", () => {
  it("flags QA / test / bla bla", () => {
    expect(isLikelyTestContent("QA Block8")).toBe(true);
    expect(isLikelyTestContent("bla bla meet")).toBe(true);
    expect(isLikelyTestContent("Летний фестиваль")).toBe(false);
  });
});
```

```ts
// lib/__tests__/admin-org-status.test.ts
import { describe, expect, it } from "vitest";
import { adminOrgStatusLabel } from "../admin-org-status";
import { statusChipVariant } from "../status-chip";

describe("adminOrgStatusLabel", () => {
  it("maps and feeds StatusChip", () => {
    expect(adminOrgStatusLabel("pending")).toBe("На проверке");
    expect(statusChipVariant(adminOrgStatusLabel("pending"))).toBe("signal");
    expect(statusChipVariant(adminOrgStatusLabel("verified"))).toBe("active");
    expect(adminOrgStatusLabel("verified", { test: true })).toBe("Тестовый");
  });
});
```

```ts
// lib/__tests__/admin-queue.test.ts
import { describe, expect, it } from "vitest";
import { nextQueueIndex } from "../admin-queue";

describe("nextQueueIndex", () => {
  it("stays on same index when a later item remains", () => {
    expect(nextQueueIndex(0, 2)).toBe(0); // removed index 0 from len 3 → len 2
  });
  it("steps back when acting on last", () => {
    expect(nextQueueIndex(2, 2)).toBe(1);
  });
  it("returns -1 when empty", () => {
    expect(nextQueueIndex(0, 0)).toBe(-1);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm vitest run lib/__tests__/admin-relative.test.ts lib/__tests__/admin-id.test.ts lib/__tests__/admin-reject-reasons.test.ts lib/__tests__/admin-test-heuristic.test.ts lib/__tests__/admin-org-status.test.ts lib/__tests__/admin-queue.test.ts`

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement minimal helpers**

Implement the six modules to match the interfaces and tests. Reuse `formatShortDate` for the compact-relative fallback branch.

- [ ] **Step 4: Run tests to verify they pass**

Run: same vitest command as Step 2. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/admin-*.ts frontend/lib/__tests__/admin-*.test.ts
git commit -m "$(cat <<'EOF'
feat(frontend): Swiss Grid admin pure helpers for A1–A3

EOF
)"
```

---

### Task 2: Admin layout — ink surface + AppHeader

**Files:**
- Modify: `frontend/components/ui/AppHeader.tsx` (admin mobile wordmark)
- Rewrite: `frontend/app/admin/layout.tsx`
- Create: `frontend/components/AdminDesktopOnly.tsx`

**Interfaces:**
- Consumes: `useAuth`, `AppHeader`, `ADMIN_NAV`, `Skeleton`
- Produces: ink-scoped admin chrome for all `/admin/*` children

**Layout contract:**

```tsx
// Pseudocode structure for admin/layout.tsx
if (!ready || !isAuthed || role !== "admin") { /* keep redirect gates; Skeleton while !roleResolved */ }
return (
  <div data-surface="ink" className="min-h-screen bg-surface text-on-surface">
    <div className="mx-auto max-w-[1360px]">
      <AppHeader admin nav={ADMIN_NAV} mobileCaption={/* from pathname map */} />
      <main>{children}</main>
    </div>
  </div>
);
```

Pathname → `mobileCaption`: `/admin` → `ОБЗОР`; starts with `/admin/moderation` → `МОДЕРАЦИЯ`; `/admin/organizers` → `ОРГАНИЗАТОРЫ`; `/admin/users` → `ПОЛЬЗОВАТЕЛИ`; `/admin/settings` → `НАСТРОЙКИ`; `/admin/complaints` → `ЖАЛОБЫ`.

`AdminDesktopOnly`: if `window` width &lt; 900 (use `matchMedia("(max-width: 899px)")`), render notice + `Link` to `/admin`; else render `children`.

AppHeader change: when `admin`, wordmark text is `PRESENCE / ADMIN` on `sm+` and `ADMIN` on `max-sm` (two spans with `hidden`/`sm:inline` as needed). Link href for admin wordmark → `/admin` (not `/`).

Remove glass nav, `rounded-card`, `bg-bg-grouped`, accent links.

- [ ] **Step 1: Implement AppHeader admin mobile wordmark + `/admin` href when admin**

- [ ] **Step 2: Rewrite layout** with `data-surface="ink"`, `ADMIN_NAV`, Skeleton gate, no glass

- [ ] **Step 3: Add `AdminDesktopOnly`**

- [ ] **Step 4: Browser smoke** — sign in as admin, `/admin` is ink, header `PRESENCE / ADMIN`, tab bar hidden

- [ ] **Step 5: Gate** `pnpm build && pnpm test && pnpm lint`

- [ ] **Step 6: Commit** — `feat(frontend): Swiss Grid admin ink layout and header`

---

### Task 3: A1 · Обзор

**Files:**
- Create: `frontend/components/AdminOverview.tsx`
- Replace: `frontend/app/admin/page.tsx`

**Interfaces:**
- Consumes: `getAdminOverview`, `listModerationEvents`, `listModerationOrganizers`, `padCount`, `formatRelativeCompactRu`, `isLikelyTestContent`, `Button` inverted, `Chip`, `Skeleton`, `EmptyState`, helpers from Task 1
- Produces: A1 desktop + mobile duty mode

**Data load (client, parallel):**

```ts
const [overview, published, pendingOrgs] = await Promise.all([
  getAdminOverview(),
  listModerationEvents("published"),
  listModerationOrganizers("pending"),
]);
```

**Tiles (desktop):**

| Caption | Value |
|---|---|
| Событий всего | `padCount(overview.events_total)` or raw if ≥100 (no pad past 2 only when &lt;100 — use `String(n)` for ≥100; users mock uses thin space `2 914` → for `—` skip) |
| Ждут модерации | `padCount(published.length)` on **signal fill** |
| Организаторов | `—` (deviation #4) |
| Пользователей | `—` (deviation #5) |

**Queues:** top 3 published (title + compact age); top pending orgs (name + age from `history[0].created_at` if present on detail — **list payload has no timestamp** → show `—` for org age, or omit age and show name only). Prefer name-only org rows with caption count `Заявки на верификацию · ${pendingOrgs.length}` rather than inventing ages.

**CTA:** `Link`/`Button` inverted `ОТКРЫТЬ ОЧЕРЕДЬ` → `/admin/moderation/events`.

**Mobile duty:** 2 tiles (Модерация = published.length signal; Верификация = pendingOrgs.length); up to 3 published rows with `age · organizer_name` + title (red if test heuristic); same CTA.

**Сигналы:** per deviation #6.

- [ ] **Step 1: Replace `app/admin/page.tsx`** with thin shell rendering `<AdminOverview />` (layout already has header)

- [ ] **Step 2: Implement `AdminOverview`** desktop grid + mobile duty (`max-sm:`)

- [ ] **Step 3: Browser verify** badge A1 desktop + 390px duty mode against HTML

- [ ] **Step 4: Gate + commit** — `feat(frontend): Swiss Grid A1 admin overview`

---

### Task 4: A2 · Модерация событий

**Files:**
- Create: `frontend/components/AdminModeration.tsx`
- Replace: `frontend/app/admin/moderation/events/page.tsx`

**Interfaces:**
- Consumes: `listModerationEvents`, `takedownEvent`, `reinstateEvent`, `fetchEventWithAuth`, `REJECT_REASON_CHIPS`, `concatenateReasons`, `revisionReason`, `nextQueueIndex`, `adminShortId`, `formatRelativeCompactRu`, `isLikelyTestContent`, `Chip` dark-active/dark-muted, `Button` inverted/destructive/dark-ghost, `Cell`/`CellStrip`, `AdminDesktopOnly`, `Skeleton`, `EmptyState`
- Produces: A2 conveyor UI

**Shell:**

```tsx
export default function Page() {
  return (
    <AdminDesktopOnly>
      <AdminModeration />
    </AdminDesktopOnly>
  );
}
```

**State:**

```ts
type Filter = "waiting" | "all";
// waiting → published only; all → published then rejected (tag status on row)
const [filter, setFilter] = useState<Filter>("waiting");
const [queue, setQueue] = useState<AdminEvent[]>([]);
const [selectedId, setSelectedId] = useState<string | null>(null);
const [reasons, setReasons] = useState<Set<string>>(new Set());
const [detail, setDetail] = useState<LiaEvent | null>(null);
```

**Selection styling:** selected row `bg-paper text-ink`; others transparent / on-surface. Test heuristic → title `text-signal` (even when selected, keep signal on title).

**Record pane:** when `selectedId` set, `fetchEventWithAuth` → cover (`coverUrl`, 120px), meta line `${adminShortId(id)} · подано ${formatRelativeCompactRu(starts_at)} назад` (copy may say «подано» even though source is starts_at — deviation #7), title, `${organizer} · ${category}`, four cells Дата / Цена / Мест / Формат, description clamp, reason chips, action bar.

**Actions:**

```ts
async function onApprove() {
  const row = queue.find((e) => e.id === selectedId);
  if (!row) return;
  if (row.status === "rejected") await reinstateEvent(row.id);
  advanceAfterRemoval(row.id); // or advance without removal if published no-op
}
async function onReject() {
  if (reasons.size < 1) { setError("Выберите причину"); return; }
  await takedownEvent(selectedId!, concatenateReasons([...reasons]));
  advanceAfterRemoval(selectedId!);
}
async function onRevision() {
  if (reasons.size < 1) { setError("Выберите причину"); return; }
  await takedownEvent(selectedId!, revisionReason([...reasons]));
  advanceAfterRemoval(selectedId!);
}
```

After removal, recompute index via `nextQueueIndex`, clear reasons, load next detail.

**Empty:** ink `EmptyState` with numeral `00`, `Очередь пуста`, CTA to `/admin`.

- [ ] **Step 1: Shell + DesktopOnly**

- [ ] **Step 2: Queue list + filter chips + selection**

- [ ] **Step 3: Record pane + reason chips + action bar wiring**

- [ ] **Step 4: Browser verify** badge A2 — select, reject with chip, auto-advance; &lt;900px notice

- [ ] **Step 5: Gate + commit** — `feat(frontend): Swiss Grid A2 moderation conveyor`

---

### Task 5: A3 · Организаторы (+ absorb moderation/organizers)

**Files:**
- Create: `frontend/components/AdminOrganizers.tsx`
- Replace: `frontend/app/admin/organizers/page.tsx`
- Replace: `frontend/app/admin/moderation/organizers/page.tsx` → redirect

**Interfaces:**
- Consumes: `listModerationOrganizers`, `searchOrganizers`, `getAdminOrganizer`, `verifyOrganizer`, `rejectOrganizer`, `revokeOrganizer`, `setOrganizerAutoVerify`, `adminOrgStatusLabel`, `isLikelyTestContent`, `adminShortId` (or last-2 of uuid for 44px col — mock uses short numeric; use last 2 hex of id padded, or `adminShortId` without `EV-` → add `adminOrgShortId` returning last 2 chars of hex stripped — **keep simple: first 2 of uuid hex uppercased**), `Chip`, `Button`, `StatusChip`, `AdminDesktopOnly`, `EmptyState`
- Produces: A3 table; verify/reject from old page live here

**Redirect page:**

```tsx
import { redirect } from "next/navigation";
export default function Page() {
  redirect("/admin/organizers?filter=pending");
}
```

**Filters:** `Все` (search with `q=""` or list merge pending∪verified∪rejected — **prefer** `searchOrganizers("")` if backend returns all; if empty-q returns [], fall back to concatenating three `listModerationOrganizers` calls). Chips: `Все · N` dark-active; `На проверке · N` signal; `Верифицированы · N` dark-muted; `С жалобами · N` → empty stub (deviation #11).

**Search:** right control — clicking focuses an inline `input` (border paper/muted) replacing the `Поиск ⌕` caption; debounce 300ms → `searchOrganizers`.

**Row primary action:**

| Condition | Label | Handler |
|---|---|---|
| pending | ПРОВЕРИТЬ | `verifyOrganizer` |
| verified (not test) | ОТКРЫТЬ | expand detail panel below row (history from `getAdminOrganizer`) |
| test heuristic | УДАЛИТЬ | opens ··· confirm → `revokeOrganizer` with reason `Тестовый организатор` |

**··· menu:** Отклонить (pending → reason prompt → `rejectOrganizer`); Отозвать (verified → reason → `revoke`); Авто-проверка toggle (`setOrganizerAutoVerify`). Reason prompt = minimal ink inline field (not browser `prompt`).

**Reject reason** for orgs: free-text field required (existing API); not the event chip set.

- [ ] **Step 1: Redirect old moderation/organizers page**

- [ ] **Step 2: Shell + DesktopOnly + table chrome**

- [ ] **Step 3: Filters, search, row actions, ··· panel**

- [ ] **Step 4: Browser verify** badge A3 — pending verify, signal chip, test name red, &lt;900px notice

- [ ] **Step 5: Gate + commit** — `feat(frontend): Swiss Grid A3 organizers registry`

---

### Task 6: Secondary admin routes (stub users + ink-safe settings/complaints)

**Files:**
- Create: `frontend/app/admin/users/page.tsx`
- Modify: `frontend/app/admin/settings/page.tsx`
- Modify: `frontend/app/admin/complaints/page.tsx`

**Not A4:** users page is only:

```tsx
import Link from "next/link";
import { EmptyState } from "@/components/ui/EmptyState";

export default function AdminUsersStub() {
  return (
    <EmptyState
      numeral="—"
      title="Пользователи"
      text="Реестр и гигиена контента появятся в следующей фазе."
      actions={
        <Link
          href="/admin"
          className="swiss-focus bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink"
        >
          К ОБЗОРУ
        </Link>
      }
    />
  );
}
```

Settings / complaints: strip `glass`, `rounded-*`, raw `text-red-600` → `text-signal`; use `Button` / `Chip` ink variants; keep behaviour.

- [ ] **Step 1: Users stub page**

- [ ] **Step 2: Restyle settings + complaints for ink**

- [ ] **Step 3: Verify** ADMIN_NAV → Пользователи shows stub; settings/complaints usable

- [ ] **Step 4: Gate + commit** — `feat(frontend): admin users stub and ink-safe secondary pages`

---

### Task 7: Deploy + verify (phase close)

**Files:**
- Create: `docs/superpowers/reports/2026-07-29-swiss-grid-phase-6.md` (short report; date may be 2026-07-30 if closing next day)
- Create: `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-6-HANDOFF.md` (Phase 7 = A4 next)

- [ ] **Step 1: Full gate** from `frontend/`: `pnpm build && pnpm test && pnpm lint`

- [ ] **Step 2: Live browser checklist** (desktop + A1 @ 390px) against HTML badges A1–A3:
  - [ ] `data-surface="ink"` on all `/admin/*`
  - [ ] `ADMIN_NAV` active underline; Пользователи → stub (not A4)
  - [ ] A1 four tiles; signal tile; queues; duty mode at 390px
  - [ ] A2 `250px 1fr`; selected paper row; reason gate; auto-advance; desktop-only &lt;900px
  - [ ] A3 filters + table columns; verify from row; moderation/organizers redirects
  - [ ] No radius/shadow regressions on admin routes
  - [ ] Tab bar hidden on `/admin`

- [ ] **Step 3: Deploy** per `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md` (only when user asks / after merge)

- [ ] **Step 4: Write report + handoff** (Phase 7 = A4 users + hygiene + backend)

- [ ] **Step 5: Commit docs** — `docs: Phase 6 admin report and handoff`

---

## Self-review (plan author)

**Spec coverage:**

| Requirement | Task |
|---|---|
| Ink surface + AppHeader admin | T2 |
| A1 tiles, queues, signals, duty mode | T3 |
| A2 queue/record, reasons, advance | T4 |
| A3 filters, table, verify absorb | T5 |
| A4 out of scope / users stub only | T6 + deviations |
| Desktop-only A2/A3 | T2 `AdminDesktopOnly` + T4/T5 |
| Pure helpers TDD | T1 |
| Deploy | T7 |
| No P1–P5 reopen | deviations header |
| Existing APIs only | deviations #2–#11 |

**Placeholder scan:** no TBD/TODO left in task steps; backend gaps named as stubs (`—`, heuristic, complaints filter empty).

**Type consistency:** `formatRelativeCompactRu`, `adminShortId`, `concatenateReasons`, `revisionReason`, `nextQueueIndex`, `adminOrgStatusLabel`, `isLikelyTestContent` names stable across tasks.

**HTML badge check:** A1 `:846`, A2 `:897`, A3 `:952` inventoried into Design fidelity contract; A4 `:985` explicitly excluded.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks (`superpowers:subagent-driven-development`)
2. **Inline Execution** — this session with `superpowers:executing-plans`, checkpoints between A1/A2/A3

**Which approach?**

Do not create the worktree or implement until you explicitly approve this plan (and pick an execution mode).
