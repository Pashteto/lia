# HANDOFF — Swiss Grid after Phase 5

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip on `main`:** see `git log -1` after push (Phase 5 O1–O5 + HTML fidelity pass)  
**Live product:** https://presence.tarski.ru (confirm deploy separately — merge ≠ deploy)

---

## What shipped

| Phase | Scope | Status on `main` |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged |
| **P2** Public core | U1 feed, U2 detail, U7 auth, U8 states | Merged |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged |
| **P4** U3 Подбор | `/search` deterministic smart-filter | Merged |
| **P5** Organizer | O1–O5 Swiss suite | Merged |

**Not shipped:**

| Design | App route | Next phase |
|---|---|---|
| **A1–A3** Admin ink | `/admin`, `/admin/moderation/events`, `/admin/organizers` | **Phase 6** ← next |
| **A4** Users/hygiene | missing | Phase 7 (+ backend) |

---

## Phase 5 notes (for context, do not reopen)

- Routes kept: `/organizer`, `/events/mine`, `/events/new`, `/organizer/applications`, `/me/organizer`, `/organizers/[id]`.
- Activity log stubbed; org tiles client-derived; O4 bulk = sequential `decideApplication`; no telegram field.
- HTML fidelity pass after merge: Stepper `Шаг NN` + paper ink|ink dividers; O2 mobile CTA/note; O3 mobile header `+`; O4 row hairlines; O5 empty logo frame.
- Locked P1–P4 deviations still locked (favorites, Yandex, Golos/Manrope, home `/`, U3 non-AI, etc.).

**Plan / report:**  
`docs/superpowers/plans/2026-07-29-swiss-grid-phase-5-organizer.md`  
`docs/superpowers/reports/2026-07-29-swiss-grid-phase-5.md`

---

## Design authority

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` (badges `A1`…`A3`) |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` |

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
- Admin = `data-surface="ink"` inversion; desktop-only except A1 duty mode.
- Every commit: `pnpm build && pnpm test && pnpm lint` from `frontend/`.

---

## Code landmarks for Phase 6 (Admin)

| Need | Where |
|---|---|
| Admin layout / ink surface | `frontend/app/admin/layout.tsx` |
| A1 overview | `frontend/app/admin/page.tsx` |
| A2 moderation | `frontend/app/admin/moderation/events/` |
| A3 organizers | `frontend/app/admin/organizers/` (+ absorb `/admin/moderation/organizers`) |
| Header | `AppHeader` `admin` + `ADMIN_NAV` (already defined) |
| Primitives | `Cell`, `Chip`, `Button`, `StatusChip`, `Skeleton`, `EmptyState` |

---

## Recommended next work

**Primary:** Phase 6 — Admin inverted A1 → A2 → A3 (master plan order). A4 stays Phase 7.

---

## Minimal prompt for the planning agent

```
You are planning (NOT implementing yet) Swiss Grid Phase 6 — Admin inverted suite (A1–A3).

Read first:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-5-HANDOFF.md
2. docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md (§ Phase 6 + Global Constraints + Known backend gaps + Decision checkpoint on admin ink surface)
3. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (§ A1–A3 + Admin inverted rules)
4. Presence Swiss Grid - Full System.dc.html badges «A1» «A2» «A3» (A4 is Phase 7 — out of scope)
5. Prior phase plans for patterns: docs/superpowers/plans/2026-07-29-swiss-grid-phase-5-organizer.md and 2026-07-29-swiss-grid-phase-4-u3-podbor.md (thin shells, TDD pure helpers, deliberate deviations header, HTML fidelity contract)

Then use superpowers:writing-plans →
docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md
(or dated 2026-07-30 if written the next calendar day)

Branch later from main (tip must include Phase 5 merge). Worktree redesign/swiss-grid-p6.
Do not implement until the plan is reviewed.
Do not reopen locked P1–P5 deviations.
Ship against existing admin APIs (moderation decide, organizer verify); flag any unavoidable backend gaps as deliberate stubs — same class as P3–P5.
Suggested screen order (master plan): A1 → A2 → A3; deploy+verify last.
A4 Users/hygiene is Phase 7 — mention only as out-of-scope.

Please check designs against the HTML badges «A1» «A2» «A3» and put a Design fidelity contract in the plan (desktop A2–A3; A1 also has mobile duty mode).
```
