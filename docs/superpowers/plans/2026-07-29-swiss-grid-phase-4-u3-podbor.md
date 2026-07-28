# Swiss Grid Phase 4 — U3 Подбор Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/search` `ComingSoon` stub with the Swiss Grid **U3 · AI-подбор** screen as a **deterministic smart-filter** (templated one-sentence answer + ≤3 `EventModule`s with `Совпало: …`), without calling `POST /discover` or any LLM.

**Architecture:** Thin server shell (`app/search/page.tsx`) SSR-fetches the same published catalogue + ordered categories as the feed, then hands them to a client body (`DiscoverBrowse`). All matching/ranking/sentence logic lives in pure helpers (`discover-intent`, `discover-rank`) covered by Vitest. Chip clicks and free-text submit resolve to the same `DiscoverIntent`; the client applies it over `fetchPublishedEvents` (TanStack Query) and optionally `fetchNearbyEvents` when geo is available for «рядом». Layout fidelity comes from the HTML badge `U3 · AI-подбор`; behaviour from the handoff README § U3.

**Tech Stack:** Next.js 16 App Router / React 19 / TypeScript, Tailwind v4 (Swiss Grid tokens), TanStack Query 5, Vitest 4 (node-only pure helpers), existing `GET /api/v1/events?status=published` + `GET /events/nearby` (no new backend).

**Spec:** `docs/superpowers/specs/2026-07-29-swiss-grid-phase-4-u3-podbor.md`. Design: `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` § U3 + EventModule AI variant. Pixel: `Presence Swiss Grid - Full System.dc.html` badge `U3 · AI-подбор` (~line 240). Handoff: `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-3-HANDOFF.md`. Master: `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` Phase 4.

**Prerequisite:** `main` at or after `85d40b2` (Phase 3 merge). Verify before starting:

```bash
git log --oneline -1   # expect Phase 3 merge or later
test -f frontend/components/ui/EventModule.tsx
test -f frontend/app/search/page.tsx
```

**Branch / worktree (at execution time only):** `redesign/swiss-grid-p4` via `superpowers:using-git-worktrees`, branched from `main` @ ≥`85d40b2`. Working directory for all `pnpm` commands: `frontend/`. **Do not implement until this plan is reviewed.**

---

## Global Constraints

Everything from the master plan (`2026-07-28-swiss-grid-redesign-master-plan.md`) applies. The ones this phase trips over constantly:

- **Zero border radius, zero shadows.** Rules are 1px solid: `#111` (`border-ink`) structural, `#DDD` (`border-rule-inner`) inner (including `Совпало` top rule on `EventModule`).
- **Categories are numerals, never colours** — via `categoryNumeral(slug, orderedCategories)` already inside `eventToModuleProps`.
- **All numbers in JetBrains Mono** (`font-mono`): dates on modules, any counts. Unknown → `—`.
- **`signal` red only for needs-attention** — never for selected chips or the submit square (submit is ink fill).
- Uppercase tracking preserved: `.cap` 0.13em, chips 0.12em, buttons 0.07em, nav 0.14em.
- Hover **inverts** (`hover-invert`), focus `swiss-focus` (2px square), transitions 120ms linear on background/color only. Loading = `Skeleton` at final dimensions — **no spinners**.
- Empty / error = `EmptyState` (U8). Escape hatch footer stays visible in every post-submit state.
- **Tap targets ≥44px on touch.** Input row / submit / chips / escape chip grow vertically if needed; type sizes stay at fidelity table values.
- Content max 1360px; Moscow-pinned time helpers; React #418 (single `<Link>` on `EventModule` — never nest anchors).
- UI copy Russian; code/commits English.
- Fonts stay **Golos Text / Manrope / JetBrains Mono** (locked P1 — Archivo/Space Grotesk lack Cyrillic). Manrope has no 900 — titles use `font-black` on `--font-ui` (Golos).
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Deliberate deviations for Phase 4 (pre-decided — do not reopen P1–P3; do not "fix" these without product sign-off)

1. **Non-AI brain.** Caption stays `AI-подбор` per handoff mock. Answers are **templated** Russian one-liners from applied filters — not model-generated. No honesty disclaimer line (not in the mock). Real `POST /discover` / LLM stays deferred (`lia-ai-provider-constraint`).
2. **Route stays `/search`.** Handoff conceptual path `/discover` is not renamed (master plan route mapping; nav already points at `/search`).
3. **«Бесплатно рядом» without geo** → free events city-wide; answer sentence **must not** claim distance («Бесплатные события в городе.»). With geo → filter to ≤5 km (same `NEAR_KM` as U5) and may mention близость.
4. **«С детьми»** has no dedicated category in taxonomy (`lecture|workshop|mediation|concert|exhibition|performance|film|festival|reading-group`). Use title/description keyword heuristics; if weak, still return ≤3 upcoming with honest reason `ближайшие события` — never invent fitness.
5. **No chat history / multi-turn.** Idle = prompt only. One submit → one answer strip → results; new submit replaces. Never a transcript.
6. **Mobile suggestion chips keep full labels** with horizontal scroll (U1 filter-bar pattern). The mock shortens to «Тихое / Бесплатно / Вечер» for density — full labels win for clarity and a11y; type may drop to 8px on `max-sm:` to match mock density.
7. **Match-reason strings omit the `Совпало:` prefix** — `EventModule` already prepends `Совпало: {matchReason}` (see `EventModule.tsx`).

**Locked from earlier phases (do not reopen):** favorites/♡ deferred; home stays `/`; U4 personal calendar; Yandex not OSM; fonts Golos/Manrope/JB Mono; numeral pins; etc.

---

## Design fidelity contract

`Presence Swiss Grid - Full System.dc.html` `:240-298` is the pixel authority; README § U3 is behaviour authority.

| Element | Desktop | Mobile |
|---|---|---|
| Prompt block pad | `18px 20px` | `13px 14px` |
| Caption | `.cap` `AI-подбор`, `mb-6px` | omitted (header caption `ПОДБОР`) |
| Title | 34px / 900 / −0.03em / lh ~0.94, two lines | 20px / 900, one line OK |
| Input row | `1px solid #111`, flex; text `12px` / `text-field-text` (`#6B665E`); pad `11px 14px` | text ~10.5px; pad `9px 10px` |
| Submit `→` | ink bg, white, 700, pad `11px 20px`, `min-h-[44px]` on touch | pad `9px 13px`, still ≥44px height |
| Suggestion chips | default `Chip`, gap 6 | gap 5; scrollable row; optional `text-[8px]` |
| Answer strip | 12.5px; pad `12px 20px`; `border-b border-ink` | 11px; pad `10px 14px` |
| Results | `grid-cols-3` hairline cells of `EventModule` + `matchReason` | stacked rows (`max-sm:grid-cols-1`) |
| Match reason | 10px; top rule `#DDD` (already in `EventModule`) | via `EventModule` mobile block |
| Escape footer | pad `10px 20px`; `.cap` left + chip right | same; tab bar below via `TabBarGate` |

Placeholder (desktop mock): `Хочу спокойный вечер с искусством и без толпы…` — use that string. Mobile may shorten placeholder to `Спокойный вечер…`.

---

## Chip → intent → answer → matchReason (locked for TDD)

| Chip label | `intentId` | Predicates (applied in ranker) | Answer template (Russian) | Per-event `matchReason` priority |
|---|---|---|---|---|
| Тихое в выходные | `quiet_weekend` | `startsAt ∈ weekendRange(now)`; prefer categories `mediation`, `reading-group`, `exhibition`; soft-boost `capacity != null && capacity ≤ 12` | `Нашла {n} {quietNoun} на выходных — тихо и без спешки.` (`n` capped display 1–3; if 0 empty path) | `группа {capacity} человек` if capacity≤12; else `тихий формат` if preferred cat; else `выходные` |
| Бесплатно рядом | `free_nearby` | `priceType === "free"`; if `userLatLon` set, also `venue.lat/lon` within 5 km via `haversineKm` | With geo: `Бесплатные события рядом — до 5 км.` Without: `Бесплатные события в городе.` | `бесплатно` (+ optional distance only if geo filtered) |
| Для двоих вечером | `evening_pair` | Moscow hour ≥ 18; `startsAt` in `[now, now+7d)`; prefer offline / `capacity ≤ 40` | `Вечерние события на ближайшие дни.` | `вечер, будний день` if Moscow weekday Mon–Fri; else `вечер` |
| С детьми | `with_kids` | Title/description match `/(дет|семь|юн(ый|ая|ое)|школ)/i`; if &lt;1 hard match, **fallback** = upcoming published (no fake kids claim) | Hard: `События, где обычно бывает с детьми.` Fallback: `Ближайшие события — уточните на странице, подойдёт ли с детьми.` | Hard: `семейный формат`; Fallback: `ближайшие события` |

**Free-text → intent** (first match wins, case-insensitive, Russian stems):

| Query contains | Maps to `intentId` |
|---|---|
| `бесплат` | `free_nearby` (`wantsNearby: true`; geo optional) |
| `выходн` | `quiet_weekend` |
| `вечер` | `evening_pair` |
| `тих` / `спокой` | `quiet_weekend` if also weekend-ish else `quiet_any` (upcoming + quiet cats) |
| `дет` / `семь` | `with_kids` |
| else | `keyword` — title/organizer/venue substring; answer `По запросу «{q}» — вот что нашлось.`; reason `совпадение по тексту` or category label if slug matched |

Additional intent ids used only by free-text: `quiet_any`, `keyword`.

**Cap results at 3.** Sort: upcoming first (`startsAt >= now`) ascending, then past descending (same as `DiscoveryFeed`). Prefer upcoming when trimming to 3.

---

## File Structure

```
frontend/
  lib/discover-intent.ts                 CREATE — chips + free-text → DiscoverIntent
  lib/discover-rank.ts                   CREATE — filter/score/take 3 + answer sentence + matchReason
  lib/__tests__/discover-intent.test.ts  CREATE
  lib/__tests__/discover-rank.test.ts    CREATE
  components/DiscoverBrowse.tsx          CREATE — client U3 body (prompt / answer / results / empty / error)
  app/search/page.tsx                    REPLACE — thin shell AppHeader + DiscoverBrowse (SSR seed)
```

**Not touched:** `EventModule` (already has `matchReason`), `USER_NAV` / `BottomTabBar` (already `/search`), `DiscoveryFeed` filter UX, backend, organizer/admin.

**Data reuse (inventory):**

| Need | Source |
|---|---|
| Published list | `fetchPublishedEvents(from?, to?)` — `frontend/lib/api.ts:164-180` (same as `DiscoveryFeed`) |
| Nearby | `fetchNearbyEvents(lat, lon, limit?)` — `api.ts:216-230` |
| Categories | `getCategories()` — `api.ts:148+`; SSR like `app/page.tsx` |
| Weekend / today windows | `weekendRange` / `todayRange` — `lib/mock-events.ts:17-40` |
| Distance | `haversineKm` — `lib/geo.ts:10` |
| Geo timeout fallback pattern | `createTimedResolver` — `lib/timed-resolver.ts` (U5 uses 6s; **Подбор only prompts geo when intent needs nearby** — do not auto-prompt on mount) |
| Module props | `eventToModuleProps(event, categories)` — `lib/event-module.ts` |
| Mock SSR fallback | `MOCK_EVENTS` — `lib/mock-events.ts` |

**Fetch strategy:** Server shell loads `fetchPublishedEvents().catch(() => MOCK_EVENTS)` + `getCategories().catch(() => [])`. Client `useQuery({ queryKey: ["events","published",null,null], queryFn: () => fetchPublishedEvents(), initialData })`. Ranking is **client-side** over that list (and optionally a one-shot nearby fetch when `free_nearby`/`keyword+geo` runs with permission). **No new backend.**

---

### Task 1: `parseDiscoverIntent` — chips + free text (TDD)

**Files:**
- Create: `frontend/lib/discover-intent.ts`
- Test: `frontend/lib/__tests__/discover-intent.test.ts`

**Interfaces:**
- Produces:

```ts
export type DiscoverIntentId =
  | "quiet_weekend"
  | "quiet_any"
  | "free_nearby"
  | "evening_pair"
  | "with_kids"
  | "keyword";

export interface DiscoverIntent {
  id: DiscoverIntentId;
  /** Original chip label or trimmed free-text (for answer templates). */
  sourceLabel: string;
  /** Substring for keyword intent; empty otherwise. */
  keyword: string;
  /** True when ranker may use userLatLon / nearby fetch. */
  wantsNearby: boolean;
}

export const DISCOVER_CHIPS: readonly { id: DiscoverIntentId; label: string }[] = [
  { id: "quiet_weekend", label: "Тихое в выходные" },
  { id: "free_nearby", label: "Бесплатно рядом" },
  { id: "evening_pair", label: "Для двоих вечером" },
  { id: "with_kids", label: "С детьми" },
];

/** Chip click → intent. */
export function intentFromChip(chipId: DiscoverIntentId): DiscoverIntent;

/** Free-text submit → intent. Empty/whitespace → null (UI should no-op). */
export function parseDiscoverQuery(raw: string): DiscoverIntent | null;
```

- Consumes: nothing from later tasks.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from "vitest";
import {
  DISCOVER_CHIPS,
  intentFromChip,
  parseDiscoverQuery,
} from "../discover-intent";

describe("DISCOVER_CHIPS", () => {
  it("lists the four handoff labels in order", () => {
    expect(DISCOVER_CHIPS.map((c) => c.label)).toEqual([
      "Тихое в выходные",
      "Бесплатно рядом",
      "Для двоих вечером",
      "С детьми",
    ]);
  });
});

describe("intentFromChip", () => {
  it("maps each chip id", () => {
    expect(intentFromChip("quiet_weekend")).toMatchObject({
      id: "quiet_weekend",
      wantsNearby: false,
      keyword: "",
    });
    expect(intentFromChip("free_nearby").wantsNearby).toBe(true);
    expect(intentFromChip("evening_pair").id).toBe("evening_pair");
    expect(intentFromChip("with_kids").id).toBe("with_kids");
  });
});

describe("parseDiscoverQuery", () => {
  it("returns null for blank input", () => {
    expect(parseDiscoverQuery("")).toBeNull();
    expect(parseDiscoverQuery("   ")).toBeNull();
  });
  it("detects free / weekend / evening / kids / quiet stems", () => {
    expect(parseDiscoverQuery("что-нибудь бесплатное").id).toBe("free_nearby");
    expect(parseDiscoverQuery("на выходные").id).toBe("quiet_weekend");
    expect(parseDiscoverQuery("вечером вдвоём").id).toBe("evening_pair");
    expect(parseDiscoverQuery("с детьми в музей").id).toBe("with_kids");
    expect(parseDiscoverQuery("тихое место").id).toBe("quiet_any");
  });
  it("falls back to keyword with trimmed sourceLabel", () => {
    const i = parseDiscoverQuery("  медиация гараж  ");
    expect(i).toEqual({
      id: "keyword",
      sourceLabel: "медиация гараж",
      keyword: "медиация гараж",
      wantsNearby: false,
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest run lib/__tests__/discover-intent.test.ts`
Expected: FAIL — cannot resolve `../discover-intent`.

- [ ] **Step 3: Implement** — `frontend/lib/discover-intent.ts`

```ts
export type DiscoverIntentId =
  | "quiet_weekend"
  | "quiet_any"
  | "free_nearby"
  | "evening_pair"
  | "with_kids"
  | "keyword";

export interface DiscoverIntent {
  id: DiscoverIntentId;
  sourceLabel: string;
  keyword: string;
  wantsNearby: boolean;
}

export const DISCOVER_CHIPS: readonly { id: DiscoverIntentId; label: string }[] = [
  { id: "quiet_weekend", label: "Тихое в выходные" },
  { id: "free_nearby", label: "Бесплатно рядом" },
  { id: "evening_pair", label: "Для двоих вечером" },
  { id: "with_kids", label: "С детьми" },
];

export function intentFromChip(chipId: DiscoverIntentId): DiscoverIntent {
  const chip = DISCOVER_CHIPS.find((c) => c.id === chipId);
  const label = chip?.label ?? chipId;
  return {
    id: chipId,
    sourceLabel: label,
    keyword: "",
    wantsNearby: chipId === "free_nearby",
  };
}

export function parseDiscoverQuery(raw: string): DiscoverIntent | null {
  const sourceLabel = raw.trim().replace(/\s+/g, " ");
  if (!sourceLabel) return null;
  const q = sourceLabel.toLowerCase();

  if (/бесплат/.test(q)) {
    return { id: "free_nearby", sourceLabel, keyword: "", wantsNearby: true };
  }
  if (/выходн/.test(q)) {
    return { id: "quiet_weekend", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/вечер/.test(q)) {
    return { id: "evening_pair", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/дет|семь/.test(q)) {
    return { id: "with_kids", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/тих|спокой/.test(q)) {
    return { id: "quiet_any", sourceLabel, keyword: "", wantsNearby: false };
  }
  return { id: "keyword", sourceLabel, keyword: sourceLabel, wantsNearby: false };
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `pnpm vitest run lib/__tests__/discover-intent.test.ts`

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/discover-intent.ts frontend/lib/__tests__/discover-intent.test.ts
git commit -m "$(cat <<'EOF'
feat(frontend): add discover intent parser for U3 chips and free text

EOF
)"
```

Then: `pnpm build && pnpm test && pnpm lint` (green gate).

---

### Task 2: `rankDiscover` + answer sentence (TDD)

**Files:**
- Create: `frontend/lib/discover-rank.ts`
- Test: `frontend/lib/__tests__/discover-rank.test.ts`

**Interfaces:**
- Consumes: `DiscoverIntent` from Task 1; `weekendRange` from `lib/mock-events`; `haversineKm` / `LatLon` from `lib/geo`; `LiaEvent` from `lib/types`.
- Produces:

```ts
export interface DiscoverHit {
  event: LiaEvent;
  matchReason: string; // WITHOUT "Совпало:" prefix
  score: number;
}

export interface DiscoverRanking {
  answer: string;
  hits: DiscoverHit[]; // length 0..3
  /** True when with_kids used the weak upcoming fallback. */
  usedKidsFallback: boolean;
  /** True when free_nearby ran without geo (city-wide free). */
  omittedDistanceClaim: boolean;
}

export function moscowHour(iso: string): number; // 0–23 Europe/Moscow

export function rankDiscover(
  events: readonly LiaEvent[],
  intent: DiscoverIntent,
  opts: {
    now: Date;
    userLatLon?: LatLon | null;
  },
): DiscoverRanking;
```

- [ ] **Step 1: Write the failing test** — cover each chip path + empty + cap-3 + kids fallback + free without geo.

```ts
import { describe, expect, it } from "vitest";
import { intentFromChip, parseDiscoverQuery } from "../discover-intent";
import { moscowHour, rankDiscover } from "../discover-rank";
import type { LiaEvent } from "../types";

function ev(partial: Partial<LiaEvent> & Pick<LiaEvent, "id" | "title" | "startsAt">): LiaEvent {
  return {
    format: "offline",
    status: "published",
    priceType: "paid",
    categories: [{ id: "c1", slug: "lecture", label: "Лекции" }],
    ...partial,
  };
}

// Fixed "now": Wednesday 2026-07-15 12:00 Moscow (UTC+3) = 09:00Z
const NOW = new Date("2026-07-15T09:00:00Z");

describe("moscowHour", () => {
  it("reads Europe/Moscow wall hour", () => {
    expect(moscowHour("2026-07-15T18:30:00+03:00")).toBe(18);
    expect(moscowHour("2026-07-15T15:00:00Z")).toBe(18);
  });
});

describe("rankDiscover", () => {
  const weekendQuiet = ev({
    id: "1",
    title: "Тихая медиация",
    startsAt: "2026-07-18T11:00:00+03:00", // Sat
    categories: [{ id: "m", slug: "mediation", label: "Медиации" }],
    capacity: 8,
  });
  const weekendLoud = ev({
    id: "2",
    title: "Большой фестиваль",
    startsAt: "2026-07-19T14:00:00+03:00", // Sun
    categories: [{ id: "f", slug: "festival", label: "Фестивали" }],
    capacity: 200,
  });
  const freeFar = ev({
    id: "3",
    title: "Бесплатная лекция",
    startsAt: "2026-07-16T19:00:00+03:00",
    priceType: "free",
    venue: { id: "v", name: "Далеко", lat: 56.0, lon: 38.0 },
  });
  const freeNear = ev({
    id: "4",
    title: "Бесплатно у парка",
    startsAt: "2026-07-16T12:00:00+03:00",
    priceType: "free",
    venue: { id: "v2", name: "Рядом", lat: 55.75, lon: 37.62 },
  });
  const evening = ev({
    id: "5",
    title: "Вечерний кинопоказ",
    startsAt: "2026-07-16T19:00:00+03:00", // Thu 19:00
    categories: [{ id: "fi", slug: "film", label: "Кино" }],
  });
  const kids = ev({
    id: "6",
    title: "Мастер-класс для детей",
    startsAt: "2026-07-17T11:00:00+03:00",
    description: "Семейная программа",
  });
  const plain = ev({
    id: "7",
    title: "Обычная лекция",
    startsAt: "2026-07-20T10:00:00+03:00",
  });

  const catalogue = [weekendQuiet, weekendLoud, freeFar, freeNear, evening, kids, plain];

  it("quiet_weekend prefers small mediation and reasons", () => {
    const r = rankDiscover(catalogue, intentFromChip("quiet_weekend"), { now: NOW });
    expect(r.hits.length).toBeGreaterThan(0);
    expect(r.hits.length).toBeLessThanOrEqual(3);
    expect(r.hits[0].event.id).toBe("1");
    expect(r.hits[0].matchReason).toMatch(/группа 8|тихий формат|выходные/);
    expect(r.answer).toMatch(/выходн/i);
  });

  it("free_nearby without geo is city-wide and omits distance claim", () => {
    const r = rankDiscover(catalogue, intentFromChip("free_nearby"), {
      now: NOW,
      userLatLon: null,
    });
    expect(r.omittedDistanceClaim).toBe(true);
    expect(r.answer).toBe("Бесплатные события в городе.");
    expect(r.hits.every((h) => h.event.priceType === "free")).toBe(true);
  });

  it("free_nearby with geo keeps near free events", () => {
    const r = rankDiscover(catalogue, intentFromChip("free_nearby"), {
      now: NOW,
      userLatLon: [55.742, 37.618],
    });
    expect(r.omittedDistanceClaim).toBe(false);
    expect(r.hits.map((h) => h.event.id)).toContain("4");
    expect(r.hits.map((h) => h.event.id)).not.toContain("3");
  });

  it("evening_pair keeps hour>=18 within 7 days", () => {
    const r = rankDiscover(catalogue, intentFromChip("evening_pair"), { now: NOW });
    expect(r.hits.some((h) => h.event.id === "5")).toBe(true);
    expect(r.hits[0].matchReason).toMatch(/вечер/);
  });

  it("with_kids hard-match vs fallback", () => {
    const hard = rankDiscover(catalogue, intentFromChip("with_kids"), { now: NOW });
    expect(hard.usedKidsFallback).toBe(false);
    expect(hard.hits[0].event.id).toBe("6");
    expect(hard.hits[0].matchReason).toBe("семейный формат");

    const weak = rankDiscover([plain], intentFromChip("with_kids"), { now: NOW });
    expect(weak.usedKidsFallback).toBe(true);
    expect(weak.hits[0].matchReason).toBe("ближайшие события");
    expect(weak.answer).toMatch(/уточните/);
  });

  it("caps at 3 and keyword path works", () => {
    const many = Array.from({ length: 5 }, (_, i) =>
      ev({
        id: `k${i}`,
        title: `медиация номер ${i}`,
        startsAt: `2026-07-${16 + i}T12:00:00+03:00`,
      }),
    );
    const r = rankDiscover(many, parseDiscoverQuery("медиация")!, { now: NOW });
    expect(r.hits).toHaveLength(3);
    expect(r.answer).toContain("медиация");
  });

  it("empty catalogue → empty hits + still a sentence", () => {
    const r = rankDiscover([], intentFromChip("evening_pair"), { now: NOW });
    expect(r.hits).toEqual([]);
    expect(r.answer.length).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run — expect FAIL** (module missing).

Run: `pnpm vitest run lib/__tests__/discover-rank.test.ts`

- [ ] **Step 3: Implement** — `frontend/lib/discover-rank.ts`

```ts
import type { DiscoverIntent } from "./discover-intent";
import { haversineKm, type LatLon } from "./geo";
import { weekendRange } from "./mock-events";
import type { LiaEvent } from "./types";

export interface DiscoverHit {
  event: LiaEvent;
  matchReason: string;
  score: number;
}

export interface DiscoverRanking {
  answer: string;
  hits: DiscoverHit[];
  usedKidsFallback: boolean;
  omittedDistanceClaim: boolean;
}

const QUIET_SLUGS = new Set(["mediation", "reading-group", "exhibition"]);
const NEAR_KM = 5;
const KIDS_RE = /дет|семь|юн(ый|ая|ое|ые)|школ/i;
const DAY_MS = 24 * 60 * 60 * 1000;

const hourFmt = new Intl.DateTimeFormat("en-GB", {
  hour: "numeric",
  hourCycle: "h23",
  timeZone: "Europe/Moscow",
});
const weekdayFmt = new Intl.DateTimeFormat("en-US", {
  weekday: "short",
  timeZone: "Europe/Moscow",
});

export function moscowHour(iso: string): number {
  return Number(hourFmt.format(new Date(iso)));
}

function isMoscowWeekday(iso: string): boolean {
  const w = weekdayFmt.format(new Date(iso));
  return w !== "Sat" && w !== "Sun";
}

function primarySlug(e: LiaEvent): string | undefined {
  return e.categories[0]?.slug;
}

function sortUpcomingFirst(a: LiaEvent, b: LiaEvent, nowMs: number): number {
  const ta = new Date(a.startsAt).getTime();
  const tb = new Date(b.startsAt).getTime();
  const aUp = ta >= nowMs;
  const bUp = tb >= nowMs;
  if (aUp !== bUp) return aUp ? -1 : 1;
  return aUp ? ta - tb : tb - ta;
}

function takeTop3(scored: DiscoverHit[], nowMs: number): DiscoverHit[] {
  return [...scored]
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score;
      return sortUpcomingFirst(a.event, b.event, nowMs);
    })
    .slice(0, 3);
}

function quietReason(e: LiaEvent): string {
  if (e.capacity != null && e.capacity <= 12) return `группа ${e.capacity} человек`;
  if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) return "тихий формат";
  return "выходные";
}

function eveningReason(e: LiaEvent): string {
  return isMoscowWeekday(e.startsAt) ? "вечер, будний день" : "вечер";
}

function withinKm(e: LiaEvent, at: LatLon): boolean {
  const lat = e.venue?.lat;
  const lon = e.venue?.lon;
  if (lat == null || lon == null) return false;
  return haversineKm(at, [lat, lon]) <= NEAR_KM;
}

function textBlob(e: LiaEvent): string {
  return [e.title, e.description ?? "", e.organizer?.name ?? "", e.venue?.name ?? ""]
    .join(" ")
    .toLowerCase();
}

export function rankDiscover(
  events: readonly LiaEvent[],
  intent: DiscoverIntent,
  opts: { now: Date; userLatLon?: LatLon | null },
): DiscoverRanking {
  const nowMs = opts.now.getTime();
  const geo = opts.userLatLon ?? null;
  let usedKidsFallback = false;
  let omittedDistanceClaim = false;
  let scored: DiscoverHit[] = [];

  if (intent.id === "quiet_weekend") {
    const { from, to } = weekendRange(opts.now);
    scored = events
      .filter((e) => {
        const t = new Date(e.startsAt).getTime();
        return t >= from.getTime() && t < to.getTime();
      })
      .map((e) => {
        let score = 1;
        if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) score += 3;
        if (e.capacity != null && e.capacity <= 12) score += 2;
        return { event: e, score, matchReason: quietReason(e) };
      });
  } else if (intent.id === "quiet_any") {
    scored = events.map((e) => {
      let score = 1;
      if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) score += 3;
      if (e.capacity != null && e.capacity <= 12) score += 2;
      const reason =
        e.capacity != null && e.capacity <= 12
          ? `группа ${e.capacity} человек`
          : e.categories.some((c) => QUIET_SLUGS.has(c.slug))
            ? "тихий формат"
            : "ближайшие события";
      return { event: e, score, matchReason: reason };
    });
  } else if (intent.id === "free_nearby") {
    const free = events.filter((e) => e.priceType === "free");
    if (geo) {
      scored = free
        .filter((e) => withinKm(e, geo))
        .map((e) => ({ event: e, score: 2, matchReason: "бесплатно" }));
      omittedDistanceClaim = false;
    } else {
      scored = free.map((e) => ({ event: e, score: 1, matchReason: "бесплатно" }));
      omittedDistanceClaim = true;
    }
  } else if (intent.id === "evening_pair") {
    const until = nowMs + 7 * DAY_MS;
    scored = events
      .filter((e) => {
        const t = new Date(e.startsAt).getTime();
        return t >= nowMs && t < until && moscowHour(e.startsAt) >= 18;
      })
      .map((e) => {
        let score = 1;
        if (e.format === "offline") score += 1;
        if (e.capacity != null && e.capacity <= 40) score += 1;
        return { event: e, score, matchReason: eveningReason(e) };
      });
  } else if (intent.id === "with_kids") {
    const hard = events.filter((e) => KIDS_RE.test(textBlob(e)));
    if (hard.length > 0) {
      scored = hard.map((e) => ({
        event: e,
        score: 2,
        matchReason: "семейный формат",
      }));
    } else {
      usedKidsFallback = true;
      scored = events.map((e) => ({
        event: e,
        score: 1,
        matchReason: "ближайшие события",
      }));
    }
  } else {
    // keyword
    const needle = intent.keyword.toLowerCase();
    scored = events
      .filter((e) => textBlob(e).includes(needle))
      .map((e) => ({
        event: e,
        score: primarySlug(e) && needle.includes(primarySlug(e)!) ? 2 : 1,
        matchReason: "совпадение по тексту",
      }));
  }

  const hits = takeTop3(scored, nowMs);
  const n = hits.length;

  let answer: string;
  switch (intent.id) {
    case "quiet_weekend":
      answer =
        n === 0
          ? "Пока тихого на выходных не нашлось — соберите вручную ниже."
          : `Нашла ${n} ${n === 1 ? "событие" : "события"} на выходных — тихо и без спешки.`;
      break;
    case "quiet_any":
      answer =
        n === 0
          ? "Тихого формата пока не нашлось — соберите вручную ниже."
          : `Нашла ${n} ${n === 1 ? "событие" : "события"} в спокойном формате.`;
      break;
    case "free_nearby":
      answer = omittedDistanceClaim
        ? "Бесплатные события в городе."
        : "Бесплатные события рядом — до 5 км.";
      if (n === 0) {
        answer = omittedDistanceClaim
          ? "Бесплатных событий пока нет — соберите вручную ниже."
          : "Бесплатных рядом не нашлось — попробуйте без геолокации или ленту.";
      }
      break;
    case "evening_pair":
      answer =
        n === 0
          ? "Вечерних событий на ближайшие дни нет — соберите вручную ниже."
          : "Вечерние события на ближайшие дни.";
      break;
    case "with_kids":
      answer = usedKidsFallback
        ? "Ближайшие события — уточните на странице, подойдёт ли с детьми."
        : n === 0
          ? "Пока ничего не нашлось — соберите вручную ниже."
          : "События, где обычно бывает с детьми.";
      break;
    default:
      answer =
        n === 0
          ? `По запросу «${intent.sourceLabel}» ничего не нашлось.`
          : `По запросу «${intent.sourceLabel}» — вот что нашлось.`;
  }

  return { answer, hits, usedKidsFallback, omittedDistanceClaim };
}
```

**UI rule (Task 3):** hide the answer strip when `hits.length === 0` — show `EmptyState` instead. Ranker still always returns a non-empty `answer` for tests.

- [ ] **Step 4: Run — expect PASS**

Run: `pnpm vitest run lib/__tests__/discover-rank.test.ts`

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/discover-rank.ts frontend/lib/__tests__/discover-rank.test.ts
git commit -m "$(cat <<'EOF'
feat(frontend): add discover ranker and templated U3 answer sentences

EOF
)"
```

Green gate: `pnpm build && pnpm test && pnpm lint`.

---

### Task 3: `DiscoverBrowse` + `/search` shell

**Files:**
- Create: `frontend/components/DiscoverBrowse.tsx`
- Replace: `frontend/app/search/page.tsx`

**Interfaces:**
- Consumes: `DISCOVER_CHIPS`, `intentFromChip`, `parseDiscoverQuery`; `rankDiscover`; `fetchPublishedEvents`, `getCategories` (SSR); `eventToModuleProps`; `EmptyState`, `Skeleton`, `Chip`, `EventModule`; `ApiCategory`, `LiaEvent`.
- Produces: public `/search` page matching U3 layout; nav «Подбор» active via existing `USER_NAV` + `BottomTabBar`.

**State machine (`phase`):**

| Phase | When | UI |
|---|---|---|
| `idle` | no successful submit yet | prompt + chips + escape; no answer; no results grid |
| `loading` | submit in flight (geo wait and/or query) | prompt + chips; 3× `Skeleton` `h-[140px]` in results grid; no answer yet |
| `results` | ranking returned ≥1 hit | answer strip + 3-col/`EventModule`s |
| `empty` | ranking returned 0 hits | `EmptyState` + escape still shown |
| `error` | events query failed with no cached data | `EmptyState` error + retry button |

Geo: **only** when `intent.wantsNearby` — call `navigator.geolocation.getCurrentPosition` with `{ timeout: 10_000, maximumAge: 60_000, enableHighAccuracy: false }`. On deny/timeout/unavailable, proceed with `userLatLon: null` (city-wide free). Do **not** use Moscow-as-user-location pretending to be «рядом». Optional: `createTimedResolver` at 6s to stop waiting — then rank without geo.

- [ ] **Step 1: Replace the page shell** — `frontend/app/search/page.tsx`

```tsx
import { DiscoverBrowse } from "@/components/DiscoverBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { MOCK_EVENTS } from "@/lib/mock-events";

export const metadata = { title: "Подбор — PRESENCE" };

// U3 · AI-подбор. Public route — deterministic smart-filter (LLM deferred).
export default async function SearchPage() {
  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents().catch(() => MOCK_EVENTS),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="ПОДБОР" />
      <DiscoverBrowse initialEvents={initialEvents} categories={categories} />
    </>
  );
}
```

- [ ] **Step 2: Implement `DiscoverBrowse`** — layout classes must match the fidelity table.

```tsx
"use client";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { EventModule } from "@/components/ui/EventModule";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  fetchPublishedEvents,
  type ApiCategory,
} from "@/lib/api";
import {
  DISCOVER_CHIPS,
  intentFromChip,
  parseDiscoverQuery,
  type DiscoverIntent,
  type DiscoverIntentId,
} from "@/lib/discover-intent";
import { rankDiscover, type DiscoverHit } from "@/lib/discover-rank";
import { eventToModuleProps } from "@/lib/event-module";
import type { LatLon } from "@/lib/geo";
import type { LiaEvent } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useState } from "react";

type Phase = "idle" | "loading" | "results" | "empty" | "error";

export function DiscoverBrowse({
  initialEvents,
  categories,
}: {
  initialEvents: LiaEvent[];
  categories: ApiCategory[];
}) {
  const router = useRouter();
  const [draft, setDraft] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [answer, setAnswer] = useState("");
  const [hits, setHits] = useState<DiscoverHit[]>([]);
  const [activeChip, setActiveChip] = useState<DiscoverIntentId | null>(null);
  const [lastIntent, setLastIntent] = useState<DiscoverIntent | null>(null);
  const [now] = useState(() => new Date());

  const { data: events = initialEvents, isError, refetch } = useQuery({
    queryKey: ["events", "published", null, null],
    queryFn: () => fetchPublishedEvents(),
    initialData: initialEvents,
  });

  async function resolveGeo(wantsNearby: boolean): Promise<LatLon | null> {
    if (!wantsNearby || typeof navigator === "undefined" || !navigator.geolocation) {
      return null;
    }
    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (pos) => resolve([pos.coords.latitude, pos.coords.longitude]),
        () => resolve(null),
        { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
      );
    });
  }

  async function runIntent(intent: DiscoverIntent, chipId: DiscoverIntentId | null) {
    setLastIntent(intent);
    setActiveChip(chipId);
    setPhase("loading");
    try {
      if (isError && events.length === 0) {
        setPhase("error");
        return;
      }
      const userLatLon = await resolveGeo(intent.wantsNearby);
      const catalogue = events.length > 0 ? events : initialEvents;
      const ranking = rankDiscover(catalogue, intent, { now, userLatLon });
      setAnswer(ranking.answer);
      setHits(ranking.hits);
      setPhase(ranking.hits.length > 0 ? "results" : "empty");
    } catch {
      setPhase("error");
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const intent = parseDiscoverQuery(draft);
    if (!intent) return;
    void runIntent(intent, null);
  }

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      <div className="border-b border-ink px-[20px] py-[18px] max-sm:px-[14px] max-sm:py-[13px]">
        <p className="cap mb-[6px] hidden sm:block">AI-подбор</p>
        <h1 className="mb-[14px] text-[34px] font-black leading-[0.94] tracking-[-0.03em] max-sm:mb-[10px] max-sm:text-[20px]">
          Что вам сейчас
          <br className="max-sm:hidden" />{" "}
          откликается?
        </h1>
        <form onSubmit={onSubmit} className="flex border border-ink">
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Хочу спокойный вечер с искусством и без толпы…"
            aria-label="Запрос для подбора"
            className="swiss-focus min-h-[44px] flex-1 bg-transparent px-[14px] py-[11px] text-[12px] text-field-text placeholder:text-field-text max-sm:px-[10px] max-sm:py-[9px] max-sm:text-[10.5px] max-sm:placeholder:text-[10.5px]"
          />
          <button
            type="submit"
            aria-label="Подобрать"
            className="swiss-focus hover-invert min-h-[44px] bg-ink px-[20px] py-[11px] text-[12px] font-bold text-white max-sm:px-[13px] max-sm:py-[9px] max-sm:text-[11px]"
          >
            →
          </button>
        </form>
        <div className="mt-[10px] flex gap-[6px] overflow-x-auto [scrollbar-width:none] max-sm:mt-[8px] max-sm:gap-[5px] [&::-webkit-scrollbar]:hidden">
          {DISCOVER_CHIPS.map((c) => (
            <Chip
              key={c.id}
              variant={activeChip === c.id ? "active" : "default"}
              className="min-h-[44px] max-sm:text-[8px]"
              onClick={() => {
                setDraft(c.label);
                void runIntent(intentFromChip(c.id), c.id);
              }}
            >
              {c.label}
            </Chip>
          ))}
        </div>
      </div>

      {phase === "results" ? (
        <p className="border-b border-ink px-[20px] py-[12px] text-[12.5px] max-sm:px-[14px] max-sm:py-[10px] max-sm:text-[11px]">
          {answer}
        </p>
      ) : null}

      {phase === "loading" ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} className="h-[140px]" />
          ))}
        </div>
      ) : null}

      {phase === "results" ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
          {hits.map((h) => (
            <EventModule
              key={h.event.id}
              {...eventToModuleProps(h.event, categories)}
              matchReason={h.matchReason}
            />
          ))}
        </div>
      ) : null}

      {phase === "empty" ? (
        <EmptyState
          numeral="00"
          title="Ничего не нашлось"
          text="Попробуйте другой запрос или соберите ленту вручную."
          actions={
            <Button
              variant="ghost"
              onClick={() => {
                setPhase("idle");
                setHits([]);
                setAnswer("");
                setActiveChip(null);
                setDraft("");
              }}
            >
              Сбросить
            </Button>
          }
        />
      ) : null}

      {phase === "error" ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить события"
          text="Проверьте соединение и попробуйте ещё раз."
          actions={
            <Button
              variant="ghost"
              onClick={() => {
                void refetch().then(() => {
                  if (lastIntent) void runIntent(lastIntent, activeChip);
                });
              }}
            >
              Повторить
            </Button>
          }
        />
      ) : null}

      <div className="flex items-center justify-between gap-[12px] border-b border-ink px-[20px] py-[10px] max-sm:px-[14px]">
        <span className="cap">Не то? Соберите вручную</span>
        <Chip
          className="min-h-[44px]"
          onClick={() => router.push("/")}
        >
          Точные фильтры →
        </Chip>
      </div>
    </main>
  );
}
```

Geo: **only** when `intent.wantsNearby`. On deny/timeout → `userLatLon: null` (city-wide free). Do **not** substitute Moscow coordinates as «рядом».

- [ ] **Step 3: Typecheck / unit tests still pass**

Run: `pnpm vitest run lib/__tests__/discover-intent.test.ts lib/__tests__/discover-rank.test.ts`
Run: `pnpm build && pnpm test && pnpm lint`

- [ ] **Step 4: Commit**

```bash
git add frontend/components/DiscoverBrowse.tsx frontend/app/search/page.tsx
git commit -m "$(cat <<'EOF'
feat(frontend): ship U3 Podbor smart-filter on /search

EOF
)"
```

---

### Task 4: Browser fidelity verification

**Files:** none (manual / browser tools). Serve handoff for side-by-side:

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
# http://localhost:8099/Presence%20Swiss%20Grid%20-%20Full%20System.dc.html
```

App: `pnpm dev` in `frontend/` (or existing stack).

- [ ] **Step 1: Desktop ≥1024** — `/search` vs HTML U3: caption, 34px title, input+ink `→`, four chips, escape footer. Submit a chip → one answer sentence + ≤3 modules with `Совпало:`.
- [ ] **Step 2: Mobile 390** — header caption `ПОДБОР`; title ~20px; stacked modules; `BottomTabBar` «Подбор» active; tap targets ≥44px.
- [ ] **Step 3: Behaviour matrix**

| Action | Expect |
|---|---|
| Each of 4 chips | answer + ≤3 modules with `Совпало:` |
| Free text «бесплатно» | free events; sentence without distance if geo denied |
| Nonsense that matches nothing | EmptyState; escape chip still there |
| Escape `Точные фильтры →` | navigates to `/` |
| Nav + tab «Подбор» | active on `/search` |
| Grep new files | no `rounded-` / `shadow-` utilities |

- [ ] **Step 4: Final green gate**

Run: `pnpm build && pnpm test && pnpm lint`
Expected: PASS (test count ≥ prior 107 + new discover tests).

- [ ] **Step 5: Commit only if verification prompted copy/CSS tweaks**; otherwise note “no code changes” in the phase report.

---

### Task 5: Phase report + merge notes (no deploy required)

- [ ] **Step 1: Write** `docs/superpowers/reports/2026-07-29-swiss-grid-phase-4.md` (short):

Include:
- Shipped: U3 smart-filter on `/search`; AI / `POST /discover` still deferred.
- Heuristic limitations: kids keywords; free+geo optional; quiet = category+capacity heuristics not true “crowd” data.
- Deviations list (copy from this plan header).
- Verify commands + fidelity checklist results.
- Deploy: optional — may bundle with a later phase per product preference (P3 deploy follow-up still open).

- [ ] **Step 2: Commit report**

```bash
git add docs/superpowers/reports/2026-07-29-swiss-grid-phase-4.md
git commit -m "$(cat <<'EOF'
docs: add Swiss Grid Phase 4 U3 Podbor report

EOF
)"
```

- [ ] **Step 3: Open PR** when human asks — title `Swiss Grid Phase 4 — U3 Подбор (smart-filter)`; base `main`; do not force-push.

---

## Self-review (planning)

| Spec requirement | Task |
|---|---|
| Replace ComingSoon; U3 layout | Task 3 |
| Four chips + free text + templated answer | Tasks 1–3 |
| ≤3 EventModule + matchReason | Task 2–3 |
| Empty / loading Skeleton / error | Task 3 |
| Escape hatch → `/` | Task 3 |
| Pure helpers + Vitest TDD | Tasks 1–2 |
| No POST /discover / LLM | Global + deviations |
| Route stays `/search` | Task 3 + deviations |
| Fidelity numbers pasted | Design fidelity contract |
| Browser + green gate | Task 4 |
| Phase report / AI deferred note | Task 5 |
| Inventory DiscoveryFeed API | File Structure |
| No reopen P1–P3 deviations | Deliberate deviations section |

**Conflicts with Global Constraints:** none expected. 44px vs mock padding: height grows, type stays. Mobile chip truncation: deliberate deviation #6 (full labels + scroll).

**Placeholder scan:** no TBD/TODO left; Task 1–3 include full test + implementation bodies.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-29-swiss-grid-phase-4-u3-podbor.md`.**

Do **not** implement until this plan is reviewed. At execution time:

1. Create worktree/branch `redesign/swiss-grid-p4` from `main` @ ≥`85d40b2` (`superpowers:using-git-worktrees`).
2. Choose execution mode:

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks (`superpowers:subagent-driven-development`).

**2. Inline Execution** — this session with `superpowers:executing-plans` and checkpoints.

Which approach?
