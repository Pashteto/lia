# HANDOFF — Swiss Grid after Phase 7 (complete)

**Date:** 2026-07-29  
**Repo:** `Pashteto/lia`  
**Git:** `main` @ `81514df` — **pushed** to `origin/main`  
**Live:** https://presence.tarski.ru · API https://api.presence.tarski.ru  
**Deploy:** images `swiss-p7-r1`; rollback `*:rollback-swiss-p7-20260729-142227`; LIA DB **020** (no P7 migration)

**Companion prompt (paste into a new agent session):**  
[`2026-07-29-swiss-grid-after-phase-7-PROMPT.md`](./2026-07-29-swiss-grid-after-phase-7-PROMPT.md)

---

## What shipped (Swiss Grid roadmap closed)

| Phase | Scope | Status |
|---|---|---|
| **P1** Foundation | Tokens, fonts, primitives | Merged + live |
| **P2** Public core | U1 / U2 / U7 / U8 | Merged + live |
| **P3** Calendar / map / profile | U4 / U5 / U6 | Merged + live |
| **P4** Подбор | U3 `/search` | Merged + live |
| **P5** Organizer | O1–O5 | Merged + live |
| **P6** Admin ink | A1–A3 | Merged + live |
| **P7** A4 + P7.3 sweep | Users + hygiene APIs + Liquid Glass deletion | Merged + live + pushed |

**Swiss Grid master plan (`2026-07-28-swiss-grid-redesign-master-plan.md`) is done.**  
Further work is product backlog / polish, not a numbered Swiss Grid phase — unless product reopens scope.

---

## Parked / optional follow-ups (do not reopen locked deviations without sign-off)

| Priority | Item | Notes |
|---|---|---|
| **1 — verify** | Live admin QA of A4 | Desktop `/admin/users`: registry columns, hygiene rail, two-step hide-all; &lt;900px → desktop-only notice. Needs admin session (`poulissimo@gmail.com`). |
| **2 — optional** | A1 «Пользователей» tile | Still `—`. Wiring count reopens locked P6 deviation #5 — needs product OK. |
| **3 — optional** | A4 search UI | `GET /admin/users?q=` exists; mock has no search field. |
| **4 — optional** | Named type utilities | ~325 intentional `text-[Npx]` remain (Swiss scale). Not Liquid Glass leftovers. |
| **5 — optional** | Lighthouse / CLS budget | Fonts preload OK; formal score not enforced. |
| **6 — ops** | `YANDEX_PLACES_KEY` | Still often unprovisioned → venue-name search degrades. |
| **7 — ops** | `GATEGUARD_AUTH_SECRET` rotation | Long-standing pending. |

---

## Locked — do not reopen

Favorites/♡ deferred; home `/`; U4 personal calendar; Yandex Maps not OSM; Golos Text / Manrope / JetBrains Mono; U3 non-AI `/search`; O1 activity stub; O4 sequential decide; A1–A4 deviations from Phase 6–7 plans (incl. A1 users tile `—`, no per-user actions, hide-all = takedown not delete, no duplicates detector, desktop-only A2–A4).

---

## Design authority (still)

| Role | Path |
|---|---|
| Spec / behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` |
| Pixel | `…/Presence Swiss Grid - Full System.dc.html` |
| Tokens | `…/tokens.css`, `…/tokens.ts` |
| P7 report | `docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md` |
| Deploy memory | `~/.claude/.../memory/lia-demo-deployment.md` |

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
```

---

## Hard constraints (carry forward)

- Zero border radius, zero shadows; 1px hairlines.
- Admin = `data-surface="ink"`.
- Mono numerals; signal red only for needs-attention / destructive.
- A2–A4 desktop-only &lt;900px.
- No TanStack Query migration for admin fetches.
- UI copy Russian; code/commits English.
- Fonts: Golos Text / Manrope / JetBrains Mono.
- Deploy: build amd64 on Mac → `docker save \| gzip \| ssh \| docker load`; recreate `app` with **all 4** compose files; frontend needs **both** `NEXT_PUBLIC_API_URL` + `NEXT_PUBLIC_YANDEX_MAPS_KEY`.

---

## Code landmarks

| Area | Path |
|---|---|
| A4 UI | `frontend/components/AdminUsers.tsx` |
| A4 page | `frontend/app/admin/users/page.tsx` |
| Helpers | `frontend/lib/admin-user-role.ts`, `admin-registration.ts`, `hygiene-labels.ts`, `admin-id.ts` |
| API client | `listAdminUsers`, `listHygieneIssues`, `hideAllHygiene` in `frontend/lib/api.ts` |
| Registry | `backend/internal/adminusers/` |
| Hygiene | `backend/internal/hygiene/` |
| Admin routes | `backend/internal/http/admin/handler.go` |
| Cover (flat) | `frontend/components/ui/EventCover.tsx`, `frontend/lib/covers.ts` |

---

## Suggested next steps (ordered)

1. **Human:** browser-verify A4 as admin on prod; optional one real «СКРЫТЬ ВСЁ» only with operator consent.
2. **Product choose:** A1 users count? A4 search? typography utility pass? Or leave Swiss Grid frozen and start non-redesign backlog.
3. If product picks work → brainstorm → writing-plans → worktree → implement → deploy (images-only unless a migration is explicitly required).
4. Ops when convenient: provision `YANDEX_PLACES_KEY`; rotate GateGuard auth secret.

---

## Minimal prompt

Full paste-ready text lives in  
[`2026-07-29-swiss-grid-after-phase-7-PROMPT.md`](./2026-07-29-swiss-grid-after-phase-7-PROMPT.md).
