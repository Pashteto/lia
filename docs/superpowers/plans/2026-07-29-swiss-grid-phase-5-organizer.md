# Swiss Grid Phase 5 — Organizer suite (O1–O5) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the five organizer workspace screens — **O1 Кабинет**, **O3 Мои события**, **O2 Создание события**, **O4 Заявки**, **O5 Профиль и верификация** — to Swiss Grid fidelity on existing routes and APIs, without reopening locked P1–P4 deviations.

**Architecture:** Thin page shells mount `AppHeader nav={ORG_NAV}` + a client body per screen (same P3/P4 pattern). New logic lives in pure Vitest-covered helpers (`org-event-status`, `org-seats`, `org-dashboard`, `org-verification`, `relative-time`). Screens reuse Phase 1 primitives (`Cell`/`CellStrip`, `Chip`, `Button`, `Stepper`, `ProgressBar`, `StatusChip`, `EventModule`, `Field`, `EmptyState`, `AuthGate`, `Skeleton`). Ship against existing organizer APIs; backend gaps become deliberate stubs or client-side loops (same class as P3/P4).

**Tech Stack:** Next.js 16 App Router / React 19 / TypeScript, Tailwind v4 (Swiss Grid tokens), TanStack Query 5, react-hook-form + zod (unchanged schema), Vitest 4 (node-only pure helpers), existing Go endpoints (`/events/mine`, CRUD events, applications decide, `/me/organizer`).

**Spec:** `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` § O1–O5. Pixel: `Presence Swiss Grid - Full System.dc.html` badges `O1 · Кабинет` (~568), `O2 · Создание события` (~635), `O3 · Мои события` (~690), `O4 · Заявки участников` (~736), `O5 · Профиль и верификация` (~782). Handoff: `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-4-HANDOFF.md`. Master: `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` Phase 5.

**Prerequisite:** `main` tip includes Phase 4 merge (`37f769e` or later; tip may be a docs commit after that). Verify before starting:

```bash
git log --oneline -5
# must include 37f769e (Phase 4 U3) somewhere in recent history
test -f frontend/components/ui/Stepper.tsx
test -f frontend/components/ui/ProgressBar.tsx
test -f frontend/components/ui/AppHeader.tsx
grep -q 'ORG_NAV' frontend/components/ui/AppHeader.tsx
```

**Branch / worktree (at execution time only):** `redesign/swiss-grid-p5` via `superpowers:using-git-worktrees`, branched from `main` @ ≥ Phase 4 merge. Working directory for all `pnpm` commands: `frontend/`. **Do not implement until this plan is reviewed.**

**Screen order:** O1 → O3 → O2 → O4 → O5 → deploy+verify (master plan).

---

## Global Constraints

Everything from the master plan (`2026-07-28-swiss-grid-redesign-master-plan.md`) applies. This phase trips over:

- **Zero border radius, zero shadows.** Rules 1px solid: `#111` structural (`border-ink` / `border-on-surface`), `#DDD` inner (`border-rule-inner`), table head `#E6E3DC` (`bg-table-head` / token).
- **Categories are numerals** via `categoryNumeral` / `eventToModuleProps` — never colours on chips for category pick.
- **All numbers in JetBrains Mono** (`font-mono`): status strip heroes, seat counters, dates, relative times when numeric, stepper numerals. Unknown → `—`.
- **`signal` red only for needs-attention:** O1 applications banner left border + red count; O3 «Модерация · N» chip; O4 «ОТКЛОНИТЬ» is **ghost** (not red fill — mock uses `.cta.gh`); destructive stays rare.
- Uppercase tracking preserved (`.cap` 0.13em, chips 0.12em, buttons 0.07em, nav 0.14em).
- Hover **inverts** (`hover-invert`), focus `swiss-focus`, transitions 120ms linear on background/color. Loading = `Skeleton` — **no spinners**.
- Empty / gated / error = `EmptyState` / `AuthGate` / designed error strip.
- **Tap targets ≥44px on touch.** Dense mock paddings grow; type sizes stay.
- Content max 1360px; Moscow formatters; React #418 (never nest anchors).
- UI copy Russian; code/commits English.
- Fonts stay **Golos Text / Manrope / JetBrains Mono** (locked P1). Manrope has no 900 — titles use `font-black` on `--font-ui`.
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Deliberate deviations for Phase 5 (pre-decided — do not reopen P1–P4; do not "fix" these without product sign-off)

1. **Routes stay mapped** (master plan): `/organizer`, `/events/mine`, `/events/new` (+ `/events/[id]/edit`), `/organizer/applications`, `/me/organizer`, public `/organizers/[id]` — **not** handoff `/org/*`.
2. **No `GET /org/summary`.** O1 tiles derive from `fetchMyEvents` (+ optional application fetches). **Activity log has no endpoint** → right rail shows a designed stub caption (`Последнее` + one `.cap` line `Лента активности появится позже`) — never fake rows.
3. **«Всего записей»** = sum of filled seats across the organizer’s events where `capacity` and `seatsRemaining` are both known; if none known, show `—` (not `00`). Do not invent RSVP totals without data.
4. **O4 stays on `/organizer/applications`.** Handoff path `/org/events/:id/applications` is conceptual. UI: event-picker list → selecting an event reveals the O4 chrome (context strip + tabs + rows + bulk bar). Deep-link `?event=<id>` selects that event.
5. **Bulk accept/reject** = sequential client loop over `decideApplication` (master plan). No new backend endpoint in this phase.
6. **Application «meta» captions** (`Была на 3 событиях · медиации`) have **no backend**. Use: trimmed `applicationAnswer` when present; else `Первая заявка` for `applied` with empty answer. Never invent attendance history.
7. **O2 autosave** has no draft resource API. Behaviour: (a) **edit mode** — debounced `patchEvent` on blur of dirty fields, header chip `ЧЕРНОВИК СОХРАНЁН · HH:mm` (Moscow); (b) **create mode** — ghost `ЧЕРНОВИК` creates via `createEvent` with `status:"draft"` then navigates to edit (or stays and switches to edit with returned id); no silent create-on-blur before first save. Stepper is **visual only** — one RHF form + existing `eventFormSchema` (schema tests must keep passing).
8. **O2 field remap into 4 steps** (visual sections, all fields still in one form instance):
   - **01 Основное:** title, category chips, description, cover
   - **02 Когда и где:** format, venue/picker, starts/ends
   - **03 Билеты:** isFree/price, capacity, signupMode + curator/external fields
   - **04 Публикация:** status draft/published, submit / moderation note
9. **Invite + feedback expanders** on mine are **not in the O3 mock**. Keep them reachable via a row overflow control `···` (opens a below-row panel) so no feature is lost — do not put them in the five mock columns.
10. **«Копия»** = client duplicate: `createEvent` with copied fields as `draft`, then navigate to `/events/{newId}/edit`. No clone endpoint.
11. **O5 contacts:** mock shows email + telegram; API has `website_url` only. Map **website_url** into the contacts stack (one or two fields: website primary; optional display of user email from `useAuth` as read-only caption if available). Do not invent a telegram field in the API.
12. **Public org follower count** is not in `getPublicOrganizer` → show `Подписчиков` as `—` (or omit the second cell’s number as `—`). Event count = `fetchEventsByOrganizer` length.
13. **Public org `logo_url`** may be absent from the public GET type — if missing, ink 34×34 / 40×40 square placeholder (mock). Do not block the screen.
14. **O1 hub AuthGate:** when logged out, show Swiss identity + `AuthGate` pattern (not the old link-card hub). Creating events still requires auth (existing).
15. **Stepper semantics differ O2 vs O5:** O2 active step is **ink-filled** (inclusive). O5 completed steps filled, **current/last on paper** (exclusive fill) — extend `Stepper` with `fillMode?: "inclusive" | "exclusive"` (default `"inclusive"`).
16. **Bottom tab bar:** hide on `/me/organizer` via `TabBarGate` (organizer mobile = header context only). Public `/organizers/[id]` keeps user tabs.

**Locked from earlier phases (do not reopen):** favorites/♡ deferred; home `/`; U4 personal calendar; Yandex not OSM; fonts Golos/Manrope/JB Mono; U3 non-AI `/search`; etc.

---

## Design fidelity contract (HTML badges checked 2026-07-29)

`Presence Swiss Grid - Full System.dc.html` is pixel authority; README § O1–O5 is behaviour authority.

| Screen | Desktop (key) | Mobile (key) |
|---|---|---|
| **O1** `:568-633` | Identity `1fr 190px` (pad 16/20, cap + H1 30px / CTA full width bottom); 4-cell strip pad 14px, values **26px mono**, «На модерации» **ink invert** + cap `#A8A299`; banner `border-left:4px solid #E2231A`, red mono 22px + 12.5px/700 + chip `Смотреть →`; bottom `1fr 1fr` next-event (title 17px/900, bar 8px) + activity | 3-cell strip (Опубл./Модер./Черн. 16px); banner shorter copy; next-event only; sticky `+ СОЗДАТЬ СОБЫТИЕ` |
| **O2** `:635-688` | Header right `ЧЕРНОВИК СОХРАНЁН · HH:mm`; 4-cell step strip (active ink); body `1fr 240px`; fields gap 11px; cover dropzone 62px; CTA `ДАЛЕЕ · {next}` + ghost `ЧЕРНОВИК`; white preview card `1px #111` | `01/04` header; 4×5px segment bar (ink/`#DCD8D0`); moderation note above `ДАЛЕЕ` |
| **O3** `:690-734` | Title 26px + small `+ СОЗДАТЬ`; chips `Все · N` / … / Модерация **signal**; head `56px 1fr 96px 110px 92px` bg `#E6E3DC`; rows same cols, 5px seat bar, stacked `Ред.`/`Копия` | chips scroll; stacked rows date+status / title / seats+bar |
| **O4** `:736-780` | Context `1fr 200px` (title 22px / filled 22px mono + 6px bar); tabs + `Выбрать все`; rows `26px 1fr 120px 150px`; 11×11 checkbox; small ПРИНЯТЬ/ОТКЛОНИТЬ; pinned bulk bar | capacity block; cards with full-width actions |
| **O5** `:782-837` | Verification strip 3 equal segments; form `1fr 250px` + white preview (34px avatar, stats footer); `СОХРАНИТЬ` | **Mobile frame = public view** (identity, 2 cells, event rows, `ПОДПИСАТЬСЯ`) — implement on `/organizers/[id]`; edit page may stack form below stepper on small screens |

---

## File Structure

```
frontend/
  lib/org-event-status.ts                 CREATE — EventStatus → RU label for StatusChip
  lib/org-seats.ts                        CREATE — filled/capacity from capacity + seatsRemaining
  lib/org-dashboard.ts                    CREATE — O1 counts + next upcoming event
  lib/org-verification.ts                 CREATE — verification_status → stepper index
  lib/relative-time.ts                    CREATE — "2 ч назад" / "вчера" (Moscow)
  lib/__tests__/org-event-status.test.ts  CREATE
  lib/__tests__/org-seats.test.ts         CREATE
  lib/__tests__/org-dashboard.test.ts     CREATE
  lib/__tests__/org-verification.test.ts  CREATE
  lib/__tests__/relative-time.test.ts     CREATE

  components/ui/Stepper.tsx               MODIFY — fillMode inclusive|exclusive
  components/ui/TabBarGate.tsx            MODIFY — hide /me/organizer

  components/OrganizerHub.tsx             CREATE — O1 client body
  app/organizer/page.tsx                  REPLACE — thin shell

  components/MyEventsBrowse.tsx           CREATE — O3 client body
  app/events/mine/page.tsx                REPLACE — thin shell

  components/CreateEventForm.tsx          REWRITE chrome — keep schema + submit logic
  app/events/new/page.tsx                 MODIFY — AppHeader + form
  app/events/[id]/edit/page.tsx          MODIFY — AppHeader + form

  components/OrganizerApplications.tsx    CREATE — O4 picker + event chrome
  components/EventApplicationsPanel.tsx   REWRITE — tabs, checkboxes, bulk, optimistic seats
  app/organizer/applications/page.tsx     REPLACE — thin shell

  components/OrganizerProfileEdit.tsx     CREATE — O5 edit body
  app/me/organizer/page.tsx               REPLACE — thin shell
  components/PublicOrganizerView.tsx      CREATE — O5 public / mobile mock
  app/organizers/[id]/page.tsx           REPLACE — thin shell + PublicOrganizerView
```

**Not touched:** admin screens, U1–U8 browse bodies (except shared Stepper/TabBarGate), backend Go code, `eventFormSchema` rules (unless a bug blocks the visual stepper — prefer UI-only validation messaging).

**Data reuse:**

| Need | Source |
|---|---|
| My events | `fetchMyEvents()` — `lib/api.ts` |
| Create/patch | `createEvent` / `patchEvent` |
| Applications | `fetchEventApplications` / `decideApplication` |
| Org profile | `getMyOrganizer` / `saveMyOrganizer` / `submitMyOrganizer` / `uploadFile` |
| Public org | `getPublicOrganizer` / `fetchEventsByOrganizer` / follow APIs |
| Module preview | `eventToModuleProps` + live form watch → synthetic `LiaEvent`-like props |
| Status chips | `statusChipVariant` + new `orgEventStatusLabel` |
| Seat bar | `ProgressBar` + `org-seats` |
| Categories | `getCategories()` |

---

### Task 1: Pure helpers — status, seats, dashboard, verification, relative time (TDD)

**Files:**
- Create: `frontend/lib/org-event-status.ts`
- Create: `frontend/lib/org-seats.ts`
- Create: `frontend/lib/org-dashboard.ts`
- Create: `frontend/lib/org-verification.ts`
- Create: `frontend/lib/relative-time.ts`
- Test: matching files under `frontend/lib/__tests__/`

**Interfaces:**
- Produces:

```ts
// org-event-status.ts
import type { EventStatus } from "@/lib/types"; // draft | pending_review | published | rejected | cancelled | …

/** Russian label that feeds StatusChip / statusChipVariant. */
export function orgEventStatusLabel(status: string): string;
// published → "Опубликовано"; draft → "Черновик"; pending_review → "На модерации";
// rejected → "Снято модератором"; cancelled → "Отменено"; unknown → status as-is

// org-seats.ts
export function seatsFill(
  event: { capacity?: number | null; seatsRemaining?: number | null },
): { filled: number; capacity: number; label: string; ratio: number } | null;
// null when capacity missing OR seatsRemaining missing
// label = `${filled} / ${capacity}`; ratio = filled/capacity clamped 0..1

export function padCount(n: number, width = 2): string;
// Math.floor, padStart; negative → "00"

// org-dashboard.ts
export interface OrgDashboardStats {
  published: number;
  pendingReview: number;
  drafts: number;
  /** Sum of seatsFill.filled across events where seatsFill != null; null if none measurable */
  totalRegistrations: number | null;
}

export function orgDashboardStats(events: ReadonlyArray<{ status?: string; capacity?: number | null; seatsRemaining?: number | null }>): OrgDashboardStats;

/** Soonest upcoming (startsAt >= now) by startsAt asc; else null. */
export function nextOrganizerEvent<T extends { startsAt: string }>(
  events: ReadonlyArray<T>,
  now?: Date,
): T | null;

// org-verification.ts
export const ORG_VERIFY_STEPS = [
  "Заявка подана",
  "Документы проверены",
  "Верифицирован ✓",
] as const;

/** 0-based current step for Stepper fillMode="exclusive". */
export function orgVerificationStep(
  status: "draft" | "pending" | "verified" | "rejected",
): number;
// draft|rejected → 0; pending → 1; verified → 2

// relative-time.ts
/** Compact RU relative for O4 / activity. Pins Europe/Moscow for calendar-day checks. */
export function formatRelativeRu(iso: string, now?: Date): string;
// <60s → "только что"; <60m → "N мин"; <24h → "N ч назад"; yesterday civil → "вчера";
// else → formatShortDate(iso)
```

- Consumes: `formatShortDate` from `lib/format.ts` (relative-time only).

- [ ] **Step 1: Write the failing tests**

```ts
// lib/__tests__/org-event-status.test.ts
import { describe, expect, it } from "vitest";
import { orgEventStatusLabel } from "../org-event-status";
import { statusChipVariant } from "../status-chip";

describe("orgEventStatusLabel", () => {
  it("maps known statuses to handoff chip strings", () => {
    expect(orgEventStatusLabel("published")).toBe("Опубликовано");
    expect(orgEventStatusLabel("draft")).toBe("Черновик");
    expect(orgEventStatusLabel("pending_review")).toBe("На модерации");
  });
  it("feeds statusChipVariant correctly", () => {
    expect(statusChipVariant(orgEventStatusLabel("published"))).toBe("active");
    expect(statusChipVariant(orgEventStatusLabel("draft"))).toBe("default");
    expect(statusChipVariant(orgEventStatusLabel("pending_review"))).toBe("signal");
  });
});
```

```ts
// lib/__tests__/org-seats.test.ts
import { describe, expect, it } from "vitest";
import { padCount, seatsFill } from "../org-seats";

describe("seatsFill", () => {
  it("derives filled from capacity - seatsRemaining", () => {
    expect(seatsFill({ capacity: 40, seatsRemaining: 12 })).toEqual({
      filled: 28,
      capacity: 40,
      label: "28 / 40",
      ratio: 0.7,
    });
  });
  it("returns null when capacity or seatsRemaining missing", () => {
    expect(seatsFill({ capacity: 40 })).toBeNull();
    expect(seatsFill({ seatsRemaining: 3 })).toBeNull();
    expect(seatsFill({})).toBeNull();
  });
});

describe("padCount", () => {
  it("pads to two digits", () => {
    expect(padCount(3)).toBe("03");
    expect(padCount(86)).toBe("86");
  });
});
```

```ts
// lib/__tests__/org-dashboard.test.ts
import { describe, expect, it } from "vitest";
import { nextOrganizerEvent, orgDashboardStats } from "../org-dashboard";

describe("orgDashboardStats", () => {
  it("counts by status and sums measurable seats", () => {
    const s = orgDashboardStats([
      { status: "published", capacity: 40, seatsRemaining: 12 },
      { status: "published", capacity: 10, seatsRemaining: 10 },
      { status: "pending_review" },
      { status: "draft" },
      { status: "draft" },
    ]);
    expect(s).toEqual({
      published: 2,
      pendingReview: 1,
      drafts: 2,
      totalRegistrations: 28, // 28 + 0
    });
  });
  it("returns null totalRegistrations when no seats measurable", () => {
    expect(orgDashboardStats([{ status: "published" }]).totalRegistrations).toBeNull();
  });
});

describe("nextOrganizerEvent", () => {
  const now = new Date("2026-07-10T12:00:00Z");
  it("picks the soonest future start", () => {
    const n = nextOrganizerEvent(
      [
        { id: "a", startsAt: "2026-07-12T13:00:00Z" },
        { id: "b", startsAt: "2026-07-11T10:00:00Z" },
        { id: "c", startsAt: "2026-07-01T10:00:00Z" },
      ],
      now,
    );
    expect(n?.id).toBe("b");
  });
  it("returns null when all past", () => {
    expect(nextOrganizerEvent([{ startsAt: "2026-07-01T10:00:00Z" }], now)).toBeNull();
  });
});
```

```ts
// lib/__tests__/org-verification.test.ts
import { describe, expect, it } from "vitest";
import { orgVerificationStep } from "../org-verification";

describe("orgVerificationStep", () => {
  it("maps statuses", () => {
    expect(orgVerificationStep("draft")).toBe(0);
    expect(orgVerificationStep("rejected")).toBe(0);
    expect(orgVerificationStep("pending")).toBe(1);
    expect(orgVerificationStep("verified")).toBe(2);
  });
});
```

```ts
// lib/__tests__/relative-time.test.ts
import { describe, expect, it } from "vitest";
import { formatRelativeRu } from "../relative-time";

describe("formatRelativeRu", () => {
  const now = new Date("2026-07-12T15:00:00+03:00");
  it("formats hours ago", () => {
    expect(formatRelativeRu("2026-07-12T13:00:00+03:00", now)).toBe("2 ч назад");
  });
  it("formats yesterday", () => {
    expect(formatRelativeRu("2026-07-11T18:00:00+03:00", now)).toBe("вчера");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm vitest run lib/__tests__/org-event-status.test.ts lib/__tests__/org-seats.test.ts lib/__tests__/org-dashboard.test.ts lib/__tests__/org-verification.test.ts lib/__tests__/relative-time.test.ts`

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement the five modules** (minimal code matching the interfaces above).

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/org-*.ts frontend/lib/relative-time.ts frontend/lib/__tests__/org-*.test.ts frontend/lib/__tests__/relative-time.test.ts
git commit -m "$(cat <<'EOF'
feat(frontend): add Phase 5 organizer pure helpers

EOF
)"
```

---

### Task 2: Stepper `fillMode` + TabBarGate `/me/organizer`

**Files:**
- Modify: `frontend/components/ui/Stepper.tsx`
- Modify: `frontend/components/ui/TabBarGate.tsx`
- Optional smoke: `frontend/app/design-preview/page.tsx` (show both fill modes if easy)

**Interfaces:**
- Consumes: Task 1 `ORG_VERIFY_STEPS` (for preview only).
- Produces: `Stepper({ steps, current, fillMode?: "inclusive" | "exclusive" })` — default `"inclusive"` (O2). Exclusive fills `i < current` only (O5).

- [ ] **Step 1: Update Stepper**

```tsx
export function Stepper({
  steps,
  current,
  fillMode = "inclusive",
  className,
}: {
  steps: string[];
  current: number;
  fillMode?: "inclusive" | "exclusive";
  className?: string;
}) {
  const filled = (i: number) =>
    fillMode === "exclusive" ? i < current : i <= current;
  // … same grid markup; use filled(i) instead of i <= current
}
```

For O5 verified (`current === 2`): steps 0 and 1 filled, step 2 on paper — matches HTML `:791-794`.

- [ ] **Step 2: TabBarGate** — add `"/me/organizer"` to `HIDDEN_PREFIXES`.

- [ ] **Step 3: `pnpm build && pnpm test && pnpm lint`**

- [ ] **Step 4: Commit** — `fix(frontend): Stepper fillMode + hide org profile tab bar`

---

### Task 3: O1 · Кабинет — `OrganizerHub`

**Files:**
- Create: `frontend/components/OrganizerHub.tsx`
- Replace: `frontend/app/organizer/page.tsx`

**Interfaces:**
- Consumes: `orgDashboardStats`, `nextOrganizerEvent`, `seatsFill`, `padCount`, `getMyOrganizer`, `fetchMyEvents`, `AppHeader`+`ORG_NAV`, `Cell`/`CellStrip`, `Button`, `Chip`, `ProgressBar`, `AuthGate`, `Skeleton`.
- Produces: Swiss O1 dashboard.

**Pending-applications banner count:** For each event with `signupMode === "application"`, `fetchEventApplications` in parallel (TanStack `useQueries` or one `Promise.all` inside a queryFn). Count `status === "applied"`. Cap concurrency mentally — organizers rarely have dozens of application events. Banner hidden when count === 0.

**Activity rail:** stub only (deviation #2).

- [ ] **Step 1: Thin shell** `app/organizer/page.tsx`:

```tsx
import { AppHeader, ORG_NAV } from "@/components/ui/AppHeader";
import { OrganizerHub } from "@/components/OrganizerHub";

export default function OrganizerPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="КАБИНЕТ" />
      <OrganizerHub />
    </>
  );
}
```

- [ ] **Step 2: Implement `OrganizerHub`** per fidelity table:
  - Logged out → `AuthGate` with title about кабинет / создание событий; primary `ВОЙТИ`.
  - Loading → `Skeleton` strips at final heights (status strip ~70px, etc.).
  - Identity: `Организатор · {name}{verified ? " ✓" : ""}` + H1 `Кабинет` 30px / mobile 20px; CTA `+ СОЗДАТЬ СОБЫТИЕ` → `/events/new`.
  - Status strip desktop 4 cells; mobile 3 (drop «Всего записей»). Invert cell when `pendingReview > 0` **or always invert the «На модерации» cell** (mock always inverts that cell even conceptually — **always invert that cell** to match HTML).
  - Banner → `/organizer/applications` (chip `Смотреть →`).
  - Next event block links to `/events/{id}`; seats via `seatsFill` + `ProgressBar`.
  - Numbers via `padCount` for strip heroes when &lt; 100; for `totalRegistrations` use `padCount` only if &lt; 100 else raw mono.

- [ ] **Step 3: Browser check** at 390px + desktop against HTML badge O1 (local `pnpm dev`).

- [ ] **Step 4: Commit** — `feat(frontend): Swiss Grid O1 organizer hub`

---

### Task 4: O3 · Мои события — `MyEventsBrowse`

**Files:**
- Create: `frontend/components/MyEventsBrowse.tsx`
- Replace: `frontend/app/events/mine/page.tsx`

**Interfaces:**
- Consumes: Task 1 helpers, `StatusChip`, `Chip`, `ProgressBar`, `PublishEventButton` (keep behind `···` overflow with invite/feedback), `AuthGate`.
- Produces: Swiss table / mobile stacked list.

**Tab model (client filter):**

| Chip | Predicate |
|---|---|
| Все | all |
| Опубликовано | `status === "published"` |
| Модерация | `status === "pending_review"` — **signal** chip variant |
| Черновики | `status === "draft"` |

Counts update from full list. Rejected/cancelled appear under Все only (or under Черновики? → **Все only**, label via `orgEventStatusLabel`).

- [ ] **Step 1: Thin shell** with `AppHeader nav={ORG_NAV} mobileCaption="МОИ СОБЫТИЯ"` + `MyEventsBrowse`.

- [ ] **Step 2: Desktop table**
  - Title bar pad `14px 20px`; H1 26px; `Button` small `+ СОЗДАТЬ` → `/events/new`.
  - Chip row pad `9px 20px`.
  - Header row `grid-cols-[56px_1fr_96px_110px_92px] bg-[#E6E3DC]` (use token `bg-table-head` if present, else arbitrary matching token sheet `#E6E3DC`).
  - Body rows same columns: `formatShortDate` mono; title 12.5px/700 + venue `.cap`; seats `seatsFill` or `—` + `ProgressBar thin`; `StatusChip`; actions column `.cap` links `Ред.` → `/events/{id}/edit`, button `Копия` → duplicate helper.

- [ ] **Step 3: Duplicate helper** (inline in component or tiny `lib/org-duplicate-event.ts` — if extracted, add a unit test for the payload mapper only):

```ts
async function duplicateAsDraft(event: LiaEvent): Promise<string> {
  const created = await createEvent({ /* map fields; status: "draft" */ });
  return created.id;
}
```

- [ ] **Step 4: Mobile stacked rows** per HTML; chips horizontally scrollable; `+` in header actions optional.

- [ ] **Step 5: Overflow `···`** row expands Invite / Applications / Feedback / Publish — preserve existing panels, Swiss-styled containers (no rounded-card shadows).

- [ ] **Step 6: Empty** → `EmptyState` numeral `00`, `Событий пока нет`, CTA `СОЗДАТЬ СОБЫТИЕ`.

- [ ] **Step 7: Verify + commit** — `feat(frontend): Swiss Grid O3 my events table`

---

### Task 5: O2 · Создание / редактирование — visual wizard over `CreateEventForm`

**Files:**
- Modify: `frontend/components/CreateEventForm.tsx` (large restyle; **keep** `eventFormSchema`, `FormValues`, `toDatetimeLocalValue`, create/patch mutations, email-verify interstitial)
- Modify: `frontend/app/events/new/page.tsx`
- Modify: `frontend/app/events/[id]/edit/page.tsx`
- Keep passing: `frontend/components/__tests__/create-event-schema.test.ts`

**Interfaces:**
- Consumes: `Stepper` (inclusive), `Field`/`Input`/`Textarea`, `Chip` for categories, `ImageUpload`, `VenuePicker`, `EventModule` (preview), `ProgressBar` segments on mobile, `ORG_NAV` or create-mode header caption.
- Produces: four-step UI; live preview; autosave chip (edit).

**Step state:** `const [step, setStep] = useState(0)` — does **not** unmount fields; hide inactive sections with `hidden={step !== i}` or `max-sm` show only current. Desktop may show only current step fields (match mock) while RHF retains all values via registered fields kept mounted (`hidden` class, not conditional unmount) — **critical for RHF**.

**Next CTA labels:**
- step0 → `ДАЛЕЕ · КОГДА И ГДЕ`
- step1 → `ДАЛЕЕ · БИЛЕТЫ`
- step2 → `ДАЛЕЕ · ПУБЛИКАЦИЯ`
- step3 → primary publish/save per existing mutation (`СОХРАНИТЬ` / submit copy already in form)

**Preview:** `useWatch` → build `EventModule` props (numeral from selected category via `categoryNumeral`, title, venue name from picker state or «—», date from `formatModuleDate`, price from `priceLabel`). Wrap in `border border-ink bg-white` white card. On mobile, preview optional below fold or omitted (mock omits) — **omit preview on `max-sm`**.

**Header:** create: `AppHeader` with `mobileCaption="НОВОЕ СОБЫТИЕ"` and `actions` showing `01/04`; desktop right caption via `actions` = autosave text. Prefer passing `actions` rather than replacing nav — for create/edit use `nav={ORG_NAV}` still so organizer can escape.

**Replace** emoji `CreateEventGate` with `AuthGate`.

**Replace** local `inputCls` rounded styles with Swiss `Field` components.

**Segmented / Switch:** restyle to Chip rows / ink switches without radius, or keep `Segmented` if already Swiss-safe — if `Segmented` still has radius, migrate signup/format controls to `Chip` toggles in this task (P3 deferred Segmented cleanup to P5).

- [ ] **Step 1: Run schema tests** — `pnpm vitest run components/__tests__/create-event-schema.test.ts` (baseline green).

- [ ] **Step 2: Restyle form chrome + stepper + sections** without changing schema.

- [ ] **Step 3: Live preview + moderation note** (copy exact RU from HTML).

- [ ] **Step 4: Edit-mode debounced autosave** — on blur, if `mode==="edit" && eventId && dirty`, `patchEvent`; set `savedAt` Date; display `ЧЕРНОВИК СОХРАНЁН · ${Moscow HH:mm}`.

- [ ] **Step 5: Re-run schema tests + full `pnpm test` + browser O2 badge compare.

- [ ] **Step 6: Commit** — `feat(frontend): Swiss Grid O2 create-event wizard chrome`

---

### Task 6: O4 · Заявки — picker + panel rewrite

**Files:**
- Create: `frontend/components/OrganizerApplications.tsx`
- Rewrite: `frontend/components/EventApplicationsPanel.tsx`
- Replace: `frontend/app/organizer/applications/page.tsx`

**Interfaces:**
- Consumes: `decideApplication`, `fetchEventApplications`, `seatsFill`, `formatRelativeRu`, `Chip`, `Button`, `ProgressBar`, `EmptyState`.
- Produces: O4 fidelity for one selected event; bulk loop helper.

**Panel API (extend props):**

```ts
interface Props {
  eventId: string;
  event: Pick<LiaEvent, "title" | "startsAt" | "capacity" | "seatsRemaining" | "signupMode">;
  /** Optimistic seat overrides from parent optional */
}
```

**Tabs:** `Новые` = `applied`; `Принятые` = `accepted` | `going`; `Отклонённые` = `declined`. Counts on chips.

**Selection:** `Set<string>` of rsvp ids (pending only). `Выбрать все` toggles all pending in current filter.

**Bulk:** 

```ts
export async function decideMany(
  eventId: string,
  ids: string[],
  decision: "accept" | "decline",
  decide = decideApplication,
): Promise<{ ok: string[]; failed: string[] }> {
  const ok: string[] = [];
  const failed: string[] = [];
  for (const id of ids) {
    try {
      await decide(eventId, id, decision);
      ok.push(id);
    } catch {
      failed.push(id);
    }
  }
  return { ok, failed };
}
```

Put `decideMany` in `lib/org-applications.ts` with a Vitest that mocks `decide` (TDD — small Task 6a).

**Optimistic seats:** on accept, if `seatsFill` known, locally increment filled by 1 (clamp to capacity) until invalidate completes. On decline, no seat change.

**Checkbox:** 11×11 square `border border-ink`, not native rounded.

- [ ] **Step 1: TDD `decideMany`** in `lib/org-applications.ts` + test.

- [ ] **Step 2: Shell page** `AppHeader` + `OrganizerApplications`.

- [ ] **Step 3: Picker state** — list application-mode events as hairline rows; `?event=` from `useSearchParams`; selecting sets id.

- [ ] **Step 4: Rewrite `EventApplicationsPanel`** to O4 layout; remove rounded-card / green-red text classes; use Swiss buttons.

- [ ] **Step 5: Browser verify** badge O4 + optimistic accept.

- [ ] **Step 6: Commit** — `feat(frontend): Swiss Grid O4 applications suite`

---

### Task 7: O5 · Профиль + public view

**Files:**
- Create: `frontend/components/OrganizerProfileEdit.tsx`
- Create: `frontend/components/PublicOrganizerView.tsx`
- Replace: `frontend/app/me/organizer/page.tsx`
- Replace: `frontend/app/organizers/[id]/page.tsx`

**Interfaces:**
- Consumes: `orgVerificationStep`, `ORG_VERIFY_STEPS`, `Stepper fillMode="exclusive"`, `getMyOrganizer`/`save`/`submit`, `getPublicOrganizer`, `fetchEventsByOrganizer`, follow APIs, `EventModule` or compact rows matching mobile mock.

**Edit page:**
- `AppHeader nav={ORG_NAV} mobileCaption="ПРОФИЛЬ"`.
- Stepper across top.
- Body `md:grid-cols-[1fr_250px]`: form (name, description textarea ~46px min height, 62×62 logo, website field(s), `СОХРАНИТЬ`; if draft/rejected show secondary `ОТПРАВИТЬ НА ПРОВЕРКУ` → `submitMyOrganizer`).
- Preview card white: avatar, name + ✓ if verified, `.cap` `Проверенный организатор` only when verified (else `Организатор` / `На проверке`), description clamp, footer Событий (from `fetchMyEvents` length or published count) / Подписчиков `—`.

**Public page:**
- User `AppHeader` (USER_NAV) — not ORG_NAV.
- Match mobile mock at all breakpoints as primary composition (identity, 2 cells, upcoming rows, `ПОДПИСАТЬСЯ` / `ВЫ ПОДПИСАНЫ` toggle using existing follow).
- Use `EventModule` or compact hairline rows; prefer compact rows to match mock density.

- [ ] **Step 1: Implement edit body + shell**

- [ ] **Step 2: Implement public view + shell**

- [ ] **Step 3: Browser** O5 desktop edit + mobile public frames

- [ ] **Step 4: Commit** — `feat(frontend): Swiss Grid O5 organizer profile`

---

### Task 8: Deploy + verify (phase close)

**Files:**
- Create: `docs/superpowers/reports/2026-07-29-swiss-grid-phase-5.md` (short report)
- Create: `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-5-HANDOFF.md`

- [ ] **Step 1: Full gate** from `frontend/`: `pnpm build && pnpm test && pnpm lint`

- [ ] **Step 2: Live browser checklist** (390 + desktop) against HTML badges O1–O5:
  - [ ] ORG_NAV active underline on each screen
  - [ ] O1 invert moderation cell + red banner only when pending &gt; 0
  - [ ] O3 table columns + signal moderation chip
  - [ ] O2 stepper + preview + schema tests green
  - [ ] O4 bulk accept loop + optimistic seats
  - [ ] O5 exclusive stepper + public subscribe CTA
  - [ ] No radius/shadow regressions on these routes
  - [ ] Tab bar hidden on organizer routes including `/me/organizer`

- [ ] **Step 3: Deploy** per `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md` (only when user asks to deploy / after merge).

- [ ] **Step 4: Write report + handoff** (Phase 6 = Admin A1–A3 next).

- [ ] **Step 5: Commit docs** — `docs: Phase 5 organizer report and handoff`

---

## Self-review (plan author)

**Spec coverage:**
| Requirement | Task |
|---|---|
| O1 dashboard tiles, banner, next event | T3 |
| O1 activity deferred | deviation #2 + T3 stub |
| O3 chips + table + seat bars | T4 |
| O2 4-step wizard + preview + autosave chip | T5 |
| O4 tabs, checkbox, bulk, optimistic seats | T6 |
| O5 stepper + form + preview + public | T7 |
| Deploy | T8 |
| Pure helpers TDD | T1 |
| ORG_NAV wired | T3–T7 shells |
| No P1–P4 reopen | deviations header |

**Placeholder scan:** no TBD/TODO left in task steps; backend gaps named as stubs/loops.

**Type consistency:** `seatsFill`, `orgDashboardStats`, `orgVerificationStep`, `decideMany`, `fillMode` names stable across tasks.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-29-swiss-grid-phase-5-organizer.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks (`superpowers:subagent-driven-development`)
2. **Inline Execution** — this session with `superpowers:executing-plans`, checkpoints between O1/O3/O2/O4/O5

**Which approach?**

Do not create the worktree or implement until you explicitly approve this plan (and pick an execution mode).
