# Organizer Page Events + Clickable Host — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the «Ведущий» host row on event detail link to the organizer page, and list a verified organizer's upcoming + past published events on `/organizers/[id]`, backed by a new additive `organizer_id` filter on the public `GET /events`.

**Architecture:** One additive backend query param (`organizer_id` on `GET /events`) resolved from a public profile id to the verified organizer's owner-user-id (reusing `organizers.Service.GetByID`), threaded into the existing `events.ListFilter.OrganizerIDs`. Frontend makes one call and splits events into upcoming/past client-side. No DB migration.

**Tech Stack:** Backend — Go monolith, go-pg, go-swagger (generated models/spec). Frontend — Next.js 16 App Router, React 19, TypeScript, Tailwind v4.

## Global Constraints

- **Published-only**: the public `GET /events` list stays `security: []` and returns only published events. No draft/private leak.
- **Profile-id addressing**: `organizer_id` is a public **profile id** (`organizers.id`), resolved server-side to the owner user id of a **verified** organizer (`verification_status == "verified"`). An unknown/unverified/malformed id returns an **empty list** (HTTP 200 `[]`), never an error and never a leak.
- **Name/email**: the resolution selects only what it needs; no email or private user fields exposed (none are added to any response).
- **No DB migration.** The only backend change is one additive query param + handler/service threading.
- **Swagger regen is `make generate-api` only** — never `make generate-all`. Generated artifacts (`internal/http/models/`, `internal/http/server/embedded_spec.go`) stay gitignored and must **not** be committed; only `backend/api/swagger.yaml` carries the schema change.
- **No nested `<Link>`**: on event detail, once the host row is a `Link`, the `VerifiedBadge` must not also be a `Link` (render its plain `<span>` form).
- **Frontend has no unit-test runner** — frontend tasks verify with `pnpm exec tsc --noEmit` + `pnpm lint`, plus a manual browser check. Do not add a test runner.

---

### Task 1: Backend — `organizer_id` filter on `GET /events`

**Files:**
- Modify: `backend/api/swagger.yaml` (the `GET /events` params, ~lines 196–213) then run `make generate-api`
- Modify: `backend/internal/events/service.go` (interface line ~53; `List` impl ~lines 327–340)
- Modify: `backend/internal/http/handlers/events.go` (`ListEvents` struct, `NewListEvents`, `Handle` ~lines 18–58)
- Modify: `backend/internal/http/module.go` (line ~250, `NewListEvents(m.events)` → pass organizers)
- Test: `backend/internal/http/handlers/events_test.go` (update `mockEventsService.List`; add a fake organizers service + two handler tests)

**Interfaces:**
- Consumes: `organizers.Service.GetByID(ctx, id uuid.UUID) (*organizers.Organizer, error)` — returns an `*Organizer` with `.OwnerUserID uuid.UUID` and `.VerificationStatus string`; returns `organizers.ErrNotFound` when absent. `events.ListFilter.OrganizerIDs []uuid.UUID` (already wired in the repo query).
- Produces: `events.Service.List(ctx, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error)` — new trailing `*uuid.UUID` param (nil = no organizer filter). API `GET /events?organizer_id={uuid}` returns that verified organizer's published events, `[]` if unresolved.

- [ ] **Step 1: Add the swagger query param**

In `backend/api/swagger.yaml`, under `/events` → `get` → `parameters` (after the `to` param, before `responses`), add:

```yaml
        - name: organizer_id
          in: query
          description: Only return events created by this verified organizer (organizers.id profile id). Unknown/unverified ids return an empty list.
          required: false
          type: string
          format: uuid
```

- [ ] **Step 2: Regenerate the swagger models (swagger-only)**

Run: `cd backend && make generate-api`
Expected: regenerates `internal/http/server/operations/events/list_events_parameters.go` with an `OrganizerID *strfmt.UUID` field. Confirm: `grep -n "OrganizerID" internal/http/server/operations/events/list_events_parameters.go` shows the bound param.

- [ ] **Step 3: Extend `events.Service.List` (write the failing test first)**

In `backend/internal/events/service.go`, this is interface-level; first add a service-level test. Append to `backend/internal/events/service_test.go` (match its existing fake-repo style — inspect the file for the repo mock's name and the `ListFilter` capture):

```go
func TestList_PassesOrganizerOwnerIDIntoFilter(t *testing.T) {
	repo := &fakeRepo{} // use whatever the existing repo mock is named in this file
	svc := NewService(repo /* + other deps the constructor needs */)
	owner := uuid.Must(uuid.NewV4())
	_, _ = svc.List(context.Background(), "published", nil, nil, &owner)
	if len(repo.lastFilter.OrganizerIDs) != 1 || repo.lastFilter.OrganizerIDs[0] != owner {
		t.Fatalf("expected OrganizerIDs=[%s], got %v", owner, repo.lastFilter.OrganizerIDs)
	}
}
```

If `service_test.go` has no repo mock that captures the `ListFilter` (e.g. tests hit a real DB and are skipped without `TEST_DATABASE_URL`), then SKIP this service-level test and rely on the handler test in Step 8 instead — note that choice in your report. Do not invent a DB-backed test.

- [ ] **Step 4: Run the test to verify it fails**

Run: `cd backend && go test ./internal/events/ -run TestList_PassesOrganizerOwnerIDIntoFilter`
Expected: compile failure (`List` takes 4 args, not 5) — confirming the signature must change. (If you skipped Step 3, skip this step.)

- [ ] **Step 5: Change the `List` signature + impl**

In `backend/internal/events/service.go`, interface (line ~53):

```go
	List(ctx context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error)
```

Impl (replace the body at ~lines 327–340):

```go
func (s *service) List(_ context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error) {
	if status != "" {
		if _, err := models.EventStatusFromString(status); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
	}

	filter := ListFilter{Status: status, From: from, To: to}
	if organizerOwnerID != nil {
		filter.OrganizerIDs = []uuid.UUID{*organizerOwnerID}
	}

	list, err := s.repo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return list, nil
}
```

(`uuid` from `github.com/gofrs/uuid` is already imported in this file.)

- [ ] **Step 6: Update the events-domain `List` mock if present**

If `backend/internal/events/service_test.go` has a fake that implements the `Service` interface (not just the repo), update its `List` method signature to match. Run `cd backend && go build ./internal/events/...` to surface any signature mismatch and fix it.

- [ ] **Step 7: Wire organizers into the `ListEvents` handler**

In `backend/internal/http/handlers/events.go`, add the organizers import and extend the handler:

```go
	organizersdomain "github.com/Pashteto/lia/internal/organizers"
```

```go
// ListEvents handler returns events, optionally filtered by status / organizer.
type ListEvents struct {
	events     eventsdomain.Service
	organizers organizersdomain.Service
}

// NewListEvents creates a ListEvents handler.
func NewListEvents(svc eventsdomain.Service, orgs organizersdomain.Service) *ListEvents {
	return &ListEvents{events: svc, organizers: orgs}
}

// Handle GET /events.
func (h *ListEvents) Handle(params eventsops.ListEventsParams) middleware.Responder {
	var from, to *time.Time
	if params.From != nil {
		t := time.Time(*params.From)
		from = &t
	}
	if params.To != nil {
		t := time.Time(*params.To)
		to = &t
	}

	// organizer_id (a public organizers.id profile id) restricts to that
	// verified organizer's events. Unknown / unverified / malformed id, or no
	// organizers service (no-DB mode) → empty list, no error, no leak.
	var organizerOwner *uuid.UUID
	if params.OrganizerID != nil {
		if h.organizers == nil {
			return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
		}
		profileID, perr := uuid.FromString(params.OrganizerID.String())
		if perr != nil {
			return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
		}
		org, oerr := h.organizers.GetByID(params.HTTPRequest.Context(), profileID)
		if oerr != nil || org == nil || org.VerificationStatus != "verified" {
			return eventsops.NewListEventsOK().WithPayload([]*apimodels.Event{})
		}
		organizerOwner = &org.OwnerUserID
	}

	list, err := h.events.List(params.HTTPRequest.Context(), "published", from, to, organizerOwner)
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

- [ ] **Step 8: Update the handler test mock + add handler tests**

In `backend/internal/http/handlers/events_test.go`:

1. Update `mockEventsService.List` to the new signature and capture the organizer arg. Find the existing `List` method on `mockEventsService` and replace it; add a `listOrganizerArg *uuid.UUID` field to the struct:

```go
func (m *mockEventsService) List(_ context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*domainmodels.Event, error) {
	m.listStatusArg = status
	m.listFromArg = from
	m.listToArg = to
	m.listOrganizerArg = organizerOwnerID
	return nil, nil
}
```

2. Add a fake organizers service implementing the full `organizersdomain.Service` interface (only `GetByID` needs real behavior; the rest return zero values). The complete interface has these 11 methods — stub all of them:

```go
type fakeOrganizers struct {
	org *organizersdomain.Organizer
	err error
}

func (f *fakeOrganizers) GetByID(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, error) {
	return f.org, f.err
}
func (f *fakeOrganizers) GetByOwner(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, error) { return nil, nil }
func (f *fakeOrganizers) Upsert(_ context.Context, _ uuid.UUID, _ organizersdomain.Input) (*organizersdomain.Organizer, error) { return nil, nil }
func (f *fakeOrganizers) Submit(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }
func (f *fakeOrganizers) Verify(_ context.Context, _, _ uuid.UUID) error { return nil }
func (f *fakeOrganizers) Reject(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (f *fakeOrganizers) Revoke(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (f *fakeOrganizers) SetAutoVerify(_ context.Context, _, _ uuid.UUID, _ bool) error { return nil }
func (f *fakeOrganizers) List(_ context.Context, _ organizersdomain.ListFilter) ([]organizersdomain.Organizer, error) { return nil, nil }
func (f *fakeOrganizers) GetWithHistory(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, []organizersdomain.HistoryEntry, error) { return nil, nil, nil }
func (f *fakeOrganizers) Overview(_ context.Context) (organizersdomain.Counts, error) { return organizersdomain.Counts{}, nil }
```

3. Add two tests:

```go
func TestListEvents_OrganizerResolvesToVerifiedOwner(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	profile := uuid.Must(uuid.NewV4())
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{org: &organizersdomain.Organizer{OwnerUserID: owner, VerificationStatus: "verified"}}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(profile.String())
	resp := h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	_ = resp
	if evSvc.listOrganizerArg == nil || *evSvc.listOrganizerArg != owner {
		t.Fatalf("expected owner %s passed to List, got %v", owner, evSvc.listOrganizerArg)
	}
}

func TestListEvents_UnverifiedOrganizerReturnsEmpty(t *testing.T) {
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{org: &organizersdomain.Organizer{VerificationStatus: "pending"}}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	if evSvc.listOrganizerArg != nil {
		t.Fatalf("expected List not to receive an organizer filter for unverified org")
	}
}
```

Add imports as needed: `net/http/httptest`, `organizersdomain "github.com/Pashteto/lia/internal/organizers"`.

- [ ] **Step 9: Update module wiring**

In `backend/internal/http/module.go` (~line 250): `api.EventsListEventsHandler = handlers.NewListEvents(m.events, m.organizers)`.

- [ ] **Step 10: Build + run tests**

Run: `cd backend && go build ./... && go test ./internal/http/handlers/... ./internal/events/...`
Expected: build clean; the two handler tests pass (and the service test if added). Fix any interface-mismatch compile errors from the signature change.

- [ ] **Step 11: Commit (source only — NO generated artifacts)**

```bash
git add backend/api/swagger.yaml backend/internal/events/service.go backend/internal/events/service_test.go backend/internal/http/handlers/events.go backend/internal/http/handlers/events_test.go backend/internal/http/module.go
git commit -m "feat(events): organizer_id filter on GET /events (verified-only, published)"
```

(Do NOT `git add` `internal/http/models/` or `embedded_spec.go` — gitignored generated code.)

---

### Task 2: Frontend — organizer page upcoming/past events

**Files:**
- Modify: `frontend/lib/api.ts` (add `fetchEventsByOrganizer`)
- Modify: `frontend/app/organizers/[id]/page.tsx` (fetch + render lists; remove TODO)

**Interfaces:**
- Consumes: API `GET /events?organizer_id={id}` (Task 1); `apiEventToLia` + `ApiEvent` (already in `api.ts`); `LiaEvent` (`.startsAt` field — confirm the property name in `lib/types.ts`); `EventCard` (`frontend/components/ui/EventCard.tsx`, prop `event: LiaEvent`).
- Produces: `fetchEventsByOrganizer(organizerId: string): Promise<LiaEvent[]>`.

- [ ] **Step 1: Add the API helper**

In `frontend/lib/api.ts`, near `fetchPublishedEvents`:

```ts
/** Fetches a verified organizer's published events via GET /events?organizer_id=. */
export async function fetchEventsByOrganizer(organizerId: string): Promise<LiaEvent[]> {
  const res = await fetch(`${API_V1}/events?organizer_id=${encodeURIComponent(organizerId)}`);
  if (!res.ok) {
    throw new Error(`fetch organizer events failed: ${res.status}`);
  }
  const data = (await res.json()) as ApiEvent[];
  return data.map(apiEventToLia);
}
```

- [ ] **Step 2: (Confirmed) event start-time field is `startsAt`**

`LiaEvent.startsAt: string` (ISO 8601) is the field — confirmed in `frontend/lib/types.ts:67`. Use `e.startsAt` in Step 3's split logic. (The `starts_at` form at types.ts:130 is the raw `ApiEvent` shape, already mapped to `startsAt` by `apiEventToLia` — do not use it here.)

- [ ] **Step 3: Fetch + render upcoming/past on the organizer page**

In `frontend/app/organizers/[id]/page.tsx`:

1. Add imports: `import { fetchEventsByOrganizer } from "@/lib/api";` (extend the existing import from `@/lib/api`), `import { EventCard } from "@/components/ui/EventCard";`, `import type { LiaEvent } from "@/lib/types";`.

2. Add events state and fetch in a `useEffect` keyed on `params.id` (alongside the existing org fetch):

```tsx
  const [events, setEvents] = useState<LiaEvent[]>([]);
  // ... inside the component, after the existing org useEffect:
  useEffect(() => {
    fetchEventsByOrganizer(params.id)
      .then(setEvents)
      .catch(() => setEvents([]));
  }, [params.id]);
```

3. Compute the split (place after `if (!org) return null;`, before the return):

```tsx
  const now = Date.now();
  const upcoming = events
    .filter((e) => new Date(e.startsAt).getTime() >= now)
    .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime());
  const past = events
    .filter((e) => new Date(e.startsAt).getTime() < now)
    .sort((a, b) => new Date(b.startsAt).getTime() - new Date(a.startsAt).getTime())
    .slice(0, 10);
```

(Use the real start-time field confirmed in Step 2 in place of `startsAt` if it differs.)

4. Render two sections at the end of the `<main>` (replace the stale TODO comment block):

```tsx
      <section className="space-y-3 pt-4">
        <h2 className="text-xl font-semibold tracking-[-0.022em]">Предстоящие мероприятия</h2>
        {upcoming.length === 0 ? (
          <p className="text-label-secondary">Пока нет предстоящих мероприятий.</p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {upcoming.map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        )}
      </section>

      {past.length > 0 && (
        <section className="space-y-3 pt-4">
          <h2 className="text-xl font-semibold tracking-[-0.022em]">Прошедшие мероприятия</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {past.map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        </section>
      )}
```

Remove the existing `{/* Published events for this organizer ... */}` TODO comment.

- [ ] **Step 4: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: clean. (If `EventCard`'s grid wrapper or tokens differ from a sibling like `DiscoveryFeed`, match the sibling.)

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/api.ts frontend/app/organizers/[id]/page.tsx
git commit -m "feat(frontend): list organizer upcoming/past events on /organizers/[id]"
```

---

### Task 3: Frontend — clickable host row on event detail

**Files:**
- Modify: `frontend/components/EventDetailView.tsx` (the «Ведущий» `Section`, ~lines 81–100)

**Interfaces:**
- Consumes: `event.organizer.profile_id` (string | undefined), `event.organizer.name`, `event.organizer.affiliation`, `event.organizer.verified`; `Link` from `next/link` (already imported at line 12); `VerifiedBadge`.

- [ ] **Step 1: Confirm the badge can render without a link**

`VerifiedBadge` (`frontend/components/VerifiedBadge.tsx`) renders a plain `<span>` when called **without** a `profileId`, and wraps it in a `Link` only when `profileId` is passed. So calling `<VerifiedBadge />` (no prop) yields the plain badge — exactly what's needed inside the new row link. Confirm by reading the component.

- [ ] **Step 2: Make the host row a link when verified**

In `frontend/components/EventDetailView.tsx`, replace the «Ведущий» `Section` body (the `<div className="flex items-center gap-3">...</div>`) with a version that wraps the row in a `Link` when `event.organizer.profile_id` is present. Render the badge WITHOUT a `profileId` (no nested link):

```tsx
        {event.organizer && (
          <Section title="Ведущий">
            {event.organizer.profile_id ? (
              <Link
                href={`/organizers/${event.organizer.profile_id}`}
                className="flex items-center gap-3 transition hover:opacity-70"
              >
                <div className="size-11 shrink-0 rounded-full bg-fill" aria-hidden />
                <div>
                  <p className="flex items-center gap-1.5 text-[17px] font-medium">
                    {event.organizer.name || "Организатор"}
                    {event.organizer.verified && <VerifiedBadge />}
                  </p>
                  {event.organizer.affiliation && (
                    <p className="text-[13px] text-label-secondary">
                      {event.organizer.affiliation}
                    </p>
                  )}
                </div>
              </Link>
            ) : (
              <div className="flex items-center gap-3">
                <div className="size-11 shrink-0 rounded-full bg-fill" aria-hidden />
                <div>
                  <p className="flex items-center gap-1.5 text-[17px] font-medium">
                    {event.organizer.name || "Организатор"}
                    {event.organizer.verified && <VerifiedBadge />}
                  </p>
                  {event.organizer.affiliation && (
                    <p className="text-[13px] text-label-secondary">
                      {event.organizer.affiliation}
                    </p>
                  )}
                </div>
              </div>
            )}
          </Section>
        )}
```

- [ ] **Step 3: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: clean.

- [ ] **Step 4: Manual browser check**

Run the frontend (`cd frontend && pnpm dev`); on a verified event's detail page, confirm clicking the organizer name/avatar navigates to `/organizers/{profile_id}`; on an unverified event, the row is plain text (no link). The «✓ Проверен» badge still shows.

- [ ] **Step 5: Commit**

```bash
git add frontend/components/EventDetailView.tsx
git commit -m "feat(frontend): clickable organizer host row on event detail"
```

---

## Final verification

- [ ] Backend: `cd backend && go build ./... && go test ./internal/http/handlers/... ./internal/events/...` → PASS.
- [ ] Frontend: `cd frontend && pnpm exec tsc --noEmit && pnpm lint` → clean.
- [ ] Confirm no generated artifacts or migration were committed (`git status` shows no `internal/http/models/`, no `embedded_spec.go`, no `backend/migrations/` additions).
- [ ] End-to-end (running stack): `GET /events?organizer_id={verified profile id}` returns that organizer's published events; an unknown id returns `[]`; the organizer page shows upcoming + past sections; the event-detail host row links to the organizer page.
