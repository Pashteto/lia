# Swiss Grid Phase 6 — Admin inverted suite (A1–A3)

**Branch:** `redesign/swiss-grid-p6`  
**Base:** `8dce925` — Phase 5 fidelity tip pin  
**HEAD:** `cd97efe`  
**Date:** 2026-07-29  
**Status:** Ready for merge (deploy **not** run this task — deferred pending human request)

---

## Shipped

| Screen | Route | Summary |
|---|---|---|
| **A1 Обзор** | `/admin` | `AdminOverview` — ink 4-tile strip (Ждут = published queue length, signal fill; Организаторов/Пользователей = `—`); queues + verify panel; Signals footer; max-sm duty mode (2 tiles + short queue) |
| **A2 Модерация** | `/admin/moderation/events` | `AdminModeration` — `250px 1fr` conveyor; Ждут/Все chips; selected paper row; reason gate; approve/reject/revision + auto-advance; `AdminDesktopOnly` &lt;900px |
| **A3 Организаторы** | `/admin/organizers` | `AdminOrganizers` — filter chips + search; table `44px 1fr 110px 90px 90px 170px`; verify/reject absorbed; complaints filter stub EmptyState; `/admin/moderation/organizers` → `?filter=pending` |
| **Users stub** | `/admin/users` | Phase-7 stub EmptyState only (ADMIN_NAV fidelity — not A4) |
| **Secondary** | `/admin/settings`, `/admin/complaints` | Ink-safe restyle (no glass/radius); not in A1–A3 fidelity contract |

**Shared infrastructure (Task 1–2):** pure helpers (`admin-relative`, `admin-id`, `admin-reject-reasons`, `admin-test-heuristic`, `admin-org-status`, `admin-queue`) with Vitest TDD; admin layout `data-surface="ink"` + `AppHeader admin` + `ADMIN_NAV`; `AdminDesktopOnly`; AppHeader max-sm wordmark `ADMIN`; TabBarGate already hides `/admin`.

---

## Deliberate deviations (pre-decided — do not reopen P1–P5)

1. Routes stay `/admin`, `/admin/moderation/events`, `/admin/organizers` — not conceptual `/admin/moderation` alone.
2. No pending_review API — A2 conveyor over `published`/`rejected` + takedown/reinstate; «НА ДОРАБОТКУ» = prefixed takedown.
3. A1 «Ждут модерации» = `listModerationEvents("published").length` (not `events_removed`).
4. A1 «Организаторов» total → `—` (API only has pending).
5. A1 «Пользователей» → `—` (A4 / Phase 7).
6. A1 Сигналы from complaints_open / test heuristic / muted stub — never invent rows.
7. Queue age uses `starts_at` + `formatRelativeCompactRu`.
8. Short IDs via `adminShortId` (`EV-` + 4 hex).
9. Org email / event / complaint counts → `—` (no fields; no N+1).
10. Test heuristic client-side; УДАЛИТЬ → revoke stand-in (no delete API).
11. A3 «С жалобами» → EmptyState stub.
12. `/admin/users` stub only — not A4 registry/hygiene.
13. Settings/complaints ink-safe only, outside A1–A3 contract.
14. A2/A3 desktop-only &lt;900px; A1 has duty mode.
15. Layout loading = ink Skeleton (no «Загрузка…»).
16. Admin mobile wordmark `ADMIN`; captions per screen.
17. No TanStack Query migration for admin fetches.

**Locked from earlier phases:** favorites deferred, home `/`, U4 personal calendar, Yandex not OSM, Golos/Manrope/JB Mono, U3 non-AI `/search`, O1 activity stub, O4 sequential decide, O5 no telegram, etc.

**Out of scope:** **A4** Users + content hygiene (Phase 7 + backend).

---

## Parked / deferred (needs human browser verify before pixel-complete claim)

| Item | Source | Notes |
|---|---|---|
| **Live pixel QA A1–A3** | Task 7 checklist | No admin login this close — structural/code pass only |
| **A1 duty mode @ 390px** | Fidelity contract | Code path present (`max-sm`); not live-verified |
| **A2 reason gate + auto-advance live** | Task 4 | Unit helpers covered; UI flows need admin session |
| **A3 verify/reject against API** | Task 5 | Code present; backend session not exercised here |
| **ADMIN_NAV underline live** | Layout / AppHeader | Code: `border-b-2 border-current`; live verify deferred |
| **Production deploy** | Task 7 Step 3 | **Deferred pending human request** |

---

## Verification

```bash
cd frontend && pnpm build && pnpm test && pnpm lint
```

| Command | Result |
|---|---|
| `pnpm build` | PASS |
| `pnpm test` | PASS — **153** tests (39 files) |
| `pnpm lint` | PASS |

### Browser fidelity (Task 7)

| Check | Result |
|---|---|
| Live admin login + HTML badges A1–A3 @ desktop / 390 | **NOT LIVE-VERIFIED** — no admin session |
| `data-surface="ink"` on `/admin/*` | PASS (layout root) |
| `ADMIN_NAV` active underline; Пользователи → stub | PASS (code: AppHeader + `/admin/users` EmptyState) |
| A1 four tiles; signal tile; queues; duty mode | PASS (code structure; duty `max-sm` not live) |
| A2 `250px 1fr`; selected paper; reason gate; advance; &lt;900px | PASS (code; live deferred) |
| A3 filters + table; verify; moderation/organizers redirect | PASS (code + redirect page) |
| No `rounded-` / `shadow-` on admin surfaces | PASS (rg Admin* + `app/admin`) |
| Tab bar hidden on `/admin` | PASS (`TabBarGate` `/admin` prefix) |

---

## Commits (since `8dce925`)

| SHA | Message |
|---|---|
| `9af68bb` | checkpoint before checking out cursor/swiss-grid-p5-fidelity |
| `28a70fa` | feat(frontend): Swiss Grid admin pure helpers for A1–A3 |
| `6ef41bb` | feat(frontend): Swiss Grid admin ink layout and header |
| `ce8cb67` | feat(frontend): Swiss Grid A1 admin overview |
| `d82cb29` | feat(frontend): Swiss Grid A2 moderation conveyor |
| `02499aa` | feat(frontend): Swiss Grid A3 organizers registry |
| `cd97efe` | feat(frontend): admin users stub and ink-safe secondary pages |

---

## Deploy

**Skipped — deploy deferred pending human request.** Runbook: `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`.

---

## Merge notes

- **PR title (when opened):** `Swiss Grid Phase 6 — Admin inverted suite (A1–A3)` — base `main` (must include Phase 5).
- **No force-push.** Rebase onto latest `main` if drifted.
- **Before claiming pixel-complete:** live browser checklist as admin against HTML badges A1–A3; exercise A2 decisions and A3 verify against running API.
