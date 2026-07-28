# HANDOFF — Swiss Grid after Phase 5

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip (worktree):** `32280a5` — Phase 5 Organizer O1–O5  
**Live product:** https://presence.tarski.ru (**deploy not run** in Phase 5 close — merge ≠ deploy)

---

## What shipped

| Phase | Scope | Status |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged |
| **P2** Public core | U1 feed, U2 detail, U7 auth, U8 states | Merged |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged |
| **P4** U3 Подбор | `/search` deterministic smart-filter | Merged |
| **P5** Organizer O1–O5 | `/organizer`, `/events/mine`, `/events/new`, `/organizer/applications`, `/me/organizer`, public `/organizers/[id]` | **Branch `redesign/swiss-grid-p5`** — ready for merge |

**Not shipped:**

| Design | App route | Next phase |
|---|---|---|
| **A1–A3** Admin ink | `/admin`, `/admin/moderation/events`, `/admin/organizers` | **Phase 6** ← next |
| **A4** Users/hygiene | missing | Phase 7 (+ backend) |

---

## Phase 5 notes (for context, do not reopen)

- All organizer routes use existing APIs; backend gaps are deliberate stubs/loops (activity log, org summary, bulk decide, follower count, telegram contact).
- O1 activity rail = designed stub caption only.
- O4 bulk = sequential `decideMany`; mobile hides bulk bar (per-row actions only).
- O2 stepper is visual; schema unchanged; create = ghost draft, edit = blur autosave.
- O5 verification stepper uses `fillMode="exclusive"`; public view matches mobile mock on `/organizers/[id]`.
- Tab bar hidden on `/me/organizer` via `TabBarGate`.

**Pixel QA ledger (still open):** live browser pass O1–O5 @ 390px + desktop vs HTML badges not completed in phase close; O4 optimistic accept not verified against live API. Do not claim pixel-complete until human checklist passes.

**Locked P1–P4 deviations:** still locked (favorites deferred, Yandex not OSM, Golos/Manrope fonts, home `/`, U4 personal calendar, U3 non-AI, etc.).

---

## Design authority

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` (badges `A1`…`A3` next; `O1`…`O5` done) |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` |
| P5 plan / report | `…/2026-07-29-swiss-grid-phase-5-organizer.md`, `…/reports/2026-07-29-swiss-grid-phase-5.md` |

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

## Code landmarks — Phase 5 (Organizer, shipped)

| Need | Where |
|---|---|
| O1 hub | `frontend/app/organizer/page.tsx`, `components/OrganizerHub.tsx` |
| O3 mine | `frontend/app/events/mine/page.tsx`, `components/MyEventsBrowse.tsx` |
| O2 create/edit | `components/CreateEventForm.tsx`, `app/events/new`, `app/events/[id]/edit` |
| O4 applications | `app/organizer/applications/`, `OrganizerApplications.tsx`, `EventApplicationsPanel.tsx` |
| O5 edit + public | `app/me/organizer/`, `OrganizerProfileEdit.tsx`; `app/organizers/[id]/`, `PublicOrganizerView.tsx` |
| Pure helpers | `lib/org-dashboard.ts`, `org-seats.ts`, `org-event-status.ts`, `org-verification.ts`, `org-applications.ts`, `org-duplicate-event.ts`, `relative-time.ts` |
| Primitives | `Cell`, `CellStrip`, `Chip`, `Button`, `AppHeader`, `Stepper` (`fillMode`), `ProgressBar`, `StatusChip`, `EventModule`, `Field`, `EmptyState`, `AuthGate` |
| Nav | `ORG_NAV` in `components/ui/AppHeader.tsx` |
| Tab bar gate | `TabBarGate.tsx` — `/me/organizer` hidden |

**Tests on branch:** **140** Vitest cases (33 files).

---

## Code landmarks — Phase 6 (Admin A1–A3, next)

| Need | Where (pre-Swiss, restyle targets) |
|---|---|
| Admin layout | `frontend/app/admin/layout.tsx` — wrap in `data-surface="ink"` |
| A1 dashboard | `frontend/app/admin/page.tsx` |
| A2 event moderation | `frontend/app/admin/moderation/events/page.tsx` |
| A3 organizers | `frontend/app/admin/organizers/page.tsx` (+ absorb `/admin/moderation/organizers`) |
| Legacy admin nav | Replace bespoke nav with `AppHeader` admin variant (`PRESENCE / ADMIN`) |
| Complaints / settings | `app/admin/complaints/`, `app/admin/settings/` — out of A1–A3 scope unless product says otherwise |

Master plan § Phase 6: four stat tiles on A1 (Ждут модерации on `#E2231A`), A2 queue/record split with rejection-reason chips, A3 filter table with contextual row actions. Desktop-only (min-width notice below 900px) except A1 duty mode.

---

## Deploy

**Not performed** in Phase 5 close. When ready, use `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`. Bundling P3–P5 deploy is a product choice.

---

## Recommended next work

**Primary:** Phase 6 — Admin inverted suite **A1 → A2 → A3** (master plan order), then deploy+verify.

---

## Minimal prompt for the planning agent

```
You are planning (NOT implementing yet) Swiss Grid Phase 6 — Admin suite (A1–A3).

Read first:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-5-HANDOFF.md
2. docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md (§ Phase 6 + Global Constraints)
3. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (§ A1–A3)
4. Presence Swiss Grid - Full System.dc.html badges «A1» «A2» «A3»
5. Prior phase plans for patterns: docs/superpowers/plans/2026-07-29-swiss-grid-phase-5-organizer.md (thin shells, TDD pure helpers, deliberate deviations header, ink surface)

Then use superpowers:writing-plans →
docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md

Branch later from main (tip must include Phase 5 merge). Worktree redesign/swiss-grid-p6.
Do not implement until the plan is reviewed.
Do not reopen locked P1–P5 deviations.
Admin uses inverted ink surface — reuse tokens; shake out any hard-coded light-surface assumptions.
Suggested screen order: A1 → A2 → A3; deploy+verify last.
```
