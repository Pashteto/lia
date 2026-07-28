# Spec — Swiss Grid Phase 4: U3 Подбор (planning brief)

> **Audience:** planning agent. Expand this into a full implementation plan via  
> **superpowers:writing-plans** → save as  
> `docs/superpowers/plans/2026-07-29-swiss-grid-phase-4-u3-podbor.md`.  
> **Do not write production code in the planning session** unless the human explicitly asks to execute.

**Goal:** Replace the `/search` `ComingSoon` stub with the Swiss Grid **U3 · AI-подбор** screen — shipped as a **deterministic smart-filter** experience that matches the handoff layout. Real model-generated answers (`POST /discover`) stay deferred until AI provider sign-off.

**Why this phase next:** After P3, U3 is the only remaining **user-layer** design screen still stubbed. Nav («Подбор») and tab bar already link to `/search`. Organizer (P5) and Admin (P6) are larger and can wait or run in a parallel worktree later.

---

## Sources of truth

| Kind | Location |
|---|---|
| Handoff (state) | `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-3-HANDOFF.md` |
| Master roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` — Phase 4 bullets + Known backend gaps |
| Design behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` — § U3 + EventModule AI variant |
| Pixel reference | `Presence Swiss Grid - Full System.dc.html` — badge `U3 · AI-подбор` (~line 240) |
| Prior patterns | Phase 2 `DiscoveryFeed` / `EventModule`; Phase 3 thin shells |

**Prerequisite:** `main` at or after `85d40b2` (Phase 3 merge). Verify:

```bash
git log --oneline -1   # expect Phase 3 merge or later
test -f frontend/components/ui/EventModule.tsx
test -f frontend/app/search/page.tsx
```

**Branch / worktree (at execution time):** `redesign/swiss-grid-p4` via `superpowers:using-git-worktrees`.

---

## Product definition (non-AI v1)

### User story

As a visitor on «Подбор», I type or tap a suggestion, see **one sentence** explaining the selection, see **up to three** event modules with `Совпало: …` reasons, and can always escape to the feed filters via «Точные фильтры →». The screen never dead-ends and never pretends to be a chat.

### Must match (layout / copy)

From README U3 + HTML:

**Desktop**

1. `AppHeader` with «Подбор» active (`USER_NAV` already has `/search`).
2. Prompt block: `.cap` `AI-подбор` → H1 34px/900 `Что вам сейчас откликается?` (two lines in mock).
3. Input row: `1px solid #111`, flex; text field `12px` / `#6B665E` placeholder; ink submit square `→` (`padding: 11px 20px`).
4. Four suggestion chips: `Тихое в выходные` · `Бесплатно рядом` · `Для двоих вечером` · `С детьми`.
5. Answer strip: single bordered row, **one sentence** `12.5px` (never a transcript).
6. Results: `1fr 1fr 1fr` of `EventModule` with `matchReason` (already supported).
7. Escape hatch footer: `.cap` `Не то? Соберите вручную` + chip `Точные фильтры →` → `/` (feed). Optional: deep-link query params if cheap — not required for v1.

**Mobile**

- Header caption `ПОДБОР`; title ~20px; denser input/chips; stacked result rows; `BottomTabBar` with Подбор active; tap targets ≥44px.

### Must implement (behaviour)

| Input | Deterministic behaviour (planner must refine exact rules + tests) |
|---|---|
| Free-text submit / chip click | Map intent → filter predicates over existing events API (same catalogue as feed / nearby) |
| Empty / no matches | U8 `EmptyState` + still show escape hatch |
| Loading | `Skeleton` modules at final dimensions — no spinner |
| Error (API) | U8 error empty with retry |

**Suggested chip → intent mapping (starting point for the plan — adjust with tests):**

| Chip | Intent sketch | Reuse |
|---|---|---|
| Тихое в выходные | Weekend window + prefer small/mediation-ish categories if detectable | `weekendRange` |
| Бесплатно рядом | `priceType === "free"` + nearby if geolocation available else free-only | `MapBrowse` / `haversineKm` patterns |
| Для двоих вечером | Evening starts (e.g. hour ≥ 18 Moscow) in coming days | Moscow time helpers |
| С детьми | Prefer family-friendly categories if tagged; else soft text match / fallback sentence | Category list from API |

Free-text: keyword / category / «бесплатно» / «выходн» heuristics → same predicate engine; answer sentence is a **templated** Russian one-liner from the applied filters (not LLM).

**Cap results at 3** (handoff shows three modules). Prefer upcoming published events; stable sort (e.g. startsAt asc).

### Explicitly out of scope (Phase 4)

- `POST /discover` / any LLM / streaming chat.
- Favorites / ♡.
- Changing feed filter UX beyond the escape-hatch link.
- Organizer / admin screens.
- Renaming `/search` → `/discover` (keep `/search` to avoid nav churn; note in plan if design insists later).

---

## Architecture expectations

Follow Phase 2–3 patterns:

```
frontend/app/search/page.tsx          REPLACE — thin shell + AppHeader + client body
frontend/components/DiscoverBrowse.tsx  CREATE — client UI (name bikeshed OK; keep Discover* or Podbor*)
frontend/lib/discover-intent.ts         CREATE — pure: parse query/chip → filters + answer template
frontend/lib/discover-rank.ts           CREATE — pure: filter/score/take 3 + matchReason strings
frontend/lib/__tests__/discover-*.ts    CREATE — TDD
```

- Prefer **pure helpers + Vitest** for all matching/ranking/sentence logic.
- Data: reuse existing list endpoints used by the feed (`getEvents` / whatever `DiscoveryFeed` uses) — **no new backend** in v1 unless the planner finds an unavoidable gap (then flag to human).
- Geolocation: optional enhancement for «рядом»; must degrade without permission (same class of problem as U5 — use timed fallback if prompting).

---

## Global constraints

Inherit master plan + Phase 3 Global Constraints verbatim (radius/shadows, mono numbers, signal-red, tracking, AuthGate/EmptyState, 44px, Moscow TZ, #418, Russian UI copy, green commits).

**Fonts:** Golos Text / Manrope / JetBrains Mono only.

**EventModule:** use existing `matchReason` prop; do not invent a parallel card.

---

## Deliberate deviations for Phase 4 (pre-decide in the implementation plan header)

Propose these in the written plan (human can override before execute):

1. **Non-AI brain.** Screen title/copy still say `AI-подбор` per handoff, but answers are templated. Optional tiny `.cap` honesty line is **not** in the mock — default to matching mock copy; if product wants disclosure, ask before shipping.
2. **Route stays `/search`.** Design path `/discover` is conceptual.
3. **«Бесплатно рядом» without geo** → free events city-wide + answer sentence that omits distance claims.
4. **«С детьми»** without a dedicated category → best-effort category/title heuristics; if weak, still return 3 upcoming + honest reason like `Совпало: ближайшие события` rather than inventing fitness.
5. **No chat history / multi-turn.** One query → one answer strip → results; new submit replaces.

---

## Fidelity contract (planner must paste measured numbers into the plan)

From HTML U3 desktop (~240–280) / mobile:

| Element | Desktop | Mobile |
|---|---|---|
| Title | 34px / 900 / -0.03em | ~20px / 900 |
| Prompt block pad | 18px 20px | 13px 14px |
| Input text | 12px / `#6B665E` | ~10.5px |
| Submit `→` | ink, 11px 20px pad | 9px 13px |
| Suggestion chips | default Chip; gap 6 | 8px type; gap 5 |
| Answer strip | 12.5px; pad 12px 20px | 11px; pad 10px 14px |
| Results | 3-col EventModule | stacked rows |
| Match reason | 10px; top rule `#DDD` | via EventModule / cap |
| Escape footer | pad 10px 20px; chip right | tab bar below |

---

## Verification (must appear as final tasks in the implementation plan)

- [ ] Desktop + 390px against HTML U3 side-by-side (serve handoff on :8099).
- [ ] Each of 4 chips produces a one-sentence answer + ≤3 modules with `Совпало:`.
- [ ] Free-text path works; empty → EmptyState; escape chip → `/`.
- [ ] Nav + tab bar «Подбор» active on `/search`.
- [ ] `pnpm build && pnpm test && pnpm lint` green.
- [ ] No `rounded-` / `shadow-` introduced on the new surface.
- [ ] Phase report notes: AI provider still deferred; any heuristic limitations.

---

## Planning checklist for the writing-plans agent

When expanding this spec into the implementation plan:

1. Inventory exact API functions `DiscoveryFeed` uses (file:line) and decide fetch strategy (client Query vs server).
2. Lock chip → filter → answer-template tables with **unit tests first** (TDD).
3. Spec `DiscoverBrowse` props/state machine (idle / loading / results / empty / error).
4. List every file Create/Modify/Test; bite-sized tasks with commits.
5. Call out any conflict with Global Constraints before execution.
6. End with browser verification + merge/deploy notes (deploy can be bundled with a later phase if product prefers).

**Success of the planning session:** a complete `2026-07-29-swiss-grid-phase-4-u3-podbor.md` implementation plan ready for `subagent-driven-development` — not a half-built `/search` page.
