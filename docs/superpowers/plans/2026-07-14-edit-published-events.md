# Editing Published Events (R2) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an event owner edit a **published** event (title, description, dates, venue, cover, categories, price, and capacity) — the event stays published — with capacity changes reconciling the waitlist (FIFO promotion) and every published edit written to `audit_log`.

**Architecture:** `events.Service.Update` currently rejects any non-draft status. We relax the gate to allow `draft`+`published`, add `Capacity` to `UpdateParams`, lock `signup_mode` on published events, own the capacity column through a dedicated transactional repo method (`SetCapacityTx`) that guards against shrinking below occupancy and promotes the waitlist, and write an `audit_log` row on published edits (mirroring `internal/moderation/repository.go:36-41`). Frontend gets an `/events/[id]/edit` page reusing `CreateEventForm` in edit mode plus «Редактировать» entry points.

**Tech Stack:** Go (go-pg, gofrs/uuid), Next.js 15/TS/react-hook-form/Zod. Spec: `docs/superpowers/specs/2026-07-14-edit-published-events-design.md`. Depends on **R1** (shared form fields) — land R1 first.

## Global Constraints

- All user-facing copy in **Russian**.
- No DB migration — schema stays **018**. Reuses `audit_log`, `event_rsvps`, `events.capacity`.
- **`signup_mode` cannot change on a non-draft event** → 422. Only content fields + capacity are editable after publish.
- **Capacity cannot be reduced below occupied seats** (`going`+`accepted`) → 409 with the count. No attendee is ever evicted.
- Capacity reconciliation (guard + column write + waitlist promotion) MUST be atomic (one transaction), race-safe under `FOR UPDATE` on the event row (mirror `internal/rsvp/repository.go:88-134`).
- Editing keeps status `published` (no re-moderation, owner never sets `pending_review`).
- Backend rebuilds need `make generate-api` first (gitignored swagger model). Do NOT use `make generate-all`.

---

### Task 1: Map `capacity` into `UpdateParams`

**Files:**
- Modify: `backend/internal/events/service.go:76-93` (`UpdateParams`)
- Modify: `backend/internal/http/formatter/event.go:314` (remove the "deferred" note, map capacity)
- Test: `backend/internal/http/formatter/event_test.go`

**Interfaces:**
- Produces: `UpdateParams.Capacity *int` (nil = preserve). `EventPatchToUpdateParams` sets it from `EventPatch.Capacity *int64`. Consumed by Task 3.

- [ ] **Step 1: Failing formatter test**

Add to `backend/internal/http/formatter/event_test.go`:

```go
func TestEventPatchToUpdateParams_Capacity(t *testing.T) {
	cap := int64(10)
	p := formatter.EventPatchToUpdateParams(&apiModels.EventPatch{Capacity: &cap})
	if p.Capacity == nil || *p.Capacity != 10 {
		t.Fatalf("want capacity 10, got %v", p.Capacity)
	}
	p2 := formatter.EventPatchToUpdateParams(&apiModels.EventPatch{})
	if p2.Capacity != nil {
		t.Fatalf("want nil capacity when omitted, got %v", *p2.Capacity)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/http/formatter/ -run TestEventPatchToUpdateParams_Capacity -v`
Expected: FAIL — `p.Capacity` undefined (field missing).

- [ ] **Step 3: Add the field + mapping**

In `events/service.go` `UpdateParams`, add after `ExternalRegistrationURL *string`:

```go
	Capacity *int
```

In `formatter/event.go`, replace the `// capacity is set at create; PATCH support deferred` line with:

```go
	if in.Capacity != nil {
		c := int(*in.Capacity)
		p.Capacity = &c
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test ./internal/http/formatter/ -run TestEventPatchToUpdateParams_Capacity -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/events/service.go backend/internal/http/formatter/event.go backend/internal/http/formatter/event_test.go
git commit -m "feat(r2): map capacity into UpdateParams"
```

---

### Task 2: Repo — occupancy-guarded capacity change with waitlist promotion

**Files:**
- Modify: `backend/internal/events/repository.go` (add `SetCapacityTx`, `WriteEditAudit`)
- Modify: `backend/internal/events/service.go` (Repository interface — add the two methods)
- Test: `backend/internal/events/repository_test.go` (integration, `//go:build integration`)

**Interfaces:**
- Produces on the `Repository` interface:
  - `SetCapacityTx(eventID uuid.UUID, newCapacity *int) (promoted int, err error)` — atomically: `FOR UPDATE` lock the event; count occupied = `going`+`accepted`; if `newCapacity != nil && *newCapacity < occupied` return `ErrCapacityBelowOccupied` (carrying `occupied`); write the `capacity` column; then FIFO-promote `waitlist`→`going` while occupied < capacity (or unlimited); return promoted count.
  - `WriteEditAudit(ctx context.Context, eventID, actorID uuid.UUID) error` — insert one `audit_log` row `action='event.edit'`.
- Produces domain error `ErrCapacityBelowOccupied` in package `events` (a sentinel wrapping the occupied count via `%d`). Consumed by Task 3 + the HTTP handler (Task 4).

- [ ] **Step 1: Failing integration test**

Add to `backend/internal/events/repository_test.go` (guarded `//go:build integration`, mirroring the moderation repo test setup):

```go
func TestSetCapacityTx_PromotesWaitlist(t *testing.T) {
	r, db := newTestRepo(t) // existing helper; else create event + rsvps directly
	ev := seedPublishedEvent(t, db, /*capacity*/ 1)
	seedRsvp(t, db, ev.ID, "going")     // occupies the 1 seat
	wl := seedRsvp(t, db, ev.ID, "waitlist")

	promoted, err := r.SetCapacityTx(ev.ID, intp(2))
	if err != nil { t.Fatalf("SetCapacityTx: %v", err) }
	if promoted != 1 { t.Fatalf("want 1 promoted, got %d", promoted) }
	assertRsvpStatus(t, db, wl, "going")
}

func TestSetCapacityTx_BelowOccupied(t *testing.T) {
	r, db := newTestRepo(t)
	ev := seedPublishedEvent(t, db, 2)
	seedRsvp(t, db, ev.ID, "going")
	seedRsvp(t, db, ev.ID, "going")
	_, err := r.SetCapacityTx(ev.ID, intp(1))
	if !errors.Is(err, events.ErrCapacityBelowOccupied) {
		t.Fatalf("want ErrCapacityBelowOccupied, got %v", err)
	}
}
```

(If the seed helpers don't exist, add small ones next to the test that `INSERT INTO events` / `event_rsvps` directly. `intp := func(n int) *int { return &n }`.)

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test -tags integration ./internal/events/ -run TestSetCapacityTx -v`
Expected: FAIL — `SetCapacityTx` / `ErrCapacityBelowOccupied` undefined.

- [ ] **Step 3: Implement the sentinel, repo methods**

In `events/service.go` error block, add:

```go
	// ErrCapacityBelowOccupied indicates a capacity change below the number of
	// occupied seats (going+accepted). The message carries the occupied count.
	ErrCapacityBelowOccupied = errors.New("capacity below occupied seats")
```

Add to the `Repository` interface (in `events/repository.go`, alongside the other methods):

```go
	SetCapacityTx(eventID uuid.UUID, newCapacity *int) (int, error)
	WriteEditAudit(ctx context.Context, eventID, actorID uuid.UUID) error
```

Implement in `events/repository.go`:

```go
func (r *pgRepository) SetCapacityTx(eventID uuid.UUID, newCapacity *int) (int, error) {
	var promoted int
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		// Lock the event row to serialize with concurrent sign-ups.
		if _, err := tx.Exec(`SELECT id FROM events WHERE id = ? FOR UPDATE`, eventID); err != nil {
			return fmt.Errorf("lock event %s: %w", eventID, err)
		}
		occupied, err := tx.Model((*models.Rsvp)(nil)).
			Where("event_id = ? AND status IN ('going','accepted')", eventID).Count()
		if err != nil {
			return fmt.Errorf("count occupied: %w", err)
		}
		if newCapacity != nil && *newCapacity < occupied {
			return fmt.Errorf("%w: %d occupied", ErrCapacityBelowOccupied, occupied)
		}
		if _, err := tx.Exec(`UPDATE events SET capacity = ?, updated_at = now() WHERE id = ?`,
			newCapacity, eventID); err != nil {
			return fmt.Errorf("set capacity: %w", err)
		}
		// Promote FIFO from waitlist while there is room (unlimited => fill all).
		for newCapacity == nil || occupied < *newCapacity {
			next := new(models.Rsvp)
			err := tx.Model(next).
				Where("event_id = ? AND status = 'waitlist'", eventID).
				Order("created_at ASC").Limit(1).Select()
			if errors.Is(err, pg.ErrNoRows) {
				break
			}
			if err != nil {
				return fmt.Errorf("find waitlist head: %w", err)
			}
			next.Status = models.RsvpGoing
			if _, err := tx.Model(next).Column("status", "updated_at").WherePK().Update(); err != nil {
				return fmt.Errorf("promote waitlist: %w", err)
			}
			occupied++
			promoted++
		}
		return nil
	})
	return promoted, err
}

func (r *pgRepository) WriteEditAudit(ctx context.Context, eventID, actorID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
		 VALUES (?, 'event.edit', 'event', ?, '{}'::jsonb)`,
		actorID, eventID); err != nil {
		return fmt.Errorf("insert edit audit: %w", err)
	}
	return nil
}
```

(Add `"github.com/Pashteto/lia/internal/models"` and `"github.com/go-pg/pg/v10"` imports if not present in `repository.go`.)

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test -tags integration ./internal/events/ -run TestSetCapacityTx -v`
Expected: PASS (both). If no integration DB is available locally, run `go build ./...` to confirm it compiles and defer the integration run to CI/box.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/events/repository.go backend/internal/events/service.go backend/internal/events/repository_test.go
git commit -m "feat(r2): SetCapacityTx (occupancy guard + waitlist promotion) + edit audit"
```

---

### Task 3: Service — allow published edits, lock signup_mode, reconcile capacity, audit

**Files:**
- Modify: `backend/internal/events/service.go:208-325` (`Update`)
- Test: `backend/internal/events/service_test.go`

**Interfaces:**
- Consumes: `UpdateParams.Capacity` (Task 1), `SetCapacityTx` / `WriteEditAudit` / `ErrCapacityBelowOccupied` (Task 2).
- Produces: `Update` returns the reloaded event for `draft` and `published`; 409 `ErrNotEditable` for other statuses; 422 `ErrInvalidInput` on a `signup_mode` change against a non-draft; capacity changes reconciled; audit written on published edits.

- [ ] **Step 1: Failing service tests (fakes)**

Add to `backend/internal/events/service_test.go` (extend the existing fake repository with the two new methods returning recorded calls):

```go
func TestUpdate_AllowsPublished(t *testing.T) {
	svc, repo := newServiceWithFakes(t)
	repo.event = publishedEvent(ownerID) // status=published, owner=ownerID
	title := "Новое название"
	_, err := svc.Update(ctx, repo.event.ID, ownerID, events.UpdateParams{Title: &title})
	if err != nil { t.Fatalf("published edit should succeed, got %v", err) }
	if !repo.editAuditWritten { t.Fatal("expected audit on published edit") }
}

func TestUpdate_RejectsSignupModeChangeOnPublished(t *testing.T) {
	svc, repo := newServiceWithFakes(t)
	repo.event = publishedEvent(ownerID)
	mode := "application"
	_, err := svc.Update(ctx, repo.event.ID, ownerID, events.UpdateParams{SignupMode: &mode})
	if !errors.Is(err, events.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_RejectsEditOfCancelled(t *testing.T) {
	svc, repo := newServiceWithFakes(t)
	repo.event = eventWithStatus(ownerID, models.EventCancelled)
	title := "x"
	_, err := svc.Update(ctx, repo.event.ID, ownerID, events.UpdateParams{Title: &title})
	if !errors.Is(err, events.ErrNotEditable) {
		t.Fatalf("want ErrNotEditable, got %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/events/ -run TestUpdate_ -v`
Expected: FAIL (published edit currently returns `ErrNotEditable`; audit flag unset; signup_mode change allowed).

- [ ] **Step 3: Implement the new gates + reconciliation**

Replace the editable gate (`service.go:226-229`):

```go
	// Draft and published events are editable. Moderation/terminal statuses are not.
	if event.Status != models.EventDraft && event.Status != models.EventPublished {
		return nil, fmt.Errorf("%w: event %s is %s", ErrNotEditable, id, event.Status)
	}
	wasPublished := event.Status == models.EventPublished

	// Signup mode is locked once published (would strip meaning from existing RSVPs).
	if wasPublished && p.SignupMode != nil && *p.SignupMode != event.SignupMode {
		return nil, fmt.Errorf("%w: режим записи нельзя изменить после публикации", ErrInvalidInput)
	}
```

Capacity is owned by `SetCapacityTx`, so do NOT let the generic field-copy touch it. Leave the existing `p.*` field assignments as-is (none assign capacity today). Replace the tail (from `if err := s.repo.Update(event); …` at `service.go:316`) with:

```go
	if err := s.repo.Update(event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	// Capacity change: guarded, atomic, promotes the waitlist.
	if p.Capacity != nil {
		if _, err := s.repo.SetCapacityTx(id, p.Capacity); err != nil {
			if errors.Is(err, ErrCapacityBelowOccupied) {
				return nil, err // handler maps to 409
			}
			return nil, fmt.Errorf("reconcile capacity: %w", err)
		}
	}

	// Audit published edits (draft edits are not audited, matching create/publish).
	if wasPublished {
		if err := s.repo.WriteEditAudit(ctx, id, ownerID); err != nil {
			return nil, fmt.Errorf("write edit audit: %w", err)
		}
	}

	reloaded, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("reload event: %w", err)
	}
	return reloaded, nil
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test ./internal/events/ -run TestUpdate_ -v`
Expected: PASS.

- [ ] **Step 5: Full gate + commit**

```bash
cd backend && go build ./... && go vet ./... && golangci-lint run
git add backend/internal/events/service.go backend/internal/events/service_test.go
git commit -m "feat(r2): edit published events, lock signup_mode, reconcile capacity, audit"
```

---

### Task 4: HTTP handler — map `ErrCapacityBelowOccupied` to 409

**Files:**
- Modify: `backend/internal/http/handlers/events_update.go:47-63` (error switch)
- Test: `backend/internal/http/handlers/events_test.go`

**Interfaces:**
- Consumes: `events.ErrCapacityBelowOccupied` (Task 2).
- Produces: `PATCH /events/{id}` returns **409** for capacity-below-occupied and for `ErrNotEditable`; **400** for `ErrInvalidInput` (incl. locked signup_mode).

- [ ] **Step 1: Failing handler test**

Add to `events_test.go`, mirroring the existing `ErrNotEditable` case at line 186:

```go
func TestUpdateEvent_CapacityBelowOccupied409(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: 3 occupied", eventsdomain.ErrCapacityBelowOccupied)}
	// …build handler + request exactly like the existing ErrNotEditable test…
	resp := h.UpdateEvent(params, principal)
	assertStatus(t, resp, http.StatusConflict)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/http/handlers/ -run TestUpdateEvent_CapacityBelowOccupied409 -v`
Expected: FAIL — falls through to 503 (default branch).

- [ ] **Step 3: Add the case**

In the error switch, add before `default:`:

```go
		case errors.Is(err, eventsdomain.ErrCapacityBelowOccupied):
			return eventsops.NewUpdateEventConflict().
				WithPayload(DefaultError(http.StatusConflict, err, nil))
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test ./internal/http/handlers/ -run TestUpdateEvent -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/handlers/events_update.go backend/internal/http/handlers/events_test.go
git commit -m "feat(r2): map capacity-below-occupied to 409"
```

---

### Task 5: Frontend — `patchEvent` API + edit page + entry points

**Files:**
- Modify: `frontend/lib/api.ts` (add `patchEvent`)
- Create: `frontend/app/events/[id]/edit/page.tsx`
- Modify: `frontend/components/CreateEventForm.tsx` (accept an optional `mode`/`initial`/`eventId` prop for edit reuse)
- Modify: `frontend/app/events/mine/page.tsx` (add «Редактировать» link per row)

**Interfaces:**
- Consumes: backend `PATCH /events/{id}` (Tasks 1-4).
- Produces: `patchEvent(id: string, patch: Partial<CreateEventInput>): Promise<LiaEvent>`; an edit route that prefills and PATCHes.

- [ ] **Step 1: Add `patchEvent`**

In `frontend/lib/api.ts` (mirror the `PublishEventButton` PATCH at `components/PublishEventButton.tsx:29-33`):

```ts
export async function patchEvent(id: string, patch: Partial<CreateEventInput>): Promise<LiaEvent> {
  const token = getToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${API_V1}/events/${id}`, {
    method: "PATCH",
    headers,
    body: JSON.stringify(patch),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`patch event failed: ${res.status} ${detail}`);
  }
  return apiEventToLia((await res.json()) as ApiEvent);
}
```

- [ ] **Step 2: Make `CreateEventForm` reusable for edit**

Add optional props: `mode?: "create" | "edit"` (default `"create"`), `eventId?: string`, `initial?: Partial<FormValues> & { coverFileId?: string; coverPreviewUrl?: string }`. When `mode==="edit"`:
- Seed `defaultValues` from `initial`.
- On submit call `patchEvent(eventId, input)` instead of `createEvent`, and **omit `signup_mode`** from the patch when the event is published (mode is locked); keep `capacity` editable.
- Render the signup-mode Segmented as **read-only/disabled** with a hint «Режим записи зафиксирован после публикации» when editing a published event.
- Header title «Редактирование события».
- When `starts_at` or `venue` changes on a published event, show the non-blocking notice: «Участники уже записаны — предупредите их об изменении самостоятельно».

- [ ] **Step 3: Create the edit page**

`frontend/app/events/[id]/edit/page.tsx` — a client page that fetches the event (owner token), maps it to form `initial` values (datetime-local strings via the same helper the detail page uses), and renders `<CreateEventForm mode="edit" eventId={id} initial={…} />`. Guard: if the fetch 403/404s (not owner), show «Событие не найдено или недоступно».

- [ ] **Step 4: Add «Редактировать» entry points**

On `frontend/app/events/mine/page.tsx`, add a `Link href={\`/events/${e.id}/edit\`}` labelled «Редактировать» on each row (both draft and published — draft edit already worked server-side).

- [ ] **Step 5: Build + lint + manual**

Run: `cd frontend && pnpm lint && pnpm build`
Manual (local backend): publish an event → edit the title + move the date → still published, changes shown; raise capacity with a waitlisted guest → the guest becomes `going`; try to lower capacity below occupied → inline 409 message.

- [ ] **Step 6: Commit**

```bash
git add frontend/lib/api.ts frontend/app/events/[id]/edit/page.tsx frontend/components/CreateEventForm.tsx frontend/app/events/mine/page.tsx
git commit -m "feat(r2): edit page + patchEvent + edit entry points"
```

---

## Self-Review

- **Spec coverage:** edit published incl. all content fields (Tasks 3/5) ✓; capacity editable with reconciliation (Tasks 1-3) ✓; capacity increase promotes FIFO (Task 2) ✓; capacity below occupied → 409 (Tasks 2-4) ✓; stays published, no re-moderation (Task 3, no status change) ✓; signup_mode locked on published → 422 (Task 3) ✓; audit on published edit (Tasks 2/3) ✓; date/venue-change warning (Task 5) ✓; edit UI reuses create form (Task 5) ✓; no migration ✓.
- **Placeholder scan:** integration seed helpers in Task 2 are described with a fallback ("add small ones … INSERT directly"); everything else is concrete. Handler test in Task 4 references "exactly like the existing ErrNotEditable test" — acceptable since that test exists verbatim at `events_test.go:186` and is being mirrored, not invented.
- **Type consistency:** `SetCapacityTx(eventID, newCapacity *int) (int, error)` and `WriteEditAudit(ctx, eventID, actorID)` are declared identically in the interface (Task 2) and called identically in the service (Task 3). `ErrCapacityBelowOccupied` defined once (Task 2), consumed in Tasks 3+4. `patchEvent` signature matches its usage in Task 5.

## Deploy

No migration (schema 018). Backend + frontend. Run `make generate-api` before the backend build. Standard build-on-Mac→`save|ssh|load`; take a pre-deploy DB dump per runbook convention. Land **after R1** (shared form fields).
