# HANDOFF — Swiss Grid after Phase 3

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip on `main`:** `85d40b2` — `Merge branch 'redesign/swiss-grid-p3' — Swiss Grid Phase 3 (U4/U5/U6)`  
**Live product:** https://presence.tarski.ru (Phase 3 may or may not be deployed yet — confirm before assuming prod matches `main`)

---

## What shipped

| Phase | Scope | Status on `main` |
|---|---|---|
| **P1** Foundation | Tokens, fonts (Golos Text/Manrope/JB Mono), Cell/Chip/Button/AppHeader/BottomTabBar/Field/… | Merged |
| **P2** Public core | U1 feed `/`, U2 event detail, U7 login/signup, U8 EmptyState/AuthGate/404 | Merged |
| **P3** Calendar / map / profile | U4 `/me/calendar`, U5 `/map`, U6 `/me` (+ redirects), `/auth/me` `created_at` | Merged + pushed |

**Not shipped (still legacy or stub):**

| Design | App route | Next phase |
|---|---|---|
| **U3** AI-подбор | `/search` = `ComingSoon` | **Phase 4** ← next |
| **O1–O5** Organizer | `/organizer`, `/events/mine`, `/events/new`, `/organizer/applications`, `/me/organizer` | Phase 5 |
| **A1–A3** Admin ink | `/admin/*` | Phase 6 |
| **A4** Users/hygiene | missing | Phase 7 (+ backend) |

Page ↔ HTML matrix (interactive): ask the IDE for canvas `swiss-grid-page-mapping.canvas.tsx` if present, or rebuild from master-plan route table.

---

## Design authority (always)

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` (badges `U1`…`A4`) |
| Map marker geometry | `…/map-embed.html` (numeral pins win over venue-label pins — P3 decision) |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` |
| Executed plans | `…/2026-07-28-swiss-grid-phase-{1,2,3}-*.md` |

Serve the handoff folder for fidelity (needs `support.js`):

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
# http://localhost:8099/Presence%20Swiss%20Grid%20-%20Full%20System.dc.html
```

---

## Hard constraints (carry into every phase)

Copy from master plan / Phase 3 Global Constraints — do not dilute:

- Zero border radius, zero shadows; 1px hairlines (`#111` structural, `#DDD` inner, `#E0DCD4` calendar cells only).
- Categories = numerals via `categoryNumeral(slug, orderedCategories)` — never colours.
- All numbers in JetBrains Mono; unknown → `—`.
- `signal` red only for needs-attention (never for selection).
- Uppercase tracking preserved (`.cap` 0.13em, chips 0.12em, buttons 0.07em, nav 0.14em).
- `hover-invert`, `swiss-focus` (2px square), 120ms linear; Skeleton not spinners.
- Empty / gated / error = `EmptyState` / `AuthGate` / designed 404.
- Tap targets ≥44px on touch (mock density loses to 44px; type sizes stay).
- Content max 1360px; Moscow-pinned formatters; React #418 (no nested `<a>`).
- UI copy Russian; code/commits English.
- Fonts stay **Golos Text / Manrope / JetBrains Mono** (Archivo & Space Grotesk lack Cyrillic).
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Deliberate deviations already locked (do not reopen without product sign-off)

Phase 3 recorded **11** deviations (favorites→Заявки, U6 read-only, Yandex not OSM, stats-from-search, no bbox, CellStrip `#111`, font substitutes, U5 mobile list, U6 four mobile tabs, …). Full list: Phase 3 plan § «Deliberate deviations».

Also locked earlier:

- Home feed stays `/` (design says `/events`).
- U4 is personal calendar at `/me/calendar` (not public all-events `/calendar`).
- Favorites / ♡ deferred (no backend).
- `POST /discover` AI deferred until provider sign-off (`lia-ai-provider-constraint`).

---

## Code landmarks for the next phase

| Need | Where |
|---|---|
| Stub to replace | `frontend/app/search/page.tsx` (`ComingSoon`) |
| Nav already points at Подбор | `USER_NAV` + `BottomTabBar` → `/search` |
| Feed modules + AI reason prop | `EventModule` already accepts `matchReason?: string` → renders `Совпало: …` |
| Time/free filters to reuse | `DiscoveryFeed` + `lib/mock-events.ts` (`todayRange`, `weekendRange`) |
| Nearby / free patterns | `MapBrowse`, `mapAreaStats`, `priceLabel` |
| Swiss shells pattern | Thin `app/*/page.tsx` + client body (`MeProfile`, `CalendarView`, `MapBrowse`) |
| Auth / empty | `AuthGate`, `EmptyState`, `Skeleton` |
| Primitives | `Cell`, `CellStrip`, `Chip`, `Button`, `AppHeader`, `Field` |

**Tests on `main`:** 107 Vitest cases (frontend). Backend `go build ./...` may fail without generated swagger/proto trees (pre-existing) — domain packages still testable.

---

## Open follow-ups (not Phase 4 blockers)

1. **Deploy Phase 3 to prod** (Task 11 of P3 plan) — runbook path in Phase 2/3 docs; confirm Yandex key + migration `000021` if not already live.
2. Signed-in U4/U6 + live map pins fidelity — needs running API + maps key (documented in P3 Task 10).
3. Backend follow-ups from P3: real `bbox` events query; multi-event `.ics`.
4. Doc nit: optional — already fixed SearchField in README on merge tip.

---

## Recommended next work

**Primary:** Phase 4 — U3 `/search` smart-filter Подбор (non-AI).  
**Planning spec:** `docs/superpowers/specs/2026-07-29-swiss-grid-phase-4-u3-podbor.md`

**Alternative (parallel after P2):** Phase 5 Organizer O1–O5 — larger; only start if product prioritizes organizer UX over closing the last public user hole (`/search` stub).

---

## Minimal prompt for the planning agent

```
You are planning (NOT implementing yet) Swiss Grid Phase 4 — U3 Подбор.

Read first:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-3-HANDOFF.md
2. docs/superpowers/specs/2026-07-29-swiss-grid-phase-4-u3-podbor.md
3. docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md (§ Phase 4 + Global constraints)
4. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (§ U3 + EventModule AI variant)
5. Presence Swiss Grid - Full System.dc.html badge «U3 · AI-подбор» (~line 240)

Then use superpowers:writing-plans to produce:
docs/superpowers/plans/2026-07-29-swiss-grid-phase-4-u3-podbor.md

Branch later from main @ ≥85d40b2. Worktree redesign/swiss-grid-p4.
Do not implement until the plan is reviewed.
Do not reopen locked deviations from P1–P3.
Ship non-AI smart-filter first; real LLM / POST /discover is explicitly deferred.
```
