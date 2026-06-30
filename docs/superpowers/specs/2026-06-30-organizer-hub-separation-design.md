# Organizer Hub — Separating the Organizer Experience

_Design spec · 2026-06-30 (rev. 2026-07-01) · frontend + one additive backend field (no migration)_

## Problem

Presence.Tarski serves two audiences from one undifferentiated navigation:

- **Attendees** — browse, search, map, personal calendar, RSVP, follow organizers.
- **Organizers** — create events, manage their events, handle applications, maintain a verified organizer profile.

Every organizer tool already exists as a working page (`/events/new`, `/events/mine`, `/me/organizer`, plus the applications-management panel). The problem is purely **information architecture**: organizer actions are scattered into the attendee surfaces. "Создать" sits as a peer of "browse" in the mobile tab bar; "Мои события" hangs in the header; the verification status is buried in `/me/organizer`. There is no front door that says "this part is for organizers."

The site is attendee-first, and the organizer flow should feel like a distinct, opt-in space — without taking anything away from attendees.

## Goals

- Give organizer tools one clear **front door** and a dedicated **home** (a hub page).
- Keep the **attendee experience fully intact** — no feature removed, moved, or gated.
- "Soft framing" only: this is presentation/IA, **not** a new permission. Anyone can still walk through the door and create events, exactly as today.
- Make «Заявки участников» a real management surface: the organizer sees their planned events, the applicant **count** per event, and the **named** list of applicants (accept/decline inline).

## Non-goals

- No "organizer mode toggle" that swaps the entire nav (considered, deferred).
- No verification gating on event creation (soft framing chosen — anyone authed can still create).
- **No database migration and no auth changes.** The only backend change is one additive read-model field (applicant name) on the existing applications endpoint — see "Backend change" below.
- No analytics, and no "Подписчики" (followers) functionality yet — it appears as a disabled "later" placeholder only.
- Applicant **email is not exposed** to organizers — name only (matches the established public-surface rule that event payloads carry organizer name but never email).

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Organizer gate | **Soft framing only** — no new restriction; UI reframing only |
| Structure | **Organizer hub page** (single `/organizer` dashboard), not a mode toggle |
| Hub contents | Create · My events · **Applications (events + counts + named applicants)** · Organizer profile (+ Followers placeholder) |
| Hub label / title | **«Организаторам»** (door label, nav label, and page heading) |
| Entry point (mobile) | **Option A** — the 5th tab "Создать" becomes "Организаторам" → hub |
| Entry point (desktop) | One **«Организаторам»** header button replacing the scattered links |
| Door visibility | Visible to **everyone** (signed in or not), consistent with soft framing |
| Home «Создать событие» button | **Removed** (the header door replaces it) |
| Applicant identity | **Name only** (additive backend field); never email |

## Design

### 1. New hub page — `/organizer`

A new route `app/organizer/page.tsx` titled **«Организаторам»**. It is a thin dashboard that links to pages that already exist — it does **not** reimplement them (the one exception is the applications section, detailed below).

Layout (top to bottom):

1. **Header row** — title «Организаторам» + subtitle «Создавайте и ведите свои события», with a primary **«+ Создать событие»** button (→ `/events/new`). (Note: this in-hub create button stays; only the *home-page* «Создать событие» button is removed.)
2. **Verification status strip** — surfaces the organizer profile's verification state (`draft` / `pending` / `verified` / `rejected`) with the matching label/colour and a link to «Профиль организатора». This makes visible the status that is currently buried inside `/me/organizer`. Source: the existing `getMyOrganizer()` call.
3. **Cards grid:**
   - **Мои события** → `/events/mine`. Shows draft / published counts when cheaply available.
   - **Заявки участников** — see the dedicated section below. Not a bare link: it's an aggregated view of the organizer's events with applicant counts and named applicants.
   - **Профиль организатора** → `/me/organizer`.
   - **Подписчики** — **disabled placeholder** card marked «(позже)». No link, no data.

### 1a. «Заявки участников» — applications management

This is the one substantive new surface (everything else is a link to an existing page). It answers the organizer's real question: *for each of my events, who is applying and how many?*

- **Source events:** the organizer's own events that use `signupMode === "application"` (the only mode with an approval queue). Fetched from the existing `/events/mine` data, filtered client-side to application-mode events.
- **Per event, show:** event title + date, an **applicant count** badge (e.g. «3 новых», derived from how many applications are still `applied`/pending vs total), and an expandable list.
- **Per applicant, show:** the applicant's **name** (new backend field — see below), their `applicationAnswer`, submission date, and status label. Pending (`applied`) applicants get inline **«Принять» / «Отклонить»** buttons.
- **Reuse:** the accept/decline action and per-applicant row rendering already exist in `EventApplicationsPanel`. This view aggregates that panel across all of the organizer's application-mode events rather than nesting it one-event-deep inside `/events/mine`. The decision call (`decideApplication`) is unchanged.
- **Empty states:** "no application-mode events yet" vs "events exist but no applications yet" are distinct messages.
- **Placement:** this can render inline within the hub (an expandable section) or as a sub-route `app/organizer/applications/page.tsx` linked from the card — the plan picks one; a sub-route is preferred if the hub page would otherwise grow too large.

This is the only part of the feature that reads applicant identity, and it is **organizer-gated by the existing endpoint** (`GET /events/{id}/applications` already returns 403/404 to non-owners).

**First-time / empty state:** a user who has never organized (no events, no organizer profile) sees a friendly intro that nudges the two starting actions — «Создать событие» and «Заполнить профиль организатора» — instead of empty count badges. The verification strip reflects the "not started / draft" state.

**Auth behaviour:** the hub renders for everyone. Actions that require auth (create, profile, applications) keep their **existing** login gating — an anonymous visitor who taps through is prompted to log in exactly as they are today on `/events/new`. The hub itself introduces no new gate.

### 2. The door — entry point (Option A)

- **Desktop header** (`components/AuthButton.tsx`): replace the standalone «Мои события» link with one **«Организаторам»** link/button pointing to `/organizer`. «Календарь», the email, and «Выйти» remain. (The `role === "admin"` «Админ» link is unaffected.)
- **Mobile tab bar** (`components/ui/TabBar.tsx`): the 5th tab — currently `{ href: "/events/new", label: "Создать", icon: <GlyphPlus /> }` — becomes `{ href: "/organizer", label: "Организаторам", icon: <GlyphOrganizer /> }`. The four attendee tabs (События · Подбор · Календарь · Карта) are unchanged. A new door/ID-card glyph replaces the plus glyph.
  - Note: `TabBar` currently hides itself on `/events/new`, event-detail, and `/admin/*`. The hub `/organizer` is a normal top-level screen and **should show** the tab bar (with "Организатор" active), so no change to the hide-logic is needed beyond confirming `/organizer` isn't matched by the existing conditions (it isn't).
- **Home page** (`app/page.tsx`): remove the standalone «Создать событие» tinted button (lines ~26–28). The header door now covers this entry; keeping both is redundant. No other home-page change.

### 3. Visibility & framing

The door is shown to all users regardless of auth state, matching the "soft framing" decision: becoming an organizer is an opt-in any visitor can take, not a privilege. No conditional rendering based on whether the user is "an organizer" is introduced (that was option C for the tab bar and was rejected in favour of the simpler A).

### What attendees keep (explicitly unchanged)

Discovery/home feed, «Подбор»/search, «Карта», `/me/calendar`, `/me/practices`, `/me/applications` (the attendee's own submitted applications — distinct from the organizer-side «Заявки участников»), event detail + RSVP/signup, and following organizers. None of these are removed, moved, or re-gated.

## Backend change (the only one — additive, no migration)

Goal: the applications response carries the applicant's **name** so the hub can list *who* applied, not just a UUID.

- **API model:** add an `applicant` object (`{ uuid, name }`) to the swagger `Rsvp` definition — additive, mirroring the `organizer` read-model already embedded on `Event`. Regenerate with **`make generate-api`** (swagger-only; do **not** use `make generate-all`).
- **Formatter:** `RsvpToAPI` (`backend/internal/http/formatter/rsvp.go`) maps the new `applicant` when the domain `Rsvp` carries it.
- **Enrichment:** the organizer-facing list path — `rsvp.ListApplications` (`backend/internal/rsvp/service.go` + `repository.go`) — populates each row's applicant name via a batched lookup keyed on `user_id`, mirroring `loadOrganizers` in `backend/internal/events/repository.go` (a JOIN to `users` in the list query is the simplest form). 
- **Scope guard:** enrich **only** the organizer endpoint (`GET /events/{id}/applications`). Do **not** add names to `MyApplications`/`MyPractices` (the user's own rows — no enrichment needed and no third-party identity there).
- **Compliance:** name only, **never email** (consistent with the documented public-surface rule). No new audit surface — listing applicants is a read the organizer already performs today; we are only adding a display name to it. No DB migration.

## Affected files

**Frontend**

| File | Change |
| --- | --- |
| `app/organizer/page.tsx` | **New** — the hub dashboard («Организаторам») |
| `app/organizer/applications/page.tsx` | **New (likely)** — aggregated applications view, if not inlined in the hub |
| `components/AuthButton.tsx` | Replace «Мои события» header link with «Организаторам» → `/organizer` |
| `components/ui/TabBar.tsx` | 5th tab → «Организаторам» (`/organizer`) + new `GlyphOrganizer` |
| `app/page.tsx` | Remove the standalone «Создать событие» button |
| `lib/types.ts` / `lib/api.ts` | Add `applicant?: { id; name }` to the `Rsvp` type + map it in `apiRsvpToLia` |
| `components/EventApplicationsPanel.tsx` | Render applicant name when present (reused by the aggregated view) |

**Backend**

| File | Change |
| --- | --- |
| swagger spec + `make generate-api` | Add `applicant` to the `Rsvp` model (regenerates `embedded_spec.go` + models) |
| `backend/internal/http/formatter/rsvp.go` | Map `applicant` in `RsvpToAPI` |
| `backend/internal/rsvp/service.go`, `repository.go` | Enrich `ListApplications` rows with applicant name (batched / JOIN) |

Reuse existing `getMyOrganizer()` / events APIs on the frontend — no new fetching primitives beyond the applications aggregation.

## Risks & edge cases

- **Двойная точка входа path** `/events/new` still exists and is still directly reachable (e.g. bookmarks, the hub's own button) — intended; the hub links to it rather than absorbing it.
- **Tab-bar active state** — `/organizer` must light up the "Организатор" tab; verify the `active = pathname === tab.href` check works for the new href.
- **Empty counts** — if fetching draft/published counts for the «Мои события» card is awkward, the card may link without counts in v1; counts are an enhancement, not a requirement.
- **Anonymous tab tap** — anon user taps «Организаторам» → hub renders → create/profile prompt login. Confirm this is no worse than today's anon `/events/new` experience.
- **Applicant-name backfill** — events/applications created before this change still resolve a name (the JOIN/lookup is by `user_id`, so existing rows get a name too; verify no null-name crash and a graceful fallback like «Участник» if a name is empty).
- **N+1 on the aggregated view** — listing applications across many events could fan out to one request per event. Acceptable at demo scale; the plan should note it and prefer the existing per-event endpoint over inventing an aggregate one, unless trivial.
- **Swagger regen discipline** — `make generate-api` only (the `make generate-all` proto step is known-broken per HANDOFF).

## Success criteria

- A signed-in user sees one clear «Организаторам» entry (header on desktop, tab on mobile) and lands on a hub that reaches Create, My events, Applications, and Organizer profile.
- The «Заявки участников» view lists the organizer's application-mode events, each with an applicant count and an expandable list of **named** applicants, with working accept/decline.
- The applications response includes an `applicant` name; **no email** is exposed; no DB migration was added.
- The attendee tab bar and all attendee routes behave exactly as before.
- The mobile tab bar no longer shows "Создать"; the home page no longer shows the standalone "Создать событие" button.
