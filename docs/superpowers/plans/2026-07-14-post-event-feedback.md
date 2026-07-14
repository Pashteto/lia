# Private Post-Event Feedback (R3) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After an event ends, a participant (active RSVP: `going`/`accepted`) can leave a 1–5 star rating + optional comment, one per person; the event owner (and admin) privately sees all feedback plus an average. No public rating anywhere.

**Architecture:** A new `internal/feedback` domain (service + repository) backed by a new `event_feedback` table (migration 000019), and a plain `net/http` handler `internal/http/feedback` mounted ahead of the swagger mux — mirroring `internal/complaints` + `internal/http/complaints`. The repository owns the gate queries (event ended/owner, active-RSVP, existing-feedback) so the domain stays self-contained, matching how `complaints` owns `EventExists`.

**Tech Stack:** Go (go-pg, gofrs/uuid), plain `net/http` + `http.ServeMux` path patterns, Next.js 15/TS. Spec: `docs/superpowers/specs/2026-07-14-post-event-feedback-design.md`. Independent of R1/R2 — can ship in parallel.

## Global Constraints

- All user-facing copy in **Russian**.
- **Migration 018 → 019.** New table `event_feedback` with `UNIQUE (event_id, user_id)`.
- Feedback is **private**: read access is **event owner or admin only**; nothing renders on public surfaces; author payload carries **name only, never email** (mirror `LoadApplicantNames`).
- "Ended" = stored end instant `COALESCE(ends_at, starts_at) < now()` (an instant comparison — timezone-agnostic; do NOT gate on civil МСК day).
- Gates return distinct Russian errors: not-ended 422, not-a-participant 403, duplicate 409, bad rating 422, non-owner read 403, anon 401.
- Plain `net/http` domains bypass swagger; they must be mounted in `internal/http/module.go`'s router and injected in `internal/application.go` (mirror `SetComplaints`). No `make generate-api` needed (no swagger model).
- `FROM scratch` prod image: no new tz dependency introduced (comparison uses DB `now()`).

---

### Task 1: Migration — `event_feedback`

**Files:**
- Create: `backend/db/migrations/000019_event_feedback.up.sql`
- Create: `backend/db/migrations/000019_event_feedback.down.sql`

**Interfaces:**
- Produces: table `event_feedback (id, event_id, user_id, rating, comment, created_at)` with `UNIQUE (event_id, user_id)` and an index on `event_id`. Consumed by Task 3 (repository).

- [ ] **Step 1: Write the up migration**

`000019_event_feedback.up.sql`:

```sql
CREATE TABLE event_feedback (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   uuid NOT NULL,
    user_id    uuid NOT NULL,
    rating     smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id)
);

-- Owner's list/aggregate query is by event.
CREATE INDEX event_feedback_event_idx ON event_feedback (event_id);
```

`000019_event_feedback.down.sql`:

```sql
DROP TABLE IF EXISTS event_feedback;
```

- [ ] **Step 2: Apply + verify locally**

Run: `cd backend && docker compose up -d && docker compose exec -T postgres psql -U postgres -d lia -c "\d event_feedback"`
Expected: table shown with the unique constraint + index. (If migrations run at app boot, check `SELECT max(version) FROM schema_migrations;` → `19`.)

- [ ] **Step 3: Commit**

```bash
git add backend/db/migrations/000019_event_feedback.up.sql backend/db/migrations/000019_event_feedback.down.sql
git commit -m "feat(r3): event_feedback migration 000019"
```

---

### Task 2: Domain — `internal/feedback` service + errors (fakes)

**Files:**
- Create: `backend/internal/feedback/service.go`
- Test: `backend/internal/feedback/service_test.go`

**Interfaces:**
- Produces:
  - `type Feedback struct { ID, EventID, UserID uuid.UUID; Rating int; Comment string; CreatedAt time.Time }`
  - `type Item struct { Rating int; Comment string; AuthorName string; CreatedAt time.Time }`
  - `type Summary struct { Average float64; Count int; Items []Item }`
  - `Repository` interface:
    - `EventGate(ctx, eventID uuid.UUID) (ownerID uuid.UUID, endsAt time.Time, exists bool, err error)`
    - `HasActiveRsvp(ctx, eventID, userID uuid.UUID) (bool, error)`
    - `ExistsForUser(ctx, eventID, userID uuid.UUID) (bool, error)`
    - `Insert(ctx, f Feedback) error`
    - `ListForEvent(ctx, eventID uuid.UUID) ([]Item, error)`
  - `Service` interface:
    - `Submit(ctx, userID, eventID uuid.UUID, rating int, comment string) error`
    - `ForOwner(ctx, eventID, requesterID uuid.UUID, isAdmin bool) (Summary, error)`
    - `MyFeedback(ctx, userID, eventID uuid.UUID) (bool, error)`
  - Errors: `ErrNotEnded` (422), `ErrNotParticipant` (403), `ErrAlreadySubmitted` (409), `ErrInvalidRating` (422), `ErrForbidden` (403), `ErrNotFound` (404).
- Consumed by Tasks 3 (repo impl) + 4 (handler).

- [ ] **Step 1: Failing service tests with a fake repo**

Create `backend/internal/feedback/service_test.go`:

```go
package feedback_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/Pashteto/lia/internal/feedback"
)

type fakeRepo struct {
	owner        uuid.UUID
	endsAt       time.Time
	exists       bool
	active       bool
	already      bool
	inserted     *feedback.Feedback
	items        []feedback.Item
}

func (f *fakeRepo) EventGate(_ context.Context, _ uuid.UUID) (uuid.UUID, time.Time, bool, error) {
	return f.owner, f.endsAt, f.exists, nil
}
func (f *fakeRepo) HasActiveRsvp(_ context.Context, _, _ uuid.UUID) (bool, error) { return f.active, nil }
func (f *fakeRepo) ExistsForUser(_ context.Context, _, _ uuid.UUID) (bool, error) { return f.already, nil }
func (f *fakeRepo) Insert(_ context.Context, fb feedback.Feedback) error { f.inserted = &fb; return nil }
func (f *fakeRepo) ListForEvent(_ context.Context, _ uuid.UUID) ([]feedback.Item, error) { return f.items, nil }

func base() *fakeRepo {
	return &fakeRepo{owner: uuid.Must(uuid.NewV4()), endsAt: time.Now().Add(-time.Hour), exists: true, active: true}
}

func TestSubmit_HappyPath(t *testing.T) {
	r := base()
	svc := feedback.NewService(r)
	if err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "ок"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if r.inserted == nil || r.inserted.Rating != 4 { t.Fatal("not inserted") }
}

func TestSubmit_NotEnded(t *testing.T) {
	r := base(); r.endsAt = time.Now().Add(time.Hour)
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrNotEnded) { t.Fatalf("want ErrNotEnded, got %v", err) }
}

func TestSubmit_NotParticipant(t *testing.T) {
	r := base(); r.active = false
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrNotParticipant) { t.Fatalf("want ErrNotParticipant, got %v", err) }
}

func TestSubmit_Duplicate(t *testing.T) {
	r := base(); r.already = true
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrAlreadySubmitted) { t.Fatalf("want ErrAlreadySubmitted, got %v", err) }
}

func TestSubmit_BadRating(t *testing.T) {
	err := feedback.NewService(base()).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 6, "")
	if !errors.Is(err, feedback.ErrInvalidRating) { t.Fatalf("want ErrInvalidRating, got %v", err) }
}

func TestForOwner_ForbiddenForNonOwner(t *testing.T) {
	r := base()
	_, err := feedback.NewService(r).ForOwner(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), false)
	if !errors.Is(err, feedback.ErrForbidden) { t.Fatalf("want ErrForbidden, got %v", err) }
}

func TestForOwner_AverageAndCount(t *testing.T) {
	r := base()
	r.items = []feedback.Item{{Rating: 5}, {Rating: 3}}
	sum, err := feedback.NewService(r).ForOwner(context.Background(), uuid.Must(uuid.NewV4()), r.owner, false)
	if err != nil { t.Fatalf("ForOwner: %v", err) }
	if sum.Count != 2 || sum.Average != 4.0 { t.Fatalf("want avg 4 count 2, got %+v", sum) }
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/feedback/ -v`
Expected: FAIL — package `feedback` does not exist.

- [ ] **Step 3: Implement the domain**

Create `backend/internal/feedback/service.go`:

```go
// Package feedback implements private post-event ratings: a participant who had
// an active RSVP on an ended event leaves a 1-5 star rating + optional comment
// (one per person); the event owner (and admin) reads them privately. See spec
// docs/superpowers/specs/2026-07-14-post-event-feedback-design.md.
package feedback

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrNotEnded         = errors.New("feedback: event has not ended")   // 422
	ErrNotParticipant   = errors.New("feedback: not a participant")     // 403
	ErrAlreadySubmitted = errors.New("feedback: already submitted")     // 409
	ErrInvalidRating    = errors.New("feedback: rating must be 1..5")    // 422
	ErrForbidden        = errors.New("feedback: not the event owner")   // 403
	ErrNotFound         = errors.New("feedback: event not found")       // 404
)

type Feedback struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	UserID    uuid.UUID
	Rating    int
	Comment   string
	CreatedAt time.Time
}

type Item struct {
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment,omitempty"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type Summary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
	Items   []Item  `json:"items"`
}

type Repository interface {
	EventGate(ctx context.Context, eventID uuid.UUID) (ownerID uuid.UUID, endsAt time.Time, exists bool, err error)
	HasActiveRsvp(ctx context.Context, eventID, userID uuid.UUID) (bool, error)
	ExistsForUser(ctx context.Context, eventID, userID uuid.UUID) (bool, error)
	Insert(ctx context.Context, f Feedback) error
	ListForEvent(ctx context.Context, eventID uuid.UUID) ([]Item, error)
}

type Service interface {
	Submit(ctx context.Context, userID, eventID uuid.UUID, rating int, comment string) error
	ForOwner(ctx context.Context, eventID, requesterID uuid.UUID, isAdmin bool) (Summary, error)
	MyFeedback(ctx context.Context, userID, eventID uuid.UUID) (bool, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Submit(ctx context.Context, userID, eventID uuid.UUID, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}
	owner, endsAt, exists, err := s.repo.EventGate(ctx, eventID)
	if err != nil {
		return err
	}
	_ = owner
	if !exists {
		return ErrNotFound
	}
	if !endsAt.Before(time.Now()) {
		return ErrNotEnded
	}
	active, err := s.repo.HasActiveRsvp(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if !active {
		return ErrNotParticipant
	}
	already, err := s.repo.ExistsForUser(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadySubmitted
	}
	return s.repo.Insert(ctx, Feedback{
		EventID: eventID, UserID: userID, Rating: rating, Comment: strings.TrimSpace(comment),
	})
}

func (s *service) ForOwner(ctx context.Context, eventID, requesterID uuid.UUID, isAdmin bool) (Summary, error) {
	owner, _, exists, err := s.repo.EventGate(ctx, eventID)
	if err != nil {
		return Summary{}, err
	}
	if !exists {
		return Summary{}, ErrNotFound
	}
	if !isAdmin && owner != requesterID {
		return Summary{}, ErrForbidden
	}
	items, err := s.repo.ListForEvent(ctx, eventID)
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{Count: len(items), Items: items}
	if len(items) > 0 {
		total := 0
		for _, it := range items {
			total += it.Rating
		}
		sum.Average = float64(total) / float64(len(items))
	}
	return sum, nil
}

func (s *service) MyFeedback(ctx context.Context, userID, eventID uuid.UUID) (bool, error) {
	return s.repo.ExistsForUser(ctx, eventID, userID)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test ./internal/feedback/ -v`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/feedback/service.go backend/internal/feedback/service_test.go
git commit -m "feat(r3): feedback domain service + gates"
```

---

### Task 3: Repository — pg implementation

**Files:**
- Create: `backend/internal/feedback/repository.go`
- Test: `backend/internal/feedback/repository_test.go` (`//go:build integration`)

**Interfaces:**
- Consumes: the `Repository` interface (Task 2).
- Produces: `NewRepository(db *pg.DB) Repository`. `ListForEvent` joins `users` for `name` only (never email).

- [ ] **Step 1: Failing integration test**

Create `backend/internal/feedback/repository_test.go` (guarded `//go:build integration`; connect via the same env the other repo tests use):

```go
//go:build integration

package feedback_test

// Insert a feedback row, then ListForEvent returns it with the author name and
// no email; ExistsForUser is true afterwards; EventGate returns the seeded owner
// and end instant. (Seed events/users/event_rsvps directly with INSERTs.)
```

Flesh out with direct `INSERT` seeds for an event (owner + `ends_at` in the past), a user (`uuid`,`name`,`email`), and an `event_rsvps` `going` row; assert `HasActiveRsvp` true, `Insert` then `ExistsForUser` true, `ListForEvent` returns one `Item` with `AuthorName` set.

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test -tags integration ./internal/feedback/ -v`
Expected: FAIL — `NewRepository` undefined.

- [ ] **Step 3: Implement the repository**

Create `backend/internal/feedback/repository.go`:

```go
package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) EventGate(ctx context.Context, eventID uuid.UUID) (uuid.UUID, time.Time, bool, error) {
	var row struct {
		OrganizerID uuid.UUID `pg:"organizer_id"`
		EndsAt      time.Time `pg:"ends_at"`
	}
	_, err := r.db.QueryOneContext(ctx, &row,
		`SELECT organizer_id, COALESCE(ends_at, starts_at) AS ends_at
		   FROM events WHERE id = ?`, eventID)
	if err == pg.ErrNoRows {
		return uuid.Nil, time.Time{}, false, nil
	}
	if err != nil {
		return uuid.Nil, time.Time{}, false, fmt.Errorf("event gate %s: %w", eventID, err)
	}
	return row.OrganizerID, row.EndsAt, true, nil
}

func (r *pgRepository) HasActiveRsvp(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	n, err := r.db.ModelContext(ctx, (*struct {
		tableName struct{} `pg:"event_rsvps"`
	})(nil)).
		Where("event_id = ? AND user_id = ? AND status IN ('going','accepted')", eventID, userID).
		Count()
	if err != nil {
		return false, fmt.Errorf("active rsvp: %w", err)
	}
	return n > 0, nil
}

func (r *pgRepository) ExistsForUser(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var exists bool
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&exists),
		`SELECT EXISTS(SELECT 1 FROM event_feedback WHERE event_id = ? AND user_id = ?)`, eventID, userID)
	if err != nil {
		return false, fmt.Errorf("exists for user: %w", err)
	}
	return exists, nil
}

func (r *pgRepository) Insert(ctx context.Context, f Feedback) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_feedback (event_id, user_id, rating, comment)
		 VALUES (?, ?, ?, NULLIF(?, ''))`,
		f.EventID, f.UserID, f.Rating, f.Comment)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}
	return nil
}

func (r *pgRepository) ListForEvent(ctx context.Context, eventID uuid.UUID) ([]Item, error) {
	var rows []Item
	// Name only — email is never selected (private author identity, public-safe).
	_, err := r.db.QueryContext(ctx, &rows,
		`SELECT f.rating AS rating, COALESCE(f.comment,'') AS comment,
		        COALESCE(u.name,'') AS author_name, f.created_at AS created_at
		   FROM event_feedback f
		   LEFT JOIN users u ON u.uuid = f.user_id
		  WHERE f.event_id = ?
		  ORDER BY f.created_at DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list feedback: %w", err)
	}
	return rows, nil
}
```

Add `var _ Repository = (*pgRepository)(nil)` to catch drift at compile time.

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test -tags integration ./internal/feedback/ -v` (or `go build ./...` if no integration DB locally).
Expected: PASS / compiles.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/feedback/repository.go backend/internal/feedback/repository_test.go
git commit -m "feat(r3): feedback pg repository (name-only author join)"
```

---

### Task 4: HTTP handler + wiring

**Files:**
- Create: `backend/internal/http/feedback/handler.go`
- Test: `backend/internal/http/feedback/handler_test.go`
- Modify: `backend/internal/http/module.go` (Set injector, build handler, mount route)
- Modify: `backend/internal/application.go` (inject the service near `SetComplaints`)

**Interfaces:**
- Consumes: `feedback.Service` (Task 2), `m.auth.Authenticate`.
- Produces routes:
  - `POST /api/v1/events/{id}/feedback` (authed participant) → 201 / 401 / 404 / 422 / 403 / 409
  - `GET /api/v1/events/{id}/feedback` (owner or admin) → 200 `Summary` / 401 / 403 / 404
  - `GET /api/v1/me/feedback?event_id=` (authed) → 200 `{submitted: bool}`

- [ ] **Step 1: Failing handler test**

Create `backend/internal/http/feedback/handler_test.go` — mirror `internal/http/complaints/handler_test.go`: a fake `feedback.Service`, assert anon POST → 401, `ErrNotEnded` → 422, `ErrNotParticipant` → 403, `ErrAlreadySubmitted` → 409, happy → 201; owner GET happy → 200, `ErrForbidden` → 403.

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/http/feedback/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the handler**

Create `backend/internal/http/feedback/handler.go` (mirror `internal/http/complaints/handler.go` for `principal`/`writeJSON`/`writeErr`):

```go
// Package feedback provides the post-event feedback HTTP surface, mounted ahead
// of the swagger mux in internal/http/module.go (mirrors internal/http/complaints).
package feedback

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	fbdomain "github.com/Pashteto/lia/internal/feedback"
	domain "github.com/Pashteto/lia/internal/models"
)

type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Feedback     fbdomain.Service
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/feedback", h.submit)
	h.mux.HandleFunc("GET /api/v1/events/{id}/feedback", h.list)
	h.mux.HandleFunc("GET /api/v1/me/feedback", h.mine)
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
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	err = h.deps.Feedback.Submit(r.Context(), u.UUID, eventID, body.Rating, body.Comment)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
	case errors.Is(err, fbdomain.ErrInvalidRating):
		writeErr(w, http.StatusUnprocessableEntity, "Оценка должна быть от 1 до 5")
	case errors.Is(err, fbdomain.ErrNotEnded):
		writeErr(w, http.StatusUnprocessableEntity, "Отзыв можно оставить после завершения события")
	case errors.Is(err, fbdomain.ErrNotParticipant):
		writeErr(w, http.StatusForbidden, "Отзыв могут оставить только участники")
	case errors.Is(err, fbdomain.ErrAlreadySubmitted):
		writeErr(w, http.StatusConflict, "Вы уже оставили отзыв")
	case errors.Is(err, fbdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
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
	isAdmin := u.Role == "admin"
	sum, err := h.deps.Feedback.ForOwner(r.Context(), eventID, u.UUID, isAdmin)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, sum)
	case errors.Is(err, fbdomain.ErrForbidden):
		writeErr(w, http.StatusForbidden, "Недостаточно прав")
	case errors.Is(err, fbdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "list failed")
	}
}

func (h *handler) mine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.FromString(r.URL.Query().Get("event_id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid event_id")
		return
	}
	submitted, err := h.deps.Feedback.MyFeedback(r.Context(), u.UUID, eventID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"submitted": submitted})
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

(Confirm `domain.User` has a `Role` field — it is synced from `users.role`, migration 000014, per the moderation foundation. If the field name differs, match it.)

- [ ] **Step 4: Wire it into the module**

In `backend/internal/http/module.go`:
1. Add imports `fbdomain "github.com/Pashteto/lia/internal/feedback"` and `feedbackhttp "github.com/Pashteto/lia/internal/http/feedback"`.
2. Add a field `feedback fbdomain.Service` to `Module` (next to `complaints`).
3. Add `func (m *Module) SetFeedback(svc fbdomain.Service) { m.feedback = svc }` (next to `SetComplaints`).
4. Build the handler near the complaints one:

```go
	var feedbackH http.Handler
	if m.feedback != nil {
		feedbackH = feedbackhttp.NewHandler(feedbackhttp.Deps{
			Authenticate: m.auth.Authenticate,
			Feedback:     m.feedback,
		})
	}
```

5. Mount in the router — **before** the complaints/base fallthrough, and note the `/me/feedback` path too:

```go
		if feedbackH != nil &&
			((strings.HasPrefix(p, "/api/v1/events/") && strings.HasSuffix(p, "/feedback")) ||
				p == "/api/v1/me/feedback") {
			feedbackH.ServeHTTP(w, r)
			return
		}
```

- [ ] **Step 5: Inject the service in application.go**

In `backend/internal/application.go`, near the `SetComplaints` call (~line 265), add:

```go
			httpModule.SetFeedback(
				feedback.NewService(feedback.NewRepository(repoModule.DB())),
			)
```

Add the import `"github.com/Pashteto/lia/internal/feedback"`.

- [ ] **Step 6: Run tests + full gate**

Run: `cd backend && go test ./internal/http/feedback/ ./internal/feedback/ -v && go build ./... && go vet ./... && golangci-lint run`
Expected: PASS / exit 0.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/http/feedback/ backend/internal/http/module.go backend/internal/application.go
git commit -m "feat(r3): feedback http handler + module wiring"
```

---

### Task 5: Frontend — submit block + organizer view

**Files:**
- Modify: `frontend/lib/api.ts` (add `submitFeedback`, `getEventFeedback`, `getMyFeedback`)
- Create: `frontend/components/FeedbackForm.tsx` (★ selector + comment)
- Modify: `frontend/components/EventDetailView.tsx` (render `FeedbackForm` when the event has ended)
- Modify: `frontend/app/me/practices/page.tsx` (past events → «Оставить отзыв» link/inline)
- Create: `frontend/components/OrganizerFeedback.tsx` (owner view: average + list) and surface it on `frontend/app/events/mine/page.tsx`

**Interfaces:**
- Consumes: the three backend endpoints (Task 4).
- Produces: `submitFeedback(eventId, rating, comment?)`, `getEventFeedback(eventId): Promise<{average, count, items}>`, `getMyFeedback(eventId): Promise<{submitted: boolean}>`.

- [ ] **Step 1: Add the API functions**

In `frontend/lib/api.ts` (mirror `submitComplaint` at line 593 for auth + error handling):

```ts
export interface FeedbackItem { rating: number; comment?: string; author_name: string; created_at: string; }
export interface FeedbackSummary { average: number; count: number; items: FeedbackItem[]; }

export async function submitFeedback(eventId: string, rating: number, comment?: string): Promise<void> {
  const token = getToken();
  const res = await fetch(`${API_V1}/events/${eventId}/feedback`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: JSON.stringify({ rating, comment: comment ?? "" }),
  });
  if (!res.ok) {
    const detail = await res.json().catch(() => ({}));
    throw new Error((detail as { error?: string }).error ?? `feedback failed: ${res.status}`);
  }
}

export async function getEventFeedback(eventId: string): Promise<FeedbackSummary> {
  const token = getToken();
  const res = await fetch(`${API_V1}/events/${eventId}/feedback`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error(`feedback fetch failed: ${res.status}`);
  return (await res.json()) as FeedbackSummary;
}

export async function getMyFeedback(eventId: string): Promise<boolean> {
  const token = getToken();
  const res = await fetch(`${API_V1}/me/feedback?event_id=${eventId}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) return false;
  return ((await res.json()) as { submitted: boolean }).submitted;
}
```

- [ ] **Step 2: Build the `FeedbackForm` component**

`frontend/components/FeedbackForm.tsx` — a client component: 5 large tappable ★ buttons (state `rating`), an optional comment `textarea`, a submit button. On submit call `submitFeedback`; on success render «Спасибо за отзыв 🙌»; render server error messages (403/409/422) inline in Russian. On mount call `getMyFeedback(eventId)` — if already submitted, show the thank-you state instead of the form.

- [ ] **Step 3: Show it on the event detail for ended events**

In `EventDetailView.tsx`, compute `ended = new Date(event.endsAt ?? event.startsAt) < new Date()`. When `ended` and the user is authed, render `<FeedbackForm eventId={event.id} />` in place of the RSVP CTA. (The server enforces participant-only; a non-participant sees the 403 message on submit — acceptable. `/me/practices` is the stronger entry point.)

- [ ] **Step 4: Add the `/me/practices` past entry point**

On the past tab of `frontend/app/me/practices/page.tsx`, add a «Оставить отзыв» link to the event (or inline `FeedbackForm`) for each past event — these rows are confirmed participants.

- [ ] **Step 5: Build the organizer view**

`frontend/components/OrganizerFeedback.tsx` — fetches `getEventFeedback(eventId)`; renders average ★ + count and the list (rating + comment + author name + date). Empty state: «Отзывы появятся после завершения события». Surface it on `frontend/app/events/mine/page.tsx` (in each event's expander) and/or link from the `/organizer` hub. Only the owner will get data (server-gated); on 403 render nothing.

- [ ] **Step 6: Build + lint + manual**

Run: `cd frontend && pnpm lint && pnpm build`
Manual (seed a past event with 2 participants via API): each participant leaves rating+comment on `/me/practices`; the owner sees the private average + comments on `/events/mine`; a non-participant submitting gets «только участники».

- [ ] **Step 7: Commit**

```bash
git add frontend/lib/api.ts frontend/components/FeedbackForm.tsx frontend/components/EventDetailView.tsx frontend/app/me/practices/page.tsx frontend/components/OrganizerFeedback.tsx frontend/app/events/mine/page.tsx
git commit -m "feat(r3): feedback submit block + organizer private view"
```

---

## Self-Review

- **Spec coverage:** ★1-5 + optional comment (Tasks 2/5) ✓; only participants with active RSVP (Task 2 gate) ✓; only after ended (Task 2 gate) ✓; one per (event,user) (Task 1 UNIQUE + Task 2 duplicate gate) ✓; private to owner+admin (Task 2 `ForOwner` + Task 4 `list`) ✓; nothing public (no public route/UI) ✓; author name not email (Task 3 join) ✓; average+count (Task 2 `ForOwner`) ✓; distinct Russian errors (Task 4) ✓; participant + organizer UI (Task 5) ✓; migration 019 (Task 1) ✓; independent of R1/R2 ✓.
- **Placeholder scan:** the integration test bodies in Task 3 Step 1 and the handler test in Task 4 Step 1 are described (mirror an existing test file) rather than fully transcribed — acceptable because they mirror a concrete existing file (`internal/http/complaints/handler_test.go`) and the domain unit tests in Task 2 are full. No "TODO"/"TBD".
- **Type consistency:** `Repository` method set is identical across Task 2 (interface), Task 3 (impl), and the fake (Task 2 test). `Service` methods `Submit`/`ForOwner`/`MyFeedback` match their handler call sites (Task 4). `Item`/`Summary` JSON field names (`author_name`, `average`, `count`, `items`) match the frontend `FeedbackItem`/`FeedbackSummary` (Task 5). Route strings match between Task 4 handler patterns and the module mount predicate.

## Deploy

Migration 018 → **019** — take a pre-migration DB dump per runbook convention. New backend domain + frontend. No `make generate-api` (no swagger model). Standard build-on-Mac→`save|ssh|load`. Verify: anon POST → 401, non-participant → 403, ended-participant → 201, duplicate → 409, owner GET → summary, non-owner GET → 403.
