# Complaints / Reports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an authenticated user report an event («Пожаловаться»), and give staff a grouped `/admin/complaints` inbox to triage and resolve reports — reusing `moderation.Takedown` to act on confirmed violations, auditing every resolution.

**Architecture:** A new `internal/complaints` domain (service + pg repository) mirrors `internal/moderation`. The complaint table uses a generic `target_type`/`target_id` (event-only CHECK now). A public authed submit handler (`internal/http/complaints`) is mounted ahead of the swagger mux; admin inbox endpoints are added to the existing `internal/http/admin` handler. `complaints.Service` is constructed with `moderation.Service` injected so the takedown branch of resolution composes the existing status transition instead of duplicating SQL. Frontend adds a report modal on event detail and a grouped admin inbox page.

**Tech Stack:** Go 1.26 modular monolith (`github.com/Pashteto/lia`), go-pg v10, PostgreSQL, plain `net/http` handlers (no go-swagger spec edits), golang-migrate SQL migrations. Frontend: Next.js App Router + TS + Tailwind v4 + pnpm.

## Global Constraints

- **No go-swagger spec edits.** All new endpoints live on plain `net/http` handlers mounted ahead of the swagger mux (same approach as `internal/http/admin`, `internal/http/organizers`, `internal/http/uploads`). Backend rebuilds still need `make generate-api` first (gitignored swagger model must exist to compile) — but this slice adds **no** swagger fields.
- **Auth model:** submit is authed (Bearer; reporter = the authenticated user). Inbox/resolve are admin-gated via the existing `h.staff(...)` wrapper (`role == "admin"` → through; 403 «Недостаточно прав»; 401 anon).
- **go-pg gotchas:** never scan SQL `NULL` into a non-pointer `uuid.UUID` (use `*uuid.UUID` for `resolved_by`); avoid `RETURNING *`; read optional text via `coalesce(col,'')`.
- **Audit/compliance (ISO 27001 / Vanta):** every resolution writes one `audit_log` row (`action='complaint.resolve'`). `reporter_user_id` is stored but exposed only to staff — never on any public surface or to the reported organizer.
- **golangci-lint is v1** locally and in CI — do not migrate config to v2.
- **Backend working dir is `backend/`**; migrations live in `backend/db/migrations/`. Frontend working dir is `frontend/`.
- **Russian UI copy** throughout user-facing strings (existing convention).

---

## File Structure

**Backend (create):**
- `backend/db/migrations/000017_complaints.up.sql` / `.down.sql` — table + indexes.
- `backend/internal/complaints/service.go` — types, errors, `Service`, `Repository` interfaces, service impl.
- `backend/internal/complaints/repository.go` — pg-backed `Repository`.
- `backend/internal/complaints/service_test.go` — unit tests (fake repo + fake moderation).
- `backend/internal/complaints/repository_test.go` — `//go:build integration` DB test.
- `backend/internal/http/complaints/handler.go` — public submit handler.
- `backend/internal/http/complaints/handler_test.go` — handler tests (fake service).

**Backend (modify):**
- `backend/internal/http/admin/handler.go` — add 3 inbox routes + `complaints_open` on overview + `Complaints` dep.
- `backend/internal/http/admin/handler_test.go` — add resolve/overview tests with a stub complaints service.
- `backend/internal/http/module.go` — `complaints` field, `SetComplaints`, mount public handler, pass into `admin.Deps`.
- `backend/internal/application.go` — build `complaints.Service` (inject the shared `moderation.Service`) and wire it.

**Frontend (create):**
- `frontend/components/ReportButton.tsx` — «Пожаловаться» control + modal (client).
- `frontend/app/admin/complaints/page.tsx` — grouped inbox.

**Frontend (modify):**
- `frontend/lib/api.ts` — `ComplaintGroup`/`ComplaintCategory` types, `COMPLAINT_CATEGORIES`, `submitComplaint`, `listComplaints`, `resolveComplaints`, extend `getAdminOverview` return type.
- `frontend/components/EventDetailView.tsx` — render `<ReportButton>`.
- `frontend/app/admin/page.tsx` — `complaints_open` stat + link to `/admin/complaints`.

---

## Task 1: Migration `000017_complaints`

**Files:**
- Create: `backend/db/migrations/000017_complaints.up.sql`
- Create: `backend/db/migrations/000017_complaints.down.sql`

**Interfaces:**
- Consumes: existing `events` table, `audit_log` table (migration 000014), `gen_random_uuid()` (pgcrypto, already enabled in 000014).
- Produces: `complaints` table with columns `id, target_type, target_id, reporter_user_id, category, note, status, resolution, resolved_by, resolved_at, created_at`; partial unique index `complaints_one_open_per_reporter`; partial index `complaints_open_target_idx`.

- [ ] **Step 1: Write the up migration**

Create `backend/db/migrations/000017_complaints.up.sql`:

```sql
CREATE TABLE complaints (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type      TEXT NOT NULL DEFAULT 'event'
                     CHECK (target_type IN ('event')),
    target_id        uuid NOT NULL,
    reporter_user_id uuid NOT NULL,
    category         TEXT NOT NULL
                     CHECK (category IN ('spam','fraud','inappropriate','duplicate','other')),
    note             TEXT,
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open','resolved','dismissed')),
    resolution       TEXT,
    resolved_by      uuid,
    resolved_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- One OPEN complaint per reporter per target (repeat-submit dedup).
CREATE UNIQUE INDEX complaints_one_open_per_reporter
    ON complaints (target_type, target_id, reporter_user_id) WHERE status = 'open';

-- Fast "events with open complaints" grouping.
CREATE INDEX complaints_open_target_idx
    ON complaints (target_type, target_id) WHERE status = 'open';
```

- [ ] **Step 2: Write the down migration**

Create `backend/db/migrations/000017_complaints.down.sql`:

```sql
DROP TABLE IF EXISTS complaints;
```

- [ ] **Step 3: Verify the migration applies and rolls back**

The repo's local Docker is flaky; use the host-run Postgres if `docker compose` is up, otherwise rely on the integration-test DB. With a reachable Postgres:

Run:
```bash
cd backend && docker compose up -d db && \
  docker compose run --rm migrate -path /migrations -database "$DATABASE_URL" up
```
Expected: migration `000017` reports applied with no error; `\d complaints` shows the table and two indexes.

If Docker is unavailable, defer verification to Task 2's integration test (which migrates the test DB). Do not block — note it in the commit.

- [ ] **Step 4: Commit**

```bash
cd backend && git add db/migrations/000017_complaints.up.sql db/migrations/000017_complaints.down.sql
git commit -m "feat(db): complaints table (migration 000017)"
```

---

## Task 2: `internal/complaints` domain (service + repository)

**Files:**
- Create: `backend/internal/complaints/service.go`
- Create: `backend/internal/complaints/repository.go`
- Create: `backend/internal/complaints/service_test.go`
- Create: `backend/internal/complaints/repository_test.go`

**Interfaces:**
- Consumes: `moderation.Service` (`Takedown(ctx, eventID, actorID, reason) error`, returns `moderation.ErrInvalidTransition` when the event is not `published`); `*pg.DB`.
- Produces:
  - `type Complaint struct { ID, TargetID, ReporterUserID uuid.UUID; TargetType, Category, Note, Status, Resolution string; ResolvedBy *uuid.UUID; ResolvedAt *time.Time; CreatedAt time.Time }`
  - `type EventReportGroup struct { TargetID uuid.UUID `json:"event_id"`; EventTitle string `json:"event_title"`; EventStatus string `json:"event_status"`; ReportCount int `json:"report_count"`; Categories map[string]int `json:"categories"`; LatestNote string `json:"latest_note"`; LatestAt time.Time `json:"latest_at"` }`
  - Errors: `ErrInvalidCategory`, `ErrTargetNotFound`, `ErrResolutionRequired`, `ErrInvalidAction`.
  - `type Service interface { Submit(ctx, reporterID uuid.UUID, targetType string, targetID uuid.UUID, category, note string) (created bool, err error); ListInbox(ctx) ([]EventReportGroup, error); TargetDetail(ctx, targetType string, targetID uuid.UUID) ([]Complaint, error); Resolve(ctx, targetType string, targetID, actorID uuid.UUID, action, resolution string) error; OpenEventCount(ctx) (int, error) }`
  - `type Repository interface { Insert(ctx, c Complaint) (bool, error); InboxGroups(ctx) ([]EventReportGroup, error); TargetComplaints(ctx, targetType string, targetID uuid.UUID) ([]Complaint, error); ResolveOpenForTarget(ctx, targetType string, targetID, actorID uuid.UUID, status, resolution string) (int, error); OpenEventCount(ctx) (int, error); EventExists(ctx, id uuid.UUID) (bool, error) }`
  - `func NewService(repo Repository, mod moderation.Service) Service`
  - `func NewRepository(db *pg.DB) Repository`

- [ ] **Step 1: Write `service.go` (types, errors, interfaces, service impl)**

Create `backend/internal/complaints/service.go`:

```go
// Package complaints implements user-filed reports against events and the
// staff resolution workflow (grouped per event; takedown reuses the moderation
// domain). See spec docs/superpowers/specs/2026-06-26-complaints-reports-design.md.
package complaints

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/moderation"
)

var validCategories = map[string]bool{
	"spam": true, "fraud": true, "inappropriate": true, "duplicate": true, "other": true,
}

// Domain errors (mapped to HTTP status by the handlers).
var (
	ErrInvalidCategory    = errors.New("complaints: invalid category")      // 400
	ErrTargetNotFound     = errors.New("complaints: target not found")      // 404
	ErrResolutionRequired = errors.New("complaints: resolution required")   // 400
	ErrInvalidAction      = errors.New("complaints: invalid resolve action") // 400
)

// Complaint is one filed report.
type Complaint struct {
	ID             uuid.UUID
	TargetType     string
	TargetID       uuid.UUID
	ReporterUserID uuid.UUID
	Category       string
	Note           string
	Status         string
	Resolution     string
	ResolvedBy     *uuid.UUID
	ResolvedAt     *time.Time
	CreatedAt      time.Time
}

// EventReportGroup is one row of the grouped admin inbox.
type EventReportGroup struct {
	TargetID    uuid.UUID      `json:"event_id"`
	EventTitle  string         `json:"event_title"`
	EventStatus string         `json:"event_status"`
	ReportCount int            `json:"report_count"`
	Categories  map[string]int `json:"categories"`
	LatestNote  string         `json:"latest_note"`
	LatestAt    time.Time      `json:"latest_at"`
}

// Repository persists complaints and resolves them atomically (with audit).
type Repository interface {
	Insert(ctx context.Context, c Complaint) (bool, error) // false = idempotent skip (open dup)
	InboxGroups(ctx context.Context) ([]EventReportGroup, error)
	TargetComplaints(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error)
	ResolveOpenForTarget(ctx context.Context, targetType string, targetID, actorID uuid.UUID, status, resolution string) (int, error)
	OpenEventCount(ctx context.Context) (int, error)
	EventExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Service is the complaints use-case layer.
type Service interface {
	Submit(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, category, note string) (bool, error)
	ListInbox(ctx context.Context) ([]EventReportGroup, error)
	TargetDetail(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error)
	Resolve(ctx context.Context, targetType string, targetID, actorID uuid.UUID, action, resolution string) error
	OpenEventCount(ctx context.Context) (int, error)
}

type service struct {
	repo Repository
	mod  moderation.Service
}

// NewService returns a complaints Service. mod is used by the takedown branch
// of Resolve to reuse the moderation status transition.
func NewService(repo Repository, mod moderation.Service) Service {
	return &service{repo: repo, mod: mod}
}

func (s *service) Submit(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, category, note string) (bool, error) {
	if targetType == "" {
		targetType = "event"
	}
	if !validCategories[category] {
		return false, ErrInvalidCategory
	}
	exists, err := s.repo.EventExists(ctx, targetID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, ErrTargetNotFound
	}
	return s.repo.Insert(ctx, Complaint{
		TargetType:     targetType,
		TargetID:       targetID,
		ReporterUserID: reporterID,
		Category:       category,
		Note:           strings.TrimSpace(note),
		Status:         "open",
	})
}

func (s *service) ListInbox(ctx context.Context) ([]EventReportGroup, error) {
	return s.repo.InboxGroups(ctx)
}

func (s *service) TargetDetail(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error) {
	if targetType == "" {
		targetType = "event"
	}
	return s.repo.TargetComplaints(ctx, targetType, targetID)
}

func (s *service) Resolve(ctx context.Context, targetType string, targetID, actorID uuid.UUID, action, resolution string) error {
	if targetType == "" {
		targetType = "event"
	}
	resolution = strings.TrimSpace(resolution)
	switch action {
	case "takedown":
		if resolution == "" {
			return ErrResolutionRequired
		}
		// Reuse the moderation transition (its own tx: status + history + audit).
		// On ErrInvalidTransition (event not published) we surface it and leave
		// the complaints open.
		if err := s.mod.Takedown(ctx, targetID, actorID, resolution); err != nil {
			return err
		}
		_, err := s.repo.ResolveOpenForTarget(ctx, targetType, targetID, actorID, "resolved", resolution)
		return err
	case "dismiss":
		_, err := s.repo.ResolveOpenForTarget(ctx, targetType, targetID, actorID, "dismissed", resolution)
		return err
	default:
		return ErrInvalidAction
	}
}

func (s *service) OpenEventCount(ctx context.Context) (int, error) {
	return s.repo.OpenEventCount(ctx)
}
```

- [ ] **Step 2: Write `service_test.go` (failing — service.go exists but tests assert behavior)**

Create `backend/internal/complaints/service_test.go`:

```go
package complaints

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/moderation"
)

type fakeRepo struct {
	inserted        Complaint
	insertCreated   bool
	insertErr       error
	eventExists     bool
	eventExistsErr  error
	resolveStatus   string
	resolveCalled   bool
	resolveErr      error
}

func (f *fakeRepo) Insert(_ context.Context, c Complaint) (bool, error) {
	f.inserted = c
	return f.insertCreated, f.insertErr
}
func (f *fakeRepo) InboxGroups(context.Context) ([]EventReportGroup, error) { return nil, nil }
func (f *fakeRepo) TargetComplaints(context.Context, string, uuid.UUID) ([]Complaint, error) {
	return nil, nil
}
func (f *fakeRepo) ResolveOpenForTarget(_ context.Context, _ string, _, _ uuid.UUID, status, _ string) (int, error) {
	f.resolveCalled = true
	f.resolveStatus = status
	return 1, f.resolveErr
}
func (f *fakeRepo) OpenEventCount(context.Context) (int, error) { return 0, nil }
func (f *fakeRepo) EventExists(context.Context, uuid.UUID) (bool, error) {
	return f.eventExists, f.eventExistsErr
}

// fakeMod records the takedown reason and returns a configurable error.
type fakeMod struct {
	moderation.Service
	takedownErr    error
	takedownReason string
}

func (m *fakeMod) Takedown(_ context.Context, _, _ uuid.UUID, reason string) error {
	m.takedownReason = reason
	return m.takedownErr
}

func TestSubmit_InvalidCategory(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	_, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "bogus", "")
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("err = %v, want ErrInvalidCategory", err)
	}
}

func TestSubmit_TargetNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{eventExists: false}, &fakeMod{})
	_, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "spam", "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("err = %v, want ErrTargetNotFound", err)
	}
}

func TestSubmit_InsertsTrimmedNote(t *testing.T) {
	repo := &fakeRepo{eventExists: true, insertCreated: true}
	svc := NewService(repo, &fakeMod{})
	created, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "spam", "  hi  ")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !created {
		t.Fatalf("created = false, want true")
	}
	if repo.inserted.Note != "hi" || repo.inserted.Status != "open" || repo.inserted.Category != "spam" {
		t.Fatalf("inserted = %+v", repo.inserted)
	}
}

func TestResolve_TakedownRequiresResolution(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "   ")
	if !errors.Is(err, ErrResolutionRequired) {
		t.Fatalf("err = %v, want ErrResolutionRequired", err)
	}
}

func TestResolve_TakedownComposesModeration(t *testing.T) {
	repo := &fakeRepo{}
	mod := &fakeMod{}
	svc := NewService(repo, mod)
	if err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "scam"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if mod.takedownReason != "scam" {
		t.Fatalf("takedown reason = %q, want scam", mod.takedownReason)
	}
	if !repo.resolveCalled || repo.resolveStatus != "resolved" {
		t.Fatalf("resolve status = %q (called=%v), want resolved", repo.resolveStatus, repo.resolveCalled)
	}
}

func TestResolve_TakedownInvalidTransitionDoesNotCloseComplaints(t *testing.T) {
	repo := &fakeRepo{}
	mod := &fakeMod{takedownErr: moderation.ErrInvalidTransition}
	svc := NewService(repo, mod)
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "scam")
	if !errors.Is(err, moderation.ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
	if repo.resolveCalled {
		t.Fatalf("ResolveOpenForTarget should not be called when takedown fails")
	}
}

func TestResolve_Dismiss(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeMod{})
	if err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "dismiss", ""); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !repo.resolveCalled || repo.resolveStatus != "dismissed" {
		t.Fatalf("resolve status = %q (called=%v), want dismissed", repo.resolveStatus, repo.resolveCalled)
	}
}

func TestResolve_InvalidAction(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "nuke", "x")
	if !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("err = %v, want ErrInvalidAction", err)
	}
}
```

- [ ] **Step 3: Run the unit tests to verify they pass**

Run: `cd backend && go test ./internal/complaints/ -run Test -v`
Expected: PASS (service.go compiles against the interfaces; all 8 tests pass).

- [ ] **Step 4: Write `repository.go`**

Create `backend/internal/complaints/repository.go`:

```go
package complaints

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed complaints Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

// Insert adds an open complaint. The partial unique index makes a repeat open
// complaint from the same reporter a no-op; returns false in that case.
func (r *pgRepository) Insert(ctx context.Context, c Complaint) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO complaints (target_type, target_id, reporter_user_id, category, note, status)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''), 'open')
		 ON CONFLICT (target_type, target_id, reporter_user_id) WHERE status = 'open' DO NOTHING`,
		c.TargetType, c.TargetID, c.ReporterUserID, c.Category, c.Note)
	if err != nil {
		return false, fmt.Errorf("insert complaint: %w", err)
	}
	return res.RowsAffected() > 0, nil
}

func (r *pgRepository) EventExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&exists),
		`SELECT EXISTS(SELECT 1 FROM events WHERE id = ?)`, id); err != nil {
		return false, fmt.Errorf("event exists: %w", err)
	}
	return exists, nil
}

// InboxGroups returns open complaints grouped by event. Aggregation is done in
// Go (clear + testable; demo-scale row counts). Rows are ordered newest-first
// per target so the first non-empty note per group is the latest.
func (r *pgRepository) InboxGroups(ctx context.Context) ([]EventReportGroup, error) {
	var rows []struct {
		TargetID  uuid.UUID `pg:"target_id"`
		Title     string    `pg:"title"`
		Status    string    `pg:"status"`
		Category  string    `pg:"category"`
		Note      string    `pg:"note"`
		CreatedAt time.Time `pg:"created_at"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT c.target_id, e.title, e.status, c.category,
		        coalesce(c.note, '') AS note, c.created_at
		 FROM complaints c
		 JOIN events e ON e.id = c.target_id
		 WHERE c.status = 'open' AND c.target_type = 'event'
		 ORDER BY c.target_id, c.created_at DESC`); err != nil {
		return nil, fmt.Errorf("inbox groups: %w", err)
	}

	byTarget := map[uuid.UUID]*EventReportGroup{}
	order := []uuid.UUID{}
	for _, row := range rows {
		g := byTarget[row.TargetID]
		if g == nil {
			g = &EventReportGroup{
				TargetID: row.TargetID, EventTitle: row.Title, EventStatus: row.Status,
				Categories: map[string]int{}, LatestAt: row.CreatedAt,
			}
			byTarget[row.TargetID] = g
			order = append(order, row.TargetID)
		}
		g.ReportCount++
		g.Categories[row.Category]++
		if g.LatestNote == "" && row.Note != "" {
			g.LatestNote = row.Note // rows are created_at DESC → newest non-empty wins
		}
	}

	out := make([]EventReportGroup, 0, len(order))
	for _, id := range order {
		out = append(out, *byTarget[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ReportCount != out[j].ReportCount {
			return out[i].ReportCount > out[j].ReportCount
		}
		return out[i].LatestAt.After(out[j].LatestAt)
	})
	return out, nil
}

func (r *pgRepository) TargetComplaints(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error) {
	var rows []Complaint
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT id, target_type, target_id, reporter_user_id, category,
		        coalesce(note, '') AS note, status, coalesce(resolution, '') AS resolution,
		        resolved_by, resolved_at, created_at
		 FROM complaints
		 WHERE target_type = ? AND target_id = ?
		 ORDER BY created_at DESC`,
		targetType, targetID); err != nil {
		return nil, fmt.Errorf("target complaints: %w", err)
	}
	return rows, nil
}

// ResolveOpenForTarget flips every open complaint for the target to `status`
// and writes one audit_log row, in one tx. Returns the affected count.
func (r *pgRepository) ResolveOpenForTarget(ctx context.Context, targetType string, targetID, actorID uuid.UUID, status, resolution string) (int, error) {
	var affected int
	err := r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE complaints
			 SET status = ?, resolution = NULLIF(?, ''), resolved_by = ?, resolved_at = now()
			 WHERE target_type = ? AND target_id = ? AND status = 'open'`,
			status, resolution, actorID, targetType, targetID)
		if err != nil {
			return fmt.Errorf("resolve complaints: %w", err)
		}
		affected = res.RowsAffected()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'complaint.resolve', ?, ?,
			         jsonb_build_object('status', ?::text, 'resolution', NULLIF(?, ''), 'resolved_count', ?::int))`,
			actorID, targetType, targetID, status, resolution, affected); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
	return affected, err
}

func (r *pgRepository) OpenEventCount(ctx context.Context) (int, error) {
	var n int
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&n),
		`SELECT count(DISTINCT target_id) FROM complaints
		 WHERE status = 'open' AND target_type = 'event'`); err != nil {
		return 0, fmt.Errorf("open event count: %w", err)
	}
	return n, nil
}
```

- [ ] **Step 5: Write `repository_test.go` (integration, build-tagged)**

Create `backend/internal/complaints/repository_test.go`. Mirrors `internal/moderation/repository_test.go` (build tag + `TEST_DATABASE_URL` gating). Note: per the roadmap the repo's integration tests are not yet wired into a local/CI DB run, so this ships but is not executed in the normal flow.

```go
//go:build integration

package complaints

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

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

func insertPublishedEvent(t *testing.T, db *pg.DB) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	if _, err := db.Exec(
		`INSERT INTO events (id, title, status, starts_at, created_at, updated_at)
		 VALUES (?, 'complaints test event', 'published', ?, now(), now())`,
		id, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("insert test event: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM complaints WHERE target_id = ?`, id) //nolint:errcheck
		db.Exec(`DELETE FROM events WHERE id = ?`, id)            //nolint:errcheck
		db.Exec(`DELETE FROM audit_log WHERE target_id = ?`, id)  //nolint:errcheck
	})
	return id
}

// TestInsert_DedupAndResolve exercises the open-dup index, grouping, and the
// cascading resolve + audit.
func TestInsert_DedupAndResolve(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	eventID := insertPublishedEvent(t, db)
	reporter := uuid.Must(uuid.NewV4())

	// First open complaint inserts.
	created, err := repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: reporter, Category: "spam", Note: "bad", Status: "open"})
	if err != nil || !created {
		t.Fatalf("first insert: created=%v err=%v", created, err)
	}
	// Repeat open complaint from same reporter is a no-op.
	created, err = repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: reporter, Category: "fraud", Note: "again", Status: "open"})
	if err != nil || created {
		t.Fatalf("dup insert: created=%v err=%v, want created=false", created, err)
	}
	// A different reporter inserts.
	if _, err := repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: uuid.Must(uuid.NewV4()), Category: "other", Note: "", Status: "open"}); err != nil {
		t.Fatalf("second reporter insert: %v", err)
	}

	groups, err := repo.InboxGroups(ctx)
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	var g *EventReportGroup
	for i := range groups {
		if groups[i].TargetID == eventID {
			g = &groups[i]
		}
	}
	if g == nil || g.ReportCount != 2 {
		t.Fatalf("group = %+v, want ReportCount 2", g)
	}

	// Dismiss cascades to all open complaints + writes one audit row.
	actor := uuid.Must(uuid.NewV4())
	n, err := repo.ResolveOpenForTarget(ctx, "event", eventID, actor, "dismissed", "not a violation")
	if err != nil || n != 2 {
		t.Fatalf("resolve: n=%d err=%v, want 2", n, err)
	}
	open, err := repo.OpenEventCount(ctx)
	if err != nil {
		t.Fatalf("open count: %v", err)
	}
	_ = open // baseline-relative; just assert the event is gone from groups
	groups, _ = repo.InboxGroups(ctx)
	for _, gr := range groups {
		if gr.TargetID == eventID {
			t.Fatalf("event still in inbox after dismiss")
		}
	}
	var auditCount int
	if _, err := db.QueryOne(pg.Scan(&auditCount),
		`SELECT count(*) FROM audit_log WHERE target_id = ? AND action = 'complaint.resolve'`, eventID); err != nil {
		t.Fatalf("audit count: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit rows = %d, want 1", auditCount)
	}
}
```

- [ ] **Step 6: Build + vet + lint the package**

Run: `cd backend && go build ./internal/complaints/ && go vet ./internal/complaints/ && go test ./internal/complaints/ -run Test`
Expected: build/vet clean; unit tests PASS (integration test is excluded without the `integration` tag).

- [ ] **Step 7: Commit**

```bash
cd backend && git add internal/complaints/
git commit -m "feat(complaints): domain service + pg repository

Grouped-by-event reports with open-dup dedup; resolve cascades + audits.
Takedown branch composes moderation.Service."
```

---

## Task 3: Public submit handler + wiring

**Files:**
- Create: `backend/internal/http/complaints/handler.go`
- Create: `backend/internal/http/complaints/handler_test.go`
- Modify: `backend/internal/http/module.go`
- Modify: `backend/internal/application.go`

**Interfaces:**
- Consumes: `complaints.Service` (`Submit` returns `(created bool, err error)` + the four domain errors); `func(token string) (*domain.User, error)` (auth); the shared `moderation.Service` already built in `application.go`.
- Produces: `POST /api/v1/events/{id}/complaints`; `func (m *Module) SetComplaints(svc complaintsdomain.Service)`.

- [ ] **Step 1: Write `handler.go`**

Create `backend/internal/http/complaints/handler.go`:

```go
// Package complaints provides the public (authed) submit handler for filing a
// report against an event: POST /api/v1/events/{id}/complaints. Mounted ahead
// of the go-swagger mux in internal/http/module.go (mirrors internal/http/organizers).
package complaints

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	domain "github.com/Pashteto/lia/internal/models"
)

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Complaints   complaintsdomain.Service
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted public complaints handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/complaints", h.submit)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *handler) principal(r *http.Request) *domain.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	u, err := h.deps.Authenticate(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || u == nil {
		return nil
	}
	return u
}

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Category string `json:"category"`
		Note     string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	created, err := h.deps.Complaints.Submit(r.Context(), u.UUID, "event", eventID, body.Category, body.Note)
	switch {
	case err == nil && created:
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
	case err == nil && !created:
		// Idempotent repeat of an already-open complaint.
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	case errors.Is(err, complaintsdomain.ErrInvalidCategory):
		writeErr(w, http.StatusBadRequest, "Некорректная категория жалобы")
	case errors.Is(err, complaintsdomain.ErrTargetNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
```

- [ ] **Step 2: Write `handler_test.go`**

Create `backend/internal/http/complaints/handler_test.go`:

```go
package complaints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	domain "github.com/Pashteto/lia/internal/models"
)

func authFn(ok bool) func(string) (*domain.User, error) {
	return func(tok string) (*domain.User, error) {
		if !ok || tok == "" {
			return nil, http.ErrNoCookie
		}
		return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Role: "common"}, nil
	}
}

// stubService implements complaintsdomain.Service; only Submit is exercised here.
type stubService struct {
	created bool
	err     error
}

func (s stubService) Submit(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _, _ string) (bool, error) {
	return s.created, s.err
}
func (s stubService) ListInbox(context.Context) ([]complaintsdomain.EventReportGroup, error) {
	return nil, nil
}
func (s stubService) TargetDetail(context.Context, string, uuid.UUID) ([]complaintsdomain.Complaint, error) {
	return nil, nil
}
func (s stubService) Resolve(context.Context, string, uuid.UUID, uuid.UUID, string, string) error {
	return nil
}
func (s stubService) OpenEventCount(context.Context) (int, error) { return 0, nil }

func newH(authOK bool, svc stubService) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(authOK), Complaints: svc})
}

func req(t *testing.T, authOK bool, svc stubService, body string) *httptest.ResponseRecorder {
	t.Helper()
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+id+"/complaints", strings.NewReader(body))
	if authOK {
		r.Header.Set("Authorization", "Bearer x")
	}
	w := httptest.NewRecorder()
	newH(authOK, svc).ServeHTTP(w, r)
	return w
}

func TestSubmit_401Anon(t *testing.T) {
	if w := req(t, false, stubService{}, `{"category":"spam"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestSubmit_201Created(t *testing.T) {
	if w := req(t, true, stubService{created: true}, `{"category":"spam"}`); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}

func TestSubmit_200Idempotent(t *testing.T) {
	if w := req(t, true, stubService{created: false}, `{"category":"spam"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestSubmit_400BadCategory(t *testing.T) {
	if w := req(t, true, stubService{err: complaintsdomain.ErrInvalidCategory}, `{"category":"x"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSubmit_404NoEvent(t *testing.T) {
	if w := req(t, true, stubService{err: complaintsdomain.ErrTargetNotFound}, `{"category":"spam"}`); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
```

- [ ] **Step 3: Run handler tests to verify they pass**

Run: `cd backend && go test ./internal/http/complaints/ -v`
Expected: PASS (5 tests).

- [ ] **Step 4: Wire `SetComplaints` + mount in `module.go`**

In `backend/internal/http/module.go`:

1. Add import (in the `Pashteto/lia/internal` block, alphabetical):
```go
	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	complaintshttp "github.com/Pashteto/lia/internal/http/complaints"
```

2. Add a struct field near `moderation moderation.Service` (line ~50):
```go
	complaints complaintsdomain.Service
```

3. Add a setter near `SetSettings` (line ~115):
```go
// SetComplaints injects the complaints domain service. Call before Init.
func (m *Module) SetComplaints(svc complaintsdomain.Service) { m.complaints = svc }
```

4. Add `Complaints` to the `admin.NewHandler(admin.Deps{...})` literal (line ~281):
```go
		Complaints:   m.complaints,
```

5. Build the public handler near the organizers handler (line ~292):
```go
	var complaintsH http.Handler
	if m.complaints != nil {
		complaintsH = complaintshttp.NewHandler(complaintshttp.Deps{
			Authenticate: m.auth.Authenticate,
			Complaints:   m.complaints,
		})
	}
```

6. Add a routing predicate in the router func, **before** the `base.ServeHTTP` fallback and after the organizers block (line ~314):
```go
		if complaintsH != nil &&
			strings.HasPrefix(p, "/api/v1/events/") && strings.HasSuffix(p, "/complaints") {
			complaintsH.ServeHTTP(w, r)
			return
		}
```

- [ ] **Step 5: Wire the service in `application.go`**

In `backend/internal/application.go`:

1. Add import (alphabetical in the internal block):
```go
	"github.com/Pashteto/lia/internal/complaints"
```

2. Replace the moderation wiring block (currently lines ~238-246) so the moderation service is built once and shared with complaints:
```go
		// Wire moderation + complaints services (require DB; reuse the same *pg.DB
		// already used by the events and rsvp repositories — no second pool opened).
		if repoModule != nil {
			modRepo := moderation.NewRepository(repoModule.DB())
			modSvc := moderation.NewService(modRepo)
			httpModule.SetModeration(
				modSvc,
				func(id uuid.UUID) (string, error) {
					return modRepo.LatestReason(context.Background(), id)
				},
			)
			httpModule.SetComplaints(
				complaints.NewService(complaints.NewRepository(repoModule.DB()), modSvc),
			)
		}
```

- [ ] **Step 6: Build + vet the whole backend**

Run: `cd backend && make generate-api && go build ./... && go vet ./... && go test ./internal/http/complaints/ ./internal/complaints/`
Expected: build/vet clean; tests PASS. (`make generate-api` regenerates the gitignored swagger model so the formatter compiles — see HANDOFF gotcha.)

- [ ] **Step 7: Run golangci-lint (v1) on the new packages**

Run: `cd backend && golangci-lint run ./internal/complaints/... ./internal/http/complaints/...`
Expected: exit 0.

- [ ] **Step 8: Commit**

```bash
cd backend && git add internal/http/complaints/ internal/http/module.go internal/application.go
git commit -m "feat(complaints): public submit endpoint + service wiring

POST /api/v1/events/{id}/complaints (authed). complaints.Service shares the
moderation.Service so resolve-by-takedown reuses the existing transition."
```

---

## Task 4: Admin inbox endpoints + overview count

**Files:**
- Modify: `backend/internal/http/admin/handler.go`
- Modify: `backend/internal/http/admin/handler_test.go`

**Interfaces:**
- Consumes: `complaints.Service` (`ListInbox`, `TargetDetail`, `Resolve`, `OpenEventCount`) + its domain errors; `moderation.ErrInvalidTransition` (already imported in handler.go).
- Produces: `GET /api/v1/admin/complaints`, `GET /api/v1/admin/complaints/events/{id}`, `POST /api/v1/admin/complaints/events/{id}/resolve`; `complaints_open` key on `/api/v1/admin/overview`.

- [ ] **Step 1: Write the failing handler tests**

In `backend/internal/http/admin/handler_test.go`, add a stub complaints service and tests. Add the import `complaintsdomain "github.com/Pashteto/lia/internal/complaints"` and `"context"` (already present) at the top, then append:

```go
type stubComplaints struct {
	complaintsdomain.Service
	resolveErr error
	openCount  int
}

func (s stubComplaints) ListInbox(context.Context) ([]complaintsdomain.EventReportGroup, error) {
	return []complaintsdomain.EventReportGroup{}, nil
}
func (s stubComplaints) TargetDetail(context.Context, string, uuid.UUID) ([]complaintsdomain.Complaint, error) {
	return nil, nil
}
func (s stubComplaints) Resolve(context.Context, string, uuid.UUID, uuid.UUID, string, string) error {
	return s.resolveErr
}
func (s stubComplaints) OpenEventCount(context.Context) (int, error) { return s.openCount, nil }

func newHandlerWithComplaints(role string, c complaintsdomain.Service) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(role), Moderation: stubMod{}, Complaints: c})
}

func TestComplaints_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/complaints", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("common", stubComplaints{}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestResolve_400OnResolutionRequired(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"takedown","resolution":""}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	c := stubComplaints{resolveErr: complaintsdomain.ErrResolutionRequired}
	newHandlerWithComplaints("admin", c).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestResolve_409OnInvalidTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"takedown","resolution":"scam"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	c := stubComplaints{resolveErr: moderation.ErrInvalidTransition}
	newHandlerWithComplaints("admin", c).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestResolve_200OK(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"dismiss"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("admin", stubComplaints{}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestOverview_IncludesComplaintsOpen(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("admin", stubComplaints{openCount: 4}).ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"complaints_open":4`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/http/admin/ -run 'TestComplaints|TestResolve_4|TestResolve_2|TestOverview_IncludesComplaintsOpen'`
Expected: FAIL — `Deps` has no `Complaints` field / routes not registered.

- [ ] **Step 3: Add the `Complaints` dep, routes, and handlers**

In `backend/internal/http/admin/handler.go`:

1. Add to the `internal` import block: `complaints "github.com/Pashteto/lia/internal/complaints"` and `"errors"`.

2. Add to `Deps`:
```go
	Complaints complaints.Service
```

3. Register routes in `NewHandler` (after the settings routes):
```go
	h.mux.HandleFunc("GET /api/v1/admin/complaints", h.staff(h.listComplaints))
	h.mux.HandleFunc("GET /api/v1/admin/complaints/events/{id}", h.staff(h.complaintDetail))
	h.mux.HandleFunc("POST /api/v1/admin/complaints/events/{id}/resolve", h.staff(h.resolveComplaints))
```

4. Extend `overview` — after the organizers block (before `writeJSON`):
```go
	if h.deps.Complaints != nil {
		if n, cerr := h.deps.Complaints.OpenEventCount(r.Context()); cerr == nil {
			resp["complaints_open"] = n
		}
	}
```

5. Add the handlers (near the other admin handlers):
```go
func (h *handler) listComplaints(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	groups, err := h.deps.Complaints.ListInbox(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	if groups == nil {
		groups = []complaints.EventReportGroup{}
	}
	writeJSON(w, http.StatusOK, groups)
}

type complaintJSON struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Note       string `json:"note,omitempty"`
	Status     string `json:"status"`
	Resolution string `json:"resolution,omitempty"`
	Reporter   string `json:"reporter_user_id"`
	CreatedAt  string `json:"created_at"`
}

func (h *handler) complaintDetail(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	items, err := h.deps.Complaints.TargetDetail(r.Context(), "event", id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "detail failed")
		return
	}
	out := make([]complaintJSON, 0, len(items))
	for _, c := range items {
		out = append(out, complaintJSON{
			ID: c.ID.String(), Category: c.Category, Note: c.Note, Status: c.Status,
			Resolution: c.Resolution, Reporter: c.ReporterUserID.String(),
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) resolveComplaints(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Action     string `json:"action"`
		Resolution string `json:"resolution"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	err := h.deps.Complaints.Resolve(r.Context(), "event", id, u.UUID, body.Action, body.Resolution)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	case errors.Is(err, complaints.ErrResolutionRequired):
		writeErr(w, http.StatusBadRequest, "Укажите причину")
	case errors.Is(err, complaints.ErrInvalidAction):
		writeErr(w, http.StatusBadRequest, "Некорректное действие")
	case errors.Is(err, moderation.ErrInvalidTransition):
		writeErr(w, http.StatusConflict, "Событие нельзя снять из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "resolve failed")
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/http/admin/ -v`
Expected: PASS (all existing + 5 new tests).

- [ ] **Step 5: Build, vet, lint**

Run: `cd backend && go build ./... && go vet ./... && golangci-lint run ./internal/http/admin/...`
Expected: clean, exit 0.

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/http/admin/handler.go internal/http/admin/handler_test.go
git commit -m "feat(admin): complaints inbox endpoints + complaints_open on overview

GET /admin/complaints (grouped), GET .../events/{id} (detail),
POST .../events/{id}/resolve (takedown|dismiss)."
```

---

## Task 5: Frontend API client

**Files:**
- Modify: `frontend/lib/api.ts`

**Interfaces:**
- Consumes: `API_V1` base, `authHeaders()` (both already in `api.ts`).
- Produces: `ComplaintCategory`, `ComplaintGroup`, `COMPLAINT_CATEGORIES`, `submitComplaint`, `listComplaints`, `resolveComplaints`; `complaints_open?` on `getAdminOverview`'s return type.

- [ ] **Step 1: Add types + the category label map**

In `frontend/lib/api.ts`, near `AdminEvent` (line ~475), add:

```ts
export type ComplaintCategory =
  | "spam"
  | "fraud"
  | "inappropriate"
  | "duplicate"
  | "other";

// Display labels (RU). Used by the report modal and the admin breakdown chips.
export const COMPLAINT_CATEGORIES: { value: ComplaintCategory; label: string }[] = [
  { value: "spam", label: "Спам" },
  { value: "fraud", label: "Мошенничество" },
  { value: "inappropriate", label: "Неуместный контент" },
  { value: "duplicate", label: "Дубликат" },
  { value: "other", label: "Другое" },
];

export interface ComplaintGroup {
  event_id: string;
  event_title: string;
  event_status: string;
  report_count: number;
  categories: Record<string, number>;
  latest_note: string;
  latest_at: string;
}
```

- [ ] **Step 2: Add `complaints_open` to the overview return type**

Modify the `getAdminOverview` return type (line ~501) to add the field:

```ts
export async function getAdminOverview(): Promise<{
  events_total: number;
  events_published: number;
  events_removed: number;
  organizers_pending?: number;
  complaints_open?: number;
}> {
```

- [ ] **Step 3: Add the three API functions**

Append after `reinstateEvent` (or anywhere in the admin/API section):

```ts
export async function submitComplaint(
  eventId: string,
  category: ComplaintCategory,
  note: string,
): Promise<void> {
  const res = await fetch(`${API_V1}/events/${eventId}/complaints`, {
    method: "POST",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ category, note }),
  });
  if (!res.ok) {
    if (res.status === 401) throw new Error("not authenticated");
    throw new Error(`complaint: ${res.status}`);
  }
}

export async function listComplaints(): Promise<ComplaintGroup[]> {
  const res = await fetch(`${API_V1}/admin/complaints`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`complaints: ${res.status}`);
  return res.json();
}

export async function resolveComplaints(
  eventId: string,
  action: "takedown" | "dismiss",
  resolution: string,
): Promise<void> {
  const res = await fetch(`${API_V1}/admin/complaints/events/${eventId}/resolve`, {
    method: "POST",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ action, resolution }),
  });
  if (!res.ok) throw new Error(`resolve: ${res.status}`);
}
```

- [ ] **Step 4: Typecheck**

Run: `cd frontend && pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
cd frontend && git add lib/api.ts
git commit -m "feat(web): complaints API client (submit, inbox, resolve)"
```

---

## Task 6: Report button on event detail

**Files:**
- Create: `frontend/components/ReportButton.tsx`
- Modify: `frontend/components/EventDetailView.tsx`

**Interfaces:**
- Consumes: `submitComplaint`, `COMPLAINT_CATEGORIES`, `ComplaintCategory` from `@/lib/api`; `useAuth` from `@/lib/auth-context`; `LoginModal` from `@/components/AuthButton`; `Button` from `@/components/ui/Button`.
- Produces: `<ReportButton eventId={string} />` (default-exported-free named export).

- [ ] **Step 1: Write `ReportButton.tsx`**

Create `frontend/components/ReportButton.tsx` (mirrors `SignupCTA`'s anon-gating + LoginModal pattern):

```tsx
"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import {
  COMPLAINT_CATEGORIES,
  submitComplaint,
  type ComplaintCategory,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export function ReportButton({ eventId }: { eventId: string }) {
  const { isAuthed } = useAuth();
  const [open, setOpen] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const [category, setCategory] = useState<ComplaintCategory>("spam");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function openModal() {
    if (!isAuthed) {
      setShowLogin(true);
      return;
    }
    setOpen(true);
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      await submitComplaint(eventId, category, note.trim());
      setDone(true);
      setOpen(false);
      setNote("");
    } catch (err) {
      if (err instanceof Error && err.message === "not authenticated") {
        setOpen(false);
        setShowLogin(true);
        return;
      }
      setError("Не удалось отправить жалобу");
    } finally {
      setBusy(false);
    }
  }

  if (done) {
    return (
      <p className="text-[13px] text-label-secondary">Жалоба отправлена. Спасибо.</p>
    );
  }

  return (
    <>
      <button
        type="button"
        onClick={openModal}
        className="text-[13px] text-label-secondary underline-offset-2 hover:underline"
      >
        Пожаловаться
      </button>

      {open ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
          onClick={() => setOpen(false)}
        >
          <div
            className="w-full max-w-md rounded-card bg-bg-secondary p-5 shadow-card"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-3 text-[18px] font-semibold">Пожаловаться на событие</h2>

            <div className="space-y-2">
              {COMPLAINT_CATEGORIES.map((c) => (
                <label key={c.value} className="flex items-center gap-2 text-[15px]">
                  <input
                    type="radio"
                    name="complaint-category"
                    value={c.value}
                    checked={category === c.value}
                    onChange={() => setCategory(c.value)}
                  />
                  {c.label}
                </label>
              ))}
            </div>

            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Комментарий (необязательно)"
              rows={3}
              className="mt-3 w-full rounded-control bg-bg-tertiary p-3 text-[15px]"
            />

            {error ? <p className="mt-2 text-[13px] text-red-500">{error}</p> : null}

            <div className="mt-4 flex justify-end gap-2">
              <Button variant="tinted" onClick={() => setOpen(false)}>
                Отмена
              </Button>
              <Button variant="filled" onClick={submit} disabled={busy}>
                {busy ? "Отправка…" : "Отправить"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {showLogin ? <LoginModal onClose={() => setShowLogin(false)} /> : null}
    </>
  );
}
```

> **Implementer note:** confirm `Button` accepts the `variant` values `"tinted"`/`"filled"` (the moderation page uses `"tinted"`). If the `filled` variant name differs, match whatever `components/ui/Button.tsx` exports — do not invent a variant.

- [ ] **Step 2: Render it in `EventDetailView.tsx`**

In `frontend/components/EventDetailView.tsx`, add the import at the top:
```tsx
import { ReportButton } from "@/components/ReportButton";
```
Then render `<ReportButton eventId={event.id} />` inside the footer area near `<SignupCTA event={event} />` (after the closing of the price/CTA row, around line 123-124), e.g. just below the `<SignupCTA>` block:
```tsx
        <div className="mt-3">
          <ReportButton eventId={event.id} />
        </div>
```

- [ ] **Step 3: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean (no eslint errors; `next build` succeeds).

- [ ] **Step 4: Commit**

```bash
cd frontend && git add components/ReportButton.tsx components/EventDetailView.tsx
git commit -m "feat(web): «Пожаловаться» report modal on event detail"
```

---

## Task 7: Admin complaints inbox page + nav

**Files:**
- Create: `frontend/app/admin/complaints/page.tsx`
- Modify: `frontend/app/admin/page.tsx`

**Interfaces:**
- Consumes: `listComplaints`, `resolveComplaints`, `COMPLAINT_CATEGORIES`, `ComplaintGroup` from `@/lib/api`; `Button`, `cn`.
- Produces: the `/admin/complaints` route; a `complaints_open` stat + link on `/admin`.

- [ ] **Step 1: Write the inbox page**

Create `frontend/app/admin/complaints/page.tsx` (mirrors `app/admin/moderation/events/page.tsx`):

```tsx
"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";

import {
  COMPLAINT_CATEGORIES,
  listComplaints,
  resolveComplaints,
  type ComplaintGroup,
} from "@/lib/api";
import { Button } from "@/components/ui/Button";

const CATEGORY_LABEL = new Map(COMPLAINT_CATEGORIES.map((c) => [c.value, c.label]));

export default function ComplaintsInbox() {
  const [items, setItems] = useState<ComplaintGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  const [pending, setPending] = useState<ComplaintGroup | null>(null); // takedown target
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    listComplaints()
      .then((data) => {
        if (!cancelled) {
          setItems(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setItems([]);
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload() {
    setLoading(true);
    setTick((n) => n + 1);
  }

  async function confirmTakedown() {
    if (!pending || !reason.trim()) return;
    try {
      await resolveComplaints(pending.event_id, "takedown", reason.trim());
      setError("");
      setPending(null);
      setReason("");
      reload();
    } catch {
      setError("Не удалось снять событие");
    }
  }

  async function onDismiss(eventId: string) {
    try {
      await resolveComplaints(eventId, "dismiss", "");
      setError("");
      reload();
    } catch {
      setError("Не удалось отклонить жалобы");
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <Link href="/admin" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ Админ
      </Link>
      <h1 className="mb-6 text-2xl font-semibold">Жалобы</h1>

      {error ? <p className="mb-4 text-[13px] text-red-500">{error}</p> : null}

      {loading ? (
        <p className="text-[15px] text-label-secondary">Загрузка…</p>
      ) : items.length === 0 ? (
        <p className="text-[15px] text-label-secondary">Жалоб нет.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((g) => (
            <li
              key={g.event_id}
              className="flex items-start justify-between gap-4 rounded-card bg-bg-secondary p-4 shadow-card-subtle"
            >
              <div className="min-w-0 flex-1 space-y-1">
                <Link
                  href={`/events/${g.event_id}`}
                  className="text-[16px] font-semibold leading-snug hover:underline"
                >
                  {g.event_title}
                </Link>
                <div className="text-[13px] text-label-secondary">
                  {g.report_count} жалоб · статус: {g.event_status}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(g.categories).map(([cat, n]) => (
                    <span
                      key={cat}
                      className="rounded-full bg-bg-tertiary px-2 py-0.5 text-[12px] text-label-secondary"
                    >
                      {CATEGORY_LABEL.get(cat) ?? cat}: {n}
                    </span>
                  ))}
                </div>
                {g.latest_note ? (
                  <div className="text-[13px] text-label-secondary">«{g.latest_note}»</div>
                ) : null}
              </div>
              <div className="flex shrink-0 flex-col gap-2">
                {g.event_status === "published" ? (
                  <Button
                    variant="tinted"
                    onClick={() => setPending(g)}
                    className="text-red-500 hover:bg-red-500/10"
                  >
                    Снять
                  </Button>
                ) : null}
                <Button variant="tinted" onClick={() => onDismiss(g.event_id)}>
                  Отклонить
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {pending ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => setPending(null)}
        >
          <div
            className="w-full max-w-md rounded-card bg-bg-secondary p-5 shadow-card"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-3 text-[18px] font-semibold">Снять «{pending.event_title}»</h2>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Причина снятия (обязательно)"
              rows={3}
              className="w-full rounded-control bg-bg-tertiary p-3 text-[15px]"
            />
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="tinted" onClick={() => setPending(null)}>
                Отмена
              </Button>
              <Button
                variant="filled"
                onClick={confirmTakedown}
                disabled={!reason.trim()}
                className="text-red-500"
              >
                Снять и закрыть жалобы
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
```

- [ ] **Step 2: Add the stat + link on the admin home**

In `frontend/app/admin/page.tsx`:

1. Extend the `counts` state type to add `complaints_open?: number;` (both the `useState` generic and the inline type).
2. Add a `<Stat>` to the grid:
```tsx
        <Stat label="Открытые жалобы" value={counts?.complaints_open} />
```
3. Add a second action link below the moderation-queue link:
```tsx
        <Link
          href="/admin/complaints"
          className={cn(
            "ml-0 mt-3 inline-flex items-center gap-1.5 rounded-control px-4 py-2.5 sm:ml-3 sm:mt-0",
            "bg-accent/12 text-accent text-[15px] font-semibold",
            "transition hover:bg-accent/20 active:scale-[0.97] motion-reduce:transform-none motion-reduce:transition-none",
          )}
        >
          Открыть жалобы →
        </Link>
```
(The grid already has 4 columns; adding a 5th stat wraps fine — `lg:grid-cols-4` will flow it to the next row.)

- [ ] **Step 3: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add app/admin/complaints/page.tsx app/admin/page.tsx
git commit -m "feat(web): admin complaints inbox + overview stat"
```

---

## Final verification

- [ ] **Backend:** `cd backend && make generate-api && go build ./... && go vet ./... && go test ./... && golangci-lint run`
  Expected: build/vet/test clean; lint exit 0. (Integration tests skip without `TEST_DATABASE_URL`.)
- [ ] **Frontend:** `cd frontend && pnpm lint && pnpm build` — clean.
- [ ] **Manual loop (against a running stack with an admin account):**
  1. As a normal user, open an event → «Пожаловаться» → pick a category + note → submit → success toast.
  2. Submit again on the same event → still success (idempotent, no duplicate in the queue).
  3. As admin, open `/admin/complaints` → the event appears with `report_count` and the category chip.
  4. **Dismiss** → event leaves the inbox; event status unchanged.
  5. File a fresh complaint → **Снять** with a reason → event status becomes `rejected` (verify in `/admin/moderation/events` «Снятые»), inbox row clears, complaints closed.
  6. `/admin` overview shows «Открытые жалобы» reflecting the open count.
- [ ] **Update HANDOFF + roadmap:** mark sub-project #3 done in `docs/superpowers/plans/2026-06-26-admin-suite-roadmap.md` and add a «Recently done» entry to `docs/HANDOFF.md`. (Deploy is a separate step — follow the full-stack deploy runbook; DB advances to migration 017.)

---

## Self-Review notes

- **Spec coverage:** data model (Task 1) ✓; domain service+repo incl. dedup/cascade/audit (Task 2) ✓; public submit authed + idempotent 200/201 (Task 3) ✓; admin inbox grouped + resolve takedown|dismiss + 409 mapping + `complaints_open` (Task 4) ✓; API client (Task 5) ✓; report modal w/ anon-gating (Task 6) ✓; admin inbox UI + nav + stat (Task 7) ✓; testing + audit/compliance covered in-task ✓. `TargetDetail`/`complaintDetail` drill-in endpoint is shipped (Task 4) though the UI drill-in panel is left minimal (row + chips) — acceptable per spec (drill-in is "shows individual reports"; the endpoint exists for it).
- **Idempotent 200 vs 201:** honored by threading a `created bool` through `Insert`→`Submit`→handler (the review point the user flagged).
- **Type consistency:** `Submit` returns `(bool, error)` everywhere (domain, stub, handler); `Resolve(action, resolution)` signature identical across service/handler/stub; `EventReportGroup` JSON tags match the frontend `ComplaintGroup` fields exactly (`event_id`, `event_title`, `event_status`, `report_count`, `categories`, `latest_note`, `latest_at`).
