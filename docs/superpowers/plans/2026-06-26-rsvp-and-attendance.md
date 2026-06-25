# RSVP domain + attendance pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a signed-in user sign up for a practice (open / by application / external), with capacity + waitlist, and see their sign-ups and applications on `/me/practices` and `/me/applications`; plus `.ics` export.

**Architecture:** New `internal/rsvp` domain module mirroring `internal/events` (model → repository → service → swagger → handlers → wiring). RSVP rows live in a new `event_rsvps` table; events gain `signup_mode`, `capacity`, `curator_question`, `external_registration_url` columns. Seat allocation and waitlist promotion run inside a single DB transaction with a row lock on the event to avoid last-seat races. Frontend gets RSVP API calls, event-detail CTA states, and two `/me/*` pages plus a minimal organizer accept/decline surface.

**Tech Stack:** Go 1.26 (modular monolith), go-pg v10, go-swagger (swagger-first codegen), gofrs/uuid, golang-migrate SQL migrations; Next.js App Router + TypeScript frontend.

**Design spec:** `docs/superpowers/specs/2026-06-26-rsvp-and-attendance-design.md`.

## Global Constraints

- **Swagger-first:** edit `backend/api/swagger.yaml`, then run `make generate-all` (generated `internal/http/models` is gitignored and MUST exist to build) then `make generate-api`. Never hand-edit generated code under `internal/http/server` or `internal/http/models`.
- **Build/test commands:** `go build ./...`; run tests with `go test -vet=off ./...` (Go 1.24+ flags the codebase's `clog.*(fmt.Sprintf(...))` idiom as non-constant format strings — pre-existing, not ours).
- **mockery under Go 1.26 won't build** — if a service interface needs a mock, hand-add it with a "hand-added" comment (see `2026-06-25-passwords-and-me-suite.md` Implementation notes). This plan's services are tested with hand-written fakes, not mockery.
- **NULL-uuid rule:** go-pg + gofrs cannot scan SQL NULL into a `uuid.UUID`. Use the zero UUID (`'00000000-0000-0000-0000-000000000000'`) + `use_zero` for "unset" foreign keys, exactly as `events.organizer_id`/`venue_id` do.
- **Privacy:** `.ics` and event responses are public surfaces — never expose participant identities or emails. `my_rsvp_status` is per-caller only.
- **Auth:** all RSVP endpoints except `.ics` require `jwt`. The handler receives `principal *apimodels.User`; the authenticated user id is `principal.UUID` — never trust a client-supplied user id.
- **Timezone:** `.ics` and any month math use `Europe/Moscow` (see `events.startOfMonthMoscow`).
- **Migrations are paired:** every `NNNNNN_name.up.sql` has a matching `.down.sql`. Next free number is `000012`.

---

## File Structure

**Backend — create:**
- `backend/db/migrations/000012_event_signup_fields.up.sql` / `.down.sql` — event signup columns
- `backend/db/migrations/000013_event_rsvps.up.sql` / `.down.sql` — `event_rsvps` table
- `backend/internal/models/rsvp.go` — `Rsvp` model + `RsvpStatus` string type
- `backend/internal/rsvp/repository.go` — persistence (replaces `doc.go`'s package doc; keep the package doc comment)
- `backend/internal/rsvp/service.go` — business logic
- `backend/internal/rsvp/service_test.go` — service tests with a fake repository
- `backend/internal/rsvp/ics.go` — `.ics` (VEVENT) generator + test `ics_test.go`
- `backend/internal/http/handlers/rsvp.go` — RSVP HTTP handlers
- `backend/internal/http/handlers/rsvp_calendar.go` — `.ics` handler

**Backend — modify:**
- `backend/internal/rsvp/doc.go` — drop "Status: skeleton" wording (package doc stays)
- `backend/internal/models/event.go` — add `SignupMode`, `Capacity`, `CuratorQuestion`, `ExternalRegistrationURL` fields; extend `Validate`; extend `Update` column list
- `backend/internal/events/repository.go` — persist the new event columns on Create/Update; add `SeatsRemaining`/`MyRsvpStatus` transient fields population hook point (see Task 8)
- `backend/internal/http/formatter/event.go` — emit new event fields
- `backend/api/swagger.yaml` — event fields, RSVP definitions, RSVP paths
- `backend/internal/http/module.go` — `SetRsvpService` + handler registration
- `backend/internal/application.go` — wire `rsvp.NewService`

**Frontend — create:**
- `frontend/app/me/practices/page.tsx`
- `frontend/app/me/applications/page.tsx`
- `frontend/components/EventApplicationsPanel.tsx` — organizer accept/decline (used on `/events/mine` event)

**Frontend — modify:**
- `frontend/lib/api.ts` — RSVP calls
- `frontend/lib/types.ts` — RSVP types + event signup fields
- `frontend/app/events/[id]/page.tsx` (and/or its CTA component) — signup CTA + states

---

## Domain reference (names later tasks depend on)

**`RsvpStatus`** (`models.RsvpStatus`, a `string`):
`RsvpGoing="going"`, `RsvpWaitlist="waitlist"`, `RsvpApplied="applied"`, `RsvpAccepted="accepted"`, `RsvpDeclined="declined"`, `RsvpWithdrawn="withdrawn"`, `RsvpCancelled="cancelled"`.

**Signup modes** (event `SignupMode string`): `"open"`, `"application"`, `"external"`.

**`rsvp.Service`** interface (Task 6):
```go
type Service interface {
    SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) (*models.Rsvp, error)
    Cancel(ctx context.Context, eventID, userID uuid.UUID) error
    MyPractices(ctx context.Context, userID uuid.UUID, tab string) ([]*PracticeRow, error)
    MyApplications(ctx context.Context, userID uuid.UUID, status string) ([]*models.Rsvp, error)
    ListApplications(ctx context.Context, eventID, organizerID uuid.UUID) ([]*models.Rsvp, error)
    Decide(ctx context.Context, eventID, organizerID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
    CalendarICS(ctx context.Context, eventID uuid.UUID) ([]byte, error)
}
```
Domain errors: `ErrInvalidInput`, `ErrNotFound`, `ErrConflict` (duplicate / wrong state), `ErrForbidden` (not organizer), `ErrExternal` (external mode — carries the URL).

---

## Task 1: Migration — event signup fields

**Files:**
- Create: `backend/db/migrations/000012_event_signup_fields.up.sql`
- Create: `backend/db/migrations/000012_event_signup_fields.down.sql`

**Interfaces:**
- Produces: columns `events.signup_mode` (text), `events.capacity` (int null), `events.curator_question` (text), `events.external_registration_url` (text).

- [ ] **Step 1: Write the up migration**

`backend/db/migrations/000012_event_signup_fields.up.sql`:
```sql
ALTER TABLE events ADD COLUMN IF NOT EXISTS signup_mode text NOT NULL DEFAULT 'open';
ALTER TABLE events ADD COLUMN IF NOT EXISTS capacity integer;
ALTER TABLE events ADD COLUMN IF NOT EXISTS curator_question text NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN IF NOT EXISTS external_registration_url text NOT NULL DEFAULT '';

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_signup_mode_check;
ALTER TABLE events ADD CONSTRAINT events_signup_mode_check
  CHECK (signup_mode IN ('open', 'application', 'external'));
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_capacity_check;
ALTER TABLE events ADD CONSTRAINT events_capacity_check
  CHECK (capacity IS NULL OR capacity > 0);
```

- [ ] **Step 2: Write the down migration**

`backend/db/migrations/000012_event_signup_fields.down.sql`:
```sql
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_signup_mode_check;
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_capacity_check;
ALTER TABLE events DROP COLUMN IF EXISTS external_registration_url;
ALTER TABLE events DROP COLUMN IF EXISTS curator_question;
ALTER TABLE events DROP COLUMN IF EXISTS capacity;
ALTER TABLE events DROP COLUMN IF EXISTS signup_mode;
```

- [ ] **Step 3: Commit**

```bash
git add backend/db/migrations/000012_event_signup_fields.*.sql
git commit -m "feat(rsvp): migration for event signup fields"
```

---

## Task 2: Migration — event_rsvps table

**Files:**
- Create: `backend/db/migrations/000013_event_rsvps.up.sql`
- Create: `backend/db/migrations/000013_event_rsvps.down.sql`

**Interfaces:**
- Produces: table `event_rsvps` with `(event_id, user_id)` unique and `(event_id, status)` index.

- [ ] **Step 1: Write the up migration**

`backend/db/migrations/000013_event_rsvps.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS event_rsvps (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id           uuid NOT NULL,
    user_id            uuid NOT NULL,
    status             text NOT NULL,
    application_answer text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT event_rsvps_status_check CHECK (
        status IN ('going','waitlist','applied','accepted','declined','withdrawn','cancelled')
    ),
    CONSTRAINT event_rsvps_event_user_unique UNIQUE (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS event_rsvps_event_status_idx ON event_rsvps (event_id, status);
CREATE INDEX IF NOT EXISTS event_rsvps_user_idx ON event_rsvps (user_id);
```

> `gen_random_uuid()` is available (pgcrypto/pg13+). If a target DB lacks it, the model's `BeforeInsert` also sets the id Go-side (Task 3), so inserts still work.

- [ ] **Step 2: Write the down migration**

`backend/db/migrations/000013_event_rsvps.down.sql`:
```sql
DROP TABLE IF EXISTS event_rsvps;
```

- [ ] **Step 3: Commit**

```bash
git add backend/db/migrations/000013_event_rsvps.*.sql
git commit -m "feat(rsvp): migration for event_rsvps table"
```

---

## Task 3: Rsvp model + RsvpStatus

**Files:**
- Create: `backend/internal/models/rsvp.go`
- Test: `backend/internal/models/rsvp_test.go`

**Interfaces:**
- Produces: `models.Rsvp` struct, `models.RsvpStatus` constants (see Domain reference), `Rsvp.Validate()`, `Rsvp.BeforeInsert`.

- [ ] **Step 1: Write the failing test**

`backend/internal/models/rsvp_test.go`:
```go
package models

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

func TestRsvpValidate(t *testing.T) {
	good := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: RsvpGoing}
	if err := good.Validate(); err != nil {
		t.Fatalf("expected valid rsvp, got %v", err)
	}

	noEvent := &Rsvp{UserID: uuid.Must(uuid.NewV4()), Status: RsvpGoing}
	if err := noEvent.Validate(); err == nil {
		t.Fatal("expected error when event_id is missing")
	}

	badStatus := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: "bogus"}
	if err := badStatus.Validate(); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestRsvpBeforeInsertSetsID(t *testing.T) {
	r := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: RsvpApplied}
	if _, err := r.BeforeInsert(context.Background()); err != nil {
		t.Fatalf("BeforeInsert: %v", err)
	}
	if r.ID == uuid.Nil {
		t.Fatal("expected ID to be generated")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -vet=off ./internal/models/ -run TestRsvp`
Expected: FAIL (undefined `Rsvp`, `RsvpGoing`, …).

- [ ] **Step 3: Write the model**

`backend/internal/models/rsvp.go`:
```go
package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// RsvpStatus is the lifecycle state of a user's registration on an event.
// Stored verbatim as text (CHECK-constrained in db/migrations/000013).
type RsvpStatus string

const (
	// RsvpGoing — confirmed seat (open mode, or accepted application within capacity).
	RsvpGoing RsvpStatus = "going"
	// RsvpWaitlist — registered but no seat free; promoted FIFO when one frees up.
	RsvpWaitlist RsvpStatus = "waitlist"
	// RsvpApplied — application submitted, awaiting the organizer's decision.
	RsvpApplied RsvpStatus = "applied"
	// RsvpAccepted — application accepted by the organizer.
	RsvpAccepted RsvpStatus = "accepted"
	// RsvpDeclined — application declined by the organizer.
	RsvpDeclined RsvpStatus = "declined"
	// RsvpWithdrawn — applicant withdrew before a decision.
	RsvpWithdrawn RsvpStatus = "withdrawn"
	// RsvpCancelled — user cancelled a going/waitlist registration.
	RsvpCancelled RsvpStatus = "cancelled"
)

// validRsvpStatuses is the set accepted by Validate (mirrors the DB CHECK).
var validRsvpStatuses = map[RsvpStatus]struct{}{
	RsvpGoing: {}, RsvpWaitlist: {}, RsvpApplied: {}, RsvpAccepted: {},
	RsvpDeclined: {}, RsvpWithdrawn: {}, RsvpCancelled: {},
}

// IsActive reports whether the status represents a live registration (counts
// toward attendance lists). Terminal statuses (declined/withdrawn/cancelled) are not active.
func (s RsvpStatus) IsActive() bool {
	switch s {
	case RsvpGoing, RsvpWaitlist, RsvpApplied, RsvpAccepted:
		return true
	default:
		return false
	}
}

// Rsvp is a user's registration on an event.
//
//nolint:govet // field alignment kept for readability
type Rsvp struct {
	tableName struct{} `pg:"event_rsvps,discard_unknown_columns"` //nolint:unused

	ID                uuid.UUID  `pg:"id,pk,type:uuid"`
	EventID           uuid.UUID  `pg:"event_id,type:uuid,use_zero"`
	UserID            uuid.UUID  `pg:"user_id,type:uuid,use_zero"`
	Status            RsvpStatus `pg:"status,use_zero"`
	ApplicationAnswer string     `pg:"application_answer,use_zero"`
	CreatedAt         time.Time  `pg:"created_at,notnull,default:now()"`
	UpdatedAt         time.Time  `pg:"updated_at,notnull,default:now()"`

	// Event is a transient read-model populated by joins (e.g. MyPractices).
	Event *Event `pg:"-"`
}

// Validate checks required fields and a known status.
func (r *Rsvp) Validate() error {
	if r.EventID == uuid.Nil {
		return newValidationError("event_id", "is required")
	}
	if r.UserID == uuid.Nil {
		return newValidationError("user_id", "is required")
	}
	if _, ok := validRsvpStatuses[r.Status]; !ok {
		return newValidationError("status", "invalid value")
	}
	return nil
}

// BeforeInsert generates an ID when missing and stamps timestamps.
func (r *Rsvp) BeforeInsert(ctx context.Context) (context.Context, error) {
	if r.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		r.ID = id
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	return ctx, nil
}

// BeforeUpdate refreshes the updated_at timestamp.
func (r *Rsvp) BeforeUpdate(ctx context.Context) (context.Context, error) {
	r.UpdatedAt = time.Now()
	return ctx, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test -vet=off ./internal/models/ -run TestRsvp`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/models/rsvp.go backend/internal/models/rsvp_test.go
git commit -m "feat(rsvp): Rsvp model + RsvpStatus"
```

---

## Task 4: Event model — signup fields

**Files:**
- Modify: `backend/internal/models/event.go`
- Modify: `backend/internal/events/repository.go` (Update column list)
- Test: `backend/internal/models/event_test.go` (add cases)

**Interfaces:**
- Produces: `Event.SignupMode string`, `Event.Capacity *int`, `Event.CuratorQuestion string`, `Event.ExternalRegistrationURL string`, plus transient `Event.SeatsRemaining *int` and `Event.MyRsvpStatus string` (populated by Task 8, not columns).

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/models/event_test.go`:
```go
func TestEventValidateSignupMode(t *testing.T) {
	base := func() *Event {
		return &Event{Title: "x", StartsAt: time.Now(), Status: EventPublished, SignupMode: "open"}
	}
	if err := base().Validate(); err != nil {
		t.Fatalf("open mode should be valid: %v", err)
	}

	app := base()
	app.SignupMode = "application"
	app.CuratorQuestion = ""
	if err := app.Validate(); err == nil {
		t.Fatal("application mode without curator_question should fail")
	}
	app.CuratorQuestion = "почему вам интересно?"
	if err := app.Validate(); err != nil {
		t.Fatalf("application mode with question should pass: %v", err)
	}

	ext := base()
	ext.SignupMode = "external"
	if err := ext.Validate(); err == nil {
		t.Fatal("external mode without url should fail")
	}
	ext.ExternalRegistrationURL = "https://org.example/signup"
	if err := ext.Validate(); err != nil {
		t.Fatalf("external mode with url should pass: %v", err)
	}

	bad := base()
	bad.SignupMode = "bogus"
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown signup_mode should fail")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -vet=off ./internal/models/ -run TestEventValidateSignupMode`
Expected: FAIL (unknown fields / no validation).

- [ ] **Step 3: Add fields to the Event struct**

In `backend/internal/models/event.go`, add after the `ExternalURL` field (line ~49):
```go
	// Signup configuration (migration 000012).
	SignupMode              string `pg:"signup_mode,use_zero"`              // open | application | external
	Capacity                *int   `pg:"capacity"`                          // nil = unlimited
	CuratorQuestion         string `pg:"curator_question,use_zero"`         // required when application
	ExternalRegistrationURL string `pg:"external_registration_url,use_zero"` // required when external

	// Transient (not columns): populated by the repository per request.
	SeatsRemaining *int   `pg:"-"` // nil = unlimited; computed = capacity - going
	MyRsvpStatus   string `pg:"-"` // caller's RsvpStatus on this event, "" if none
```

- [ ] **Step 4: Extend Validate**

In `Event.Validate()` (before the final `return nil`):
```go
	switch e.SignupMode {
	case "", "open":
		// ok; empty defaults to open at the DB layer
	case "application":
		if e.CuratorQuestion == "" {
			return newValidationError("curator_question", "is required for application signup")
		}
	case "external":
		if e.ExternalRegistrationURL == "" {
			return newValidationError("external_registration_url", "is required for external signup")
		}
	default:
		return newValidationError("signup_mode", "invalid value")
	}
	if e.Capacity != nil && *e.Capacity <= 0 {
		return newValidationError("capacity", "must be positive")
	}
```

- [ ] **Step 5: Persist new columns on Create/Update**

`Create` uses `tx.Model(event).Insert()` which inserts all non-`pg:"-"` columns, so the new columns insert automatically. For `Update`, add the columns to the explicit `Column(...)` list in `backend/internal/events/repository.go` (the `Update` method, ~line 112):
```go
				"external_ticket_url", "starts_at", "ends_at", "published_at",
				"signup_mode", "capacity", "curator_question", "external_registration_url",
				"updated_at",
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd backend && go test -vet=off ./internal/models/ -run TestEventValidateSignupMode`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/models/event.go backend/internal/models/event_test.go backend/internal/events/repository.go
git commit -m "feat(rsvp): event signup_mode/capacity fields + validation"
```

---

## Task 5: RSVP repository

**Files:**
- Create: `backend/internal/rsvp/repository.go`
- Modify: `backend/internal/rsvp/doc.go` (drop "Status: skeleton")
- Test: `backend/internal/rsvp/repository_test.go` (pure-logic helpers only; DB methods are integration-covered via the service fake in Task 6)

**Interfaces:**
- Consumes: `models.Rsvp`, `models.Event`.
- Produces: `rsvp.Repository` interface + `pgRepository`:
```go
type Repository interface {
    GetEvent(id uuid.UUID) (*models.Event, error)
    GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error)   // pg.ErrNoRows when none
    GetRsvpByID(id uuid.UUID) (*models.Rsvp, error)
    CountActiveSeats(eventID uuid.UUID) (int, error)               // status = going
    // SignUpTx allocates a seat or waitlist slot atomically, honoring capacity.
    // It re-activates an existing (event,user) row when present (terminal → new status).
    SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error)
    // CancelTx terminates the caller's active row and, if a going seat was freed,
    // promotes the oldest waitlist row to going. Returns ErrNoRows when nothing active.
    CancelTx(eventID, userID uuid.UUID) error
    // DecideTx sets an applied row to accepted (or waitlist if full) / declined.
    DecideTx(rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
    ListByUser(userID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error) // joins Event
    ListByEvent(eventID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error)
}

// SeatDecider returns the status to assign given seats currently taken and capacity
// (nil capacity = unlimited). Lets the service encode open-vs-application policy.
type SeatDecider func(seatsTaken int, capacity *int) models.RsvpStatus
```

- [ ] **Step 1: Write the failing test (seat-decision helper)**

`backend/internal/rsvp/repository_test.go`:
```go
package rsvp

import (
	"testing"

	"github.com/Pashteto/lia/internal/models"
)

func TestSeatAvailable(t *testing.T) {
	cap2 := 2
	cases := []struct {
		name     string
		taken    int
		capacity *int
		want     bool
	}{
		{"unlimited", 100, nil, true},
		{"room left", 1, &cap2, true},
		{"exactly full", 2, &cap2, false},
		{"over full", 3, &cap2, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := seatAvailable(c.taken, c.capacity); got != c.want {
				t.Fatalf("seatAvailable(%d,%v)=%v want %v", c.taken, c.capacity, got, c.want)
			}
		})
	}
	_ = models.RsvpGoing
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -vet=off ./internal/rsvp/ -run TestSeatAvailable`
Expected: FAIL (undefined `seatAvailable`).

- [ ] **Step 3: Write the repository**

`backend/internal/rsvp/repository.go` (mirror `events/repository.go` style — `pg.DB`, `RunInTransaction`, `Order`, `pg.In`):
```go
package rsvp

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// SeatDecider returns the status to assign given seats taken and capacity.
type SeatDecider func(seatsTaken int, capacity *int) models.RsvpStatus

// seatAvailable reports whether one more going seat fits under capacity.
// A nil capacity means unlimited.
func seatAvailable(seatsTaken int, capacity *int) bool {
	if capacity == nil {
		return true
	}
	return seatsTaken < *capacity
}

// Repository defines RSVP persistence.
type Repository interface {
	GetEvent(id uuid.UUID) (*models.Event, error)
	GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error)
	GetRsvpByID(id uuid.UUID) (*models.Rsvp, error)
	CountActiveSeats(eventID uuid.UUID) (int, error)
	SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error)
	CancelTx(eventID, userID uuid.UUID) error
	DecideTx(rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
	ListByUser(userID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error)
	ListByEvent(eventID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error)
}

type pgRepository struct{ db *pg.DB }

// NewRepository creates a PostgreSQL-backed RSVP repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) GetEvent(id uuid.UUID) (*models.Event, error) {
	e := &models.Event{ID: id}
	if err := r.db.Model(e).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get event %s: %w", id, err)
	}
	return e, nil
}

func (r *pgRepository) GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error) {
	out := new(models.Rsvp)
	if err := r.db.Model(out).
		Where("event_id = ? AND user_id = ?", eventID, userID).Select(); err != nil {
		return nil, fmt.Errorf("get rsvp for event %s user %s: %w", eventID, userID, err)
	}
	return out, nil
}

func (r *pgRepository) GetRsvpByID(id uuid.UUID) (*models.Rsvp, error) {
	out := &models.Rsvp{ID: id}
	if err := r.db.Model(out).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get rsvp %s: %w", id, err)
	}
	return out, nil
}

func (r *pgRepository) CountActiveSeats(eventID uuid.UUID) (int, error) {
	n, err := r.db.Model((*models.Rsvp)(nil)).
		Where("event_id = ? AND status = ?", eventID, models.RsvpGoing).Count()
	if err != nil {
		return 0, fmt.Errorf("count seats for event %s: %w", eventID, err)
	}
	return n, nil
}

// SignUpTx locks the event row, counts going seats, and inserts/re-activates the
// caller's row with the status chosen by decide(). UNIQUE(event,user) guarantees
// one row; a prior terminal row is transitioned in place.
func (r *pgRepository) SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error) {
	var result *models.Rsvp
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		// Lock the event row to serialize concurrent sign-ups (last-seat race).
		var capacity *int
		if _, err := tx.QueryOne(pg.Scan(&capacity),
			`SELECT capacity FROM events WHERE id = ? FOR UPDATE`, eventID); err != nil {
			return fmt.Errorf("lock event %s: %w", eventID, err)
		}

		seats, err := tx.Model((*models.Rsvp)(nil)).
			Where("event_id = ? AND status = ?", eventID, models.RsvpGoing).Count()
		if err != nil {
			return fmt.Errorf("count seats: %w", err)
		}
		status := decide(seats, capacity)

		existing := new(models.Rsvp)
		err = tx.Model(existing).Where("event_id = ? AND user_id = ?", eventID, userID).Select()
		switch {
		case err == nil:
			if existing.Status.IsActive() {
				return ErrConflict // already registered/applied
			}
			existing.Status = status
			existing.ApplicationAnswer = answer
			if _, uerr := tx.Model(existing).
				Column("status", "application_answer", "updated_at").WherePK().Update(); uerr != nil {
				return fmt.Errorf("reactivate rsvp: %w", uerr)
			}
			result = existing
		case errors.Is(err, pg.ErrNoRows):
			row := &models.Rsvp{EventID: eventID, UserID: userID, Status: status, ApplicationAnswer: answer}
			if _, ierr := tx.Model(row).Insert(); ierr != nil {
				return fmt.Errorf("insert rsvp: %w", ierr)
			}
			result = row
		default:
			return fmt.Errorf("select existing rsvp: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CancelTx terminates the caller's active row. If a going seat was freed, the
// oldest waitlist row (by created_at) is promoted to going — same transaction.
func (r *pgRepository) CancelTx(eventID, userID uuid.UUID) error {
	return r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		if _, err := tx.Exec(`SELECT id FROM events WHERE id = ? FOR UPDATE`, eventID); err != nil {
			return fmt.Errorf("lock event %s: %w", eventID, err)
		}
		cur := new(models.Rsvp)
		if err := tx.Model(cur).Where("event_id = ? AND user_id = ?", eventID, userID).Select(); err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				return pg.ErrNoRows
			}
			return fmt.Errorf("select rsvp: %w", err)
		}
		if !cur.Status.IsActive() {
			return pg.ErrNoRows // nothing active to cancel
		}
		freedSeat := cur.Status == models.RsvpGoing || cur.Status == models.RsvpAccepted

		newStatus := models.RsvpCancelled
		if cur.Status == models.RsvpApplied || cur.Status == models.RsvpAccepted {
			newStatus = models.RsvpWithdrawn
		}
		cur.Status = newStatus
		if _, err := tx.Model(cur).Column("status", "updated_at").WherePK().Update(); err != nil {
			return fmt.Errorf("cancel rsvp: %w", err)
		}

		if freedSeat {
			next := new(models.Rsvp)
			err := tx.Model(next).
				Where("event_id = ? AND status = ?", eventID, models.RsvpWaitlist).
				Order("created_at ASC").Limit(1).Select()
			if err == nil {
				next.Status = models.RsvpGoing
				if _, uerr := tx.Model(next).Column("status", "updated_at").WherePK().Update(); uerr != nil {
					return fmt.Errorf("promote waitlist: %w", uerr)
				}
				logger.Log().Infof("rsvp: promoted %s from waitlist on event %s", next.ID, eventID)
			} else if !errors.Is(err, pg.ErrNoRows) {
				return fmt.Errorf("find waitlist head: %w", err)
			}
		}
		return nil
	})
}

// DecideTx accepts (or waitlists if full) / declines an applied row.
func (r *pgRepository) DecideTx(rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	var result *models.Rsvp
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		row := &models.Rsvp{ID: rsvpID}
		if err := tx.Model(row).WherePK().Select(); err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				return pg.ErrNoRows
			}
			return fmt.Errorf("select rsvp %s: %w", rsvpID, err)
		}
		if row.Status != models.RsvpApplied {
			return ErrConflict // already decided / not an application
		}
		if !accept {
			row.Status = models.RsvpDeclined
		} else {
			var capacity *int
			if _, err := tx.QueryOne(pg.Scan(&capacity),
				`SELECT capacity FROM events WHERE id = ? FOR UPDATE`, row.EventID); err != nil {
				return fmt.Errorf("lock event: %w", err)
			}
			seats, err := tx.Model((*models.Rsvp)(nil)).
				Where("event_id = ? AND status = ?", row.EventID, models.RsvpGoing).Count()
			if err != nil {
				return fmt.Errorf("count seats: %w", err)
			}
			if seatAvailable(seats, capacity) {
				row.Status = models.RsvpAccepted
			} else {
				row.Status = models.RsvpWaitlist
			}
		}
		if _, err := tx.Model(row).Column("status", "updated_at").WherePK().Update(); err != nil {
			return fmt.Errorf("update decision: %w", err)
		}
		result = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *pgRepository) ListByUser(userID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error) {
	var rows []*models.Rsvp
	q := r.db.Model(&rows).Where("user_id = ?", userID)
	if len(statuses) > 0 {
		q = q.Where("status IN (?)", pg.In(statuses))
	}
	if err := q.Order("created_at DESC").Select(); err != nil {
		return nil, fmt.Errorf("list rsvps for user %s: %w", userID, err)
	}
	return r.attachEvents(rows)
}

func (r *pgRepository) ListByEvent(eventID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error) {
	var rows []*models.Rsvp
	q := r.db.Model(&rows).Where("event_id = ?", eventID)
	if len(statuses) > 0 {
		q = q.Where("status IN (?)", pg.In(statuses))
	}
	if err := q.Order("created_at ASC").Select(); err != nil {
		return nil, fmt.Errorf("list rsvps for event %s: %w", eventID, err)
	}
	return rows, nil
}

// attachEvents batch-loads the Event for each rsvp (no N+1).
func (r *pgRepository) attachEvents(rows []*models.Rsvp) ([]*models.Rsvp, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	seen := map[uuid.UUID]struct{}{}
	for _, row := range rows {
		if _, ok := seen[row.EventID]; !ok {
			seen[row.EventID] = struct{}{}
			ids = append(ids, row.EventID)
		}
	}
	var events []*models.Event
	if err := r.db.Model(&events).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return nil, fmt.Errorf("attach events: %w", err)
	}
	byID := make(map[uuid.UUID]*models.Event, len(events))
	for _, e := range events {
		byID[e.ID] = e
	}
	for _, row := range rows {
		row.Event = byID[row.EventID]
	}
	return rows, nil
}
```

- [ ] **Step 4: Update the package doc**

In `backend/internal/rsvp/doc.go`, change the last paragraph from `// Status: skeleton. Implement following...` to:
```go
// Implemented following the events module pattern (internal/events).
// See docs/superpowers/specs/2026-06-26-rsvp-and-attendance-design.md.
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test -vet=off ./internal/rsvp/ -run TestSeatAvailable`
Expected: PASS. Then `cd backend && go build ./internal/rsvp/` — Expected: builds (note: `ErrConflict` is defined in Task 6; build of the full package completes after that task — if building Task 5 alone, temporarily expect an undefined `ErrConflict` until Task 6 adds errors. Sequence Task 6 immediately after.)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/rsvp/repository.go backend/internal/rsvp/repository_test.go backend/internal/rsvp/doc.go
git commit -m "feat(rsvp): repository with transactional seat + waitlist logic"
```

---

## Task 6: RSVP service

**Files:**
- Create: `backend/internal/rsvp/service.go`
- Test: `backend/internal/rsvp/service_test.go`

**Interfaces:**
- Consumes: `rsvp.Repository`, `models.Rsvp`, `models.Event`, `seatAvailable`/`SeatDecider`.
- Produces: the `rsvp.Service` interface (see Domain reference), `NewService(repo Repository) Service`, errors `ErrInvalidInput`, `ErrNotFound`, `ErrConflict`, `ErrForbidden`, `ErrExternal`, and `PracticeRow{ Rsvp *models.Rsvp; Event *models.Event }`.

- [ ] **Step 1: Write the failing tests (with a fake repository)**

`backend/internal/rsvp/service_test.go`:
```go
package rsvp

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/go-pg/pg/v10"
	"github.com/Pashteto/lia/internal/models"
)

// fakeRepo is an in-memory Repository for service tests.
type fakeRepo struct {
	event   *models.Event
	rsvps   map[uuid.UUID]*models.Rsvp // by rsvp id
	seats   int                        // current going count
}

func newFake(e *models.Event) *fakeRepo {
	return &fakeRepo{event: e, rsvps: map[uuid.UUID]*models.Rsvp{}}
}

func (f *fakeRepo) GetEvent(id uuid.UUID) (*models.Event, error) {
	if f.event == nil || f.event.ID != id {
		return nil, pg.ErrNoRows
	}
	return f.event, nil
}
func (f *fakeRepo) GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error) {
	for _, r := range f.rsvps {
		if r.EventID == eventID && r.UserID == userID {
			return r, nil
		}
	}
	return nil, pg.ErrNoRows
}
func (f *fakeRepo) GetRsvpByID(id uuid.UUID) (*models.Rsvp, error) {
	if r, ok := f.rsvps[id]; ok {
		return r, nil
	}
	return nil, pg.ErrNoRows
}
func (f *fakeRepo) CountActiveSeats(uuid.UUID) (int, error) { return f.seats, nil }
func (f *fakeRepo) SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error) {
	if r, _ := f.GetUserRsvp(eventID, userID); r != nil && r.Status.IsActive() {
		return nil, ErrConflict
	}
	status := decide(f.seats, f.event.Capacity)
	if status == models.RsvpGoing {
		f.seats++
	}
	row := &models.Rsvp{ID: uuid.Must(uuid.NewV4()), EventID: eventID, UserID: userID, Status: status, ApplicationAnswer: answer}
	f.rsvps[row.ID] = row
	return row, nil
}
func (f *fakeRepo) CancelTx(eventID, userID uuid.UUID) error {
	r, err := f.GetUserRsvp(eventID, userID)
	if err != nil || !r.Status.IsActive() {
		return pg.ErrNoRows
	}
	if r.Status == models.RsvpGoing {
		f.seats--
		// promote oldest waitlist
		for _, w := range f.rsvps {
			if w.EventID == eventID && w.Status == models.RsvpWaitlist {
				w.Status = models.RsvpGoing
				f.seats++
				break
			}
		}
	}
	r.Status = models.RsvpCancelled
	return nil
}
func (f *fakeRepo) DecideTx(rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	r, ok := f.rsvps[rsvpID]
	if !ok {
		return nil, pg.ErrNoRows
	}
	if r.Status != models.RsvpApplied {
		return nil, ErrConflict
	}
	if !accept {
		r.Status = models.RsvpDeclined
	} else if seatAvailable(f.seats, f.event.Capacity) {
		r.Status = models.RsvpAccepted
	} else {
		r.Status = models.RsvpWaitlist
	}
	return r, nil
}
func (f *fakeRepo) ListByUser(userID uuid.UUID, st []models.RsvpStatus) ([]*models.Rsvp, error) {
	var out []*models.Rsvp
	for _, r := range f.rsvps {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListByEvent(eventID uuid.UUID, st []models.RsvpStatus) ([]*models.Rsvp, error) {
	var out []*models.Rsvp
	for _, r := range f.rsvps {
		if r.EventID == eventID {
			out = append(out, r)
		}
	}
	return out, nil
}

func openEvent(cap *int) *models.Event {
	return &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "open", Capacity: cap}
}

func TestSignUpOpenFillsThenWaitlists(t *testing.T) {
	cap1 := 1
	e := openEvent(&cap1)
	svc := NewService(newFake(e))
	r1, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if err != nil || r1.Status != models.RsvpGoing {
		t.Fatalf("first signup want going, got %v err %v", r1, err)
	}
	r2, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if err != nil || r2.Status != models.RsvpWaitlist {
		t.Fatalf("second signup want waitlist, got %v err %v", r2, err)
	}
}

func TestSignUpDuplicateConflicts(t *testing.T) {
	e := openEvent(nil)
	svc := NewService(newFake(e))
	u := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, u, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SignUp(context.Background(), e.ID, u, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCancelPromotesWaitlist(t *testing.T) {
	cap1 := 1
	e := openEvent(&cap1)
	f := newFake(e)
	svc := NewService(f)
	uGoing := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, uGoing, ""); err != nil {
		t.Fatal(err)
	}
	uWait := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, uWait, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Cancel(context.Background(), e.ID, uGoing); err != nil {
		t.Fatal(err)
	}
	promoted, _ := f.GetUserRsvp(e.ID, uWait)
	if promoted.Status != models.RsvpGoing {
		t.Fatalf("waitlisted user should be promoted, got %s", promoted.Status)
	}
}

func TestSignUpExternalReturnsErrExternal(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), SignupMode: "external", ExternalRegistrationURL: "https://x"}
	svc := NewService(newFake(e))
	_, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if !errors.Is(err, ErrExternal) {
		t.Fatalf("want ErrExternal, got %v", err)
	}
}

func TestApplicationDecideAcceptDecline(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application", CuratorQuestion: "?"}
	f := newFake(e)
	svc := NewService(f)
	app, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "хочу прийти")
	if err != nil || app.Status != models.RsvpApplied {
		t.Fatalf("want applied, got %v err %v", app, err)
	}
	got, err := svc.Decide(context.Background(), e.ID, e.OrganizerID, app.ID, true)
	if err != nil || got.Status != models.RsvpAccepted {
		t.Fatalf("accept want accepted, got %v err %v", got, err)
	}
}

func TestDecideByNonOrganizerForbidden(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application", CuratorQuestion: "?"}
	f := newFake(e)
	svc := NewService(f)
	app, _ := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "x")
	_, err := svc.Decide(context.Background(), e.ID, uuid.Must(uuid.NewV4()), app.ID, true)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test -vet=off ./internal/rsvp/`
Expected: FAIL (undefined `NewService`, `ErrExternal`, …).

- [ ] **Step 3: Write the service**

`backend/internal/rsvp/service.go`:
```go
package rsvp

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// Domain errors. The HTTP layer maps these to status codes.
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")  // duplicate registration / wrong state
	ErrForbidden    = errors.New("forbidden") // caller is not the event organizer
	ErrExternal     = errors.New("external registration") // signup happens on organizer's site
)

// PracticeRow is one attendance row for /me/practices: the rsvp plus its event.
type PracticeRow struct {
	Rsvp  *models.Rsvp
	Event *models.Event
}

// Service is the RSVP business-logic interface.
type Service interface {
	SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) (*models.Rsvp, error)
	Cancel(ctx context.Context, eventID, userID uuid.UUID) error
	MyPractices(ctx context.Context, userID uuid.UUID, tab string) ([]*PracticeRow, error)
	MyApplications(ctx context.Context, userID uuid.UUID, status string) ([]*models.Rsvp, error)
	ListApplications(ctx context.Context, eventID, organizerID uuid.UUID) ([]*models.Rsvp, error)
	Decide(ctx context.Context, eventID, organizerID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
	CalendarICS(ctx context.Context, eventID uuid.UUID) ([]byte, error)
}

type service struct{ repo Repository }

// NewService creates an RSVP service backed by the given repository.
func NewService(repo Repository) Service { return &service{repo: repo} }

func isNoRows(err error) bool { return errors.Is(err, pg.ErrNoRows) }

func (s *service) SignUp(_ context.Context, eventID, userID uuid.UUID, answer string) (*models.Rsvp, error) {
	if eventID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("%w: event and user are required", ErrInvalidInput)
	}
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}

	switch event.SignupMode {
	case "external":
		// Caller registers on the organizer's site; surface the URL via the error.
		return nil, fmt.Errorf("%w: %s", ErrExternal, event.ExternalRegistrationURL)
	case "application":
		row, err := s.repo.SignUpTx(eventID, userID,
			func(int, *int) models.RsvpStatus { return models.RsvpApplied }, answer)
		return wrapSignupErr(row, err)
	default: // "" or "open"
		row, err := s.repo.SignUpTx(eventID, userID, openSeatDecider, "")
		return wrapSignupErr(row, err)
	}
}

// openSeatDecider gives a going seat when capacity allows, else waitlist.
func openSeatDecider(seatsTaken int, capacity *int) models.RsvpStatus {
	if seatAvailable(seatsTaken, capacity) {
		return models.RsvpGoing
	}
	return models.RsvpWaitlist
}

func wrapSignupErr(row *models.Rsvp, err error) (*models.Rsvp, error) {
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("sign up: %w", err)
	}
	return row, nil
}

func (s *service) Cancel(_ context.Context, eventID, userID uuid.UUID) error {
	if err := s.repo.CancelTx(eventID, userID); err != nil {
		if isNoRows(err) {
			return fmt.Errorf("%w: no active registration", ErrNotFound)
		}
		return fmt.Errorf("cancel: %w", err)
	}
	return nil
}

func (s *service) MyPractices(_ context.Context, userID uuid.UUID, tab string) ([]*PracticeRow, error) {
	rows, err := s.repo.ListByUser(userID,
		[]models.RsvpStatus{models.RsvpGoing, models.RsvpWaitlist, models.RsvpAccepted})
	if err != nil {
		return nil, fmt.Errorf("my practices: %w", err)
	}
	out := make([]*PracticeRow, 0, len(rows))
	for _, r := range rows {
		if r.Event == nil {
			continue
		}
		isPast := r.Event.StartsAt.Before(nowFn())
		if tab == "past" && !isPast {
			continue
		}
		if tab != "past" && isPast {
			continue
		}
		out = append(out, &PracticeRow{Rsvp: r, Event: r.Event})
	}
	return out, nil
}

func (s *service) MyApplications(_ context.Context, userID uuid.UUID, status string) ([]*models.Rsvp, error) {
	want := []models.RsvpStatus{models.RsvpApplied, models.RsvpAccepted, models.RsvpDeclined, models.RsvpWithdrawn}
	if status != "" {
		want = []models.RsvpStatus{models.RsvpStatus(status)}
	}
	rows, err := s.repo.ListByUser(userID, want)
	if err != nil {
		return nil, fmt.Errorf("my applications: %w", err)
	}
	return rows, nil
}

func (s *service) ListApplications(_ context.Context, eventID, organizerID uuid.UUID) ([]*models.Rsvp, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return nil, fmt.Errorf("%w: not the organizer", ErrForbidden)
	}
	rows, err := s.repo.ListByEvent(eventID,
		[]models.RsvpStatus{models.RsvpApplied, models.RsvpAccepted, models.RsvpDeclined})
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	return rows, nil
}

func (s *service) Decide(_ context.Context, eventID, organizerID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return nil, fmt.Errorf("%w: not the organizer", ErrForbidden)
	}
	row, err := s.repo.DecideTx(rsvpID, accept)
	if err != nil {
		switch {
		case isNoRows(err):
			return nil, fmt.Errorf("%w: rsvp %s", ErrNotFound, rsvpID)
		case errors.Is(err, ErrConflict):
			return nil, err
		default:
			return nil, fmt.Errorf("decide: %w", err)
		}
	}
	if row.EventID != eventID {
		return nil, fmt.Errorf("%w: rsvp does not belong to event", ErrInvalidInput)
	}
	logger.Log().Infof("rsvp %s decided accept=%v -> %s", rsvpID, accept, row.Status)
	return row, nil
}

func (s *service) CalendarICS(_ context.Context, eventID uuid.UUID) ([]byte, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.StartsAt.IsZero() {
		return nil, fmt.Errorf("%w: event has no start time", ErrInvalidInput)
	}
	return buildICS(event), nil
}
```

Add a package-level `nowFn` indirection so tests are deterministic. Append to `service.go`:
```go
// nowFn returns the current time; overridable in tests.
var nowFn = func() time.Time { return time.Now() }
```
and add `"time"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test -vet=off ./internal/rsvp/`
Expected: PASS. (`buildICS` is added in Task 7; if running before Task 7, stub it as `func buildICS(*models.Event) []byte { return nil }` in `service.go` temporarily, then move it to `ics.go` in Task 7. Recommended: do Task 7 immediately after.)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/rsvp/service.go backend/internal/rsvp/service_test.go
git commit -m "feat(rsvp): service with signup/cancel/decide/practices logic"
```

---

## Task 7: .ics generator

**Files:**
- Create: `backend/internal/rsvp/ics.go`
- Test: `backend/internal/rsvp/ics_test.go`

**Interfaces:**
- Consumes: `models.Event`.
- Produces: `func buildICS(e *models.Event) []byte`.

- [ ] **Step 1: Write the failing test**

`backend/internal/rsvp/ics_test.go`:
```go
package rsvp

import (
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

func TestBuildICS(t *testing.T) {
	start := time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC)
	e := &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		Title:       "Чтение вслух",
		Description: "встреча",
		StartsAt:    start,
	}
	out := string(buildICS(e))
	for _, want := range []string{
		"BEGIN:VCALENDAR", "BEGIN:VEVENT", "END:VEVENT", "END:VCALENDAR",
		"SUMMARY:Чтение вслух", "UID:" + e.ID.String(), "DTSTART:20260701T180000Z",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("ics missing %q in:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -vet=off ./internal/rsvp/ -run TestBuildICS`
Expected: FAIL (undefined `buildICS`, or stub returns nil).

- [ ] **Step 3: Write the generator**

`backend/internal/rsvp/ics.go`:
```go
package rsvp

import (
	"fmt"
	"strings"
	"time"

	"github.com/Pashteto/lia/internal/models"
)

// buildICS renders a minimal RFC-5545 VEVENT for the event. Times are emitted in
// UTC (Z) — calendar apps localize for display; no OAuth, per spec (.ics only).
func buildICS(e *models.Event) []byte {
	const layout = "20060102T150405Z"
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Lia//RSVP//RU\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s\r\n", e.ID.String())
	fmt.Fprintf(&b, "DTSTAMP:%s\r\n", nowFn().UTC().Format(layout))
	fmt.Fprintf(&b, "DTSTART:%s\r\n", e.StartsAt.UTC().Format(layout))
	if e.EndsAt != nil && !e.EndsAt.IsZero() {
		fmt.Fprintf(&b, "DTEND:%s\r\n", e.EndsAt.UTC().Format(layout))
	}
	fmt.Fprintf(&b, "SUMMARY:%s\r\n", icsEscape(e.Title))
	if e.Description != "" {
		fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", icsEscape(e.Description))
	}
	if e.Venue != nil && e.Venue.Address != "" {
		fmt.Fprintf(&b, "LOCATION:%s\r\n", icsEscape(e.Venue.Address))
	}
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String())
}

// icsEscape escapes the RFC-5545 special characters in a text value.
func icsEscape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n")
	return r.Replace(s)
}

var _ = time.Now // time imported for layout doc; nowFn lives in service.go
```
(Remove the trailing `var _ = time.Now` line and the `"time"` import if `go build` reports them unused — they're only there to document the layout constant.)

If you added a temporary `buildICS` stub in Task 6, delete it now.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test -vet=off ./internal/rsvp/ -run TestBuildICS && go test -vet=off ./internal/rsvp/`
Expected: PASS (all rsvp tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/rsvp/ics.go backend/internal/rsvp/ics_test.go backend/internal/rsvp/service.go
git commit -m "feat(rsvp): .ics VEVENT generator"
```

---

## Task 8: Event formatter + repository — expose signup fields & seats

**Files:**
- Modify: `backend/internal/http/formatter/event.go`
- Modify: `backend/internal/events/repository.go` (populate `SeatsRemaining`; `MyRsvpStatus` is set by the handler layer — see note)
- Test: `backend/internal/http/formatter/event_test.go` (add a case)

**Interfaces:**
- Consumes: `Event.SignupMode/Capacity/SeatsRemaining/MyRsvpStatus`, swagger `apiModels.Event` fields added in Task 9's swagger edit (`signup_mode`, `capacity`, `seats_remaining`, `my_rsvp_status`, `curator_question`, `external_registration_url`).
- Produces: `EventToAPI` emitting the new fields.

> **Note on SeatsRemaining/MyRsvpStatus population:** computing these per event requires an RSVP count and the caller's row. To avoid a cross-module dependency from `events.Repository` into `rsvp`, populate them in the events repository with two small batch queries against `event_rsvps` directly (the table, not the rsvp package). Add a `loadSeats(events)` helper that, for the listed event ids, counts `status='going'` and (when a caller id is in context) the caller's status. For this first slice, keep it simple: compute `seats_remaining` in `loadSeats` for all reads; populate `my_rsvp_status` only on the single-event detail path where the caller is known (see Step 3).

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/http/formatter/event_test.go`:
```go
func TestEventToAPIIncludesSignupFields(t *testing.T) {
	cap := 10
	remaining := 4
	e := &domainModels.Event{
		ID:             uuid.Must(uuid.NewV4()),
		Title:          "x",
		StartsAt:       time.Now(),
		Status:         domainModels.EventPublished,
		SignupMode:     "application",
		Capacity:       &cap,
		CuratorQuestion: "почему?",
		SeatsRemaining: &remaining,
		MyRsvpStatus:   "applied",
	}
	out := EventToAPI(e)
	if out.SignupMode != "application" {
		t.Fatalf("signup_mode = %q", out.SignupMode)
	}
	if out.Capacity == nil || *out.Capacity != 10 {
		t.Fatalf("capacity = %v", out.Capacity)
	}
	if out.SeatsRemaining == nil || *out.SeatsRemaining != 4 {
		t.Fatalf("seats_remaining = %v", out.SeatsRemaining)
	}
	if out.MyRsvpStatus != "applied" {
		t.Fatalf("my_rsvp_status = %q", out.MyRsvpStatus)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -vet=off ./internal/http/formatter/ -run TestEventToAPIIncludesSignupFields`
Expected: FAIL (fields not on `apiModels.Event` yet → compile error). This task's swagger field additions are bundled into Task 9; if Task 9 hasn't run, this test won't compile. **Run Task 9's Step 1 (swagger edit + codegen) before this test compiles.** Order: do Task 9 swagger+codegen first, then return here.

- [ ] **Step 3: Extend EventToAPI**

In `backend/internal/http/formatter/event.go`, inside `EventToAPI`, after the cover block:
```go
	out.SignupMode = event.SignupMode
	out.CuratorQuestion = event.CuratorQuestion
	out.ExternalRegistrationURL = event.ExternalRegistrationURL
	if event.Capacity != nil {
		c := int64(*event.Capacity)
		out.Capacity = &c
	}
	if event.SeatsRemaining != nil {
		s := int64(*event.SeatsRemaining)
		out.SeatsRemaining = &s
	}
	out.MyRsvpStatus = event.MyRsvpStatus
```

- [ ] **Step 4: Populate SeatsRemaining in the events repository**

In `backend/internal/events/repository.go`, add a `loadSeats` helper and call it from `GetByID` and `List` (after `loadOrganizers`):
```go
// loadSeats populates SeatsRemaining on each event that has a capacity, by
// counting going RSVPs in a single query (no N+1). Events with nil capacity are
// left as unlimited (SeatsRemaining stays nil).
func (r *pgRepository) loadSeats(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	for _, e := range events {
		if e.Capacity != nil {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		EventID uuid.UUID `pg:"event_id"`
		Going   int       `pg:"going"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT event_id, COUNT(*) AS going FROM event_rsvps
		 WHERE event_id IN (?) AND status = 'going' GROUP BY event_id`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load seats: %w", err)
	}
	goingByID := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		goingByID[row.EventID] = row.Going
	}
	for _, e := range events {
		if e.Capacity == nil {
			continue
		}
		remaining := *e.Capacity - goingByID[e.ID]
		if remaining < 0 {
			remaining = 0
		}
		e.SeatsRemaining = &remaining
	}
	return nil
}
```
Then add `if err := r.loadSeats([]*models.Event{event}); err != nil { return nil, err }` in `GetByID` and `if err := r.loadSeats(list); err != nil { return nil, err }` in `List`, after the `loadOrganizers` calls.

> `MyRsvpStatus` for the detail page is set in the RSVP handler path (Task 9) when the caller is authenticated; the public events handlers leave it "".

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test -vet=off ./internal/http/formatter/ -run TestEventToAPIIncludesSignupFields`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/http/formatter/event.go backend/internal/http/formatter/event_test.go backend/internal/events/repository.go
git commit -m "feat(rsvp): expose signup fields + seats_remaining on event API"
```

---

## Task 9: Swagger + RSVP handlers + wiring

**Files:**
- Modify: `backend/api/swagger.yaml`
- Create: `backend/internal/http/handlers/rsvp.go`
- Create: `backend/internal/http/handlers/rsvp_calendar.go`
- Modify: `backend/internal/http/module.go`
- Modify: `backend/internal/application.go`

**Interfaces:**
- Consumes: `rsvp.Service`, generated ops under `internal/http/server/operations/rsvp`.
- Produces: registered endpoints `POST/DELETE /events/{id}/rsvp`, `GET /me/practices`, `GET /me/applications`, `GET /events/{id}/applications`, `POST /events/{id}/applications/{rsvpId}/decision`, `GET /events/{id}/calendar.ics`.

- [ ] **Step 1: Add the new event fields to swagger**

In `backend/api/swagger.yaml`, under `Event:` `properties:` (after `organizer:`), add:
```yaml
      signup_mode:
        type: string
        enum: [open, application, external]
      capacity:
        type: integer
        format: int64
        x-nullable: true
      curator_question:
        type: string
      external_registration_url:
        type: string
      seats_remaining:
        type: integer
        format: int64
        readOnly: true
        x-nullable: true
      my_rsvp_status:
        type: string
        readOnly: true
```
Add the same `signup_mode`, `capacity`, `curator_question`, `external_registration_url` to `EventInput:` and `EventPatch:` `properties:` (so organizers can set them on create/edit). Then wire them in `EventFromAPIInput`/`EventPatchToUpdateParams` in `formatter/event.go` (mirror the existing `Format`/`PriceType` handling: copy `in.SignupMode`, parse `in.Capacity` to `*int`, copy the two text fields).

- [ ] **Step 2: Add RSVP definitions + paths to swagger**

Add definitions:
```yaml
  Rsvp:
    type: object
    properties:
      id: { type: string, format: uuid, readOnly: true }
      event_id: { type: string, format: uuid, readOnly: true }
      user_id: { type: string, format: uuid, readOnly: true }
      status: { type: string, readOnly: true }
      application_answer: { type: string }
      created_at: { type: string, format: date-time, readOnly: true }
      event: { $ref: "#/definitions/Event" }
  RsvpInput:
    type: object
    properties:
      application_answer: { type: string }
  DecisionInput:
    type: object
    required: [decision]
    properties:
      decision: { type: string, enum: [accept, decline] }
```
Add paths (tag `rsvp`; all `jwt` except `calendar.ics` which is `security: []`):
- `POST /events/{id}/rsvp` → `signUp` (param `id` path uuid; body `RsvpInput`; 201 `Rsvp`; 401; 404; 409 (already registered); 422 (external — body carries the URL in `Error.detail`)).
- `DELETE /events/{id}/rsvp` → `cancelRsvp` (204; 401; 404).
- `GET /me/practices` → `myPractices` (query `tab` enum [upcoming, past] default upcoming; 200 array `Rsvp`; 401).
- `GET /me/applications` → `myApplications` (query `status` optional; 200 array `Rsvp`; 401).
- `GET /events/{id}/applications` → `listEventApplications` (200 array `Rsvp`; 401; 403; 404).
- `POST /events/{id}/applications/{rsvpId}/decision` → `decideApplication` (body `DecisionInput`; 200 `Rsvp`; 401; 403; 404; 409).
- `GET /events/{id}/calendar.ics` → `eventCalendar` (`produces: [text/calendar]`; 200 string; 404; 422). Mark `security: []`.

- [ ] **Step 3: Regenerate the API**

Run:
```bash
cd backend && make generate-all && make generate-api
```
Expected: regenerates `internal/http/models` and `internal/http/server/operations/rsvp` (+ updates events ops). `go build ./internal/http/...` should compile the generated code.

- [ ] **Step 4: Write the RSVP handlers**

`backend/internal/http/handlers/rsvp.go` — mirror `handlers/events.go` structure (one struct + `New…` + `Handle` per op; map domain errors to generated responders). Key mappings: `ErrNotFound`→404, `ErrConflict`→409, `ErrForbidden`→403, `ErrExternal`→422 with `DefaultError(422, errors.New(url), nil)`, `ErrInvalidInput`→400, else 503. Each authed handler reads `principal *apimodels.User`, parses `uuid.FromString(principal.UUID.String())`, and returns 401 when principal is nil (mirror `ListMyEvents`). For the `signUp` handler, after a successful `SignUp`, set the returned event-bearing payload's `my_rsvp_status` from the new row's status.

Representative handler (the rest follow the same shape):
```go
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	rsvpops "github.com/Pashteto/lia/internal/http/server/operations/rsvp"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
	"github.com/Pashteto/lia/pkg/logger"
)

// SignUp handles POST /events/{id}/rsvp.
type SignUp struct{ rsvp rsvpdomain.Service }

// NewSignUp constructs a SignUp handler.
func NewSignUp(svc rsvpdomain.Service) *SignUp { return &SignUp{rsvp: svc} }

// Handle registers the caller on the event.
func (h *SignUp) Handle(params rsvpops.SignUpParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewSignUpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	userID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewSignUpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewSignUpBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}
	answer := ""
	if params.Body != nil {
		answer = params.Body.ApplicationAnswer
	}
	row, err := h.rsvp.SignUp(params.HTTPRequest.Context(), eventID, userID, answer)
	if err != nil {
		logger.Log().Errorf("signup event %s: %s", eventID, err.Error())
		switch {
		case errors.Is(err, rsvpdomain.ErrNotFound):
			return rsvpops.NewSignUpNotFound().WithPayload(DefaultError(http.StatusNotFound, err, nil))
		case errors.Is(err, rsvpdomain.ErrConflict):
			return rsvpops.NewSignUpConflict().WithPayload(DefaultError(http.StatusConflict, err, nil))
		case errors.Is(err, rsvpdomain.ErrExternal):
			return rsvpops.NewSignUpUnprocessableEntity().WithPayload(DefaultError(http.StatusUnprocessableEntity, err, nil))
		default:
			return rsvpops.NewSignUpServiceUnavailable().WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}
	return rsvpops.NewSignUpCreated().WithPayload(formatter.RsvpToAPI(row))
}
```
Add a `formatter.RsvpToAPI(*models.Rsvp) *apiModels.Rsvp` in `formatter/event.go` (or a new `formatter/rsvp.go`) mapping fields incl. nested `Event` via `EventToAPI` when set. Implement the remaining handlers (`CancelRsvp`, `MyPractices`, `MyApplications`, `ListEventApplications`, `DecideApplication`) in the same file following the same error mapping.

- [ ] **Step 5: Write the calendar handler**

`backend/internal/http/handlers/rsvp_calendar.go`:
```go
package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	rsvpops "github.com/Pashteto/lia/internal/http/server/operations/rsvp"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
)

// EventCalendar handles GET /events/{id}/calendar.ics (public).
type EventCalendar struct{ rsvp rsvpdomain.Service }

// NewEventCalendar constructs an EventCalendar handler.
func NewEventCalendar(svc rsvpdomain.Service) *EventCalendar { return &EventCalendar{rsvp: svc} }

// Handle returns the event as a downloadable .ics.
func (h *EventCalendar) Handle(params rsvpops.EventCalendarParams) middleware.Responder {
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewEventCalendarNotFound()
	}
	data, err := h.rsvp.CalendarICS(params.HTTPRequest.Context(), eventID)
	if err != nil {
		if errors.Is(err, rsvpdomain.ErrNotFound) {
			return rsvpops.NewEventCalendarNotFound()
		}
		return rsvpops.NewEventCalendarUnprocessableEntity()
	}
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="event.ics"`)
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, bytesReader(data))
	})
}
```
Add a tiny `bytesReader` helper (or use `bytes.NewReader`) — prefer `bytes.NewReader(data)` and import `"bytes"`.

- [ ] **Step 6: Wire the service and register handlers**

In `backend/internal/http/module.go`:
- add `rsvp eventsdomain`-style field: `rsvp rsvpdomain.Service` (import `rsvpdomain "github.com/Pashteto/lia/internal/rsvp"`),
- add `func (m *Module) SetRsvpService(svc rsvpdomain.Service) { m.rsvp = svc }`,
- in `initAPI`, inside the `if m.events != nil` block (or a new `if m.rsvp != nil` block), register:
```go
	if m.rsvp != nil {
		api.RsvpSignUpHandler = handlers.NewSignUp(m.rsvp)
		api.RsvpCancelRsvpHandler = handlers.NewCancelRsvp(m.rsvp)
		api.RsvpMyPracticesHandler = handlers.NewMyPractices(m.rsvp)
		api.RsvpMyApplicationsHandler = handlers.NewMyApplications(m.rsvp)
		api.RsvpListEventApplicationsHandler = handlers.NewListEventApplications(m.rsvp)
		api.RsvpDecideApplicationHandler = handlers.NewDecideApplication(m.rsvp)
		api.RsvpEventCalendarHandler = handlers.NewEventCalendar(m.rsvp)
	}
```
(The exact generated handler-field names come from the `operationId`s + tag; adjust to match what codegen produced.)

In `backend/internal/application.go`, after the events wiring, add:
```go
	if repoModule != nil {
		app.rsvpSvc = rsvpdomain.NewService(rsvpdomain.NewRepository(repoModule.DB()))
		logger.Log().Info("rsvp module wired to repository")
	}
```
add the `rsvpSvc rsvpdomain.Service` field to the app struct + the import, and after `httpModule.SetEventsService(...)` add `httpModule.SetRsvpService(app.rsvpSvc)`.

- [ ] **Step 7: Build + test**

Run:
```bash
cd backend && go build ./... && go test -vet=off ./internal/rsvp/ ./internal/http/formatter/ ./internal/models/
```
Expected: build green; tests PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/api/swagger.yaml backend/internal/http/ backend/internal/application.go
git commit -m "feat(rsvp): swagger, handlers, .ics endpoint, wiring"
```

---

## Task 10: Frontend — API client + types

**Files:**
- Modify: `frontend/lib/types.ts`
- Modify: `frontend/lib/api.ts`

**Interfaces:**
- Produces: types `RsvpStatus`, `SignupMode`, `Rsvp`; functions `signUp`, `cancelRsvp`, `fetchMyPractices`, `fetchMyApplications`, `fetchEventApplications`, `decideApplication`, `eventCalendarUrl`; `LiaEvent` gains `signupMode`, `capacity?`, `seatsRemaining?`, `myRsvpStatus?`, `curatorQuestion?`, `externalRegistrationUrl?`.

- [ ] **Step 1: Extend types**

In `frontend/lib/types.ts` add:
```ts
export type SignupMode = "open" | "application" | "external";
export type RsvpStatus =
  | "going" | "waitlist" | "applied" | "accepted"
  | "declined" | "withdrawn" | "cancelled";

export interface Rsvp {
  id: string;
  eventId: string;
  status: RsvpStatus;
  applicationAnswer?: string;
  createdAt: string;
  event?: LiaEvent;
}
```
Add to `LiaEvent`: `signupMode?: SignupMode; capacity?: number; seatsRemaining?: number; myRsvpStatus?: RsvpStatus | ""; curatorQuestion?: string; externalRegistrationUrl?: string;`. Add to `ApiEvent`: `signup_mode?`, `capacity?`, `seats_remaining?`, `my_rsvp_status?`, `curator_question?`, `external_registration_url?`. Map them in `apiEventToLia` (in `api.ts`).

- [ ] **Step 2: Add the RSVP API calls**

In `frontend/lib/api.ts` (mirror `fetchMyEvents`'s Bearer pattern and `apiEventToLia` mapping):
```ts
function apiRsvpToLia(r: any): Rsvp {
  return {
    id: r.id,
    eventId: r.event_id,
    status: r.status,
    applicationAnswer: r.application_answer || undefined,
    createdAt: r.created_at,
    event: r.event ? apiEventToLia(r.event) : undefined,
  };
}

export async function signUp(eventId: string, applicationAnswer?: string): Promise<Rsvp> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_V1}/events/${eventId}/rsvp`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ application_answer: applicationAnswer ?? "" }),
  });
  if (res.status === 422) {
    const body = await res.json().catch(() => ({}));
    throw new Error(`EXTERNAL:${body?.detail ?? ""}`); // caller opens organizer URL
  }
  if (!res.ok) throw new Error(`sign up failed: ${res.status}`);
  return apiRsvpToLia(await res.json());
}

export async function cancelRsvp(eventId: string): Promise<void> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_V1}/events/${eventId}/rsvp`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok && res.status !== 204) throw new Error(`cancel failed: ${res.status}`);
}

export async function fetchMyPractices(tab: "upcoming" | "past" = "upcoming"): Promise<Rsvp[]> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_V1}/me/practices?tab=${tab}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`fetch practices failed: ${res.status}`);
  return (await res.json()).map(apiRsvpToLia);
}

export async function fetchMyApplications(status?: string): Promise<Rsvp[]> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const q = status ? `?status=${status}` : "";
  const res = await fetch(`${API_V1}/me/applications${q}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`fetch applications failed: ${res.status}`);
  return (await res.json()).map(apiRsvpToLia);
}

export async function fetchEventApplications(eventId: string): Promise<Rsvp[]> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_V1}/events/${eventId}/applications`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`fetch event applications failed: ${res.status}`);
  return (await res.json()).map(apiRsvpToLia);
}

export async function decideApplication(eventId: string, rsvpId: string, decision: "accept" | "decline"): Promise<Rsvp> {
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_V1}/events/${eventId}/applications/${rsvpId}/decision`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ decision }),
  });
  if (!res.ok) throw new Error(`decision failed: ${res.status}`);
  return apiRsvpToLia(await res.json());
}

export function eventCalendarUrl(eventId: string): string {
  return `${API_V1}/events/${eventId}/calendar.ics`;
}
```
Add `Rsvp` to the type import block at the top of `api.ts`.

- [ ] **Step 3: Verify the frontend compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: no type errors. Commit.

```bash
git add frontend/lib/api.ts frontend/lib/types.ts
git commit -m "feat(rsvp): frontend api client + types"
```

---

## Task 11: Frontend — event detail signup CTA

**Files:**
- Modify: `frontend/app/events/[id]/page.tsx` (and/or extract a `SignupCTA` client component)

**Interfaces:**
- Consumes: `signUp`, `cancelRsvp`, `eventCalendarUrl`, `LiaEvent.{signupMode,seatsRemaining,myRsvpStatus,externalRegistrationUrl,curatorQuestion}`.

- [ ] **Step 1: Add a SignupCTA client component**

Create `frontend/components/SignupCTA.tsx` (client component) that renders by `event.signupMode` and `event.myRsvpStatus`:
- `external`: link button "Записаться на сайте организатора" → `event.externalRegistrationUrl` (new tab) + caption "Запись ведёт организатор".
- `open`, no `myRsvpStatus`: "Записаться" (or "В лист ожидания" when `seatsRemaining === 0`) → `signUp(event.id)` → on success show "Вы записаны"/"Вы в листе ожидания" + "Отписаться" → `cancelRsvp(event.id)`.
- `open`, `myRsvpStatus==="going"`: "Вы записаны" + "Отписаться".
- `open`, `myRsvpStatus==="waitlist"`: "Вы в листе ожидания" + "Покинуть лист".
- `application`, no status: "Подать заявку" → opens a sheet with `event.curatorQuestion` + textarea → `signUp(event.id, answer)`.
- `application`, `applied`: "Заявка отправлена" + "Отозвать заявку".
- `application`, `accepted`/`declined`: show the terminal state.
- Always: "В календарь" link → `eventCalendarUrl(event.id)` (download; works logged-out).
Show seats counter when `event.capacity != null`: "Осталось мест: {seatsRemaining}".

Handle the `EXTERNAL:` error prefix from `signUp` by opening the returned URL.

- [ ] **Step 2: Mount it on the event page**

Replace the stubbed "Записаться" button in `frontend/app/events/[id]/page.tsx` with `<SignupCTA event={event} />`. Since the page is a server component, pass the loaded `event` as a prop to the client `SignupCTA`.

- [ ] **Step 3: Verify**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: builds. Manually verify against a locally running stack (see `docs/HANDOFF.md` for run instructions): open mode sign-up flips to "Вы записаны"; full event offers waitlist; "В календарь" downloads an `.ics`. Commit.

```bash
git add frontend/components/SignupCTA.tsx frontend/app/events/[id]/page.tsx
git commit -m "feat(rsvp): event detail signup CTA + states"
```

---

## Task 12: Frontend — /me/practices and /me/applications

**Files:**
- Create: `frontend/app/me/practices/page.tsx`
- Create: `frontend/app/me/applications/page.tsx`

**Interfaces:**
- Consumes: `fetchMyPractices`, `fetchMyApplications`, `cancelRsvp`.

- [ ] **Step 1: Build /me/practices**

Client page with tabs "Предстоящие"/"Прошедшие" → `fetchMyPractices("upcoming"|"past")`. Each row: `event.startsAt` (formatted), `event.title`, `event.venue?.name`, a status chip (`going`→"вы записаны", `waitlist`→"в листе ожидания", `accepted`→"заявка принята"), and a contextual action ("Отписаться"/"Покинуть лист" via `cancelRsvp`, refetch on success). Redirect to sign-in when `getToken()` is empty, preserving `?next=/me/practices` (mirror how existing auth-gated pages redirect).

- [ ] **Step 2: Build /me/applications**

Client page with tabs "В ожидании"/"Принятые"/"Отклонённые"/"Отозванные" mapped to statuses `applied|accepted|declined|withdrawn` → `fetchMyApplications(status)`. Card per row: `event.title`, the curator question (`event.curatorQuestion`), the user's answer (`applicationAnswer`, expandable), status + `createdAt`. From "В ожидании": "Отозвать заявку" → `cancelRsvp(event.id)`.

- [ ] **Step 3: Verify + commit**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: builds. Commit.

```bash
git add frontend/app/me/practices/page.tsx frontend/app/me/applications/page.tsx
git commit -m "feat(rsvp): /me/practices + /me/applications pages"
```

---

## Task 13: Frontend — organizer applications panel

**Files:**
- Create: `frontend/components/EventApplicationsPanel.tsx`
- Modify: the organizer-facing event view reached from `frontend/app/events/mine` (the event card or its detail) to mount the panel for events the caller organizes.

**Interfaces:**
- Consumes: `fetchEventApplications`, `decideApplication`.

- [ ] **Step 1: Build the panel**

`EventApplicationsPanel.tsx` (client): given an `eventId`, loads `fetchEventApplications(eventId)`; for each `applied` row shows the applicant's answer + "Принять"/"Отклонить" → `decideApplication(eventId, rsvpId, "accept"|"decline")`, refetching on success. Shows decided rows (accepted/declined) read-only. Only render for `signup_mode === "application"` events.

- [ ] **Step 2: Mount on the organizer's event**

On the "Мои события" surface (`frontend/app/events/mine`), when an event is `application` mode and owned by the caller, render `<EventApplicationsPanel eventId={event.id} />` (e.g. behind a "Заявки" expander on the event row/detail). Keep styling minimal — the full `/o` cabinet is a later slice.

- [ ] **Step 3: Verify + commit**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: builds. Manually verify: as the organizer, an applicant's application appears and "Принять" flips it to accepted; the applicant then sees "Заявка принята" on `/me/applications`. Commit.

```bash
git add frontend/components/EventApplicationsPanel.tsx frontend/app/events/mine
git commit -m "feat(rsvp): organizer applications accept/decline panel"
```

---

## Self-Review

**Spec coverage (spec §2/§4/§5/§6/§7):**
- All three signup modes → Task 4 (model), Task 6 (service `SignUp` switch), Task 11 (CTA).
- Capacity + waitlist + auto-promotion → Task 1 (column), Task 5 (`SignUpTx`/`CancelTx`), Task 6 tests `TestSignUpOpenFillsThenWaitlists`/`TestCancelPromotesWaitlist`.
- Applications + organizer decision → Task 5 (`DecideTx`), Task 6 (`Decide`/`ListApplications`), Task 9 (endpoints), Task 13 (organizer UI).
- `/me/practices` + `/me/applications` → Task 12.
- `.ics` export → Task 7 (generator), Task 9 (public endpoint), Task 11 ("В календарь").
- `seats_remaining` / `my_rsvp_status` on event API → Task 8.
- Concurrency / last-seat race → Task 5 (`SELECT … FOR UPDATE` in `SignUpTx`/`DecideTx`/`CancelTx`).
- Auth required except `.ics` → Task 9 swagger `security` + handler principal checks.
- Privacy (no participant identities on public surfaces) → `.ics` carries event data only; `my_rsvp_status` per-caller (Task 8 note).

**Out-of-scope confirmed absent:** no saved-events table/endpoint, no reflections UI, no notifications delivery, no `/o` cabinet, no recurring — matches spec §2 non-goals.

**Placeholder scan:** backend correctness-critical code (model, repository, service, ics) is fully spelled out with tests. Transport (Task 9) and frontend (Tasks 10–13) reference existing in-repo patterns by exact path (`handlers/events.go` `ListMyEvents`, `fetchMyEvents`, `apiEventToLia`) rather than re-printing them — those files exist for the implementer to mirror; generated handler-field names must be read off codegen output (called out in Task 9 Step 6).

**Type consistency:** `RsvpStatus` string values, `Service` method signatures, error names (`ErrConflict`/`ErrForbidden`/`ErrExternal`), and `SeatDecider`/`seatAvailable` are used identically across Tasks 3, 5, 6, 9. Frontend `Rsvp`/`SignupMode`/`RsvpStatus` names match across Tasks 10–13.

**Known sequencing note:** Task 8's formatter test only compiles after Task 9 Step 1–3 (swagger fields + codegen) add the fields to `apiModels.Event`. Run Task 9's swagger+codegen before Task 8's test step (flagged in Task 8 Step 2). Similarly Task 5 references `ErrConflict` (defined in Task 6) and `buildICS` (Task 7) — implement Tasks 5→6→7 back-to-back, building the package only after Task 7.
