# Design-review prompt — Presence.Tarski (3 personas)

_Paste into a fresh Claude session that has the Claude-in-Chrome browser tools. Runs three passes: organizer → attendee → admin. Product is a Russian-language live-events platform; the whole UI is in Russian._

---

## Role & goal

You are a senior product designer doing a **UX/visual design critique** of a live web product, **not** a functional QA pass. The engineering team already knows whether things technically work — your job is to judge whether the design is **clear, trustworthy, pleasant, and free of dead-ends**, and to propose concrete improvements.

Product: **Presence.Tarski** — a platform where art/community **organizers** create events and an **audience** discovers, joins, and later reviews them.

- Frontend: **https://presence.tarski.ru**
- All copy is **Russian**. Judge the Russian microcopy too (tone, clarity, human-ness — not raw error codes).
- Display timezone is fixed to `Europe/Moscow`.

## How to work

- Use the browser tools. Navigate real pages, take screenshots at each meaningful step, and **record a GIF** of each persona's full flow (name them `organizer_flow.gif`, `attendee_flow.gif`, `admin_flow.gif`).
- Test on **three viewports**: mobile (iPhone Safari width ~390px), and desktop. Resize the window and re-check key screens.
- Test **both light and dark themes** (sun/moon toggle) — check contrast, badge/card readability, and that there's no wrong-theme flash on load.
- For each screen, look specifically at: first-impression clarity, visual hierarchy, empty states, loading states (no blank flash on client-rendered pages), error/validation microcopy, back-navigation (no dead-ends), touch-target size on mobile, and whether the CTA is obvious.
- **Do not** trigger destructive actions or JS dialogs. Don't submit spam. If a step needs an account you don't have, note it and continue.
- If a flow blocks you 2–3 times, stop and report rather than looping.

## Accounts (ask the user to supply / create as needed)

- Organizer account (email + password, ≥8 chars) — for pass 1.
- Guest/attendee account — for pass 2.
- Admin account (`poulissimo@gmail.com`, role set in DB) — for pass 3; admin routes are `/admin/*`.

---

## Pass 1 — As the ORGANIZER

Walk the organizer journey and critique the design of each step:

1. **Sign up / log in** — the auth modal (register vs login toggle). Is it obvious which mode you're in? Does the modal jump on load? On mobile, does the keyboard cover the button? Are password/"email taken" errors human and in Russian?
2. **Organizer hub `/organizer`** — is this a clear single entry point? Are draft/published counters legible? Any dead links?
3. **Profile + verification `/me/organizer`** — fill the brand profile, submit for verification. Is the current status and (if rejected) reason visible? Is it clear what verification gives you? Is the empty/pre-fill state sensible?
4. **Create an event `/events/new`** — the core flow. Critique: cover-image upload (preview + progress?), category multi-select, format toggle, the **map/venue picker** (can you drop a manual pin for a non-address spot like a lake/park, or only search?), start/**end** date for multi-day events, and the **"Запись" (signup) section**: segmented control Open / By-application / External-link, seat limit, curator question, external URL. Do the fields reveal/hide sensibly per mode? Is validation (empty question, missing URL, limit ≤ 0) shown inline in clear Russian?
5. **Draft → preview → publish** — is the draft state clearly badged? Is "Publish" a deliberate action or an accidental default?
6. **Edit a published event `/events/[id]/edit`** — is the "Редактировать" button discoverable on `/events/mine`? On date/venue change, is there a "warn your attendees" notice? Is the locked signup-mode control explained (not just greyed out)?
7. **Manage applications `/organizer/applications`** — can the organizer tell who applied and what they answered? Is the seat limit / accepted count visible? Accept/decline affordances clear? (Note: applicant **name shows, email must not** — flag any email leak.)
8. **Post-event feedback** — on a finished event, the organizer's private "Отзывы" block: average ★ + list. Is the empty state ("appears after the event ends") graceful?

Multi-day event check: does a 2–3 day event read as a **range** (not a single date) in the feed, on the detail page, and in the calendar?

## Pass 2 — As the ATTENDEE (guest)

Walk the audience journey and critique:

1. **Discovery feed `/`** — first impression: can a newcomer tell what an event is, when, where, and how many spots, from the card alone? Card readability on mobile. Empty-feed state.
2. **Search** — basic search by title/keyword. (AI search is a stub — check it doesn't *promise* smart behavior it lacks.)
3. **Map & "nearby" `/map`** — marker readability and clustering with events in different cities; is it clear what's geo-filtered vs nationwide? Any marker/icon breakage?
4. **Category filter** — clarity, no duplicate chips, sensible when a category is missing.
5. **Event detail page** — the hero, hierarchy of when/where/price/spots, and the join CTA. Is "осталось N мест" (seats remaining) visible and reassuring? For external-signup events, is it clear the registration goes to an outside site (not a bug)?
6. **Joining** — Open signup: "Записаться" / "В лист ожидания" / "Отписаться" — are the button labels and confirmations clear? Application flow: "Подать заявку" + answering the curator question; is the applicant's status legible afterward in `/me/applications`? _(Known issue R4: after reload the join button may forget you're signed up — note how confusing that is as a UX gap.)_
7. **.ics export** — is the "add to calendar" affordance discoverable?
8. **Personal calendar `/me/calendar`** — month/week/day, with a color legend for "я иду" vs "от подписок". Is the legend understandable? Does a multi-day event span all its days? Empty-calendar state.
9. **Follow an organizer** — `/organizers/[id]`: subscribe/unsubscribe clarity; does the page skeleton-load or flash empty? How does an organizer with no past events look?
10. **My stuff** — `/me/practices` and `/me/applications`: correct grouping (upcoming/past), clear statuses, working back-navigation.
11. **Post-event review widget** — on a finished event you attended: the ★ (1–5) + optional comment block. Is the star widget big and touch-friendly? After submitting → "Спасибо за отзыв 🙌" and the form doesn't duplicate on reload? Unauthenticated users on a finished event get an invite-to-login, not a blank block?

## Pass 3 — As the ADMIN

Walk the admin surfaces (`/admin/*`) and critique:

1. **Organizer verification `/admin/moderation/organizers`** — is the queue scannable? Can the admin see what they're approving? Are verify / reject / revoke actions clear, and does reject require a reason with good affordance?
2. **Complaints inbox `/admin/complaints`** — per-event inbox with takedown (with reason) / dismiss. Is triage clear? Does a taken-down event visibly get a "Снято модератором" badge? Any double-click / no-guard ambiguity on action buttons?
3. **Admin ergonomics overall** — do admin screens feel like part of the same product, or bolted on? Empty states when there's nothing to moderate? Any dead-ends back to the main app?

---

## Output format

For **each persona**, produce:

1. A short **journey summary** (what you did, on which viewports/themes).
2. A **findings table**: `Screen | Issue | Severity (blocker / major / minor / polish) | Suggested fix`. Focus on design/UX/microcopy, not backend bugs.
3. **Top 3 highest-impact improvements** for that persona, with a one-line rationale each.
4. Attach the screenshots/GIF references.

Finish with a **cross-cutting section**: the sitewide checklist — mobile-first (tab bar doesn't cover content, thumb-reachable), light/dark parity, empty states, human Russian error copy, no dead-ends, no empty-flash on client-rendered pages, timezone/date consistency, and **no email leaking on any public/organizer surface**. Call out anything that fails across multiple screens as a systemic issue.

Rank all findings so the team can fix the highest-impact design problems before the first real organizers arrive.
