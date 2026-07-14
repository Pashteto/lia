# Editing Published Events (R2) — Design

_Design spec · 2026-07-14 · frontend + backend (no migration; reuses audit_log + waitlist promotion)_

## Problem

Once an event is published it is effectively immutable through the product. `events.Update` allows edits only when the event is **draft**; `published`/`pending_review`/`rejected`/`cancelled` return **409 `ErrNotEditable`** (`internal/events/service.go:227`). There is no edit page in the UI at all (`/events/[id]/edit` does not exist); the only post-create mutation surfaced is **publish**. Capacity is additionally frozen — PATCH support was explicitly deferred (`internal/http/formatter/event.go:314`; `UpdateParams` has no `Capacity` field).

**Concrete pain (from the first-clients QA):** a real organizer who mistypes the time, address, or price of a published event cannot fix it — the only recourse is cancel + recreate, which loses the event's URL and everyone's RSVPs.

## Goals

- Allow the owner to **edit a published event** and have it **stay published**.
- Editable fields: title, description, dates (`starts_at`/`ends_at`), venue, cover, categories, price, and **capacity**.
- Changing **capacity** correctly reconciles the waitlist (increasing the limit promotes FIFO from `waitlist` → `going`).
- Every edit of a published event is written to `audit_log`.

## Non-goals

- No DB migration.
- **`signup_mode` cannot change after publish** — switching modes would strip the meaning of already-submitted applications/RSVPs. Only content fields + capacity are editable.
- No moderation state transitions — the event stays `published` (moderation remains reactive via complaints/takedown; owner never sets `pending_review`).
- No participant notifications (there is no notifications domain) — we surface an honest warning instead.
- Reducing capacity does **not** evict existing confirmed attendees.

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Edit scope | **All content fields + capacity** (not signup_mode) |
| Capacity increase | Reconcile waitlist: FIFO-promote `waitlist` → `going` in one tx |
| Capacity decrease | **Cannot go below currently-occupied seats** (`going`+`accepted`); attempt → 409 with a clear message. No one is evicted. |
| Status after edit | Stays `published` (no re-moderation) |
| Audit | Each published-event edit writes `audit_log` in the same tx |
| Edit UI | New `/events/[id]/edit` reusing `CreateEventForm` in edit mode |

## Design

### Backend — `internal/events/service.go`

1. **Editable-status gate.** Replace the draft-only guard: allow `Update` when status ∈ {`draft`, `published`}. Keep 409 `ErrNotEditable` for `pending_review`/`rejected`/`cancelled`.
2. **Capacity in `UpdateParams`.** Add a `Capacity *int` field (nil = unchanged; distinguish "omitted" from "set to unlimited" — treat explicit null as unlimited only if the API models it; otherwise omitted = unchanged). Formatter maps it (remove the "PATCH deferred" note at `formatter/event.go:314`).
3. **Capacity reconciliation (transactional, `FOR UPDATE`):**
   - Compute `occupied = count(going + accepted)` under lock (reuse `rsvp/repository.go:206-219`).
   - **New cap < occupied** → 409 «Нельзя уменьшить лимит ниже числа записавшихся (N)».
   - **New cap > old cap (or old unlimited→limited with room)** → promote the oldest `waitlist` rows to `going` until the cap is filled (reuse the FIFO promotion at `rsvp/repository.go:136-174`).
   - **Unlimited → limited** and **limited → unlimited** handled as the boundary cases of the above.
   - All of this + the field updates + the `audit_log` write happen in **one transaction**.
4. **Audit.** On a `published` edit, write an `audit_log` row (actor = owner, action = `event.edit`, target = event id, diff summary optional). Draft edits need not audit (consistent with today).

### Backend — `PATCH /api/v1/events/{id}`

Already exists (owner-gated). Extend the accepted body with `capacity`. Reject `signup_mode` changes on non-draft (422 «Режим записи нельзя изменить после публикации»).

### Frontend

- **New page `/events/[id]/edit`** — reuses `CreateEventForm` in an **edit mode**: prefill from `GET /events/{id}` (owner fetch), submit via `PATCH`, on success route back to the event. Signup-mode control is **read-only / hidden** when the event is published (mode locked); capacity remains editable.
- **Entry points:** «Редактировать» button on `/events/mine` rows (owner) and on the event detail for the owner (next to/instead of the publish affordance for already-published events).
- **Capacity-change affordance:** when the owner raises the limit, show «N из листа ожидания получат место»; when they try to lower below occupied, show the 409 message inline.

### Design/UX

- **Date/venue change warning:** since there are no notifications, show a non-blocking notice «Участники уже записаны — предупредите их об изменении самостоятельно» when `starts_at`/venue changes on a published event.
- Locked `signup_mode` shows a subtle «Режим записи зафиксирован после публикации» hint.
- Edit form mirrors create form layout so it feels familiar (shared component).

## Data flow

Owner opens `/events/[id]/edit` → prefilled form → `PATCH /events/{id}` → service verifies ownership + editable status → within one tx: apply field changes, reconcile capacity (promote/deny), write `audit_log` → event stays `published` → redirect to detail.

## Error handling

- Non-owner → 403. Non-editable status → 409 `ErrNotEditable`. Capacity below occupied → 409 with count. `signup_mode` change attempt → 422. All messages Russian, surfaced inline.

## Testing

- **Integration (backend):** publish → PATCH title → still published; PATCH capacity up with a waitlist → oldest waitlist row promoted to `going`; PATCH capacity below occupied → 409; PATCH signup_mode on published → 422; non-owner PATCH → 403; audit_log row written on published edit.
- **Unit:** capacity reconciliation boundaries (unlimited↔limited, exact-occupied edge).
- **Manual:** organizer fixes a typo + moves the date on a live event; a capacity bump promotes a waitlisted guest.

## Rollout

No migration (schema stays 018). Backend + frontend. Standard build-on-Mac→`save|ssh|load` deploy. Depends conceptually on R1 (shared capacity/mode form plumbing) — build R1 first, then this reuses the fields.
