# Personal calendar + organizer follow/subscribe — Design

_2026-06-27. A user-facing feature (NOT part of the admin suite): a personal
calendar at `/me/calendar` that overlays the two streams of events relevant to a
signed-in user, plus the net-new follow/subscribe relationship the "followed
organizers" stream depends on._

## Problem & goal

Users discover events via the list/map feed and track their own sign-ups at
`/me/practices`, but there is no time-oriented view pulling together everything
relevant to them. Goal: a Google-Calendar-style **month / week / day** calendar
showing:

1. **Attending** — events the user has an active RSVP to (`going` / `accepted` / `waitlist`).
2. **From followed** — events from organizers the user **subscribes to**.

An event can be both. The two sources are visually distinguished (colour + legend).

The RSVP stream already existed (`internal/rsvp`, `/me/practices`). The
**follow/subscribe relationship did not exist at all** and is built here.

## Decisions

- **Build both** the follow system and the calendar together (the calendar's
  "followed" stream is meaningless without follows).
- **Views**: month (default) + week + day.
- **Marking**: colour + legend — accent = attending, amber = from a followed
  organizer, accent + amber ring = both.
- **No date library** on the frontend — native `Intl`/`Date` (consistent with
  the rest of the app; `lib/format.ts`).

## Data model

`organizer_follows` (migration `000018`):

```sql
CREATE TABLE organizer_follows (
    user_id           uuid NOT NULL,
    organizer_user_id uuid NOT NULL,   -- the organizer's OWNER user id (= events.organizer_id)
    created_at        timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, organizer_user_id)
);
CREATE INDEX organizer_follows_user_idx      ON organizer_follows (user_id);
CREATE INDEX organizer_follows_organizer_idx ON organizer_follows (organizer_user_id);
```

**Key decision — store the owner user id, not `organizers.id`.** `events.organizer_id`
already equals the organizer's owner user id, so listing a follower's events is a
direct `events.organizer_id IN (followed_owner_ids)` with no indirection. The
**public API addresses organizers by `organizers.id`** (the profile id the
frontend holds); handlers resolve profile → owner via `organizers.GetByID` at the
edge. Composite PK gives uniqueness + idempotent follow. No FKs — matches the
existing loose-reference convention (`events.organizer_id`, `event_rsvps.user_id`).

**Verified-only.** You may only follow a **verified** organizer profile; an
unknown or non-verified profile id returns **404** (mirrors the public organizer
page — no existence leak for pending/rejected/draft).

**Compliance.** Follow/unfollow are low-sensitivity user-preference actions and
deliberately do **not** write `audit_log` (unlike moderation/verification
transitions). Flagged here so it can be revisited if follow data is ever
reclassified as engagement/behavioural data under a retention or privacy control.

## API

All under the `/api/v1/` basePath; plain `net/http`, mounted ahead of the
go-swagger mux (no swagger route edits), same as `internal/http/organizers`.

| Method + path | Auth | Behaviour |
|---|---|---|
| `POST /me/follows/{organizerId}` | jwt | Follow (organizerId = `organizers.id`). 404 if not verified. `200 {"following":true}`. Idempotent (`ON CONFLICT DO NOTHING`). |
| `DELETE /me/follows/{organizerId}` | jwt | Unfollow. `200 {"following":false}`. |
| `GET /me/follows` | jwt | `[{profile_id, name, logo_url?}]` (logo_url deferred, shared with org page). |
| `GET /me/calendar?from&to` | jwt | Calendar events in `[from, to)`. RFC3339; `to` defaults to `from + 90d`, window capped at 366d; `to <= from` → 400. |
| `GET /organizers/{id}` | public | Gains `is_following` (true only for an authenticated caller who follows it; false otherwise / anon). |

**Calendar response** — each row is the standard API `Event` plus two flags:

```json
{ "...event fields...": "…", "attending": true, "from_followed": false }
```

The flags live on a thin response wrapper (`*apimodels.Event` embedded +
`attending`/`from_followed`); the domain `Event` and `EventToAPI` are untouched.

## Calendar aggregation

Two streams merged in Go (reusing existing batch-loaded queries — no new SQL idioms):

1. **Followed**: `follows.ListEventsFromFollowed(userID, from, to)` → `ListFollowedOwnerIDs`
   then `events.ListForCalendar(ownerIDs, from, to)` (published-only, range-filtered,
   fully enriched). Flag `from_followed`.
2. **Attending**: `rsvp.ListActiveEventsInRange(userID, from, to)` → reuses
   `ListByUser({going,waitlist,applied,accepted})` + `attachEvents`, filtered to
   `[from, to)`. Flag `attending`.

Merge by event id in a map, OR-ing the flags (an event in both → both true), then
re-enrich the **deduped id set** via `events.GetEnriched(ids)` so every row is
shaped identically, and serialise through `EventToAPI`. `GetEnriched` sizes its
limit to the exact id set so a busy calendar never silently truncates.

Events are single-occurrence (`starts_at` + optional `ends_at`, `timestamptz`),
so each lands on exactly one day.

## Frontend

- `lib/calendar.ts` — civil-date math: a calendar day is a UTC-midnight `Date`,
  all arithmetic uses UTC methods (no DST/zone drift); events are bucketed by
  their **Europe/Moscow** civil day (`Intl` `en-CA` → `YYYY-MM-DD`), matching how
  dates render elsewhere. `monthGrid` (42 cells), `weekGrid` (7), day; `todayCivil`,
  `shiftMonth`, labels.
- `app/me/calendar/page.tsx` — `"use client"`, the `/me/*` auth-gate pattern,
  `Segmented` month/week/day + prev/next/today nav, TanStack Query keyed by
  `[view, range]` (range widened ±1 day for the tz boundary; bucketing is exact).
  Colour-coded `EventChip` (month/week) / agenda rows (day) + a legend.
- `app/organizers/[id]/page.tsx` — Follow/Following toggle (authed-only, optimistic,
  reverts on error), seeded from `is_following`.
- API client (`lib/api.ts`): `followOrganizer`, `unfollowOrganizer`,
  `fetchFollowedOrganizers`, `fetchCalendar`; `getPublicOrganizer` now sends the
  bearer token (when present) so the backend can compute `is_following`. Types
  `CalendarEvent` (+`attending`/`fromFollowed`), `FollowedOrganizer`.
- Nav: "Календарь" in `AuthButton` (desktop) + `TabBar` (mobile).

## Out of scope / deferred

- `logo_url` on the follow list (shared follow-up with the organizer page).
- Surfacing a followed organizer's published events on its public page.
- Recurring events (the model is single-occurrence).
- Integration tests against a real DB.
