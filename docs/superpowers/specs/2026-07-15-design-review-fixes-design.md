# Presence.Tarski — Design-Review Fix Plan (Design Spec)

**Date:** 2026-07-15 · **Author:** design-review remediation pass
**Sources:** `docs/qa/design_pass/Presence_Tarski_design_review.md` (comprehensive markdown review) + the live HTML pass (artifact `e62b24c4`, saved intent under `docs/qa/design_pass/`).
**Site:** presence.tarski.ru (frontend `frontend/`, Next 16 + Tailwind 4; backend Go monolith `backend/`).

## Decisions locked in brainstorming

- **Scope:** full-stack. Backend changes allowed (needed for R4 and revoke-409 semantics).
- **Sequencing:** phased P0 → P1 → P2. Each phase is independently shippable to prod.
- **Publish safety:** draft-by-default **and** a styled in-app confirmation modal (belt-and-suspenders).
- Branch off current `feat/organizer-suite-r1-r2-r3` or a fresh `fix/design-review-p0`.

## Reconciliation with live code (what's already fixed — do NOT redo)

The two reviews captured different code states. Verified against current `frontend/`:

- **Theme instability** (markdown #2 systemic "Major") — **already fixed.** `app/layout.tsx` runs a pre-paint script + persists to `localStorage`; `components/ui/ThemeSwitch.tsx` reads the `<html>` class via `useSyncExternalStore` as the single source of truth. Spot-check only; no code change planned.
- **Verified-badge collision** (markdown "Major", absolute-positioned overlap) — **fixed.** `components/VerifiedBadge.tsx` is now `inline-flex`, rendered in normal flow in `EventCard`. Only a *Low* cosmetic "wraps to its own line on long names" remains → item #21.
- **Map auto-load markers** (markdown "Major") — **already auto-loads** on mount in `components/MapBrowse.tsx` (geolocation → `load`). Remaining real map issue is the bright light tiles + thin popup → item #14.
- **Admin revoke exists** — markdown said "no revoke action"; code confirms it exists in `app/admin/organizers/page.tsx`. The real defect is the false-error / missing in-flight guard → item #2.

## Fix list — deduped, code-verified, phased

Severity tags: **[B]** blocker · **[Maj]** major · **[Min]** minor/polish. `(fe)` frontend-only, `(be)` needs backend.

### P0 — Blockers, confirmed bugs & systemic

**1. R4 — detail page forgets join/apply status on reload [B] (fe+be)**
- Backend: `GET /events/{id}` must populate `my_rsvp_status` for the authenticated caller (currently unset — see the "we do NOT trust event.myRsvpStatus" comment in `components/SignupCTA.tsx:95`).
- Frontend: remove the workaround; initialize `localStatus` from `event.myRsvpStatus`. Covers both open signup (Записаться → Вы записаны/Отписаться) and application (Подать заявку → Заявка отправлена).
- Done when: after signup/apply, a hard reload of `/events/{id}` renders the correct joined/applied state.

**2. Admin action buttons: false-error + no in-flight guard [Maj/bug] (fe)**
- `app/admin/organizers/page.tsx` `onRevoke`/`onToggleAuto` (and the same shape on reject / takedown / dismiss): disable the button while the request is in flight; on success refetch status; treat `409 "already in that state"` as success (not a red error).
- Apply consistently across all admin action handlers (revoke, reject, takedown, dismiss, auto-verify toggle).
- Done when: a double-click revoke shows the committed `rejected` state, no red error.

**3. Publish safety [Maj/trust] (fe)**
- `/events/new` (`components/CreateEventForm.tsx`): default status = **Черновик**, not Опубликовать.
- `components/PublishEventButton.tsx`: replace `window.confirm(...)` with the styled in-app confirmation modal (same visual language as the admin reject/takedown modal).
- Done when: new events save as draft by default; quick-publish shows a styled RU modal, never a native dialog.

**4. Multi-day events never read as a range [Maj/systemic] (fe)**
- Add a range formatter in `lib/format.ts` (uses existing `endsAt`) → "15–17 августа". Apply on: feed card (`EventCard`), event detail (`EventDetailView`), `/events/mine` card, and span the pill across all days in the calendar (`lib/calendar.ts` + calendar view).
- Done when: a 15–17 Aug event shows the range on all four surfaces.

**5. Native file inputs (English "Choose File / No file chosen") [Maj/systemic] (fe)**
- One styled RU upload component (label «Загрузить обложку / логотип», thumbnail preview + progress). Use on organizer logo (`/me/organizer`) and event cover (`/events/new` + `/events/[id]/edit`).
- Done when: no raw `<input type=file>` visible; consistent styled control across all three screens.

**6. Native datetime inputs [Maj/systemic] (fe)**
- Styled RU date-time control for Начало/Окончание with an explicit **(МСК)** timezone label. Replaces raw `datetime-local` (English `dd/mm/yyyy` + keyboard-entry friction observed live).
- Done when: themed picker, MSK labeled, keyboard entry works.

### P1 — Major friction / trust / first-impression

**7. Owner sees attendee view on own event [Maj] (fe)** — On `/events/{id}` for the owner: show status badge (Черновик/Опубликовано), Редактировать + Управление заявками actions, hide the apply CTA. (`EventDetailView` / `SignupCTA` gating on `event.isOwner`/current user.)

**8. Feed leads with past events [Maj] (fe)** — `DiscoveryFeed`: sort upcoming-first (or section past events); a newcomer's first impression shouldn't be finished events.

**9. Category filters expose only 3 of 8 [Maj] (fe)** — Expose all 8 categories (overflow / «ещё» chip or filter sheet). `FILTERS` in `DiscoveryFeed`.

**10. Venue empty on edit form [Maj/data-loss] (fe)** — Pre-fill Место (name + pin) when editing a published event (`/events/[id]/edit`). Avoids wiping venue on save.

**11. Calendar shows pending application as confirmed [Maj] (fe)** — Distinguish tentative (pending application) vs confirmed attendance (outline vs solid pill). `lib/calendar.ts` + calendar view.

**12. Verification unexplained on `/me/organizer` [Maj] (fe)** — Add a one-line "why verify" explainer + a persistent status card ("На проверке — обычно до N дней") + surface the submit-for-verification step inline (not hidden behind Save).

**13. Organizer hub `/organizer` weak hierarchy [Maj] (fe)** — Real card affordance (borders/hover/icons); single-source the profile-status (currently duplicated); badge the disabled «Подписчики» tile as **«Скоро»** (decided: badge, not hide). Give the empty hub a first-run 1→2 path (профиль → первое событие).

**14. Map bright tiles in dark UI [Maj] (fe)** — Dark tile theme (e.g. Carto dark) in `components/map/LeafletMap.tsx`; enrich marker popup with date/venue/price (currently title-only).

**15. Public organizer page thin [Min→Maj] (fe)** — Surface org bio/description + logo on `/organizers/[id]` (data is collected in the profile form). Logo resolution may be deferred if blocked on storage.

### P2 — Minor / polish / copy

**16. `«1 жалоб»` plural bug [Min] (fe)** — Russian plural rules for the complaint count (`app/admin/complaints/page.tsx:101`): 1 жалоба / 2 жалобы / 5 жалоб. Add a small `pluralRu` helper.

**17. Dismiss guard on complaints [Min] (fe)** — Lightweight confirm or undo toast on Отклонить (currently one-click, while takedown requires a reason).

**18. Review widget gates only on submit [Min] (fe)** — Disable/replace the star+comment form upfront for non-participants with an explanatory line (`components/FeedbackForm.tsx` / `OrganizerFeedback.tsx`).

**19. Terminology consolidation [Min/copy] (fe)** — One word per concept. Decided default: concept = «события»; keep «записи»/«заявки» for the two RSVP types; rename **«Мои практики» → «Мои записи»** (`/me/practices`). Confirm final vocabulary during implementation.

**20. Grab-bag polish [Min] (fe)**:
- Verified badge wrap tweak on long organizer names (`nowrap`/flex).
- `/me/organizer` back-link → organizer hub (currently «‹ События»); standardize arrow glyph (← vs ‹).
- Auth modal: center + dim (currently offset popover); reserve stable height across login↔register.
- Empty states: friendly line + «Смотреть события» CTA (feed/calendar/applications/admin).
- Admin event-moderation dates: drop seconds, align to public format ("3 июля в 11:00").
- Applications panel: show accepted-vs-limit ("1 / 12 принято").
- Purge dev/test junk from moderation tables before real orgs arrive (data hygiene, not code).

### Deferred / out of scope for these fixes
- True mobile-width re-test (automation tooling limitation) — re-run on a real device.
- Theme system hardening — verified already fixed; spot-check only.

## Testing approach

- **P0 regressions** get explicit verification: R4 reload state, revoke double-click, publish-default draft, multi-day range on all 4 surfaces. Where a component test harness exists (`vitest`, see `components/__tests__/`), add unit tests for the range formatter and the RU pluralizer (pure functions — cheap, high-value).
- Each phase ends with a live smoke pass on presence.tarski.ru (or local) driving the affected flow before merge, per the repo's verify practice.
- Backend R4 change: unit-test that `GET /events/{id}` returns `my_rsvp_status` for the caller and empty for anonymous.

## Rollout

Three shippable increments (P0 → P1 → P2). P0 is the pre-pilot gate: blockers + confirmed bugs + systemic native-control cleanup. P1/P2 can follow after the first orgs arrive without blocking onboarding.
