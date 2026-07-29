# HANDOFF — Swiss Grid after Phase 6

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip:** `redesign/swiss-grid-p6` @ `cd97efe` (merge to `main` separately)  
**Live product:** https://presence.tarski.ru (confirm deploy separately — merge ≠ deploy; **Phase 6 deploy deferred pending human request**)

---

## What shipped

| Phase | Scope | Status |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged |
| **P2** Public core | U1 feed, U2 detail, U7 auth, U8 states | Merged |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged |
| **P4** U3 Подбор | `/search` deterministic smart-filter | Merged |
| **P5** Organizer | O1–O5 Swiss suite | Merged |
| **P6** Admin ink | A1–A3 + users stub + ink secondary | Branch ready — merge/deploy pending |

**Not shipped:**

| Design | App route | Next phase |
|---|---|---|
| **A4** Users / content hygiene | `/admin/users` is stub only | **Phase 7** ← next (+ backend) |

---

## Phase 6 notes (for context, do not reopen)

- Ink surface on admin layout; `ADMIN_NAV`; A2/A3 desktop-only &lt;900px; A1 duty mode on `max-sm`.
- A2 is a visual conveyor over published/rejected + takedown/reinstate — not a pre-publish desk.
- Stub tiles/`—` for missing overview fields (org total, users); complaints filter EmptyState; test heuristic client-side.
- `/admin/users` = Phase-7 placeholder; `/admin/moderation/organizers` redirects to A3 `?filter=pending`.
- Locked P1–P5 deviations still locked.

**Plan / report:**  
`docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md`  
`docs/superpowers/reports/2026-07-29-swiss-grid-phase-6.md`

---

## Design authority

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` (badge `A4` ~985) |
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

## Code landmarks for Phase 7 (A4 + hygiene + backend)

| Need | Where |
|---|---|
| Users stub to replace | `frontend/app/admin/users/page.tsx` |
| Admin layout / ink | `frontend/app/admin/layout.tsx` |
| ADMIN_NAV | `AppHeader` `ADMIN_NAV` (`/admin/users` already linked) |
| Test heuristic (reuse) | `frontend/lib/admin-test-heuristic.ts` |
| Complaints page | `frontend/app/admin/complaints/page.tsx` |
| Backend gaps | Users registry API; bulk hide-from-feed; per-org complaint counts; delete content hygiene |

---

## Recommended next work

**Primary:** Phase 7 — **A4** Users registry + content hygiene (+ required backend). Do not reopen A1–A3 deviations without product sign-off. Live pixel QA of A1–A3 as admin before claiming Phase 6 pixel-complete.

---

## Minimal prompt for the planning agent

```
You are planning (NOT implementing yet) Swiss Grid Phase 7 — Admin A4 Users + content hygiene (+ backend as needed).

Read first:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-6-HANDOFF.md
2. docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md (§ Phase 7 / remaining admin + Known backend gaps)
3. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (§ A4 + Admin inverted rules)
4. Presence Swiss Grid - Full System.dc.html badge «A4» (~985)
5. Phase 6 plan/report for locked A1–A3 deviations:
   docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md
   docs/superpowers/reports/2026-07-29-swiss-grid-phase-6.md

Then use superpowers:writing-plans →
docs/superpowers/plans/YYYY-MM-DD-swiss-grid-phase-7-admin-a4.md

Branch later from main (tip must include Phase 6 merge). Worktree redesign/swiss-grid-p7.
Do not implement until the plan is reviewed.
Do not reopen locked P1–P6 deviations.
Replace `/admin/users` stub with A4; call out backend work explicitly (registry, hygiene actions, complaint counts if still missing).
Ship ink surface continuity with A1–A3.

Please check designs against HTML badge «A4» and put a Design fidelity contract in the plan.
```
