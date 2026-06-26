# Personal calendar + organizer follow/subscribe — Plan

_Status: **DONE — built, reviewed, deployed live, merged to `main` (`d254578`) 2026-06-27.**
Spec: [`../specs/2026-06-27-calendar-and-follows-design.md`](../specs/2026-06-27-calendar-and-follows-design.md).
Deploy: [`../runbooks/2026-06-27-calendar-follows-deploy.md`](../runbooks/2026-06-27-calendar-follows-deploy.md)._

A user-facing personal calendar (`/me/calendar`, month/week/day) overlaying the
user's active RSVPs and events from organizers they follow, colour-coded. Includes
the net-new follow/subscribe system the "followed" stream depends on.

## Part A — Backend (follow system + calendar endpoint)

- **A1. Migration `000018_organizer_follows`** — table keyed by `(user_id, organizer_user_id)`
  where `organizer_user_id` = the organizer's OWNER user id (`= events.organizer_id`),
  two indexes, composite PK. Additive.
- **A2. `internal/follows`** — `repository.go` (`Add` `ON CONFLICT DO NOTHING` /
  `Remove` / `IsFollowing` / `ListFollowedOwnerIDs` / `ListFollowedOrganizers` JOIN);
  `service.go` (`Follow`/`Unfollow` resolve profile→owner via `organizers.GetByID`,
  verified-only → `ErrNotFound`; `IsFollowing`; `ListFollowed`; `ListEventsFromFollowed`).
- **A3. events** — `ListFilter` gains `OrganizerIDs` / `IDs` / `From` / `To` (+ WHERE
  clauses in `List`); service `ListForCalendar` (published, owner ids, range) +
  `GetEnriched(ids)` (limit sized to the id set — no silent truncation).
- **A4. rsvp** — `ListActiveEventsInRange` reusing `ListByUser({going,waitlist,applied,accepted})`
  + `attachEvents`, filtered to `[from,to)`.
- **A5. `internal/http/follows`** — plain net/http handler (mirrors `internal/http/organizers`):
  follow/unfollow/list + `GET /me/calendar` (merge both streams by event id, OR flags,
  re-enrich deduped set via `GetEnriched` → `EventToAPI`, wrapper adds `attending`/`from_followed`).
  `is_following` added to the public organizer handler.
- **A6. Wiring** — `module.go` `SetFollows` + router branch (`/me/follows*`, `/me/calendar`) +
  pass `Follows` into org Deps; `application.go` constructs `followsSvc` after events + organizers.

## Part B — Frontend

- **B1.** `lib/api.ts` `followOrganizer`/`unfollowOrganizer`/`fetchFollowedOrganizers`/`fetchCalendar`,
  `getPublicOrganizer` sends bearer for `is_following`; types `CalendarEvent`, `FollowedOrganizer`.
- **B2.** Follow toggle on `/organizers/[id]` (authed, optimistic, seeded from `is_following`).
- **B3.** `/me/calendar` page + `lib/calendar.ts` (UTC-civil-date grid math, Europe/Moscow
  bucketing, no date lib); `Segmented` month/week/day + nav; colour+legend chips; auth gate.
- **B4.** Nav entry — "Календарь" in `AuthButton` (desktop) + `TabBar` (mobile).

## Verification (done)

- `go build` + 35 backend test packages pass (new `internal/follows` service test:
  verified-only gate, owner resolution, range delegation); frontend `tsc`+`eslint`+`next build` clean.
- High-effort code review (8 finder angles → verify): one fix applied (`GetEnriched`
  limit sized to id set); other candidates refuted (`WHERE id IN` constraint, idiomatic
  JS date semantics, always-zoned backend timestamps, no grid day-gap).
- Prod (live): migration `18|f` + table/indexes; health 200; anon 401 on the three new
  endpoints; authed `/me/calendar`+`/me/follows`→200 `[]` (merge path); follow non-verified→404;
  site + `/me/calendar` + organizer routes 200.

## Not prod-verified

The visual followed/attending/both flow needs a verified org + a logged-in user
following it (the auto-mode classifier blocks bootstrapping that from a throwaway
prod account). Anon/auth gating and the empty-merge path ARE verified.
