# Organizer Page Events + Clickable Host — Design

_Design spec · 2026-07-01 · frontend + one additive backend query param (no migration)_

## Problem

Two gaps surfaced from the live demo:

1. **Event detail, «Ведущий» block** — only the «✓ Проверен» badge links to the organizer page. The organizer's name (e.g. «Музей «Гараж»») is not clickable, so the obvious tap target does nothing.
2. **Public organizer page (`/organizers/[id]`)** — renders only the name, badge, description, website, and a follow button. It shows none of the organizer's events, so the page is nearly empty. The code already carries a TODO anticipating an "event-list-by-organizer public filter."

## Goals

- Make the **whole host row** (avatar + name) on event detail a link to the organizer page when the organizer is verified (has a `profile_id`).
- Show the organizer's **upcoming events** (primary list) and **past events** (secondary section) on the public organizer page.

## Non-goals

- No pagination, no category filters on the organizer page.
- No permission/visibility changes — the public events list stays **published-only**.
- No DB migration. The only backend change is one additive query parameter on an existing endpoint.
- The host row links only when `profile_id` is present (verified organizer). Unverified organizers stay plain text (consistent with today — `VerifiedBadge` only links when given a `profileId`).

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Past events | **Both lists now** — «Предстоящие» (primary) + «Прошедшие» (separate section below) |
| Backend filter | Add `organizer_id` query param to public `GET /events` (reuses internal `ListFilter.OrganizerIDs`) |
| `organizer_id` value | The **profile id** (`organizers.id`), resolved server-side to the owner user id of a **verified** organizer |
| Fetch strategy | **One** request `GET /events?organizer_id={id}`, split client-side by `starts_at` vs now |
| Past cap | Display-only cap (~10 most-recent) on the frontend; not a backend limit |

## Design

### Part 1 — Clickable host row (frontend only)

File: `frontend/components/EventDetailView.tsx` (the «Ведущий» `Section`, ~lines 81–100).

Today the row is a static `div` with the name `<p>` and the `VerifiedBadge` (the badge is the only link). Change: when `event.organizer.profile_id` is set, wrap the **avatar + name + affiliation** block in a `Link href={`/organizers/${event.organizer.profile_id}`}` so the entire row is the tap target.

- The `VerifiedBadge` stays rendered for the «✓ Проверен» visual, but is no longer itself a nested link (avoid a `<Link>` inside a `<Link>` — render the badge's plain `<span>` form, i.e. call `VerifiedBadge` without a `profileId`, or render the badge markup directly). The outer row link now carries navigation.
- When `profile_id` is absent (unverified organizer), render the row exactly as today — plain, no link.
- Preserve current styling (avatar circle, `text-[17px] font-medium` name, affiliation line). Add a subtle affordance consistent with other tappable rows (e.g. `hover:opacity-70`).

This is presentation-only; no data or API change.

### Part 2 — Organizer events

#### Backend (additive, no migration)

Expose published events for a given organizer on the existing public list endpoint.

- **Swagger** (`backend/api/swagger.yaml`, `GET /events`): add an optional `organizer_id` query param (`type: string, format: uuid`). Additive; regenerate with **`make generate-api`** only (never `make generate-all`). Generated artifacts stay gitignored/uncommitted.
- **Handler** (`backend/internal/http/handlers/events.go`, the `listEvents` path that already maps `from`/`to` into the events `ListFilter`): when `organizer_id` is present, resolve it from a **profile id** (`organizers.id`) to the organizer's **owner user id**, gating on **verified** status (reuse the resolution the follows feature already does: `organizers.id` → `owner_user_id` where `verification_status = 'verified'`). If the id doesn't resolve to a verified organizer, return an **empty list** (no leak, no error). Set `ListFilter.OrganizerIDs = [ownerUserID]`.
- The endpoint stays `security: []` and **published-only** (existing behavior). `from`/`to` continue to work and compose with `organizer_id` (not used by the organizer page in this iteration, but the combination must remain valid).
- Resolution lookup must select only what it needs (no email / private fields).

#### Frontend (`app/organizers/[id]/page.tsx`)

- Add an API helper `fetchEventsByOrganizer(organizerId)` → `GET /events?organizer_id={id}` returning `LiaEvent[]` (reuse the existing `apiEventToLia` mapping; same shape as `fetchPublishedEvents`).
- On the organizer page, after the org loads, fetch its events once. Split client-side using `starts_at`:
  - **Предстоящие**: `starts_at >= now`, ascending by `starts_at` — primary list of `EventCard`s.
  - **Прошедшие**: `starts_at < now`, **descending** by `starts_at` (most recent first), sliced to the first ~10 for display — a separate section rendered below, under an «Прошедшие» heading.
- Empty states: if no upcoming, show «Пока нет предстоящих мероприятий»; if no past, omit the «Прошедшие» section entirely.
- Reuse the existing `EventCard` component for consistency with the discovery feed.
- Remove the stale TODO comment now that the filter exists.

«now» is the client's current time; events are bucketed by their `starts_at` instant. Day-precision nuance (an event starting earlier today) is acceptable — bucketing by instant is fine for this view.

## Affected files

**Frontend**
| File | Change |
| --- | --- |
| `frontend/components/EventDetailView.tsx` | Wrap host row in a `Link` to `/organizers/{profile_id}`; badge as plain span |
| `frontend/app/organizers/[id]/page.tsx` | Fetch + render upcoming / past event lists; remove TODO |
| `frontend/lib/api.ts` | Add `fetchEventsByOrganizer(id)` helper |

**Backend**
| File | Change |
| --- | --- |
| `backend/api/swagger.yaml` + `make generate-api` | Add `organizer_id` query param to `GET /events` |
| `backend/internal/http/handlers/events.go` | Resolve `organizer_id` (profile→verified owner) into `ListFilter.OrganizerIDs` |
| (resolution) | Reuse the organizers profile→owner lookup used by the follows feature; verified-only |

## Risks & edge cases

- **Profile→owner resolution**: if `organizer_id` is malformed or not a verified organizer, return `[]` (not 404/500) — the page simply shows no events. Mirrors the no-leak posture of the public organizer endpoint.
- **`<Link>` nesting**: the badge must not remain a `Link` once the row is a `Link`. Render the badge as a plain span on event detail.
- **Past-list size**: capped in the UI only; a prolific organizer's full published history is still returned by the single query. Acceptable at demo scale; note as a follow-up if histories grow (then add server-side `to=now` + limit).
- **Swagger regen**: `make generate-api` only (the `generate-all` proto step is known-broken here); generated models remain gitignored and must not be committed.

## Success criteria

- Tapping the organizer name (or avatar) on a verified event's «Ведущий» block navigates to that organizer's page.
- The organizer page lists the organizer's upcoming events (primary) and past events (secondary, most-recent-first), published-only.
- `GET /events?organizer_id={verified profile id}` returns that organizer's published events; an unknown/unverified id returns `[]`; no email or private fields exposed; no DB migration added.
