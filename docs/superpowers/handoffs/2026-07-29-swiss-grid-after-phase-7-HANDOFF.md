# HANDOFF — Swiss Grid after Phase 7

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Branch tip:** `main` @ `c740cf5` (was `redesign/swiss-grid-p7`)  
**Live product:** https://presence.tarski.ru (deploy with P7 cutover)

---

## What shipped

| Phase | Scope | Status |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged |
| **P2** Public core | U1 feed, U2 detail, U7 auth, U8 states | Merged |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged |
| **P4** U3 Подбор | `/search` deterministic smart-filter | Merged |
| **P5** Organizer | O1–O5 Swiss suite | Merged |
| **P6** Admin ink | A1–A3 + ink secondary | Merged (base of P7) |
| **P7** Admin A4 + P7.3 sweep | Users registry + hygiene APIs + Liquid Glass deletion | Merged to main + deployed 2026-07-29 |

**Not shipped (post–Swiss Grid):**

| Item | Notes |
|---|---|
| Named type-utility migration | ~325 intentional `text-[Npx]` remain; optional cleanup |
| A1 users tile | Still `—` |
| A4 search UI | API `q` ready; UI deferred |
| Lighthouse budget enforcement | Smoke after deploy |

---

## Phase 7 notes (for context, do not reopen)

- `/admin/users` is real A4: `1fr 300px`, registry columns `44px 1fr 96px 84px 96px`, hygiene rail, two-step «СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ».
- Backend: `internal/adminusers`, `internal/hygiene`; hide-all reuses `moderation.Takedown`.
- No migration; suspicious price ≥ 100 000 ₽; derived role includes «Админ».
- Locked P1–P6 deviations still locked.

**Plan / report:**  
`docs/superpowers/plans/2026-07-29-swiss-grid-phase-7-admin-a4.md`  
`docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md`

---

## Design authority

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel reference | `…/Presence Swiss Grid - Full System.dc.html` |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` |

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
```

---

## Hard constraints (every phase)

- Zero border radius, zero shadows; 1px hairlines.
- Admin = `data-surface="ink"`.
- Mono numerals; signal red only for needs-attention / destructive.
- A2–A4 desktop-only &lt;900px.
- No TanStack Query migration for admin fetches.
- UI copy Russian; code/commits English.
- Fonts: Golos Text / Manrope / JetBrains Mono.

---

## Code landmarks

| Area | Path |
|---|---|
| A4 UI | `frontend/components/AdminUsers.tsx` |
| A4 page | `frontend/app/admin/users/page.tsx` |
| Helpers | `frontend/lib/admin-user-role.ts`, `admin-registration.ts`, `hygiene-labels.ts`, `admin-id.ts` |
| API client | `frontend/lib/api.ts` — `listAdminUsers`, `listHygieneIssues`, `hideAllHygiene` |
| Registry | `backend/internal/adminusers/` |
| Hygiene | `backend/internal/hygiene/` |
| Routes | `backend/internal/http/admin/handler.go` |

---

## Recommended next work

1. Live-verify `/admin/users` against badge A4 after deploy.
2. Optional: migrate remaining `text-[Npx]` to named utilities; wire A1 users tile; A4 search UI.
3. Product backlog beyond Swiss Grid roadmap.

---

## Minimal prompt for next planning agent

> Phase 7 A4 is on `redesign/swiss-grid-p7` (see `docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md`). Next: finish branch (merge options), then plan master-plan §7.3 final Swiss Grid sweep — delete leftover Liquid Glass / ThemeSwitch / cover gradients / decorative CategoryGlyph; tighten typography; Lighthouse/CLS. Do not reopen P1–P7 deviations. Design authority remains the Swiss Grid handoff README + Full System HTML.
