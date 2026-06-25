# RSVP domain + attendance pages — design (first slice)

**Date:** 2026-06-26
**Status:** approved (brainstorming) → next: writing-plans
**Scope owner:** Lia monolith (`backend/internal/rsvp`, frontend `/me/*` + event detail)
**Grounded in:** `docs/design_agent_prompt.md` §5.3, §5.14.7, §5.14.8; `docs/event_discovery_mvp_technical_stack.md` §2.1, §5.5, §13; `backend/internal/rsvp/doc.go` (skeleton).

---

## 1. Goal

Turn the `rsvp` skeleton into a working domain so a signed-in user can sign up for a
practice, and see what they've signed up for. This is the P0 slice flagged in
`docs/HANDOFF.md` and Phase 3 item 7 of
`docs/superpowers/plans/2026-06-25-passwords-and-me-suite.md` (currently *blocked on the RSVP domain*).

The detail-page **"Записаться"** button is currently a stub. This slice makes it real.

## 2. Scope (confirmed)

**In:**
- All three signup modes: `open` / `application` / `external`.
- Capacity limit + waitlist (with auto-promotion on free-up).
- Applications with curator question + organizer accept/decline.
- Pages `/me/practices` and `/me/applications`.
- `.ics` calendar export.
- Module `backend/internal/rsvp` built on the `internal/events` pattern.

**Out (explicit non-goals — later slices):**
- Saved events / bookmarks (`/me/saved`) — separate `saved_events` domain (roadmap item 5).
- Past-tab reflections ("Как прошло?") — feeds recommendations, separate.
- Transactional notifications (`application_accepted`, `waitlist_promoted`, reminders) — notifications domain is blocked. Status is reflected in the UI on page load only; no email/push in this slice.
- Organizer cabinet `/o` — only a **minimal** organizer accept/decline surface is built here (see §6).
- Recurring practices.
- Guest (no-login) sign-up flow.

## 3. Decisions on open spec questions (`design_agent_prompt.md` §789–795)

1. **Sign-up requires authentication.** No guest flow. `.ics` download works without login (§486).
2. **"Сохранить" not included** (separate domain).
3. **Reflections deferred.**
4. **Transactional notifications deferred** — UI reflects status on load; no push/email.
5. **Application mode needs an organizer action.** Since `/o` does not exist, a minimal
   organizer-side accept/decline surface is added on the organizer's own event (reached via
   `/events/mine`). Without it, application mode is non-functional.

## 4. Data model (Lia migrations)

### 4.1 `events` — new columns
| Column | Type | Notes |
|---|---|---|
| `signup_mode` | enum `open` \| `application` \| `external` | default `open` |
| `capacity` | int, nullable | `null` = unlimited |
| `curator_question` | text, nullable | required when `signup_mode = application` |
| `external_registration_url` | text, nullable | required when `signup_mode = external` |

Follow the existing `events` enum-as-string pattern (cf. `EventStatus` /
`StatusSQL` in `backend/internal/models/event.go`): store the string column, expose a typed
Go enum, convert in `BeforeInsert`/`BeforeUpdate`/`AfterSelect`.

### 4.2 `event_rsvps` — new table
| Column | Type | Notes |
|---|---|---|
| `id` | uuid PK | |
| `event_id` | uuid FK → events | |
| `user_id` | uuid FK → users | |
| `status` | enum | `going \| waitlist \| applied \| accepted \| declined \| withdrawn \| cancelled` |
| `application_answer` | text, nullable | for `application` mode |
| `created_at` | timestamptz | also the waitlist ordering key |
| `updated_at` | timestamptz | |

- **UNIQUE `(event_id, user_id)`** — one row per user per event. Re-signing after
  `cancelled`/`withdrawn` reuses the row (status transition), not a second row.
- Index `(event_id, status)` for seat counting and waitlist ordering.
- `seats_remaining = capacity − count(status = going)`, computed on read (not stored).
- Waitlist position = rank by `created_at` among `status = waitlist` for the event (computed, not stored), to avoid renumbering on every change.

### 4.3 Status semantics
- `open` mode: sign-up → `going` if seats available (or `capacity` is null), else `waitlist`.
- `application` mode: sign-up → `applied`. Organizer decision → `accepted` (→ effectively a confirmed seat) or `declined`. If accepting would exceed capacity → `waitlist`.
- `external` mode: no local RSVP row; `POST /rsvp` returns 409 with the registration URL.
- User cancel: `going`/`accepted` → `cancelled`; `waitlist` → `cancelled`; `applied`/`accepted`(pending) → `withdrawn`.

## 5. Backend — `internal/rsvp` module

Mirror `internal/events`: `repository.go`, `service.go`, `service_test.go`; register in
`module.go`. Swagger-first: edit `backend/api/swagger.yaml`, then `make generate-all` (generated
`internal/http/models` is gitignored and must exist to build) then `make generate-api`.

### 5.1 Endpoints
| Method + path | Auth | Purpose |
|---|---|---|
| `POST /events/{id}/rsvp` | jwt | Sign up. Body `{application_answer?}`. Behaviour by `signup_mode` (§4.3). |
| `DELETE /events/{id}/rsvp` | jwt | Cancel / leave waitlist / withdraw application. **Transactional**: on freeing a `going` seat, auto-promote the oldest `waitlist` row to `going`. |
| `GET /me/practices?tab=upcoming\|past` | jwt | Caller's confirmed attendance (`going`/`waitlist`/`accepted`) joined to event + date. |
| `GET /me/applications?status=...` | jwt | Caller's applications across statuses. |
| `GET /events/{id}/applications` | jwt (organizer of event) | List applications for the organizer's own event. 403 if not organizer. |
| `POST /events/{id}/applications/{rsvpId}/decision` | jwt (organizer) | Body `{decision: accept\|decline}`. accept → `accepted` (or `waitlist` if full); decline → `declined`. |
| `GET /events/{id}/calendar.ics` | public | VEVENT generated from `event_dates`, tz `Europe/Moscow`. No OAuth. |

### 5.2 Event formatter additions
Extend `EventToAPI` (`backend/internal/http/formatter/event.go`) with `signup_mode`,
`capacity`, `seats_remaining`, and — for authenticated requests — `my_rsvp_status`
(and waitlist position when applicable). Batch-load the caller's RSVP per event the same
way `loadOrganizers`/`loadVenues` batch-load (no N+1).

### 5.3 Concurrency / edge cases
- **Last-seat race**: seat allocation runs in a transaction with `SELECT … FOR UPDATE`
  on the event row (or an advisory lock keyed by event id) so two concurrent sign-ups can't
  both take the final seat.
- **Double sign-up**: UNIQUE `(event_id, user_id)` → 409.
- **Auto-promotion**: on cancel of a `going`/`accepted` seat, promote oldest `waitlist` → `going`, same transaction.
- **External mode**: `POST /rsvp` → 409 + `external_registration_url`.
- **Missing dates**: `.ics` for an event without `event_dates` → 422 (nothing to export).
- **Decision on non-applied / already-decided row** → 409.

## 6. Frontend

### 6.1 Event detail (`/events/[id]`)
CTA by `signup_mode`:
- `open`: **"Записаться"** → after success "Вы записаны" + "Отписаться". Seats counter shown; when full → **"В лист ожидания"**, then "Вы N-й в листе ожидания" + "Покинуть лист".
- `application`: **"Подать заявку"** → sheet with `curator_question`. States: "Заявка отправлена" / "Заявка принята" / "Заявка отклонена"; pending → "Отозвать заявку".
- `external`: **"Записаться на сайте организатора"** → opens `external_registration_url` (new tab) with caption "Запись ведёт организатор".
- **"В календарь"** → downloads `.ics` (works logged-out).

### 6.2 `/me/practices`
Tabs "Предстоящие" / "Прошедшие". Row: date, title, venue, status chip
(записан / в листе ожидания), contextual action. (Past-tab reflection UI deferred.)

### 6.3 `/me/applications`
Tabs "В ожидании" / "Принятые" / "Отклонённые" / "Отозванные". Card: curator question,
user's answer (expandable), status + timestamp. From "В ожидании": withdraw.

### 6.4 Organizer applications (minimal)
On the organizer's own event (reached via `/events/mine`): a list of applications with
"Принять" / "Отклонить" per row, backed by `GET /events/{id}/applications` +
`POST …/decision`. Minimal styling — full `/o` cabinet is a later slice.

### 6.5 API client
`frontend/lib/api.ts` + `lib/types.ts`: `rsvp()`, `cancelRsvp()`, `fetchMyPractices()`,
`fetchMyApplications()`, `fetchEventApplications()`, `decideApplication()`, plus mapping the
new event fields (`signupMode`, `capacity`, `seatsRemaining`, `myRsvpStatus`).

## 7. Testing (TDD)

Service-level tests drive the build:
- open: sign up to capacity → `going`; one past capacity → `waitlist`.
- cancel a `going` seat → oldest `waitlist` promoted to `going`.
- application: applied → accept → `accepted`; applied → decline → `declined`; accept when full → `waitlist`.
- external mode `POST /rsvp` → 409 with URL.
- double sign-up → 409.
- anon → 401 on all jwt routes; `.ics` reachable anon.
- decision by non-organizer → 403; decision on already-decided row → 409.
- `.ics` output is valid VEVENT with `Europe/Moscow`.

Follow `internal/events/service_test.go` conventions.

## 8. Dependencies & follow-ups

- **Notifications** (`application_accepted`, `waitlist_promoted`, reminders) — next domain; this slice leaves hooks but no delivery.
- **Saved events** — separate `saved_events` domain.
- **Organizer cabinet `/o`** — replaces the minimal accept/decline surface here.
- Codegen gotchas: see `docs/superpowers/plans/2026-06-25-passwords-and-me-suite.md`
  "Implementation notes" (mockery under Go 1.26, `make generate-all` before build, `-vet=off`).
