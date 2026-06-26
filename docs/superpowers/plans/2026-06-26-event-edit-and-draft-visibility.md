# Event edit endpoint + draft visibility lockdown — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an owner edit and publish their own draft events via `PATCH /events/{id}`, and stop non-owners (incl. anonymous) from seeing or editing non-published events.

**Architecture:** Add an `Update` path through the existing events module (repository → service → handler), mirroring the existing venue update. Lock down the read path by giving `GET /events/{id}` optional auth + a status/ownership gate, and forcing the public `GET /events` list to published-only. Owner-facing lifecycle is `draft → published` (with `cancelled`); `pending_review`/`rejected` are dropped from the API surface but kept as legacy enum values.

**Tech Stack:** Go, go-pg v10, go-swagger (server generated from `backend/api/swagger.yaml` via `make generate-api`), gofrs/uuid.

## Global Constraints

- Module path: `github.com/Pashteto/lia`. All Go commands run from `backend/` (the Makefile, `api/swagger.yaml`, and `go.mod` live there).
- Codegen: after editing `backend/api/swagger.yaml`, regenerate with `make generate-api` (runs `swagger generate server`). Generated files under `internal/http/server/...` and `internal/http/models/...` are committed.
- Domain errors map to HTTP status in the handler layer: `ErrInvalidInput`→400, `ErrNotFound`→404, `ErrNotEditable`→409, anything else→503. Auth failures→401.
- Owner-settable statuses: `{draft, published, cancelled}` only. Editable-while: `draft` only. Non-owner access to a non-published event returns **404** (do not leak existence).
- Follow existing patterns: service unit tests use in-memory fakes (`internal/events/service_test.go`); repository DB methods are not unit-tested (mirror `internal/venues/repository.go` `Update`, which has no DB test).
- TDD, DRY, YAGNI, frequent commits.

---

### Task 1: Events domain — `Update` path + `ErrNotEditable` + published-only list filter

Adds the whole domain layer for editing (repository method, service method, params struct, sentinel error) with service-level TDD. This task is self-contained and has no dependency on codegen.

**Files:**
- Modify: `backend/internal/events/service.go` (add `ErrNotEditable`, `UpdateParams`, `Service.Update`, `notFound` helper)
- Modify: `backend/internal/events/repository.go` (add `Repository.Update` + `pgRepository.Update`)
- Modify: `backend/internal/events/service_test.go` (add `Update` + filter-capture to `mockRepo`; add tests)

**Interfaces:**
- Produces (consumed by Tasks 3–4):
  - `events.UpdateParams` struct (all pointer/optional fields):
    ```go
    type UpdateParams struct {
        Title       *string
        Description *string
        Format      *string
        PriceType   *string
        PriceMin    *int64
        PriceMax    *int64
        ExternalURL *string
        VenueID     *uuid.UUID
        CoverFileID *uuid.UUID
        CategoryIDs []uuid.UUID // nil = preserve, non-nil = replace
        StartsAt    *time.Time
        EndsAt      *time.Time
        Status      *string
    }
    ```
  - `events.Service.Update(ctx context.Context, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error)`
  - `events.ErrNotEditable error`

- [ ] **Step 1: Write the failing service tests**

Append to `backend/internal/events/service_test.go`. First extend `mockRepo` (add the `updated` capture, the `listFilter` capture, and the `Update` method; make `List` record the filter):

```go
// --- additions to mockRepo (place fields in the struct literal area) ---
// updated captures the event passed to Update.
// listFilter captures the filter passed to List.
// Add these fields to the mockRepo struct:
//     updated    *models.Event
//     updateErr  error
//     listFilter ListFilter

func (m *mockRepo) Update(event *models.Event) error {
	m.updated = event
	return m.updateErr
}
```

Change the existing `List` method to capture the filter:

```go
func (m *mockRepo) List(f ListFilter) ([]*models.Event, error) {
	m.listFilter = f
	return m.list, nil
}
```

Now add the tests:

```go
func ownedDraft(owner uuid.UUID) *models.Event {
	return &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		OrganizerID: owner,
		Title:       "Draft",
		Status:      models.EventDraft,
		StartsAt:    time.Now().Add(24 * time.Hour),
	}
}

func TestService_Update_NonOwner_ReturnsNotFound(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.Update(context.Background(), ev.ID, other, UpdateParams{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Update_PublishedIsLocked(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	ev.Status = models.EventPublished
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{})
	if !errors.Is(err, ErrNotEditable) {
		t.Fatalf("expected ErrNotEditable, got %v", err)
	}
}

func TestService_Update_AppliesOnlyProvidedFields(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	ev.Description = "keep me"
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	newTitle := "Updated Title"
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Title: &newTitle}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.Title != "Updated Title" {
		t.Fatalf("title not applied: %q", repo.updated.Title)
	}
	if repo.updated.Description != "keep me" {
		t.Fatalf("omitted field not preserved: %q", repo.updated.Description)
	}
}

func TestService_Update_PublishSetsPublishedAt(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	published := "published"
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Status: &published}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.Status != models.EventPublished {
		t.Fatalf("status not set to published: %v", repo.updated.Status)
	}
	if repo.updated.PublishedAt == nil {
		t.Fatal("expected PublishedAt to be set on publish")
	}
}

func TestService_Update_RejectsNonSettableStatus(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	pending := "pending_review"
	_, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Status: &pending})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/events/... 2>&1 | head -30`
Expected: compile failure — `mockRepo` has no field `updated`/`updateErr`/`listFilter` is unused, `svc.Update` undefined, `ErrNotEditable` undefined.

- [ ] **Step 3: Add the `Update` method to the repository interface and implementation**

In `backend/internal/events/repository.go`, add to the `Repository` interface (after `Create`):

```go
	// Update persists changes to an existing event. When event.CategoryIDs is
	// non-nil, the event_categories links are replaced to match. Returns
	// pg.ErrNoRows when no row matches.
	Update(event *models.Event) error
```

Add the implementation (place it after `pgRepository.Create`). Note `pg` is already imported as `github.com/go-pg/pg/v10`:

```go
func (r *pgRepository) Update(event *models.Event) error {
	logger.Log().Infof("updating event: %s", event.ID)

	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		res, err := tx.Model(event).
			Column(
				"title", "description", "venue_id", "cover_file_id", "status",
				"format", "price_type", "price_min", "price_max",
				"external_ticket_url", "starts_at", "ends_at", "published_at",
				"updated_at",
			).
			WherePK().
			Update()
		if err != nil {
			return fmt.Errorf("update event %s: %w", event.ID, err)
		}
		if res.RowsAffected() == 0 {
			return pg.ErrNoRows
		}

		// Replace category links only when the caller provided a new set.
		// A nil CategoryIDs means "preserve existing links".
		if event.CategoryIDs != nil {
			if _, err := tx.Exec(`DELETE FROM event_categories WHERE event_id = ?`, event.ID); err != nil {
				return fmt.Errorf("clear event %s categories: %w", event.ID, err)
			}
			for _, cid := range event.CategoryIDs {
				if _, err := tx.Exec(
					`INSERT INTO event_categories (event_id, category_id) VALUES (?, ?)
					 ON CONFLICT DO NOTHING`,
					event.ID, cid,
				); err != nil {
					return fmt.Errorf("link event %s to category %s: %w", event.ID, cid, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update event %s: %w", event.ID, err)
	}
	return nil
}
```

- [ ] **Step 4: Add `ErrNotEditable`, `UpdateParams`, the not-found helper, and `Service.Update`**

In `backend/internal/events/service.go`, add to the error block:

```go
	// ErrNotEditable indicates the event is in a status that cannot be edited
	// (only draft events are editable).
	ErrNotEditable = errors.New("not editable")
```

Add `Update` to the `Service` interface (after `Create`):

```go
	// Update applies a partial update to an event owned by ownerID. Only draft
	// events are editable; non-owners get ErrNotFound (existence is not leaked).
	Update(ctx context.Context, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error)
```

Add the `UpdateParams` type (near the top, after the `Service` interface):

```go
// UpdateParams is a partial event update. A nil pointer field means "preserve
// the current value"; a non-nil field overwrites it. CategoryIDs is nil to
// preserve, non-nil to replace the category set.
type UpdateParams struct {
	Title       *string
	Description *string
	Format      *string
	PriceType   *string
	PriceMin    *int64
	PriceMax    *int64
	ExternalURL *string
	VenueID     *uuid.UUID
	CoverFileID *uuid.UUID
	CategoryIDs []uuid.UUID
	StartsAt    *time.Time
	EndsAt      *time.Time
	Status      *string
}

// ownerSettableStatus reports whether an owner may set the given status via the
// edit endpoint. Moderation statuses (pending_review, rejected) are excluded.
func ownerSettableStatus(s models.EventStatus) bool {
	switch s {
	case models.EventDraft, models.EventPublished, models.EventCancelled:
		return true
	default:
		return false
	}
}

// isNoRows reports whether err is (or wraps) a go-pg "no rows" error. Mirrors
// the detection used in GetByID.
func isNoRows(err error) bool {
	if wrapped := errors.Unwrap(err); wrapped != nil &&
		wrapped.Error() == "pg: no rows in result set" {
		return true
	}
	return false
}
```

Add the `Update` method (after `GetByID`):

```go
func (s *service) Update(ctx context.Context, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidInput)
	}

	event, err := s.repo.GetByID(id)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	// Non-owner access is indistinguishable from not-found (no existence leak).
	if event.OrganizerID != ownerID {
		return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
	}

	// Only drafts are editable.
	if event.Status != models.EventDraft {
		return nil, fmt.Errorf("%w: event %s is %s", ErrNotEditable, id, event.Status)
	}

	if p.Title != nil {
		event.Title = *p.Title
	}
	if p.Description != nil {
		event.Description = *p.Description
	}
	if p.Format != nil {
		event.Format = *p.Format
	}
	if p.PriceType != nil {
		event.PriceType = *p.PriceType
	}
	if p.PriceMin != nil {
		event.PriceMin = p.PriceMin
	}
	if p.PriceMax != nil {
		event.PriceMax = p.PriceMax
	}
	if p.ExternalURL != nil {
		event.ExternalURL = *p.ExternalURL
	}
	if p.VenueID != nil {
		event.VenueID = *p.VenueID
	}
	if p.CoverFileID != nil {
		event.CoverFileID = *p.CoverFileID
	}
	if p.StartsAt != nil {
		event.StartsAt = *p.StartsAt
	}
	if p.EndsAt != nil {
		event.EndsAt = p.EndsAt
	}

	if p.Status != nil {
		target, err := models.EventStatusFromString(*p.Status)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		if !ownerSettableStatus(target) {
			return nil, fmt.Errorf("%w: status %q is not settable", ErrInvalidInput, *p.Status)
		}
		event.Status = target
		if target == models.EventPublished && event.PublishedAt == nil {
			now := time.Now()
			event.PublishedAt = &now
		}
	}

	if p.CategoryIDs != nil {
		resolved, err := s.categories.Validate(ctx, p.CategoryIDs)
		if err != nil {
			if errors.Is(err, categories.ErrInvalidInput) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("validate categories: %w", err)
		}
		event.CategoryIDs = p.CategoryIDs
		event.Categories = resolved
	}

	if p.VenueID != nil {
		venue, err := s.venues.Validate(ctx, event.VenueID)
		if err != nil {
			if errors.Is(err, venues.ErrInvalidInput) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("validate venue: %w", err)
		}
		event.Venue = venue
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	if err := s.repo.Update(event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	reloaded, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("reload event: %w", err)
	}
	return reloaded, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/events/... 2>&1 | tail -20`
Expected: PASS (all events tests, including the five new `TestService_Update_*`).

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/events/service.go internal/events/repository.go internal/events/service_test.go
git commit -m "feat(events): add owner Update service + repository method

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Swagger spec — add `EventPatch` + `PATCH /events/{id}`, lock list to published, restrict `EventInput.status`

Edits the API contract and regenerates the server. After this task the generated `apimodels.EventPatch`, the `events.UpdateEventParams`/responders, and the `ListEventsParams` without `Status` exist.

**Files:**
- Modify: `backend/api/swagger.yaml`
- Regenerate: `backend/internal/http/models/*`, `backend/internal/http/server/operations/events/*` (via `make generate-api`)

**Interfaces:**
- Produces (consumed by Tasks 3–5):
  - `apimodels.EventPatch` (fields: `Title string`, `Description string`, `Format string`, `PriceType string`, `PriceMin int64`, `PriceMax int64`, `ExternalTicketURL string`, `VenueID strfmt.UUID`, `CoverFileID *strfmt.UUID`, `CategoryIds []strfmt.UUID`, `StartsAt strfmt.DateTime`, `EndsAt strfmt.DateTime`, `Status string`)
  - `eventsops.UpdateEventParams{ HTTPRequest, ID strfmt.UUID, Body *apimodels.EventPatch }`
  - Responders: `NewUpdateEventOK().WithPayload(*apimodels.Event)`, `NewUpdateEventBadRequest()`, `NewUpdateEventUnauthorized()`, `NewUpdateEventNotFound()`, `NewUpdateEventConflict()`, `NewUpdateEventServiceUnavailable()`
  - `api.EventsUpdateEventHandler` field on `operations.LiaAPIAPI`
  - `eventsops.ListEventsParams` no longer has a `Status` field

- [ ] **Step 1: Restrict `EventInput.status` enum**

In `backend/api/swagger.yaml`, under `EventInput.properties.status.enum`, change the list to only:

```yaml
      status:
        type: string
        enum:
          - draft
          - published
          - cancelled
```

- [ ] **Step 2: Add the `EventPatch` definition**

In `backend/api/swagger.yaml`, in the `definitions:` block (next to `EventInput`), add:

```yaml
  EventPatch:
    type: object
    description: >-
      Partial update for an event. Every field is optional; omitted fields are
      preserved. Only draft events can be patched.
    properties:
      title:
        type: string
        minLength: 1
        maxLength: 255
      description:
        type: string
      category_ids:
        type: array
        items:
          type: string
          format: uuid
      status:
        type: string
        enum:
          - draft
          - published
          - cancelled
      format:
        type: string
        enum:
          - offline
          - online
      price_type:
        type: string
        enum:
          - free
          - fixed
          - from
      price_min:
        type: integer
        format: int64
      price_max:
        type: integer
        format: int64
      external_ticket_url:
        type: string
      starts_at:
        type: string
        format: date-time
      ends_at:
        type: string
        format: date-time
      venue_id:
        type: string
        format: uuid
      cover_file_id:
        type: string
        format: uuid
        x-nullable: true
```

- [ ] **Step 3: Add the `patch` operation under `/events/{id}`**

In `backend/api/swagger.yaml`, under the `/events/{id}:` path (alongside the existing `get:`), add a sibling `patch:` key:

```yaml
    patch:
      summary: Update an event
      description: >-
        Updates an event owned by the authenticated user. Only draft events are
        editable; setting status to "published" publishes the event. Non-owners
        receive 404 (existence is not leaked).
      operationId: updateEvent
      tags:
        - events
      security:
        - jwt: []
      parameters:
        - name: id
          in: path
          description: Event UUID
          required: true
          type: string
          format: uuid
        - name: body
          in: body
          required: true
          schema:
            $ref: "#/definitions/EventPatch"
      responses:
        200:
          description: Updated event
          schema:
            $ref: "#/definitions/Event"
        400:
          description: Invalid input
          schema:
            $ref: "#/definitions/Error"
        401:
          description: Unauthorized
          schema:
            $ref: "#/definitions/Error"
        404:
          description: Event not found or not owned by caller
          schema:
            $ref: "#/definitions/Error"
        409:
          description: Event is not in an editable status
          schema:
            $ref: "#/definitions/Error"
        503:
          description: Service unavailable (database disabled)
          schema:
            $ref: "#/definitions/Error"
```

- [ ] **Step 4: Remove the `status` query param from `GET /events`**

In `backend/api/swagger.yaml`, in the `/events:` `get:` operation, delete the `status` query parameter entry (the `parameters:` item with `name: status`, `in: query`). If `status` is the only parameter, remove the `parameters:` key entirely so the operation has none. Leave the rest of the operation (responses) unchanged.

- [ ] **Step 5: Validate and regenerate**

Run: `cd backend && make swagger-validate`
Expected: `Swagger spec is valid`

Run: `cd backend && make generate-api`
Expected: completes without error; regenerates files under `internal/http/server/...` and `internal/http/models/...`.

- [ ] **Step 6: Verify the generated symbols and that the tree still builds**

Run: `cd backend && ls internal/http/server/operations/events/ | grep update_event && ls internal/http/models/event_patch.go`
Expected: `update_event*.go` files exist and `internal/http/models/event_patch.go` exists.

Run: `cd backend && go build ./... 2>&1 | head -30`
Expected: it FAILS only in `internal/http/handlers` (the `ListEvents` handler still references the now-removed `params.Status`). This is fixed in Task 5. Confirm there are no other failures (no errors mentioning `internal/events`, `internal/http/models`, or `internal/http/server`).

- [ ] **Step 7: Commit**

```bash
cd backend && git add api/swagger.yaml internal/http/models internal/http/server
git commit -m "feat(api): add PATCH /events/{id}, EventPatch; lock list to published

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Formatter — map `EventPatch` → `UpdateParams`, flip create default to draft

**Files:**
- Modify: `backend/internal/http/formatter/event.go`
- Modify: `backend/internal/http/formatter/event_test.go`

**Interfaces:**
- Consumes: `apimodels.EventPatch` (Task 2), `events.UpdateParams` (Task 1)
- Produces (consumed by Task 4): `formatter.EventPatchToUpdateParams(in *apiModels.EventPatch) eventsdomain.UpdateParams`

- [ ] **Step 1: Write the failing formatter tests**

Append to `backend/internal/http/formatter/event_test.go`:

```go
func TestEventFromAPIInput_DefaultsToDraft(t *testing.T) {
	title := "X"
	starts := strfmt.DateTime(time.Now())
	in := &apiModels.EventInput{Title: &title, StartsAt: &starts}

	ev, err := EventFromAPIInput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Status != domainModels.EventDraft {
		t.Fatalf("expected default status draft, got %v", ev.Status)
	}
}

func TestEventPatchToUpdateParams_MapsProvidedFields(t *testing.T) {
	title := "New"
	in := &apiModels.EventPatch{Title: title, Status: "published"}

	p := EventPatchToUpdateParams(in)
	if p.Title == nil || *p.Title != "New" {
		t.Fatalf("title not mapped: %+v", p.Title)
	}
	if p.Status == nil || *p.Status != "published" {
		t.Fatalf("status not mapped: %+v", p.Status)
	}
	if p.Description != nil {
		t.Fatalf("omitted field should be nil, got %+v", p.Description)
	}
}
```

Note: confirm the test file's imports include `apiModels`, `domainModels`, `strfmt`, and `time` (the existing tests in this file already use these aliases; add any missing import).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/http/formatter/... 2>&1 | head -30`
Expected: FAIL — `EventPatchToUpdateParams` undefined, and `TestEventFromAPIInput_DefaultsToDraft` fails because the current default is `"published"`.

- [ ] **Step 3: Flip the create default to draft**

In `backend/internal/http/formatter/event.go`, in `EventFromAPIInput`, change:

```go
	status := defaultStr(in.Status, "published")
```

to:

```go
	status := defaultStr(in.Status, "draft")
```

Also update the doc comment above `EventFromAPIInput` to say the default is `draft` (replace the sentence that claims new events default to published/visible).

- [ ] **Step 4: Add the `EventPatchToUpdateParams` mapper**

In `backend/internal/http/formatter/event.go`, add the events domain import to the import block:

```go
	eventsdomain "github.com/Pashteto/lia/internal/events"
```

Add the function (the helpers `parseOptionalUUID` already exist in this file):

```go
// EventPatchToUpdateParams converts an API EventPatch into the domain
// UpdateParams. A zero/empty API value maps to a nil pointer ("preserve");
// category_ids maps to nil when absent (preserve) and to a slice when present
// (replace). Clearing a field to empty is not supported via PATCH.
func EventPatchToUpdateParams(in *apiModels.EventPatch) eventsdomain.UpdateParams {
	var p eventsdomain.UpdateParams
	if in == nil {
		return p
	}
	if in.Title != "" {
		v := in.Title
		p.Title = &v
	}
	if in.Description != "" {
		v := in.Description
		p.Description = &v
	}
	if in.Format != "" {
		v := in.Format
		p.Format = &v
	}
	if in.PriceType != "" {
		v := in.PriceType
		p.PriceType = &v
	}
	if in.PriceMin != 0 {
		v := in.PriceMin
		p.PriceMin = &v
	}
	if in.PriceMax != 0 {
		v := in.PriceMax
		p.PriceMax = &v
	}
	if in.ExternalTicketURL != "" {
		v := in.ExternalTicketURL
		p.ExternalURL = &v
	}
	if id, ok := parseOptionalUUID(in.VenueID); ok {
		p.VenueID = &id
	}
	if in.CoverFileID != nil {
		if id, ok := parseOptionalUUID(*in.CoverFileID); ok {
			p.CoverFileID = &id
		}
	}
	if !time.Time(in.StartsAt).IsZero() {
		t := time.Time(in.StartsAt)
		p.StartsAt = &t
	}
	if !time.Time(in.EndsAt).IsZero() {
		t := time.Time(in.EndsAt)
		p.EndsAt = &t
	}
	if in.Status != "" {
		v := in.Status
		p.Status = &v
	}
	if len(in.CategoryIds) > 0 {
		ids := make([]uuid.UUID, 0, len(in.CategoryIds))
		for _, raw := range in.CategoryIds {
			if parsed, err := uuid.FromString(raw.String()); err == nil {
				ids = append(ids, parsed)
			}
		}
		p.CategoryIDs = ids
	}
	return p
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/http/formatter/... 2>&1 | tail -20`
Expected: PASS. If a pre-existing test asserted the old `published` default, update that assertion to `domainModels.EventDraft` and re-run.

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/http/formatter/event.go internal/http/formatter/event_test.go
git commit -m "feat(formatter): EventPatch->UpdateParams mapper; default create to draft

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: `UpdateEvent` handler + service-interface mock + wiring

**Files:**
- Create: `backend/internal/http/handlers/events_update.go`
- Modify: `backend/internal/http/handlers/events_test.go` (add `Update` to `mockEventsService`; add handler tests)
- Modify: `backend/internal/http/module.go` (wire `EventsUpdateEventHandler`)

**Interfaces:**
- Consumes: `eventsops.UpdateEventParams` + responders (Task 2), `formatter.EventPatchToUpdateParams` (Task 3), `eventsdomain.Service.Update` / `ErrNotEditable` / `ErrNotFound` / `ErrInvalidInput` (Task 1)
- Produces: `handlers.NewUpdateEvent(svc eventsdomain.Service) *UpdateEvent`

- [ ] **Step 1: Add `Update` to the handler test's `mockEventsService` and write failing tests**

In `backend/internal/http/handlers/events_test.go`, add to `mockEventsService` a capture + method:

```go
// add fields to mockEventsService:
//     updated     *domainmodels.Event
//     updateErr   error
//     updateOwner uuid.UUID

func (m *mockEventsService) Update(_ context.Context, id, ownerID uuid.UUID, _ eventsdomain.UpdateParams) (*domainmodels.Event, error) {
	m.updateOwner = ownerID
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	ev := &domainmodels.Event{ID: id, OrganizerID: ownerID, Title: "T", Status: domainmodels.EventDraft, StartsAt: time.Now()}
	m.updated = ev
	return ev, nil
}
```

Add the tests:

```go
func updateParams(t *testing.T) eventsops.UpdateEventParams {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/events/x", nil)
	return eventsops.UpdateEventParams{
		HTTPRequest: req,
		ID:          strfmt.UUID(uuid.Must(uuid.NewV4()).String()),
		Body:        &models.EventPatch{},
	}
}

func testPrincipal() *models.User {
	pu := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	return &models.User{UUID: pu, Email: &email, Name: &name, Status: &status}
}

func TestUpdateEvent_Unauthenticated_Returns401(t *testing.T) {
	h := NewUpdateEvent(&mockEventsService{})
	resp := h.Handle(updateParams(t), nil)
	if _, ok := resp.(*eventsops.UpdateEventUnauthorized); !ok {
		t.Fatalf("expected *UpdateEventUnauthorized, got %T", resp)
	}
}

func TestUpdateEvent_NotFound_Returns404(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: event x", eventsdomain.ErrNotFound)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventNotFound); !ok {
		t.Fatalf("expected *UpdateEventNotFound, got %T", resp)
	}
}

func TestUpdateEvent_Locked_Returns409(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: event x is published", eventsdomain.ErrNotEditable)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventConflict); !ok {
		t.Fatalf("expected *UpdateEventConflict, got %T", resp)
	}
}

func TestUpdateEvent_Invalid_Returns400(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: bad", eventsdomain.ErrInvalidInput)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventBadRequest); !ok {
		t.Fatalf("expected *UpdateEventBadRequest, got %T", resp)
	}
}

func TestUpdateEvent_Success_Returns200(t *testing.T) {
	h := NewUpdateEvent(&mockEventsService{})
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventOK); !ok {
		t.Fatalf("expected *UpdateEventOK, got %T", resp)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/http/handlers/... 2>&1 | head -30`
Expected: FAIL — `NewUpdateEvent` undefined (and `mockEventsService` now satisfies the interface once Task 1's interface method exists).

- [ ] **Step 3: Implement the `UpdateEvent` handler**

Create `backend/internal/http/handlers/events_update.go`:

```go
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	"github.com/Pashteto/lia/pkg/logger"
)

// UpdateEvent handler applies a partial update to an event owned by the caller.
type UpdateEvent struct {
	events eventsdomain.Service
}

// NewUpdateEvent creates an UpdateEvent handler.
func NewUpdateEvent(svc eventsdomain.Service) *UpdateEvent {
	return &UpdateEvent{events: svc}
}

// Handle PATCH /events/{id}.
func (h *UpdateEvent) Handle(params eventsops.UpdateEventParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return eventsops.NewUpdateEventUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	ownerID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return eventsops.NewUpdateEventUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	id, err := uuid.FromString(params.ID.String())
	if err != nil {
		return eventsops.NewUpdateEventBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}

	p := formatter.EventPatchToUpdateParams(params.Body)

	updated, err := h.events.Update(params.HTTPRequest.Context(), id, ownerID, p)
	if err != nil {
		logger.Log().Errorf("update event %s: %s", id, err.Error())
		switch {
		case errors.Is(err, eventsdomain.ErrInvalidInput):
			return eventsops.NewUpdateEventBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, eventsdomain.ErrNotFound):
			return eventsops.NewUpdateEventNotFound().
				WithPayload(DefaultError(http.StatusNotFound, err, nil))
		case errors.Is(err, eventsdomain.ErrNotEditable):
			return eventsops.NewUpdateEventConflict().
				WithPayload(DefaultError(http.StatusConflict, err, nil))
		default:
			return eventsops.NewUpdateEventServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	return eventsops.NewUpdateEventOK().WithPayload(formatter.EventToAPI(updated))
}
```

- [ ] **Step 4: Wire the handler in `module.go`**

In `backend/internal/http/module.go`, inside the `if m.events != nil {` block, add:

```go
		api.EventsUpdateEventHandler = handlers.NewUpdateEvent(m.events)
```

- [ ] **Step 5: Run the handler tests to verify they pass**

Run: `cd backend && go test ./internal/http/handlers/... 2>&1 | tail -20`
Expected: PASS (the five new `TestUpdateEvent_*`). Note: the package still won't fully build until Task 5 fixes `ListEvents`; if `go test` reports the `params.Status` build error, proceed to Task 5 and re-run this command at Task 5 Step 5.

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/http/handlers/events_update.go internal/http/handlers/events_test.go internal/http/module.go
git commit -m "feat(http): add UpdateEvent handler + wiring

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Draft visibility — optional auth on GetByID + published-only list

**Files:**
- Modify: `backend/internal/http/handlers/events.go` (`GetEventByID` optional auth + gate; `ListEvents` forces published)
- Modify: `backend/internal/http/handlers/events_test.go` (visibility + list tests)
- Modify: `backend/internal/http/module.go` (pass `m.auth.CheckAuth` to `NewGetEventByID`)

**Interfaces:**
- Consumes: `m.auth.CheckAuth` — signature `func(token string) (*apimodels.User, error)` (from `internal/http/auth`)
- Produces: `handlers.NewGetEventByID(svc eventsdomain.Service, checkAuth func(string) (*apimodels.User, error)) *GetEventByID`

- [ ] **Step 1: Write the failing visibility + list tests**

In `backend/internal/http/handlers/events_test.go`, add a `GetByID` capability to `mockEventsService` and a `List` capture. Update the existing stub methods:

```go
// replace the existing stub bodies on mockEventsService:
//   - add field:  getByID *domainmodels.Event
//   - add field:  listStatusArg string

func (m *mockEventsService) GetByID(context.Context, string) (*domainmodels.Event, error) {
	return m.getByID, nil
}
func (m *mockEventsService) List(_ context.Context, status string) ([]*domainmodels.Event, error) {
	m.listStatusArg = status
	return nil, nil
}
```

Add the tests:

```go
func getByIDParams(t *testing.T, auth string) eventsops.GetEventByIDParams {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events/x", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return eventsops.GetEventByIDParams{
		HTTPRequest: req,
		ID:          strfmt.UUID(uuid.Must(uuid.NewV4()).String()),
	}
}

func TestGetEventByID_AnonymousDraft_Returns404(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: owner,
		Status: domainmodels.EventDraft, Title: "D", StartsAt: time.Now(),
	}}
	// checkAuth returns unauthorized for any token (anonymous caller).
	h := NewGetEventByID(svc, func(string) (*models.User, error) { return nil, errors.New("no auth") })

	resp := h.Handle(getByIDParams(t, ""))
	if _, ok := resp.(*eventsops.GetEventByIDNotFound); !ok {
		t.Fatalf("expected *GetEventByIDNotFound, got %T", resp)
	}
}

func TestGetEventByID_OwnerDraft_Returns200(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: owner,
		Status: domainmodels.EventDraft, Title: "D", StartsAt: time.Now(),
	}}
	ownerPrincipal := &models.User{}
	pu := strfmt.UUID(owner.String())
	ownerPrincipal.UUID = pu
	h := NewGetEventByID(svc, func(string) (*models.User, error) { return ownerPrincipal, nil })

	resp := h.Handle(getByIDParams(t, "Bearer tok"))
	if _, ok := resp.(*eventsops.GetEventByIDOK); !ok {
		t.Fatalf("expected *GetEventByIDOK, got %T", resp)
	}
}

func TestGetEventByID_AnonymousPublished_Returns200(t *testing.T) {
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()),
		Status: domainmodels.EventPublished, Title: "P", StartsAt: time.Now(),
	}}
	h := NewGetEventByID(svc, func(string) (*models.User, error) { return nil, errors.New("no auth") })

	resp := h.Handle(getByIDParams(t, ""))
	if _, ok := resp.(*eventsops.GetEventByIDOK); !ok {
		t.Fatalf("expected *GetEventByIDOK, got %T", resp)
	}
}

func TestListEvents_ForcesPublished(t *testing.T) {
	svc := &mockEventsService{}
	h := NewListEvents(svc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events", nil)
	h.Handle(eventsops.ListEventsParams{HTTPRequest: req})
	if svc.listStatusArg != "published" {
		t.Fatalf("expected list to force published, got %q", svc.listStatusArg)
	}
}
```

Ensure the test file imports `errors` (add it if missing).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/http/handlers/... 2>&1 | head -30`
Expected: FAIL — `NewGetEventByID` takes one arg (signature mismatch) and `ListEvents` still references `params.Status` (build error).

- [ ] **Step 3: Update `GetEventByID` for optional auth + status gate**

In `backend/internal/http/handlers/events.go`, replace the `GetEventByID` struct, constructor, and `Handle` with:

```go
// GetEventByID handler returns a single event by UUID. Non-published events are
// visible only to their owner; everyone else gets 404 (existence not leaked).
type GetEventByID struct {
	events    eventsdomain.Service
	checkAuth func(string) (*apimodels.User, error)
}

// NewGetEventByID creates a GetEventByID handler. checkAuth resolves the caller
// from the Authorization header; it may be nil (treated as always-anonymous).
func NewGetEventByID(svc eventsdomain.Service, checkAuth func(string) (*apimodels.User, error)) *GetEventByID {
	return &GetEventByID{events: svc, checkAuth: checkAuth}
}

// Handle GET /events/{id}.
func (h *GetEventByID) Handle(params eventsops.GetEventByIDParams) middleware.Responder {
	event, err := h.events.GetByID(params.HTTPRequest.Context(), params.ID.String())
	if err != nil {
		logger.Log().Errorf("get event %s: %s", params.ID.String(), err.Error())
		switch {
		case errors.Is(err, eventsdomain.ErrInvalidInput):
			return eventsops.NewGetEventByIDBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, eventsdomain.ErrNotFound):
			return eventsops.NewGetEventByIDNotFound().
				WithPayload(DefaultError(http.StatusNotFound, err, nil))
		default:
			return eventsops.NewGetEventByIDServiceUnavailable().
				WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
		}
	}

	// Non-published events are visible only to the owner.
	if event.Status.String() != "published" && !h.callerOwns(params, event.OrganizerID.String()) {
		return eventsops.NewGetEventByIDNotFound().
			WithPayload(DefaultError(http.StatusNotFound, errors.New("event not found"), nil))
	}

	return eventsops.NewGetEventByIDOK().WithPayload(formatter.EventToAPI(event))
}

// callerOwns reports whether the (optional) authenticated caller owns the event.
// Any auth failure is treated as anonymous (not the owner).
func (h *GetEventByID) callerOwns(params eventsops.GetEventByIDParams, organizerID string) bool {
	if h.checkAuth == nil {
		return false
	}
	u, err := h.checkAuth(params.HTTPRequest.Header.Get("Authorization"))
	if err != nil || u == nil {
		return false
	}
	return u.UUID.String() == organizerID
}
```

- [ ] **Step 4: Force `ListEvents` to published-only**

In `backend/internal/http/handlers/events.go`, replace the body of `ListEvents.Handle` so it no longer reads `params.Status`:

```go
// Handle GET /events. The public list only returns published events; owners use
// GET /events/mine to see their drafts.
func (h *ListEvents) Handle(params eventsops.ListEventsParams) middleware.Responder {
	list, err := h.events.List(params.HTTPRequest.Context(), "published")
	if err != nil {
		logger.Log().Errorf("list events: %s", err.Error())
		if errors.Is(err, eventsdomain.ErrInvalidInput) {
			return eventsops.NewListEventsBadRequest().
				WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		}
		return eventsops.NewListEventsServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Event, 0, len(list))
	for _, e := range list {
		payload = append(payload, formatter.EventToAPI(e))
	}

	return eventsops.NewListEventsOK().WithPayload(payload)
}
```

- [ ] **Step 5: Update `module.go` wiring**

In `backend/internal/http/module.go`, change the GetEventByID registration to pass the auth checker:

```go
		api.EventsGetEventByIDHandler = handlers.NewGetEventByID(m.events, m.auth.CheckAuth)
```

- [ ] **Step 6: Run the full backend test + build**

Run: `cd backend && go build ./... 2>&1 | head -20`
Expected: builds clean (no errors).

Run: `cd backend && go test ./... 2>&1 | tail -30`
Expected: PASS across packages (events, formatter, handlers, etc.).

- [ ] **Step 7: Lint**

Run: `cd backend && make lint 2>&1 | tail -20` (or `golangci-lint run` if `make lint` is unavailable)
Expected: no new lint errors in the files touched.

- [ ] **Step 8: Commit**

```bash
cd backend && git add internal/http/handlers/events.go internal/http/handlers/events_test.go internal/http/module.go
git commit -m "feat(http): owner-only draft visibility on GET /events/{id}; list published-only

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Self-review notes (coverage vs. spec)

- **PATCH endpoint, owner-only, editable-while-draft, publish-via-PATCH** → Tasks 1 (service rules), 2 (contract), 4 (handler). ✅
- **404 for non-owner / non-existent** → Task 1 (`ErrNotFound` for non-owner), Tasks 4 & 5 (404 mapping/gate). ✅
- **409 for locked (published/cancelled) status** → Task 1 (`ErrNotEditable`), Task 4 (409 mapping). ✅
- **Draft visibility on GET /events/{id} (optional auth)** → Task 5. ✅
- **Public GET /events published-only + status param removed** → Task 2 (spec), Task 5 (handler + test). ✅
- **Status set restricted to {draft, published, cancelled}; pending_review/rejected dropped from API** → Task 1 (`ownerSettableStatus`), Task 2 (`EventInput`/`EventPatch` enums). Legacy enum values untouched in DB/Go. ✅
- **Create default flips to draft** → Task 3. ✅
- **Partial-update convention (zero = preserve; category_ids non-nil = replace)** → Task 1 (service), Task 3 (mapper). ✅
- **Non-goals (no moderation actor, no admin bypass, no cancel-of-published, no enum migration)** → respected; nothing in the plan adds them. ✅

No placeholders; signatures are consistent across tasks (`Update(ctx, id, ownerID, UpdateParams)`, `EventPatchToUpdateParams`, `NewGetEventByID(svc, checkAuth)`, `NewUpdateEvent(svc)`).

---

## Deployment status (2026-06-26)

- **Backend** (`PATCH /events/{id}` owner-edit, draft default, owner-only draft
  visibility) and the **publish-draft** frontend button: **LIVE** on
  `https://api.lia.pashteto.com` / `https://lia.pashteto.com` — shipped as part
  of the full-stack cutover, see
  `../runbooks/2026-06-26-rsvp-moderation-fullstack-deploy.md` (no migration was
  needed for event-edit itself; that deploy's migrations 012–014 are RSVP /
  moderation).
- **Frontend navigation follow-up** (back links on secondary pages + persistent
  mobile TabBar): committed (`aee829b`, `1be101a`, `0f3cd05`), **DEPLOYED live
  2026-06-26** (frontend-only, no migration; verified 200 + back link in `/map`
  HTML) — `../runbooks/2026-06-26-nav-back-buttons-frontend-redeploy.md`.
  Backend was NOT redeployed this round: the working tree carries an untracked
  `000015_organizers` migration from concurrent work, so a backend ship would
  apply unreviewed migration/code to prod — deferred until that's committed.
- All work sits on the shared branch `feat/event-edit-and-draft-visibility`,
  intermixed with concurrent RSVP/moderation/liquid-glass commits — not a clean
  feature branch; coordinate before any `main` merge.
