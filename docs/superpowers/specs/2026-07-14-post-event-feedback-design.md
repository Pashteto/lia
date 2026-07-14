# Private Post-Event Feedback (R3) — Design

_Design spec · 2026-07-14 · new `internal/feedback` domain + migration 000019 + frontend_

## Problem

The first-clients pilot's **primary goal** is that friendly organizations «collect feedback from their audience after the events» to learn what to improve. The platform has **no feedback mechanism** — only «Пожаловаться» (negative-only, routed to admin moderation). Organizers today would have to collect feedback entirely off-platform.

## Goals

- After an event **ends**, a participant can leave a **star rating (1–5) + optional comment**.
- The organizer (event owner) sees all feedback for their event, plus an aggregate (average + count).
- Feedback is **private to the organizer** (and admin) — there is no public rating.

## Non-goals

- **No public ratings/reviews** — nothing renders on the public event or organizer page.
- No per-organizer aggregate reputation across events (v1 is per-event only).
- No custom organizer-authored questionnaires (fixed shape: rating + comment).
- No editing/deleting of feedback by the participant in v1 (YAGNI). Admin can remove abusive feedback via the existing moderation pattern.
- No notifications/reminders to leave feedback (no notifications domain).

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Visibility | **Private to the event owner** (+ admin). No public rating. |
| Format | **★ rating (1–5) required + optional free-text comment** |
| Who may leave | Users with an **active RSVP** (`going`/`accepted`) on the event |
| When | Only after the event has **ended** (`ends_at` else `starts_at` < now, МСК) |
| Cardinality | **One feedback per (event, user)** |
| Author identity to organizer | **Name only, never email** (consistent with applications) |

## Design

### Data — migration `000019_event_feedback`

```
event_feedback (
  id          uuid PK,
  event_id    uuid NOT NULL REFERENCES events(id),
  user_id     uuid NOT NULL,          -- the participant
  rating      smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment     text NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, user_id)          -- one feedback per participant
)
```
Index on `event_id` for the organizer's list/aggregate query.

### Backend — new domain `internal/feedback` (plain net/http, mounted ahead of swagger, mirroring `internal/complaints`)

- **`POST /api/v1/events/{id}/feedback`** (authed). Gates, each with a distinct Russian error:
  - Event exists → else 404.
  - Event **ended** (`ends_at` else `starts_at` < now in `Europe/Moscow`) → else 422 «Отзыв можно оставить после завершения события».
  - Caller has an **active RSVP** (`going`/`accepted`) on the event → else 403 «Отзыв могут оставить только участники».
  - No existing feedback from this user → else 409 «Вы уже оставили отзыв».
  - Body: `rating` 1–5 (required), `comment` (optional). Invalid rating → 422.
  - Success → 201.
- **`GET /api/v1/events/{id}/feedback`** — **owner-or-admin only**. Returns `{ average, count, items:[{rating, comment, author:{name}, created_at}] }`. Non-owner/anon → 403/401 (no leak). Author enriched with **name only** via batched name-load (reuse `LoadApplicantNames` pattern; never email).
- **`GET /api/v1/me/feedback?event_id=`** (authed, optional) — returns the caller's own feedback for an event (for button state), or 204/empty.

Service composes `events` (for the ended-check + ownership) and `rsvp` (for the active-RSVP check). All checks server-side; the `me/feedback` read never exposes others' data.

### Frontend

- **Participant — event detail (`/events/[id]`, `EventDetailView`):** when the event has ended and the caller had an active RSVP, replace the RSVP CTA with a **«Как всё прошло?»** block: a large touch-friendly ★ selector + optional comment textarea + submit. After submit → «Спасибо за отзыв 🙌» (and, on reload, the block reflects the already-left state via `me/feedback`). If ended but the user wasn't a participant → no block.
- **Participant — `/me/practices` (past tab):** each past event where feedback is possible shows a «Оставить отзыв» affordance (secondary entry point into the same flow).
- **Organizer — feedback view:** a **«Отзывы»** section on the owner's event view (in `/events/mine` expander or a dedicated `/events/[id]/feedback`), linked from the `/organizer` hub. Shows average ★, count, and the list (rating + comment + author name + date). Empty state: «Отзывы появятся после завершения события».

### Design/UX

- ★ widget is the emotional core — large, animated fill, keyboard/tap accessible.
- Comment clearly optional («необязательно»).
- Before the event ends, the block is absent (not a disabled teaser) to avoid confusion.
- Organizer view is calm and private — framed as «для вас», reinforcing candour.

## Data flow

Event ends → participant opens detail (or `/me/practices`) → sees feedback block → `POST …/feedback` (server re-checks ended + active-RSVP + not-duplicate) → row stored. Organizer opens their event's «Отзывы» → `GET …/feedback` (owner-gated) → average + list rendered privately.

## Error handling

Distinct Russian messages for: not-ended (422), not-a-participant (403), duplicate (409), bad rating (422), non-owner read (403). No raw codes/stacks leak to UI.

## Privacy / compliance

Feedback is personal data. Read access is strictly **owner + admin**; nothing is public or indexed. Author email is never exposed (name only). Abusive feedback removable by admin via the existing moderation/takedown pattern (future hook, not built in v1). This is an intentional, documented privacy boundary — mirrors the complaints/applications posture.

## Testing

- **Integration (backend):** post before event ends → 422; post as non-participant → 403; happy post as participant on ended event → 201; duplicate → 409; owner GET → list+average; non-owner GET → 403; anon → 401; author payload has name, no email.
- **Unit:** aggregate (average/count) with 0, 1, N rows; ended-check timezone boundary (`Europe/Moscow`, `ends_at` vs `starts_at` fallback).
- **Manual (the pilot's actual goal):** run/seed a past event with 2+ participants, each leaves a rating+comment, organizer sees the private average + comments; a non-participant cannot.

## Rollout

Migration 018 → **019**. New backend domain + frontend. Standard build-on-Mac→`save|ssh|load` deploy; pre-migration DB dump per the deploy runbook convention. Fully independent of R1/R2 — can ship in parallel.
