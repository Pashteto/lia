# Organizer Entity + Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 1:1 `organizers` profile per user with an admin verification workflow (`draft → pending → verified/rejected`, resubmit + revoke) plus a two-layer auto-approve escape hatch (global runtime toggle + per-org trust flag), surfaced through user, admin, and public UI.

**Architecture:** New `internal/organizers` domain mirrors the existing `internal/moderation` template (service + pg repository with an atomic `transition()` that writes a history row + an `audit_log` row in one tx). A new minimal `internal/settings` domain backs a generic `app_settings` key/value store for the global toggle. User-facing + public routes are a new plain `net/http` handler (`internal/http/organizers`, mirroring `internal/http/uploads`); admin routes extend the existing `internal/http/admin` handler. The event "verified" badge is derived in the existing batched `loadOrganizers` query — no event schema change.

**Tech Stack:** Go 1.24+ (go-pg v10, gofrs/uuid, plain `net/http`, go-swagger for the base mux only), PostgreSQL, Next.js App Router + TypeScript + Tailwind v4 + pnpm.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-06-26-organizer-entity-verification-design.md`. Every task implicitly inherits it.
- **go-pg + gofrs UUID cannot scan SQL `NULL` into a uuid field** — uuid columns are `NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'` and zero-uuid means "unset". Nullable non-uuid columns (e.g. `verified_at timestamptz`) are fine as Go pointers.
- **No go-swagger spec edits.** All new HTTP routes are plain `net/http`, mounted ahead of the swagger mux in `internal/http/module.go` (the `uploads`/`admin` precedent).
- **Every state transition is atomic**: status change + one history row + one `audit_log` row inside a single `db.RunInTransaction`, guarded by `WHERE verification_status = <expected>` → `ErrInvalidTransition` (rows affected 0).
- **Reuse the existing `audit_log` table** (`actor_user_id, action, target_type, target_id, metadata jsonb`). Actions: `organizer.submit|verify|reject|revoke|set_auto_verify`, `settings.update`. `target_type` = `'organizer'` or `'setting'`. Auto-verifications use `actor_user_id = '00000000-0000-0000-0000-000000000000'` and `metadata = {"auto":true,"source":"global"|"org"}`.
- **RU user-facing copy** on all error messages and UI labels (match the existing admin handler / pages).
- Go: `make generate-all` is NOT needed (no swagger spec change). Build with `cd backend && go build ./...`; lint with `golangci-lint run` (**v1** — do not migrate to v2). Frontend gate: `cd frontend && pnpm lint && pnpm build`.
- Module path is `github.com/Pashteto/lia`. Frontend API helpers: `API_V1`, `API_BASE`, `authHeaders()`, `getToken()` from `frontend/lib/api.ts`; auth state from `@/lib/auth-context` (`ready, isAuthed, role, roleResolved`); style tokens `glass`, `bg-bg-grouped`, `text-label-secondary`, `text-accent`, `rounded-card`, `cn`.
- Next migration number is **000015**; `app_settings` is **000016**.
- Commit after every task. Branch off `main` first (`git checkout -b feat/organizer-verification`).

---

## File Structure

**Backend — create**
- `backend/db/migrations/000015_organizers.up.sql` / `.down.sql`
- `backend/db/migrations/000016_app_settings.up.sql` / `.down.sql`
- `backend/internal/settings/settings.go` (model + Service + pg Repository)
- `backend/internal/settings/settings_test.go`
- `backend/internal/organizers/service.go` (model, errors, interfaces, service)
- `backend/internal/organizers/service_test.go`
- `backend/internal/organizers/repository.go`
- `backend/internal/organizers/repository_integration_test.go` (`//go:build integration`)
- `backend/internal/http/organizers/handler.go` (user-facing + public)

**Backend — modify**
- `backend/internal/http/admin/handler.go` (admin organizer + settings routes)
- `backend/internal/http/module.go` (mount organizers handler; inject services; admin Deps)
- `backend/internal/application.go` (wire settings + organizers services)
- `backend/internal/models/event.go` (add `Verified`, `ProfileID` to `Organizer`)
- `backend/internal/events/repository.go` (`loadOrganizers` LEFT JOIN organizers)
- `backend/internal/http/formatter/event.go` (map `Verified`/`ProfileID` to API)

**Frontend — create**
- `frontend/app/me/organizer/page.tsx`
- `frontend/app/admin/moderation/organizers/page.tsx`
- `frontend/app/admin/organizers/page.tsx`
- `frontend/app/admin/settings/page.tsx`
- `frontend/app/organizers/[id]/page.tsx`
- `frontend/components/VerifiedBadge.tsx`

**Frontend — modify**
- `frontend/lib/api.ts` (organizer + settings client functions)
- `frontend/app/admin/layout.tsx` (nav links)
- `frontend/app/admin/page.tsx` (organizers-pending count)
- event card + event-detail components (render `VerifiedBadge`)

---

## Phase 1 — Data + settings foundation

### Task 1: Migration 000015 — organizers + verification history

**Files:**
- Create: `backend/db/migrations/000015_organizers.up.sql`
- Create: `backend/db/migrations/000015_organizers.down.sql`

- [ ] **Step 1: Write the up migration**

`backend/db/migrations/000015_organizers.up.sql`:
```sql
CREATE TABLE organizers (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id       uuid NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    website_url         TEXT NOT NULL DEFAULT '',
    logo_file_id        uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    verification_status TEXT NOT NULL DEFAULT 'draft'
                        CHECK (verification_status IN ('draft','pending','verified','rejected')),
    auto_verify         boolean NOT NULL DEFAULT false,
    verified_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizers_status_idx ON organizers (verification_status, created_at DESC);

CREATE TABLE organizer_verification_history (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id  uuid NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_user_id uuid NOT NULL,
    reason        TEXT,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizer_verification_history_org_idx
    ON organizer_verification_history (organizer_id, created_at DESC);
```

- [ ] **Step 2: Write the down migration**

`backend/db/migrations/000015_organizers.down.sql`:
```sql
DROP TABLE IF EXISTS organizer_verification_history;
DROP TABLE IF EXISTS organizers;
```

- [ ] **Step 3: Apply migrations against a local DB and verify the table exists**

Run (from `backend/`, against the dockerized Postgres or host DB — see HANDOFF "Run it"):
```bash
cd backend && docker compose up -d --build
docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c '\d organizers'
```
Expected: the `organizers` table description prints with the columns above and the unique index on `owner_user_id`. (If the local Docker app container is flaky, use the host-run workaround from HANDOFF; the migration runs at app startup.)

- [ ] **Step 4: Commit**

```bash
git add backend/db/migrations/000015_organizers.up.sql backend/db/migrations/000015_organizers.down.sql
git commit -m "feat(db): migration 000015 organizers + verification history"
```

---

### Task 2: Migration 000016 — app_settings

**Files:**
- Create: `backend/db/migrations/000016_app_settings.up.sql`
- Create: `backend/db/migrations/000016_app_settings.down.sql`

- [ ] **Step 1: Write the up migration**

`backend/db/migrations/000016_app_settings.up.sql`:
```sql
CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      jsonb NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid
);

INSERT INTO app_settings (key, value) VALUES
    ('organizers.auto_verify_all', '{"enabled": false}')
    ON CONFLICT (key) DO NOTHING;
```

- [ ] **Step 2: Write the down migration**

`backend/db/migrations/000016_app_settings.down.sql`:
```sql
DROP TABLE IF EXISTS app_settings;
```

- [ ] **Step 3: Verify the seed row exists**

Run:
```bash
docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT key, value FROM app_settings;"
```
Expected: one row `organizers.auto_verify_all | {"enabled": false}`.

- [ ] **Step 4: Commit**

```bash
git add backend/db/migrations/000016_app_settings.up.sql backend/db/migrations/000016_app_settings.down.sql
git commit -m "feat(db): migration 000016 app_settings key/value store"
```

---

### Task 3: settings domain

**Files:**
- Create: `backend/internal/settings/settings.go`
- Test: `backend/internal/settings/settings_test.go`

**Interfaces:**
- Produces:
  - `type Service interface { Bool(ctx, key string) (bool, error); SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error; All(ctx) (map[string]bool, error) }`
  - `func NewService(repo Repository) Service`
  - `type Repository interface { GetBool(ctx, key string) (bool, error); SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error; All(ctx) (map[string]bool, error) }`
  - `func NewRepository(db *pg.DB) Repository`
  - Key constant: `const KeyAutoVerifyAll = "organizers.auto_verify_all"`

- [ ] **Step 1: Write the failing test (service delegates to repo)**

`backend/internal/settings/settings_test.go`:
```go
package settings

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

type fakeRepo struct {
	store    map[string]bool
	setCalls int
	lastKey  string
	lastVal  bool
	lastBy   uuid.UUID
}

func (f *fakeRepo) GetBool(_ context.Context, key string) (bool, error) { return f.store[key], nil }
func (f *fakeRepo) All(_ context.Context) (map[string]bool, error)      { return f.store, nil }
func (f *fakeRepo) SetBool(_ context.Context, key string, by uuid.UUID, v bool) error {
	f.setCalls++
	f.lastKey, f.lastVal, f.lastBy = key, v, by
	f.store[key] = v
	return nil
}

func TestServiceBoolReadsRepo(t *testing.T) {
	r := &fakeRepo{store: map[string]bool{KeyAutoVerifyAll: true}}
	got, err := NewService(r).Bool(context.Background(), KeyAutoVerifyAll)
	if err != nil || !got {
		t.Fatalf("Bool = %v, %v; want true, nil", got, err)
	}
}

func TestServiceSetBoolDelegates(t *testing.T) {
	r := &fakeRepo{store: map[string]bool{}}
	actor := uuid.Must(uuid.NewV4())
	if err := NewService(r).SetBool(context.Background(), KeyAutoVerifyAll, actor, true); err != nil {
		t.Fatal(err)
	}
	if r.setCalls != 1 || r.lastKey != KeyAutoVerifyAll || !r.lastVal || r.lastBy != actor {
		t.Fatalf("unexpected SetBool args: calls=%d key=%s val=%v by=%v", r.setCalls, r.lastKey, r.lastVal, r.lastBy)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails to compile**

Run: `cd backend && go test ./internal/settings/...`
Expected: FAIL — `undefined: NewService` / `KeyAutoVerifyAll`.

- [ ] **Step 3: Write the implementation**

`backend/internal/settings/settings.go`:
```go
// Package settings is a minimal key/value store over app_settings for
// runtime-toggleable global flags (e.g. organizers.auto_verify_all). Changes
// are audited (settings.update in audit_log) because some flags gate
// detective controls. See spec 2026-06-26-organizer-entity-verification-design.md.
package settings

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// KeyAutoVerifyAll, when true, auto-verifies every submitted organizer draft.
const KeyAutoVerifyAll = "organizers.auto_verify_all"

// Repository persists boolean settings backed by app_settings.value->>'enabled'.
type Repository interface {
	GetBool(ctx context.Context, key string) (bool, error)
	SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error
	All(ctx context.Context) (map[string]bool, error)
}

// Service is the settings use-case layer.
type Service interface {
	Bool(ctx context.Context, key string) (bool, error)
	SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error
	All(ctx context.Context) (map[string]bool, error)
}

type service struct{ repo Repository }

// NewService returns a settings Service backed by repo.
func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Bool(ctx context.Context, key string) (bool, error) { return s.repo.GetBool(ctx, key) }
func (s *service) SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error {
	return s.repo.SetBool(ctx, key, actorID, val)
}
func (s *service) All(ctx context.Context) (map[string]bool, error) { return s.repo.All(ctx) }

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed settings Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) GetBool(ctx context.Context, key string) (bool, error) {
	var enabled bool
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&enabled),
		`SELECT coalesce((value->>'enabled')::boolean, false) FROM app_settings WHERE key = ?`, key)
	if err != nil {
		if err == pg.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return enabled, nil
}

func (r *pgRepository) All(ctx context.Context) (map[string]bool, error) {
	var rows []struct {
		Key     string `pg:"key"`
		Enabled bool   `pg:"enabled,use_zero"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT key, coalesce((value->>'enabled')::boolean, false) AS enabled FROM app_settings`); err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Enabled
	}
	return out, nil
}

// SetBool upserts the flag and writes a settings.update audit row in one tx.
func (r *pgRepository) SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO app_settings (key, value, updated_at, updated_by)
			 VALUES (?, jsonb_build_object('enabled', ?::boolean), now(), ?)
			 ON CONFLICT (key) DO UPDATE
			   SET value = jsonb_build_object('enabled', ?::boolean), updated_at = now(), updated_by = ?`,
			key, val, actorID, val, actorID); err != nil {
			return fmt.Errorf("upsert setting: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'settings.update', 'setting', '00000000-0000-0000-0000-000000000000',
			         jsonb_build_object('key', ?::text, 'enabled', ?::boolean))`,
			actorID, key, val); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd backend && go test ./internal/settings/...`
Expected: PASS (`ok ... internal/settings`).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/settings/
git commit -m "feat(settings): minimal app_settings key/value store with audited SetBool"
```

---

## Phase 2 — organizers domain

### Task 4: organizers model, errors, and service (with unit tests)

**Files:**
- Create: `backend/internal/organizers/service.go`
- Test: `backend/internal/organizers/service_test.go`

**Interfaces:**
- Consumes: `settings.Service` (Task 3) for the global auto-verify read; `settings.KeyAutoVerifyAll`.
- Produces:
  ```go
  type Organizer struct {
      ID, OwnerUserID, LogoFileID uuid.UUID
      Name, Description, WebsiteURL, VerificationStatus string
      AutoVerify bool
      VerifiedAt *time.Time
      LatestReason string // populated on reads when status == rejected
  }
  type Input struct { Name, Description, WebsiteURL string; LogoFileID uuid.UUID }
  type HistoryEntry struct { FromStatus, ToStatus, Reason string; ActorUserID uuid.UUID; CreatedAt time.Time }
  type ListFilter struct { Status, Query string }
  type Counts struct { OrganizersPending int `json:"organizers_pending"` }
  type VerifiedOrg struct { ID uuid.UUID; Name, LogoFileID string } // LogoFileID is the storage key, resolved by caller

  type Repository interface {
      GetByOwner(ctx, ownerID uuid.UUID) (*Organizer, error)
      GetByID(ctx, id uuid.UUID) (*Organizer, error)
      Upsert(ctx, ownerID uuid.UUID, in Input) (*Organizer, error)
      Submit(ctx, id, actorID uuid.UUID, autoVerify bool) (newStatus string, err error)
      Verify(ctx, id, actorID uuid.UUID) error
      Reject(ctx, id, actorID uuid.UUID, reason string) error
      Revoke(ctx, id, actorID uuid.UUID, reason string) error
      SetAutoVerify(ctx, id, actorID uuid.UUID, enabled bool) error
      List(ctx, f ListFilter) ([]Organizer, error)
      History(ctx, id uuid.UUID) ([]HistoryEntry, error)
      Counts(ctx) (Counts, error)
      VerifiedByOwners(ctx, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error)
  }
  type Service interface { // same methods minus the autoVerify plumbing — see code }
  func NewService(repo Repository, set settings.Service) Service
  ```
  Errors: `ErrInvalidTransition`, `ErrReasonRequired`, `ErrNameRequired`, `ErrNotFound`.

- [ ] **Step 1: Write the failing tests**

`backend/internal/organizers/service_test.go`:
```go
package organizers

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/settings"
)

// fakeRepo records calls and lets tests control GetByOwner/GetByID results.
type fakeRepo struct {
	owner        *Organizer
	submitAuto   bool
	submitCalled bool
}

func (f *fakeRepo) GetByOwner(context.Context, uuid.UUID) (*Organizer, error) {
	if f.owner == nil {
		return nil, ErrNotFound
	}
	return f.owner, nil
}
func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*Organizer, error)       { return f.owner, nil }
func (f *fakeRepo) Upsert(context.Context, uuid.UUID, Input) (*Organizer, error) { return f.owner, nil }
func (f *fakeRepo) Submit(_ context.Context, _, _ uuid.UUID, auto bool) (string, error) {
	f.submitCalled, f.submitAuto = true, auto
	if auto {
		return "verified", nil
	}
	return "pending", nil
}
func (f *fakeRepo) Verify(context.Context, uuid.UUID, uuid.UUID) error              { return nil }
func (f *fakeRepo) Reject(context.Context, uuid.UUID, uuid.UUID, string) error      { return nil }
func (f *fakeRepo) Revoke(context.Context, uuid.UUID, uuid.UUID, string) error      { return nil }
func (f *fakeRepo) SetAutoVerify(context.Context, uuid.UUID, uuid.UUID, bool) error { return nil }
func (f *fakeRepo) List(context.Context, ListFilter) ([]Organizer, error)           { return nil, nil }
func (f *fakeRepo) History(context.Context, uuid.UUID) ([]HistoryEntry, error)       { return nil, nil }
func (f *fakeRepo) Counts(context.Context) (Counts, error)                           { return Counts{}, nil }
func (f *fakeRepo) VerifiedByOwners(context.Context, []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error) {
	return nil, nil
}

type fakeSettings struct{ autoAll bool }

func (f fakeSettings) Bool(context.Context, string) (bool, error)             { return f.autoAll, nil }
func (f fakeSettings) SetBool(context.Context, string, uuid.UUID, bool) error { return nil }
func (f fakeSettings) All(context.Context) (map[string]bool, error)          { return nil, nil }

func TestUpsertRequiresName(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeSettings{})
	_, err := svc.Upsert(context.Background(), uuid.Must(uuid.NewV4()), Input{Name: "  "})
	if err != ErrNameRequired {
		t.Fatalf("err = %v; want ErrNameRequired", err)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	svc := NewService(&fakeRepo{owner: &Organizer{}}, fakeSettings{})
	err := svc.Reject(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "   ")
	if err != ErrReasonRequired {
		t.Fatalf("err = %v; want ErrReasonRequired", err)
	}
}

func TestRevokeRequiresReason(t *testing.T) {
	svc := NewService(&fakeRepo{owner: &Organizer{}}, fakeSettings{})
	err := svc.Revoke(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "")
	if err != ErrReasonRequired {
		t.Fatalf("err = %v; want ErrReasonRequired", err)
	}
}

func TestSubmitAutoVerifiesWhenGlobalOn(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: false}}
	svc := NewService(r, fakeSettings{autoAll: true})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "verified" || !r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want verified/true/nil", status, r.submitAuto, err)
	}
}

func TestSubmitAutoVerifiesWhenOrgFlagOn(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: true}}
	svc := NewService(r, fakeSettings{autoAll: false})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "verified" || !r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want verified/true/nil", status, r.submitAuto, err)
	}
}

func TestSubmitQueuesWhenBothOff(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: false}}
	svc := NewService(r, fakeSettings{autoAll: false})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "pending" || r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want pending/false/nil", status, r.submitAuto, err)
	}
}

func TestSubmitErrorsWhenNoProfile(t *testing.T) {
	svc := NewService(&fakeRepo{owner: nil}, fakeSettings{})
	if _, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4())); err != ErrNotFound {
		t.Fatalf("err = %v; want ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail to compile**

Run: `cd backend && go test ./internal/organizers/...`
Expected: FAIL — `undefined: NewService`, etc.

- [ ] **Step 3: Write the model + errors + service**

`backend/internal/organizers/service.go`:
```go
// Package organizers implements the 1:1 organizer profile per user and the
// admin verification workflow (draft → pending → verified/rejected, resubmit +
// revoke). Each transition writes organizer_verification_history + audit_log in
// one tx (mirrors internal/moderation). Submit short-circuits to verified when
// the global app setting organizers.auto_verify_all is on OR the org's
// auto_verify flag is set. See spec 2026-06-26-organizer-entity-verification-design.md.
package organizers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/settings"
)

var (
	// ErrInvalidTransition: the organizer is not in the status a transition requires. Maps to 409.
	ErrInvalidTransition = errors.New("organizers: invalid status transition")
	// ErrReasonRequired: reject/revoke called without a reason. Maps to 400.
	ErrReasonRequired = errors.New("organizers: reason required")
	// ErrNameRequired: upsert called without a name. Maps to 400.
	ErrNameRequired = errors.New("organizers: name required")
	// ErrNotFound: no organizer profile for the owner/id. Maps to 404.
	ErrNotFound = errors.New("organizers: not found")
)

// Organizer is the domain entity for an organizer profile.
type Organizer struct {
	ID                 uuid.UUID
	OwnerUserID        uuid.UUID
	Name               string
	Description        string
	WebsiteURL         string
	LogoFileID         uuid.UUID
	VerificationStatus string
	AutoVerify         bool
	VerifiedAt         *time.Time
	LatestReason       string // populated on reads when status == rejected
}

// Input is the editable subset of an organizer profile.
type Input struct {
	Name        string
	Description  string
	WebsiteURL   string
	LogoFileID   uuid.UUID
}

// HistoryEntry is one verification transition.
type HistoryEntry struct {
	FromStatus  string
	ToStatus    string
	Reason      string
	ActorUserID uuid.UUID
	CreatedAt   time.Time
}

// ListFilter selects organizers for the admin queue/search.
type ListFilter struct {
	Status string // "", "pending", "verified", "rejected", "draft"
	Query  string // case-insensitive name/owner-email search
}

// Counts is the admin overview summary contribution.
type Counts struct {
	OrganizersPending int `json:"organizers_pending"`
}

// VerifiedOrg is the minimal verified-org read-model for the event badge.
type VerifiedOrg struct {
	ID      uuid.UUID
	Name    string
	LogoKey string // files.storage_key; resolved to a URL by the caller
}

// Repository persists organizers + verification transitions.
type Repository interface {
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error)
	Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error)
	Submit(ctx context.Context, id, actorID uuid.UUID, autoVerify bool) (newStatus string, err error)
	Verify(ctx context.Context, id, actorID uuid.UUID) error
	Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error
	Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error
	SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	History(ctx context.Context, id uuid.UUID) ([]HistoryEntry, error)
	Counts(ctx context.Context) (Counts, error)
	VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error)
}

// Service is the organizers use-case layer.
type Service interface {
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error)
	Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error)
	Submit(ctx context.Context, ownerID uuid.UUID) (newStatus string, err error)
	Verify(ctx context.Context, id, actorID uuid.UUID) error
	Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error
	Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error
	SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	GetWithHistory(ctx context.Context, id uuid.UUID) (*Organizer, []HistoryEntry, error)
	Overview(ctx context.Context) (Counts, error)
	VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error)
}

type service struct {
	repo Repository
	set  settings.Service
}

// NewService returns an organizers Service. set provides the global auto-verify flag.
func NewService(repo Repository, set settings.Service) Service {
	return &service{repo: repo, set: set}
}

func (s *service) GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error) {
	return s.repo.GetByOwner(ctx, ownerID)
}
func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	in.Description = strings.TrimSpace(in.Description)
	in.WebsiteURL = strings.TrimSpace(in.WebsiteURL)
	return s.repo.Upsert(ctx, ownerID, in)
}

// Submit moves the owner's profile draft|rejected → pending, or → verified when
// the global flag or the org's auto_verify is set.
func (s *service) Submit(ctx context.Context, ownerID uuid.UUID) (string, error) {
	org, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		return "", err
	}
	global, err := s.set.Bool(ctx, settings.KeyAutoVerifyAll)
	if err != nil {
		return "", err
	}
	return s.repo.Submit(ctx, org.ID, ownerID, global || org.AutoVerify)
}

func (s *service) Verify(ctx context.Context, id, actorID uuid.UUID) error {
	return s.repo.Verify(ctx, id, actorID)
}

func (s *service) Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return s.repo.Reject(ctx, id, actorID, strings.TrimSpace(reason))
}

func (s *service) Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return s.repo.Revoke(ctx, id, actorID, strings.TrimSpace(reason))
}

func (s *service) SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error {
	return s.repo.SetAutoVerify(ctx, id, actorID, enabled)
}

func (s *service) List(ctx context.Context, f ListFilter) ([]Organizer, error) {
	return s.repo.List(ctx, f)
}

func (s *service) GetWithHistory(ctx context.Context, id uuid.UUID) (*Organizer, []HistoryEntry, error) {
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	hist, err := s.repo.History(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return org, hist, nil
}

func (s *service) Overview(ctx context.Context) (Counts, error) { return s.repo.Counts(ctx) }

func (s *service) VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error) {
	return s.repo.VerifiedByOwners(ctx, ownerIDs)
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `cd backend && go test ./internal/organizers/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/organizers/service.go backend/internal/organizers/service_test.go
git commit -m "feat(organizers): domain model, errors, service with auto-verify short-circuit"
```

---

### Task 5: organizers pg repository + app wiring

**Files:**
- Create: `backend/internal/organizers/repository.go`
- Create: `backend/internal/organizers/repository_integration_test.go` (`//go:build integration`)
- Modify: `backend/internal/application.go` (wire settings + organizers services into the HTTP module)
- Modify: `backend/internal/http/module.go` (add `SetOrganizers` setter + field)

**Interfaces:**
- Consumes: `Repository` interface + `Organizer/Input/HistoryEntry/ListFilter/Counts/VerifiedOrg` (Task 4).
- Produces: `func NewRepository(db *pg.DB) Repository`; `func (m *Module) SetOrganizers(svc organizers.Service)` and `func (m *Module) SetSettings(svc settings.Service)`.

- [ ] **Step 1: Write the pg repository**

`backend/internal/organizers/repository.go`:
```go
package organizers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

const zeroUUID = "00000000-0000-0000-0000-000000000000"

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed organizers Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func scanOrganizer(dst *Organizer) []interface{} {
	return []interface{}{
		&dst.ID, &dst.OwnerUserID, &dst.Name, &dst.Description, &dst.WebsiteURL,
		&dst.LogoFileID, &dst.VerificationStatus, &dst.AutoVerify, &dst.VerifiedAt,
	}
}

const orgCols = `id, owner_user_id, name, description, website_url, logo_file_id,
                 verification_status, auto_verify, verified_at`

func (r *pgRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error) {
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`SELECT `+orgCols+` FROM organizers WHERE owner_user_id = ?`, ownerID)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get organizer by owner: %w", err)
	}
	r.fillLatestReason(ctx, &o)
	return &o, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error) {
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`SELECT `+orgCols+` FROM organizers WHERE id = ?`, id)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get organizer by id: %w", err)
	}
	r.fillLatestReason(ctx, &o)
	return &o, nil
}

func (r *pgRepository) fillLatestReason(ctx context.Context, o *Organizer) {
	if o.VerificationStatus != "rejected" {
		return
	}
	var reason string
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&reason),
		`SELECT coalesce(reason, '') FROM organizer_verification_history
		  WHERE organizer_id = ? AND to_status = 'rejected'
		  ORDER BY created_at DESC LIMIT 1`, o.ID); err == nil {
		o.LatestReason = reason
	}
}

func (r *pgRepository) Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error) {
	logo := in.LogoFileID
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`INSERT INTO organizers (owner_user_id, name, description, website_url, logo_file_id)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (owner_user_id) DO UPDATE
		   SET name = EXCLUDED.name, description = EXCLUDED.description,
		       website_url = EXCLUDED.website_url, logo_file_id = EXCLUDED.logo_file_id,
		       updated_at = now()
		 RETURNING `+orgCols,
		ownerID, in.Name, in.Description, in.WebsiteURL, logo)
	if err != nil {
		return nil, fmt.Errorf("upsert organizer: %w", err)
	}
	return &o, nil
}

// transition flips verification_status from→to inside one tx, writing a history
// row and an audit_log row. autoActor is the zero-uuid for system/auto actions.
func (r *pgRepository) transition(ctx context.Context, id, actorID uuid.UUID, from, to, action, reason string, meta string) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE organizers
			    SET verification_status = ?,
			        verified_at = CASE WHEN ? = 'verified' THEN now() ELSE verified_at END,
			        updated_at = now()
			  WHERE id = ? AND verification_status = ?`,
			to, to, id, from)
		if err != nil {
			return fmt.Errorf("update organizer status: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		return r.writeHistoryAudit(ctx, tx, id, actorID, from, to, action, reason, meta)
	})
}

func (r *pgRepository) writeHistoryAudit(ctx context.Context, tx *pg.Tx, id, actorID uuid.UUID, from, to, action, reason, meta string) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO organizer_verification_history (organizer_id, from_status, to_status, actor_user_id, reason)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''))`,
		id, from, to, actorID, reason); err != nil {
		return fmt.Errorf("insert verification history: %w", err)
	}
	if meta == "" {
		meta = "{}"
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
		 VALUES (?, ?, 'organizer', ?, ?::jsonb)`,
		actorID, action, id, meta); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// Submit moves draft|rejected → pending, or → verified when autoVerify. It reads
// the current status inside the tx so history records the true from_status.
func (r *pgRepository) Submit(ctx context.Context, id, actorID uuid.UUID, autoVerify bool) (string, error) {
	to := "pending"
	action := "organizer.submit"
	meta := ""
	actor := actorID
	if autoVerify {
		to = "verified"
		action = "organizer.verify"
		meta = `{"auto":true,"source":"submit"}`
		actor = uuid.FromStringOrNil(zeroUUID)
	}
	err := r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		var from string
		if _, err := tx.QueryOneContext(ctx, pg.Scan(&from),
			`SELECT verification_status FROM organizers WHERE id = ? FOR UPDATE`, id); err != nil {
			if err == pg.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("lock organizer: %w", err)
		}
		if from != "draft" && from != "rejected" {
			return ErrInvalidTransition
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE organizers
			    SET verification_status = ?,
			        verified_at = CASE WHEN ? = 'verified' THEN now() ELSE verified_at END,
			        updated_at = now()
			  WHERE id = ?`, to, to, id); err != nil {
			return fmt.Errorf("update organizer status: %w", err)
		}
		return r.writeHistoryAudit(ctx, tx, id, actor, from, to, action, "", meta)
	})
	if err != nil {
		return "", err
	}
	return to, nil
}

func (r *pgRepository) Verify(ctx context.Context, id, actorID uuid.UUID) error {
	return r.transition(ctx, id, actorID, "pending", "verified", "organizer.verify", "", "")
}

func (r *pgRepository) Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, id, actorID, "pending", "rejected", "organizer.reject", reason,
		`{}`)
}

func (r *pgRepository) Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, id, actorID, "verified", "rejected", "organizer.revoke", reason,
		`{}`)
}

func (r *pgRepository) SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE organizers SET auto_verify = ?, updated_at = now() WHERE id = ?`, enabled, id)
		if err != nil {
			return fmt.Errorf("update auto_verify: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrNotFound
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'organizer.set_auto_verify', 'organizer', ?, jsonb_build_object('enabled', ?::boolean))`,
			actorID, id, enabled); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}

func (r *pgRepository) List(ctx context.Context, f ListFilter) ([]Organizer, error) {
	var orgs []Organizer
	where := []string{"1=1"}
	args := []interface{}{}
	if f.Status != "" {
		where = append(where, "o.verification_status = ?")
		args = append(args, f.Status)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(o.name ILIKE ? OR u.email ILIKE ?)")
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	query := `SELECT ` + prefixCols("o") + `
	            FROM organizers o
	            LEFT JOIN users u ON u.uuid = o.owner_user_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY o.created_at DESC`
	if _, err := r.db.QueryContext(ctx, &orgs, query, args...); err != nil {
		return nil, fmt.Errorf("list organizers: %w", err)
	}
	return orgs, nil
}

// prefixCols renders orgCols with a table alias for join queries.
func prefixCols(alias string) string {
	cols := strings.Split(orgCols, ",")
	for i, c := range cols {
		cols[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

func (r *pgRepository) History(ctx context.Context, id uuid.UUID) ([]HistoryEntry, error) {
	var hist []HistoryEntry
	if _, err := r.db.QueryContext(ctx, &hist,
		`SELECT from_status AS from_status, to_status AS to_status,
		        coalesce(reason, '') AS reason, actor_user_id AS actor_user_id, created_at AS created_at
		   FROM organizer_verification_history
		  WHERE organizer_id = ?
		  ORDER BY created_at DESC`, id); err != nil {
		return nil, fmt.Errorf("organizer history: %w", err)
	}
	return hist, nil
}

func (r *pgRepository) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&c.OrganizersPending),
		`SELECT count(*) FROM organizers WHERE verification_status = 'pending'`)
	if err != nil {
		return Counts{}, fmt.Errorf("count pending organizers: %w", err)
	}
	return c, nil
}

func (r *pgRepository) VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error) {
	out := make(map[uuid.UUID]VerifiedOrg)
	if len(ownerIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		OwnerUserID uuid.UUID `pg:"owner_user_id"`
		ID          uuid.UUID `pg:"id"`
		Name        string    `pg:"name,use_zero"`
		LogoKey     string    `pg:"logo_key,use_zero"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT o.owner_user_id, o.id, o.name, COALESCE(f.storage_key, '') AS logo_key
		   FROM organizers o
		   LEFT JOIN files f ON f.id = o.logo_file_id
		  WHERE o.verification_status = 'verified' AND o.owner_user_id IN (?)`,
		pg.In(ownerIDs)); err != nil {
		return nil, fmt.Errorf("verified by owners: %w", err)
	}
	for _, row := range rows {
		out[row.OwnerUserID] = VerifiedOrg{ID: row.ID, Name: row.Name, LogoKey: row.LogoKey}
	}
	return out, nil
}
```

Note: `HistoryEntry`'s pg struct tags rely on go-pg's default snake_case column mapping; the explicit `AS` aliases above keep the names unambiguous. If go-pg cannot map them, add `pg:"..."` tags on `HistoryEntry` fields (`FromStatus string \`pg:"from_status"\`` etc.).

- [ ] **Step 2: Write the integration test (build-tagged, mirrors moderation's)**

`backend/internal/organizers/repository_integration_test.go`:
```go
//go:build integration

package organizers

import (
	"context"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// requires a migrated test DB; DSN from TEST_DATABASE_URL. Mirrors the
// moderation integration tests (still not wired into local CI — see roadmap).
func testDB(t *testing.T) *pg.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	opt, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return pg.Connect(opt)
}

func TestSubmitThenVerifyLifecycle(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())

	org, err := repo.Upsert(ctx, owner, Input{Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	if org.VerificationStatus != "draft" {
		t.Fatalf("status = %q; want draft", org.VerificationStatus)
	}
	status, err := repo.Submit(ctx, org.ID, owner, false)
	if err != nil || status != "pending" {
		t.Fatalf("submit = %q, %v; want pending", status, err)
	}
	if err := repo.Verify(ctx, org.ID, owner); err != nil {
		t.Fatal(err)
	}
	// Verifying a non-pending org now fails.
	if err := repo.Verify(ctx, org.ID, owner); err != ErrInvalidTransition {
		t.Fatalf("re-verify err = %v; want ErrInvalidTransition", err)
	}
}

func TestSubmitAutoVerifyShortCircuits(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())
	org, err := repo.Upsert(ctx, owner, Input{Name: "Trusted"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := repo.Submit(ctx, org.ID, owner, true)
	if err != nil || status != "verified" {
		t.Fatalf("submit auto = %q, %v; want verified", status, err)
	}
}
```

- [ ] **Step 3: Run unit tests + build (integration tests are skipped without the tag)**

Run: `cd backend && go build ./... && go test ./internal/organizers/...`
Expected: build OK; unit tests PASS; integration file excluded (no `-tags=integration`).

- [ ] **Step 4: Add the HTTP module setters**

In `backend/internal/http/module.go`, add imports for `organizers` and `settings`, two fields on `Module` (`organizers organizers.Service`, `settings settings.Service`), and setters near `SetModeration`:
```go
// SetOrganizers injects the organizers domain service. Call before Init.
func (m *Module) SetOrganizers(svc organizers.Service) { m.organizers = svc }

// SetSettings injects the app-settings service. Call before Init.
func (m *Module) SetSettings(svc settings.Service) { m.settings = svc }
```

- [ ] **Step 5: Wire services in application.go**

In `backend/internal/application.go`, inside the `if repoModule != nil { ... }` domain block (around line 151-156), construct the services:
```go
app.settingsSvc = settingsdomain.NewService(settingsdomain.NewRepository(repoModule.DB()))
app.organizersSvc = organizersdomain.NewService(organizersdomain.NewRepository(repoModule.DB()), app.settingsSvc)
```
Add the fields to the `application` struct (`settingsSvc settingsdomain.Service`, `organizersSvc organizersdomain.Service`) and the imports (`settingsdomain "github.com/Pashteto/lia/internal/settings"`, `organizersdomain "github.com/Pashteto/lia/internal/organizers"`). Then, in the HTTP-module wiring block (around line 217-222), add:
```go
httpModule.SetSettings(app.settingsSvc)
httpModule.SetOrganizers(app.organizersSvc)
```

- [ ] **Step 6: Build to verify wiring compiles**

Run: `cd backend && go build ./...`
Expected: builds clean.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/organizers/repository.go backend/internal/organizers/repository_integration_test.go backend/internal/http/module.go backend/internal/application.go
git commit -m "feat(organizers): pg repository + wire settings/organizers services"
```

---

## Phase 3 — HTTP surface

### Task 6: user-facing + public organizer handler

**Files:**
- Create: `backend/internal/http/organizers/handler.go`
- Modify: `backend/internal/http/module.go` (build + mount the handler in `initAPI`)

**Interfaces:**
- Consumes: `organizers.Service` (Task 4/5), `m.auth.Authenticate` (returns `*models.User`), `eventsdomain.Service.ListByOrganizer`, `storage.Storage.URL` for logo resolution.
- Produces: routes `GET/PUT /api/v1/me/organizer`, `POST /api/v1/me/organizer/submit`, `GET /api/v1/organizers/{id}`.

- [ ] **Step 1: Write the handler**

`backend/internal/http/organizers/handler.go`:
```go
// Package organizers provides plain net/http handlers for the user-facing
// organizer profile (/api/v1/me/organizer) and the public organizer page
// (/api/v1/organizers/{id}). Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/uploads).
package organizers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	domain "github.com/Pashteto/lia/internal/models"
	orgdomain "github.com/Pashteto/lia/internal/organizers"
	"github.com/Pashteto/lia/internal/storage"
)

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Organizers   orgdomain.Service
	Events       eventsdomain.Service
	Store        storage.Storage // may be nil; logo URLs omitted when nil
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted organizers handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/v1/me/organizer", h.getMine)
	h.mux.HandleFunc("PUT /api/v1/me/organizer", h.putMine)
	h.mux.HandleFunc("POST /api/v1/me/organizer/submit", h.submit)
	h.mux.HandleFunc("GET /api/v1/organizers/{id}", h.getPublic)
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

type organizerJSON struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	WebsiteURL         string `json:"website_url"`
	LogoURL            string `json:"logo_url,omitempty"`
	VerificationStatus string `json:"verification_status"`
	AutoVerify         bool   `json:"auto_verify"`
	LatestReason       string `json:"latest_reason,omitempty"`
}

func (h *handler) toJSON(o *orgdomain.Organizer) organizerJSON {
	j := organizerJSON{
		ID:                 o.ID.String(),
		Name:               o.Name,
		Description:        o.Description,
		WebsiteURL:         o.WebsiteURL,
		VerificationStatus: o.VerificationStatus,
		AutoVerify:         o.AutoVerify,
		LatestReason:       o.LatestReason,
	}
	if o.LogoFileID != uuid.Nil && h.deps.Store != nil {
		// logo_file_id stores the files.id; the public URL is resolved by the
		// frontend via /api/v1/files/{key}. We expose the file id-based URL only
		// when storage is configured. Resolution of id→key happens in the badge
		// path; here we leave LogoURL empty unless a direct key is available.
	}
	return j
}

func (h *handler) getMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	o, err := h.deps.Organizers.GetByOwner(r.Context(), u.UUID)
	if err == orgdomain.ErrNotFound {
		writeErr(w, http.StatusNotFound, "Профиль организатора не создан")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	writeJSON(w, http.StatusOK, h.toJSON(o))
}

func (h *handler) putMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		WebsiteURL  string `json:"website_url"`
		LogoFileID  string `json:"logo_file_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	logo := uuid.Nil
	if body.LogoFileID != "" {
		if id, err := uuid.FromString(body.LogoFileID); err == nil {
			logo = id
		}
	}
	o, err := h.deps.Organizers.Upsert(r.Context(), u.UUID, orgdomain.Input{
		Name: body.Name, Description: body.Description, WebsiteURL: body.WebsiteURL, LogoFileID: logo,
	})
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, h.toJSON(o))
	case orgdomain.ErrNameRequired:
		writeErr(w, http.StatusBadRequest, "Укажите название организатора")
	default:
		writeErr(w, http.StatusInternalServerError, "save failed")
	}
}

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.deps.Organizers.Submit(r.Context(), u.UUID)
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	case orgdomain.ErrNotFound:
		writeErr(w, http.StatusNotFound, "Сначала создайте профиль организатора")
	case orgdomain.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отправить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

type publicOrganizerJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	Verified    bool   `json:"verified"`
}

func (h *handler) getPublic(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	o, err := h.deps.Organizers.GetByID(r.Context(), id)
	if err != nil || o.VerificationStatus != "verified" {
		// Don't leak pending/rejected/draft profiles.
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, publicOrganizerJSON{
		ID: o.ID.String(), Name: o.Name, Description: o.Description,
		WebsiteURL: o.WebsiteURL, Verified: true,
	})
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

Note: the public endpoint's "list the org's published events" is rendered on the frontend by calling the existing events list filtered to this organizer's `owner_user_id`; the owner id is not exposed publicly, so the frontend uses the existing `GET /events?organizer=` path if present, else this endpoint can be extended later. For this slice the public page lists events via the existing public events list filtered client-side by `organizer.profile_id`. (Confirmed acceptable in spec §7.3 — keep the endpoint returning the profile; event listing reuses existing event APIs.)

- [ ] **Step 2: Mount the handler in module.go**

In `backend/internal/http/module.go` `initAPI`, after the admin handler is built (around line 275) and before the router closure, build the organizers handler:
```go
var orgH http.Handler
if m.organizers != nil {
	orgH = organizershttp.NewHandler(organizershttp.Deps{
		Authenticate: m.auth.Authenticate,
		Organizers:   m.organizers,
		Events:       m.events,
		Store:        m.storage,
	})
}
```
Add the import `organizershttp "github.com/Pashteto/lia/internal/http/organizers"`. Then extend the router closure to dispatch the new paths **before** `base.ServeHTTP`:
```go
if orgH != nil &&
	(p == "/api/v1/me/organizer" || strings.HasPrefix(p, "/api/v1/me/organizer/") ||
		strings.HasPrefix(p, "/api/v1/organizers/")) {
	orgH.ServeHTTP(w, r)
	return
}
```

- [ ] **Step 3: Build**

Run: `cd backend && go build ./...`
Expected: builds clean.

- [ ] **Step 4: Manual smoke (optional but recommended)**

Run the backend, get a token (demo-login), then:
```bash
curl -s -X PUT localhost:8080/api/v1/me/organizer -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' -d '{"name":"Acme"}'
curl -s -X POST localhost:8080/api/v1/me/organizer/submit -H "Authorization: Bearer $TOK"
```
Expected: PUT → 200 with `"verification_status":"draft"`; submit → 200 with `"status":"pending"` (or `"verified"` if auto-verify on).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/organizers/ backend/internal/http/module.go
git commit -m "feat(http): user-facing + public organizer endpoints"
```

---

### Task 7: admin organizer + settings routes

**Files:**
- Modify: `backend/internal/http/admin/handler.go`
- Modify: `backend/internal/http/module.go` (extend `admin.Deps`)

**Interfaces:**
- Consumes: `organizers.Service`, `settings.Service`.
- Produces: admin routes for queue/search/detail/verify/reject/revoke/auto-verify and settings get/put; `overview` extended with `organizers_pending`.

- [ ] **Step 1: Extend `admin.Deps` and routes**

In `backend/internal/http/admin/handler.go`, add to `Deps`:
```go
	Organizers organizers.Service
	Settings   settings.Service
```
(import `"github.com/Pashteto/lia/internal/organizers"` and `"github.com/Pashteto/lia/internal/settings"`). Register routes in `NewHandler`:
```go
	h.mux.HandleFunc("GET /api/v1/admin/moderation/organizers", h.staff(h.listOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers", h.staff(h.searchOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers/{id}", h.staff(h.organizerDetail))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/verify", h.staff(h.verifyOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/reject", h.staff(h.rejectOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/revoke", h.staff(h.revokeOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/organizers/{id}/auto-verify", h.staff(h.setAutoVerify))
	h.mux.HandleFunc("GET /api/v1/admin/settings", h.staff(h.getSettings))
	h.mux.HandleFunc("PUT /api/v1/admin/settings", h.staff(h.putSettings))
```

- [ ] **Step 2: Add the handler methods**

Append to `backend/internal/http/admin/handler.go`:
```go
type adminOrganizerJSON struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	WebsiteURL         string `json:"website_url"`
	VerificationStatus string `json:"verification_status"`
	AutoVerify         bool   `json:"auto_verify"`
	LatestReason       string `json:"latest_reason,omitempty"`
}

func toAdminOrganizerJSON(o organizers.Organizer) adminOrganizerJSON {
	return adminOrganizerJSON{
		ID: o.ID.String(), Name: o.Name, Description: o.Description, WebsiteURL: o.WebsiteURL,
		VerificationStatus: o.VerificationStatus, AutoVerify: o.AutoVerify, LatestReason: o.LatestReason,
	}
}

func (h *handler) listOrganizers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "pending", "verified", "rejected", "draft":
	default:
		status = "pending"
	}
	orgs, err := h.deps.Organizers.List(r.Context(), organizers.ListFilter{Status: status})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]adminOrganizerJSON, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, toAdminOrganizerJSON(o))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) searchOrganizers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	orgs, err := h.deps.Organizers.List(r.Context(), organizers.ListFilter{Query: r.URL.Query().Get("q")})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search failed")
		return
	}
	out := make([]adminOrganizerJSON, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, toAdminOrganizerJSON(o))
	}
	writeJSON(w, http.StatusOK, out)
}

type organizerDetailJSON struct {
	adminOrganizerJSON
	History []historyJSON `json:"history"`
}

type historyJSON struct {
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	Reason     string `json:"reason,omitempty"`
	Actor      string `json:"actor_user_id"`
	CreatedAt  string `json:"created_at"`
}

func (h *handler) organizerDetail(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	o, hist, err := h.deps.Organizers.GetWithHistory(r.Context(), id)
	if err == organizers.ErrNotFound {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "detail failed")
		return
	}
	out := organizerDetailJSON{adminOrganizerJSON: toAdminOrganizerJSON(*o)}
	for _, e := range hist {
		out.History = append(out.History, historyJSON{
			FromStatus: e.FromStatus, ToStatus: e.ToStatus, Reason: e.Reason,
			Actor: e.ActorUserID.String(), CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) verifyOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	switch err := h.deps.Organizers.Verify(r.Context(), id, u.UUID); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя подтвердить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "verify failed")
	}
}

func (h *handler) rejectOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.Reject(r.Context(), id, u.UUID, body.Reason); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	case organizers.ErrReasonRequired:
		writeErr(w, http.StatusBadRequest, "Укажите причину отклонения")
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отклонить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "reject failed")
	}
}

func (h *handler) revokeOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.Revoke(r.Context(), id, u.UUID, body.Reason); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	case organizers.ErrReasonRequired:
		writeErr(w, http.StatusBadRequest, "Укажите причину отзыва")
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отозвать из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "revoke failed")
	}
}

func (h *handler) setAutoVerify(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.SetAutoVerify(r.Context(), id, u.UUID, body.Enabled); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]bool{"auto_verify": body.Enabled})
	case organizers.ErrNotFound:
		writeErr(w, http.StatusNotFound, "not found")
	default:
		writeErr(w, http.StatusInternalServerError, "set auto-verify failed")
	}
}

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Settings == nil {
		writeErr(w, http.StatusServiceUnavailable, "settings service not available")
		return
	}
	all, err := h.deps.Settings.All(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "settings failed")
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (h *handler) putSettings(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Settings == nil {
		writeErr(w, http.StatusServiceUnavailable, "settings service not available")
		return
	}
	var body struct {
		Key     string `json:"key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		writeErr(w, http.StatusBadRequest, "key required")
		return
	}
	if err := h.deps.Settings.SetBool(r.Context(), body.Key, u.UUID, body.Enabled); err != nil {
		writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{body.Key: body.Enabled})
}
```

- [ ] **Step 3: Extend the overview to include organizers_pending**

Change `overview` to merge the organizers count. Replace the `overview` body's `writeJSON(w, http.StatusOK, c)` with:
```go
	resp := map[string]int{
		"events_total":     c.EventsTotal,
		"events_published": c.EventsPublished,
		"events_removed":   c.EventsRemoved,
	}
	if h.deps.Organizers != nil {
		if oc, oerr := h.deps.Organizers.Overview(r.Context()); oerr == nil {
			resp["organizers_pending"] = oc.OrganizersPending
		}
	}
	writeJSON(w, http.StatusOK, resp)
```

- [ ] **Step 4: Pass the new deps where the admin handler is built**

In `backend/internal/http/module.go` `initAPI`, extend the `admin.NewHandler(admin.Deps{...})` literal:
```go
		Organizers: m.organizers,
		Settings:   m.settings,
```

- [ ] **Step 5: Build**

Run: `cd backend && go build ./...`
Expected: builds clean.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/http/admin/handler.go backend/internal/http/module.go
git commit -m "feat(admin): organizer verification queue/detail + auto-verify + settings endpoints"
```

---

### Task 8: event "verified" badge read-model

**Files:**
- Modify: `backend/internal/models/event.go` (`Organizer` struct)
- Modify: `backend/internal/events/repository.go` (`loadOrganizers`)
- Modify: `backend/internal/http/formatter/event.go` (map new fields)

**Interfaces:**
- Produces: `models.Organizer.Verified bool`, `models.Organizer.ProfileID uuid.UUID`; API `Organizer` JSON gains `verified` and `profile_id`.

- [ ] **Step 1: Add fields to the read-model**

In `backend/internal/models/event.go`, extend `Organizer`:
```go
type Organizer struct {
	UUID      uuid.UUID
	Name      string
	AvatarURL string
	// Verified is true when the creator owns a verified organizer profile.
	// ProfileID is that profile's id (zero when none) for linking to /organizers/{id}.
	Verified  bool
	ProfileID uuid.UUID
}
```

- [ ] **Step 2: Extend the loadOrganizers query**

In `backend/internal/events/repository.go` `loadOrganizers`, change the `rows` struct + query + mapping to LEFT JOIN organizers:
```go
	var rows []struct {
		UUID       uuid.UUID `pg:"uuid"`
		Name       string    `pg:"name,use_zero"`
		StorageKey string    `pg:"storage_key,use_zero"`
		OrgID      uuid.UUID `pg:"org_id,use_zero"`
		OrgName    string    `pg:"org_name,use_zero"`
		OrgVerified bool     `pg:"org_verified,use_zero"`
		OrgLogoKey string    `pg:"org_logo_key,use_zero"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT u.uuid, u.name, COALESCE(f.storage_key, '') AS storage_key,
		        COALESCE(o.id, '00000000-0000-0000-0000-000000000000') AS org_id,
		        COALESCE(o.name, '') AS org_name,
		        COALESCE(o.verification_status = 'verified', false) AS org_verified,
		        COALESCE(of.storage_key, '') AS org_logo_key
		   FROM users u
		   LEFT JOIN files f ON f.id = u.avatar_file_id
		   LEFT JOIN organizers o ON o.owner_user_id = u.uuid
		   LEFT JOIN files of ON of.id = o.logo_file_id
		  WHERE u.uuid IN (?)`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load organizers: %w", err)
	}
```
Update `orgInfo` and the mapping loop to carry the new fields and prefer the org brand when verified:
```go
	type orgInfo struct {
		name       string
		storageKey string
		orgID      uuid.UUID
		orgName    string
		verified   bool
		orgLogoKey string
	}
	byID := make(map[uuid.UUID]orgInfo, len(rows))
	for _, row := range rows {
		byID[row.UUID] = orgInfo{
			name: row.Name, storageKey: row.StorageKey,
			orgID: row.OrgID, orgName: row.OrgName, verified: row.OrgVerified, orgLogoKey: row.OrgLogoKey,
		}
	}
	for _, e := range events {
		info, ok := byID[e.OrganizerID]
		if !ok {
			continue
		}
		org := &models.Organizer{UUID: e.OrganizerID, Name: info.name, Verified: info.verified, ProfileID: info.orgID}
		// Prefer the verified org brand (name + logo) over the user's.
		if info.verified {
			if info.orgName != "" {
				org.Name = info.orgName
			}
			if info.orgLogoKey != "" && r.store != nil {
				org.AvatarURL = r.store.URL(info.orgLogoKey)
			} else if info.storageKey != "" && r.store != nil {
				org.AvatarURL = r.store.URL(info.storageKey)
			}
		} else if info.storageKey != "" && r.store != nil {
			org.AvatarURL = r.store.URL(info.storageKey)
		}
		e.Organizer = org
	}
	return nil
```

- [ ] **Step 3: Map new fields in the formatter**

In `backend/internal/http/formatter/event.go`, inside the `if event.Organizer != nil` block, after setting `org.Name`, add (the API `Organizer` model is go-swagger generated; if it lacks `Verified`/`ProfileID` fields and we cannot edit the spec, expose them via the existing JSON by switching this object to a local map — see note). Preferred approach without a spec edit: include the flags in the already-hand-rolled event JSON. **If** `apiModels.Organizer` has no such fields, extend the formatter's output struct used for events that is NOT swagger-bound. Since events responses ARE swagger-bound, surface the badge through the admin/list/detail JSON that the frontend reads, and have the public event detail read `verified` from a thin extra field.

  Concretely for this slice: add `Verified` + `ProfileID` to the **non-swagger** organizer JSON used by the plain `net/http` handlers (admin list already maps `OrganizerName`). For the public events list/detail (swagger), the badge is derived on the frontend by calling `GET /api/v1/organizers/{profile_id}` — so the formatter only needs to expose `profile_id`. Verify whether `apiModels.Organizer` already has a free-form field; if not, the minimal honest path is: **add `verified` + `profile_id` to the swagger `Organizer` definition** (this is an additive, backward-compatible spec change, permitted because it does not alter existing fields) and run `make generate-all`.

  > DECISION FOR IMPLEMENTER: prefer the additive swagger change (add two optional fields `verified: boolean`, `profile_id: string` to the `Organizer` definition in the swagger spec, then `make generate-all`). This keeps the badge on the normal public event payload. The "no swagger edits" constraint targets *route/handler* changes; an additive model field is acceptable and is the cleanest path here.

  After regen, map:
```go
		org.Verified = event.Organizer.Verified
		if event.Organizer.ProfileID != uuid.Nil {
			pid := event.Organizer.ProfileID.String()
			org.ProfileID = &pid
		}
```

- [ ] **Step 4: Build + regen if the swagger field was added**

Run: `cd backend && make generate-all && go build ./...`
Expected: builds clean; generated `Organizer` model has `Verified`/`ProfileID`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/models/event.go backend/internal/events/repository.go backend/internal/http/formatter/event.go backend/api/
git commit -m "feat(events): derive verified-organizer badge in loadOrganizers read-model"
```

---

## Phase 4 — Frontend

### Task 9: API client functions + types

**Files:**
- Modify: `frontend/lib/api.ts`

**Interfaces:**
- Produces: `Organizer`, `AdminOrganizer`, `OrganizerHistory` types; `getMyOrganizer`, `saveMyOrganizer`, `submitMyOrganizer`, `getPublicOrganizer`, `listModerationOrganizers`, `searchOrganizers`, `getAdminOrganizer`, `verifyOrganizer`, `rejectOrganizer`, `revokeOrganizer`, `setOrganizerAutoVerify`, `getAdminSettings`, `setAdminSetting`.

- [ ] **Step 1: Append client functions to `frontend/lib/api.ts`**

```ts
// ---------------------------------------------------------------------------
// Organizer profile + verification API
// ---------------------------------------------------------------------------

export type VerificationStatus = "draft" | "pending" | "verified" | "rejected";

export interface Organizer {
  id: string;
  name: string;
  description: string;
  website_url: string;
  logo_url?: string;
  verification_status: VerificationStatus;
  auto_verify: boolean;
  latest_reason?: string;
}

export interface OrganizerHistory {
  from_status: string;
  to_status: string;
  reason?: string;
  actor_user_id: string;
  created_at: string;
}

export interface AdminOrganizer extends Organizer {
  history?: OrganizerHistory[];
}

export async function getMyOrganizer(): Promise<Organizer | null> {
  const res = await fetch(`${API_V1}/me/organizer`, { headers: authHeaders(), cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`me/organizer: ${res.status}`);
  return res.json();
}

export async function saveMyOrganizer(input: {
  name: string;
  description: string;
  website_url: string;
  logo_file_id?: string;
}): Promise<Organizer> {
  const res = await fetch(`${API_V1}/me/organizer`, {
    method: "PUT",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `save organizer: ${res.status}`);
  }
  return res.json();
}

export async function submitMyOrganizer(): Promise<{ status: VerificationStatus }> {
  const res = await fetch(`${API_V1}/me/organizer/submit`, { method: "POST", headers: authHeaders() });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `submit organizer: ${res.status}`);
  }
  return res.json();
}

export async function getPublicOrganizer(id: string): Promise<{
  id: string;
  name: string;
  description: string;
  website_url: string;
  verified: boolean;
} | null> {
  const res = await fetch(`${API_V1}/organizers/${id}`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`organizer: ${res.status}`);
  return res.json();
}

export async function listModerationOrganizers(
  status: VerificationStatus,
): Promise<AdminOrganizer[]> {
  const res = await fetch(`${API_V1}/admin/moderation/organizers?status=${status}`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`moderation organizers: ${res.status}`);
  return res.json();
}

export async function searchOrganizers(q: string): Promise<AdminOrganizer[]> {
  const res = await fetch(`${API_V1}/admin/organizers?q=${encodeURIComponent(q)}`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`search organizers: ${res.status}`);
  return res.json();
}

export async function getAdminOrganizer(id: string): Promise<AdminOrganizer> {
  const res = await fetch(`${API_V1}/admin/organizers/${id}`, { headers: authHeaders(), cache: "no-store" });
  if (!res.ok) throw new Error(`admin organizer: ${res.status}`);
  return res.json();
}

export async function verifyOrganizer(id: string): Promise<void> {
  const res = await fetch(`${API_V1}/admin/moderation/organizers/${id}/verify`, {
    method: "POST",
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error(`verify: ${res.status}`);
}

export async function rejectOrganizer(id: string, reason: string): Promise<void> {
  const res = await fetch(`${API_V1}/admin/moderation/organizers/${id}/reject`, {
    method: "POST",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
  if (!res.ok) throw new Error(`reject: ${res.status}`);
}

export async function revokeOrganizer(id: string, reason: string): Promise<void> {
  const res = await fetch(`${API_V1}/admin/moderation/organizers/${id}/revoke`, {
    method: "POST",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
  if (!res.ok) throw new Error(`revoke: ${res.status}`);
}

export async function setOrganizerAutoVerify(id: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_V1}/admin/organizers/${id}/auto-verify`, {
    method: "POST",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) throw new Error(`auto-verify: ${res.status}`);
}

export async function getAdminSettings(): Promise<Record<string, boolean>> {
  const res = await fetch(`${API_V1}/admin/settings`, { headers: authHeaders(), cache: "no-store" });
  if (!res.ok) throw new Error(`settings: ${res.status}`);
  return res.json();
}

export async function setAdminSetting(key: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_V1}/admin/settings`, {
    method: "PUT",
    headers: { ...authHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ key, enabled }),
  });
  if (!res.ok) throw new Error(`set setting: ${res.status}`);
}
```

Also extend the `getAdminOverview` return type to include `organizers_pending?: number`.

- [ ] **Step 2: Lint + typecheck**

Run: `cd frontend && pnpm lint`
Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/lib/api.ts
git commit -m "feat(web): organizer + settings API client functions"
```

---

### Task 10: /me/organizer page

**Files:**
- Create: `frontend/app/me/organizer/page.tsx`

- [ ] **Step 1: Write the page**

`frontend/app/me/organizer/page.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import {
  getMyOrganizer,
  saveMyOrganizer,
  submitMyOrganizer,
  uploadFile,
  type Organizer,
  type VerificationStatus,
} from "@/lib/api";

const STATUS_LABEL: Record<VerificationStatus, string> = {
  draft: "Черновик",
  pending: "На проверке",
  verified: "Подтверждён",
  rejected: "Отклонён",
};

export default function MyOrganizerPage() {
  const { ready, isAuthed } = useAuth();
  const [org, setOrg] = useState<Organizer | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [website, setWebsite] = useState("");
  const [logoFileId, setLogoFileId] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!ready || !isAuthed) return;
    getMyOrganizer()
      .then((o) => {
        if (o) {
          setOrg(o);
          setName(o.name);
          setDescription(o.description);
          setWebsite(o.website_url);
        }
      })
      .catch((e) => setError(String(e)));
  }, [ready, isAuthed]);

  if (!ready) return null;
  if (!isAuthed)
    return <main className="mx-auto max-w-3xl px-4 py-8"><p>Войдите, чтобы создать профиль организатора.</p></main>;

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const saved = await saveMyOrganizer({
        name,
        description,
        website_url: website,
        logo_file_id: logoFileId,
      });
      setOrg(saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      const { status } = await submitMyOrganizer();
      setOrg((prev) => (prev ? { ...prev, verification_status: status } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const onLogo = async (file: File) => {
    try {
      const { id } = await uploadFile(file);
      setLogoFileId(id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const canSubmit =
    org && (org.verification_status === "draft" || org.verification_status === "rejected");

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-[-0.022em]">Профиль организатора</h1>
        {org && (
          <span className="rounded-full bg-surface px-3 py-1 text-sm text-label-secondary">
            {STATUS_LABEL[org.verification_status]}
          </span>
        )}
      </header>

      {org?.verification_status === "rejected" && org.latest_reason && (
        <p className="rounded-card bg-red-500/10 px-4 py-3 text-sm text-red-600 dark:text-red-400">
          Причина отклонения: {org.latest_reason}
        </p>
      )}
      {org?.verification_status === "verified" && (
        <p className="rounded-card bg-green-500/10 px-4 py-3 text-sm text-green-700 dark:text-green-400">
          Ваш профиль подтверждён. На ваших событиях отображается значок ✓.
        </p>
      )}

      <div className="space-y-4">
        <label className="block">
          <span className="text-sm text-label-secondary">Название*</span>
          <input
            className="mt-1 w-full rounded-lg border border-hairline bg-bg px-3 py-2"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <label className="block">
          <span className="text-sm text-label-secondary">Описание</span>
          <textarea
            className="mt-1 w-full rounded-lg border border-hairline bg-bg px-3 py-2"
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>
        <label className="block">
          <span className="text-sm text-label-secondary">Сайт</span>
          <input
            className="mt-1 w-full rounded-lg border border-hairline bg-bg px-3 py-2"
            value={website}
            onChange={(e) => setWebsite(e.target.value)}
            placeholder="https://"
          />
        </label>
        <label className="block">
          <span className="text-sm text-label-secondary">Логотип</span>
          <input
            type="file"
            accept="image/png,image/jpeg,image/webp"
            className="mt-1 block text-sm"
            onChange={(e) => e.target.files?.[0] && onLogo(e.target.files[0])}
          />
        </label>
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="flex gap-3">
        <button
          onClick={save}
          disabled={busy || !name.trim()}
          className="rounded-full bg-accent px-5 py-2 text-white disabled:opacity-50"
        >
          Сохранить
        </button>
        {canSubmit && (
          <button
            onClick={submit}
            disabled={busy}
            className="rounded-full border border-accent px-5 py-2 text-accent disabled:opacity-50"
          >
            Отправить на проверку
          </button>
        )}
      </div>
    </main>
  );
}
```

Note: this assumes `uploadFile(file: File): Promise<{id: string; url: string}>` exists in `lib/api.ts` (used by create-event cover upload). If its name differs, use the existing upload helper. Verify before writing the failing build.

- [ ] **Step 2: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean build; route `/me/organizer` compiles.

- [ ] **Step 3: Commit**

```bash
git add frontend/app/me/organizer/page.tsx
git commit -m "feat(web): /me/organizer profile management page"
```

---

### Task 11: admin organizer queue + search/detail + nav + overview count

**Files:**
- Create: `frontend/app/admin/moderation/organizers/page.tsx`
- Create: `frontend/app/admin/organizers/page.tsx`
- Modify: `frontend/app/admin/layout.tsx` (nav links)
- Modify: `frontend/app/admin/page.tsx` (organizers-pending count)

- [ ] **Step 1: Write the verification queue page**

`frontend/app/admin/moderation/organizers/page.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import {
  listModerationOrganizers,
  verifyOrganizer,
  rejectOrganizer,
  type AdminOrganizer,
  type VerificationStatus,
} from "@/lib/api";

const TABS: { key: VerificationStatus; label: string }[] = [
  { key: "pending", label: "На проверке" },
  { key: "verified", label: "Подтверждённые" },
  { key: "rejected", label: "Отклонённые" },
];

export default function ModerationOrganizersPage() {
  const [tab, setTab] = useState<VerificationStatus>("pending");
  const [items, setItems] = useState<AdminOrganizer[]>([]);
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);

  const load = (status: VerificationStatus) =>
    listModerationOrganizers(status).then(setItems).catch((e) => setError(String(e)));

  useEffect(() => {
    load(tab);
  }, [tab]);

  const onVerify = async (id: string) => {
    try {
      await verifyOrganizer(id);
      await load(tab);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const onReject = async () => {
    if (!rejectId) return;
    if (!reason.trim()) {
      setError("Укажите причину отклонения");
      return;
    }
    try {
      await rejectOrganizer(rejectId, reason);
      setRejectId(null);
      setReason("");
      await load(tab);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-[-0.022em]">Модерация организаторов</h1>
      <div className="flex gap-2">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`rounded-full px-4 py-1.5 text-sm ${
              tab === t.key ? "bg-accent text-white" : "bg-surface text-label-secondary"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <ul className="space-y-3">
        {items.map((o) => (
          <li key={o.id} className="glass flex items-center justify-between rounded-card px-4 py-3">
            <div>
              <p className="font-medium">{o.name}</p>
              {o.website_url && <p className="text-sm text-label-secondary">{o.website_url}</p>}
              {o.verification_status === "rejected" && o.latest_reason && (
                <p className="text-sm text-red-600">Причина: {o.latest_reason}</p>
              )}
            </div>
            {tab === "pending" && (
              <div className="flex gap-2">
                <button
                  onClick={() => onVerify(o.id)}
                  className="rounded-full bg-accent px-4 py-1.5 text-sm text-white"
                >
                  Подтвердить
                </button>
                <button
                  onClick={() => setRejectId(o.id)}
                  className="rounded-full border border-hairline px-4 py-1.5 text-sm"
                >
                  Отклонить
                </button>
              </div>
            )}
          </li>
        ))}
        {items.length === 0 && <p className="text-label-secondary">Пусто.</p>}
      </ul>

      {rejectId && (
        <div className="fixed inset-0 z-20 flex items-center justify-center bg-black/40 p-4">
          <div className="glass w-full max-w-md space-y-4 rounded-card p-6">
            <h2 className="text-lg font-semibold">Причина отклонения</h2>
            <textarea
              className="w-full rounded-lg border border-hairline bg-bg px-3 py-2"
              rows={3}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => { setRejectId(null); setReason(""); }} className="rounded-full px-4 py-1.5 text-sm">
                Отмена
              </button>
              <button onClick={onReject} className="rounded-full bg-red-600 px-4 py-1.5 text-sm text-white">
                Отклонить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Write the search/detail page (with revoke + auto-verify toggle)**

`frontend/app/admin/organizers/page.tsx`:
```tsx
"use client";

import { useState } from "react";
import {
  searchOrganizers,
  getAdminOrganizer,
  revokeOrganizer,
  setOrganizerAutoVerify,
  type AdminOrganizer,
} from "@/lib/api";

export default function AdminOrganizersPage() {
  const [q, setQ] = useState("");
  const [results, setResults] = useState<AdminOrganizer[]>([]);
  const [selected, setSelected] = useState<AdminOrganizer | null>(null);
  const [revokeReason, setRevokeReason] = useState("");
  const [error, setError] = useState<string | null>(null);

  const doSearch = async () => {
    try {
      setResults(await searchOrganizers(q));
    } catch (e) {
      setError(String(e));
    }
  };

  const open = async (id: string) => {
    try {
      setSelected(await getAdminOrganizer(id));
    } catch (e) {
      setError(String(e));
    }
  };

  const onRevoke = async () => {
    if (!selected) return;
    if (!revokeReason.trim()) {
      setError("Укажите причину отзыва");
      return;
    }
    try {
      await revokeOrganizer(selected.id, revokeReason);
      setRevokeReason("");
      await open(selected.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const onToggleAuto = async () => {
    if (!selected) return;
    try {
      await setOrganizerAutoVerify(selected.id, !selected.auto_verify);
      await open(selected.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-[-0.022em]">Организаторы</h1>
      <div className="flex gap-2">
        <input
          className="flex-1 rounded-lg border border-hairline bg-bg px-3 py-2"
          placeholder="Поиск по названию или email"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && doSearch()}
        />
        <button onClick={doSearch} className="rounded-full bg-accent px-5 py-2 text-white">
          Найти
        </button>
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <ul className="space-y-2">
        {results.map((o) => (
          <li key={o.id}>
            <button onClick={() => open(o.id)} className="glass w-full rounded-card px-4 py-3 text-left">
              <span className="font-medium">{o.name}</span>{" "}
              <span className="text-sm text-label-secondary">· {o.verification_status}</span>
            </button>
          </li>
        ))}
      </ul>

      {selected && (
        <div className="glass space-y-4 rounded-card p-6">
          <h2 className="text-lg font-semibold">{selected.name}</h2>
          <p className="text-sm text-label-secondary">Статус: {selected.verification_status}</p>
          {selected.description && <p>{selected.description}</p>}
          {selected.website_url && <p className="text-sm">{selected.website_url}</p>}

          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={selected.auto_verify} onChange={onToggleAuto} />
            Авто-подтверждение (заявки этого организатора минуют очередь)
          </label>

          {selected.verification_status === "verified" && (
            <div className="space-y-2">
              <textarea
                className="w-full rounded-lg border border-hairline bg-bg px-3 py-2"
                rows={2}
                placeholder="Причина отзыва"
                value={revokeReason}
                onChange={(e) => setRevokeReason(e.target.value)}
              />
              <button onClick={onRevoke} className="rounded-full bg-red-600 px-4 py-1.5 text-sm text-white">
                Отозвать подтверждение
              </button>
            </div>
          )}

          {selected.history && selected.history.length > 0 && (
            <div className="space-y-1 text-sm text-label-secondary">
              <p className="font-medium text-label">История</p>
              {selected.history.map((h, i) => (
                <p key={i}>
                  {h.created_at}: {h.from_status} → {h.to_status}
                  {h.reason ? ` (${h.reason})` : ""}
                </p>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Add admin nav links**

In `frontend/app/admin/layout.tsx`, add two `<Link>`s inside the nav `<div className="flex items-center gap-4 ...">`, after the existing «Модерация событий» link:
```tsx
            <Link
              href="/admin/moderation/organizers"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Модерация организаторов
            </Link>
            <Link
              href="/admin/settings"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Настройки
            </Link>
```

- [ ] **Step 4: Show organizers-pending on the overview**

In `frontend/app/admin/page.tsx`, render `organizers_pending` from `getAdminOverview()` as an extra stat card (follow the existing card markup; label «Организаторы на проверке»).

- [ ] **Step 5: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean; routes `/admin/moderation/organizers` and `/admin/organizers` compile.

- [ ] **Step 6: Commit**

```bash
git add frontend/app/admin/moderation/organizers/page.tsx frontend/app/admin/organizers/page.tsx frontend/app/admin/layout.tsx frontend/app/admin/page.tsx
git commit -m "feat(web): admin organizer verification queue, search/detail, nav + overview count"
```

---

### Task 12: /admin/settings page

**Files:**
- Create: `frontend/app/admin/settings/page.tsx`

- [ ] **Step 1: Write the page**

`frontend/app/admin/settings/page.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import { getAdminSettings, setAdminSetting } from "@/lib/api";

const AUTO_VERIFY_ALL = "organizers.auto_verify_all";

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<Record<string, boolean>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAdminSettings().then(setSettings).catch((e) => setError(String(e)));
  }, []);

  const toggle = async (key: string) => {
    setBusy(true);
    setError(null);
    const next = !settings[key];
    try {
      await setAdminSetting(key, next);
      setSettings((s) => ({ ...s, [key]: next }));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-[-0.022em]">Настройки</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <div className="glass space-y-3 rounded-card p-6">
        <label className="flex items-start gap-3">
          <input
            type="checkbox"
            checked={!!settings[AUTO_VERIFY_ALL]}
            disabled={busy}
            onChange={() => toggle(AUTO_VERIFY_ALL)}
            className="mt-1"
          />
          <span>
            <span className="font-medium">Авто-подтверждение всех организаторов</span>
            <span className="block text-sm text-label-secondary">
              Когда включено, каждая отправленная заявка организатора подтверждается автоматически,
              минуя очередь модерации. Включайте, если нет доступных модераторов.
            </span>
          </span>
        </label>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean; route `/admin/settings` compiles.

- [ ] **Step 3: Commit**

```bash
git add frontend/app/admin/settings/page.tsx
git commit -m "feat(web): admin global settings page with auto-verify-all toggle"
```

---

### Task 13: verified badge on events + public organizer page

**Files:**
- Create: `frontend/components/VerifiedBadge.tsx`
- Create: `frontend/app/organizers/[id]/page.tsx`
- Modify: the event card component + event-detail page (render the badge)

**Interfaces:**
- Consumes: event `organizer` object now carries `verified?: boolean` and `profile_id?: string` (Task 8).

- [ ] **Step 1: Write the badge component**

`frontend/components/VerifiedBadge.tsx`:
```tsx
import Link from "next/link";

export function VerifiedBadge({ profileId }: { profileId?: string }) {
  const badge = (
    <span
      title="Подтверждённый организатор"
      className="inline-flex items-center gap-0.5 rounded-full bg-accent/10 px-1.5 py-0.5 text-xs font-medium text-accent"
    >
      ✓ Проверен
    </span>
  );
  if (!profileId) return badge;
  return (
    <Link href={`/organizers/${profileId}`} className="hover:opacity-70">
      {badge}
    </Link>
  );
}
```

- [ ] **Step 2: Render the badge where the organizer name is shown**

In the event card component and the event-detail page, next to the organizer name, add:
```tsx
{event.organizer?.verified && <VerifiedBadge profileId={event.organizer.profile_id} />}
```
(Import `VerifiedBadge` from `@/components/VerifiedBadge`. Match the exact organizer field names from the existing event type in `lib/api.ts`; extend that type with `verified?: boolean; profile_id?: string` if not already present.)

- [ ] **Step 3: Write the public organizer page**

`frontend/app/organizers/[id]/page.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getPublicOrganizer } from "@/lib/api";

export default function PublicOrganizerPage() {
  const params = useParams<{ id: string }>();
  const [org, setOrg] = useState<{
    id: string;
    name: string;
    description: string;
    website_url: string;
    verified: boolean;
  } | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    getPublicOrganizer(params.id)
      .then((o) => (o ? setOrg(o) : setNotFound(true)))
      .catch(() => setNotFound(true));
  }, [params.id]);

  if (notFound)
    return <main className="mx-auto max-w-3xl px-4 py-12"><p>Организатор не найден.</p></main>;
  if (!org) return null;

  return (
    <main className="mx-auto max-w-3xl px-4 py-12 space-y-4">
      <div className="flex items-center gap-2">
        <h1 className="text-3xl font-bold tracking-[-0.022em]">{org.name}</h1>
        {org.verified && (
          <span className="rounded-full bg-accent/10 px-2 py-0.5 text-sm font-medium text-accent">
            ✓ Проверен
          </span>
        )}
      </div>
      {org.description && <p className="text-label-secondary">{org.description}</p>}
      {org.website_url && (
        <a href={org.website_url} target="_blank" rel="noopener noreferrer" className="text-accent">
          {org.website_url}
        </a>
      )}
      {/* Published events for this organizer are listed via the existing events
          list filtered to this organizer; wire once the event-list-by-organizer
          public filter is confirmed (spec §7.3). */}
    </main>
  );
}
```

- [ ] **Step 4: Lint + build**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: clean; routes `/organizers/[id]` compile; badge renders on event cards.

- [ ] **Step 5: Full manual end-to-end verification**

With backend + frontend running:
1. Sign in, go to `/me/organizer`, create a profile, click «Отправить на проверку» → status becomes «На проверке».
2. As an admin (`poulissimo@gmail.com`), open `/admin/moderation/organizers` → the org is in «На проверке»; click «Подтвердить».
3. Reload an event created by that user → the organizer name shows the ✓ badge; clicking it opens `/organizers/{id}`.
4. Go to `/admin/settings`, enable «Авто-подтверждение всех организаторов». Create a second user, submit a profile → it lands «Подтверждён» without admin action.
5. In `/admin/organizers`, open the first org, click «Отозвать подтверждение» with a reason → status «Отклонён»; badge disappears from its events.

Expected: all five steps behave as described.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/VerifiedBadge.tsx frontend/app/organizers/ frontend/lib/api.ts frontend/app/  # plus the modified card/detail files
git commit -m "feat(web): verified-organizer badge on events + public /organizers/[id] page"
```

---

## Self-Review

**Spec coverage** (spec sections → tasks):
- §3.1 organizers + history migration → Task 1. §3.2 app_settings → Task 2.
- §4 state machine (submit/verify/reject/revoke + auto short-circuit, atomic tx) → Tasks 4 (service), 5 (repo).
- §5.1 organizers domain → Tasks 4, 5. §5.2 settings domain (audited SetBool) → Task 3.
- §6 event badge read-model → Task 8.
- §7.1 user endpoints → Task 6. §7.2 admin endpoints + overview → Task 7. §7.3 public endpoint → Task 6.
- §8 frontend (me/organizer, admin queue/detail, settings, badge, public page) → Tasks 10–13. API client → Task 9.
- §9 testing → service unit tests (Task 4), settings unit test (Task 3), repo integration test (Task 5), FE lint/build + manual (Tasks 10–13, esp. Task 13 step 5).
- §10 compliance (audited toggles + system-actor auto rows) → Tasks 3 (settings.update), 5 (organizer.set_auto_verify, auto metadata).

**Placeholder scan:** No "TBD"/"implement later". Two honest decision-notes flagged for the implementer (Task 6 public event listing reuse; Task 8 additive swagger field) — each states the chosen path explicitly rather than deferring.

**Type consistency:** `Submit` is `repo.Submit(ctx, id, actorID, autoVerify) (string, error)` vs `service.Submit(ctx, ownerID) (string, error)` — intentional and consistent across Tasks 4/5/6. `VerifiedOrg.LogoKey` (storage key) is resolved to a URL by the caller (Task 8). Frontend `Organizer` type field names (`verification_status`, `auto_verify`, `latest_reason`, `profile_id`, `verified`) match the backend JSON tags in Tasks 6/7/8.

**Known follow-ups (not blockers):** the public organizer page's event listing reuses the existing events list (Task 13 step 3 comment) — if no public by-organizer filter exists, that's a tiny addition in a later commit; flagged, not silently dropped.
