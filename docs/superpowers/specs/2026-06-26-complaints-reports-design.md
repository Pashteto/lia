# Complaints / Reports — Design (admin-suite sub-project #3)

_Date: 2026-06-26. Status: **spec, awaiting review**. Umbrella tracker: [`../plans/2026-06-26-admin-suite-roadmap.md`](../plans/2026-06-26-admin-suite-roadmap.md) (#3). Depends on sub-projects 0 (RBAC + admin shell) and 1 (event moderation), both DONE + LIVE._

## Goal

Let an authenticated user report ("Пожаловаться") a problem with an **event**, and give
staff a **grouped inbox** at `/admin/complaints` to triage and resolve those reports —
reusing the existing `moderation.Takedown` to act on confirmed violations, and auditing
every resolution.

## Scope

**In:** event complaints only (the one public surface that needs it today). The data model
uses a generic `target_type`/`target_id` so organizers/users can be added later without a
structural migration.

**Out (deliberately deferred):** reporting organizers or users (no public entry point yet);
reporter notifications of outcome; abuse-scoring/auto-hide on report-count thresholds;
complaint analytics. These can layer on later without reworking this slice.

## Decisions (from brainstorming)

1. **Target scope:** events only this slice; generic `target_type`/`target_id` column kept open.
2. **Reason input:** predefined **category** (allowlist) + **optional free-text note**.
3. **Dedup:** **one OPEN complaint per (reporter, target)** via a partial unique index. A repeat
   submit while one is open is an idempotent no-op success. After the complaint is resolved/
   dismissed, the same user may file a fresh one.
4. **Inbox model:** **grouped by event**; resolving an event **cascades** to all its open
   complaints in one operation. Takedown reused once per event, not per complaint.
5. **Auth:** submit is behind auth (reporter = the authenticated user). Inbox/resolve are
   admin-gated (`role == "admin"`, reusing the existing `requireStaff` gate).

## Data model — migration `000017_complaints`

```sql
CREATE TABLE complaints (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type      TEXT NOT NULL DEFAULT 'event'
                     CHECK (target_type IN ('event')),     -- generic; event-only for now
    target_id        uuid NOT NULL,
    reporter_user_id uuid NOT NULL,
    category         TEXT NOT NULL
                     CHECK (category IN ('spam','fraud','inappropriate','duplicate','other')),
    note             TEXT,                                  -- optional free text
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open','resolved','dismissed')),
    resolution       TEXT,                                  -- moderator note on close
    resolved_by      uuid,
    resolved_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- one OPEN complaint per reporter per target (repeat-submit dedup):
CREATE UNIQUE INDEX complaints_one_open_per_reporter
    ON complaints (target_type, target_id, reporter_user_id) WHERE status = 'open';

-- fast "events with open complaints" grouping:
CREATE INDEX complaints_open_target_idx
    ON complaints (target_type, target_id) WHERE status = 'open';
```

Notes:
- No FK to `events` — `target_id` is generic by design (mirrors `audit_log.target_id`).
  Existence is validated in the service on submit (404 if the event is missing).
- Widening to organizers/users later = a one-line CHECK migration on `target_type`; no
  structural change.
- go-pg gotcha: nullable `uuid` columns (`resolved_by`) cannot scan SQL NULL into a
  `uuid.UUID`. Model `resolved_by` as `*uuid.UUID` (or scan via the resolve path that always
  sets it), and avoid `RETURNING *`. Optional text (`note`, `resolution`) → `*string` or
  `coalesce(...,'')` on read, per the existing repo conventions.

## Domain — `internal/complaints` (mirrors `internal/moderation`)

`service.go` + `repository.go`, interface-first.

### Types

```go
type Category string // spam|fraud|inappropriate|duplicate|other (validated against allowlist)

type Complaint struct {
    ID, TargetID, ReporterUserID uuid.UUID
    TargetType, Category, Note, Status, Resolution string
    ResolvedBy *uuid.UUID
    ResolvedAt *time.Time
    CreatedAt  time.Time
}

// EventReportGroup is one row of the grouped inbox.
type EventReportGroup struct {
    TargetID    uuid.UUID         `json:"event_id"`
    EventTitle  string            `json:"event_title"`
    EventStatus string            `json:"event_status"`   // published|rejected|draft — drives which actions are valid
    ReportCount int               `json:"report_count"`   // open complaints for this event
    Categories  map[string]int    `json:"categories"`     // open-count breakdown by category
    LatestNote  string            `json:"latest_note"`     // newest non-empty note, for preview
    LatestAt    time.Time         `json:"latest_at"`
}
```

### Service interface

```go
type Service interface {
    Submit(ctx, reporterID uuid.UUID, targetType, targetID, category, note) error
    ListInbox(ctx) ([]EventReportGroup, error)
    TargetDetail(ctx, targetType string, targetID uuid.UUID) ([]Complaint, error)
    Resolve(ctx, targetType string, targetID, actorID uuid.UUID, action, resolution string) error
    OpenEventCount(ctx) (int, error)   // distinct events with ≥1 open complaint, for /admin/overview
}
```

- **Submit:** validate `category` against the allowlist (`ErrInvalidCategory` → 400); validate
  the event exists (`ErrTargetNotFound` → 404). Insert. On the partial-unique-index conflict
  (`ON CONFLICT … DO NOTHING`, or detect `pg` unique violation) → idempotent no-op success.
- **ListInbox:** grouped query over open complaints joined to `events` for title/status,
  `count(*)`, category breakdown, latest note. Sorted by `report_count DESC, LatestAt DESC`.
- **TargetDetail:** individual open + recently-resolved complaints for one event (for drill-in).
- **Resolve:** `action ∈ {takedown, dismiss}`.
  - `takedown` → `resolution` required (`ErrResolutionRequired` → 400); calls
    `moderation.Service.Takedown(event, actor, resolution)` **first** (reason = resolution).
    If that returns `ErrInvalidTransition` (event not `published`) → surface 409, do NOT close
    the complaints. On success, flip all open complaints for the target to `resolved`.
  - `dismiss` → flip all open complaints for the target to `dismissed` (no event change;
    valid even if the event is already `rejected`/down). `resolution` optional.
  - Either action: set `resolved_by`, `resolved_at`, `resolution` on the affected rows in one
    tx, and write **one** `audit_log` row (`action='complaint.resolve'`, `target_type='event'`,
    `target_id=event`, metadata `{action, resolution, resolved_count}`). Compliance requirement
    — every resolution is audited.

### Repository interface

```go
type Repository interface {
    Insert(ctx, c Complaint) error                 // ON CONFLICT DO NOTHING on the open-partial index
    InboxGroups(ctx) ([]EventReportGroup, error)
    TargetComplaints(ctx, targetType, targetID) ([]Complaint, error)
    ResolveOpenForTarget(ctx, targetType, targetID, actorID uuid.UUID, status, resolution string) (int, error) // cascading tx + audit; returns affected count
    OpenEventCount(ctx) (int, error)
    EventExists(ctx, id uuid.UUID) (bool, string, error)  // existence + current status (status reused by Resolve to pre-check)
}
```

### Coupling

`complaints.Service` is constructed with `moderation.Service` injected, so the takedown branch
of `Resolve` composes the existing transition instead of duplicating status SQL. The moderation
take-down (its own tx: status + `event_status_history` + `audit_log`) and the complaint-close
(`ResolveOpenForTarget` tx) run back-to-back; if takedown fails the complaints are left open.
(Both txs against the same event under staff action — acceptable for demo scale; a single
combined tx would require reaching into moderation's internals and is not worth the coupling.)

## HTTP surfaces

### Public submit — new `internal/http/complaints/handler.go`

Plain `net/http`, mirrors `internal/http/organizers`. Mounted in `internal/http/module.go`
ahead of the go-swagger mux, nil-degrading in no-DB mode.

- `POST /api/v1/events/{id}/complaints` — **authed** (Bearer required → 401 anon).
  Body: `{ "category": "spam|fraud|inappropriate|duplicate|other", "note": "optional" }`.
  Reporter = authenticated user. Responses: **201** created, **200** idempotent repeat,
  **400** invalid category, **404** unknown event, **401** anon.

Routing predicate in `module.go`: path matches `^/api/v1/events/{id}/complaints$` (a suffix
check `strings.HasPrefix(p, "/api/v1/events/") && strings.HasSuffix(p, "/complaints")`), routed
to the complaints handler before falling through to the swagger `events` routes.

### Admin inbox — added to existing `internal/http/admin/handler.go`

Staff-gated via the existing `h.staff(...)` wrapper.

- `GET  /api/v1/admin/complaints` → grouped inbox (`ListInbox`).
- `GET  /api/v1/admin/complaints/events/{id}` → drill-in (`TargetDetail`).
- `POST /api/v1/admin/complaints/events/{id}/resolve` →
  body `{ "action": "takedown"|"dismiss", "resolution": "..." }`.
  `takedown` requires non-empty `resolution` (→ 400 otherwise); **409** if event not `published`.
- `GET  /api/v1/admin/overview` gains `complaints_open` (distinct events with open complaints) —
  same pattern as `organizers_pending`.

`admin.Deps` gains `Complaints complaints.Service`; `module.go` wires it.

No go-swagger spec edits (these are all on the plain-net/http admin + complaints handlers,
mounted ahead of the swagger mux — same approach as moderation/organizers).

## Frontend

### Reporter — event detail (`frontend/app/events/[id]/`)

- Subtle «Пожаловаться» control near the event meta. Opens a small modal: category radios
  (Спам / Мошенничество / Неуместный контент / Дубликат / Другое) + optional note textarea →
  `POST /api/v1/events/{id}/complaints`.
- If anon, route through the existing login modal first, then continue.
- Success → toast «Жалоба отправлена». Idempotent repeat shows the same success.

### Moderator — new `frontend/app/admin/complaints/page.tsx`

- «Жалобы» entry in the admin shell/nav, gated like the other admin pages (4-state gate).
- Grouped table: event title + status badge, report count, category-breakdown chips, newest
  note preview. Row click → drill-in panel listing individual reports.
- Per-event actions:
  - **Снять с публикации** (takedown) — opens a reason modal (reuses the moderation reason-modal
    pattern); the reason is the resolution/takedown reason. Hidden/disabled when the event is not
    `published`.
  - **Отклонить** (dismiss) — optional resolution note; always available.
  - Both call `…/resolve` and refresh the list.
- Admin overview card gains an «Открытые жалобы» stat linking to `/admin/complaints`.

## Testing

**Backend** (mirror `internal/moderation` test layout):
- `repository_test.go`: insert; idempotent conflict on the open-partial index; `InboxGroups`
  shape + counts + category breakdown; `ResolveOpenForTarget` cascades + affected count + audit
  row; dismiss-when-event-already-down.
- `service_test.go`: category validation/error mapping; `Resolve` takedown composes
  `moderation.Takedown` (fake/stub moderation service); takedown requires resolution; 409
  propagation when not published.
- Integration test behind `//go:build integration` against a migrated test DB (submit → inbox →
  resolve loop). Note: per the roadmap, the repo's integration tests are not yet wired into a
  local/CI DB run — this test ships but joins that backlog.

**Frontend:** `tsc` / `eslint` / `next build` clean; manual verify of submit → inbox → takedown
and submit → inbox → dismiss loops.

## Audit / compliance (ISO 27001 / Vanta)

- Every resolution writes an `audit_log` row (`complaint.resolve`) with actor, action, resolution,
  and affected count — resolutions are a moderation/change-management control surface.
- Reporter identity (`reporter_user_id`) is stored but **never exposed** on any public surface or
  to the reported organizer; it is visible only to staff in the admin inbox. No reporter PII in
  event/organizer payloads.
- Submit is authenticated (no anonymous flooding) and deduped (one open complaint per reporter
  per target). Existing global rate-limit middleware still applies.

## Build order / dependencies

1. Migration `000017_complaints` (+ down).
2. `internal/complaints` domain (service + repository + tests).
3. Wire `complaints.Service` into `module.go`; inject `moderation.Service`.
4. Public submit handler `internal/http/complaints` + mount.
5. Admin inbox endpoints in `internal/http/admin/handler.go` + `complaints_open` on overview.
6. Frontend: event-detail report modal; `/admin/complaints` page + nav + overview stat.
7. Verify (backend build/vet/test + golangci v1; frontend lint/build; manual loop).

Reminder (from HANDOFF): backend rebuilds need `make generate-api` first (gitignored swagger
model). This slice adds **no** swagger-spec fields, but the generated model must still exist to
compile.
