# Organizer Hub — Separating the Organizer Experience

_Design spec · 2026-06-30 · frontend-only_

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

## Non-goals

- No "organizer mode toggle" that swaps the entire nav (considered, deferred).
- No verification gating on event creation (soft framing chosen — anyone authed can still create).
- **No backend, database, API, or auth changes.** Frontend only.
- No analytics, and no "Подписчики" (followers) functionality yet — it appears as a disabled "later" placeholder only.

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Organizer gate | **Soft framing only** — no new restriction; UI reframing only |
| Structure | **Organizer hub page** (single `/organizer` dashboard), not a mode toggle |
| Hub contents | Create · My events · Applications · Organizer profile (+ Followers placeholder) |
| Entry point (mobile) | **Option A** — the 5th tab "Создать" becomes "Организатор" → hub |
| Entry point (desktop) | One "🪪 Кабинет организатора" header button replacing the scattered links |
| Door visibility | Visible to **everyone** (signed in or not), consistent with soft framing |

## Design

### 1. New hub page — `/organizer`

A new route `app/organizer/page.tsx` titled **«Кабинет организатора»** ("Organizer office"). It is a thin dashboard that links to pages that already exist — it does **not** reimplement them.

Layout (top to bottom):

1. **Header row** — title «Кабинет организатора» + subtitle «Создавайте и ведите свои события», with a primary **«+ Создать событие»** button (→ `/events/new`).
2. **Verification status strip** — surfaces the organizer profile's verification state (`draft` / `pending` / `verified` / `rejected`) with the matching label/colour and a link to «Профиль организатора». This makes visible the status that is currently buried inside `/me/organizer`. Source: the existing `getMyOrganizer()` call.
3. **Cards grid:**
   - **Мои события** → `/events/mine`. Shows draft / published counts when cheaply available.
   - **Заявки участников** → the existing applications-management surface (the `EventApplicationsPanel` currently reached via `/events/mine`). Links to wherever that panel lives today; no new applications logic.
   - **Профиль организатора** → `/me/organizer`.
   - **Подписчики** — **disabled placeholder** card marked «(позже)». No link, no data.

**First-time / empty state:** a user who has never organized (no events, no organizer profile) sees a friendly intro that nudges the two starting actions — «Создать событие» and «Заполнить профиль организатора» — instead of empty count badges. The verification strip reflects the "not started / draft" state.

**Auth behaviour:** the hub renders for everyone. Actions that require auth (create, profile, applications) keep their **existing** login gating — an anonymous visitor who taps through is prompted to log in exactly as they are today on `/events/new`. The hub itself introduces no new gate.

### 2. The door — entry point (Option A)

- **Desktop header** (`components/AuthButton.tsx`): replace the standalone «Мои события» link with one **«🪪 Кабинет организатора»** link/button pointing to `/organizer`. «Календарь», the email, and «Выйти» remain. (The `role === "admin"` «Админ» link is unaffected.)
- **Mobile tab bar** (`components/ui/TabBar.tsx`): the 5th tab — currently `{ href: "/events/new", label: "Создать", icon: <GlyphPlus /> }` — becomes `{ href: "/organizer", label: "Организатор", icon: <GlyphOrganizer /> }`. The four attendee tabs (События · Подбор · Календарь · Карта) are unchanged. A new door/ID-card glyph replaces the plus glyph.
  - Note: `TabBar` currently hides itself on `/events/new`, event-detail, and `/admin/*`. The hub `/organizer` is a normal top-level screen and **should show** the tab bar (with "Организатор" active), so no change to the hide-logic is needed beyond confirming `/organizer` isn't matched by the existing conditions (it isn't).
- **Home page** (`app/page.tsx`): remove the standalone «Создать событие» tinted button (lines ~26–28). The header door now covers this entry; keeping both is redundant. No other home-page change.

### 3. Visibility & framing

The door is shown to all users regardless of auth state, matching the "soft framing" decision: becoming an organizer is an opt-in any visitor can take, not a privilege. No conditional rendering based on whether the user is "an organizer" is introduced (that was option C for the tab bar and was rejected in favour of the simpler A).

### What attendees keep (explicitly unchanged)

Discovery/home feed, «Подбор»/search, «Карта», `/me/calendar`, `/me/practices`, `/me/applications` (the attendee's own submitted applications — distinct from the organizer-side «Заявки участников»), event detail + RSVP/signup, and following organizers. None of these are removed, moved, or re-gated.

## Affected files (frontend only)

| File | Change |
| --- | --- |
| `app/organizer/page.tsx` | **New** — the hub dashboard |
| `components/AuthButton.tsx` | Replace «Мои события» header link with «Кабинет организатора» → `/organizer` |
| `components/ui/TabBar.tsx` | 5th tab → «Организатор» (`/organizer`) + new glyph |
| `app/page.tsx` | Remove the standalone «Создать событие» button |

Possible small additions: a hub-specific card/section component if `app/organizer/page.tsx` grows beyond a comfortable single file; a new `GlyphOrganizer` in `TabBar.tsx` alongside the existing glyphs. Reuse existing `getMyOrganizer()` / events APIs — no new data fetching primitives.

## Risks & edge cases

- **Двойная точка входа path** `/events/new` still exists and is still directly reachable (e.g. bookmarks, the hub's own button) — intended; the hub links to it rather than absorbing it.
- **Tab-bar active state** — `/organizer` must light up the "Организатор" tab; verify the `active = pathname === tab.href` check works for the new href.
- **Empty counts** — if fetching draft/published counts for the «Мои события» card is awkward, the card may link without counts in v1; counts are an enhancement, not a requirement.
- **Anonymous tab tap** — anon user taps «Организатор» → hub renders → create/profile prompt login. Confirm this is no worse than today's anon `/events/new` experience.

## Success criteria

- A signed-in user sees one clear «Кабинет организатора» entry (header on desktop, tab on mobile) and lands on a hub that reaches Create, My events, Applications, and Organizer profile.
- The attendee tab bar and all attendee routes behave exactly as before.
- No backend/API/migration changes; the diff is confined to the four frontend files above (plus any small new component).
- The mobile tab bar no longer shows "Создать"; the home page no longer shows the standalone "Создать событие" button.
