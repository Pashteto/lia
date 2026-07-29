# Swiss Grid Phase 7 — Admin A4 Пользователи и контент-гигиена

**Branch:** `redesign/swiss-grid-p7`  
**Base:** `c3195b8` — Phase 6 tip (A2 filters fix)  
**HEAD:** `fe8c4dd`  
**Date:** 2026-07-29  
**Status:** Ready for merge (deploy **not** run this task — deferred pending human request)

---

## Shipped

| Screen / API | Route | Summary |
|---|---|---|
| **A4** | `/admin/users` | `AdminUsers` — `1fr 300px` registry + hygiene rail; `AdminDesktopOnly` &lt;900px |
| **Users API** | `GET /api/v1/admin/users` | Aggregate registry over users ⟕ RSVPs ⟕ organizers; `q`/`limit`/`offset` |
| **Hygiene list** | `GET /api/v1/admin/hygiene` | `{issues,count}` over published events (`test_data` / `suspicious_price`) |
| **Hygiene hide** | `POST /api/v1/admin/hygiene/hide-all` | Bulk `moderation.Takedown`; `{hidden,skipped}` |

**Backend packages:** `internal/adminusers`, `internal/hygiene` — wired via `SetAdminUsers` / `SetHygiene`, 503 when unwired.

**Frontend helpers (Vitest):** `adminUserRoleLabel`, `formatRegistrationMonth`, `hygieneKindLabel` / `hygieneIssueValue` / `hygieneIssueSource`, `adminUserShortId`.

---

## Deliberate deviations (pre-decided)

1. Route stays `/admin/users`.
2. No migration — images-only deploy; prod DB level unchanged.
3. Derived role: Тестовый → Админ → Организатор → Зритель (`Админ` added vs handoff three labels).
4. Test heuristic client-side; Go detector mirrors the same regex.
5. Bookings = RSVPs in `going|applied|accepted|waitlist`.
6. Deleted users excluded (`status = active`).
7. Hygiene over `published` only.
8. Suspicious price threshold = `100000` ₽.
9. No search UI (API supports `q`); «Показать ещё» pagination.
10. Hide-all = bulk takedown, never delete.
11. No per-user actions.
12. Desktop-only &lt;900px.
13. No TanStack Query migration.
14. A1 «Пользователей» tile stays `—` (Phase-8 follow-up).

**Pixel departures:** 44px tap targets; real `MM.YYYY` (not mock «тест»); «ПОКАЗАТЬ ЕЩЁ»; two-step confirm on destructive CTA.

---

## Parked / deferred

| Item | Notes |
|---|---|
| Search UI on A4 | API ready; mock has none |
| A1 users tile count | Locked P6 #5 |
| Per-user ban/role/impersonate | Out of scope |
| Duplicates detector | Mock prose only — not claimed in UI |
| Live pixel QA / admin session | Code + unit pass; live verify after deploy |
| Production deploy | Deferred pending human request |

---

## Verification

### Backend

```bash
cd backend && go test ./internal/adminusers/ ./internal/hygiene/ ./internal/http/admin/
```

| Package | Result |
|---|---|
| `internal/adminusers` | PASS |
| `internal/hygiene` | PASS (10 tests) |
| `internal/http/admin` | PASS (incl. 401/403/200/503 for users + hygiene) |

`go build ./...` requires local swagger generate (pre-existing env gap — same as prior phases on clean worktrees).

### Frontend

```bash
cd frontend && pnpm build && pnpm test && pnpm lint
```

| Command | Result |
|---|---|
| `pnpm build` | PASS |
| `pnpm test` | PASS — **170** tests (42 files) |
| `pnpm lint` | PASS |

### Fidelity greps (Task 6)

- No `rounded-` / `shadow-` on A4 files
- No Tailwind palette colors (`text-red-` etc.)
- No «Загрузка…» prose
- Grids `1fr 300px` and `44px 1fr 96px 84px 96px` present

---

## Commits

| SHA | Subject |
|---|---|
| `61c8098` | feat(backend): admin user registry endpoint for A4 |
| `c399614` | feat(backend): content hygiene detection and bulk hide for A4 |
| `b721b8f` | feat(frontend): A4 admin users and hygiene api client and helpers |
| `8523be7` | feat(frontend): Swiss Grid A4 user registry and hygiene rail |
| `fe8c4dd` | docs: Phase 7 A4 plan, report, and handoff |

---

## Deploy

**Not run.** When requested: images-only (no `migrate up`); rebuild backend + frontend with both `NEXT_PUBLIC_*` build-args; follow `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`; prune Docker after.

---

## Merge notes

- Rebase onto latest `main` if drifted; no force-push.
- Phase 6 should already be on `main` at `c3195b8` tip used as base.
