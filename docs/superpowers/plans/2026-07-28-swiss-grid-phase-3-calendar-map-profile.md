# Swiss Grid Phase 3 — U4 Календарь, U5 Карта, U6 Профиль Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the three remaining user-layer screens — the personal calendar (`/me/calendar` = U4), the map browser (`/map` = U5), and a new consolidated profile (`/me` = U6, absorbing `/me/practices` + `/me/applications` + followed organizers) — in the Swiss Grid system on top of the Phase 1 primitives and Phase 2 patterns.

**Architecture:** Screens are thin server shells that render a client body; the client body owns `AppHeader` when the header needs live data (U4's month caption). All new logic lands in pure, unit-tested helpers (`lib/geo.ts`, `lib/map-stats.ts`, `lib/rsvp-labels.ts`, `lib/member-since.ts`, additions to `lib/calendar.ts` / `lib/format.ts`); components stay declarative and are verified in the browser. The only shared-component change is a Yandex Maps upgrade (`components/map/YandexMap.tsx`) that adds square numbered ink markers, viewport reporting, and pin selection — no map-library migration (master plan decision checkpoint 3). One backend line is added (`created_at` on `/auth/me`) so U6 can print «Участник с марта 2026».

**Tech Stack:** Next.js 16 App Router / React 19 / TypeScript, Tailwind v4 (Swiss Grid tokens from Phase 1), TanStack Query 5, Yandex Maps JS API v2.1, Vitest 4 (node-only — pure helpers only, no DOM), Go monolith backend (one handler line).

**Spec:** `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` — screens **U4** (`/calendar`), **U5** (`/map`), **U6** (`/me`). Visual reference: `Presence Swiss Grid - Full System.dc.html`, badges `U4 · Календарь` (line ~301), `U5 · Карта` (line ~360), `U6 · Мои записи и профиль` (line ~409). Marker/tile treatment: `map-embed.html`.

**Prerequisite:** Phase 2 (`redesign/swiss-grid-p2`, tip `13253c7`) is reviewed, merged to `main`, and deployed. Phase 3 branches from that merge commit. Verify before starting:

```bash
git log --oneline -1            # must be the Phase 2 merge, not 49773a4
ls frontend/app/login frontend/app/not-found.tsx frontend/components/AuthForm.tsx
```

Run in an isolated worktree (`superpowers:using-git-worktrees`), branch `redesign/swiss-grid-p3`. Working directory for all `pnpm` commands: `frontend/`. Working directory for `go` commands: `backend/`.

---

## Global Constraints

Everything from the master plan (`2026-07-28-swiss-grid-redesign-master-plan.md`) applies. The ones this phase trips over constantly:

- **Zero border radius, zero shadows.** Rules are 1px solid: `#111` (`border-ink` / `border-on-surface`) for structural divisions, `#DDD` (`border-rule-inner`) for inner separations, `#E0DCD4` (`border-rule-grid`) **for calendar cell rules only**.
- **Categories are numerals, never colours** — via `categoryNumeral(slug, orderedCategories)`. **Map/list numerals are different**: they are the *positional index in the current result list* (`01`…`NN`), and the list numeral must equal the pin numeral.
- **All numbers in JetBrains Mono** (`font-mono`): dates, times, counts, distances, seat counters, radii. Unknown numbers render `—`.
- **`signal` red only for needs-attention.** A *selected* map pin or a *selected* calendar day is NOT red — selection inverts (paper fill / ink text / ink hairline).
- Uppercase runs keep tracking: `.cap` 0.13em, `.lbl` 0.14em, `.kick` 0.18em, chip 0.12em, button 0.07em, nav 0.14em, map pill 0.1em.
- Hover **inverts** (`hover-invert`), focus is `swiss-focus` (2px square outline), transitions 120ms linear on background/color only. No motion on layout, no spinners — `Skeleton` boxes at final dimensions.
- Every empty / loading / gated / error surface is designed: `EmptyState` (U8 pattern), `AuthGate`, `Skeleton`.
- **Tap targets ≥44px on touch.** The prototype's 26px mobile calendar rows and 9px list paddings must grow — the handoff explicitly says so (README → Responsive). Where this plan's numbers differ from the prototype, the 44px rule wins.
- Content caps at 1360px; 48px gutters desktop, 20px below 900px; layouts collapse below 720px (`max-sm:` = Tailwind's 640px breakpoint is what the codebase uses — keep using `sm:`/`max-sm:`/`md:`/`max-md:` consistently with Phase 2).
- Preserve: React #418 single-`<Link>` rule (never nest an `<a>` inside an `<a>`, never put an `<a>` inside a `<button>`), Europe/Moscow-pinned formatters (every new date helper pins `timeZone: "Europe/Moscow"` or operates on UTC-midnight civil dates), `NEXT_PUBLIC_*` build args declared as `ARG` in `frontend/Dockerfile`.
- All UI copy Russian (handoff copy is final); code, comments and commit messages English.
- Every commit must be green: `pnpm build && pnpm test && pnpm lint` from `frontend/` (plus `go build ./... && go test ./...` from `backend/` for Task 1).

## Deliberate deviations from the handoff (pre-decided — do not "fix" backwards)

1. **U6 «Избранное» tab → «Заявки».** There is no favorites backend (master plan, Known backend gaps). The four tabs are **Предстоящие · Прошедшие · Заявки · Подписки**; «Заявки» carries the content of the retired `/me/applications` page (statuses applied / accepted / declined / withdrawn).
2. **U6 is read-only.** The handoff calls the profile "a table of facts"; the 4th cell holds the status chip and nothing else. Cancelling a registration / withdrawing an application therefore happens on the event page (`SignupCTA` already renders «Отменить запись» / «Покинуть лист» / withdraw for `going`/`waitlist`/`applied`), and leaving feedback on a past event happens on the event page too (Phase 2 put `FeedbackForm` there for ended events). No feature is lost; each costs one extra click. **Flag this to the user at review time** — if they object, the fallback is a 5th 92px action column with a small ghost «ОТМЕНИТЬ», which breaks the handoff's column widths.
3. **U4 view modes are «Месяц» / «Список».** The handoff month bar shows chips `Месяц / Список / ← / →`; the existing page's «Неделя» / «День» segmented control is retired (with `Segmented` left in place — `CreateEventForm` still uses it until Phase 5). «Список» is a flat month agenda.
4. **U4 «ДОБАВИТЬ В КАЛЕНДАРЬ»** links to the `.ics` of the selected day's **earliest** event (`eventCalendarUrl(id)`), with the event title in the `title` attribute; hidden when the day is empty. There is no multi-event `.ics` endpoint, and per-row calendar links would need anchors inside anchors. Per-event `.ics` + Google links remain on the event page (`SignupCTA` footer).
5. **U5 stays on Yandex Maps 2.1** (master plan checkpoint 3): grayscale via a CSS `filter` on the map container, square ink numbered markers via `ymaps.templateLayoutFactory`, zoom controls hidden. Yandex's own copyright stays visible (their licence requires it); the handoff's OSM attribution note does not apply.
6. **U5 stats reflect the last search, not the live viewport** — except «Радиус», which reports the radius the *current* viewport would search. Recomputing «Всего / Бесплатно / Сегодня» on every pan would re-render the pin layer on every mouse move. The `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill is what commits the viewport to a new search.
7. **No bbox endpoint exists.** `GET /events/nearby?lat&lon&limit` is radius-based with a hard-coded 50 km in SQL (`backend/internal/events/repository.go:307-349`). U5 searches at the viewport centre and filters client-side; a real `bbox` query is out of scope (note it in the phase report as backend follow-up).
8. **`CellStrip` dividers move from `rule-inner` to `on-surface`** (Task 3, Step 1) — the reference markup divides cells in a strip with `.r` = `1px solid #111`, not `#DDD` (`Presence Swiss Grid - Full System.dc.html:369-374` for U5, `:430-441` for U6). This retroactively corrects U2's fact strip, which is an improvement, not a regression.
9. **Fonts stay Golos Text / Manrope / JetBrains Mono** (Phase 1 decision, `app/globals.css:84-86`). The handoff's Archivo and Space Grotesk have **no Cyrillic coverage** on Google Fonts, and every string in this product is Russian. `--font-ui` = Golos Text substitutes Archivo, `--font-alt` = Manrope substitutes Space Grotesk. Consequence to respect: **Manrope has no 900 weight** — `font-alt` text must never ask for `font-black`; the 900 weights in the reference (`.ttl`, `.wm`, price values) are `--font-ui` and render correctly.
10. **U5 keeps a list below the card on mobile.** The reference mobile frame is stat strip → map → one selected card → tab bar, and its header caption reads `СПИСКОМ`, i.e. the design assumes a *toggle* to a list screen (`Presence Map Screens.html:125` has the same `СПИСКОМ` CTA on desktop). Rather than build a second screen, the list is stacked under the card and the mobile caption is `КАРТА`. Nothing else about the mobile frame changes.
11. **U6 shows all four tab chips on mobile** (the reference mobile frame shows only `Предстоящие / Прошедшие`, `:459`). The 262px mock has no room for four; a real 390px viewport does with horizontal scroll. Dropping two chips on mobile would strand «Заявки» and «Подписки» with no route to them at all, since Task 3 retires `/me/applications`.

## Design fidelity contract

`Presence Swiss Grid - Full System.dc.html` is the pixel authority for these three screens; `README.md` is the authority for behaviour and for anything the mock only gestures at. The mock's shared classes translate to our primitives like this:

| Reference class | Definition (`:20-49`) | Our equivalent |
|---|---|---|
| `.cap` | 9px / 400 / 0.13em / caps / `#8A857C` / lh 1.3, **body face** | `cap` utility (see the font note below) |
| `.val` | 700 / 12px / lh 1.25 | `text-[12px] font-bold leading-[1.25]` (the `Cell` value) |
| `.num` | mono / 700 | `font-mono font-bold` |
| `.ttl` | 900 / −0.03em / lh 0.94 | `font-black tracking-[-0.03em] leading-[0.94]` |
| `.chip` | 9px / 0.12em / caps / 1px border / 4px 9px, **body face** | `Chip` default (see the font note below) |
| `.chip.on` | `#111` fill, white text | `Chip variant="active"` |
| `.cta` / `.cta.gh` | ink fill / ghost, 11px / 700 / 0.07em / 11px pad | `Button` primary / ghost |
| `.cell` | 10px 14px | `Cell` default padding |
| `.t` `.b` `.r` | 1px solid `#111` | `border-ink` (page-level) / `border-on-surface` (inside a surface-blind component) |
| `#e0dcd4` cell rules | `--rule-grid` | `border-rule-grid` — **calendar day cells only** |
| `#eceae4` blank cells | `--cell-blank` | `bg-cell-blank` |
| `.lbl` `.kick` `.note` `.sechd` `.badge` | **alt face** (Space Grotesk), explicitly set | `lbl` / `kick` utilities on `--font-alt` |

**Font-face note (fixed in Task 3, Step 1).** The reference sets `font-family` on exactly five classes — `.lbl`, `.kick`, `.note`, `.sechd`, `.badge` — and lets everything else inherit the body face. So `.cap`, `.chip`, `.val` and `.ttl` are all the **primary** face, and only the editorial furniture switches to the alt face. Phase 1 put `cap` and `Chip` on `--font-alt`, which is one substitution too far: it makes every caption and every chip in the app read in Manrope where the mock reads in the body face. Task 3 Step 1 moves both back to `--font-ui`; `lbl` and `kick` correctly stay on `--font-alt`. This retroactively touches U1/U2/U7/U8, which is why Task 10 carries a Phase 1–2 regression pass.

Values the mock overrides inline, which the code must therefore override too (these are the ones easiest to get wrong):

**U4 (`:301-358`)** — desktop: month bar `14px 20px`, title 26px; grid cells `5px 7px`, date 10px mono, category tag 8px/700/0.06em pinned bottom; rail cells `11px 14px`, selected-day value **16px/900**, `Записан` chip `8px` / `2px 6px`, agenda title 12.5px/700/lh 1.1, venue `.cap` `mt-4px`. Mobile: weekday captions `5px 0` at **7.5px**, day numerals 9px, agenda rows `9px 14px` with **9px** mono times and 11.5px titles.

**U5 (`:360-407`)** — desktop: stat values `.val.num` = **12px** (not 26px — these are `Cell` defaults, not hero numbers); rail filter row `8px 14px` gap 5px with **8px** chips; rail rows `9px 14px`, numeral 10px mono, title 12px/700/lh 1.1; pill `6px 12px`, 9px/700/0.1em, `left:50%; top:12px`, `z-index:500`, `width:max-content`. Mobile: stat cells `8px 12px` with **11px** values; card `11px 14px`, mono `01 · 0.8 КМ` 10px, price 11px/900, title 13px/700/lh 1.05.

**U6 (`:409-468`)** — desktop: identity left `16px 20px`, `.cap mb-6px`, name 30px; stat values **15px**; tab row `9px 20px` gap 6px; rows `56px 1fr 118px 134px`, date 11px mono centred, title 13px/700/lh 1.1, `.cap mt-3px`, organizer `.val` 11px, status cell `10px 8px` with a chip at **8px** / `3px 7px`; closing note `16px 20px` centred. Mobile: identity `13px 14px`, `.cap mb-5px`, name 20px; three stat cells `8px 10px` with **12px** values; tab row `8px 14px` gap 5px with 8px chips; rows `10px 14px`, mono `12.07 · 16:00` 10px, status chip **7px** / `2px 5px`, title 12px/700/lh 1.1.

Where the 44px tap-target rule (README → Responsive) collides with a mock number, the mock's *type and rules* still win — only the row height grows. That is why the mobile calendar grid is 44px instead of 26px and mobile list rows get `min-h-[44px]`, while every font size, border colour and horizontal padding above is honoured exactly.

## File Structure

```
backend/
  internal/http/admin/handler.go              MODIFY — add "created_at" to the /auth/me payload

frontend/
  lib/api.ts                                  MODIFY — getMe() returns createdAt
  lib/member-since.ts                          CREATE — "Участник с марта 2026"
  lib/rsvp-labels.ts                           CREATE — RsvpStatus → { long, short } RU labels
  lib/format.ts                                MODIFY — add formatShortDate ("12.07")
  lib/calendar.ts                              MODIFY — add monthTitle, monthCaption, selectedDayLabel, monthGridTrimmed
  lib/geo.ts                                   CREATE — haversineKm, radiusKmFromBounds, radiusLabel, distanceLabel
  lib/map-stats.ts                             CREATE — mapAreaStats()
  lib/__tests__/member-since.test.ts            CREATE
  lib/__tests__/rsvp-labels.test.ts             CREATE
  lib/__tests__/format-short-date.test.ts       CREATE
  lib/__tests__/calendar-labels.test.ts         CREATE
  lib/__tests__/geo.test.ts                     CREATE
  lib/__tests__/map-stats.test.ts               CREATE
  components/ui/Cell.tsx                       MODIFY — CellStrip divider → border-on-surface; Cell gains valueClassName
  components/ui/Chip.tsx                       MODIFY — drop font-alt (the reference's .chip is the body face)
  components/ui/AppHeader.tsx                  MODIFY — add «Профиль» (/me) to USER_NAV
  components/ui/BottomTabBar.tsx               MODIFY — «Я» tab targets /me
  components/MeProfile.tsx                     CREATE — U6 client body
  app/me/page.tsx                              CREATE — U6 shell
  app/me/practices/page.tsx                    REPLACE — permanent redirect → /me
  app/me/applications/page.tsx                 REPLACE — permanent redirect → /me?tab=applications
  components/CalendarView.tsx                  CREATE — U4 client body (owns AppHeader for the live month caption)
  app/me/calendar/page.tsx                     REPLACE — thin shell rendering <CalendarView />
  app/globals.css                              MODIFY — cap utility → --font-ui (Task 3); .swiss-pin marker styles (Task 7)
  components/map/YandexMap.tsx                 MODIFY — numbered square pins, hideControls, onViewportChange, onPinClick, activePinId
  components/MapBrowse.tsx                     REWRITE — U5 layout
  app/map/page.tsx                             REPLACE — AppHeader + MapBrowse shell
  components/ui/SearchField.tsx                DELETE — zero consumers since Phase 2
```

Not touched: `lib/api.ts` data functions other than `getMe`, `auth-context.tsx`, `SignupCTA`, `EventDetailView`, `DiscoveryFeed`, admin/organizer screens.

---

### Task 1: `/auth/me` returns `created_at` + `memberSince()` helper (TDD)

U6's identity strip opens with «Участник с марта 2026». The registration date exists in the DB (`internal/models/user.go:24` `CreatedAt time.Time`) and is loaded on every authentication (`internal/http/auth/auth.go:96` → `ensureUser` → `GetUserByEmail`), but `/auth/me` does not serialize it.

**Files:**
- Modify: `backend/internal/http/admin/handler.go` (the `me` handler, lines 92–101)
- Modify: `frontend/lib/api.ts` (`getMe`, lines 600–617)
- Create: `frontend/lib/member-since.ts`
- Test: `frontend/lib/__tests__/member-since.test.ts`

**Interfaces:**
- Produces: `GET /auth/me` JSON gains `created_at` (RFC3339 string, Go `time.Time` default marshalling).
- Produces: `getMe(): Promise<{ id: string; email: string; name: string; role: string; emailVerified: boolean; createdAt?: string } | null>` — additive, existing consumers (`lib/auth-context.tsx`) keep compiling.
- Produces: `memberSince(iso: string | null | undefined): string | null` — `"Участник с марта 2026"`, `null` when the input is missing, unparseable, or the Go zero time.

- [ ] **Step 1: Write the failing test** — `frontend/lib/__tests__/member-since.test.ts`

```ts
import { describe, expect, it } from "vitest";
import { memberSince } from "../member-since";

describe("memberSince", () => {
  it("renders the genitive month and year", () => {
    expect(memberSince("2026-03-14T10:00:00Z")).toBe("Участник с марта 2026");
    expect(memberSince("2026-07-01T00:00:00Z")).toBe("Участник с июля 2026");
  });
  it("uses the Moscow civil month at the boundary", () => {
    // 2026-01-31T22:00Z is already 2026-02-01 01:00 in Moscow (UTC+3).
    expect(memberSince("2026-01-31T22:00:00Z")).toBe("Участник с февраля 2026");
  });
  it("returns null for missing / zero / unparseable input", () => {
    expect(memberSince(undefined)).toBeNull();
    expect(memberSince(null)).toBeNull();
    expect(memberSince("")).toBeNull();
    expect(memberSince("0001-01-01T00:00:00Z")).toBeNull();
    expect(memberSince("not a date")).toBeNull();
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `pnpm vitest run lib/__tests__/member-since.test.ts`
Expected: FAIL — `Failed to resolve import "../member-since"`.

- [ ] **Step 3: Implement** — `frontend/lib/member-since.ts`

```ts
// Genitive month names: Intl's ru-RU "long" month is nominative and appends
// "г." to the year ("март 2026 г."), which is not the handoff string.
const MONTHS_GENITIVE = [
  "января", "февраля", "марта", "апреля", "мая", "июня",
  "июля", "августа", "сентября", "октября", "ноября", "декабря",
] as const;

// en-CA yields "YYYY-MM-DD"; pinned to Moscow like every other formatter.
const moscowDayFmt = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone: "Europe/Moscow",
});

/** U6 identity caption: "Участник с марта 2026". Null when unknown — the
 * caller drops the caption rather than printing a placeholder date. */
export function memberSince(iso: string | null | undefined): string | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  // The backend serializes an unset time.Time as year 0001 (same rule as
  // ends_at in lib/api.ts) — treat anything pre-1971 as "unknown".
  if (d.getUTCFullYear() <= 1970) return null;
  const [year, month] = moscowDayFmt.format(d).split("-").map(Number);
  return `Участник с ${MONTHS_GENITIVE[month - 1]} ${year}`;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm vitest run lib/__tests__/member-since.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Add `created_at` to the backend payload**

In `backend/internal/http/admin/handler.go`, the `me` handler currently writes:

```go
	writeJSON(w, http.StatusOK, map[string]any{
		"id": u.UUID.String(), "email": u.Email, "name": u.Name, "role": u.Role,
		"email_verified": u.EmailVerified,
	})
```

Change it to:

```go
	writeJSON(w, http.StatusOK, map[string]any{
		"id": u.UUID.String(), "email": u.Email, "name": u.Name, "role": u.Role,
		"email_verified": u.EmailVerified,
		// Registration month for the profile identity strip (U6).
		"created_at": u.CreatedAt,
	})
```

- [ ] **Step 6: Map it in the frontend client**

In `frontend/lib/api.ts`, extend `getMe`'s return type and mapping:

```ts
export async function getMe(): Promise<{
  id: string;
  email: string;
  name: string;
  role: string;
  emailVerified: boolean;
  /** RFC3339 registration timestamp; absent on older backends. */
  createdAt?: string;
} | null> {
  const res = await fetch(`${API_BASE}/auth/me`, { headers: authHeaders(), cache: "no-store" });
  if (!res.ok) return null;
  const data = await res.json();
  return {
    id: data.id,
    email: data.email,
    name: data.name,
    role: data.role,
    emailVerified: !!data.email_verified,
    createdAt: typeof data.created_at === "string" ? data.created_at : undefined,
  };
}
```

- [ ] **Step 7: Verify both sides build**

Run (from `backend/`): `go build ./... && go test ./...`
Expected: PASS.
Run (from `frontend/`): `pnpm build && pnpm test && pnpm lint`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/http/admin/handler.go frontend/lib/api.ts frontend/lib/member-since.ts frontend/lib/__tests__/member-since.test.ts
git commit -m "feat: expose user created_at on /auth/me and add memberSince() label"
```

---

### Task 2: U6 helpers — RSVP status labels + `formatShortDate` (TDD)

**Files:**
- Create: `frontend/lib/rsvp-labels.ts`
- Modify: `frontend/lib/format.ts` (append only — do not touch existing exports)
- Test: `frontend/lib/__tests__/rsvp-labels.test.ts`, `frontend/lib/__tests__/format-short-date.test.ts`

**Interfaces:**
- Consumes: `RsvpStatus` (`lib/types.ts:18-25` — `"going" | "waitlist" | "applied" | "accepted" | "declined" | "withdrawn" | "cancelled"`), `statusChipVariant(status: string): "active" | "default" | "signal"` (`lib/status-chip.ts`).
- Produces: `rsvpStatusLabel(status: RsvpStatus): { long: string; short: string }` — `long` feeds `<StatusChip status={long} />` on desktop, `short` feeds the compact mobile chip. Unknown values fall back to `{ long: "—", short: "—" }`.
- Produces: `formatShortDate(iso: string): string` — `"12.07"`, Moscow civil day.

**Chip-tone contract (why the labels are what they are):** `statusChipVariant` already maps «Подтверждено» → `active` and «Ожидает» → `signal`; every other label falls through to `default`. Red therefore appears only on `applied` («Ожидает» — the organizer owes the user an answer), which matches the handoff's enumeration of red occurrences. Waitlist/declined/withdrawn/cancelled are legitimate, non-actionable states and stay outline chips. `lib/status-chip.ts` needs **no** change; the test below locks that in.

- [ ] **Step 1: Write the failing tests**

`frontend/lib/__tests__/rsvp-labels.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { rsvpStatusLabel } from "../rsvp-labels";
import { statusChipVariant } from "../status-chip";
import type { RsvpStatus } from "../types";

describe("rsvpStatusLabel", () => {
  it("confirmed states share the Подтверждено label", () => {
    expect(rsvpStatusLabel("going")).toEqual({ long: "Подтверждено", short: "ОК" });
    expect(rsvpStatusLabel("accepted")).toEqual({ long: "Подтверждено", short: "ОК" });
  });
  it("pending states", () => {
    expect(rsvpStatusLabel("applied")).toEqual({ long: "Ожидает", short: "ЖДЁМ" });
    expect(rsvpStatusLabel("waitlist")).toEqual({ long: "В листе ожидания", short: "ЛИСТ" });
  });
  it("closed states", () => {
    expect(rsvpStatusLabel("declined")).toEqual({ long: "Отклонена", short: "НЕТ" });
    expect(rsvpStatusLabel("withdrawn")).toEqual({ long: "Отозвана", short: "ОТОЗВ" });
    expect(rsvpStatusLabel("cancelled")).toEqual({ long: "Отменено", short: "ОТМ" });
  });
  it("unknown status degrades to em dashes", () => {
    expect(rsvpStatusLabel("nonsense" as RsvpStatus)).toEqual({ long: "—", short: "—" });
  });
});

describe("rsvp label → chip tone", () => {
  it("only «Ожидает» is red; confirmed is ink-filled; the rest are outlines", () => {
    expect(statusChipVariant(rsvpStatusLabel("applied").long)).toBe("signal");
    expect(statusChipVariant(rsvpStatusLabel("going").long)).toBe("active");
    expect(statusChipVariant(rsvpStatusLabel("accepted").long)).toBe("active");
    for (const s of ["waitlist", "declined", "withdrawn", "cancelled"] as RsvpStatus[]) {
      expect(statusChipVariant(rsvpStatusLabel(s).long)).toBe("default");
    }
  });
});
```

`frontend/lib/__tests__/format-short-date.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { formatShortDate } from "../format";

describe("formatShortDate", () => {
  it("renders DD.MM in Moscow", () => {
    expect(formatShortDate("2026-07-12T13:00:00Z")).toBe("12.07");
  });
  it("uses the Moscow civil day across the UTC midnight boundary", () => {
    // 22:00Z on the 11th is already 01:00 on the 12th in Moscow.
    expect(formatShortDate("2026-07-11T22:00:00Z")).toBe("12.07");
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `pnpm vitest run lib/__tests__/rsvp-labels.test.ts lib/__tests__/format-short-date.test.ts`
Expected: FAIL — `../rsvp-labels` unresolved, `formatShortDate` is not a function.

- [ ] **Step 3: Implement** — `frontend/lib/rsvp-labels.ts`

```ts
import type { RsvpStatus } from "./types";

export interface RsvpLabel {
  /** Desktop chip text; routed through statusChipVariant() for its tone. */
  long: string;
  /** Compact mobile chip text (U6 mobile rows show «ОК» / «ЖДЁМ»). */
  short: string;
}

const LABELS: Record<RsvpStatus, RsvpLabel> = {
  going: { long: "Подтверждено", short: "ОК" },
  accepted: { long: "Подтверждено", short: "ОК" },
  applied: { long: "Ожидает", short: "ЖДЁМ" },
  waitlist: { long: "В листе ожидания", short: "ЛИСТ" },
  declined: { long: "Отклонена", short: "НЕТ" },
  withdrawn: { long: "Отозвана", short: "ОТОЗВ" },
  cancelled: { long: "Отменено", short: "ОТМ" },
};

const UNKNOWN: RsvpLabel = { long: "—", short: "—" };

/** RSVP status → the two Russian labels U6 renders. «Ожидает» is the only one
 * that resolves to a red chip (the organizer owes the user an answer). */
export function rsvpStatusLabel(status: RsvpStatus): RsvpLabel {
  return LABELS[status] ?? UNKNOWN;
}
```

Append to `frontend/lib/format.ts` (after `attendanceShort`; `shortDateFmt` already exists at line 86 — reuse it, do not declare a second formatter):

```ts
/** "12.07" — Moscow civil day, for the U6 date column. */
export function formatShortDate(iso: string): string {
  return shortDateFmt.format(new Date(iso));
}
```

- [ ] **Step 4: Run the full suite**

Run: `pnpm test`
Expected: all suites PASS (previous suites + 2 new files).

- [ ] **Step 5: Commit**

```bash
git add lib/rsvp-labels.ts lib/format.ts lib/__tests__/rsvp-labels.test.ts lib/__tests__/format-short-date.test.ts
git commit -m "feat(frontend): RSVP status labels + short date helper for U6"
```

---

### Task 3: U6 `/me` — consolidated profile

**Files:**
- Modify: `frontend/components/ui/Cell.tsx` (`CellStrip` divider colour — deviation 8; `valueClassName`), `frontend/components/ui/Chip.tsx` + `frontend/app/globals.css` (caption/chip typeface — fidelity contract)
- Create: `frontend/components/MeProfile.tsx`
- Create: `frontend/app/me/page.tsx`
- Replace: `frontend/app/me/practices/page.tsx`, `frontend/app/me/applications/page.tsx` (redirects)
- Modify: `frontend/components/ui/AppHeader.tsx` (`USER_NAV` gains «Профиль»), `frontend/components/ui/BottomTabBar.tsx` («Я» → `/me`)

**Interfaces:**
- Consumes: `AppHeader` + `USER_NAV` + `AuthNavControl`, `Cell`, `Chip`, `StatusChip`, `EmptyState`, `Skeleton`, `AuthGate`, `useAuth()` → `{ isAuthed, ready }`, `getMe()` (Task 1), `fetchMyPractices(tab: "upcoming" | "past")`, `fetchMyApplications(status?: string)`, `fetchFollowedOrganizers()`, `memberSince` (Task 1), `rsvpStatusLabel` + `formatShortDate` (Task 2), `formatStartTime`, `pluralRu`.
- Produces: route `/me`; `MeProfile` is self-contained (no props). Later phases and the tab bar link to `/me` and `/me?tab=applications`.

Reference markup: `Presence Swiss Grid - Full System.dc.html:409-468`. Desktop identity strip `1fr 200px`, tab chip row, rows on `56px 1fr 118px 134px`, inline empty note. Mobile: identity block, three stat cells, chip row, compact rows.

- [ ] **Step 1: Fix the shared primitives against the reference** — `frontend/components/ui/Cell.tsx`, `frontend/components/ui/Chip.tsx`, `frontend/app/globals.css`

Four small corrections, all of them things the mock states and Phase 1 approximated. Do them first: Tasks 5 and 8 build on the corrected primitives.

**(a) Captions and chips move to the primary face.** Per the font-face note in the fidelity contract, the reference only switches typeface on `.lbl` / `.kick` / `.note` / `.sechd` / `.badge`. In `app/globals.css`, change the `cap` utility's family (leave `lbl` and `kick` alone):

```css
/* Caption: 9px / 0.13em / uppercase / muted-2, primary face — the reference
 * sets no font-family on .cap, so it inherits the body face. */
@utility cap {
  font-family: var(--font-ui);
  ...
}
```

And in `components/ui/Chip.tsx`, drop `font-alt` from the base class (the reference's `.chip` has no `font-family` either):

```tsx
  const base = cn(
    "inline-flex items-center whitespace-nowrap border px-[9px] py-[4px] text-[9px] uppercase tracking-[0.12em]",
    VARIANTS[variant],
    as === "button" && "cursor-pointer swiss-focus hover-invert",
    className,
  );
```

**(b) `CellStrip` dividers become structural.** Replace the `CellStrip` divider class (`[&>*+*]:border-rule-inner`) with the structural rule the reference uses:

```tsx
/** Horizontal strip of cells divided by hairlines, closed by a bottom rule.
 * Dividers are structural (#111 on paper / #F2F0EC on ink), matching the
 * reference's `.r` class — not the lighter inner-separation rule. */
export function CellStrip({ children, cols, className }: { children: ReactNode; cols: number; className?: string }) {
  return (
    <div
      className={cn("grid border-b border-on-surface [&>*+*]:border-l [&>*+*]:border-on-surface", className)}
      style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}
    >
      {children}
    </div>
  );
}
```

**(c) `Cell` gets a value-size override.** `Cell` hard-codes its value at `.val`'s 12px, but the reference overrides that size per screen (U6 desktop 15px / mobile 12px, U5 desktop 12px / mobile 11px — see the fidelity contract). Add one prop rather than letting callers wrap the value:

```tsx
export interface CellProps {
  caption: ReactNode;
  value: ReactNode;
  /** Numeric value → JetBrains Mono (handoff: ALL numbers in mono). */
  mono?: boolean;
  /** Roomy 16/20 padding instead of dense 10/14. */
  roomy?: boolean;
  /** Ink-filled emphasis cell (e.g. O1 «На модерации»). */
  invert?: boolean;
  /** Per-screen size override on the value; the reference sets it inline
   * (U6 15px, U5 mobile 11px). Merged last, so `text-[15px]` wins over 12px. */
  valueClassName?: string;
  className?: string;
}

export function Cell({ caption, value, mono, roomy, invert, valueClassName, className }: CellProps) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-col gap-[4px]",
        roomy ? "px-[20px] py-[16px]" : "px-[14px] py-[10px]",
        invert && "bg-on-surface text-surface",
        className,
      )}
    >
      <span className={cn("cap", invert && "text-text-dim-dark-2")}>{caption}</span>
      <span className={cn("text-[12px] font-bold leading-[1.25]", mono && "font-mono", valueClassName)}>
        {value}
      </span>
    </div>
  );
}
```

**(d) Verify the retroactive reach.** (a) and (b) change screens shipped in Phases 1–2, so check them before moving on: `pnpm dev`, then the feed `/`, an event page (`/events/[id]` — its fact strip is the only `CellStrip` consumer, and must now be divided by ink hairlines rather than `#DDD`), `/login`, and a 404. Captions and chips should read in the body face; nothing should shift position, since Golos Text and Manrope have close metrics at 9px. `Cell`'s new prop is optional and unused by those screens, so it cannot affect them.

- [ ] **Step 2: Add «Профиль» to the desktop nav** — `frontend/components/ui/AppHeader.tsx`

```tsx
export const USER_NAV: NavItem[] = [
  { href: "/", label: "События" },
  { href: "/search", label: "Подбор" },
  { href: "/me/calendar", label: "Календарь" },
  { href: "/map", label: "Карта" },
  { href: "/me", label: "Профиль" },
  { href: "/organizer", label: "Организаторам" },
];
```

`/me` is a prefix of `/me/calendar`, so on the calendar route both items match — `AppHeader`'s longest-prefix rule (already implemented, lines 32–34) correctly underlines «Календарь» only. Verify in the browser on both routes.

- [ ] **Step 3: Retarget the mobile «Я» tab** — `frontend/components/ui/BottomTabBar.tsx`

```tsx
const TABS = [
  { href: "/", label: "Лента" },
  { href: "/search", label: "Подбор" },
  { href: "/map", label: "Карта" },
  { href: "/me", label: "Я" },
] as const;

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}
```

(The `/me/practices` special case in the old `isActive` goes away: `/me` is now a real prefix of every `/me/*` route, so the generic rule covers it.)

- [ ] **Step 4: Implement the client body** — `frontend/components/MeProfile.tsx`

```tsx
"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthGate } from "@/components/ui/AuthGate";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { Cell } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import {
  fetchFollowedOrganizers,
  fetchMyApplications,
  fetchMyPractices,
  getMe,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatShortDate, formatStartTime } from "@/lib/format";
import { memberSince } from "@/lib/member-since";
import { rsvpStatusLabel } from "@/lib/rsvp-labels";
import type { FollowedOrganizer, Rsvp } from "@/lib/types";

type Tab = "upcoming" | "past" | "applications" | "follows";

const TABS: { key: Tab; label: string }[] = [
  { key: "upcoming", label: "Предстоящие" },
  { key: "past", label: "Прошедшие" },
  { key: "applications", label: "Заявки" },
  { key: "follows", label: "Подписки" },
];

function isTab(value: string | null): value is Tab {
  return value === "upcoming" || value === "past" || value === "applications" || value === "follows";
}

/** Count for a tab chip: "—" until the list has loaded (numbers never guess). */
function countLabel(n: number | undefined): string {
  return n == null ? "—" : String(n);
}

/** U6 registration row: mono date · title+context · organizer · status.
 * The whole row is ONE <Link> — nothing inside may be an anchor (React #418). */
function RegistrationRow({ rsvp }: { rsvp: Rsvp }) {
  const event = rsvp.event;
  const status = rsvpStatusLabel(rsvp.status);
  const context = event
    ? [event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—"), formatStartTime(event.startsAt)]
        .filter(Boolean)
        .join(" · ")
    : "Событие недоступно";

  return (
    <Link
      href={`/events/${rsvp.eventId}`}
      className="block border-b border-on-surface swiss-focus hover-invert"
    >
      {/* Desktop — 56px 1fr 118px 134px */}
      <div className="hidden grid-cols-[56px_1fr_118px_134px] items-center sm:grid">
        <span className="border-r border-on-surface px-[8px] py-[10px] text-center font-mono text-[11px] font-bold">
          {event ? formatShortDate(event.startsAt) : "—"}
        </span>
        <span className="border-r border-on-surface px-[14px] py-[10px]">
          <span className="block text-[13px] font-bold leading-[1.1]">
            {event?.title ?? `Событие #${rsvp.eventId.slice(0, 8)}`}
          </span>
          <span className="cap mt-[3px] block">{context}</span>
        </span>
        <span className="border-r border-on-surface px-[14px] py-[10px]">
          <span className="cap block">Организатор</span>
          <span className="mt-[2px] block truncate text-[11px] font-bold">
            {event?.organizer?.name || "—"}
            {event?.organizer?.verified ? " ✓" : ""}
          </span>
        </span>
        <span className="flex items-center justify-center px-[8px] py-[10px]">
          <StatusChip status={status.long} className="px-[7px] py-[3px] text-[8px]" />
        </span>
      </div>

      {/* Mobile — compact row, ≥44px tall */}
      <div className="flex min-h-[56px] flex-col justify-center gap-[3px] px-[14px] py-[10px] sm:hidden">
        <span className="flex items-baseline justify-between gap-[8px]">
          <span className="font-mono text-[10px] font-bold">
            {event ? `${formatShortDate(event.startsAt)} · ${formatStartTime(event.startsAt)}` : "—"}
          </span>
          <StatusChip status={status.long} className="px-[5px] py-[2px] text-[7px]">
            {status.short}
          </StatusChip>
        </span>
        <span className="text-[12px] font-bold leading-[1.1]">
          {event?.title ?? `Событие #${rsvp.eventId.slice(0, 8)}`}
        </span>
      </div>
    </Link>
  );
}

/** Subscriptions row: organizer name + «Открыть →», one <Link> per row. */
function FollowRow({ org }: { org: FollowedOrganizer }) {
  return (
    <Link
      href={`/organizers/${org.profileId}`}
      className="flex items-center justify-between gap-[10px] border-b border-on-surface px-[14px] py-[12px] swiss-focus hover-invert"
    >
      <span className="min-w-0">
        <span className="cap block">Организатор</span>
        <span className="mt-[2px] block truncate text-[13px] font-bold leading-[1.1]">{org.name}</span>
      </span>
      <span className="lbl shrink-0">Открыть →</span>
    </Link>
  );
}

function RowSkeletons() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 3 }, (_, i) => (
        <Skeleton key={i} className="h-[56px] w-full border-x-0 border-t-0" />
      ))}
    </div>
  );
}

export function MeProfile() {
  const { isAuthed, ready } = useAuth();
  const searchParams = useSearchParams();
  const requested = searchParams.get("tab");
  const [tab, setTab] = useState<Tab>(isTab(requested) ? requested : "upcoming");

  const authed = ready && isAuthed;

  const me = useQuery({ queryKey: ["me"], queryFn: getMe, enabled: authed });
  const upcoming = useQuery({
    queryKey: ["my-practices", "upcoming"],
    queryFn: () => fetchMyPractices("upcoming"),
    enabled: authed,
  });
  const past = useQuery({
    queryKey: ["my-practices", "past"],
    queryFn: () => fetchMyPractices("past"),
    enabled: authed,
  });
  const applications = useQuery({
    queryKey: ["my-applications", "all"],
    queryFn: () => fetchMyApplications(),
    enabled: authed,
  });
  const follows = useQuery({
    queryKey: ["my-follows"],
    queryFn: fetchFollowedOrganizers,
    enabled: authed,
  });

  const header = <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="ПРОФИЛЬ" />;

  if (!ready) {
    return (
      <>
        {header}
        <main className="mx-auto max-w-[1360px] px-[20px] py-[26px]">
          <Skeleton className="h-[120px] w-full" />
        </main>
      </>
    );
  }

  if (!isAuthed) {
    return (
      <>
        {header}
        <AuthGate
          title="Войдите, чтобы видеть свои записи"
          reassurance="Лента и карта доступны без входа."
        />
      </>
    );
  }

  const active =
    tab === "upcoming" ? upcoming
    : tab === "past" ? past
    : tab === "applications" ? applications
    : follows;

  const counts: Record<Tab, number | undefined> = {
    upcoming: upcoming.data?.length,
    past: past.data?.length,
    applications: applications.data?.length,
    follows: follows.data?.length,
  };

  const since = memberSince(me.data?.createdAt);

  return (
    <>
      {header}
      <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
        {/* Identity strip — 1fr 200px desktop */}
        <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
          <div className="border-r border-on-surface px-[20px] py-[16px] max-md:border-r-0 max-md:px-[14px] max-md:py-[13px]">
            {since ? <p className="cap mb-[6px] max-md:mb-[5px]">{since}</p> : null}
            <h1 className="text-[30px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[20px]">
              {me.data?.name?.trim() || me.data?.email || "Профиль"}
            </h1>
          </div>
          {/* Desktop: two stacked cells, 15px values. Mobile: three-cell strip
              below at 12px / 8px-10px padding (reference :423-426 / :454-458). */}
          <div className="grid grid-rows-2 max-md:hidden">
            <Cell
              caption="Посещено"
              value={countLabel(counts.past)}
              mono
              valueClassName="text-[15px]"
              className="border-b border-on-surface"
            />
            <Cell caption="Подписки" value={countLabel(counts.follows)} mono valueClassName="text-[15px]" />
          </div>
          <div className="grid grid-cols-3 border-b border-ink md:hidden [&>*+*]:border-l [&>*+*]:border-on-surface">
            <Cell caption="Посещено" value={countLabel(counts.past)} mono className="px-[10px] py-[8px]" />
            <Cell caption="Заявки" value={countLabel(counts.applications)} mono className="px-[10px] py-[8px]" />
            <Cell caption="Подписки" value={countLabel(counts.follows)} mono className="px-[10px] py-[8px]" />
          </div>
        </div>

        {/* Tab chips with counts — 9px/20px gap 6 desktop, 8px/14px gap 5 mobile */}
        <div className="flex gap-[6px] overflow-x-auto border-b border-ink px-[20px] py-[9px] max-sm:gap-[5px] max-sm:px-[14px] max-sm:py-[8px] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          {TABS.map((t) => (
            <Chip
              key={t.key}
              variant={tab === t.key ? "active" : "default"}
              onClick={() => setTab(t.key)}
              aria-pressed={tab === t.key}
              className="max-sm:text-[8px]"
            >
              {t.label} · {countLabel(counts[t.key])}
            </Chip>
          ))}
        </div>

        {/* Rows */}
        {active.isPending ? (
          <RowSkeletons />
        ) : active.isError ? (
          <EmptyState
            numeral="!"
            title="Не удалось загрузить данные"
            text="Проверьте соединение и попробуйте обновить страницу."
          />
        ) : tab === "follows" ? (
          follows.data && follows.data.length > 0 ? (
            <div className="flex flex-col">
              {follows.data.map((org) => (
                <FollowRow key={org.profileId} org={org} />
              ))}
            </div>
          ) : (
            <EmptyState
              title="Подписок пока нет"
              text="Подпишитесь на организатора — его события появятся в вашем календаре."
              actions={
                <Link
                  href="/"
                  className="swiss-focus border border-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover-invert"
                >
                  Найти события
                </Link>
              }
            />
          )
        ) : (active.data as Rsvp[] | undefined)?.length ? (
          <>
            <div className="flex flex-col">
              {(active.data as Rsvp[]).map((rsvp) => (
                <RegistrationRow key={rsvp.id} rsvp={rsvp} />
              ))}
            </div>
            <div className="px-[20px] py-[16px] text-center">
              <p className="cap mb-[8px]">Больше записей пока нет</p>
              <Link
                href="/"
                className="swiss-focus inline-flex items-center border border-ink px-[9px] py-[4px] text-[9px] uppercase tracking-[0.12em] hover-invert"
              >
                Найти события →
              </Link>
            </div>
          </>
        ) : (
          <EmptyState
            title={
              tab === "upcoming" ? "Записей пока нет"
              : tab === "past" ? "Прошедших событий пока нет"
              : "Заявок пока нет"
            }
            text={
              tab === "upcoming"
                ? "Когда вы запишетесь на событие, оно появится здесь."
                : tab === "past"
                  ? "Здесь появятся события, на которых вы побывали."
                  : "Заявки на события с отбором участников появятся здесь."
            }
            actions={
              <Link
                href="/"
                className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
              >
                Найти событие
              </Link>
            }
          />
        )}
      </main>
    </>
  );
}
```

Note on `StatusChip` in the mobile branch: it currently renders `{status}` as its own child and takes no `children`. **Extend it** so the mobile short label works — modify `frontend/components/ui/StatusChip.tsx`:

```tsx
import { Chip, type ChipVariant } from "@/components/ui/Chip";
import { statusChipVariant } from "@/lib/status-chip";
import type { ReactNode } from "react";

const TONE_TO_VARIANT: Record<ReturnType<typeof statusChipVariant>, ChipVariant> = {
  active: "active",
  default: "default",
  signal: "signal",
};

/** Status chip per the handoff map (published→ink fill, draft→outline,
 * moderation/waiting/test→signal). Non-interactive. `children` overrides the
 * visible text (U6 mobile shows «ОК» / «ЖДЁМ») while `status` still picks the tone. */
export function StatusChip({
  status,
  className,
  children,
}: {
  status: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <Chip as="span" variant={TONE_TO_VARIANT[statusChipVariant(status)]} className={className}>
      {children ?? status}
    </Chip>
  );
}
```

- [ ] **Step 5: Implement the route shell** — `frontend/app/me/page.tsx`

```tsx
import { Suspense } from "react";
import { MeProfile } from "@/components/MeProfile";

export const metadata = { title: "Профиль — PRESENCE" };

// U6 · Мои записи и профиль. MeProfile reads ?tab= via useSearchParams, which
// needs a Suspense boundary in the App Router.
export default function MePage() {
  return (
    <Suspense fallback={null}>
      <MeProfile />
    </Suspense>
  );
}
```

- [ ] **Step 6: Replace the two old pages with redirects**

`frontend/app/me/practices/page.tsx` (whole file):

```tsx
import { permanentRedirect } from "next/navigation";

// Consolidated into U6 (/me). Kept as a redirect so old links and bookmarks work.
export default function MyPracticesPage() {
  permanentRedirect("/me");
}
```

`frontend/app/me/applications/page.tsx` (whole file):

```tsx
import { permanentRedirect } from "next/navigation";

// Consolidated into U6 (/me), «Заявки» tab.
export default function MyApplicationsPage() {
  permanentRedirect("/me?tab=applications");
}
```

- [ ] **Step 7: Verify**

Run: `pnpm build && pnpm test && pnpm lint`
Expected: PASS. Fix any now-unused imports the build flags in the replaced files.

Browser (`pnpm dev`, signed in), desktop and 390px:
- `/me` — identity strip prints «Участник с … 2026» + name; Посещено/Подписки cells are mono; four tab chips carry counts and switch; rows are on the `56px 1fr 118px 134px` grid with ink dividers; hovering a row inverts it; the status chip is ink-filled for «Подтверждено» and red only for «Ожидает».
- 390px — identity block, three-cell stat strip, scrollable chip row, compact rows with mono `12.07 · 16:00` + short chip; rows ≥44px; the fixed tab bar does not cover the last row.
- Empty tabs render the U8 empty state; signed out renders `AuthGate` with «ВОЙТИ» → `/login?next=/me`.
- `/me/practices` → `/me`; `/me/applications` → `/me` with «Заявки» pre-selected.
- The mobile «Я» tab and the desktop «Профиль» nav item are both active on `/me`; on `/me/calendar` only «Календарь» is underlined.

- [ ] **Step 8: Commit**

Two commits — the shared-primitive corrections reach Phase 1–2 screens and should be revertable on their own:

```bash
git add components/ui/Cell.tsx components/ui/Chip.tsx app/globals.css
git commit -m "fix(frontend): align cap/chip typeface and cell strip rules with the handoff"

git add components/MeProfile.tsx app/me/page.tsx app/me/practices/page.tsx app/me/applications/page.tsx \
        components/ui/AppHeader.tsx components/ui/BottomTabBar.tsx components/ui/StatusChip.tsx
git commit -m "feat(frontend): U6 consolidated /me profile (registrations, applications, follows)"
```

---

### Task 4: U4 calendar helpers — month labels + trimmed grid (TDD)

**Files:**
- Modify: `frontend/lib/calendar.ts` (append; existing exports untouched)
- Test: `frontend/lib/__tests__/calendar-labels.test.ts`

**Interfaces:**
- Consumes: `civil(year, month0, day)`, `addDays`, `monthGrid`, `sameMonth`, `civilKey`, `WEEKDAY_LABELS` (all already exported from `lib/calendar.ts`).
- Produces: `monthTitle(anchor: Date): string` — `"Июль 2026"` (nominative, no «г.»; rendered in Archivo/Golos 900 at 26px per the reference's `.ttl`, **not** mono).
- Produces: `monthCaption(anchor: Date): string` — `"ИЮЛЬ ’26"` for the mobile header caption.
- Produces: `selectedDayLabel(day: Date): string` — `"12 июля, вс"`.
- Produces: `selectedDayLabelLong(day: Date): string` — `"12 июля, воскресенье"` (mobile caption strip).
- Produces: `monthGridTrimmed(anchor: Date): Date[]` — `monthGrid(anchor)` minus a trailing week that lies entirely outside the anchor month (35 or 42 cells).

- [ ] **Step 1: Write the failing tests** — `frontend/lib/__tests__/calendar-labels.test.ts`

```ts
import { describe, expect, it } from "vitest";
import {
  civil,
  civilKey,
  monthCaption,
  monthGridTrimmed,
  monthTitle,
  sameMonth,
  selectedDayLabel,
  selectedDayLabelLong,
} from "../calendar";

describe("monthTitle / monthCaption", () => {
  it("nominative month + full year", () => {
    expect(monthTitle(civil(2026, 6, 1))).toBe("Июль 2026");
    expect(monthTitle(civil(2026, 0, 1))).toBe("Январь 2026");
  });
  it("mobile caption is uppercase with an apostrophed year", () => {
    expect(monthCaption(civil(2026, 6, 1))).toBe("ИЮЛЬ ’26");
  });
});

describe("selectedDayLabel", () => {
  it("day, genitive month, short weekday", () => {
    // 12 July 2026 is a Sunday.
    expect(selectedDayLabel(civil(2026, 6, 12))).toBe("12 июля, вс");
    // 13 July 2026 is a Monday.
    expect(selectedDayLabel(civil(2026, 6, 13))).toBe("13 июля, пн");
  });
  it("long form spells the weekday out", () => {
    expect(selectedDayLabelLong(civil(2026, 6, 12))).toBe("12 июля, воскресенье");
  });
});

describe("monthGridTrimmed", () => {
  it("keeps Monday-first alignment and covers the whole month", () => {
    const cells = monthGridTrimmed(civil(2026, 6, 1));
    expect(cells.length % 7).toBe(0);
    expect(cells[0].getUTCDay()).toBe(1); // Monday
    expect(cells.some((c) => civilKey(c) === "2026-07-01")).toBe(true);
    expect(cells.some((c) => civilKey(c) === "2026-07-31")).toBe(true);
  });
  it("drops a trailing week that is entirely in the next month", () => {
    // June 2026 starts on a Monday with 30 days → exactly 5 rows, so
    // monthGrid()'s 6th row is all July and must be dropped.
    const june = civil(2026, 5, 1);
    const juneCells = monthGridTrimmed(june);
    expect(juneCells.length).toBe(35);
    expect(juneCells.slice(28).some((c) => sameMonth(c, june))).toBe(true);
  });
  it("keeps the sixth row when the month needs it", () => {
    // March 2026 starts on a Sunday (Monday index 6) with 31 days → 6 rows.
    const march = civil(2026, 2, 1);
    const marchCells = monthGridTrimmed(march);
    expect(marchCells.length).toBe(42);
    expect(marchCells.slice(35).some((c) => sameMonth(c, march))).toBe(true);
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `pnpm vitest run lib/__tests__/calendar-labels.test.ts`
Expected: FAIL — `monthTitle` / `monthCaption` / `selectedDayLabel` / `selectedDayLabelLong` / `monthGridTrimmed` are not exported.

The row counts above were verified against the real 2026 calendar: June 2026 starts on a Monday with 30 days (5 rows), March 2026 starts on a Sunday with 31 days (6 rows). If you change the fixtures, recompute rather than guess — `node -e 'const f=new Date(Date.UTC(2026,5,1)); console.log((f.getUTCDay()+6)%7)'` gives the Monday-based index of the 1st, and `rows = Math.ceil((index + daysInMonth) / 7)`. The contract is: `length % 7 === 0`, the whole month is covered, and no trailing week is entirely foreign.

- [ ] **Step 3: Implement** — append to `frontend/lib/calendar.ts`

```ts
// Nominative month names for the U4 month title. Intl's ru-RU long month is
// nominative too, but it renders the year as "2026 г." — the handoff does not.
const MONTHS_NOMINATIVE = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
] as const;

// "12 июля" — genitive day+month of a civil (UTC-midnight) date.
const dayMonthCivilFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "UTC",
});
const WEEKDAYS_LONG = [
  "понедельник", "вторник", "среда", "четверг", "пятница", "суббота", "воскресенье",
] as const;

/** "Июль 2026" — the U4 month bar title (Archivo 900 / 26px, not mono). */
export function monthTitle(anchor: Date): string {
  return `${MONTHS_NOMINATIVE[anchor.getUTCMonth()]} ${anchor.getUTCFullYear()}`;
}

/** "ИЮЛЬ ’26" — mobile header caption. */
export function monthCaption(anchor: Date): string {
  return `${MONTHS_NOMINATIVE[anchor.getUTCMonth()].toUpperCase()} ’${String(
    anchor.getUTCFullYear(),
  ).slice(2)}`;
}

/** "12 июля, вс" — agenda rail header value. */
export function selectedDayLabel(day: Date): string {
  return `${dayMonthCivilFmt.format(day)}, ${WEEKDAY_LABELS[mondayIndex(day)].toLowerCase()}`;
}

/** "12 июля, воскресенье" — mobile selected-date caption strip. */
export function selectedDayLabelLong(day: Date): string {
  return `${dayMonthCivilFmt.format(day)}, ${WEEKDAYS_LONG[mondayIndex(day)]}`;
}

/**
 * The month grid without a trailing week that lies entirely outside the anchor
 * month. monthGrid() always returns 6 rows; months that fit in 5 would render an
 * empty band of blanks, which the handoff's `repeat(7,1fr)` × 5 grid does not have.
 */
export function monthGridTrimmed(anchor: Date): Date[] {
  const cells = monthGrid(anchor);
  const lastWeek = cells.slice(35);
  return lastWeek.some((c) => sameMonth(c, anchor)) ? cells : cells.slice(0, 35);
}
```

(`mondayIndex` is module-private in this file and defined above these additions — no import needed. `WEEKDAY_LABELS` is `["Пн","Вт",…]`, so `.toLowerCase()` yields `"пн"`.)

- [ ] **Step 4: Run the tests**

Run: `pnpm test`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/calendar.ts lib/__tests__/calendar-labels.test.ts
git commit -m "feat(frontend): calendar month labels and trimmed month grid for U4"
```

---

### Task 5: U4 `/me/calendar` — Swiss month grid + agenda rail

**Files:**
- Create: `frontend/components/CalendarView.tsx`
- Replace: `frontend/app/me/calendar/page.tsx`

**Interfaces:**
- Consumes: `AppHeader`/`USER_NAV`/`AuthNavControl`, `Chip`, `EmptyState`, `Skeleton`, `AuthGate`, `useAuth()`, `fetchCalendar(from: Date, to: Date): Promise<CalendarEvent[]>`, `getCategories(): Promise<ApiCategory[]>`, `eventCalendarUrl(eventId: string): string`, `categoryNumeral(slug, ordered)`, `moscowTime(d: Date)`, `civilKey`, `addDays`, `todayCivil`, `shiftMonth`, `sameMonth`, `eventDayKeys`, `WEEKDAY_LABELS`, and the Task 4 helpers.
- Produces: route `/me/calendar` (unchanged path — the master plan keeps routes).

Reference markup: `Presence Swiss Grid - Full System.dc.html:301-358`. Desktop month bar `14px 20px` with a 26px/900 title left and chips `Месяц / Список / ← / →` right; body `1fr 230px`; weekday caption strip; `repeat(7,1fr)` grid with `grid-auto-rows:1fr`; day cells `5px 7px`, rules `#E0DCD4`, mono 10px date, event days ink-filled with a white date and an 8px/700 category numeral pinned bottom, out-of-month blanks `#ECEAE4`. Right rail: header cell (`Выбрано` / `12 июля, вс`), one block per event (mono time + optional `Записан` chip, 12.5px/700 title, venue caption), ghost `ДОБАВИТЬ В КАЛЕНДАРЬ` pinned bottom.

- [ ] **Step 1: Implement** — `frontend/components/CalendarView.tsx`

```tsx
"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthGate } from "@/components/ui/AuthGate";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { eventCalendarUrl, fetchCalendar, getCategories } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import {
  WEEKDAY_LABELS,
  addDays,
  civilKey,
  eventDayKeys,
  monthCaption,
  monthGridTrimmed,
  monthTitle,
  moscowTime,
  sameMonth,
  selectedDayLabel,
  selectedDayLabelLong,
  shiftMonth,
  todayCivil,
} from "@/lib/calendar";
import { categoryNumeral } from "@/lib/category-numerals";
import { cn } from "@/lib/cn";
import type { CalendarEvent } from "@/lib/types";

type Mode = "month" | "list";

/** One agenda block: mono time, «Записан» chip when the user attends, title, venue. */
function AgendaBlock({
  event,
  categories,
  dense,
}: {
  event: CalendarEvent;
  categories: ReadonlyArray<{ slug: string }>;
  dense?: boolean;
}) {
  const cat = event.categories[0];
  return (
    <div className={cn("border-b border-on-surface", dense ? "px-[14px] py-[9px]" : "px-[14px] py-[11px]")}>
      <div className="mb-[4px] flex items-baseline justify-between gap-[8px] max-sm:mb-[3px]">
        <span className="font-mono text-[10px] font-bold max-sm:text-[9px]">
          {moscowTime(new Date(event.startsAt))}
        </span>
        {event.attending ? (
          <Chip as="span" className="px-[6px] py-[2px] text-[8px]">
            Записан
          </Chip>
        ) : cat ? (
          <span className="font-mono text-[9px] font-bold">{categoryNumeral(cat.slug, categories)}</span>
        ) : null}
      </div>
      <Link
        href={`/events/${event.id}`}
        className="swiss-focus block text-[12.5px] font-bold leading-[1.1] underline-offset-2 hover:underline max-sm:text-[11.5px]"
      >
        {event.title}
      </Link>
      {event.venue?.name ? <p className="cap mt-[4px]">{event.venue.name}</p> : null}
    </div>
  );
}

export function CalendarView() {
  const { isAuthed, ready } = useAuth();
  const [mode, setMode] = useState<Mode>("month");
  const [anchor, setAnchor] = useState<Date>(() => todayCivil());
  const [selected, setSelected] = useState<Date>(() => todayCivil());

  const authed = ready && isAuthed;
  const cells = useMemo(() => monthGridTrimmed(anchor), [anchor]);
  const rangeStart = cells[0];
  const rangeEnd = addDays(cells[cells.length - 1], 1);

  const categoriesQuery = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
    staleTime: 60 * 60 * 1000,
  });
  const categories = categoriesQuery.data ?? [];

  const {
    data: events = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ["calendar", "month", civilKey(rangeStart), civilKey(rangeEnd)],
    // Widen ±1 day so no visible-day event is missed at the Moscow tz boundary;
    // exact placement is done client-side by Moscow civil day.
    queryFn: () => fetchCalendar(addDays(rangeStart, -1), addDays(rangeEnd, 1)),
    enabled: authed,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    placeholderData: (prev) => prev,
  });

  // Bucket events by every Europe/Moscow civil day they span.
  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const ev of events) {
      for (const key of eventDayKeys(ev.startsAt, ev.endsAt)) {
        const list = map.get(key) ?? [];
        list.push(ev);
        map.set(key, list);
      }
    }
    for (const list of map.values()) list.sort((a, b) => a.startsAt.localeCompare(b.startsAt));
    return map;
  }, [events]);

  const selectedKey = civilKey(selected);
  const dayEvents = byDay.get(selectedKey) ?? [];
  const icsEvent = dayEvents.find((e) => e.attending) ?? dayEvents[0];

  function go(delta: number) {
    const next = shiftMonth(anchor, delta);
    setAnchor(next);
    // Keep the selection inside the visible month so the rail is never empty
    // for an invisible day.
    setSelected(next);
  }

  const header = (
    <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption={monthCaption(anchor)} />
  );

  if (!ready) {
    return (
      <>
        {header}
        <main className="mx-auto max-w-[1360px] px-[20px] py-[26px]">
          <Skeleton className="h-[120px] w-full" />
        </main>
      </>
    );
  }

  if (!isAuthed) {
    return (
      <>
        {header}
        <AuthGate
          title="Войдите, чтобы видеть свой календарь"
          reassurance="Лента и карта доступны без входа."
        />
      </>
    );
  }

  return (
    <>
      {header}
      <main className="mx-auto flex max-w-[1360px] flex-col pb-[64px] max-sm:pb-[88px]">
        {/* Month bar */}
        <div className="flex items-baseline justify-between gap-[10px] border-b border-ink px-[20px] py-[14px] max-sm:px-[14px] max-sm:py-[11px]">
          <h1 className="text-[26px] font-black leading-[0.94] tracking-[-0.03em] max-sm:text-[20px]">
            {monthTitle(anchor)}
          </h1>
          <div className="flex shrink-0 gap-[6px]">
            <Chip variant={mode === "month" ? "active" : "default"} onClick={() => setMode("month")}>
              Месяц
            </Chip>
            <Chip variant={mode === "list" ? "active" : "default"} onClick={() => setMode("list")}>
              Список
            </Chip>
            <Chip onClick={() => go(-1)} aria-label="Предыдущий месяц">
              ←
            </Chip>
            <Chip onClick={() => go(1)} aria-label="Следующий месяц">
              →
            </Chip>
          </div>
        </div>

        {isError ? (
          <EmptyState
            numeral="!"
            title="Не удалось загрузить календарь"
            text="Проверьте соединение и попробуйте обновить страницу."
          />
        ) : isPending && events.length === 0 ? (
          <div className="grid grid-cols-7 border-b border-ink">
            {Array.from({ length: 35 }, (_, i) => (
              <Skeleton key={i} className="h-[56px] max-sm:h-[44px]" />
            ))}
          </div>
        ) : mode === "list" ? (
          /* «Список» — flat month agenda */
          events.length === 0 ? (
            <EmptyState
              title="В этом месяце ничего нет"
              text="Записи и события ваших подписок появятся здесь."
            />
          ) : (
            <div className="flex flex-col">
              {cells
                .filter((c) => sameMonth(c, anchor) && (byDay.get(civilKey(c))?.length ?? 0) > 0)
                .map((c) => (
                  <section key={civilKey(c)}>
                    <p className="cap border-b border-on-surface bg-surface-head px-[14px] py-[6px]">
                      {selectedDayLabel(c)}
                    </p>
                    {(byDay.get(civilKey(c)) ?? []).map((ev) => (
                      <AgendaBlock key={`${civilKey(c)}-${ev.id}`} event={ev} categories={categories} dense />
                    ))}
                  </section>
                ))}
            </div>
          )
        ) : (
          /* «Месяц» — grid + agenda rail */
          <div className="grid grid-cols-[1fr_230px] border-b border-ink max-md:grid-cols-1">
            <div className="flex flex-col border-r border-on-surface max-md:border-r-0">
              {/* Weekday strip */}
              <div className="grid grid-cols-7 border-b border-ink">
                {WEEKDAY_LABELS.map((w) => (
                  <span key={w} className="cap py-[6px] text-center max-sm:py-[5px] max-sm:text-[7.5px]">
                    {w}
                  </span>
                ))}
              </div>
              {/* Day grid */}
              <div className="grid grid-cols-7 auto-rows-[minmax(56px,1fr)] max-sm:auto-rows-[44px]">
                {cells.map((cell) => {
                  const key = civilKey(cell);
                  const inMonth = sameMonth(cell, anchor);
                  const cellClass =
                    "flex min-w-0 flex-col items-start border-r border-b border-rule-grid px-[7px] py-[5px] text-left";

                  // Leading/trailing cells are blanks (#ECEAE4) in the reference:
                  // context, not content. They carry no fill, no category tag and
                  // no interaction — the arrows move between months.
                  if (!inMonth) {
                    return (
                      <div key={key} className={cn(cellClass, "bg-cell-blank")} aria-hidden>
                        <span className="font-mono text-[10px] font-bold text-muted-2 max-sm:text-[9px]">
                          {cell.getUTCDate()}
                        </span>
                      </div>
                    );
                  }

                  const cellEvents = byDay.get(key) ?? [];
                  const filled = cellEvents.length > 0;
                  const cat = cellEvents[0]?.categories[0];
                  const isSelected = key === selectedKey;
                  return (
                    <button
                      key={key}
                      type="button"
                      onClick={() => setSelected(cell)}
                      aria-pressed={isSelected}
                      aria-label={`${selectedDayLabelLong(cell)}, событий: ${cellEvents.length}`}
                      className={cn(
                        cellClass,
                        "swiss-focus",
                        // One background class only — twMerge would otherwise have
                        // to arbitrate between bg-ink and bg-cell-blank.
                        filled && "bg-ink text-paper",
                        isSelected && "outline-2 -outline-offset-2 outline",
                        isSelected && (filled ? "outline-paper" : "outline-ink"),
                      )}
                    >
                      <span className="font-mono text-[10px] font-bold max-sm:text-[9px]">
                        {cell.getUTCDate()}
                      </span>
                      {filled && cat ? (
                        <span className="mt-auto font-mono text-[8px] font-bold tracking-[0.06em] max-sm:hidden">
                          {categoryNumeral(cat.slug, categories)}
                        </span>
                      ) : null}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Agenda rail. Desktop header cell = «Выбрано» + 16px/900 short date
                (reference :329). Mobile is a single caption strip carrying the
                long weekday, with no «Выбрано» label (reference :349). */}
            <div className="flex flex-col max-md:border-t max-md:border-ink">
              <div className="border-b border-on-surface px-[14px] py-[11px] max-md:py-[9px]">
                <p className="cap max-md:hidden">Выбрано</p>
                <p className="mt-[4px] text-[16px] font-black leading-[1.25] max-md:hidden">
                  {selectedDayLabel(selected)}
                </p>
                <p className="cap md:hidden">{selectedDayLabelLong(selected)}</p>
              </div>
              {dayEvents.length === 0 ? (
                <p className="cap px-[14px] py-[16px]">В этот день ничего нет</p>
              ) : (
                dayEvents.map((ev) => (
                  <AgendaBlock key={ev.id} event={ev} categories={categories} />
                ))
              )}
              {icsEvent ? (
                <div className="mt-auto px-[14px] py-[11px]">
                  <a
                    href={eventCalendarUrl(icsEvent.id)}
                    download
                    title={icsEvent.title}
                    className="swiss-focus hover-invert block border border-ink px-[11px] py-[11px] text-center text-[11px] font-bold uppercase tracking-[0.07em]"
                  >
                    Добавить в календарь
                  </a>
                </div>
              ) : null}
            </div>
          </div>
        )}
      </main>
    </>
  );
}
```

- [ ] **Step 2: Replace the route** — `frontend/app/me/calendar/page.tsx` (whole file)

```tsx
import { CalendarView } from "@/components/CalendarView";

export const metadata = { title: "Календарь — PRESENCE" };

// U4 · Календарь. CalendarView owns AppHeader because the mobile header caption
// («ИЮЛЬ ’26») tracks the client-side month state.
export default function CalendarPage() {
  return <CalendarView />;
}
```

- [ ] **Step 3: Verify**

Run: `pnpm build && pnpm test && pnpm lint`
Expected: PASS. `Segmented` is now unused by this page but still imported by `CreateEventForm` — do not delete it.

Browser (signed in with at least one attended event and one followed organizer's event in the current month), desktop and 390px, against the reference screen:
- Month bar: `Июль 2026` at 26px/900, chips `Месяц`(active) `Список` `←` `→`; arrows change the month and the mobile header caption.
- Weekday captions are centred and tracked; grid rules are `#E0DCD4`, out-of-month cells `#ECEAE4`, event days ink-filled with a white mono date and an 8px category numeral bottom-left.
- Clicking a day selects it (2px inset outline, paper on filled cells) and the rail updates; the rail header shows `Выбрано` / `12 июля, вс`.
- Agenda blocks show mono time, `Записан` chip on attended events, the category numeral otherwise, title link, venue caption.
- `ДОБАВИТЬ В КАЛЕНДАРЬ` downloads an `.ics` for the day's first attended event (check the file opens in Calendar).
- `Список` renders the month as day-grouped agenda sections with `#E6E3DC` day headers.
- 390px: 44px grid rows (tap targets), the rail sits under the grid with an ink top rule, tab bar clears the content, «Календарь» is not in the 4-tab set (tab bar shows Лента/Подбор/Карта/Я — expected, the handoff says pick one fixed set).
- Signed out: `AuthGate`; cold load: skeleton cells, never a spinner.

- [ ] **Step 4: Commit**

```bash
git add components/CalendarView.tsx app/me/calendar/page.tsx
git commit -m "feat(frontend): U4 swiss calendar — month grid, agenda rail, list mode"
```

---

### Task 6: Geo + map-stat helpers (TDD)

**Files:**
- Create: `frontend/lib/geo.ts`, `frontend/lib/map-stats.ts`
- Test: `frontend/lib/__tests__/geo.test.ts`, `frontend/lib/__tests__/map-stats.test.ts`

**Interfaces:**
- Consumes: `LiaEvent` (`lib/types.ts:59`), `moscowDayKey(d: Date): string` (`lib/calendar.ts:33`).
- Produces: `type LatLon = [number, number]` (`[lat, lon]`, the same order Yandex 2.1 uses).
- Produces: `type MapBounds = [LatLon, LatLon]` — `[[latSW, lonSW], [latNE, lonNE]]`, exactly what `ymaps.Map#getBounds()` returns.
- Produces: `haversineKm(a: LatLon, b: LatLon): number`.
- Produces: `radiusKmFromBounds(bounds: MapBounds): number` — centre-to-NE-corner distance; the radius a "search this area" would need.
- Produces: `radiusLabel(km: number | null | undefined): string` — `"5 км"` (≥10 → integer; <10 → one decimal; invalid → `"—"`).
- Produces: `distanceLabel(distanceM: number | null | undefined): string | null` — `"0.8 км"`, `null` when unknown (callers omit the segment).
- Produces: `mapAreaStats(events: ReadonlyArray<Pick<LiaEvent, "priceType" | "startsAt">>, radiusKm: number | null, now?: Date): { total: string; radius: string; free: string; today: string }`.

- [ ] **Step 1: Write the failing tests**

`frontend/lib/__tests__/geo.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { distanceLabel, haversineKm, radiusKmFromBounds, radiusLabel } from "../geo";

describe("haversineKm", () => {
  it("is zero for the same point", () => {
    expect(haversineKm([55.7558, 37.6173], [55.7558, 37.6173])).toBe(0);
  });
  it("matches a known Moscow distance (Kremlin → Garage ≈ 2.0 km)", () => {
    const km = haversineKm([55.752, 37.6175], [55.7351, 37.6053]);
    expect(km).toBeGreaterThan(1.8);
    expect(km).toBeLessThan(2.4);
  });
  it("is symmetric", () => {
    const a: [number, number] = [55.75, 37.61];
    const b: [number, number] = [55.8, 37.7];
    expect(haversineKm(a, b)).toBeCloseTo(haversineKm(b, a), 9);
  });
});

describe("radiusKmFromBounds", () => {
  it("measures centre → north-east corner", () => {
    // ~0.09° of latitude ≈ 10 km; a symmetric box around 55.75 N.
    const km = radiusKmFromBounds([
      [55.7, 37.5],
      [55.8, 37.7],
    ]);
    expect(km).toBeGreaterThan(6);
    expect(km).toBeLessThan(10);
  });
  it("is zero for a degenerate box", () => {
    expect(radiusKmFromBounds([[55.75, 37.61], [55.75, 37.61]])).toBe(0);
  });
});

describe("radiusLabel", () => {
  it("integers at 10 km and above", () => {
    expect(radiusLabel(12.4)).toBe("12 км");
    expect(radiusLabel(10)).toBe("10 км");
  });
  it("one decimal below 10 km", () => {
    expect(radiusLabel(4.96)).toBe("5.0 км");
    expect(radiusLabel(0.83)).toBe("0.8 км");
  });
  it("em dash when unknown", () => {
    expect(radiusLabel(null)).toBe("—");
    expect(radiusLabel(undefined)).toBe("—");
    expect(radiusLabel(Number.NaN)).toBe("—");
  });
});

describe("distanceLabel", () => {
  it("kilometres with one decimal", () => {
    expect(distanceLabel(800)).toBe("0.8 км");
    expect(distanceLabel(2100)).toBe("2.1 км");
  });
  it("null when unknown", () => {
    expect(distanceLabel(null)).toBeNull();
    expect(distanceLabel(undefined)).toBeNull();
  });
});
```

`frontend/lib/__tests__/map-stats.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { mapAreaStats } from "../map-stats";

const EVENTS = [
  { priceType: "free" as const, startsAt: "2026-07-12T13:00:00Z" },
  { priceType: "free" as const, startsAt: "2026-07-15T16:00:00Z" },
  { priceType: "fixed" as const, startsAt: "2026-07-12T07:00:00Z" },
  // 22:00Z on the 11th is 01:00 on the 12th in Moscow — counts as "today".
  { priceType: "from" as const, startsAt: "2026-07-11T22:00:00Z" },
];

const NOW = new Date("2026-07-12T09:00:00Z");

describe("mapAreaStats", () => {
  it("counts total, free and today (Moscow civil day)", () => {
    expect(mapAreaStats(EVENTS, 5, NOW)).toEqual({
      total: "4",
      radius: "5.0 км",
      free: "2",
      today: "3",
    });
  });
  it("empty area renders zeros and an em-dash radius", () => {
    expect(mapAreaStats([], null, NOW)).toEqual({
      total: "0",
      radius: "—",
      free: "0",
      today: "0",
    });
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `pnpm vitest run lib/__tests__/geo.test.ts lib/__tests__/map-stats.test.ts`
Expected: FAIL — modules not found.

- [ ] **Step 3: Implement** — `frontend/lib/geo.ts`

```ts
/** [latitude, longitude] — the argument order Yandex Maps 2.1 uses. */
export type LatLon = [number, number];
/** [[latSW, lonSW], [latNE, lonNE]] — the shape of ymaps.Map#getBounds(). */
export type MapBounds = [LatLon, LatLon];

const EARTH_RADIUS_KM = 6371;
const rad = (deg: number) => (deg * Math.PI) / 180;

/** Great-circle distance in kilometres. */
export function haversineKm(a: LatLon, b: LatLon): number {
  const dLat = rad(b[0] - a[0]);
  const dLon = rad(b[1] - a[1]);
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(rad(a[0])) * Math.cos(rad(b[0])) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** The radius a "search this area" at the viewport centre would need to cover
 * the visible corners. Reported in the U5 «Радиус» cell. */
export function radiusKmFromBounds(bounds: MapBounds): number {
  const [[latSW, lonSW], [latNE, lonNE]] = bounds;
  const centre: LatLon = [(latSW + latNE) / 2, (lonSW + lonNE) / 2];
  return haversineKm(centre, [latNE, lonNE]);
}

/** "12 км" | "5.0 км" | "—". Coarse above 10 km, precise below it. */
export function radiusLabel(km: number | null | undefined): string {
  if (km == null || !Number.isFinite(km)) return "—";
  return km >= 10 ? `${Math.round(km)} км` : `${km.toFixed(1)} км`;
}

/** "0.8 км" for a nearby-result distance; null when the backend sent none. */
export function distanceLabel(distanceM: number | null | undefined): string | null {
  if (distanceM == null || !Number.isFinite(distanceM)) return null;
  return `${(distanceM / 1000).toFixed(1)} км`;
}
```

`frontend/lib/map-stats.ts`:

```ts
import { moscowDayKey } from "./calendar";
import { radiusLabel } from "./geo";
import type { LiaEvent } from "./types";

export interface MapAreaStats {
  total: string;
  radius: string;
  free: string;
  today: string;
}

type StatEvent = Pick<LiaEvent, "priceType" | "startsAt">;

/** The four U5 stat cells. All values are strings because they render in mono
 * and an unknown radius must print «—» rather than a number. */
export function mapAreaStats(
  events: ReadonlyArray<StatEvent>,
  radiusKm: number | null,
  now: Date = new Date(),
): MapAreaStats {
  const todayKey = moscowDayKey(now);
  let free = 0;
  let today = 0;
  for (const e of events) {
    if (e.priceType === "free") free += 1;
    if (moscowDayKey(new Date(e.startsAt)) === todayKey) today += 1;
  }
  return {
    total: String(events.length),
    radius: radiusLabel(radiusKm),
    free: String(free),
    today: String(today),
  };
}
```

- [ ] **Step 4: Run the tests** — `pnpm test`, all PASS. If the haversine bounds in the first test disagree, correct the *expectations* (they are deliberately loose ranges), never the formula.

- [ ] **Step 5: Commit**

```bash
git add lib/geo.ts lib/map-stats.ts lib/__tests__/geo.test.ts lib/__tests__/map-stats.test.ts
git commit -m "feat(frontend): geo distance helpers and U5 area statistics"
```

---

### Task 7: `YandexMap` — square numbered ink markers, viewport reporting, pin selection

**Files:**
- Modify: `frontend/components/map/YandexMap.tsx`
- Modify: `frontend/app/globals.css` (append `.swiss-pin` block)

**Interfaces:**
- Consumes: `radiusKmFromBounds`, `type MapBounds`, `type LatLon` (Task 6).
- Produces (additive — `VenueMap` keeps working unchanged):
  - `MapPin` gains `numeral?: string` (rendered inside the square marker; falls back to the label when absent).
  - `YandexMap` gains `hideControls?: boolean`, `activePinId?: string | null`, `onPinClick?: (id: string) => void`, `onViewportChange?: (viewport: { center: LatLon; radiusKm: number }) => void`.
- Produces: global CSS classes `.swiss-pin` / `.swiss-pin--on` (Yandex injects marker HTML outside React, so Tailwind classes are not available there).

- [ ] **Step 1: Append the marker styles** — `frontend/app/globals.css` (at the end of the file)

Two handoff sources disagree about the marker and the disagreement must be settled before writing code. `map-embed.html`'s `.vpin` (`:14-17`) is a **venue-name label** — an ink box containing `ГАРАЖ` — and `Presence Map Screens.html` uses the same `.pin`. But the README calls those files "earlier map explorations" (`:495`) and states the rule for production (`:472`): *"Markers: square, ink-filled, numbered in JetBrains Mono, numbers matching the list order. No teardrop pins, no shadows, no rounded corners."* U5's own copy agrees — «номера совпадают с пинами» (`Full System :363`). **The numeral wins; the label pin is the older idea.** What we take verbatim from `.vpin` is its *geometry*: the box, the 2px × 9px stem at `left:9px`, and the `[9, 30]` anchor that puts the stem's tip on the coordinate.

```css
/* ——— Map marker ———
 * Yandex injects placemark markup outside React, so these rules cannot be
 * Tailwind utilities. Square, ink-filled, numbered in mono, with the 2px stem
 * and left:9px anchor geometry of docs/Redesign/5/.../map-embed.html (.vpin);
 * the numeral (not a venue label) per README → Maps. Selection inverts to
 * paper; it is never red (red means "needs attention"). */
.swiss-pin {
  /* Yandex puts the layout's top-left corner on the coordinate, so shift the
   * box up by its own height plus the stem to land the stem tip on the point.
   * −9px on x matches .vpin's iconAnchor x. */
  position: relative;
  display: inline-block;
  transform: translate(-9px, calc(-100% - 9px));
  min-width: 22px;
  padding: 4px 6px;
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--paper);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  line-height: 1;
  text-align: center;
  white-space: nowrap;
}

.swiss-pin::after {
  content: "";
  position: absolute;
  left: 9px;
  top: 100%;
  width: 2px;
  height: 9px;
  background: var(--ink);
}

/* Selected pin: paper fill, ink numeral, ink stem. */
.swiss-pin--on {
  background: var(--paper);
  color: var(--ink);
}
```

- [ ] **Step 2: Rewrite** — `frontend/components/map/YandexMap.tsx`

```tsx
"use client";

import { useEffect, useRef, useState } from "react";
import { radiusKmFromBounds, type LatLon, type MapBounds } from "@/lib/geo";

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
  /** Positional numeral shown inside the square marker ("01"). Must match the
   * numeral of the same event in the list rail (handoff U5). */
  numeral?: string;
}

export interface MapViewport {
  center: LatLon;
  radiusKm: number;
}

const KEY = process.env.NEXT_PUBLIC_YANDEX_MAPS_KEY ?? "";

// Event titles are user-supplied and land in balloon HTML — escape them.
function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// Load the JS API v2.1 script exactly once across every map instance on the page.
let loaderPromise: Promise<void> | null = null;
function loadYmaps(): Promise<void> {
  const w = window as unknown as { ymaps?: { ready: (cb: () => void) => void } };
  if (loaderPromise) return loaderPromise;
  loaderPromise = new Promise<void>((resolve, reject) => {
    if (w.ymaps) {
      w.ymaps.ready(() => resolve());
      return;
    }
    const script = document.createElement("script");
    script.src = `https://api-maps.yandex.ru/2.1/?apikey=${KEY}&lang=ru_RU`;
    script.onload = () => w.ymaps!.ready(() => resolve());
    script.onerror = () => reject(new Error("yandex maps failed to load"));
    document.head.appendChild(script);
  });
  return loaderPromise;
}

export function YandexMap({
  center,
  zoom = 13,
  marker,
  draggableMarker = false,
  onMarkerMove,
  pins,
  hideControls = false,
  activePinId = null,
  onPinClick,
  onViewportChange,
  className = "h-64 w-full",
}: {
  center: LatLon;
  zoom?: number;
  marker?: LatLon;
  draggableMarker?: boolean;
  onMarkerMove?: (lat: number, lon: number) => void;
  pins?: MapPin[];
  /** U5 hides the zoom control; the venue map keeps it. */
  hideControls?: boolean;
  activePinId?: string | null;
  onPinClick?: (id: string) => void;
  /** Debounced (250ms) viewport report — drives the U5 «Радиус» cell and the
   * "search this area" pill. */
  onViewportChange?: (viewport: MapViewport) => void;
  className?: string;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  // ymaps objects are untyped (the JS API ships no bundled TS types).
  const mapRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const markerRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const pinRefs = useRef<any[]>([]); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onMoveRef = useRef(onMarkerMove);
  const onPinClickRef = useRef(onPinClick);
  const onViewportRef = useRef(onViewportChange);
  const viewportTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [ready, setReady] = useState(false);

  // Keep callbacks current without re-creating the map or its geo objects.
  useEffect(() => {
    onMoveRef.current = onMarkerMove;
    onPinClickRef.current = onPinClick;
    onViewportRef.current = onViewportChange;
  });

  // init once
  useEffect(() => {
    if (!KEY) return;
    let cancelled = false;
    loadYmaps()
      .then(() => {
        if (cancelled || !elRef.current || mapRef.current) return;
        const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
        // v2.1 takes [lat, lon] — same order as our props, no conversion.
        const map = new ymaps.Map(elRef.current, {
          center,
          zoom,
          controls: hideControls ? [] : ["zoomControl"],
        });
        map.events.add("boundschange", () => {
          if (!onViewportRef.current) return;
          if (viewportTimer.current) clearTimeout(viewportTimer.current);
          viewportTimer.current = setTimeout(() => {
            const bounds = map.getBounds() as MapBounds;
            const c = map.getCenter() as LatLon;
            onViewportRef.current?.({
              center: [c[0], c[1]],
              radiusKm: radiusKmFromBounds(bounds),
            });
          }, 250);
        });
        mapRef.current = map;
        setReady(true);
      })
      .catch(() => {
        /* leave placeholder; page stays up */
      });
    return () => {
      cancelled = true;
      if (viewportTimer.current) clearTimeout(viewportTimer.current);
      mapRef.current?.destroy?.();
      mapRef.current = null;
      setReady(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // recenter
  useEffect(() => {
    if (!ready) return;
    mapRef.current?.setCenter(center, zoom);
  }, [ready, center, zoom]);

  // single marker (static or draggable)
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps) return;
    if (markerRef.current) {
      map.geoObjects.remove(markerRef.current);
      markerRef.current = null;
    }
    if (marker) {
      const pm = new ymaps.Placemark(marker, {}, { draggable: draggableMarker });
      pm.events.add("dragend", () => {
        const c = pm.geometry.getCoordinates(); // [lat, lon]
        onMoveRef.current?.(c[0], c[1]);
      });
      map.geoObjects.add(pm);
      markerRef.current = pm;
    }
  }, [ready, marker, draggableMarker]);

  // multi-pin layer — square ink markers numbered in mono
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps) return;

    // Two layouts (normal / selected) built per run; $[properties.iconContent]
    // is the documented v2.1 substitution for per-placemark text.
    const layoutFor = (active: boolean) =>
      ymaps.templateLayoutFactory.createClass(
        `<div class="swiss-pin${active ? " swiss-pin--on" : ""}">$[properties.iconContent]</div>`,
      );
    const normalLayout = layoutFor(false);
    const activeLayout = layoutFor(true);

    pinRefs.current.forEach((pm) => map.geoObjects.remove(pm));
    pinRefs.current = [];
    (pins ?? []).forEach((p) => {
      const label = escapeHtml(p.label ?? "");
      const balloon = p.href ? `<a href="${escapeHtml(p.href)}">${label}</a>` : label;
      const isActive = p.id === activePinId;
      const pm = new ymaps.Placemark(
        [p.lat, p.lon],
        {
          hintContent: label,
          balloonContent: balloon,
          iconContent: escapeHtml(p.numeral ?? label),
        },
        {
          iconLayout: isActive ? activeLayout : normalLayout,
          // Hit area in coordinate space *after* .swiss-pin's translate: the box
          // sits above-right of the point, its stem tip on it. Slightly generous
          // so a 22px square is comfortably clickable. Verify in Step 3.
          iconShape: { type: "Rectangle", coordinates: [[-9, -31], [15, -7]] },
          // The balloon is redundant when the caller handles selection itself.
          openBalloonOnClick: !onPinClickRef.current,
        },
      );
      pm.events.add("click", () => onPinClickRef.current?.(p.id));
      map.geoObjects.add(pm);
      pinRefs.current.push(pm);
    });
  }, [ready, pins, activePinId]);

  if (!KEY) {
    return (
      <div
        className={`${className} flex items-center justify-center border border-rule-inner bg-cell-blank text-[11.5px] text-text-dim`}
      >
        Карта недоступна
      </div>
    );
  }
  return <div ref={elRef} className={className} />;
}
```

Caller contract to respect (documented here because it is a footgun): **`pins` must be a memoized array.** The pin effect re-creates every placemark whenever the `pins` reference or `activePinId` changes; passing a freshly-mapped array on each render would rebuild the layer on every keystroke. `MapBrowse` (Task 8) memoizes it.

- [ ] **Step 3: Verify the venue map did not regress**

Run: `pnpm build && pnpm test && pnpm lint`
Expected: PASS (`VenueMap` passes no new props and keeps its default zoom control; its `className` no longer needs to override `rounded-control`, which the new default drops).

Browser: open any event with coordinates — the venue map still renders grayscale inside its ink hairline box with a working zoom control and a single default marker.

The marker anchor and hit shape can only be judged visually, so check them on `/map` after Task 8 and adjust the two numbers together if they disagree:
- Zoom in hard on one pin. The **stem tip**, not the box, must sit on the venue. If the box floats away from its building, the `translate` is wrong; if clicks land beside the square, `iconShape` is wrong.
- Clicking anywhere on the visible square must select the pin, and clicking 20px away must not.

- [ ] **Step 4: Commit**

```bash
git add components/map/YandexMap.tsx app/globals.css
git commit -m "feat(frontend): numbered square map markers, viewport reporting, pin selection"
```

---

### Task 8: U5 `/map` — stat strip, numbered list rail, grayscale map

**Files:**
- Rewrite: `frontend/components/MapBrowse.tsx`
- Replace: `frontend/app/map/page.tsx`

**Interfaces:**
- Consumes: `AppHeader`/`USER_NAV`/`AuthNavControl`, `Cell`/`CellStrip`, `Chip`, `EmptyState`, `Skeleton`, `YandexMap` + `MapPin` + `MapViewport` (Task 7), `fetchNearbyEvents(lat, lon, limit)`, `mapAreaStats` (Task 6), `distanceLabel`/`haversineKm`/`type LatLon` (Task 6), `priceLabel`, `type LiaEvent`.
- Produces: route `/map` (unchanged).

Reference markup: `Presence Swiss Grid - Full System.dc.html:360-407`. Desktop: four-`Cell` stat strip (Всего / Радиус / Бесплатно / Сегодня, all mono), body `230px 1fr`, left rail = filter chips row (`Все / Рядом / Free`) then numbered rows (mono numeral, 12px/700 title, `venue · distance` caption), right = the map with the `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill centred 12px from the top (paper bg, 1px ink border, 9px/700, 0.1em). Map centre `55.7420, 37.6180`, zoom 12 desktop / 11 mobile. Mobile: two-cell stat strip → map → one selected-event card (mono `01 · 0.8 КМ` + `FREE`, title 13px/700) → tab bar.

- [ ] **Step 1: Rewrite** — `frontend/components/MapBrowse.tsx`

```tsx
"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";

import { Cell, CellStrip } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { fetchNearbyEvents } from "@/lib/api";
import { cn } from "@/lib/cn";
import { distanceLabel, haversineKm, type LatLon } from "@/lib/geo";
import { mapAreaStats } from "@/lib/map-stats";
import { priceLabel } from "@/lib/price-label";
import type { LiaEvent } from "@/lib/types";
import type { MapPin, MapViewport } from "@/components/map/YandexMap";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

// Handoff U5 default view.
const MOSCOW: LatLon = [55.742, 37.618];
const SEARCH_LIMIT = 200;
const PIN_CAP = 100;
const NEAR_KM = 5;

type Filter = "all" | "near" | "free";

const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "Все" },
  { key: "near", label: "Рядом" },
  { key: "free", label: "Free" },
];

/** Positional numeral: the list index IS the pin number (handoff U5). */
function numeralAt(index: number): string {
  return String(index + 1).padStart(2, "0");
}

export function MapBrowse() {
  const [center, setCenter] = useState<LatLon>(MOSCOW);
  const [searchCenter, setSearchCenter] = useState<LatLon>(MOSCOW);
  const [viewport, setViewport] = useState<MapViewport | null>(null);
  const [events, setEvents] = useState<LiaEvent[]>([]);
  const [truncated, setTruncated] = useState(false);
  const [filter, setFilter] = useState<Filter>("all");
  const [activeId, setActiveId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (at: LatLon) => {
    setLoading(true);
    setError(null);
    try {
      const all = await fetchNearbyEvents(at[0], at[1], SEARCH_LIMIT);
      const withCoords = all.filter((e) => e.venue?.lat != null && e.venue?.lon != null);
      setTruncated(withCoords.length > PIN_CAP);
      setEvents(withCoords.slice(0, PIN_CAP));
      setSearchCenter(at);
      setActiveId(null);
    } catch {
      setError("Не удалось загрузить события в этой области.");
    } finally {
      setLoading(false);
    }
  }, []);

  // Open on the user's position when they allow it, Moscow otherwise.
  useEffect(() => {
    if (!navigator.geolocation) {
      void load(MOSCOW);
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const at: LatLon = [pos.coords.latitude, pos.coords.longitude];
        setCenter(at);
        void load(at);
      },
      () => void load(MOSCOW),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const visible = useMemo(() => {
    if (filter === "free") return events.filter((e) => e.priceType === "free");
    if (filter === "near")
      return events.filter(
        (e) => haversineKm(searchCenter, [e.venue!.lat!, e.venue!.lon!]) <= NEAR_KM,
      );
    return events;
  }, [events, filter, searchCenter]);

  // Memoized so YandexMap does not rebuild the placemark layer on every render.
  const pins = useMemo<MapPin[]>(
    () =>
      visible.map((e, i) => ({
        id: e.id,
        lat: e.venue!.lat!,
        lon: e.venue!.lon!,
        label: e.title,
        href: `/events/${e.id}`,
        numeral: numeralAt(i),
      })),
    [visible],
  );

  // Stats describe the last search; only «Радиус» tracks the live viewport.
  const stats = useMemo(
    () => mapAreaStats(visible, viewport?.radiusKm ?? null),
    [visible, viewport?.radiusKm],
  );

  const activeIndex = visible.findIndex((e) => e.id === activeId);
  const active = activeIndex >= 0 ? visible[activeIndex] : visible[0];
  const activeNumeral = activeIndex >= 0 ? numeralAt(activeIndex) : visible.length ? "01" : "—";

  const filterRow = (
    <div className="flex gap-[5px] border-b border-on-surface px-[14px] py-[8px]">
      {FILTERS.map((f) => (
        <Chip
          key={f.key}
          variant={filter === f.key ? "active" : "default"}
          onClick={() => setFilter(f.key)}
          aria-pressed={filter === f.key}
          className="text-[8px]"
        >
          {f.label}
        </Chip>
      ))}
    </div>
  );

  const listRail = (
    <>
      {filterRow}
      {loading ? (
        <div className="flex flex-col">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-[52px] border-x-0 border-t-0" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <p className="cap px-[14px] py-[16px]">Событий в этой области нет</p>
      ) : (
        <ul className="flex flex-col">
          {visible.map((e, i) => {
            const dist = distanceLabel(e.distanceM);
            return (
              <li key={e.id}>
                <Link
                  href={`/events/${e.id}`}
                  onMouseEnter={() => setActiveId(e.id)}
                  onFocus={() => setActiveId(e.id)}
                  className={cn(
                    "flex min-h-[44px] items-start gap-[8px] border-b border-on-surface px-[14px] py-[9px] swiss-focus hover-invert",
                    e.id === activeId && "bg-ink text-paper",
                  )}
                >
                  <span className="font-mono text-[10px] font-bold leading-[1.4]">{numeralAt(i)}</span>
                  <span className="min-w-0">
                    <span className="block text-[12px] font-bold leading-[1.1]">{e.title}</span>
                    <span className="cap mt-[3px] block truncate">
                      {[e.venue?.name ?? "—", dist].filter(Boolean).join(" · ")}
                    </span>
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
      {truncated ? (
        <p className="cap border-b border-on-surface px-[14px] py-[8px]">
          Показаны первые <span className="font-mono">{PIN_CAP}</span> событий
        </p>
      ) : null}
    </>
  );

  const mapPane = (mobile: boolean) => (
    <div className="relative min-h-0">
      <div className="h-full [filter:grayscale(1)_contrast(1.05)]">
        <YandexMap
          center={center}
          zoom={mobile ? 11 : 12}
          pins={pins}
          hideControls
          activePinId={activeId}
          onPinClick={setActiveId}
          onViewportChange={setViewport}
          className="h-full w-full"
        />
      </div>
      <button
        type="button"
        onClick={() => {
          const at = viewport?.center ?? center;
          setCenter(at);
          void load(at);
        }}
        className="swiss-focus hover-invert absolute left-1/2 top-[12px] z-[500] w-max -translate-x-1/2 border border-ink bg-paper px-[12px] py-[6px] text-[9px] font-bold uppercase tracking-[0.1em] text-ink"
      >
        Искать в этой области
      </button>
    </div>
  );

  if (error && events.length === 0) {
    return (
      <EmptyState
        numeral="!"
        title="Не удалось загрузить карту"
        text="Проверьте соединение и попробуйте обновить страницу."
      />
    );
  }

  return (
    <main className="mx-auto flex max-w-[1360px] flex-col max-sm:pb-[57px]">
      {/* Stat strip — four cells desktop at `.cell`/`.val` defaults, two on
          mobile at 8px/12px padding and an 11px value (reference :369-374, :394-397) */}
      <CellStrip cols={4} className="max-sm:hidden">
        <Cell caption="Всего" value={stats.total} mono />
        <Cell caption="Радиус" value={stats.radius} mono />
        <Cell caption="Бесплатно" value={stats.free} mono />
        <Cell caption="Сегодня" value={stats.today} mono />
      </CellStrip>
      <CellStrip cols={2} className="sm:hidden">
        <Cell caption="Всего" value={stats.total} mono valueClassName="text-[11px]" className="px-[12px] py-[8px]" />
        <Cell caption="Радиус" value={stats.radius} mono valueClassName="text-[11px]" className="px-[12px] py-[8px]" />
      </CellStrip>

      {/* Desktop: 230px rail + map */}
      <div className="hidden h-[calc(100vh-160px)] min-h-[420px] grid-cols-[230px_1fr] border-b border-ink sm:grid">
        <div className="flex flex-col overflow-y-auto border-r border-on-surface">{listRail}</div>
        {mapPane(false)}
      </div>

      {/* Mobile: map → selected card */}
      <div className="flex h-[calc(100vh-210px)] min-h-[360px] flex-col border-b border-ink sm:hidden">
        {mapPane(true)}
      </div>
      {active ? (
        <Link
          href={`/events/${active.id}`}
          className="block border-b border-ink px-[14px] py-[11px] swiss-focus hover-invert sm:hidden"
        >
          <span className="mb-[4px] flex items-baseline justify-between">
            <span className="font-mono text-[10px] font-bold uppercase">
              {activeNumeral}
              {distanceLabel(active.distanceM) ? ` · ${distanceLabel(active.distanceM)!.toUpperCase()}` : ""}
            </span>
            <span className="text-[11px] font-black">
              {priceLabel(active.priceMin, active.priceType)}
            </span>
          </span>
          <span className="block text-[13px] font-bold leading-[1.05]">{active.title}</span>
        </Link>
      ) : null}
      {error ? (
        <p className="border-b border-rule-inner px-[20px] py-[9px] text-[11.5px] text-signal">{error}</p>
      ) : null}
      {/* The list is the primary affordance on mobile too — below the card. */}
      <div className="flex flex-col sm:hidden">{listRail}</div>
    </main>
  );
}
```

- [ ] **Step 2: Replace the route** — `frontend/app/map/page.tsx` (whole file)

```tsx
import { MapBrowse } from "@/components/MapBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";

export const metadata = { title: "Карта — PRESENCE" };

// U5 · Карта. Public route — no auth gate.
export default function MapPage() {
  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="КАРТА" />
      <MapBrowse />
    </>
  );
}
```

- [ ] **Step 3: Verify**

Run: `pnpm build && pnpm test && pnpm lint`
Expected: PASS.

Browser (`pnpm dev`; requires `NEXT_PUBLIC_YANDEX_MAPS_KEY` in `frontend/.env.local` — without it the map renders the «Карта недоступна» box and the rest of the screen must still be correct), desktop and 390px:
- Four mono stat cells divided by ink hairlines; «Радиус» updates ~250ms after panning/zooming; the other three change only after a search.
- `230px 1fr` split; rail rows are numbered `01…NN` in mono and **the numerals match the markers on the map**; markers are square, ink-filled, mono-numbered, with a 2px stem and no zoom control.
- Hovering/focusing a rail row inverts it and inverts the corresponding marker to paper; clicking a marker selects the row (and on mobile swaps the bottom card); clicking a row opens the event.
- The map reads as a printed plan (grayscale + slight contrast); the `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill sits centred 12px from the top, is **not** desaturated into illegibility, and re-runs the search at the current centre.
- Chips `Все / Рядом / Free` filter and renumber both the list and the pins consistently.
- 390px: two-cell strip → map → selected card (mono `01 · 0.8 КМ` + `FREE`) → list; rows ≥44px; the fixed tab bar does not cover the last row.
- Empty area → «Событий в этой области нет»; network failure with no data → U8 error state.

- [ ] **Step 4: Commit**

```bash
git add components/MapBrowse.tsx app/map/page.tsx
git commit -m "feat(frontend): U5 swiss map — stat strip, numbered rail, grayscale pins"
```

---

### Task 9: Dead-code sweep

**Files:**
- Delete: `frontend/components/ui/SearchField.tsx`

- [ ] **Step 1: Confirm zero consumers**

Run: `grep -rn "SearchField" app components lib`
Expected: only the component's own file (Phase 2 removed the last usage from `DiscoveryFeed`). If anything else appears, stop and leave the file in place.

- [ ] **Step 2: Delete and re-verify**

```bash
git rm components/ui/SearchField.tsx
```

Run: `pnpm build && pnpm test && pnpm lint` — PASS.

- [ ] **Step 3: Confirm what must NOT be deleted this phase**

Run: `grep -rln "Segmented\|FilterChip\|EventCard" app components`
Expected consumers, all owned by later phases — leave them alone:
- `Segmented` → `components/CreateEventForm.tsx` (Phase 5)
- `FilterChip` → `app/admin/moderation/organizers/page.tsx`, `app/admin/moderation/events/page.tsx` (Phase 6)
- `EventCard` → `app/organizers/[id]/page.tsx` (Phase 5), `app/events/mine/page.tsx` (Phase 5)

The `TEMP Phase-1 compat` blocks in `app/globals.css` stay until Phase 7 — the admin and organizer screens still consume `glass`, the legacy radius/shadow aliases, and the old Apple-HIG colour tokens.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore(frontend): drop unused SearchField primitive"
```

---

### Task 10: Full-screen browser verification (controller, not a subagent)

No new files. Verify in the live dev browser at desktop (≥1024) and 390px against the reference screens opened side by side (`Presence Swiss Grid - Full System.dc.html`, badges U4/U5/U6).

Open the reference by serving the folder (it needs `support.js` and the `map-embed.html` iframe, so `file://` is not enough):

```bash
cd "docs/Redesign/5/design_handoff_presence_swiss_grid" && python3 -m http.server 8099
# then open http://localhost:8099/Presence%20Swiss%20Grid%20-%20Full%20System.dc.html
```

- [ ] **Step 0: Measured fidelity pass.** For each of the three screens, put the mock and the running page side by side and check the numbers from the **Design fidelity contract** with devtools — not by eye. The five that have historically drifted: type sizes on values (`Cell` value 15px on U6 / 11px on U5 mobile), chip sizes in dense cells (8px desktop status, 7px mobile status, 8px filter chips), the divider colour on each rule (`#111` structural vs `#E0DCD4` calendar cells vs `#ECEAE4` blank fills), horizontal cell padding (14px default, 12px U5 mobile, 10px U6 mobile, 8px U6 date/status columns), and uppercase tracking (0.13em caps, 0.12em chips, 0.1em map pill, 0.07em buttons). Record any deliberate difference in the phase report; do not leave an undocumented one.

- [ ] `/me` — U6: identity strip `1fr 200px` with «Участник с … 2026», Посещено/Подписки mono cells; four counting tab chips; rows on `56px 1fr 118px 134px` with ink dividers; red only on «Ожидает»; empty note + `Найти события →` chip after the last row; mobile three-cell strip and compact rows with short chips.
- [ ] `/me/practices` → `/me`; `/me/applications` → `/me` on the «Заявки» tab.
- [ ] `/me/calendar` — U4: 26px/900 month title; `Месяц / Список / ← / →` chips; `#E0DCD4` grid rules; ink-filled event days with white mono dates and category numerals; `#ECEAE4` out-of-month blanks; agenda rail with `Записан` chips and a working `ДОБАВИТЬ В КАЛЕНДАРЬ` `.ics`; «Список» mode; 44px mobile rows.
- [ ] `/map` — U5: mono stat strip; `230px 1fr`; list numerals ≡ pin numerals; square ink markers; grayscale tiles; `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill re-searches; selection inverts (never red); mobile card.
- [ ] Cross-screen: nav underline is «Профиль» on `/me`, «Календарь» on `/me/calendar`, «Карта» on `/map`; mobile tab bar highlights «Я» across `/me/*`; keyboard Tab shows 2px square outlines on every chip, row, day cell, and the map pill; no `rounded-`/`shadow-` regressions (`grep -rn "rounded-\|shadow-" app components | grep -v node_modules` returns only the TEMP-shim consumers of later phases).
- [ ] Regression pass on Phase 1–2 surfaces: `/` feed, `/events/[id]` detail (fact strip dividers are now ink — confirm it reads correctly), `/login`, `/signup`, `/no-such-page` 404, `AuthGate` on gated routes.
- [ ] Not-yet-migrated screens still function (visually stale is expected): `/search`, `/events/mine`, `/organizer/*`, `/organizers/[id]`, `/admin/*`, `/events/new`.
- [ ] `pnpm build && pnpm test && pnpm lint` at HEAD; `go build ./... && go test ./...` in `backend/`.

Then: whole-branch review (subagent-driven-development flow) → merge `redesign/swiss-grid-p3` to `main`.

---

### Task 11: Deploy Phase 3 to prod (controller-run, follows the runbook)

No destructive DB steps this phase. Follow `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md` and the Phase 1+2 deploy runbook written at the end of Phase 2.

- [ ] **Step 1: Pre-flight.** Frontend build needs BOTH build-args (`NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`, `NEXT_PUBLIC_YANDEX_MAPS_KEY=…`) — the map is now the centre of a whole screen, so an empty key would ship a «Карта недоступна» box to prod. Confirm both `ARG`s exist in `frontend/Dockerfile` and that the key is Referer-restricted to `presence.tarski.ru`.
- [ ] **Step 2: Build.** amd64 images on the Mac → `docker save | ssh | docker load`; tag `swiss-p3-r1`; keep `rollback-swiss-p2` tags of the running images. `next/font` still fetches Golos/Manrope/JetBrains Mono at build time — the build stage needs outbound network to `fonts.googleapis.com` / `fonts.gstatic.com`.
- [ ] **Step 3: Backend.** This phase changes one handler (`/auth/me` payload) and adds **no migration**. Recreate the backend container with all 4 compose files + `--no-build`.
- [ ] **Step 4: Frontend.** Recreate `lia-frontend-presence` on :3002.
- [ ] **Step 5: Verify live on https://presence.tarski.ru** — `/me` (identity строка prints a real registration month, not a missing caption → proves the backend rolled out), `/me/calendar` (month grid, `.ics` download), `/map` (real Yandex tiles grayscale, numbered square pins, pill re-search), the `/me/practices` and `/me/applications` redirects, plus a Phase 1–2 regression sweep. Check `GET /auth/me` returns `created_at` (`curl -H "Authorization: Bearer …" https://api.presence.tarski.ru/auth/me`).
- [ ] **Step 6: Housekeeping.** Docker prune on the box (builder prune + dangling images + trim `rollback-*` to the last ~3 — the 20 GB disk fills otherwise). Update `docs/HANDOFF.md`; write `docs/superpowers/runbooks/<date>-swiss-p3-deploy.md`; record in the phase report the two backend follow-ups this phase deferred: a real `bbox`/viewport events query for U5, and `GET /me` profile stats so U6 does not derive Посещено/Подписки from three list fetches.

---

## Self-Review

**1. Spec coverage (README → Screens → U4, U5, U6; Interactions; Maps):**

| Spec requirement | Task |
|---|---|
| U4 month bar (26px/900 title, `Месяц/Список/←/→` chips) | 5 (Step 1), labels from 4 |
| U4 body `1fr 230px`, weekday strip, `repeat(7,1fr)` grid, `#E0DCD4` rules, ink-filled event days + white numeral + category numeral, `#ECEAE4` blanks | 5 |
| U4 agenda rail (`Выбрано` header, mono time, `Записан` chip, title, venue, ghost `ДОБАВИТЬ В КАЛЕНДАРЬ`) | 5 (deviation 4 documented) |
| U4 mobile compact grid + selected-date caption strip (long weekday, no «Выбрано») + agenda + tab bar | 5 (44px rows per the tap-target rule) |
| U5 four-`Cell` mono stat strip (Всего / Радиус / Бесплатно / Сегодня) | 6 (`mapAreaStats`), 8 |
| U5 `230px 1fr`, filter chips, numbered list rows with `venue · distance`, list numerals ≡ pin numerals | 8 |
| U5 floating `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill (paper, 1px ink, 9px/700, 0.1em, above tiles) | 8 |
| U5 mobile two-cell strip → map → selected card (`01 · 0.8 КМ` + `FREE`) → tab bar | 8 |
| Maps: grayscale `grayscale(1) contrast(1.05)`, square ink mono-numbered markers, controls hidden | 7 (CSS + layouts), 8 (filter wrapper) |
| U6 identity strip `1fr 200px` («Участник с марта 2026», 30px/900 name, Посещено/Подписки cells) | 1 (`memberSince`, backend `created_at`), 3 |
| U6 counting tab chips | 3 (deviation 1: Заявки replaces Избранное) |
| U6 rows on `56px 1fr 118px 134px` (mono date, title+context, organizer cell, centred status chip) | 3, labels from 2 |
| U6 inline empty note (`Больше записей пока нет` + `Найти события →`) | 3 |
| U6 mobile identity + three stat cells + chips + compact rows with `ОК`/`ЖДЁМ` | 3 (the third cell is Заявки, not Избранное — deviation 1; all four chips stay — deviation 11) |
| Status → chip variant map | 2 (test locks the tone routing; `lib/status-chip.ts` unchanged) |
| Loading = skeleton cells at final dimensions, numbers show `—` | 3, 5, 8 (`countLabel`, `radiusLabel`, `Skeleton`) |
| Empty / auth-gated / error follow U8 | 3, 5, 8 (`EmptyState`, `AuthGate`) |
| Hover inverts, 2px square focus, 120ms linear | inherited from Phase 1 utilities; asserted in Task 10 |
| Tap targets ≥44px | 3, 5, 8 (explicit `min-h`/`auto-rows`) |
| Nav: active item = 2px bottom rule; mobile user nav is the tab bar | 3 (Steps 2–3), verified in 10 |

Deliberately **not** covered here (owned elsewhere): U6 «Избранное» (no backend — master plan defers), a public all-events calendar (master plan: follow-up), a real bbox events query (Task 11 Step 6 records it), `GET /me` stats endpoint (same).

**2. Placeholder scan:** clean. Every code step carries complete, runnable code; every command has an expected result. Three numbers are deliberately left to be confirmed in the browser rather than asserted here, each with the failure mode described so the implementer knows what "wrong" looks like: the `.swiss-pin` `translate` and the matching `iconShape` rectangle (Task 7 Step 3), and the two viewport-height calculations in `MapBrowse` (`calc(100vh-160px)` / `calc(100vh-210px)`, which depend on the final header and tab-bar heights). The calendar row counts and the haversine distances were computed against the real 2026 calendar and real Moscow coordinates, so those expectations are exact, not guesses.

**3. Type consistency:**
- `LatLon` / `MapBounds` / `MapViewport`: declared in `lib/geo.ts` (Task 6), consumed by `YandexMap` (Task 7) and `MapBrowse` (Task 8) under those exact names. `YandexMap`'s `center`/`marker` props are widened from `[number, number]` to `LatLon` — structurally identical, so `VenueMap` keeps compiling.
- `MapPin.numeral` (Task 7) is produced by `numeralAt(i)` (Task 8); the same function feeds the list rail, which is what guarantees "list numerals ≡ pin numerals".
- `mapAreaStats(events, radiusKm, now?)` returns `{ total, radius, free, today }` — all `string`, consumed as `Cell value` (`ReactNode`) in Task 8. Its `StatEvent` is `Pick<LiaEvent, "priceType" | "startsAt">`, satisfied by `LiaEvent`.
- `rsvpStatusLabel(status): { long, short }` (Task 2) → `long` into `StatusChip status`, `short` into `StatusChip children` (the prop added in Task 3). `statusChipVariant` keeps its `"active" | "default" | "signal"` return type; no set changes.
- `memberSince(iso?): string | null` (Task 1) ← `getMe().createdAt?: string` (Task 1) — the optional-to-nullable handoff is explicit in the signature.
- `monthTitle` / `monthCaption` / `selectedDayLabel` / `selectedDayLabelLong` / `monthGridTrimmed` (Task 4) all take a civil `Date` (UTC midnight) exactly like the existing `lib/calendar.ts` exports, and `CalendarView` (Task 5) only ever passes `todayCivil()` / `shiftMonth()` / `monthGridTrimmed()` output.
- `formatShortDate` (Task 2) reuses `shortDateFmt` already declared in `lib/format.ts:86` — no duplicate formatter, no new timezone assumption.
- `eventCalendarUrl(eventId)` (`lib/api.ts:544`) and `priceLabel(priceRub, kind)` (`lib/price-label.ts:7`) are used with their existing signatures. `LiaEvent.priceType` is `"free" | "fixed" | "from"` and `priceLabel`'s `PriceKind` is the same union, so the U5 card passes it straight through.
- `Cell` gains one optional prop (`valueClassName`, Task 3 Step 1). Every existing call site omits it, so `CellProps` stays backwards-compatible and U2's fact strip is unaffected.

**4. Design fidelity:** every number in the three screens now traces to a line in `Presence Swiss Grid - Full System.dc.html` via the **Design fidelity contract** section, and each of the eleven documented deviations names the reason and the authority that overrides the mock (README rule, missing backend, or the 44px tap-target rule). The two conflicts inside the handoff itself are settled explicitly rather than silently: the marker is a **numeral**, not `map-embed.html`'s venue label (README → Maps wins over the file the README itself labels an earlier exploration), and the typeface is **Golos Text / Manrope** because Archivo and Space Grotesk have no Cyrillic. Three details in the mock are deliberately *not* copied and are called out where they occur: the mobile calendar's 26px rows (tap targets), U5's mobile `СПИСКОМ` toggle (the list is stacked instead), and U6's two-chip mobile row (all four chips stay, since «Заявки» has no other route after the redirect).
