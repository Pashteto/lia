# Category Normalization Implementation Plan

> **Status: ✅ Implemented and merged to `main` (2026-06-13).** Executed subagent-driven (all tasks reviewed); verified end-to-end (live API + frontend SSR). Kept as the implementation record.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the denormalized `events.category` text column with a curated, seeded `categories` taxonomy and a many-to-many `event_categories` join, wired end-to-end through the backend module, the API, and the frontend.

**Architecture:** New `internal/categories` Go module (model → repository → service → HTTP handler) owns the taxonomy and a `GET /categories` endpoint. The `events` module owns the `event_categories` links (part of the event aggregate): `events.Create` validates incoming `category_ids` against the categories service and writes the join rows in one transaction; reads load each event's categories via a join. The API (`swagger.yaml`, regenerated) swaps the flat `category` string for a `categories` array (read) and `category_ids` (write). The frontend maps the array, renders chips, and offers a multi-select picker.

**Tech Stack:** Go modular monolith (go-pg, go-swagger, golang-migrate, PostgreSQL); Next.js App Router + TypeScript + Tailwind v4 + TanStack Query + React Hook Form + Zod.

**Spec:** `docs/superpowers/specs/2026-06-13-category-normalization-design.md`

**Branch:** `feat/category-normalization` (already checked out).

---

## Conventions for this plan

- Backend commands run from `backend/`. Frontend commands run from `frontend/`.
- **golangci-lint is v1** — do not migrate `.golangci.yml` to v2 (project gotcha).
- Generated code under `internal/http/server/` is gitignored; regenerate with `make generate-all` after editing `api/swagger.yaml`. Never hand-edit generated files.
- Local Docker can be flaky (project gotcha). If the app container dies, run the binary on the host against the containerized Postgres: `go build -o /tmp/lia ./cmd/lia.go` then `serve` with `DATABASE_*`/`HTTP_*` env. Postgres stays in compose.
- Commit after each task. Use conventional-commit messages with the `Co-Authored-By` trailer the repo uses.

---

## File Structure

**Backend — create:**
- `db/migrations/000006_categories_table.up.sql` / `.down.sql` — taxonomy table + seed.
- `db/migrations/000007_event_categories.up.sql` / `.down.sql` — join table, backfill, drop `events.category`.
- `internal/models/category.go` — `Category` domain model.
- `internal/categories/repository.go` — go-pg `Repository` (`List`, `GetByIDs`).
- `internal/categories/service.go` — `Service` (`List`, `Validate`) + domain errors.
- `internal/categories/service_test.go` — service unit tests (mock repo).
- `internal/http/handlers/categories.go` — `ListCategories` HTTP handler.

**Backend — modify:**
- `internal/models/event.go` — add `CategoryIDs` (input) and `Categories` (output) non-persisted fields; drop the `Category` column field.
- `internal/events/repository.go` — transactional `Create` (event + join rows); load categories in `GetByID`/`List`.
- `internal/events/service.go` — accept a `CategoryValidator`; validate `CategoryIDs` in `Create`.
- `internal/events/service_test.go` — update `NewService` calls; add category cases.
- `api/swagger.yaml` — `Category` definition; `Event.categories`; `EventInput.category_ids`; remove flat `category`; `/categories` path.
- `internal/http/formatter/event.go` — map categories both directions; add `CategoryToAPI`.
- `internal/http/module.go` — `SetCategoriesService` + register `ListCategories`.
- `internal/application.go` — construct the categories service; pass it into events + HTTP.

**Frontend — modify:**
- `lib/types.ts` — `EventCategory.id`; `LiaEvent.categories[]`; `ApiEvent.categories[]`; `CreateEventInput.category_ids`.
- `lib/api.ts` — map `categories[]`; add `getCategories()`; send `category_ids`.
- `lib/mock-events.ts` — convert each event's `category` to `categories[]`; decouple `FILTERS` type.
- `components/ui/EventCard.tsx` — render first category.
- `app/events/[id]/page.tsx` — render category chips.
- `components/DiscoveryFeed.tsx` — filter by `categories.some(...)`.
- `components/CreateEventForm.tsx` — fetch categories, multi-select picker, submit `category_ids`.

---

## Task 1: Migration — categories table + seed

**Files:**
- Create: `db/migrations/000006_categories_table.up.sql`
- Create: `db/migrations/000006_categories_table.down.sql`

- [ ] **Step 1: Write the up migration**

`db/migrations/000006_categories_table.up.sql`:

```sql
-- Curated event category taxonomy. Replaces the denormalized events.category
-- text column (migration 000005) — see 000007 for the join + backfill.
CREATE TABLE IF NOT EXISTS categories
(
    id          uuid NOT NULL
        CONSTRAINT category_id_pkey PRIMARY KEY,
    slug        text NOT NULL
        CONSTRAINT category_slug_unique UNIQUE,
    label       text NOT NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- reuse update_updated_at_column() from migration 000001
CREATE TRIGGER update_category_updated_at
    BEFORE UPDATE
    ON categories
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

-- Seed the curated set. gen_random_uuid() is built into PostgreSQL 13+
-- (the project's postgis image ships PG 14+).
INSERT INTO categories (id, slug, label, sort_order) VALUES
    (gen_random_uuid(), 'lecture',     'Лекции',        10),
    (gen_random_uuid(), 'workshop',    'Мастер-классы', 20),
    (gen_random_uuid(), 'mediation',   'Медиации',      30),
    (gen_random_uuid(), 'concert',     'Концерты',      40),
    (gen_random_uuid(), 'exhibition',  'Выставки',      50),
    (gen_random_uuid(), 'performance', 'Спектакли',     60),
    (gen_random_uuid(), 'film',        'Кино',          70),
    (gen_random_uuid(), 'festival',    'Фестивали',     80)
ON CONFLICT (slug) DO NOTHING;
```

- [ ] **Step 2: Write the down migration**

`db/migrations/000006_categories_table.down.sql`:

```sql
DROP TABLE IF EXISTS categories;
```

- [ ] **Step 3: Apply and verify the migration**

Bring up Postgres + run migrations:

```bash
docker compose up -d --build
```

Then verify the seed:

```bash
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT slug, label, sort_order FROM categories ORDER BY sort_order;"
```

Expected: 8 rows, `lecture … festival`, sort_order 10–80.

> If `docker compose exec postgres` is unavailable because the app container is flaky, Postgres still runs — connect with `psql "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" -c "..."`.

- [ ] **Step 4: Commit**

```bash
git add db/migrations/000006_categories_table.up.sql db/migrations/000006_categories_table.down.sql
git commit -m "feat(backend): add categories taxonomy table + seed (migration 000006)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Migration — event_categories join + backfill + drop denormalized column

**Files:**
- Create: `db/migrations/000007_event_categories.up.sql`
- Create: `db/migrations/000007_event_categories.down.sql`

- [ ] **Step 1: Write the up migration**

`db/migrations/000007_event_categories.up.sql`:

```sql
-- Many-to-many between events and the curated categories taxonomy.
CREATE TABLE IF NOT EXISTS event_categories
(
    event_id    uuid NOT NULL
        REFERENCES events(id) ON DELETE CASCADE,
    category_id uuid NOT NULL
        REFERENCES categories(id) ON DELETE RESTRICT,
    PRIMARY KEY (event_id, category_id)
);

-- reverse lookup: "events in category X"
CREATE INDEX IF NOT EXISTS event_categories_category_idx
    ON event_categories USING btree(category_id);

-- Backfill from the denormalized events.category text. The create form stored
-- free-text Russian LABELS (not slugs), so match on categories.label,
-- case-insensitive and trimmed. Non-empty values that match no seeded label are
-- left uncategorized (acceptable at current demo-scale data volume).
INSERT INTO event_categories (event_id, category_id)
SELECT e.id, c.id
FROM events e
JOIN categories c ON lower(btrim(e.category)) = lower(c.label)
WHERE btrim(coalesce(e.category, '')) <> ''
ON CONFLICT DO NOTHING;

-- Drop the now-normalized denormalized column + its index.
DROP INDEX IF EXISTS event_category_idx;
ALTER TABLE events DROP COLUMN IF EXISTS category;
```

- [ ] **Step 2: Write the down migration**

`db/migrations/000007_event_categories.down.sql`:

```sql
-- Re-add the denormalized column and repopulate it from the first linked
-- category's label (by sort_order) per event.
ALTER TABLE events ADD COLUMN IF NOT EXISTS category text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS event_category_idx
    ON events USING btree(category);

UPDATE events e
SET category = sub.label
FROM (
    SELECT ec.event_id,
           c.label,
           row_number() OVER (PARTITION BY ec.event_id ORDER BY c.sort_order) AS rn
    FROM event_categories ec
    JOIN categories c ON c.id = ec.category_id
) sub
WHERE sub.event_id = e.id AND sub.rn = 1;

DROP TABLE IF EXISTS event_categories;
```

- [ ] **Step 3: Apply and verify**

```bash
docker compose up -d --build
docker compose exec postgres psql -U dev -d lia_dev -c "\d event_categories"
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT column_name FROM information_schema.columns WHERE table_name='events' AND column_name='category';"
```

Expected: `event_categories` table exists with the FKs + PK; the second query returns **0 rows** (column dropped).

- [ ] **Step 4: Verify the down migration is reversible, then re-up**

```bash
migrate -path ./db/migrations -database "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" down 1
docker compose exec postgres psql -U dev -d lia_dev -c "SELECT column_name FROM information_schema.columns WHERE table_name='events' AND column_name='category';"
migrate -path ./db/migrations -database "postgres://dev:dev@localhost:5432/lia_dev?sslmode=disable" up
```

Expected: after `down 1` the `category` column is back (1 row); after `up` it's gone again. (`migrate` is installed via `make migrate-install`.)

- [ ] **Step 5: Commit**

```bash
git add db/migrations/000007_event_categories.up.sql db/migrations/000007_event_categories.down.sql
git commit -m "feat(backend): add event_categories join, backfill, drop events.category (000007)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Category domain model

**Files:**
- Create: `internal/models/category.go`

- [ ] **Step 1: Write the model**

`internal/models/category.go`:

```go
package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Category is a curated event category in the taxonomy (migration 000006).
//
//nolint:govet // field alignment kept for readability and conventional ordering
type Category struct {
	tableName struct{} `pg:"categories,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID        uuid.UUID `pg:"id,pk,type:uuid"`
	Slug      string    `pg:"slug,notnull"`
	Label     string    `pg:"label,notnull"`
	SortOrder int       `pg:"sort_order,use_zero"`
	CreatedAt time.Time `pg:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,notnull,default:now()"`
}

// BeforeInsert generates a UUID if missing and stamps timestamps. Categories are
// seeded via migration today; this keeps the model usable if inserts are added.
func (c *Category) BeforeInsert(ctx context.Context) (context.Context, error) {
	if c.ID == uuid.Nil {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		c.ID = newUUID
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return ctx, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/models/...`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/models/category.go
git commit -m "feat(backend): add Category domain model

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Categories repository

**Files:**
- Create: `internal/categories/repository.go`

- [ ] **Step 1: Write the repository**

`internal/categories/repository.go`:

```go
// Package categories is the category-taxonomy domain module of the Lia monolith.
// It owns the curated categories list and validation of category references.
package categories

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// Repository defines category persistence operations.
type Repository interface {
	// List returns all categories ordered by sort_order.
	List() ([]*models.Category, error)
	// GetByIDs returns the categories matching the given ids (order by sort_order).
	GetByIDs(ids []uuid.UUID) ([]*models.Category, error)
}

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed category repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) List() ([]*models.Category, error) {
	var list []*models.Category
	if err := r.db.Model(&list).Order("sort_order ASC").Select(); err != nil {
		return nil, fmt.Errorf("list categories from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetByIDs(ids []uuid.UUID) ([]*models.Category, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*models.Category
	if err := r.db.Model(&list).
		Where("id IN (?)", pg.In(ids)).
		Order("sort_order ASC").
		Select(); err != nil {
		return nil, fmt.Errorf("get categories by ids from db: %w", err)
	}
	return list, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/categories/...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/categories/repository.go
git commit -m "feat(backend): add categories repository

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Categories service (List + Validate) — TDD

**Files:**
- Create: `internal/categories/service.go`
- Test: `internal/categories/service_test.go`

- [ ] **Step 1: Write the failing test**

`internal/categories/service_test.go`:

```go
package categories

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

type mockRepo struct {
	list   []*models.Category
	byIDs  []*models.Category
	getErr error
}

func (m *mockRepo) List() ([]*models.Category, error) { return m.list, nil }
func (m *mockRepo) GetByIDs([]uuid.UUID) ([]*models.Category, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.byIDs, nil
}

func cat(slug string) *models.Category {
	id, _ := uuid.NewV4()
	return &models.Category{ID: id, Slug: slug, Label: slug}
}

func TestService_List(t *testing.T) {
	svc := NewService(&mockRepo{list: []*models.Category{cat("lecture")}})
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 category, got %d", len(got))
	}
}

func TestService_Validate_Empty(t *testing.T) {
	svc := NewService(&mockRepo{})
	got, err := svc.Validate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Validate(nil) returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil categories, got %v", got)
	}
}

func TestService_Validate_AllResolve(t *testing.T) {
	c := cat("lecture")
	svc := NewService(&mockRepo{byIDs: []*models.Category{c}})
	got, err := svc.Validate(context.Background(), []uuid.UUID{c.ID})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resolved category, got %d", len(got))
	}
}

func TestService_Validate_UnknownID(t *testing.T) {
	// repo resolves zero rows for the requested id -> invalid input.
	svc := NewService(&mockRepo{byIDs: nil})
	unknown, _ := uuid.NewV4()
	_, err := svc.Validate(context.Background(), []uuid.UUID{unknown})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_DeduplicatesRequestedIDs(t *testing.T) {
	c := cat("lecture")
	// one row resolved; request the same id twice -> still valid (dedup).
	svc := NewService(&mockRepo{byIDs: []*models.Category{c}})
	_, err := svc.Validate(context.Background(), []uuid.UUID{c.ID, c.ID})
	if err != nil {
		t.Fatalf("expected duplicate ids to validate, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/categories/ -run TestService -v`
Expected: FAIL — `undefined: NewService` / `undefined: ErrInvalidInput`.

- [ ] **Step 3: Write the service**

`internal/categories/service.go`:

```go
package categories

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// ErrInvalidInput indicates a category reference failed validation.
var ErrInvalidInput = errors.New("invalid input")

// Service is the categories business-logic interface.
type Service interface {
	// List returns the full curated taxonomy, ordered by sort_order.
	List(ctx context.Context) ([]*models.Category, error)
	// Validate resolves the given category ids, returning the matching
	// categories. Returns ErrInvalidInput if any id does not exist. An empty
	// input is valid and resolves to nil.
	Validate(ctx context.Context, ids []uuid.UUID) ([]*models.Category, error)
}

type service struct {
	repo Repository
}

// NewService creates a categories service backed by the given repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(_ context.Context) ([]*models.Category, error) {
	list, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return list, nil
}

func (s *service) Validate(_ context.Context, ids []uuid.UUID) ([]*models.Category, error) {
	unique := dedupeUUIDs(ids)
	if len(unique) == 0 {
		return nil, nil
	}

	found, err := s.repo.GetByIDs(unique)
	if err != nil {
		return nil, fmt.Errorf("resolve categories: %w", err)
	}
	if len(found) != len(unique) {
		return nil, fmt.Errorf("%w: one or more category_ids do not exist", ErrInvalidInput)
	}
	return found, nil
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/categories/ -run TestService -v`
Expected: PASS (all 5 cases).

- [ ] **Step 5: Commit**

```bash
git add internal/categories/service.go internal/categories/service_test.go
git commit -m "feat(backend): add categories service with validation

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Event model — category fields

**Files:**
- Modify: `internal/models/event.go`

- [ ] **Step 1: Replace the denormalized Category field with normalized fields**

In `internal/models/event.go`, replace this block:

```go
	// Category / Venue* are denormalized for now (see migration 000005); they
	// move to dedicated categories / venues modules later.
	Category    string      `pg:"category,use_zero"`
	VenueName   string      `pg:"venue_name,use_zero"`
	VenueMetro  string      `pg:"venue_metro,use_zero"`
```

with:

```go
	// VenueName / VenueMetro remain denormalized until the venues module lands.
	VenueName  string `pg:"venue_name,use_zero"`
	VenueMetro string `pg:"venue_metro,use_zero"`
	// Category is normalized into the categories taxonomy (migration 000006/7).
	// CategoryIDs is write-only input (set from the API), Categories is the
	// loaded read model; neither is a column on the events table.
	CategoryIDs []uuid.UUID `pg:"-"`
	Categories  []*Category `pg:"-"`
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/models/...`
Expected: success. (`uuid` is already imported in this file.)

- [ ] **Step 3: Commit**

```bash
git add internal/models/event.go
git commit -m "feat(backend): normalize Event category fields (CategoryIDs/Categories)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Events service — validate categories on Create — TDD

**Files:**
- Modify: `internal/events/service.go`
- Modify: `internal/events/service_test.go`

- [ ] **Step 1: Update the test to the new constructor + add category cases**

Replace `internal/events/service_test.go` in full:

```go
package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/models"
)

// mockRepo is an in-memory Repository for tests.
type mockRepo struct {
	created *models.Event
	getErr  error
	get     *models.Event
	list    []*models.Event
}

func (m *mockRepo) Create(event *models.Event) error {
	m.created = event
	return nil
}

func (m *mockRepo) GetByID(uuid.UUID) (*models.Event, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.get, nil
}

func (m *mockRepo) List(ListFilter) ([]*models.Event, error) {
	return m.list, nil
}

// mockValidator is an in-memory CategoryValidator.
type mockValidator struct {
	resolved []*models.Category
	err      error
}

func (m *mockValidator) Validate(context.Context, []uuid.UUID) ([]*models.Category, error) {
	return m.resolved, m.err
}

func validEvent() *models.Event {
	return &models.Event{
		Title:    "Память и архив",
		Status:   models.EventPublished,
		StartsAt: time.Now().Add(24 * time.Hour),
	}
}

func TestService_Create(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{})

	if err := svc.Create(context.Background(), validEvent()); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected repository.Create to be called")
	}
}

func TestService_Create_WithCategories(t *testing.T) {
	id, _ := uuid.NewV4()
	resolved := []*models.Category{{ID: id, Slug: "lecture", Label: "Лекции"}}
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{resolved: resolved})

	ev := validEvent()
	ev.CategoryIDs = []uuid.UUID{id}
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(repo.created.Categories) != 1 {
		t.Fatalf("expected resolved categories on the event, got %d", len(repo.created.Categories))
	}
}

func TestService_Create_UnknownCategory(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{err: categories.ErrInvalidInput})

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.CategoryIDs = []uuid.UUID{bad}
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	err := svc.Create(context.Background(), &models.Event{}) // missing title/starts_at
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_InvalidUUID(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	_, err := svc.List(context.Background(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_OK(t *testing.T) {
	repo := &mockRepo{list: []*models.Event{validEvent()}}
	svc := NewService(repo, &mockValidator{})

	got, err := svc.List(context.Background(), "published")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/events/ -run TestService -v`
Expected: FAIL — `NewService` now wants a second argument / `CategoryValidator` undefined.

- [ ] **Step 3: Update the service**

In `internal/events/service.go`:

1. Add the `categories` import:

```go
	"github.com/Pashteto/lia/internal/categories"
```

2. Add the validator interface after the `Service` interface block:

```go
// CategoryValidator resolves and validates category ids. Satisfied by
// categories.Service. Kept as a local interface so the events service stays
// testable with a fake.
type CategoryValidator interface {
	Validate(ctx context.Context, ids []uuid.UUID) ([]*models.Category, error)
}
```

3. Replace the `service` struct and `NewService`:

```go
type service struct {
	repo       Repository
	categories CategoryValidator
}

// NewService creates an events service backed by the given repository and a
// category validator.
func NewService(repo Repository, categories CategoryValidator) Service {
	return &service{repo: repo, categories: categories}
}
```

4. Replace the `Create` method:

```go
func (s *service) Create(ctx context.Context, event *models.Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidInput)
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	resolved, err := s.categories.Validate(ctx, event.CategoryIDs)
	if err != nil {
		if errors.Is(err, categories.ErrInvalidInput) {
			return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		return fmt.Errorf("validate categories: %w", err)
	}
	event.Categories = resolved

	if err := s.repo.Create(event); err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	logger.Log().Infof("event created via service: %s", event.ID)
	return nil
}
```

Note: change the `Create` receiver's first param from `_ context.Context` to `ctx context.Context` (it's now used).

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/events/ -run TestService -v`
Expected: PASS (all cases). It will still fail to *build the package's other consumers* (application.go) until Task 11 — that's expected; the `events` package itself compiles and tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/events/service.go internal/events/service_test.go
git commit -m "feat(backend): validate category_ids in events.Create

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Events repository — transactional Create + load categories

**Files:**
- Modify: `internal/events/repository.go`

- [ ] **Step 1: Add the context import**

At the top of `internal/events/repository.go`, add `"context"` to the import block.

- [ ] **Step 2: Replace Create with a transactional version**

```go
func (r *pgRepository) Create(event *models.Event) error {
	logger.Log().Infof("creating event: %s", event.Title)

	// Insert the event and its category links atomically. No Returning("*"):
	// a nullable venue_id read back as NULL cannot be scanned into uuid.UUID.
	// ID and timestamps are set Go-side in BeforeInsert.
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		if _, err := tx.Model(event).Insert(); err != nil {
			return fmt.Errorf("insert event %q: %w", event.Title, err)
		}
		for _, c := range event.Categories {
			if _, err := tx.Exec(
				`INSERT INTO event_categories (event_id, category_id) VALUES (?, ?)
				 ON CONFLICT DO NOTHING`,
				event.ID, c.ID,
			); err != nil {
				return fmt.Errorf("link event %s to category %s: %w", event.ID, c.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	logger.Log().Infof("event created: %s (ID: %s)", event.Title, event.ID)
	return nil
}
```

> The service sets `event.Categories` (resolved) before calling the repo; the join rows come from there, so the link insert uses already-validated category IDs.

- [ ] **Step 3: Add a category loader and call it from GetByID/List**

Add this helper at the end of the file:

```go
// loadCategories populates Categories on each event via the event_categories
// join, in a single query (no N+1).
func (r *pgRepository) loadCategories(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	byID := make(map[uuid.UUID]*models.Event, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
		byID[e.ID] = e
		e.Categories = nil
	}

	var rows []struct {
		EventID uuid.UUID `pg:"event_id"`
		ID      uuid.UUID `pg:"id"`
		Slug    string    `pg:"slug"`
		Label   string    `pg:"label"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT ec.event_id, c.id, c.slug, c.label
		 FROM event_categories ec
		 JOIN categories c ON c.id = ec.category_id
		 WHERE ec.event_id IN (?)
		 ORDER BY c.sort_order ASC`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load event categories: %w", err)
	}

	for _, row := range rows {
		if e, ok := byID[row.EventID]; ok {
			e.Categories = append(e.Categories, &models.Category{
				ID: row.ID, Slug: row.Slug, Label: row.Label,
			})
		}
	}
	return nil
}
```

Then in `GetByID`, before `return event, nil`:

```go
	if err := r.loadCategories([]*models.Event{event}); err != nil {
		return nil, err
	}
```

And in `List`, before `return list, nil`:

```go
	if err := r.loadCategories(list); err != nil {
		return nil, err
	}
```

- [ ] **Step 4: Verify the package builds**

Run: `go build ./internal/events/...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/events/repository.go
git commit -m "feat(backend): write + load event_categories in events repository

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: API contract — swagger.yaml + regenerate

**Files:**
- Modify: `api/swagger.yaml`

- [ ] **Step 1: Add the `/categories` path**

In `api/swagger.yaml`, under `paths:`, after the `/events/{id}:` block and before `/health:`, add:

```yaml
  /categories:
    get:
      summary: List categories
      description: Returns the curated category taxonomy, ordered by sort_order.
      operationId: listCategories
      tags:
        - categories
      security: []
      responses:
        200:
          description: List of categories
          schema:
            type: array
            items:
              $ref: "#/definitions/Category"
        503:
          description: Service unavailable (database disabled)
          schema:
            $ref: "#/definitions/Error"
```

- [ ] **Step 2: In the `Event` definition, replace the flat `category` with `categories`**

Remove:

```yaml
      category:
        type: string
        description: Category label (denormalized for now)
```

Add (next to `description`):

```yaml
      categories:
        type: array
        items:
          $ref: "#/definitions/Category"
```

- [ ] **Step 3: In the `EventInput` definition, replace `category` with `category_ids`**

Remove:

```yaml
      category:
        type: string
```

Add (next to `description`):

```yaml
      category_ids:
        type: array
        items:
          type: string
          format: uuid
```

- [ ] **Step 4: Add the `Category` definition**

Under `definitions:`, after the `EventInput` block, add:

```yaml
  Category:
    type: object
    required:
      - id
      - slug
      - label
    properties:
      id:
        type: string
        format: uuid
      slug:
        type: string
      label:
        type: string
```

- [ ] **Step 5: Validate and regenerate**

```bash
make swagger-validate
make generate-all
```

Expected: `swagger validate` reports the spec is valid; `generate-all` regenerates `internal/http/server/` (including `operations/categories/` and `models.Category`, `models.Event.Categories`, `models.EventInput.CategoryIds`).

- [ ] **Step 6: Confirm the generated names**

Run: `grep -n "Categories\|CategoryIds" internal/http/models/event.go internal/http/models/event_input.go`
Expected: `Event` has `Categories []*Category`; `EventInput` has `CategoryIds []strfmt.UUID`. Note the exact field names — Task 10 depends on them. (If go-swagger named them differently, use the names it produced.)

- [ ] **Step 7: Commit (spec only; generated code is gitignored)**

```bash
git add api/swagger.yaml
git commit -m "feat(api): categories array + category_ids + GET /categories

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Formatter + categories HTTP handler

**Files:**
- Modify: `internal/http/formatter/event.go`
- Create: `internal/http/handlers/categories.go`

- [ ] **Step 1: Add CategoryToAPI and wire it into EventToAPI**

In `internal/http/formatter/event.go`, add a new function:

```go
// CategoryToAPI converts a domain Category to its API representation.
func CategoryToAPI(c *domainModels.Category) *apiModels.Category {
	if c == nil {
		return nil
	}
	return &apiModels.Category{
		ID:    strfmt.UUID(c.ID.String()),
		Slug:  &c.Slug,
		Label: &c.Label,
	}
}
```

> `Slug`/`Label` are `*string` in the generated model because they're `required`. If go-swagger generated them as plain `string`, drop the `&`. Confirm against the generated `models.Category` from Task 9 Step 6.

In `EventToAPI`, remove `Category: event.Category,` from the `out` literal and, before `return out`, add:

```go
	out.Categories = make([]*apiModels.Category, 0, len(event.Categories))
	for _, c := range event.Categories {
		out.Categories = append(out.Categories, CategoryToAPI(c))
	}
```

- [ ] **Step 2: Map category_ids in EventFromAPIInput**

In `EventFromAPIInput`, remove `Category: in.Category,` from the `event` literal and, before `return event, nil`, add:

```go
	for _, raw := range in.CategoryIds {
		if parsed, err := uuid.FromString(raw.String()); err == nil {
			event.CategoryIDs = append(event.CategoryIDs, parsed)
		}
	}
```

- [ ] **Step 3: Write the categories handler**

`internal/http/handlers/categories.go`:

```go
package handlers

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	categoriesdomain "github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	categoriesops "github.com/Pashteto/lia/internal/http/server/operations/categories"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListCategories handler returns the curated category taxonomy.
type ListCategories struct {
	categories categoriesdomain.Service
}

// NewListCategories creates a ListCategories handler.
func NewListCategories(svc categoriesdomain.Service) *ListCategories {
	return &ListCategories{categories: svc}
}

// Handle GET /categories.
func (h *ListCategories) Handle(params categoriesops.ListCategoriesParams) middleware.Responder {
	list, err := h.categories.List(params.HTTPRequest.Context())
	if err != nil {
		logger.Log().Errorf("list categories: %s", err.Error())
		return categoriesops.NewListCategoriesServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Category, 0, len(list))
	for _, c := range list {
		payload = append(payload, formatter.CategoryToAPI(c))
	}

	return categoriesops.NewListCategoriesOK().WithPayload(payload)
}
```

> Confirm the generated responder names (`NewListCategoriesOK`, `NewListCategoriesServiceUnavailable`) and the params type against `internal/http/server/operations/categories/` from Task 9. Match the pattern in `handlers/events.go`.

- [ ] **Step 4: Verify build**

Run: `go build ./internal/http/...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/http/formatter/event.go internal/http/handlers/categories.go
git commit -m "feat(backend): formatter category mapping + GET /categories handler

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Wire the categories module

**Files:**
- Modify: `internal/http/module.go`
- Modify: `internal/application.go`

- [ ] **Step 1: Add categories to the HTTP module**

In `internal/http/module.go`:

1. Add the import:

```go
	categoriesdomain "github.com/Pashteto/lia/internal/categories"
```

2. Add a field to `Module`:

```go
	categories categoriesdomain.Service
```

3. Add a setter after `SetEventsService`:

```go
// SetCategoriesService injects the categories domain service. Call before Init.
func (m *Module) SetCategoriesService(svc categoriesdomain.Service) {
	m.categories = svc
}
```

4. In `initAPI`, after the events handler registration block, add:

```go
	if m.categories != nil {
		api.CategoriesListCategoriesHandler = handlers.NewListCategories(m.categories)
	}
```

> Confirm the generated field name `CategoriesListCategoriesHandler` on `operations.LiaAPIAPI` (grep it in `internal/http/server/operations/lia_api_api.go`).

- [ ] **Step 2: Construct + inject the categories service in application.go**

In `internal/application.go`:

1. Add the import:

```go
	categoriesdomain "github.com/Pashteto/lia/internal/categories"
```

2. Add a field to `App`:

```go
	// categoriesSvc is the categories domain service. Nil when the DB is disabled.
	categoriesSvc categoriesdomain.Service
```

3. Replace the events-wiring block:

```go
	if repoModule != nil {
		app.eventsSvc = eventsdomain.NewService(eventsdomain.NewRepository(repoModule.DB()))
		logger.Log().Info("events module wired to repository")
	}
```

with:

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

4. In the HTTP module registration block, after `httpModule.SetEventsService(app.eventsSvc)`, add:

```go
		httpModule.SetCategoriesService(app.categoriesSvc)
```

- [ ] **Step 3: Build the whole backend**

Run: `go build ./...`
Expected: success — application.go now satisfies the new `events.NewService` signature.

- [ ] **Step 4: Commit**

```bash
git add internal/http/module.go internal/application.go
git commit -m "feat(backend): wire categories module into HTTP + application

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Backend verification (build, vet, test, lint, live API)

**Files:** none (verification only).

- [ ] **Step 1: Build, vet, test**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: all pass.

- [ ] **Step 2: Lint as CI does (golangci-lint v1)**

```bash
golangci-lint run ./...
```
Expected: exit 0. (Install v1 if missing — do NOT use v2; `.golangci.yml` is v1 format.)

- [ ] **Step 3: Bring the stack up and exercise the live API**

```bash
docker compose up -d --build
curl -s localhost:8080/api/v1/health
curl -s localhost:8080/api/v1/categories | jq '. | length, .[0]'
```
Expected: health OK; categories returns 8 entries, first is `{id, slug:"lecture", label:"Лекции"}`.

> If the app container is flaky, run the binary on the host (see Conventions) and curl the same endpoints.

- [ ] **Step 4: Create an event with categories and read it back**

```bash
# grab two category ids
IDS=$(curl -s localhost:8080/api/v1/categories | jq -r '[.[0].id, .[1].id]')
LECTURE=$(echo "$IDS" | jq -r '.[0]')
WORKSHOP=$(echo "$IDS" | jq -r '.[1]')

CREATED=$(curl -s -X POST localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"Тест категорий\",\"status\":\"published\",\"starts_at\":\"2026-07-01T18:00:00Z\",\"category_ids\":[\"$LECTURE\",\"$WORKSHOP\"]}")
echo "$CREATED" | jq '{id, categories}'

EVID=$(echo "$CREATED" | jq -r '.id')
curl -s localhost:8080/api/v1/events/$EVID | jq '.categories | map(.slug)'
```
Expected: the created event and the GET both return 2 categories (`["lecture","workshop"]`).

- [ ] **Step 5: Confirm an unknown category id is rejected**

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -d '{"title":"Bad cat","status":"draft","starts_at":"2026-07-01T18:00:00Z","category_ids":["00000000-0000-0000-0000-000000000123"]}'
```
Expected: `400`.

- [ ] **Step 6: Commit (if any lint-driven tweaks were needed)**

```bash
git add -A && git commit -m "chore(backend): verification fixups for category normalization

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" || echo "nothing to commit"
```

---

## Task 13: Frontend types

**Files:**
- Modify: `lib/types.ts`

- [ ] **Step 1: Add `id` to EventCategory**

In `lib/types.ts`, change `EventCategory`:

```ts
export interface EventCategory {
  /** Stable category id (uuid) from the backend. */
  id: string;
  /** Stable slug used for filtering. */
  slug: string;
  /** Russian display label, e.g. "Медиации". */
  label: string;
}
```

- [ ] **Step 2: Change `LiaEvent.category` to `categories`**

Replace:

```ts
  /** Optional: the backend events model has no category yet. */
  category?: EventCategory;
```

with:

```ts
  /** Categories from the curated taxonomy (many-to-many). */
  categories: EventCategory[];
```

- [ ] **Step 3: Update the API event shape**

In `ApiEvent`, replace `category?: string;` with:

```ts
  categories?: { id: string; slug: string; label: string }[];
```

- [ ] **Step 4: Verify the EventInput type (used by api.ts)**

`ApiEvent` may include the input shape lower in the file. If `lib/types.ts` defines an event-input type with `category?: string`, replace it with `category_ids?: string[];`. (The canonical input type lives in `lib/api.ts` — handled in Task 14. If `types.ts` has no input type, skip.)

- [ ] **Step 5: Commit**

```bash
git add lib/types.ts
git commit -m "feat(frontend): types for category[] taxonomy

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Frontend api.ts — map categories, getCategories, send category_ids

**Files:**
- Modify: `lib/api.ts`

- [ ] **Step 1: Map the categories array in apiEventToLia**

Replace:

```ts
    category: e.category ? { slug: e.category, label: e.category } : undefined,
```

with:

```ts
    categories: (e.categories ?? []).map((c) => ({
      id: c.id,
      slug: c.slug,
      label: c.label,
    })),
```

- [ ] **Step 2: Add a category type + getCategories fetch**

After the `apiEventToLia` function, add:

```ts
/** A category from the curated taxonomy. */
export interface ApiCategory {
  id: string;
  slug: string;
  label: string;
}

/** Fetches the curated category taxonomy. Throws on network/HTTP error. */
export async function getCategories(): Promise<ApiCategory[]> {
  const res = await fetch(`${API_V1}/categories`, { next: { revalidate: 300 } });
  if (!res.ok) {
    throw new Error(`fetch categories failed: ${res.status}`);
  }
  return (await res.json()) as ApiCategory[];
}
```

- [ ] **Step 3: Update CreateEventInput**

In `CreateEventInput`, replace `category?: string;` with:

```ts
  category_ids?: string[];
```

- [ ] **Step 4: Verify build of the lib**

Run: `pnpm exec tsc --noEmit`
Expected: errors remain only in files not yet updated (mock-events, EventCard, detail, DiscoveryFeed, CreateEventForm) — `api.ts`/`types.ts` themselves are consistent. (These get fixed in Tasks 15–17.)

- [ ] **Step 5: Commit**

```bash
git add lib/api.ts
git commit -m "feat(frontend): map categories[], add getCategories, send category_ids

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: Frontend mock data + filter type

**Files:**
- Modify: `lib/mock-events.ts`

- [ ] **Step 1: Decouple the FILTERS type and convert mock events to categories arrays**

Replace the top of `lib/mock-events.ts` (the import + `FILTERS`) with:

```ts
import type { LiaEvent } from "./types";

// A discovery filter chip. Special chips (all/today/weekend/nearby) are not
// real categories, so this is its own shape rather than EventCategory.
export interface DiscoveryFilter {
  slug: string;
  label: string;
}

// Mock data for the Discovery scaffold. Content (titles, organizers) is drawn
// from the curatorial copy in design/screens/discovery.html. The deployed demo
// (lia.pashteto.com) renders from this when the backend is unreachable.

export const FILTERS: DiscoveryFilter[] = [
  { slug: "all", label: "Все" },
  { slug: "today", label: "Сегодня" },
  { slug: "weekend", label: "Выходные" },
  { slug: "mediation", label: "Медиации" },
  { slug: "workshop", label: "Мастер-классы" },
  { slug: "lecture", label: "Лекции" },
  { slug: "nearby", label: "Рядом" },
];
```

- [ ] **Step 2: Convert each event's `category` to a `categories` array**

For every event in `MOCK_EVENTS`, replace its single-category line with an array, including an `id`. Use the seeded slugs. Examples:

```ts
    // was: category: { slug: "mediation", label: "Медиации" },
    categories: [{ id: "cat-mediation", slug: "mediation", label: "Медиации" }],
```
```ts
    // was: category: { slug: "workshop", label: "Мастер-классы" },
    categories: [{ id: "cat-workshop", slug: "workshop", label: "Мастер-классы" }],
```
```ts
    // was: category: { slug: "lecture", label: "Лекции" },
    categories: [{ id: "cat-lecture", slug: "lecture", label: "Лекции" }],
```

For the fourth event (`evt-zebald`, currently `mediation`), give it **two** categories to exercise the multi-category UI end-to-end in the demo:

```ts
    categories: [
      { id: "cat-mediation", slug: "mediation", label: "Медиации" },
      { id: "cat-lecture", slug: "lecture", label: "Лекции" },
    ],
```

(The `cat-*` ids are mock-only stand-ins; real ids come from the backend.)

- [ ] **Step 3: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: `mock-events.ts` no longer errors (remaining errors are in EventCard/detail/DiscoveryFeed/CreateEventForm).

- [ ] **Step 4: Commit**

```bash
git add lib/mock-events.ts
git commit -m "feat(frontend): mock events use categories[]; decouple FILTERS type

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Frontend read rendering (card, detail, discovery filter)

**Files:**
- Modify: `components/ui/EventCard.tsx`
- Modify: `app/events/[id]/page.tsx`
- Modify: `components/DiscoveryFeed.tsx`

- [ ] **Step 1: EventCard — show the first category**

In `components/ui/EventCard.tsx`, replace:

```tsx
          {event.category ? (
            <Kicker>{event.category.label}</Kicker>
          ) : (
            <span />
          )}
```

with:

```tsx
          {event.categories.length > 0 ? (
            <Kicker>{event.categories[0].label}</Kicker>
          ) : (
            <span />
          )}
```

- [ ] **Step 2: Detail — render all categories as chips**

In `app/events/[id]/page.tsx`, replace:

```tsx
          {event.category && <Kicker>{event.category.label}</Kicker>}
```

with:

```tsx
          {event.categories.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {event.categories.map((c) => (
                <span
                  key={c.id}
                  className="rounded-full bg-fill px-2.5 py-1 text-[12px] font-medium uppercase tracking-[0.03em] text-label-secondary"
                >
                  {c.label}
                </span>
              ))}
            </div>
          )}
```

- [ ] **Step 3: DiscoveryFeed — filter over the array**

In `components/DiscoveryFeed.tsx`, replace:

```tsx
      const matchesFilter = active === "all" || e.category?.slug === active;
```

with:

```tsx
      const matchesFilter =
        active === "all" || e.categories.some((c) => c.slug === active);
```

- [ ] **Step 4: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: only `CreateEventForm.tsx` still errors (fixed next).

- [ ] **Step 5: Commit**

```bash
git add components/ui/EventCard.tsx app/events/[id]/page.tsx components/DiscoveryFeed.tsx
git commit -m "feat(frontend): render categories[] on card, detail, discovery filter

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: Frontend create form — multi-select category picker

**Files:**
- Modify: `components/CreateEventForm.tsx`

- [ ] **Step 1: Update the schema and imports**

In `components/CreateEventForm.tsx`:

1. Update the api import:

```tsx
import { createEvent, getCategories, type CreateEventInput } from "@/lib/api";
```

2. Add a query import (alongside `useMutation`):

```tsx
import { useMutation, useQuery } from "@tanstack/react-query";
```

3. In the zod `schema`, replace `category: z.string().optional(),` with:

```tsx
  categoryIds: z.array(z.string()).optional(),
```

4. In `useForm` `defaultValues`, add `categoryIds: []`.

- [ ] **Step 2: Fetch categories and render a chip multi-select**

Inside the component, after `const isFree = useWatch(...)`, add the query:

```tsx
  const { data: categories = [] } = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
  });
```

Replace the existing "Категория" `<Field>`:

```tsx
          <Field label="Категория">
            <input
              className={inputCls}
              placeholder="Медиации, Мастер-классы, Лекции…"
              {...register("category")}
            />
          </Field>
```

with a controlled multi-select:

```tsx
          <Field label="Категории">
            <Controller
              control={control}
              name="categoryIds"
              render={({ field }) => {
                const selected = field.value ?? [];
                const toggle = (id: string) =>
                  field.onChange(
                    selected.includes(id)
                      ? selected.filter((s) => s !== id)
                      : [...selected, id],
                  );
                return (
                  <div className="flex flex-wrap gap-2">
                    {categories.length === 0 && (
                      <span className="text-[13px] text-label-secondary">
                        Категории недоступны (бэкенд офлайн)
                      </span>
                    )}
                    {categories.map((c) => {
                      const on = selected.includes(c.id);
                      return (
                        <button
                          key={c.id}
                          type="button"
                          onClick={() => toggle(c.id)}
                          className={cn(
                            "rounded-full px-3 py-1.5 text-[15px] transition",
                            on
                              ? "bg-accent text-white"
                              : "bg-fill text-label",
                          )}
                        >
                          {c.label}
                        </button>
                      );
                    })}
                  </div>
                );
              }}
            />
          </Field>
```

- [ ] **Step 3: Submit category_ids**

In `onSubmit`, replace `category: v.category || undefined,` with:

```tsx
      category_ids: v.categoryIds && v.categoryIds.length > 0 ? v.categoryIds : undefined,
```

- [ ] **Step 4: Verify**

Run: `pnpm exec tsc --noEmit`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add components/CreateEventForm.tsx
git commit -m "feat(frontend): multi-select category picker on create form

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Frontend verification (lint, build, end-to-end)

**Files:** none (verification only).

- [ ] **Step 1: Lint + build**

```bash
pnpm lint && pnpm build
```
Expected: both clean.

- [ ] **Step 2: End-to-end against the live backend (Playwright)**

With the backend up (Task 12) and `pnpm dev` running pointed at it (`NEXT_PUBLIC_API_URL=http://localhost:8080`):

- [ ] Open `http://localhost:3000` — Discovery renders; category filter chips work (selecting "Лекции" narrows the list by `categories.some`).
- [ ] Open `/events/new` — the category picker shows the 8 seeded chips; select **two** (e.g. Лекции + Мастер-классы), fill title + start, set status Опубликовать, Save.
- [ ] After redirect to `/events/{id}` — both category chips render on the detail screen.
- [ ] Back on Discovery — the new event appears and its card kicker shows the first category.

Capture screenshots of the create form (two chips selected) and the detail screen (two chips) as evidence.

- [ ] **Step 3: Verify the mock fallback (deploy parity)**

Stop the backend; reload Discovery. Expected: mock events render with categories (and `evt-zebald` shows two), confirming the deployed demo (mock-only) will show multi-category.

- [ ] **Step 4: Commit any verification fixups**

```bash
git add -A && git commit -m "chore(frontend): verification fixups for category picker

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" || echo "nothing to commit"
```

---

## Task 19: Open the PR

**Files:** none.

- [ ] **Step 1: Push and open the PR**

```bash
git push -u origin feat/category-normalization
gh pr create --base main \
  --title "feat: normalize event categories into a curated taxonomy" \
  --body "Implements docs/superpowers/specs/2026-06-13-category-normalization-design.md. Replaces the denormalized events.category text with a seeded categories taxonomy + event_categories join, full-stack. Deploy is frontend-demo-only (mock); backend not deployed.

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
```

- [ ] **Step 2: After merge — deploy the frontend demo**

Per the spec's Deployment section and the deploy memory, redeploy the **frontend-only** demo on oracle-1 (mock data, now multi-category). This requires explicit SSH authorization for `129.146.183.89`. Do not deploy the backend.

---

## Self-Review (completed during planning)

- **Spec coverage:** data model (T1–T3, T6, T8), curated seed (T1), many-to-many join (T2, T8), `GET /categories` (T9–T11), `category_ids` write contract + validation (T7, T9, T10), 400 on unknown id (T7, T12), frontend types/api/picker/rendering/mocks (T13–T17), deploy = frontend-demo-only/mock parity (T15, T18 Step 3, T19 Step 2), backend tests + golangci v1 + frontend lint/build/e2e (T5, T7, T12, T18). All spec sections map to a task.
- **Slug-vs-label backfill:** T2 matches on `categories.label` (the create form stored labels), per the spec correction.
- **Type consistency:** `CategoryIDs []uuid.UUID` (input) / `Categories []*models.Category` (output) used consistently across T6/T7/T8/T10; `NewService(repo, validator)` signature matches between T7 (def) and T11 (call); frontend `categories: EventCategory[]` consistent across T13–T17; generated names (`Categories`, `CategoryIds`, `CategoriesListCategoriesHandler`) flagged for confirmation in T9/T10/T11 since go-swagger output isn't editable by hand.
- **Placeholders:** none — every code step carries full code; generated-name confirmations are explicit verification steps, not TODOs.
