# Swiss Grid Phase 7 — Admin A4 Пользователи и контент-гигиена Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/admin/users` Phase-6 stub with the real **A4** screen — a user registry table plus a content-hygiene rail with a destructive «СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ» — backed by three new Go endpoints (`GET /api/v1/admin/users`, `GET /api/v1/admin/hygiene`, `POST /api/v1/admin/hygiene/hide-all`) built in this same phase.

**Architecture:** Two new backend domain packages follow the existing monolith convention (`internal/moderation`, `internal/complaints`): **`internal/adminusers`** = service + pg repository over the existing `users` / `event_rsvps` / `organizers` tables (one aggregate query, no N+1, **no migration**); **`internal/hygiene`** = a *pure* `Detect()` over published-event candidates supplied by the existing `events.Service`, plus `HideAll()` that reuses `moderation.Service.Takedown` so every hide is transactional and audit-logged. Both are injected into the existing `internal/http/admin` handler (`admin.Deps`) and degrade to 503 when nil (no-DB mode), exactly like `Moderation` / `Complaints`. On the frontend, `app/admin/users/page.tsx` mounts `AdminDesktopOnly` + a new client component `components/AdminUsers.tsx` that renders the `1fr 300px` split; all string/number formatting lives in pure Vitest-covered helpers.

**Tech Stack:** Go 1.x monolith (net/http `ServeMux` admin handler, go-pg v10, gofrs/uuid), Postgres; Next.js 16 App Router / React 19 / TypeScript, Tailwind v4 (Swiss Grid tokens + `data-surface="ink"`), Vitest 4 (node-only pure helpers). Admin pages keep the `useEffect` + `tick` fetch pattern — **no TanStack Query migration** (master-plan risk #10).

**Spec / authority:**

| Role | Source |
|---|---|
| Behaviour | `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` § *A4 · Пользователи и контент-гигиена* + § *Part 3 — Admin (inverted)* + § *Data & state per screen* row `A4` |
| Pixel | `docs/Redesign/5/design_handoff_presence_swiss_grid/Presence Swiss Grid - Full System.dc.html` — badge **«A4 · Пользователи и контент-гигиена»** (~line 985) |
| Roadmap | `docs/superpowers/plans/2026-07-28-swiss-grid-redesign-master-plan.md` § Phase 7 (7.1–7.2), § Global Constraints, § Known backend gaps |
| Handoff | `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-6-HANDOFF.md` |
| Locked P6 deviations | `docs/superpowers/plans/2026-07-29-swiss-grid-phase-6-admin.md`, `docs/superpowers/reports/2026-07-29-swiss-grid-phase-6.md` |

Open the pixel reference with:

```bash
cd docs/Redesign/5/design_handoff_presence_swiss_grid && python3 -m http.server 8099
# then http://localhost:8099/Presence%20Swiss%20Grid%20-%20Full%20System.dc.html and scroll to badge A4
```

**Prerequisite:** `main` tip must include the Phase 6 merge. Verify before starting:

```bash
git log --oneline -5      # must include c3195b8 (A2 filters fix) / cd97efe (admin users stub)
test -f frontend/app/admin/users/page.tsx           # the stub being replaced
test -f frontend/components/AdminOrganizers.tsx     # A3 table pattern to mirror
test -f frontend/components/AdminDesktopOnly.tsx
test -f frontend/lib/admin-test-heuristic.ts
test -f frontend/lib/admin-id.ts
grep -q 'padCount' frontend/lib/org-seats.ts
grep -q '"/admin/users"' frontend/components/ui/AppHeader.tsx
grep -q 'GET /api/v1/admin/organizers' backend/internal/http/admin/handler.go
```

**Branch / worktree (at execution time only):** `redesign/swiss-grid-p7` via `superpowers:using-git-worktrees`, branched from `main` at the Phase-6 tip. Working directory for `pnpm` commands: `frontend/`; for `go` / `make` commands: `backend/`. **Do not implement until this plan is reviewed.**

**Screen / task order:** backend registry → backend hygiene → frontend helpers → A4 left registry → A4 right hygiene rail → states & page wiring → verify + docs + **deploy last**.

---

## Global Constraints

Everything in the master plan's *Global Constraints* applies. The ones this phase trips over:

- **Admin = `data-surface="ink"`** (already on `app/admin/layout.tsx`): bg `#111`, text `#F2F0EC`, structural rules `#F2F0EC` (`border-paper`), inner rules `#3A3733` (`border-rule-inner`), table head `#1C1A18` (`bg-surface-head`), secondary `#8A857C` (`text-muted-2`), body `#CFCABF` (`text-text-dim`).
- **Zero border radius, zero shadows.** 1px hairlines only (2px only for nav active underline / focus outline).
- **All numbers in JetBrains Mono** (`font-mono`): user IDs, registration month, booking counts, hygiene counts, prices. Unknown → `—`.
- **`signal` red (`#E2231A`) only for needs-attention/destructive:** test-account names, «Тестовый» role chip, the «Гигиена контента · N» caption, the «СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ» footer button. Nowhere else.
- Uppercase tracking preserved: `.cap` 0.13em, chips 0.12em, buttons 0.07em.
- Hover on admin fills `#1C1A18` via `hover-invert` under ink; focus `swiss-focus` (paper outline); transitions 120ms linear on background/colour only.
- Loading = `Skeleton` cells at final dimensions — **never spinners, never «Загрузка…» prose**.
- Empty / gated / error surfaces use `EmptyState` (U8 pattern: mono numeral, one sentence, 1–2 actions).
- **Tap targets ≥44px** on interactive rows/buttons (mock's 8–9px padding grows; type sizes stay).
- Content max 1360px; Moscow-pinned formatters (`Europe/Moscow`); React #418 fix preserved.
- UI copy Russian; code, comments, and commit messages English.
- Fonts stay **Golos Text / Manrope / JetBrains Mono** (locked P1).
- **A4 is desktop-only** below 900px (`AdminDesktopOnly`, same as A2/A3 — locked P6 deviation 14). No A4 mobile composition exists in the handoff.
- Frontend: every commit runs `pnpm build && pnpm test && pnpm lint` from `frontend/`.
- Backend: every commit runs `go build ./... && go test ./...` from `backend/` (plus `golangci-lint run ./...` if the binary is installed).

---

## Design fidelity contract — A4

**Pixel authority:** `Presence Swiss Grid - Full System.dc.html`, badge **«A4 · Пользователи и контент-гигиена»**. **Behaviour authority:** README § A4 + the Phase-6 handoff. Both must hold at desktop; below 900px the desktop-only notice takes over.

| # | Requirement (from the badge A4 markup) | Implementation |
|---|---|---|
| F1 | Page body is a **`1fr 300px`** grid filling the height under the admin header | `grid grid-cols-[1fr_300px] min-h-[…]` on the A4 root |
| F2 | Left column has a **1px paper right rule** (`border-right:1px solid #f2f0ec`) | `border-r border-paper` |
| F3 | Registry table columns exactly **`44px 1fr 96px 84px 96px`** | `grid-cols-[44px_1fr_96px_84px_96px]` shared const `GRID` |
| F4 | Header row: `bg #1C1A18`, bottom rule paper, `.cap` `#8A857C` labels **ID · Пользователь · Регистрация · Записей · Роль**; ID centred, «Пользователь» padded `12px` | `bg-surface-head border-b border-paper`, `.cap text-muted-2` |
| F5 | Row rules are **inner** `#3A3733` | `border-b border-rule-inner` |
| F6 | ID cell: mono 9.5px, `#8A857C`, centred | `font-mono text-[9.5px] text-muted-2 text-center` |
| F7 | User cell: name 12px/700 (red for test accounts) over `.cap` `#8A857C` email | `text-[12px] font-bold` + `cap text-muted-2`; `text-signal` when test |
| F8 | Регистрация: mono 10px `#CFCABF`, format `MM.YYYY` | `font-mono text-[10px] text-text-dim` |
| F9 | Записей: mono 11px/700, zero-padded to 2 | `font-mono text-[11px] font-bold` + `padCount` |
| F10 | Роль: chip 7.5px, `padding 2px 6px`, muted border+text (`#8A857C`); red border+text for «Тестовый» | `Chip` `dark-muted` / `signal`, `text-[7.5px] px-[6px] py-[2px]` |
| F11 | Right rail header: `.cap` **«Гигиена контента · N»** in **`#E2231A`, 700**, bottom rule paper, padding `11px 16px` | `cap font-bold text-signal border-b border-paper px-[16px] py-[11px]` |
| F12 | One block per issue: `.cap` `#8A857C` type caption → value 11px/700 line-height 1.2 → `.cap` `#8A857C` source caption; bottom rule inner; padding `10px 16px` | see `HygieneBlock` in Task 5 |
| F13 | Footer CTA pinned to the bottom (`margin-top:auto`), full width, **`#E2231A` fill, white text**, 10px/700, tracking 0.07em, label **«СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ»** | `mt-auto` + `Button variant="destructive" className="w-full min-h-[44px] text-[10px]"` |
| F14 | Admin header stays `PRESENCE / ADMIN` with **Пользователи** active-underlined | already `ADMIN_NAV` + `AppHeader admin` (P6) |
| F15 | Ink surface continuity with A1–A3: zero radius, zero shadows, 1px hairlines, mono numbers, red only for destructive/needs-attention | enforced by tokens + review grep in Task 7 |

**Deliberate departures from the mock's literal pixels** (justified, listed here so review can reject them individually):

- Row vertical padding grows so interactive rows/buttons clear **44px** (Global Constraint; mock is 8–9px).
- Mock's test user shows `reg: "тест"` in the Регистрация column — we always render the real `MM.YYYY` (or `—`). Fabricated column values are not acceptable in-product.
- Mock has no pagination and no search. We add a **«ПОКАЗАТЬ ЕЩЁ»** footer row under the table when a full page came back (registry is unbounded in production). Search UI is **not** built in this phase (the API supports `q`; see deviation 9).
- The destructive CTA gets a **two-step inline confirm** (Подтвердить / Отмена) before firing — a one-click bulk unpublish is not shippable.

---

## Deliberate deviations for Phase 7 (pre-decided — do not "fix" without product sign-off)

1. **Route stays `/admin/users`** (handoff route matches; nothing to remap).
2. **No migration.** Every field A4 needs already exists: `users(uuid, email, name, role, created_at, status)` (migrations 000002 + 000014), `event_rsvps(user_id, status)` (000013, indexed `event_rsvps_user_idx`), `organizers(owner_user_id)` (000015). Prod stays at DB **019/021** — the deploy is images-only, no `migrate up`.
3. **«Роль» is derived, not stored.** Priority: test heuristic → `Тестовый`; `role == "admin"` → `Админ`; has an `organizers` row → `Организатор`; else → `Зритель`. `Админ` is an addition to the handoff's three labels — staff accounts must not read as «Зритель».
4. **Test-account detection stays client-side** (locked P6 deviation 10): `isLikelyTestContent(name, email)` from `lib/admin-test-heuristic.ts`. The backend hygiene detector uses its **own, deliberately identical** regex on the Go side (`internal/hygiene`), because SQL/Go cannot import the TS helper. Both patterns are commented as mirrors of each other.
5. **Booking count = RSVPs in `going | applied | accepted | waitlist`.** `declined | withdrawn | cancelled` do not count. One `LEFT JOIN` aggregate — never per-user queries.
6. **Deleted users excluded** (`users.status = 'active'`). The registry is an operations tool, not an audit log.
7. **Hygiene detects over `status = 'published'` events only** — the panel's stated purpose is "what leaks into the public feed". Drafts and already-rejected events are out.
8. **Suspicious price threshold = `100000 ₽`** (`hygiene.SuspiciousPriceRUB`), applied to `price_min`/`price_max` on non-free events. One event yields **at most one** issue; `test_data` wins over `suspicious_price`.
9. **No search field on A4.** The mock has none; the endpoint accepts `q` for a follow-up. Registry is paginated (`limit`/`offset`, page size 100) with a «ПОКАЗАТЬ ЕЩЁ» footer.
10. **«СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ» = bulk takedown**, not delete. There is no delete-event API (locked P6 deviation 10) and destroying user content from an admin button is not on the table. Each hide is `moderation.Takedown` → `published → rejected` + `event_status_history` + `audit_log` row, reason `Гигиена контента: тестовые данные` / `Гигиена контента: подозрительная цена`. Events that already moved status are counted as `skipped`, not failed.
11. **No per-user actions** (no ban / role change / impersonate). The mock's user rows have no action column; inventing writes here is out of scope.
12. **A4 is desktop-only below 900px** via `AdminDesktopOnly` (locked P6 deviation 14).
13. **No TanStack Query migration** for admin fetches (master-plan risk #10).
14. **A1 «Пользователей» tile stays `—`.** Wiring the new count into A1 would reopen a locked P6 deviation (#5); it is listed as a Phase-8 follow-up in the report instead.

**Locked from earlier phases (do not reopen):** favorites/♡ deferred; home `/`; U4 personal calendar; Yandex Maps not OSM; Golos Text / Manrope / JetBrains Mono; U3 non-AI `/search`; O1 activity stub; O4 sequential decide; A1–A3 deviations 1–17 from the Phase-6 plan.

---

## File structure

**Backend (`backend/`)**

| Path | Responsibility |
|---|---|
| `internal/adminusers/service.go` (new) | `Row`, `Filter`, `Repository`, `Service`; input normalisation (limit clamp, trim, offset floor) |
| `internal/adminusers/service_test.go` (new) | Pure unit tests over a recording stub repository |
| `internal/adminusers/repository.go` (new) | One go-pg aggregate query: users ⟕ rsvp counts ⟕ organizers |
| `internal/adminusers/repository_test.go` (new) | `//go:build integration` — `TEST_DATABASE_URL`, mirrors `internal/moderation/repository_test.go` |
| `internal/hygiene/detect.go` (new) | Pure `Detect([]Candidate) []Issue` + `Kind`, `SuspiciousPriceRUB`, test regex |
| `internal/hygiene/detect_test.go` (new) | Pure unit tests (TDD driver for the whole package) |
| `internal/hygiene/service.go` (new) | `Service.List` (events → candidates → Detect), `Service.HideAll` (Takedown loop) |
| `internal/hygiene/service_test.go` (new) | Unit tests with fake events + fake moderator |
| `internal/http/admin/handler.go` (modify) | 3 routes + JSON DTOs + `Deps.Users` / `Deps.Hygiene` |
| `internal/http/admin/handler_test.go` (modify) | 401/403/200 + shape tests for the 3 routes |
| `internal/http/module.go` (modify) | `SetAdminUsers` / `SetHygiene` setters → `admin.Deps` |
| `internal/application.go` (modify) | Wire both services inside the existing `if repoModule != nil` block |

**Frontend (`frontend/`)**

| Path | Responsibility |
|---|---|
| `lib/api.ts` (modify) | `AdminUser`, `HygieneIssue`, `listAdminUsers`, `listHygieneIssues`, `hideAllHygiene` |
| `lib/admin-user-role.ts` (new) | `adminUserRoleLabel` — derived role label |
| `lib/admin-registration.ts` (new) | `formatRegistrationMonth` — Moscow `MM.YYYY` / `—` |
| `lib/hygiene-labels.ts` (new) | `hygieneKindLabel`, `hygieneIssueValue`, `hygieneIssueSource` |
| `lib/admin-id.ts` (modify) | `adminUserShortId` — 4 hex, no `EV-` prefix |
| `lib/__tests__/admin-user-role.test.ts`, `admin-registration.test.ts`, `hygiene-labels.test.ts` (new); `admin-id.test.ts` (modify) | Vitest node-only |
| `components/AdminUsers.tsx` (new) | A4 screen: `1fr 300px`, registry table, hygiene rail, states |
| `app/admin/users/page.tsx` (rewrite) | Thin shell: `AdminDesktopOnly` + `AdminUsers` (deletes the P6 stub) |

**Docs**

| Path | Responsibility |
|---|---|
| `docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md` (new) | Phase report (shipped / deviations / verification / commits) |
| `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md` (new) | Next-session handoff |

---

## Task 1: Backend — user registry (`internal/adminusers`) + `GET /api/v1/admin/users`

**Files:**
- Create: `backend/internal/adminusers/service.go`, `backend/internal/adminusers/service_test.go`, `backend/internal/adminusers/repository.go`, `backend/internal/adminusers/repository_test.go`
- Modify: `backend/internal/http/admin/handler.go`, `backend/internal/http/admin/handler_test.go`, `backend/internal/http/module.go`, `backend/internal/application.go`

**Interfaces:**
- Consumes: `*pg.DB` from `repoModule.DB()`; `domain.User` for the staff gate.
- Produces:
  - `adminusers.Row{ID uuid.UUID; Name, Email, Role string; CreatedAt time.Time; Bookings int; IsOrganizer bool}`
  - `adminusers.Filter{Query string; Limit, Offset int}`, `adminusers.DefaultLimit = 100`, `adminusers.MaxLimit = 500`
  - `adminusers.Repository interface { List(ctx context.Context, f Filter) ([]Row, error) }`
  - `adminusers.Service interface { List(ctx context.Context, f Filter) ([]Row, error) }`, `adminusers.NewService(Repository) Service`, `adminusers.NewRepository(*pg.DB) Repository`
  - HTTP: `GET /api/v1/admin/users?q=&limit=&offset=` → `[]{id,name,email,role,created_at,bookings,is_organizer}`

- [ ] **Step 1: Write the failing service test**

Create `backend/internal/adminusers/service_test.go`:

```go
package adminusers

import (
	"context"
	"testing"
)

type recordingRepo struct {
	got  Filter
	rows []Row
	err  error
}

func (r *recordingRepo) List(_ context.Context, f Filter) ([]Row, error) {
	r.got = f
	return r.rows, r.err
}

func TestList_DefaultsLimitAndTrimsQuery(t *testing.T) {
	repo := &recordingRepo{}
	if _, err := NewService(repo).List(context.Background(), Filter{Query: "  анна  "}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.got.Limit != DefaultLimit {
		t.Fatalf("Limit = %d, want %d", repo.got.Limit, DefaultLimit)
	}
	if repo.got.Query != "анна" {
		t.Fatalf("Query = %q, want %q", repo.got.Query, "анна")
	}
}

func TestList_ClampsLimitAndFloorsOffset(t *testing.T) {
	repo := &recordingRepo{}
	if _, err := NewService(repo).List(context.Background(), Filter{Limit: 5000, Offset: -3}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.got.Limit != MaxLimit {
		t.Fatalf("Limit = %d, want %d", repo.got.Limit, MaxLimit)
	}
	if repo.got.Offset != 0 {
		t.Fatalf("Offset = %d, want 0", repo.got.Offset)
	}
}

func TestList_PassesThroughRows(t *testing.T) {
	repo := &recordingRepo{rows: []Row{{Name: "Анна", Bookings: 14}}}
	rows, err := NewService(repo).List(context.Background(), Filter{Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 || rows[0].Bookings != 14 {
		t.Fatalf("rows = %+v, want one row with 14 bookings", rows)
	}
	if repo.got.Limit != 10 || repo.got.Offset != 20 {
		t.Fatalf("filter = %+v, want limit 10 offset 20", repo.got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/adminusers/`
Expected: FAIL — `no required module provides package .../internal/adminusers` / undefined `NewService`.

- [ ] **Step 3: Write the service**

Create `backend/internal/adminusers/service.go`:

```go
// Package adminusers implements the staff-only user registry behind the A4
// admin screen (Пользователи и контент-гигиена). It reads the local users
// table joined with RSVP counts and organizer ownership; it performs no
// writes — A4 has no per-user actions by design.
package adminusers

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// Page size bounds for the registry listing.
const (
	DefaultLimit = 100
	MaxLimit     = 500
)

// Row is one registry line: the A4 table needs exactly these fields.
type Row struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Role        string // users.role: "common" | "admin"
	CreatedAt   time.Time
	Bookings    int  // RSVPs in going|applied|accepted|waitlist
	IsOrganizer bool // has an organizers row as owner
}

// Filter is the listing query. Query matches name or email (case-insensitive,
// substring); Limit/Offset paginate over created_at DESC.
type Filter struct {
	Query  string
	Limit  int
	Offset int
}

// Repository reads registry rows.
type Repository interface {
	List(ctx context.Context, f Filter) ([]Row, error)
}

// Service is the registry use-case layer.
type Service interface {
	List(ctx context.Context, f Filter) ([]Row, error)
}

type service struct{ repo Repository }

// NewService returns a registry Service backed by repo.
func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) List(ctx context.Context, f Filter) ([]Row, error) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(ctx, f)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd backend && go test ./internal/adminusers/`
Expected: PASS (3 tests).

- [ ] **Step 5: Write the repository**

Create `backend/internal/adminusers/repository.go`:

```go
package adminusers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed registry Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

// row mirrors the aliased columns of the listing query. go-pg maps snake_case
// columns onto these fields by name.
// users.created_at is NOT NULL (migration 000002), so a plain time.Time is safe.
type row struct {
	ID          uuid.UUID `pg:"id"`
	Name        string    `pg:"name"`
	Email       string    `pg:"email"`
	Role        string    `pg:"role"`
	CreatedAt   time.Time `pg:"created_at"`
	Bookings    int       `pg:"bookings"`
	IsOrganizer bool      `pg:"is_organizer"`
}

// listSQL joins the three sources A4 needs in a single statement:
//   - event_rsvps aggregated per user (only statuses that mean "записан")
//   - organizers ownership (role derivation on the client)
//
// Deleted users are excluded: the registry is an operations tool.
const listSQL = `
SELECT u.uuid                        AS id,
       coalesce(u.name, '')          AS name,
       coalesce(u.email, '')         AS email,
       coalesce(u.role, 'common')    AS role,
       u.created_at                  AS created_at,
       coalesce(r.n, 0)              AS bookings,
       (o.id IS NOT NULL)            AS is_organizer
  FROM users u
  LEFT JOIN (
        SELECT user_id, count(*) AS n
          FROM event_rsvps
         WHERE status IN ('going', 'applied', 'accepted', 'waitlist')
         GROUP BY user_id
  ) r ON r.user_id = u.uuid
  LEFT JOIN organizers o ON o.owner_user_id = u.uuid
 WHERE u.status = 'active'
   AND (?0 = '' OR u.name ILIKE ?1 OR u.email ILIKE ?1)
 ORDER BY u.created_at DESC, u.uuid DESC
 LIMIT ?2 OFFSET ?3`

func (r *pgRepository) List(ctx context.Context, f Filter) ([]Row, error) {
	var rows []row
	like := "%" + f.Query + "%"
	if _, err := r.db.QueryContext(ctx, &rows, listSQL, f.Query, like, f.Limit, f.Offset); err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	out := make([]Row, 0, len(rows))
	for _, x := range rows {
		out = append(out, Row{
			ID: x.ID, Name: x.Name, Email: x.Email, Role: x.Role,
			CreatedAt: x.CreatedAt, Bookings: x.Bookings, IsOrganizer: x.IsOrganizer,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Write the integration test for the repository**

Create `backend/internal/adminusers/repository_test.go` (skips without a DB, exactly like `internal/moderation/repository_test.go`):

```go
//go:build integration

package adminusers

import (
	"context"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// Run with a migrated Postgres:
//
//	TEST_DATABASE_URL=postgres://lia:lia@localhost:5432/lia_test?sslmode=disable \
//	  go test -tags=integration ./internal/adminusers/ -v
func openTestDB(t *testing.T) *pg.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	opts, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	db := pg.Connect(opts)
	if _, err := db.Exec("SELECT 1"); err != nil {
		db.Close()
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestList_CountsOnlyActiveRsvpStatuses(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	uid := uuid.Must(uuid.NewV4())
	eid1 := uuid.Must(uuid.NewV4())
	eid2 := uuid.Must(uuid.NewV4())
	if _, err := db.Exec(
		`INSERT INTO users (uuid, email, name, status, role) VALUES (?, ?, 'Registry Probe', 'active', 'common')`,
		uid, "registry-probe-"+uid.String()+"@example.test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE uuid = ?`, uid) })

	if _, err := db.Exec(
		`INSERT INTO event_rsvps (event_id, user_id, status) VALUES (?, ?, 'going'), (?, ?, 'cancelled')`,
		eid1, uid, eid2, uid); err != nil {
		t.Fatalf("insert rsvps: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM event_rsvps WHERE user_id = ?`, uid) })

	rows, err := repo.List(ctx, Filter{Query: "Registry Probe", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Bookings != 1 {
		t.Fatalf("Bookings = %d, want 1 (cancelled must not count)", rows[0].Bookings)
	}
	if rows[0].IsOrganizer {
		t.Fatalf("IsOrganizer = true, want false")
	}
}
```

- [ ] **Step 7: Run both test sets**

Run: `cd backend && go test ./internal/adminusers/ && go test -tags=integration ./internal/adminusers/`
Expected: unit tests PASS; integration test either PASS (with `TEST_DATABASE_URL`) or SKIP.

- [ ] **Step 8: Add the HTTP route (failing handler test first)**

Append to `backend/internal/http/admin/handler_test.go`:

```go
type stubUsers struct {
	got  adminusers.Filter
	rows []adminusers.Row
}

func (s *stubUsers) List(_ context.Context, f adminusers.Filter) ([]adminusers.Row, error) {
	s.got = f
	return s.rows, nil
}

func TestAdminUsers_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Users: &stubUsers{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAdminUsers_200Shape(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	users := &stubUsers{rows: []adminusers.Row{{
		ID: id, Name: "Анна", Email: "anna@example.com", Role: "common",
		CreatedAt: time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC), Bookings: 14, IsOrganizer: false,
	}}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?q=%20анна%20&limit=7&offset=3", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Users: users}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0]["id"] != id.String() || got[0]["bookings"].(float64) != 14 {
		t.Fatalf("body = %v", got)
	}
	if got[0]["created_at"] != "2026-03-04T10:00:00Z" {
		t.Fatalf("created_at = %v, want RFC3339 UTC", got[0]["created_at"])
	}
	if users.got.Limit != 7 || users.got.Offset != 3 || users.got.Query != " анна " {
		t.Fatalf("filter = %+v (query is trimmed by the service, not the handler)", users.got)
	}
}

func TestAdminUsers_503WhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin")}).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
```

Add the imports this file now needs at the top of `handler_test.go`: `"time"` and `adminusers "github.com/Pashteto/lia/internal/adminusers"`.

- [ ] **Step 9: Run the handler test to verify it fails**

Run: `cd backend && go test ./internal/http/admin/`
Expected: FAIL — `unknown field Users in struct literal`.

- [ ] **Step 10: Implement the route**

In `backend/internal/http/admin/handler.go`:

Add the import `adminusers "github.com/Pashteto/lia/internal/adminusers"`, add the dep field:

```go
	Users        adminusers.Service
```

Register the route in `NewHandler` (next to the organizers routes):

```go
	h.mux.HandleFunc("GET /api/v1/admin/users", h.staff(h.listUsers))
```

Add the DTO + handler at the end of the file:

```go
type adminUserJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	Bookings    int    `json:"bookings"`
	IsOrganizer bool   `json:"is_organizer"`
}

// listUsers serves the A4 registry. Pagination is bounded by the service
// (DefaultLimit / MaxLimit); q matches name or email.
func (h *handler) listUsers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Users == nil {
		writeErr(w, http.StatusServiceUnavailable, "users service not available")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	rows, err := h.deps.Users.List(r.Context(), adminusers.Filter{
		Query: q.Get("q"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "users list failed")
		return
	}
	out := make([]adminUserJSON, 0, len(rows))
	for _, u := range rows {
		out = append(out, adminUserJSON{
			ID: u.ID.String(), Name: u.Name, Email: u.Email, Role: u.Role,
			CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			Bookings:  u.Bookings, IsOrganizer: u.IsOrganizer,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
```

Add `"strconv"` to the import block.

- [ ] **Step 11: Run the handler test to verify it passes**

Run: `cd backend && go test ./internal/http/admin/`
Expected: PASS (existing tests + 3 new).

- [ ] **Step 12: Wire the service into the app**

In `backend/internal/http/module.go`:

```go
	adminUsers         adminusersdomain.Service
```

(field on `Module`, next to `organizers`), the import `adminusersdomain "github.com/Pashteto/lia/internal/adminusers"`, the setter:

```go
// SetAdminUsers injects the staff-only user registry service (A4). Call before Init.
func (m *Module) SetAdminUsers(svc adminusersdomain.Service) { m.adminUsers = svc }
```

and the dep in the `admin.NewHandler(admin.Deps{…})` literal:

```go
		Users:        m.adminUsers,
```

In `backend/internal/application.go`, inside the existing `if repoModule != nil {` block (right after the `SetComplaints` call), add:

```go
			httpModule.SetAdminUsers(
				adminusers.NewService(adminusers.NewRepository(repoModule.DB())),
			)
```

with the import `"github.com/Pashteto/lia/internal/adminusers"`.

- [ ] **Step 13: Build and test the whole backend**

Run: `cd backend && go build ./... && go test ./...`
Expected: PASS. If `golangci-lint` is installed, also run `golangci-lint run ./internal/adminusers/ ./internal/http/admin/` and expect no findings.

- [ ] **Step 14: Commit**

```bash
git add backend/internal/adminusers backend/internal/http/admin backend/internal/http/module.go backend/internal/application.go
git commit -m "feat(backend): admin user registry endpoint for A4"
```

---

## Task 2: Backend — content hygiene (`internal/hygiene`) + `GET /api/v1/admin/hygiene` + `POST /api/v1/admin/hygiene/hide-all`

**Files:**
- Create: `backend/internal/hygiene/detect.go`, `backend/internal/hygiene/detect_test.go`, `backend/internal/hygiene/service.go`, `backend/internal/hygiene/service_test.go`
- Modify: `backend/internal/http/admin/handler.go`, `backend/internal/http/admin/handler_test.go`, `backend/internal/http/module.go`, `backend/internal/application.go`

**Interfaces:**
- Consumes: `eventsdomain.Service.List(ctx, "published", nil, nil, nil) ([]*models.Event, error)` (already loads `Organizer`, `PriceType`, `PriceMin`, `PriceMax`); `moderation.Service.Takedown(ctx, eventID, actorID uuid.UUID, reason string) error` and `moderation.ErrInvalidTransition`.
- Produces:
  - `hygiene.Kind` (`KindTestData = "test_data"`, `KindSuspiciousPrice = "suspicious_price"`), `hygiene.SuspiciousPriceRUB int64 = 100000`
  - `hygiene.Candidate{EventID uuid.UUID; Title, OrganizerName, PriceType string; PriceMin, PriceMax *int64}`
  - `hygiene.Issue{Kind Kind; EventID uuid.UUID; Title, OrganizerName string; PriceRUB *int64}`
  - `hygiene.Detect([]Candidate) []Issue` (pure), `hygiene.ReasonFor(Kind) string`
  - `hygiene.Result{Hidden, Skipped int}`
  - `hygiene.Service interface { List(ctx) ([]Issue, error); HideAll(ctx, actorID uuid.UUID) (Result, error) }`, `hygiene.NewService(Events, Moderator) Service`
  - HTTP: `GET /api/v1/admin/hygiene` → `{"issues":[…],"count":N}`; `POST /api/v1/admin/hygiene/hide-all` → `{"hidden":N,"skipped":M}`

- [ ] **Step 1: Write the failing detector test**

Create `backend/internal/hygiene/detect_test.go`:

```go
package hygiene

import (
	"testing"

	"github.com/gofrs/uuid"
)

func ptr(v int64) *int64 { return &v }

func TestDetect_FlagsTestTitlesAndOrganizers(t *testing.T) {
	issues := Detect([]Candidate{
		{EventID: uuid.Must(uuid.NewV4()), Title: "QA Тур в Геленджик (Блок 8)", OrganizerName: "QA Block8"},
		{EventID: uuid.Must(uuid.NewV4()), Title: "bla bla meet", OrganizerName: "kornkorn10"},
		{EventID: uuid.Must(uuid.NewV4()), Title: "Летний фестиваль медиаискусства", OrganizerName: "Музей «Гараж»"},
	})
	if len(issues) != 2 {
		t.Fatalf("issues = %d, want 2", len(issues))
	}
	if issues[0].Kind != KindTestData || issues[0].OrganizerName != "QA Block8" {
		t.Fatalf("issue[0] = %+v", issues[0])
	}
}

func TestDetect_FlagsOrganizerNameEvenWhenTitleIsClean(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "Лекция о городе", OrganizerName: "Тестовый организатор"},
	})
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v, want one test_data", issues)
	}
}

func TestDetect_FlagsSuspiciousPrice(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "Лабораторная сцена", OrganizerName: "Электротеатр", PriceType: "paid", PriceMin: ptr(100500)},
		{Title: "Концерт", OrganizerName: "Винзавод", PriceType: "paid", PriceMin: ptr(2000)},
	})
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	if issues[0].Kind != KindSuspiciousPrice || issues[0].PriceRUB == nil || *issues[0].PriceRUB != 100500 {
		t.Fatalf("issue = %+v", issues[0])
	}
}

func TestDetect_IgnoresFreeEventsAndNilPrices(t *testing.T) {
	if got := Detect([]Candidate{
		{Title: "Медиация", OrganizerName: "ГМИИ", PriceType: "free", PriceMin: ptr(999999)},
		{Title: "Читательская группа", OrganizerName: "Дом культуры", PriceType: "paid"},
	}); len(got) != 0 {
		t.Fatalf("issues = %+v, want none", got)
	}
}

func TestDetect_TestDataWinsOverPrice(t *testing.T) {
	issues := Detect([]Candidate{
		{Title: "QA прайс", OrganizerName: "QA Block8", PriceType: "paid", PriceMin: ptr(500000)},
	})
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v, want a single test_data issue", issues)
	}
}

func TestReasonFor_IsRussianAndPrefixed(t *testing.T) {
	if got := ReasonFor(KindTestData); got != "Гигиена контента: тестовые данные" {
		t.Fatalf("ReasonFor(test_data) = %q", got)
	}
	if got := ReasonFor(KindSuspiciousPrice); got != "Гигиена контента: подозрительная цена" {
		t.Fatalf("ReasonFor(suspicious_price) = %q", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/hygiene/`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Write the detector**

Create `backend/internal/hygiene/detect.go`:

```go
// Package hygiene detects content that should not be in the public feed —
// test/QA leftovers and nonsense prices — and hides it in bulk through the
// moderation domain. It backs the A4 «Гигиена контента» rail.
package hygiene

import (
	"regexp"

	"github.com/gofrs/uuid"
)

// Kind enumerates the detected issue types rendered by the A4 rail.
type Kind string

const (
	KindTestData        Kind = "test_data"
	KindSuspiciousPrice Kind = "suspicious_price"
)

// SuspiciousPriceRUB is the whole-ruble threshold above which a ticket price
// reads as data-entry nonsense rather than a real price (the handoff's
// «от 100 500 ₽» case). Deliberate constant, not configuration.
const SuspiciousPriceRUB int64 = 100000

// testPattern deliberately mirrors frontend/lib/admin-test-heuristic.ts
// (`/QA|тест|test|\.test\b|bla\s*bla/i`). Keep the two in sync: TypeScript
// cannot be imported here, and the frontend colours rows with its own copy.
var testPattern = regexp.MustCompile(`(?i)QA|тест|test|bla\s*bla`)

// Candidate is one published event considered for hygiene detection.
type Candidate struct {
	EventID       uuid.UUID
	Title         string
	OrganizerName string
	PriceType     string // "free" | anything else
	PriceMin      *int64
	PriceMax      *int64
}

// Issue is one detected problem — one rendered block in the A4 rail.
type Issue struct {
	Kind          Kind      `json:"kind"`
	EventID       uuid.UUID `json:"event_id"`
	Title         string    `json:"title"`
	OrganizerName string    `json:"organizer_name,omitempty"`
	PriceRUB      *int64    `json:"price_rub,omitempty"`
}

// Detect is pure: it maps candidates to issues, preserving input order. An
// event yields at most one issue and test data outranks a suspicious price
// (hiding it is the same action either way; the caption should name the
// stronger signal).
func Detect(candidates []Candidate) []Issue {
	issues := make([]Issue, 0, len(candidates))
	for _, c := range candidates {
		switch {
		case testPattern.MatchString(c.Title) || testPattern.MatchString(c.OrganizerName):
			issues = append(issues, Issue{
				Kind: KindTestData, EventID: c.EventID,
				Title: c.Title, OrganizerName: c.OrganizerName,
			})
		case suspiciousPrice(c) != nil:
			issues = append(issues, Issue{
				Kind: KindSuspiciousPrice, EventID: c.EventID,
				Title: c.Title, OrganizerName: c.OrganizerName, PriceRUB: suspiciousPrice(c),
			})
		}
	}
	return issues
}

// suspiciousPrice returns the offending price, or nil when the candidate's
// price is free, unset, or plausible.
func suspiciousPrice(c Candidate) *int64 {
	if c.PriceType == "free" {
		return nil
	}
	for _, p := range []*int64{c.PriceMin, c.PriceMax} {
		if p != nil && *p >= SuspiciousPriceRUB {
			return p
		}
	}
	return nil
}

// ReasonFor is the moderation reason written to event_status_history and
// audit_log when an issue is hidden from the feed.
func ReasonFor(k Kind) string {
	if k == KindSuspiciousPrice {
		return "Гигиена контента: подозрительная цена"
	}
	return "Гигиена контента: тестовые данные"
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd backend && go test ./internal/hygiene/`
Expected: PASS (6 tests).

- [ ] **Step 5: Write the failing service test**

Create `backend/internal/hygiene/service_test.go`:

```go
package hygiene

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

type fakeEvents struct {
	gotStatus string
	events    []*models.Event
	err       error
}

func (f *fakeEvents) List(_ context.Context, status string, _, _ *time.Time, _ *uuid.UUID) ([]*models.Event, error) {
	f.gotStatus = status
	return f.events, f.err
}

type fakeModerator struct {
	calls []struct {
		id     uuid.UUID
		reason string
	}
	errs map[uuid.UUID]error
}

func (f *fakeModerator) Takedown(_ context.Context, id, _ uuid.UUID, reason string) error {
	f.calls = append(f.calls, struct {
		id     uuid.UUID
		reason string
	}{id, reason})
	return f.errs[id]
}

func event(title, org string) *models.Event {
	return &models.Event{
		ID: uuid.Must(uuid.NewV4()), Title: title,
		Organizer: &models.Organizer{Name: org}, PriceType: "free",
	}
}

func TestList_OnlyPublishedAndDetects(t *testing.T) {
	ev := &fakeEvents{events: []*models.Event{event("QA Тур", "QA Block8"), event("Лекция", "Гараж")}}
	issues, err := NewService(ev, &fakeModerator{}).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ev.gotStatus != "published" {
		t.Fatalf("status = %q, want published", ev.gotStatus)
	}
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestHideAll_TakesDownEachIssueWithItsReason(t *testing.T) {
	e := event("bla bla meet", "kornkorn10")
	ev := &fakeEvents{events: []*models.Event{e, event("Лекция", "Гараж")}}
	mod := &fakeModerator{}
	res, err := NewService(ev, mod).HideAll(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil {
		t.Fatalf("HideAll: %v", err)
	}
	if res.Hidden != 1 || res.Skipped != 0 {
		t.Fatalf("result = %+v, want hidden 1", res)
	}
	if len(mod.calls) != 1 || mod.calls[0].id != e.ID || mod.calls[0].reason != ReasonFor(KindTestData) {
		t.Fatalf("calls = %+v", mod.calls)
	}
}

func TestHideAll_SkipsAlreadyMovedEvents(t *testing.T) {
	e := event("QA Тур", "QA Block8")
	mod := &fakeModerator{errs: map[uuid.UUID]error{e.ID: moderation.ErrInvalidTransition}}
	res, err := NewService(&fakeEvents{events: []*models.Event{e}}, mod).
		HideAll(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil {
		t.Fatalf("HideAll: %v", err)
	}
	if res.Hidden != 0 || res.Skipped != 1 {
		t.Fatalf("result = %+v, want skipped 1", res)
	}
}

func TestHideAll_ReturnsRealErrors(t *testing.T) {
	e := event("QA Тур", "QA Block8")
	boom := errors.New("boom")
	mod := &fakeModerator{errs: map[uuid.UUID]error{e.ID: boom}}
	if _, err := NewService(&fakeEvents{events: []*models.Event{e}}, mod).
		HideAll(context.Background(), uuid.Must(uuid.NewV4())); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
```

- [ ] **Step 6: Run the test to verify it fails**

Run: `cd backend && go test ./internal/hygiene/`
Expected: FAIL — undefined `NewService`.

- [ ] **Step 7: Write the service**

Create `backend/internal/hygiene/service.go`:

```go
package hygiene

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

// Events is the subset of the events domain service this package needs.
type Events interface {
	List(ctx context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error)
}

// Moderator is the subset of the moderation service used to hide events.
// Reusing it keeps every hide transactional and audit-logged.
type Moderator interface {
	Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error
}

// Result reports a bulk hide. Skipped counts events whose status had already
// moved (someone else got there first) — not a failure.
type Result struct {
	Hidden  int `json:"hidden"`
	Skipped int `json:"skipped"`
}

// Service is the hygiene use-case layer.
type Service interface {
	List(ctx context.Context) ([]Issue, error)
	HideAll(ctx context.Context, actorID uuid.UUID) (Result, error)
}

type service struct {
	events Events
	mod    Moderator
}

// NewService returns a hygiene Service over the published-events feed.
func NewService(events Events, mod Moderator) Service {
	return &service{events: events, mod: mod}
}

func (s *service) List(ctx context.Context) ([]Issue, error) {
	events, err := s.events.List(ctx, "published", nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list published events: %w", err)
	}
	return Detect(candidates(events)), nil
}

func (s *service) HideAll(ctx context.Context, actorID uuid.UUID) (Result, error) {
	issues, err := s.List(ctx)
	if err != nil {
		return Result{}, err
	}
	var res Result
	for _, issue := range issues {
		switch err := s.mod.Takedown(ctx, issue.EventID, actorID, ReasonFor(issue.Kind)); {
		case err == nil:
			res.Hidden++
		case errors.Is(err, moderation.ErrInvalidTransition):
			res.Skipped++
		default:
			return res, fmt.Errorf("hide %s: %w", issue.EventID, err)
		}
	}
	return res, nil
}

// candidates maps loaded events onto the detector's input. Organizer is a
// transient read-model and may be nil when the event has no organizer row.
func candidates(events []*models.Event) []Candidate {
	out := make([]Candidate, 0, len(events))
	for _, e := range events {
		if e == nil {
			continue
		}
		c := Candidate{
			EventID: e.ID, Title: e.Title,
			PriceType: e.PriceType, PriceMin: e.PriceMin, PriceMax: e.PriceMax,
		}
		if e.Organizer != nil {
			c.OrganizerName = e.Organizer.Name
		}
		out = append(out, c)
	}
	return out
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/hygiene/`
Expected: PASS (10 tests).

- [ ] **Step 9: Write the failing handler tests for both routes**

Append to `backend/internal/http/admin/handler_test.go`:

```go
type stubHygiene struct {
	issues  []hygiene.Issue
	result  hygiene.Result
	hideErr error
	actor   uuid.UUID
}

func (s *stubHygiene) List(context.Context) ([]hygiene.Issue, error) { return s.issues, nil }

func (s *stubHygiene) HideAll(_ context.Context, actorID uuid.UUID) (hygiene.Result, error) {
	s.actor = actorID
	return s.result, s.hideErr
}

func TestHygiene_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Hygiene: &stubHygiene{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHygiene_200ShapeIncludesCount(t *testing.T) {
	eid := uuid.Must(uuid.NewV4())
	stub := &stubHygiene{issues: []hygiene.Issue{{
		Kind: hygiene.KindTestData, EventID: eid, Title: "QA Тур", OrganizerName: "QA Block8",
	}}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Hygiene: stub}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Issues []map[string]any `json:"issues"`
		Count  int              `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Count != 1 || len(body.Issues) != 1 || body.Issues[0]["kind"] != "test_data" {
		t.Fatalf("body = %+v", body)
	}
	if body.Issues[0]["event_id"] != eid.String() {
		t.Fatalf("event_id = %v", body.Issues[0]["event_id"])
	}
}

func TestHygieneHideAll_200ReturnsCounts(t *testing.T) {
	stub := &stubHygiene{result: hygiene.Result{Hidden: 3, Skipped: 1}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/hygiene/hide-all", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Hygiene: stub}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["hidden"] != 3 || body["skipped"] != 1 {
		t.Fatalf("body = %v", body)
	}
	if stub.actor == uuid.Nil {
		t.Fatalf("actor id not passed through")
	}
}

func TestHygieneHideAll_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/hygiene/hide-all", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Hygiene: &stubHygiene{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHygiene_503WhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin")}).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
```

Add the import `hygiene "github.com/Pashteto/lia/internal/hygiene"` to the test file.

- [ ] **Step 10: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/http/admin/`
Expected: FAIL — `unknown field Hygiene in struct literal`.

- [ ] **Step 11: Implement both routes**

In `backend/internal/http/admin/handler.go` add the import `hygiene "github.com/Pashteto/lia/internal/hygiene"`, the dep:

```go
	Hygiene      hygiene.Service
```

the routes in `NewHandler`:

```go
	h.mux.HandleFunc("GET /api/v1/admin/hygiene", h.staff(h.listHygiene))
	h.mux.HandleFunc("POST /api/v1/admin/hygiene/hide-all", h.staff(h.hideAllHygiene))
```

and the handlers:

```go
// listHygiene serves the A4 «Гигиена контента» rail: detected test data and
// nonsense prices currently visible in the public feed.
func (h *handler) listHygiene(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Hygiene == nil {
		writeErr(w, http.StatusServiceUnavailable, "hygiene service not available")
		return
	}
	issues, err := h.deps.Hygiene.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hygiene list failed")
		return
	}
	if issues == nil {
		issues = []hygiene.Issue{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "count": len(issues)})
}

// hideAllHygiene takes down every currently detected event (published →
// rejected, audit-logged per event). Events whose status already moved are
// reported as skipped, not failed.
func (h *handler) hideAllHygiene(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Hygiene == nil {
		writeErr(w, http.StatusServiceUnavailable, "hygiene service not available")
		return
	}
	res, err := h.deps.Hygiene.HideAll(r.Context(), u.UUID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hygiene hide-all failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}
```

- [ ] **Step 12: Run the handler tests to verify they pass**

Run: `cd backend && go test ./internal/http/admin/`
Expected: PASS.

- [ ] **Step 13: Wire the service into the app**

In `backend/internal/http/module.go`: field `hygiene hygienedomain.Service` on `Module`, import `hygienedomain "github.com/Pashteto/lia/internal/hygiene"`, setter:

```go
// SetHygiene injects the content-hygiene service (A4 rail). Call before Init.
func (m *Module) SetHygiene(svc hygienedomain.Service) { m.hygiene = svc }
```

and `Hygiene: m.hygiene,` in the `admin.Deps` literal.

In `backend/internal/application.go`, inside the same `if repoModule != nil` block, after the `SetAdminUsers` call (hygiene needs both `modSvc` and the events service, so guard on events like the invitations wiring does):

```go
			if app.eventsSvc != nil {
				httpModule.SetHygiene(hygiene.NewService(app.eventsSvc, modSvc))
			}
```

with the import `"github.com/Pashteto/lia/internal/hygiene"`.

- [ ] **Step 14: Build and test the whole backend**

Run: `cd backend && go build ./... && go test ./...`
Expected: PASS. Run `golangci-lint run ./internal/hygiene/ ./internal/http/admin/` if available; expect no findings.

- [ ] **Step 15: Commit**

```bash
git add backend/internal/hygiene backend/internal/http/admin backend/internal/http/module.go backend/internal/application.go
git commit -m "feat(backend): content hygiene detection and bulk hide for A4"
```

---

## Task 3: Frontend — API client + pure helpers (TDD)

**Files:**
- Create: `frontend/lib/admin-user-role.ts`, `frontend/lib/admin-registration.ts`, `frontend/lib/hygiene-labels.ts`
- Create: `frontend/lib/__tests__/admin-user-role.test.ts`, `frontend/lib/__tests__/admin-registration.test.ts`, `frontend/lib/__tests__/hygiene-labels.test.ts`
- Modify: `frontend/lib/admin-id.ts`, `frontend/lib/__tests__/admin-id.test.ts`, `frontend/lib/api.ts`

**Interfaces:**
- Consumes: `isLikelyTestContent` (`lib/admin-test-heuristic.ts`), `priceLabel` (`lib/price-label.ts`), `authHeaders` + `API_V1` (`lib/api.ts`).
- Produces:
  - `type AdminUser = { id: string; name: string; email: string; role: string; created_at: string; bookings: number; is_organizer: boolean }`
  - `type HygieneKind = "test_data" | "suspicious_price"`
  - `type HygieneIssue = { kind: HygieneKind; event_id: string; title: string; organizer_name?: string; price_rub?: number }`
  - `listAdminUsers(opts?: { q?: string; limit?: number; offset?: number }): Promise<AdminUser[]>`
  - `listHygieneIssues(): Promise<HygieneIssue[]>`
  - `hideAllHygiene(): Promise<{ hidden: number; skipped: number }>`
  - `adminUserRoleLabel(user: { role: string; is_organizer: boolean }, opts?: { test?: boolean }): string`
  - `formatRegistrationMonth(iso: string | null | undefined): string`
  - `hygieneKindLabel(kind: string): string`, `hygieneIssueValue(issue: HygieneIssueLike): string`, `hygieneIssueSource(issue: HygieneIssueLike): string | null`
  - `adminUserShortId(id: string): string`

- [ ] **Step 1: Write the failing helper tests**

Create `frontend/lib/__tests__/admin-user-role.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { adminUserRoleLabel } from "@/lib/admin-user-role";

describe("adminUserRoleLabel", () => {
  it("labels a plain account Зритель", () => {
    expect(adminUserRoleLabel({ role: "common", is_organizer: false })).toBe("Зритель");
  });

  it("labels an organizer owner Организатор", () => {
    expect(adminUserRoleLabel({ role: "common", is_organizer: true })).toBe("Организатор");
  });

  it("labels staff Админ, outranking the organizer flag", () => {
    expect(adminUserRoleLabel({ role: "admin", is_organizer: true })).toBe("Админ");
  });

  it("labels test accounts Тестовый, outranking everything", () => {
    expect(adminUserRoleLabel({ role: "admin", is_organizer: true }, { test: true })).toBe("Тестовый");
  });
});
```

Create `frontend/lib/__tests__/admin-registration.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { formatRegistrationMonth } from "@/lib/admin-registration";

describe("formatRegistrationMonth", () => {
  it("renders MM.YYYY", () => {
    expect(formatRegistrationMonth("2026-03-04T10:00:00Z")).toBe("03.2026");
  });

  it("uses Moscow day boundaries", () => {
    // 2025-12-31T22:00Z is 2026-01-01T01:00 in Moscow.
    expect(formatRegistrationMonth("2025-12-31T22:00:00Z")).toBe("01.2026");
  });

  it("returns an em dash for unknown or zero timestamps", () => {
    expect(formatRegistrationMonth(null)).toBe("—");
    expect(formatRegistrationMonth("")).toBe("—");
    expect(formatRegistrationMonth("not a date")).toBe("—");
    expect(formatRegistrationMonth("0001-01-01T00:00:00Z")).toBe("—");
  });
});
```

Create `frontend/lib/__tests__/hygiene-labels.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  hygieneIssueSource,
  hygieneIssueValue,
  hygieneKindLabel,
} from "@/lib/hygiene-labels";

describe("hygieneKindLabel", () => {
  it("names the two detected kinds", () => {
    expect(hygieneKindLabel("test_data")).toBe("Тестовые данные");
    expect(hygieneKindLabel("suspicious_price")).toBe("Подозрительная цена");
  });

  it("falls back for an unknown kind rather than printing the slug", () => {
    expect(hygieneKindLabel("something_new")).toBe("Требует проверки");
  });
});

describe("hygieneIssueValue", () => {
  it("quotes the event title for test data", () => {
    expect(
      hygieneIssueValue({ kind: "test_data", title: "QA Тур в Геленджик (Блок 8)" }),
    ).toBe("«QA Тур в Геленджик (Блок 8)»");
  });

  it("renders the offending price with the от prefix", () => {
    expect(
      hygieneIssueValue({ kind: "suspicious_price", title: "Лабораторная сцена", price_rub: 100500 }),
    ).toBe("от 100 500 ₽");
  });

  it("falls back to the title when a price issue has no price", () => {
    expect(hygieneIssueValue({ kind: "suspicious_price", title: "Лабораторная сцена" })).toBe(
      "«Лабораторная сцена»",
    );
  });

  it("returns an em dash when there is nothing to show", () => {
    expect(hygieneIssueValue({ kind: "test_data", title: "" })).toBe("—");
  });
});

describe("hygieneIssueSource", () => {
  it("prefixes the organizer name", () => {
    expect(hygieneIssueSource({ kind: "test_data", title: "x", organizer_name: "QA Block8" })).toBe(
      "организатор QA Block8",
    );
  });

  it("returns null when the organizer is unknown", () => {
    expect(hygieneIssueSource({ kind: "test_data", title: "x" })).toBeNull();
    expect(hygieneIssueSource({ kind: "test_data", title: "x", organizer_name: "  " })).toBeNull();
  });
});
```

Append to `frontend/lib/__tests__/admin-id.test.ts`:

```ts
import { adminUserShortId } from "@/lib/admin-id";

describe("adminUserShortId", () => {
  it("returns the first four hex chars, uppercased, without a prefix", () => {
    expect(adminUserShortId("2f1a9c40-1111-2222-3333-444455556666")).toBe("2F1A");
  });

  it("returns an em dash for empty or malformed ids", () => {
    expect(adminUserShortId("")).toBe("—");
    expect(adminUserShortId("zzzz-nope")).toBe("—");
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd frontend && pnpm test`
Expected: FAIL — cannot resolve `@/lib/admin-user-role`, `@/lib/admin-registration`, `@/lib/hygiene-labels`; `adminUserShortId` is not exported.

- [ ] **Step 3: Write the helpers**

Create `frontend/lib/admin-user-role.ts`:

```ts
/** A4 role chip label. Derived, not stored: the backend only knows
 * users.role ("common" | "admin") and whether an organizers row exists.
 * Priority: test account → staff → organizer → viewer. */
export function adminUserRoleLabel(
  user: { role: string; is_organizer: boolean },
  opts: { test?: boolean } = {},
): string {
  if (opts.test) return "Тестовый";
  if (user.role === "admin") return "Админ";
  if (user.is_organizer) return "Организатор";
  return "Зритель";
}
```

Create `frontend/lib/admin-registration.ts`:

```ts
// en-CA yields "YYYY-MM-DD"; pinned to Moscow like every other formatter.
const moscowDayFmt = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone: "Europe/Moscow",
});

/** A4 «Регистрация» column: mono MM.YYYY, or «—» when unknown. Go serializes
 * an unset time.Time as year 0001 — treat anything pre-1971 as unknown
 * (same rule as lib/member-since.ts). */
export function formatRegistrationMonth(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  if (d.getUTCFullYear() <= 1970) return "—";
  const [year, month] = moscowDayFmt.format(d).split("-");
  return `${month}.${year}`;
}
```

Create `frontend/lib/hygiene-labels.ts`:

```ts
import { priceLabel } from "./price-label";

export type HygieneIssueLike = {
  kind: string;
  title: string;
  organizer_name?: string;
  price_rub?: number;
};

const KIND_LABEL: Record<string, string> = {
  test_data: "Тестовые данные",
  suspicious_price: "Подозрительная цена",
};

/** Issue-type caption of an A4 hygiene block. Unknown kinds get a neutral
 * label rather than leaking a backend slug into the UI. */
export function hygieneKindLabel(kind: string): string {
  return KIND_LABEL[kind] ?? "Требует проверки";
}

/** The offending value, 11px/700 in the block: the price for a price issue,
 * the quoted event title otherwise. */
export function hygieneIssueValue(issue: HygieneIssueLike): string {
  if (issue.kind === "suspicious_price" && issue.price_rub) {
    return priceLabel(issue.price_rub, "from");
  }
  const title = issue.title?.trim();
  return title ? `«${title}»` : "—";
}

/** Source caption under the value. Null when the organizer is unknown — the
 * caller drops the line rather than printing a placeholder. */
export function hygieneIssueSource(issue: HygieneIssueLike): string | null {
  const name = issue.organizer_name?.trim();
  return name ? `организатор ${name}` : null;
}
```

Append to `frontend/lib/admin-id.ts`:

```ts
/** Display id for the A4 user registry: first 4 hex of the UUID, uppercased.
 * No `EV-` prefix — the column is 44px and the rows are users, not events. */
export function adminUserShortId(id: string): string {
  if (!id) return "—";
  const hex = id.replace(/-/g, "").match(/^[0-9a-fA-F]{4}/);
  return hex ? hex[0].toUpperCase() : "—";
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd frontend && pnpm test`
Expected: PASS — 153 previous tests + 15 new.

- [ ] **Step 5: Add the API client functions**

Append to `frontend/lib/api.ts` (next to the other admin fetchers, after `setOrganizerAutoVerify`):

```ts
export type AdminUser = {
  id: string;
  name: string;
  email: string;
  role: string;
  created_at: string;
  bookings: number;
  is_organizer: boolean;
};

export type HygieneKind = "test_data" | "suspicious_price";

export type HygieneIssue = {
  kind: HygieneKind;
  event_id: string;
  title: string;
  organizer_name?: string;
  price_rub?: number;
};

/** A4 registry page. Server clamps limit to 500 and sorts created_at DESC. */
export async function listAdminUsers(
  opts: { q?: string; limit?: number; offset?: number } = {},
): Promise<AdminUser[]> {
  const params = new URLSearchParams();
  if (opts.q) params.set("q", opts.q);
  if (opts.limit != null) params.set("limit", String(opts.limit));
  if (opts.offset) params.set("offset", String(opts.offset));
  const qs = params.toString();
  const res = await fetch(`${API_V1}/admin/users${qs ? `?${qs}` : ""}`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`admin users: ${res.status}`);
  return res.json();
}

/** A4 hygiene rail: test data / suspicious prices currently in the feed. */
export async function listHygieneIssues(): Promise<HygieneIssue[]> {
  const res = await fetch(`${API_V1}/admin/hygiene`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`hygiene: ${res.status}`);
  const body: { issues?: HygieneIssue[] } = await res.json();
  return body.issues ?? [];
}

/** Destructive: takes down every detected event (published → rejected). */
export async function hideAllHygiene(): Promise<{ hidden: number; skipped: number }> {
  const res = await fetch(`${API_V1}/admin/hygiene/hide-all`, {
    method: "POST",
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error(`hygiene hide-all: ${res.status}`);
  return res.json();
}
```

- [ ] **Step 6: Verify build + tests + lint**

Run: `cd frontend && pnpm build && pnpm test && pnpm lint`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/lib
git commit -m "feat(frontend): A4 admin users and hygiene api client and helpers"
```

---

## Task 4: Frontend — A4 left column (user registry)

**Files:**
- Create: `frontend/components/AdminUsers.tsx`
- Modify: `frontend/app/admin/users/page.tsx`

**Interfaces:**
- Consumes: `listAdminUsers`, `AdminUser` (Task 3); `adminUserShortId`, `adminUserRoleLabel`, `formatRegistrationMonth`; `isLikelyTestContent`; `padCount` from `lib/org-seats`; `Chip`, `Button`, `Skeleton`, `EmptyState`, `AdminDesktopOnly`.
- Produces: `AdminUsers` React client component (default rail wired in Task 5), `PAGE_SIZE = 100`.

- [ ] **Step 1: Create the component with the registry column only**

Create `frontend/components/AdminUsers.tsx`:

```tsx
"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatRegistrationMonth } from "@/lib/admin-registration";
import { adminUserShortId } from "@/lib/admin-id";
import { isLikelyTestContent } from "@/lib/admin-test-heuristic";
import { adminUserRoleLabel } from "@/lib/admin-user-role";
import { listAdminUsers, type AdminUser } from "@/lib/api";
import { cn } from "@/lib/cn";
import { padCount } from "@/lib/org-seats";

const GRID = "grid-cols-[44px_1fr_96px_84px_96px]";
const HEADS = ["ID", "Пользователь", "Регистрация", "Записей", "Роль"] as const;
const PAGE_SIZE = 100;

/** A4 · Пользователи и контент-гигиена — 1fr 300px split. */
export function AdminUsers() {
  return (
    <div className="grid min-h-[calc(100vh-56px)] grid-cols-[1fr_300px]">
      <UserRegistry />
    </div>
  );
}

function UserRegistry() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [more, setMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    listAdminUsers({ limit: PAGE_SIZE })
      .then((rows) => {
        if (cancelled) return;
        setUsers(rows);
        setMore(rows.length === PAGE_SIZE);
        setError(false);
      })
      .catch(() => {
        if (!cancelled) {
          setUsers([]);
          setError(true);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  const loadMore = useCallback(async () => {
    if (loadingMore) return;
    setLoadingMore(true);
    try {
      const rows = await listAdminUsers({ limit: PAGE_SIZE, offset: users.length });
      setUsers((prev) => [...prev, ...rows]);
      setMore(rows.length === PAGE_SIZE);
    } catch {
      setMore(false);
      setError(true);
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, users.length]);

  if (loading) {
    return (
      <div className="flex flex-col gap-[8px] border-r border-paper px-[16px] py-[14px]">
        <Skeleton className="h-[28px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
      </div>
    );
  }

  if (error && users.length === 0) {
    return (
      <div className="border-r border-paper">
        <EmptyState
          numeral="!"
          title="Не удалось загрузить реестр"
          text="Проверьте соединение и попробуйте ещё раз."
          actions={
            <Button
              variant="inverted"
              onClick={() => {
                setLoading(true);
                setTick((n) => n + 1);
              }}
            >
              Повторить
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col border-r border-paper">
      <div className={cn("grid border-b border-paper bg-surface-head", GRID)}>
        {HEADS.map((h, i) => (
          <span
            key={h}
            className={cn(
              "cap px-[8px] py-[6px] text-muted-2",
              i === 0 && "text-center",
              i === 1 && "px-[12px]",
            )}
          >
            {h}
          </span>
        ))}
      </div>

      {users.length === 0 ? (
        <EmptyState
          numeral="00"
          title="Пользователей нет"
          text="В реестре пока нет активных аккаунтов."
          actions={
            <Link
              href="/admin"
              className="swiss-focus inline-flex min-h-[44px] items-center justify-center bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink"
            >
              К обзору
            </Link>
          }
        />
      ) : (
        users.map((u) => {
          const test = isLikelyTestContent(u.name, u.email);
          const role = adminUserRoleLabel(u, { test });
          return (
            <div
              key={u.id}
              className={cn("grid min-h-[44px] items-center border-b border-rule-inner", GRID)}
            >
              <span
                className="px-[6px] py-[11px] text-center font-mono text-[9.5px] text-muted-2"
                title={u.id}
                data-id={u.id}
              >
                {adminUserShortId(u.id)}
              </span>

              <span className="min-w-0 px-[12px] py-[10px]">
                <span
                  className={cn(
                    "block truncate text-[12px] font-bold leading-[1.25]",
                    test && "text-signal",
                  )}
                >
                  {u.name || "—"}
                </span>
                <span className="cap mt-[2px] block truncate text-muted-2">
                  {u.email || "—"}
                </span>
              </span>

              <span className="px-[8px] py-[10px] font-mono text-[10px] text-text-dim">
                {formatRegistrationMonth(u.created_at)}
              </span>

              <span className="px-[8px] py-[10px] font-mono text-[11px] font-bold">
                {padCount(u.bookings)}
              </span>

              <span className="px-[8px] py-[10px]">
                <Chip
                  as="span"
                  variant={test ? "signal" : "dark-muted"}
                  className="px-[6px] py-[2px] text-[7.5px]"
                >
                  {role}
                </Chip>
              </span>
            </div>
          );
        })
      )}

      {more ? (
        <div className="px-[16px] py-[12px]">
          <Button
            variant="dark-ghost"
            size="sm"
            disabled={loadingMore}
            onClick={loadMore}
            className="min-h-[44px] px-[11px]"
          >
            {loadingMore ? "Загружаем…" : "Показать ещё"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
```

- [ ] **Step 2: Mount it on the route (replacing the P6 stub)**

Replace the entire contents of `frontend/app/admin/users/page.tsx`:

```tsx
import { AdminDesktopOnly } from "@/components/AdminDesktopOnly";
import { AdminUsers } from "@/components/AdminUsers";

export default function Page() {
  return (
    <AdminDesktopOnly>
      <AdminUsers />
    </AdminDesktopOnly>
  );
}
```

- [ ] **Step 3: Verify build + tests + lint**

Run: `cd frontend && pnpm build && pnpm test && pnpm lint`
Expected: all PASS (the right column is empty until Task 5 — that is expected at this checkpoint).

- [ ] **Step 4: Commit**

```bash
git add frontend/components/AdminUsers.tsx frontend/app/admin/users/page.tsx
git commit -m "feat(frontend): Swiss Grid A4 user registry table"
```

---

## Task 5: Frontend — A4 right rail (Гигиена контента + СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ)

**Files:**
- Modify: `frontend/components/AdminUsers.tsx`

**Interfaces:**
- Consumes: `listHygieneIssues`, `hideAllHygiene`, `HygieneIssue` (Task 3); `hygieneKindLabel`, `hygieneIssueValue`, `hygieneIssueSource`; `Button`, `Skeleton`.
- Produces: `HygieneRail` (internal to `AdminUsers.tsx`), rendered as the second grid column.

- [ ] **Step 1: Add the rail component**

In `frontend/components/AdminUsers.tsx`, extend the imports:

```tsx
import {
  hideAllHygiene,
  listAdminUsers,
  listHygieneIssues,
  type AdminUser,
  type HygieneIssue,
} from "@/lib/api";
import {
  hygieneIssueSource,
  hygieneIssueValue,
  hygieneKindLabel,
} from "@/lib/hygiene-labels";
```

and render the rail from the root component:

```tsx
export function AdminUsers() {
  return (
    <div className="grid min-h-[calc(100vh-56px)] grid-cols-[1fr_300px]">
      <UserRegistry />
      <HygieneRail />
    </div>
  );
}
```

Append the rail at the end of the file:

```tsx
function HygieneRail() {
  const [issues, setIssues] = useState<HygieneIssue[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [hiding, setHiding] = useState(false);
  const [done, setDone] = useState<{ hidden: number; skipped: number } | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    listHygieneIssues()
      .then((rows) => {
        if (cancelled) return;
        setIssues(rows);
        setError("");
      })
      .catch(() => {
        if (!cancelled) {
          setIssues([]);
          setError("Не удалось загрузить гигиену контента");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload() {
    setLoading(true);
    setConfirming(false);
    setTick((n) => n + 1);
  }

  async function onHideAll() {
    if (hiding) return;
    setHiding(true);
    try {
      const res = await hideAllHygiene();
      setDone(res);
      setError("");
      reload();
    } catch {
      setError("Не удалось скрыть события");
    } finally {
      setHiding(false);
    }
  }

  return (
    <div className="flex flex-col">
      <div className="border-b border-paper px-[16px] py-[11px]">
        <span className="cap font-bold text-signal">
          Гигиена контента · <span className="font-mono">{loading ? "—" : issues.length}</span>
        </span>
      </div>

      {loading ? (
        <div className="flex flex-col gap-[8px] px-[16px] py-[10px]">
          <Skeleton className="h-[52px] w-full" />
          <Skeleton className="h-[52px] w-full" />
          <Skeleton className="h-[52px] w-full" />
        </div>
      ) : error ? (
        <div className="flex flex-col items-start gap-[8px] px-[16px] py-[10px]">
          <p className="text-[11px] text-signal">{error}</p>
          <Button variant="dark-ghost" size="sm" className="min-h-[44px]" onClick={reload}>
            Повторить
          </Button>
        </div>
      ) : issues.length === 0 ? (
        <div className="px-[16px] py-[10px]">
          <p className="text-[11px] font-bold leading-[1.2]">Проблем не найдено</p>
          <p className="cap mt-[3px] text-muted-2">
            {done
              ? `скрыто ${done.hidden}, пропущено ${done.skipped}`
              : "в ленте нет тестовых данных и странных цен"}
          </p>
        </div>
      ) : (
        issues.map((issue) => {
          const source = hygieneIssueSource(issue);
          return (
            <div
              key={`${issue.kind}-${issue.event_id}`}
              className="border-b border-rule-inner px-[16px] py-[10px]"
            >
              <div className="cap mb-[3px] text-muted-2">{hygieneKindLabel(issue.kind)}</div>
              <div className="text-[11px] font-bold leading-[1.2]">
                {hygieneIssueValue(issue)}
              </div>
              {source ? <div className="cap mt-[3px] text-muted-2">{source}</div> : null}
            </div>
          );
        })
      )}

      {!loading && issues.length > 0 ? (
        <div className="mt-auto px-[16px] py-[11px]">
          {confirming ? (
            <div className="flex flex-col gap-[8px]">
              <p className="text-[11px] leading-[1.35] text-text-dim">
                Скрыть <span className="font-mono font-bold">{issues.length}</span> событий из
                ленты? Их можно вернуть в «Модерации».
              </p>
              <div className="flex gap-[8px]">
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={hiding}
                  onClick={onHideAll}
                  className="min-h-[44px] flex-1 text-[10px]"
                >
                  {hiding ? "Скрываем…" : "Подтвердить"}
                </Button>
                <Button
                  variant="dark-ghost"
                  size="sm"
                  disabled={hiding}
                  onClick={() => setConfirming(false)}
                  className="min-h-[44px] px-[10px] text-[10px]"
                >
                  Отмена
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="destructive"
              onClick={() => setConfirming(true)}
              className="w-full min-h-[44px] py-[9px] text-[10px]"
            >
              Скрыть всё из ленты
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
}
```

- [ ] **Step 2: Verify build + tests + lint**

Run: `cd frontend && pnpm build && pnpm test && pnpm lint`
Expected: all PASS.

- [ ] **Step 3: Manual smoke against a local backend (optional but preferred)**

Run, in two terminals:

```bash
cd backend && make run
cd frontend && pnpm dev
```

Sign in as an admin, open `http://localhost:3000/admin/users` and confirm: the table renders real rows, the rail lists issues (or «Проблем не найдено»), and clicking «СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ` shows the confirm step **without** firing a request.
If no local DB is available, note it and rely on Task 7's live verification instead.

- [ ] **Step 4: Commit**

```bash
git add frontend/components/AdminUsers.tsx
git commit -m "feat(frontend): Swiss Grid A4 content hygiene rail"
```

---

## Task 6: Frontend — fidelity sweep and state coverage

**Files:**
- Modify: `frontend/components/AdminUsers.tsx` (fixes found by the sweep)

- [ ] **Step 1: Run the grep checks**

```bash
cd frontend
rg -n "rounded-|shadow-" components/AdminUsers.tsx app/admin/users/page.tsx    # expect: no matches
rg -n "text-(green|amber|red|blue)-" components/AdminUsers.tsx                 # expect: no matches
rg -n "Загрузка\.\.\.|Loading" components/AdminUsers.tsx                       # expect: no matches
rg -n "grid-cols-\[1fr_300px\]|grid-cols-\[44px_1fr_96px_84px_96px\]" components/AdminUsers.tsx
```

Expected: the first three return nothing; the last returns both grid definitions. Fix any hit before continuing.

- [ ] **Step 2: Walk the fidelity contract**

Check each row F1–F15 of the *Design fidelity contract* against the component source with the reference open (`python3 -m http.server 8099`, badge A4). Fix mismatches in `AdminUsers.tsx`. Specifically confirm:

- left column carries `border-r border-paper` (F2) and the rail does not;
- header cells use `.cap text-muted-2` on `bg-surface-head` with `border-b border-paper` (F4);
- every numeric cell (`ID`, `Регистрация`, `Записей`, the rail count, the confirm count) is `font-mono` (Global Constraint);
- red appears only on: test names, the «Тестовый» chip, the rail caption, the destructive button, and error text (F15).

- [ ] **Step 3: Verify the four states by forcing them**

With the app running (`pnpm dev`), confirm each state renders per U8 / handoff rules:

| State | How to force | Expected |
|---|---|---|
| Loading | throttle network in devtools | ink `Skeleton` cells in both columns; rail count `—`; no spinner |
| Registry error | stop the backend, reload | `EmptyState` `!` + «Не удалось загрузить реестр» + «ПОВТОРИТЬ» |
| Registry empty | `listAdminUsers` returns `[]` (e.g. hit an empty DB) | `EmptyState` `00` + «Пользователей нет» + «К ОБЗОРУ» |
| Hygiene clean | no matching events | «Проблем не найдено» + caption, **no** destructive CTA |
| Desktop gate | resize below 900px | «Админ-инструменты — с экрана от 900px» + «К обзору» |

- [ ] **Step 4: Keyboard pass**

Tab through the page: the «Показать ещё», «Скрыть всё из ленты», «Подтвердить», «Отмена» and «Повторить` controls must all take a visible 2px square focus outline (`swiss-focus`) and be reachable in DOM order. Fix any control that is not focusable.

- [ ] **Step 5: Verify build + tests + lint**

Run: `cd frontend && pnpm build && pnpm test && pnpm lint`
Expected: all PASS.

- [ ] **Step 6: Commit (only if the sweep changed anything)**

```bash
git add frontend/components/AdminUsers.tsx
git commit -m "fix(frontend): A4 fidelity and state polish"
```

---

## Task 7: Verification, docs, review, deploy

**Files:**
- Create: `docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md`, `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md`

- [ ] **Step 1: Full verification run**

```bash
cd backend && go build ./... && go test ./... && golangci-lint run ./... ; cd ..
cd frontend && pnpm build && pnpm test && pnpm lint ; cd ..
```

Record the actual outputs (test counts included) — they go into the report verbatim. **Do not claim any of these passed without pasting the output** (`superpowers:verification-before-completion`).

- [ ] **Step 2: Request code review**

Use `superpowers:requesting-code-review` on the branch diff. Handle feedback with `superpowers:receiving-code-review`. Re-run Step 1 after any change.

- [ ] **Step 3: Write the phase report**

Create `docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md` mirroring the Phase-6 report's structure: **Shipped** table (A4 route + the three endpoints), **Deliberate deviations** (copy the 14 above with any review amendments), **Parked / deferred** (search UI; A1 «Пользователей» tile still `—`; per-user actions; hygiene kinds beyond the two), **Verification** table with the real command outputs, **Commits** table, **Deploy** section, **Merge notes**.

- [ ] **Step 4: Merge to `main`**

Follow `superpowers:finishing-a-development-branch`. No force-push; rebase onto latest `main` if it drifted.

- [ ] **Step 5: Deploy (backend + frontend) — LAST**

Follow `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`. Phase 7 specifics:

- **No migration.** Prod DB stays where it is (deviation 2) — do **not** run `migrate up`, do **not** scp any `.sql`.
- **Backend image must be rebuilt** (new endpoints): build on the Mac for `linux/amd64`, `docker save | ssh | docker load`, then recreate with **all 4 compose files** + `--no-build`.
- **Frontend image must be rebuilt** with **both** build-args:
  `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` and `NEXT_PUBLIC_YANDEX_MAPS_KEY=<key>` (a missing arg silently degrades the site — see `lia-demo-deployment`).
- After verify, **prune Docker** (builder prune + dangling image prune + trim `rollback-*` to the last ~3) — the box has a 20 GB disk (`lia-deploy-image-cleanup`).

- [ ] **Step 6: Live verification on https://presence.tarski.ru**

As an admin account, at desktop width:

```bash
# smoke the API directly first
curl -s -H "Authorization: Bearer $TOKEN" https://api.presence.tarski.ru/api/v1/admin/users?limit=3 | head -c 400
curl -s -H "Authorization: Bearer $TOKEN" https://api.presence.tarski.ru/api/v1/admin/hygiene | head -c 400
```

Then in the browser on `/admin/users`, confirm against badge A4: `1fr 300px` split; the five columns at `44px 1fr 96px 84px 96px`; mono ID / month / count; role chips; test rows red; the rail caption in red with the mono count; the destructive footer CTA and its confirm step; and — only if there is a genuine issue to clear and the operator agrees — one real «Подтвердить», then check `/admin/moderation/events` shows the hidden events under «Все» as `rejected`. Screenshot desktop + the <900px notice.

- [ ] **Step 7: Write the handoff and commit the docs**

Create `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md` in the Phase-6 handoff's shape: what shipped (P1–P7), what remains (master-plan **7.3** final sweep: delete dead Liquid Glass code, `ThemeSwitch`, `lib/covers.ts` gradients, decorative `CategoryGlyph`, zero remaining `text-[Npx]` grep, font preload, Lighthouse/CLS), design authority table, hard constraints, code landmarks, recommended next work, and a minimal prompt for the next planning agent.

```bash
git add docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md
git commit -m "docs: Phase 7 A4 report and handoff"
```

- [ ] **Step 8: Update memory**

Update `~/.claude/projects/-Users-dodonovpavel-gateway-fm-REAL-WORLD-ASSETS-1-lia/memory/lia-project-state.md` and `lia-demo-deployment.md`: Phase 7 merged/deployed state, the three new admin endpoints, "no migration — prod DB level unchanged", and the new image tags.

---

## Self-review notes

**Spec coverage** — README § A4 requirements mapped to tasks: `1fr 300px` split → Task 4/5 (F1); registry columns `44px 1fr 96px 84px 96px` → Task 4 (F3); mono ID / name+email / mono registration month / mono booking count / role chip → Task 4 (F6–F10); test accounts in red → Task 4 (deviation 4); «Гигиена контента · N» panel with one block per issue (type caption / value 11px-700 / source caption) → Task 5 (F11–F12); destructive footer «СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ» → Task 5 (F13); `GET /admin/users` + `GET /admin/hygiene` (README data table row A4) → Tasks 1–2, plus `POST /admin/hygiene/hide-all` which the footer CTA requires; admin inversion + desktop-only → Tasks 4/6 and the layout inherited from P6.

**Known gaps deliberately left open** (all listed as deviations, none silently dropped): no search UI (9); no per-user actions (11); A1 users tile stays `—` (14); hygiene detects only the two designed kinds — duplicates, mentioned in the mock's prose note, have no detector and are **not** claimed anywhere in the UI.

**Type consistency** — `adminusers.Row`/`Filter`/`Repository`/`Service` used identically in Tasks 1's service, repository, handler and wiring; `hygiene.Issue`/`Kind`/`Result`/`Service` identical across Task 2's detector, service, handler and wiring; the JSON keys emitted by `adminUserJSON` (`id,name,email,role,created_at,bookings,is_organizer`) match the TS `AdminUser` exactly, and `hygiene.Issue`'s struct tags (`kind,event_id,title,organizer_name,price_rub`) match the TS `HygieneIssue` exactly; `padCount` is imported from `@/lib/org-seats` where it already lives (not redefined).
