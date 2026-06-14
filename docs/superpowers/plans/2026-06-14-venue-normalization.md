# Venue Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the denormalized `events.venue_name` / `events.venue_metro` text columns with a dedicated `venues` entity that events reference by `venue_id`, with a pick-or-create searchable typeahead, end-to-end (backend + go-swagger API + Next.js frontend).

**Architecture:** New `internal/venues` Go module (model → repository → service → HTTP) owns the venue entity, a `GET /venues?q=` search, and a `POST /venues` find-or-create. Events keep their existing loose `venue_id uuid NOT NULL DEFAULT <zero>` reference (no DB FK — zero UUID = "no venue", consistent with `organizer_id`); the events service validates a non-zero `venue_id` via the venues service, and the events repository loads each event's venue (batched, no N+1). The frontend create form gets a debounced typeahead that searches venues and creates new ones inline.

**Tech Stack:** Go modular monolith (go-pg, go-swagger, golang-migrate, PostgreSQL); Next.js App Router + TypeScript + Tailwind v4 + TanStack Query + React Hook Form + Zod.

**Spec:** `docs/superpowers/specs/2026-06-14-venue-normalization-design.md`

**Branch:** `feat/venue-normalization` (already checked out).

---

## Conventions for this plan

- Backend commands run from `backend/`. Frontend commands from `frontend/`.
- **golangci-lint is v1.** The installed binary may be v2 (fails on the v1 config). To lint as CI does: `GOBIN=/tmp/glci-v1 go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8` then `/tmp/glci-v1/golangci-lint run ./...`. Do NOT migrate `.golangci.yml` to v2. Watch for `fieldalignment` (govet enable-all): order pointer/slice/string/interface fields before fixed-size fields like `uuid.UUID`/`time.Time` in struct *definitions* (composite literals are exempt).
- Generated code under `internal/http/server/` + `internal/http/models/` is gitignored; regenerate with `make generate-api` after editing `api/swagger.yaml`. Never hand-edit generated files. (gopls/IDE diagnostics may show generated symbols as "undefined" right after regen — trust `go build`, not the IDE.)
- Docker + Postgres are available; the DB is `postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable`. `migrate` CLI is installed. If the app container is flaky, run the binary on the host against the containerized Postgres.
- Commit after each task with the trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

## File Structure

**Backend — create:**
- `db/migrations/000008_venues_table.up.sql` / `.down.sql` — venues table + backfill + drop denormalized columns.
- `internal/models/venue.go` — `Venue` domain model.
- `internal/venues/repository.go` — go-pg repo (`Search`, `GetByID`, `GetByIDs`, `FindOrCreateByName`).
- `internal/venues/service.go` — `Service` (`Search`, `GetByID`, `Create`, `Validate`) + errors.
- `internal/venues/service_test.go` — service unit tests.
- `internal/http/handlers/venues.go` — `ListVenues` + `CreateVenue` handlers.

**Backend — modify:**
- `internal/models/event.go` — replace `VenueName`/`VenueMetro` with `Venue *Venue` (`pg:"-"`); keep `VenueID`.
- `internal/events/service.go` — add `VenueValidator`; validate `venue_id` in `Create`; 3-arg `NewService`.
- `internal/events/service_test.go` — add venue mock; update `NewService` calls.
- `internal/events/repository.go` — load venue in `GetByID`/`List` (batched).
- `internal/http/formatter/event.go` — `VenueToAPI`; map `event.Venue`; drop `VenueName`/`VenueMetro`.
- `internal/http/module.go` — register `ListVenues`/`CreateVenue`; `SetVenuesService`.
- `internal/application.go` — construct venues service; inject into events + HTTP.
- `api/swagger.yaml` — `Venue`/`VenueInput` defs; `Event.venue`; drop `venue_name`/`venue_metro`; `/venues` paths.

**Frontend — modify:**
- `lib/types.ts` — `Venue` += `district?`; `ApiEvent` nested `venue`; `CreateEventInput` `venue_id`.
- `lib/api.ts` — map `event.venue`; `searchVenues`, `createVenue`.
- `lib/mock-events.ts` — align venue shape.
- `components/VenuePicker.tsx` — **new** debounced search + create typeahead.
- `components/CreateEventForm.tsx` — use `VenuePicker`, submit `venue_id`.

---

## Task 1: Migration — venues table + backfill + drop denormalized columns

**Files:**
- Create: `db/migrations/000008_venues_table.up.sql`
- Create: `db/migrations/000008_venues_table.down.sql`

- [ ] **Step 1: Write the up migration**

`db/migrations/000008_venues_table.up.sql`:

```sql
-- Dedicated venue entity. Replaces the denormalized events.venue_name /
-- events.venue_metro columns (migration 000005). Identity only — coordinates
-- and PostGIS "events nearby" are a separate later spec.
CREATE TABLE IF NOT EXISTS venues
(
    id          uuid NOT NULL
        CONSTRAINT venue_id_pkey PRIMARY KEY,
    name        text NOT NULL,
    address     text NOT NULL DEFAULT '',
    metro       text NOT NULL DEFAULT '',
    district    text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- case-insensitive search + find-or-create lookup
CREATE INDEX IF NOT EXISTS venue_name_lower_idx
    ON venues (lower(name));

-- reuse update_updated_at_column() from migration 000001
CREATE TRIGGER update_venue_updated_at
    BEFORE UPDATE
    ON venues
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

-- Backfill: one venue per distinct normalized venue_name, carrying its metro.
-- gen_random_uuid() is built into PostgreSQL 13+.
INSERT INTO venues (id, name, metro)
SELECT gen_random_uuid(), v.name, v.metro
FROM (
    SELECT DISTINCT ON (lower(btrim(venue_name)))
           btrim(venue_name)  AS name,
           btrim(venue_metro) AS metro
    FROM events
    WHERE btrim(coalesce(venue_name, '')) <> ''
    ORDER BY lower(btrim(venue_name)), venue_name
) v;

-- Link events to the matching venue.
UPDATE events e
SET venue_id = ven.id
FROM venues ven
WHERE lower(btrim(e.venue_name)) = lower(ven.name)
  AND btrim(coalesce(e.venue_name, '')) <> '';

-- Drop the now-normalized denormalized columns.
ALTER TABLE events
    DROP COLUMN IF EXISTS venue_name,
    DROP COLUMN IF EXISTS venue_metro;
```

- [ ] **Step 2: Write the down migration**

`db/migrations/000008_venues_table.down.sql`:

```sql
-- Re-add the denormalized columns and repopulate from the linked venue.
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS venue_name  text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS venue_metro text NOT NULL DEFAULT '';

UPDATE events e
SET venue_name  = ven.name,
    venue_metro = ven.metro
FROM venues ven
WHERE ven.id = e.venue_id;

DROP TABLE IF EXISTS venues;
```

(`events.venue_id` keeps its zero-UUID default; rows with no venue stay blank. No FK constraint is added, so there are no ordering concerns.)

- [ ] **Step 3: Apply and verify**

```bash
docker compose up -d --build
migrate -path ./db/migrations -database "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" up
docker compose exec postgres psql -U dev -d lia_dev -c "\d venues"
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT column_name FROM information_schema.columns WHERE table_name='events' AND column_name IN ('venue_name','venue_metro');"
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT count(*) FROM venues;"
```
Expected: `venues` exists with the trigger + `venue_name_lower_idx`; the second query returns **0 rows** (columns dropped); venue count ≥ the number of distinct non-empty venue names that existed.

- [ ] **Step 4: Verify reversibility, then re-up**

```bash
migrate -path ./db/migrations -database "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" down 1
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT column_name FROM information_schema.columns WHERE table_name='events' AND column_name='venue_name';"
migrate -path ./db/migrations -database "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" up
```
Expected: after `down 1` `venue_name` is back (1 row); after `up` it's gone again.

- [ ] **Step 5: Commit**

```bash
git add db/migrations/000008_venues_table.up.sql db/migrations/000008_venues_table.down.sql
git commit -m "feat(backend): add venues table, backfill, drop venue_name/venue_metro (000008)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Venue domain model

**Files:**
- Create: `internal/models/venue.go`

- [ ] **Step 1: Write the model**

`internal/models/venue.go`:

```go
package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Venue is a physical place an event happens at (migration 000008). Identity
// only; coordinates/geo arrive in a later spec.
//
//nolint:govet // field alignment kept for readability and conventional ordering
type Venue struct {
	tableName struct{} `pg:"venues,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID        uuid.UUID `pg:"id,pk,type:uuid"`
	Name      string    `pg:"name,notnull"`
	Address   string    `pg:"address,use_zero"`
	Metro     string    `pg:"metro,use_zero"`
	District  string    `pg:"district,use_zero"`
	CreatedAt time.Time `pg:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,notnull,default:now()"`
}

// BeforeInsert generates a UUID if missing and stamps timestamps.
func (v *Venue) BeforeInsert(ctx context.Context) (context.Context, error) {
	if v.ID == uuid.Nil {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		v.ID = newUUID
	}
	now := time.Now()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	return ctx, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/models/...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/models/venue.go
git commit -m "feat(backend): add Venue domain model

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Venues repository

**Files:**
- Create: `internal/venues/repository.go`

- [ ] **Step 1: Write the repository**

`internal/venues/repository.go`:

```go
// Package venues is the venue domain module of the Lia monolith. It owns the
// venue entity, search, and find-or-create. Identity only — geo arrives later.
package venues

import (
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// DefaultSearchLimit caps Search results when no limit is given.
const DefaultSearchLimit = 20

// Repository defines venue persistence operations.
type Repository interface {
	// Search returns venues whose name matches q (case-insensitive substring),
	// ordered by name. Empty q returns the first `limit` venues by name.
	Search(q string, limit int) ([]*models.Venue, error)
	// GetByID returns a single venue by primary key.
	GetByID(id uuid.UUID) (*models.Venue, error)
	// GetByIDs returns the venues matching the given ids.
	GetByIDs(ids []uuid.UUID) ([]*models.Venue, error)
	// FindOrCreateByName returns an existing venue whose lower(name) matches
	// v.Name, else inserts v and returns it.
	FindOrCreateByName(v *models.Venue) (*models.Venue, error)
}

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed venue repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Search(q string, limit int) ([]*models.Venue, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	var list []*models.Venue
	query := r.db.Model(&list)
	if strings.TrimSpace(q) != "" {
		query = query.Where("name ILIKE ?", "%"+strings.TrimSpace(q)+"%")
	}
	if err := query.Order("name ASC").Limit(limit).Select(); err != nil {
		return nil, fmt.Errorf("search venues from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.Venue, error) {
	venue := &models.Venue{ID: id}
	if err := r.db.Model(venue).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get venue %s from db: %w", id, err)
	}
	return venue, nil
}

func (r *pgRepository) GetByIDs(ids []uuid.UUID) ([]*models.Venue, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*models.Venue
	if err := r.db.Model(&list).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return nil, fmt.Errorf("get venues by ids from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) FindOrCreateByName(v *models.Venue) (*models.Venue, error) {
	existing := new(models.Venue)
	err := r.db.Model(existing).
		Where("lower(name) = lower(?)", v.Name).
		Limit(1).
		Select()
	if err == nil {
		return existing, nil
	}
	if err != pg.ErrNoRows {
		return nil, fmt.Errorf("find venue by name: %w", err)
	}
	if _, err := r.db.Model(v).Insert(); err != nil {
		return nil, fmt.Errorf("insert venue %q: %w", v.Name, err)
	}
	return v, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/venues/...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/venues/repository.go
git commit -m "feat(backend): add venues repository

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Venues service (Search/GetByID/Create/Validate) — TDD

**Files:**
- Create: `internal/venues/service.go`
- Test: `internal/venues/service_test.go`

- [ ] **Step 1: Write the failing test**

`internal/venues/service_test.go`:

```go
package venues

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

type mockRepo struct {
	searchResult []*models.Venue
	getResult    *models.Venue
	getErr       error
	created      *models.Venue
}

func (m *mockRepo) Search(string, int) ([]*models.Venue, error) { return m.searchResult, nil }
func (m *mockRepo) GetByID(uuid.UUID) (*models.Venue, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResult, nil
}
func (m *mockRepo) GetByIDs([]uuid.UUID) ([]*models.Venue, error) { return nil, nil }
func (m *mockRepo) FindOrCreateByName(v *models.Venue) (*models.Venue, error) {
	m.created = v
	return v, nil
}

func venue(name string) *models.Venue {
	id, _ := uuid.NewV4()
	return &models.Venue{ID: id, Name: name}
}

func TestService_Search(t *testing.T) {
	svc := NewService(&mockRepo{searchResult: []*models.Venue{venue("Винзавод")}})
	got, err := svc.Search(context.Background(), "вин", 0)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 venue, got %d", len(got))
	}
}

func TestService_Create_OK(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	got, err := svc.Create(context.Background(), &models.Venue{Name: "  Винзавод  ", Metro: "Чкаловская"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if got == nil || repo.created == nil {
		t.Fatal("expected venue created")
	}
	if repo.created.Name != "Винзавод" {
		t.Fatalf("expected trimmed name, got %q", repo.created.Name)
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), &models.Venue{Name: "   "})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_Zero(t *testing.T) {
	svc := NewService(&mockRepo{})
	got, err := svc.Validate(context.Background(), uuid.Nil)
	if err != nil || got != nil {
		t.Fatalf("expected (nil,nil) for zero id, got (%v,%v)", got, err)
	}
}

func TestService_Validate_Unknown(t *testing.T) {
	svc := NewService(&mockRepo{getErr: errors.New("pg: no rows in result set")})
	id, _ := uuid.NewV4()
	_, err := svc.Validate(context.Background(), id)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_OK(t *testing.T) {
	v := venue("Винзавод")
	svc := NewService(&mockRepo{getResult: v})
	got, err := svc.Validate(context.Background(), v.ID)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if got == nil || got.ID != v.ID {
		t.Fatalf("expected resolved venue, got %v", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/venues/ -run TestService -v`
Expected: FAIL — `undefined: NewService` / `undefined: ErrInvalidInput`.

- [ ] **Step 3: Write the service**

`internal/venues/service.go`:

```go
package venues

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// ErrInvalidInput indicates a venue failed validation or a referenced id is unknown.
var ErrInvalidInput = errors.New("invalid input")

// Service is the venues business-logic interface.
type Service interface {
	// Search returns venues matching q (see Repository.Search).
	Search(ctx context.Context, q string, limit int) ([]*models.Venue, error)
	// GetByID returns a single venue by id.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Venue, error)
	// Create validates (name required), trims the name, and find-or-creates.
	Create(ctx context.Context, v *models.Venue) (*models.Venue, error)
	// Validate resolves a non-zero venue id; returns (nil,nil) for the zero id
	// (meaning "no venue"), or ErrInvalidInput if a non-zero id does not exist.
	Validate(ctx context.Context, id uuid.UUID) (*models.Venue, error)
}

type service struct {
	repo Repository
}

// NewService creates a venues service backed by the given repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Search(_ context.Context, q string, limit int) ([]*models.Venue, error) {
	list, err := s.repo.Search(q, limit)
	if err != nil {
		return nil, fmt.Errorf("search venues: %w", err)
	}
	return list, nil
}

func (s *service) GetByID(_ context.Context, id uuid.UUID) (*models.Venue, error) {
	v, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get venue: %w", err)
	}
	return v, nil
}

func (s *service) Create(_ context.Context, v *models.Venue) (*models.Venue, error) {
	if v == nil {
		return nil, fmt.Errorf("%w: venue is required", ErrInvalidInput)
	}
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	v.Address = strings.TrimSpace(v.Address)
	v.Metro = strings.TrimSpace(v.Metro)
	v.District = strings.TrimSpace(v.District)

	created, err := s.repo.FindOrCreateByName(v)
	if err != nil {
		return nil, fmt.Errorf("create venue: %w", err)
	}
	return created, nil
}

func (s *service) Validate(_ context.Context, id uuid.UUID) (*models.Venue, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	v, err := s.repo.GetByID(id)
	if err != nil {
		if wrapped := errors.Unwrap(err); (wrapped != nil && wrapped.Error() == "pg: no rows in result set") ||
			err.Error() == "pg: no rows in result set" {
			return nil, fmt.Errorf("%w: venue %s does not exist", ErrInvalidInput, id)
		}
		return nil, fmt.Errorf("validate venue: %w", err)
	}
	return v, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/venues/ -run TestService -v`
Expected: PASS (all 6 cases).

- [ ] **Step 5: Commit**

```bash
git add internal/venues/service.go internal/venues/service_test.go
git commit -m "feat(backend): add venues service with search/create/validate

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Event model — venue field

**Files:**
- Modify: `internal/models/event.go`

- [ ] **Step 1: Replace the denormalized venue fields**

In `internal/models/event.go`, replace:

```go
	// VenueName / VenueMetro remain denormalized until the venues module lands.
	VenueName  string `pg:"venue_name,use_zero"`
	VenueMetro string `pg:"venue_metro,use_zero"`
```

with:

```go
	// Venue is normalized into the venues entity (migration 000008). It is the
	// loaded read model (not a column); VenueID is the loose reference (zero
	// UUID = "no venue", no DB FK — see migration comment).
	Venue *Venue `pg:"-"`
```

(`VenueID uuid.UUID` a few lines above stays unchanged.)

- [ ] **Step 2: Verify the models package builds**

Run: `go build ./internal/models/...`
Expected: success. (Do NOT run `go build ./...` — the formatter/events repo still reference the old fields until later tasks.)

- [ ] **Step 3: Commit**

```bash
git add internal/models/event.go
git commit -m "feat(backend): normalize Event venue field (Venue *Venue)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Events service — validate venue on Create — TDD

**Files:**
- Modify: `internal/events/service.go`
- Modify: `internal/events/service_test.go`

Context: the events service currently has `NewService(repo Repository, categories CategoryValidator)` and validates categories in `Create`. Add a `VenueValidator` as a third dependency.

- [ ] **Step 1: Update the test (add a venue mock + 3-arg NewService + venue cases)**

In `internal/events/service_test.go`:

1. Add the import (with the existing imports):

```go
	"github.com/Pashteto/lia/internal/venues"
```

2. Add a venue mock after the existing `mockValidator` type:

```go
// mockVenueValidator is an in-memory VenueValidator.
type mockVenueValidator struct {
	resolved *models.Venue
	err      error
}

func (m *mockVenueValidator) Validate(context.Context, uuid.UUID) (*models.Venue, error) {
	return m.resolved, m.err
}
```

3. Update EVERY `NewService(repo, &mockValidator{...})` call to pass a third arg `&mockVenueValidator{}`. For example `NewService(repo, &mockValidator{})` becomes `NewService(repo, &mockValidator{}, &mockVenueValidator{})`. (There are calls in `TestService_Create`, `TestService_Create_WithCategories`, `TestService_Create_UnknownCategory`, `TestService_Create_InvalidInput`, `TestService_GetByID_InvalidUUID`, `TestService_List_InvalidStatus`, `TestService_List_OK`.)

4. Add two new tests:

```go
func TestService_Create_WithVenue(t *testing.T) {
	id, _ := uuid.NewV4()
	resolved := &models.Venue{ID: id, Name: "Винзавод"}
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{resolved: resolved})

	ev := validEvent()
	ev.VenueID = id
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created.Venue == nil || repo.created.Venue.ID != id {
		t.Fatalf("expected resolved venue on the event, got %v", repo.created.Venue)
	}
}

func TestService_Create_UnknownVenue(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{err: venues.ErrInvalidInput})

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.VenueID = bad
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/events/ -run TestService -v`
Expected: FAIL — `NewService` wants 3 args / `VenueValidator` undefined.

- [ ] **Step 3: Update the service**

In `internal/events/service.go`:

1. Add the import `"github.com/Pashteto/lia/internal/venues"`.

2. After the `CategoryValidator` interface, add:

```go
// VenueValidator resolves and validates a venue id. Satisfied by venues.Service.
type VenueValidator interface {
	Validate(ctx context.Context, id uuid.UUID) (*models.Venue, error)
}
```

3. Replace the `service` struct and `NewService`:

```go
type service struct {
	repo       Repository
	categories CategoryValidator
	venues     VenueValidator
}

// NewService creates an events service backed by the given repository, a
// category validator, and a venue validator.
func NewService(repo Repository, categories CategoryValidator, venues VenueValidator) Service {
	return &service{repo: repo, categories: categories, venues: venues}
}
```

4. In `Create`, after the category validation block (`event.Categories = resolved`) and before `s.repo.Create(event)`, add:

```go
	venue, err := s.venues.Validate(ctx, event.VenueID)
	if err != nil {
		if errors.Is(err, venues.ErrInvalidInput) {
			return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		return fmt.Errorf("validate venue: %w", err)
	}
	event.Venue = venue
```

(`errors` and `fmt` are already imported.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/events/ -run TestService -v`
Expected: all PASS. (Do NOT `go build ./...` — `application.go` still calls the 2-arg `NewService` until Task 10.)

- [ ] **Step 5: Commit**

```bash
git add internal/events/service.go internal/events/service_test.go
git commit -m "feat(backend): validate venue_id in events.Create

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Events repository — load venue

**Files:**
- Modify: `internal/events/repository.go`

Context: the repo already loads categories via `loadCategories` in `GetByID`/`List`. Add a parallel `loadVenues`. `Create` needs no change (the `venue_id` column is set from `event.VenueID`).

- [ ] **Step 1: Add the loadVenues helper at the end of the file**

```go
// loadVenues populates Venue on each event whose venue_id is set (non-zero),
// in a single query (no N+1).
func (r *pgRepository) loadVenues(events []*models.Event) error {
	ids := make([]uuid.UUID, 0, len(events))
	seen := make(map[uuid.UUID]struct{})
	for _, e := range events {
		if e.VenueID != uuid.Nil {
			if _, ok := seen[e.VenueID]; !ok {
				seen[e.VenueID] = struct{}{}
				ids = append(ids, e.VenueID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []struct {
		Name     string    `pg:"name"`
		Address  string    `pg:"address"`
		Metro    string    `pg:"metro"`
		District string    `pg:"district"`
		ID       uuid.UUID `pg:"id"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT id, name, address, metro, district FROM venues WHERE id IN (?)`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load event venues: %w", err)
	}

	byID := make(map[uuid.UUID]*models.Venue, len(rows))
	for i := range rows {
		byID[rows[i].ID] = &models.Venue{
			ID: rows[i].ID, Name: rows[i].Name, Address: rows[i].Address,
			Metro: rows[i].Metro, District: rows[i].District,
		}
	}
	for _, e := range events {
		if v, ok := byID[e.VenueID]; ok {
			e.Venue = v
		}
	}
	return nil
}
```

- [ ] **Step 2: Call loadVenues from GetByID and List**

In `GetByID`, right after the existing `loadCategories` call (before `return event, nil`):

```go
	if err := r.loadVenues([]*models.Event{event}); err != nil {
		return nil, err
	}
```

In `List`, right after the existing `loadCategories(list)` call (before `return list, nil`):

```go
	if err := r.loadVenues(list); err != nil {
		return nil, err
	}
```

- [ ] **Step 3: Verify the package builds and tests pass**

Run: `go build ./internal/events/... && go test ./internal/events/`
Expected: success; tests pass (they use a mock repo, unaffected).

- [ ] **Step 4: Commit**

```bash
git add internal/events/repository.go
git commit -m "feat(backend): load event venue in events repository

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: API contract — swagger.yaml + regenerate

**Files:**
- Modify: `api/swagger.yaml`

- [ ] **Step 1: Add the `/venues` paths**

Under `paths:`, after the `/categories:` block (added by the category work) and before `/health:`, add:

```yaml
  /venues:
    get:
      summary: Search venues
      description: Returns venues whose name matches q (case-insensitive substring), ordered by name.
      operationId: listVenues
      tags:
        - venues
      security: []
      parameters:
        - name: q
          in: query
          description: Name search substring
          required: false
          type: string
        - name: limit
          in: query
          description: Max results (default 20)
          required: false
          type: integer
      responses:
        200:
          description: List of venues
          schema:
            type: array
            items:
              $ref: "#/definitions/Venue"
        503:
          description: Service unavailable (database disabled)
          schema:
            $ref: "#/definitions/Error"
    post:
      summary: Create venue
      description: Creates a venue (find-or-create by normalized name).
      operationId: createVenue
      tags:
        - venues
      security: []
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: "#/definitions/VenueInput"
      responses:
        201:
          description: Venue created (or existing match returned)
          schema:
            $ref: "#/definitions/Venue"
        400:
          description: Invalid input
          schema:
            $ref: "#/definitions/Error"
        503:
          description: Service unavailable (database disabled)
          schema:
            $ref: "#/definitions/Error"
```

- [ ] **Step 2: In the `Event` definition, replace the flat venue fields with `venue`**

Remove from `Event`:

```yaml
      venue_name:
        type: string
      venue_metro:
        type: string
```

Add (next to `venue_id`):

```yaml
      venue:
        $ref: "#/definitions/Venue"
```

(Keep `venue_id` in `Event`.)

- [ ] **Step 3: In the `EventInput` definition, remove the flat venue fields**

Remove from `EventInput`:

```yaml
      venue_name:
        type: string
      venue_metro:
        type: string
```

(Keep `venue_id` in `EventInput` — it is the write reference.)

- [ ] **Step 4: Add the `Venue` and `VenueInput` definitions**

Under `definitions:`, add:

```yaml
  Venue:
    type: object
    required:
      - id
      - name
    properties:
      id:
        type: string
        format: uuid
      name:
        type: string
      address:
        type: string
      metro:
        type: string
      district:
        type: string

  VenueInput:
    type: object
    required:
      - name
    properties:
      name:
        type: string
        minLength: 1
      address:
        type: string
      metro:
        type: string
      district:
        type: string
```

- [ ] **Step 5: Validate and regenerate**

```bash
make swagger-validate
make generate-api
```
Expected: spec valid; `internal/http/server/operations/venues/` generated; `models.Venue`, `models.VenueInput` generated; `models.Event` gains `Venue *Venue`; `models.EventInput` loses `VenueName`/`VenueMetro`.

- [ ] **Step 6: Confirm generated names (report them)**

Run:
```bash
grep -n "Venue" internal/http/models/event.go internal/http/models/event_input.go
cat internal/http/models/venue.go
grep -rn "ListVenues\|CreateVenue" internal/http/server/operations/lia_api_api.go
ls internal/http/server/operations/venues/
```
Confirm and report: `Event.Venue *Venue`; `EventInput` no longer has venue name/metro (still has `VenueID`); `Venue` field pointer-ness (`Name *string`? `ID *strfmt.UUID`?); handler fields `VenuesListVenuesHandler`, `VenuesCreateVenueHandler`; responder constructors (`NewListVenuesOK`, `NewCreateVenueCreated`, `NewCreateVenueBadRequest`, `NewListVenuesServiceUnavailable`, `NewCreateVenueServiceUnavailable`) and params types. The next task depends on these exact names.

- [ ] **Step 7: Commit (spec only)**

```bash
git add api/swagger.yaml
git commit -m "feat(api): Venue/VenueInput, Event.venue, GET/POST /venues

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Formatter + venues HTTP handlers

**Files:**
- Modify: `internal/http/formatter/event.go`
- Create: `internal/http/handlers/venues.go`

Note: match the pointer-ness of the generated `models.Venue` confirmed in Task 8 Step 6. The code below assumes `Venue.ID *strfmt.UUID` and `Venue.Name *string` (required fields → pointers, matching the `Category` precedent), and other fields plain `string`. **If the generator produced different types, adjust the `&` accordingly.**

- [ ] **Step 1: Add VenueToAPI and wire it into the formatter**

In `internal/http/formatter/event.go`:

1. Add:

```go
// VenueToAPI converts a domain Venue to its API representation.
func VenueToAPI(v *domainModels.Venue) *apiModels.Venue {
	if v == nil {
		return nil
	}
	id := strfmt.UUID(v.ID.String())
	name := v.Name
	return &apiModels.Venue{
		ID:       &id,
		Name:     &name,
		Address:  v.Address,
		Metro:    v.Metro,
		District: v.District,
	}
}
```

2. In `EventToAPI`, remove these two lines from the `out` literal:

```go
		VenueName:         event.VenueName,
		VenueMetro:        event.VenueMetro,
```

and, before `return out`, add:

```go
	out.Venue = VenueToAPI(event.Venue)
```

3. In `EventFromAPIInput`, remove these two lines from the `event` literal:

```go
		VenueName:   in.VenueName,
		VenueMetro:  in.VenueMetro,
```

(The `in.VenueID` → `event.VenueID` mapping a few lines below stays.)

- [ ] **Step 2: Write the venues handlers**

Read `internal/http/handlers/events.go` and `categories.go` first to match the `DefaultError` signature and import aliases. Create `internal/http/handlers/venues.go`:

```go
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	venuesops "github.com/Pashteto/lia/internal/http/server/operations/venues"
	"github.com/Pashteto/lia/internal/models"
	venuesdomain "github.com/Pashteto/lia/internal/venues"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListVenues handler searches venues.
type ListVenues struct {
	venues venuesdomain.Service
}

// NewListVenues creates a ListVenues handler.
func NewListVenues(svc venuesdomain.Service) *ListVenues {
	return &ListVenues{venues: svc}
}

// Handle GET /venues.
func (h *ListVenues) Handle(params venuesops.ListVenuesParams) middleware.Responder {
	q := ""
	if params.Q != nil {
		q = *params.Q
	}
	limit := 0
	if params.Limit != nil {
		limit = int(*params.Limit)
	}

	list, err := h.venues.Search(params.HTTPRequest.Context(), q, limit)
	if err != nil {
		logger.Log().Errorf("search venues: %s", err.Error())
		return venuesops.NewListVenuesServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Venue, 0, len(list))
	for _, v := range list {
		payload = append(payload, formatter.VenueToAPI(v))
	}
	return venuesops.NewListVenuesOK().WithPayload(payload)
}

// CreateVenue handler creates (find-or-create) a venue.
type CreateVenue struct {
	venues venuesdomain.Service
}

// NewCreateVenue creates a CreateVenue handler.
func NewCreateVenue(svc venuesdomain.Service) *CreateVenue {
	return &CreateVenue{venues: svc}
}

// Handle POST /venues.
func (h *CreateVenue) Handle(params venuesops.CreateVenueParams) middleware.Responder {
	in := params.Body
	domain := &models.Venue{}
	if in != nil {
		if in.Name != nil {
			domain.Name = *in.Name
		}
		domain.Address = in.Address
		domain.Metro = in.Metro
		domain.District = in.District
	}

	created, err := h.venues.Create(params.HTTPRequest.Context(), domain)
	if err != nil {
		logger.Log().Errorf("create venue: %s", err.Error())
		if errors.Is(err, venuesdomain.ErrInvalidInput) {
			return venuesops.NewCreateVenueBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return venuesops.NewCreateVenueServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	return venuesops.NewCreateVenueCreated().WithPayload(formatter.VenueToAPI(created))
}
```

(Match `VenueInput`'s generated field types: `Name *string` if required; `Address`/`Metro`/`District` likely plain `string` — adjust if the generator differs.)

- [ ] **Step 3: Verify build**

Run: `go build ./internal/http/formatter/... ./internal/http/handlers/...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/http/formatter/event.go internal/http/handlers/venues.go
git commit -m "feat(backend): venue formatter mapping + GET/POST /venues handlers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Wire the venues module

**Files:**
- Modify: `internal/http/module.go`
- Modify: `internal/application.go`

- [ ] **Step 1: HTTP module**

In `internal/http/module.go`:

1. Add import `venuesdomain "github.com/Pashteto/lia/internal/venues"`.
2. Add field to `Module`: `venues venuesdomain.Service`.
3. Add a setter after `SetCategoriesService`:

```go
// SetVenuesService injects the venues domain service. Call before Init.
func (m *Module) SetVenuesService(svc venuesdomain.Service) {
	m.venues = svc
}
```

4. In `initAPI`, after the categories handler registration block, add:

```go
	if m.venues != nil {
		api.VenuesListVenuesHandler = handlers.NewListVenues(m.venues)
		api.VenuesCreateVenueHandler = handlers.NewCreateVenue(m.venues)
	}
```

- [ ] **Step 2: application.go**

In `internal/application.go`:

1. Add import `venuesdomain "github.com/Pashteto/lia/internal/venues"`.
2. Add field to `App`: `venuesSvc venuesdomain.Service`.
3. Replace the existing repo-wiring block:

```go
	if repoModule != nil {
		app.categoriesSvc = categoriesdomain.NewService(categoriesdomain.NewRepository(repoModule.DB()))
		app.eventsSvc = eventsdomain.NewService(
			eventsdomain.NewRepository(repoModule.DB()),
			app.categoriesSvc,
		)
		logger.Log().Info("events + categories modules wired to repository")
	}
```

with:

```go
	if repoModule != nil {
		app.categoriesSvc = categoriesdomain.NewService(categoriesdomain.NewRepository(repoModule.DB()))
		app.venuesSvc = venuesdomain.NewService(venuesdomain.NewRepository(repoModule.DB()))
		app.eventsSvc = eventsdomain.NewService(
			eventsdomain.NewRepository(repoModule.DB()),
			app.categoriesSvc,
			app.venuesSvc,
		)
		logger.Log().Info("events + categories + venues modules wired to repository")
	}
```

4. After `httpModule.SetCategoriesService(app.categoriesSvc)`, add:

```go
		httpModule.SetVenuesService(app.venuesSvc)
```

- [ ] **Step 3: Build the whole backend**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/http/module.go internal/application.go
git commit -m "feat(backend): wire venues module into HTTP + application

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Backend verification

**Files:** none (verification only).

- [ ] **Step 1: build/vet/test**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: all pass.

- [ ] **Step 2: lint (golangci-lint v1)**

```bash
GOBIN=/tmp/glci-v1 go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
/tmp/glci-v1/golangci-lint run ./...
```
Expected: exit 0. Fix any `fieldalignment` findings by reordering struct fields (e.g. in the `loadVenues` anonymous struct, the `uuid.UUID` is placed last — keep pointer/string fields first).

- [ ] **Step 3: live API**

```bash
docker compose up -d --build
curl -s localhost:8080/api/v1/health
# create a venue
VEN=$(curl -s -X POST localhost:8080/api/v1/venues -H 'Content-Type: application/json' -d '{"name":"Винзавод","metro":"Чкаловская"}')
echo "$VEN" | jq '{id, name, metro}'
VID=$(echo "$VEN" | jq -r '.id')
# find-or-create idempotency: same name returns the same id
VEN2=$(curl -s -X POST localhost:8080/api/v1/venues -H 'Content-Type: application/json' -d '{"name":"винзавод"}')
echo "same id? $([ "$(echo "$VEN2" | jq -r .id)" = "$VID" ] && echo yes || echo no)"
# search
curl -s "localhost:8080/api/v1/venues?q=вин" | jq '[.[].name]'
# create an event with the venue, read it back
EV=$(curl -s -X POST localhost:8080/api/v1/events -H 'Content-Type: application/json' -d "{\"title\":\"Тест venue\",\"status\":\"published\",\"starts_at\":\"2026-08-01T18:00:00Z\",\"venue_id\":\"$VID\"}")
echo "$EV" | jq '{id, venue: .venue.name}'
EVID=$(echo "$EV" | jq -r '.id')
curl -s localhost:8080/api/v1/events/$EVID | jq '.venue | {id, name, metro}'
# empty name -> 400
curl -s -o /dev/null -w "empty venue name -> %{http_code}\n" -X POST localhost:8080/api/v1/venues -H 'Content-Type: application/json' -d '{"name":"  "}'
# unknown venue_id -> 400
curl -s -o /dev/null -w "unknown venue_id -> %{http_code}\n" -X POST localhost:8080/api/v1/events -H 'Content-Type: application/json' -d '{"title":"x","status":"draft","starts_at":"2026-08-01T18:00:00Z","venue_id":"00000000-0000-0000-0000-0000000000aa"}'
```
Expected: health OK; venue created; find-or-create returns the same id for "винзавод"; search finds it; event create + GET both show the venue; empty name → 400; unknown venue_id → 400. Report actual output.

- [ ] **Step 4: Commit (only if a fix was needed)**

```bash
git add -A && git commit -m "chore(backend): verification fixups for venue normalization

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" || echo "nothing to commit"
```

---

## Task 12: Frontend types

**Files:**
- Modify: `lib/types.ts`

- [ ] **Step 1: Edit the types**

In `lib/types.ts`:

1. Add `district?` to `Venue`:

```ts
export interface Venue {
  id: string;
  name: string;
  /** Метро / district label, e.g. "Парк культуры". */
  metro?: string;
  address?: string;
  district?: string;
}
```

2. In `ApiEvent`, replace `venue_name?: string;` and `venue_metro?: string;` with a nested venue (keep `venue_id?`):

```ts
  venue_id?: string;
  venue?: { id: string; name: string; address?: string; metro?: string; district?: string };
```

(Remove the old `venue_name?` / `venue_metro?` lines.)

- [ ] **Step 2: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: errors only in `lib/api.ts` (and possibly `CreateEventForm.tsx`) — the files updated in later tasks. `lib/types.ts` itself clean. Report remaining errors.

- [ ] **Step 3: Commit**

```bash
git add lib/types.ts
git commit -m "feat(frontend): types for nested venue object

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Frontend api.ts — map venue, searchVenues, createVenue

**Files:**
- Modify: `lib/api.ts`

- [ ] **Step 1: Map the nested venue in apiEventToLia**

In `lib/api.ts`, replace:

```ts
    venue: e.venue_name
      ? { id: e.venue_id ?? "", name: e.venue_name, metro: e.venue_metro }
      : undefined,
```

with:

```ts
    venue: e.venue
      ? {
          id: e.venue.id,
          name: e.venue.name,
          metro: e.venue.metro,
          address: e.venue.address,
          district: e.venue.district,
        }
      : undefined,
```

- [ ] **Step 2: Add a venue type + search/create functions**

After `apiEventToLia`, add:

```ts
/** A venue from the backend. */
export interface ApiVenue {
  id: string;
  name: string;
  address?: string;
  metro?: string;
  district?: string;
}

/** Searches venues by name substring. Throws on network/HTTP error. */
export async function searchVenues(q: string, limit = 20): Promise<ApiVenue[]> {
  const params = new URLSearchParams();
  if (q.trim()) params.set("q", q.trim());
  params.set("limit", String(limit));
  const res = await fetch(`${API_V1}/venues?${params.toString()}`);
  if (!res.ok) {
    throw new Error(`search venues failed: ${res.status}`);
  }
  return (await res.json()) as ApiVenue[];
}

/** Creates (find-or-create) a venue. Throws on network/HTTP error. */
export async function createVenue(input: {
  name: string;
  address?: string;
  metro?: string;
  district?: string;
}): Promise<ApiVenue> {
  const res = await fetch(`${API_V1}/venues`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`create venue failed: ${res.status} ${detail}`);
  }
  return (await res.json()) as ApiVenue;
}
```

- [ ] **Step 3: Update CreateEventInput**

In `interface CreateEventInput`, replace:

```ts
  venue_name?: string;
  venue_metro?: string;
```

with:

```ts
  venue_id?: string;
```

- [ ] **Step 4: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: `lib/api.ts` clean; remaining errors only in `CreateEventForm.tsx` (and `mock-events.ts` if its venue shape needs `district` alignment — handled next). Report.

- [ ] **Step 5: Commit**

```bash
git add lib/api.ts
git commit -m "feat(frontend): map nested venue, add searchVenues/createVenue, venue_id input

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Frontend mock data — align venue shape

**Files:**
- Modify: `lib/mock-events.ts`

- [ ] **Step 1: Confirm the mock venues still type-check**

The mock events already use `venue: { id, name, metro }` objects, which satisfy the updated `Venue` type (`district?` is optional). No change is required unless `tsc` reports an error.

Run: `pnpm exec tsc --noEmit`
- If `lib/mock-events.ts` reports **no** error, skip to Step 2 (no edit; this task is a no-op confirmation).
- If it reports an error (e.g. a venue field mismatch), fix the offending mock venue object to match `Venue { id, name, metro?, address?, district? }` and re-run `tsc`.

- [ ] **Step 2: Commit (only if a change was made)**

```bash
git add lib/mock-events.ts
git commit -m "chore(frontend): align mock venue shape" || echo "no change needed"
```

(If no change was needed, note that in the report and proceed — the mock venues already render the demo's venue lines offline.)

---

## Task 15: Frontend — venue typeahead picker

**Files:**
- Create: `components/VenuePicker.tsx`
- Modify: `components/CreateEventForm.tsx`

- [ ] **Step 1: Create the VenuePicker component**

`components/VenuePicker.tsx`:

```tsx
"use client";

import { createVenue, searchVenues, type ApiVenue } from "@/lib/api";
import { cn } from "@/lib/cn";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

const inputCls =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

/**
 * Venue typeahead: debounced search over GET /venues; pick a result (sets
 * value to its id) or create a new venue inline (POST /venues, then select it).
 * `value` is the selected venue id ("" = none); `onChange` reports the new id.
 */
export function VenuePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [selected, setSelected] = useState<ApiVenue | null>(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  const { data: results = [] } = useQuery({
    queryKey: ["venues", debounced],
    queryFn: () => searchVenues(debounced),
    enabled: open,
  });

  const createMut = useMutation({
    mutationFn: (name: string) => createVenue({ name }),
    onSuccess: (venue) => {
      setSelected(venue);
      onChange(venue.id);
      setQuery(venue.name);
      setOpen(false);
    },
  });

  // Clear selection if the parent resets value.
  useEffect(() => {
    if (value === "" && selected) {
      setSelected(null);
      setQuery("");
    }
  }, [value, selected]);

  const pick = (v: ApiVenue) => {
    setSelected(v);
    onChange(v.id);
    setQuery(v.name);
    setOpen(false);
  };

  const trimmed = query.trim();
  const exactExists = results.some(
    (v) => v.name.toLowerCase() === trimmed.toLowerCase(),
  );

  return (
    <div className="relative">
      <input
        className={inputCls}
        placeholder="Площадка — начните вводить название"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
          if (selected) {
            setSelected(null);
            onChange("");
          }
        }}
        onFocus={() => setOpen(true)}
      />
      {open && (trimmed !== "" || results.length > 0) && (
        <div className="absolute z-20 mt-1 max-h-60 w-full overflow-auto rounded-control bg-bg-secondary shadow-card">
          {results.map((v) => (
            <button
              key={v.id}
              type="button"
              onClick={() => pick(v)}
              className="block w-full px-3.5 py-2.5 text-left text-[15px] hover:bg-fill"
            >
              {v.name}
              {v.metro ? (
                <span className="text-label-secondary"> · м. {v.metro}</span>
              ) : null}
            </button>
          ))}
          {trimmed !== "" && !exactExists && (
            <button
              type="button"
              onClick={() => createMut.mutate(trimmed)}
              disabled={createMut.isPending}
              className={cn(
                "block w-full px-3.5 py-2.5 text-left text-[15px] text-accent hover:bg-fill",
                createMut.isPending && "opacity-50",
              )}
            >
              {createMut.isPending ? "Создание…" : `Создать «${trimmed}»`}
            </button>
          )}
          {results.length === 0 && trimmed === "" && (
            <div className="px-3.5 py-2.5 text-[13px] text-label-secondary">
              Начните вводить название площадки
            </div>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify the component type-checks**

Run: `pnpm exec tsc --noEmit`
Expected: `VenuePicker.tsx` clean; remaining error only in `CreateEventForm.tsx` (still references the old venue fields). Report.

- [ ] **Step 3: Wire VenuePicker into CreateEventForm**

In `components/CreateEventForm.tsx`:

1. Add the import:

```tsx
import { VenuePicker } from "@/components/VenuePicker";
```

2. In the zod `schema`, replace:

```tsx
  venueName: z.string().optional(),
  venueMetro: z.string().optional(),
```

with:

```tsx
  venueId: z.string().optional(),
```

3. In `useForm` `defaultValues`, add `venueId: ""`.

4. In the "Место и время" `<Section>`, replace the two venue `<Field>` blocks:

```tsx
          <Field label="Место">
            <input className={inputCls} placeholder="Площадка / venue" {...register("venueName")} />
          </Field>
          <Field label="Метро">
            <input className={inputCls} placeholder="Ближайшее метро" {...register("venueMetro")} />
          </Field>
```

with a single venue picker field:

```tsx
          <Field label="Место">
            <Controller
              control={control}
              name="venueId"
              render={({ field }) => (
                <VenuePicker value={field.value ?? ""} onChange={field.onChange} />
              )}
            />
          </Field>
```

5. In `onSubmit`, replace:

```tsx
      venue_name: v.venueName || undefined,
      venue_metro: v.venueMetro || undefined,
```

with:

```tsx
      venue_id: v.venueId || undefined,
```

(`Controller` is already imported in this file.)

- [ ] **Step 4: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: ZERO errors.

- [ ] **Step 5: Commit**

```bash
git add components/VenuePicker.tsx components/CreateEventForm.tsx
git commit -m "feat(frontend): venue typeahead picker (search + create inline)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Frontend verification

**Files:** none (verification only).

- [ ] **Step 1: lint + build**

```bash
pnpm lint && pnpm build
```
Expected: both clean.

- [ ] **Step 2: live smoke test (backend up from Task 11)**

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080 pnpm dev &
sleep 5
# event detail with a venue (use an event id created in Task 11, or list)
curl -s "localhost:8080/api/v1/events?status=published" | jq '[.[] | {title, venue: .venue.name}]'
EVID=$(curl -s "localhost:8080/api/v1/events?status=published" | jq -r '[.[] | select(.venue != null)][0].id')
echo "detail id: $EVID"
curl -s localhost:3000/events/$EVID | grep -oE "Винзавод|Чкаловская" | sort -u
# create page returns 200 and has the "Место" field
curl -s -o /dev/null -w "create page -> %{http_code}\n" localhost:3000/events/new
```
Expected: events JSON shows venue names; the detail SSR HTML contains the venue name/metro; create page 200. Note: the VenuePicker dropdown is client-rendered (TanStack Query), so it won't appear in raw SSR HTML — confirm via build + the type-checked code, and (bonus) a Playwright click-through if browser tooling is available: type in the picker → pick or create → submit → detail shows the venue. Stop the dev server when done.

- [ ] **Step 3: mock fallback (deploy parity)**

Stop the backend; reload Discovery/detail. Mock events still render venue names (`venue.name`/`metro`) offline. The create-form picker shows its empty/"начните вводить" state with no backend — graceful. Confirm and report.

- [ ] **Step 4: Commit (only if a fix was needed)**

```bash
git add -A && git commit -m "chore(frontend): verification fixups for venue picker

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" || echo "nothing to commit"
```

---

## Task 17: Finish the branch

**Files:** none.

- [ ] **Step 1: Update HANDOFF + project memory**

Edit `docs/HANDOFF.md`: move venues out of "What's next" into the recently-done note (venue identity normalized; geo still a separate future spec); keep "AI-search", "Auth+RSVP", "Images", and the new "venue geo" item in the backlog. Commit.

- [ ] **Step 2: Final verification + complete via finishing-a-development-branch**

Run backend `go test ./...` + frontend `pnpm build`; then use the `superpowers:finishing-a-development-branch` skill to present merge/PR options. Frontend-demo redeploy (mock data) is optional and requires explicit SSH authorization for oracle-1 — do not deploy without it.

---

## Self-Review (completed during planning)

- **Spec coverage:** venues table + lower(name) index (T1, no unique constraint per spec); loose `venue_id` no-FK (T1 keeps the column, T5/T7 keep `VenueID`); model (T2); search/get/getbyids/find-or-create (T3); service search/create(validate name + find-or-create)/validate(zero→nil, unknown→400) (T4); event `Venue *Venue` read model (T5); events validate venue_id (T6) + load venue no-N+1 (T7); swagger `Venue`/`VenueInput`/`Event.venue`/drop flat/`/venues` paths (T8); formatter + handlers incl. 400 on empty name and find-or-create idempotency surfaced (T9, T11 live test); wiring (T10); full backend verify incl golangci v1 + live API (T11); frontend nested venue types (T12), api map + search/create + venue_id (T13), mock align (T14), typeahead pick-or-create inline (T15) wired into the create form, verify + mock parity (T16); HANDOFF + finish (T17). All spec sections map to a task.
- **Inline pick-or-create** (the clarified requirement): T15 `VenuePicker` searches existing venues AND offers "Создать «…»" inline — no separate management screen.
- **Type consistency:** `Venue *Venue`/`VenueID uuid.UUID` (T5) used by T6/T7/T9; `NewService(repo, categories, venues)` defined T6, called T10; venues `Service` methods (`Search`/`GetByID`/`Create`/`Validate`) consistent across T4/T6/T9/T10; frontend `ApiVenue` + nested `venue` consistent across T12/T13/T15; generated names (`Event.Venue`, `VenuesListVenuesHandler`, `VenuesCreateVenueHandler`, responder constructors, `Venue`/`VenueInput` pointer-ness) flagged for confirmation in T8/T9 since go-swagger output can't be hand-edited.
- **Placeholders:** none — every code step carries full code; generated-name confirmations are explicit verification steps.
- **go-pg note:** `FindOrCreateByName` and `Validate` compare against `pg.ErrNoRows` / the "pg: no rows in result set" string (matching the existing `events/service.go:86` pattern) — consistent with the codebase.
