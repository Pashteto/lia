# HANDOFF — Swiss Grid after Phase 4

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip on `main`:** `37f769e` — Phase 4 U3 Подбор + HTML fidelity fixes  
**Live product:** https://presence.tarski.ru (confirm deploy separately — merge ≠ deploy)

---

## What shipped

| Phase | Scope | Status on `main` |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged |
| **P2** Public core | U1 feed, U2 detail, U7 auth, U8 states | Merged |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged |
| **P4** U3 Подбор | `/search` deterministic smart-filter (non-AI) | Merged |

**Not shipped:**

| Design | App route | Next phase |
|---|---|---|
| **O1–O5** Organizer | `/organizer`, `/events/mine`, `/events/new`, `/organizer/applications`, `/me/organizer` | **Phase 5** ← next |
| **A1–A3** Admin ink | `/admin/*` | Phase 6 |
| **A4** Users/hygiene | missing | Phase 7 (+ backend) |

---

## Phase 4 notes (for context, do not reopen)

- Screen still titled `AI-подбор`; answers are **templated**, not LLM. `POST /discover` deferred (`lia-ai-provider-constraint`).
- Route stays `/search` (not `/discover`).
- Pure helpers: `frontend/lib/discover-intent.ts`, `frontend/lib/discover-rank.ts`; UI: `DiscoverBrowse` + thin `app/search/page.tsx`.
- Geo for «Бесплатно рядом»: browser geolocation + client `haversineKm` on published catalogue — **no** `fetchNearbyEvents`.
- Mobile chips use short labels (Тихое / Бесплатно / Вечер / С детьми); desktop full labels.
- Fidelity: HTML badge `U3 · AI-подбор`; README § U3. EventModule `matchReason` appends under date/price (README “append”; mock omits price — intentional).

**Locked P1–P3 deviations:** still locked (favorites deferred, Yandex not OSM, Golos/Manrope fonts, home `/`, U4 personal calendar, etc.).

---

## Design authority

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` (badges `O1`…`O5`) |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` |
| P4 plan / report | `…/2026-07-29-swiss-grid-phase-4-u3-podbor.md`, `…/reports/2026-07-29-swiss-grid-phase-4.md` |

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
```

---

## Hard constraints (every phase)

- Zero border radius, zero shadows; 1px hairlines.
- Categories = numerals via `categoryNumeral`; all numbers JetBrains Mono; unknown → `—`.
- `signal` red only for needs-attention; uppercase tracking preserved.
- `hover-invert`, `swiss-focus`, 120ms linear; Skeleton not spinners.
- Empty / gated / error = `EmptyState` / `AuthGate` / designed 404.
- Tap targets ≥44px; content max 1360px; Moscow formatters; React #418.
- UI copy Russian; code/commits English; fonts Golos Text / Manrope / JetBrains Mono.
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Code landmarks for Phase 5 (Organizer)

| Need | Where |
|---|---|
| O1 shell | `frontend/app/organizer/page.tsx` |
| O3 mine | `frontend/app/events/mine/page.tsx` |
| O2 create/edit | `frontend/components/CreateEventForm.tsx`, `app/events/new`, `app/events/[id]/edit` |
| O4 applications | `frontend/app/organizer/applications/`, `EventApplicationsPanel` |
| O5 org profile | `frontend/app/me/organizer/`, public `app/organizers/[id]` |
| Primitives | `Cell`, `CellStrip`, `Chip`, `Button`, `AppHeader`, `Stepper`, `ProgressBar`, `StatusChip`, `EventModule` |
| Auth gate | `AuthGate` — organizer routes already gated |

**Tests on `main`:** ~120 Vitest cases (frontend).

---

## Recommended next work

**Primary:** Phase 5 — Organizer suite O1 → O3 → O2 → O4 → O5 (master plan order).

---

## Minimal prompt for the planning agent

```
You are planning (NOT implementing yet) Swiss Grid Phase 5 — Organizer suite (O1–O5).

Read first:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-4-HANDOFF.md
2. docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md (§ Phase 5 + Global Constraints + Known backend gaps)
3. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (§ O1–O5)
4. Presence Swiss Grid - Full System.dc.html badges «O1» «O2» «O3» «O4» «O5»
5. Prior phase plans for patterns: docs/superpowers/plans/2026-07-28-swiss-grid-phase-3-*.md and 2026-07-29-swiss-grid-phase-4-u3-podbor.md (thin shells, TDD pure helpers, deliberate deviations header)

Then use superpowers:writing-plans →
docs/superpowers/plans/2026-07-29-swiss-grid-phase-5-organizer.md

Branch later from main (tip must include Phase 4 merge). Worktree redesign/swiss-grid-p5.
Do not implement until the plan is reviewed.
Do not reopen locked P1–P4 deviations.
Ship against existing organizer APIs; flag any unavoidable backend gaps (e.g. activity log, org summary, multi-event bulk) as deliberate stubs or client-side loops — same class as P3/P4.
Suggested screen order (master plan): O1 → O3 → O2 → O4 → O5; deploy+verify last.
```
