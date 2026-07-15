# Presence.Tarski — UX / Visual Design Review

**Reviewer:** Senior product designer (design critique, not functional QA)
**Product:** presence.tarski.ru — Russian-language live-events platform (organizers ↔ audience)
**Date:** 14–15 July 2026 (Europe/Moscow) · **Display TZ verified:** Europe/Moscow
**Method:** Live walkthrough via Chrome automation. Fresh organizer + attendee accounts registered; admin = poulissimo@gmail.com. Desktop (1440px) tested; **light and dark themes** both checked.
**Recordings (saved to your Chrome downloads):** `organizer_flow.gif`, `attendee_flow.gif`, `admin_flow.gif`

> ⚠️ **Coverage limitation — mobile.** True mobile-width rendering could not be captured: the automation browser clamps its render viewport to a desktop minimum, so window resizes to ~390px did not re-flow the page. Mobile-specific findings below are therefore inferred, not visually confirmed. Re-running this review on a real device / device-emulation build is recommended for the mobile checklist. Everything else was verified live.

---

## Severity legend
**Blocker** = breaks or badly misleads a core task · **Major** = clear friction, confusion, or trust damage · **Minor** = noticeable rough edge · **Polish** = small refinement.

---

# PASS 1 — ORGANIZER

**Journey (desktop, dark + light):** Registered a fresh account → organizer hub `/organizer` → filled & submitted brand profile for verification `/me/organizer` → created a 3‑day "По заявке" event with cover, categories, map pin, seat limit and curator question `/events/new` → previewed → published → edited the published event `/events/[id]/edit` → managed the incoming application `/organizer/applications`.

## Findings

| Screen | Issue | Severity | Suggested fix |
|---|---|---|---|
| Event create `/events/new` | **Publish is the *default* status** ("Опубликовать" pre-selected over "Черновик"), while the header action is a neutral **"Сохранить"** — an organizer can publish accidentally thinking they only saved a draft. | **Major** | Default new events to **Черновик**; make publishing an explicit, differently-labelled action ("Опубликовать" button), or relabel the header to reflect the current status. |
| Events list `/events/mine` | The quick **"Опубликовать"** link fires a **native browser `confirm()` dialog** (blocking, unstyled) — inconsistent with the polished in-app modals used elsewhere (reject/takedown), and it froze the session. | **Major** | Replace with the same styled in-app confirmation modal used in admin. Never use native `confirm()/alert()`. |
| Detail / feed / calendar | **Multi-day events never read as a range.** A 15–17 Aug event shows only "15 августа в 18:00" on the detail page, the feed card, the "Мои события" card, **and** the calendar (pill on the start day only). | **Major** | Render a date range everywhere ("15–17 августа") and span multi-day events across all their days in the calendar. |
| Event detail (owner view) | Viewing your **own** event you see the attendee **"Подать заявку"** CTA, **no draft badge, and no Edit/Manage control** — owner view is identical to attendee view. | **Major** | Give the owner a distinct view: status badge, "Редактировать"/"Управление заявками" actions, hide the apply CTA. |
| Profile / verification `/me/organizer` | **No verification story on the page where you verify.** No explanation of *what verification gives you*, the "Отправить на проверку" button only appears *after* Save (2-step, unexplained), and no "what happens next / how long" after submitting. | **Major** | Add a short "why verify" line + a persistent status card ("На проверке — обычно до N дней"). Surface both steps upfront. |
| Cover / logo upload | **Raw native `<input type=file>`** showing English **"Choose File / No file chosen"** on both the event cover ("Обложка") and organizer logo ("Логотип") — jarring in an otherwise Russian, custom-styled UI, with no image preview or upload progress. | **Major** | Custom Russian upload control with drag-drop, thumbnail preview and progress. |
| Edit published event | **Venue field is empty on the edit form** ("Место" shows the placeholder, not the saved "Парк Горького"), even though dates and categories pre-fill — risk of wiping the venue on save. | **Major** | Pre-populate the venue (name + pin) when editing. |
| Organizer hub `/organizer` | Weak hierarchy: the four tiles are **plain text with no card/clickable affordance**; large empty dead-space below on desktop; the critical **"Профиль не отправлен"** status is low-contrast and **duplicated 3×** (status line, top-right link, and a card — two of them link to the same `/me/organizer`). "Подписчики" is a disabled "(позже)" placeholder shown as if live. | **Major** | Turn tiles into real cards with hover/borders; make the profile-status prominent and single-sourced; hide or badge ("Скоро") the disabled Подписчики tile. |
| Date/time inputs | **No timezone label** near Начало/Окончание despite the fixed Moscow display — an organizer typing a time can't tell it's Moscow time. | **Minor** | Append "(Мск)" to the time fields; keep native picker but label the zone. |
| New-organizer onboarding | After registration the modal just closes to the public feed (identical to a guest) — **no success toast, no "create your first event" nudge**. | **Minor** | Add a welcome/next-step for brand-new organizers. |
| Auth modal | The "Вход"/"Регистрация" surface opens as an **offset floating popover** (not a centered, dimmed modal) and **jumps/resizes** when toggling login↔register (register adds the "Имя" field). | **Minor** | Center + dim it like the (well-built) application modal; reserve stable height across modes. |
| Back navigation | `/me/organizer` back-link is **"‹ События"** (wrong destination — should return to the organizer hub); `/organizer/applications` correctly uses "← Организаторам". Different glyphs (‹ vs ←). | **Minor** | Point profile's back to the hub; standardise the arrow glyph. |
| Applications `/organizer/applications` | Shows applicant **name + answer + status**, clear Принять/Отклонить, and count "Заявок: N · ожидают: N" — but **not the seat limit** (e.g. "1 / 12 принято"). | **Minor** | Show accepted-vs-limit so the organizer sees capacity at a glance. |
| Event detail | For the created event, **no embedded map** in "Место" (seed events show one); the **УЧАСТНИКОВ stat is "—"** while the action bar says "Осталось мест: 12" (inconsistent); "Пожаловаться" link is very low-contrast. | **Minor** | Show the mini-map when a pin exists; reconcile the participants/seats numbers. |

### ✔ What works well (organizer)
The **map/venue picker** is genuinely good — free-text venue *plus* "Указать на карте" with Nominatim geocoding, disambiguation (Moscow vs Minsk vs Kazan) and drag-the-pin. The create form's **sectioning, human placeholders** ("Например, «Читаем Зебальда»"), **segmented controls**, and **conditional field reveal** (По заявке → curator question; Внешняя ссылка → URL) are all well done. **Draft is clearly badged** on `/events/mine`. On the edit form the **locked signup mode is explained** ("Режим записи зафиксирован после публикации") and a **"warn attendees" notice** appears on date change — both good (the warning does fire even with 0 attendees, so consider gating it to count > 0). Status badges (Черновик → На проверке) give good feedback. **No applicant email is exposed** anywhere on the applications surface (verified in the DOM).

### Top 3 organizer improvements
1. **Fix the publish safety model** — draft-by-default + a native `confirm()` on the other publish path means accidental or jarring publishing. This is the highest-trust-risk item for a brand-new organizer.
2. **Make multi-day events legible** — the range is core information and it's missing on every surface.
3. **Give verification a face on `/me/organizer`** — explain what it's for, set expectations, and surface both save/submit steps.

---

# PASS 2 — ATTENDEE

**Journey (desktop, dark + light):** Registered a guest → browsed feed `/` → search → map `/map` → opened event details → **applied** to an application-event (curator question) → **signed up** to an open event → checked `/me/applications`, `/me/practices`, `/me/calendar` → followed an organizer `/organizers/[id]` → exercised the post-event review widget.

## Findings

| Screen | Issue | Severity | Suggested fix |
|---|---|---|---|
| Event detail (all types) | **Reload forgets your status (known issue R4, confirmed for BOTH signup types).** After signing up ("Записаться"→"Вы записаны") or applying ("Подать заявку"→"Заявка отправлена"), a page reload reverts the button to the join CTA as if you never joined. Data persists (it shows in `/me/applications`), but the detail page doesn't reflect it — confusing, invites double-signup. | **Blocker (UX)** | On load, fetch the current user's relationship to the event and render the correct state (Вы записаны / Заявка отправлена / Отписаться). |
| Global theme | **Theme is unstable & inconsistent (systemic).** (a) The sun/moon toggle **desyncs** — the first click moves the knob but leaves `html.dark`; only a second click switches. (b) The choice **doesn't persist across page loads** — the *same* URL rendered dark, then light after reload. (c) **Different routes render different themes** in one session (feed dark; calendar/map/applications light). | **Major** | Single source of truth for theme, read on first paint (no flash), persisted; make the toggle reflect the actual applied theme. |
| Feed `/` | **Verified badge collides on cards.** "✓ Проверен" renders inline after the organizer on some cards, but as an **absolutely-positioned badge that overlaps the price/organizer text whenever the organizer name wraps to two lines** (e.g. "ГЭС‑2", "Новая Третьяковка"). Reproduces in both themes. | **Major** | One consistent badge treatment in normal flow; never absolute-position over card content. |
| Feed `/` | **Past events lead the feed.** Today is 14 July, yet the feed opens with 3/5/8/10/12 July events (past) in ascending order, with no past/upcoming distinction. | **Major** | Sort upcoming-first (or hide/section past events); a newcomer's first impression shouldn't be finished events. |
| Feed filters | The category chips expose only **3 of 8 categories** (Медиации, Мастер-классы, Лекции). **Концерты, Выставки, Спектакли, Кино, Фестивали have no filter chip** even though events use them. | **Major** | Expose all categories (overflow/"More" chip) or a filter sheet. |
| Map `/map` | **No markers until you click "Искать в этой области"** — the map opens empty, so a newcomer may conclude there are no events. Marker popup is title-only (no date/venue/price). | **Major** | Auto-load markers for the initial view; enrich the popup with date/place/price and a thumbnail. |
| Calendar `/me/calendar` | A **pending application is shown as confirmed attendance** (solid blue "Вы участвуете" pill). Also multi-day events appear only on the start day (see Pass 1 range issue). | **Major** | Distinguish pending vs confirmed (e.g. outline/te ntative style); span multi-day events. |
| Review widget | Participant gating is enforced **only on submit** — a non-participant can fill stars + comment and is then rejected with "Отзыв могут оставить только участники". | **Minor** | Disable/replace the form upfront for non-participants with an explanatory line. |
| Organizer page `/organizers/[id]` | **No organizer bio/description/logo** shown, even though the profile form collects them. | **Minor** | Render the organizer's description and logo on their public page. |
| `/me/practices` & terminology | The section is "**Мои практики**", but related concepts elsewhere are "события", "записи", "заявки", "мероприятия" — inconsistent vocabulary for the same ideas. | **Minor** | Consolidate terminology (pick one word per concept). |
| Empty states | Uniformly terse grey text ("Пока нет предстоящих записей.", empty calendar grid) with no illustration or next-step CTA. | **Minor/Polish** | Add a friendly line + a "Смотреть события" CTA. |
| Feed header | The floating rounded header lets cards **peek around its transparent side margins** on scroll. | **Polish** | Full-bleed backdrop or blur behind the sticky header. |

### ✔ What works well (attendee)
**Search** is excellent — instant filtering, a clear ✕, and an honest placeholder that makes no false "AI" promise. The **application modal** is a proper centered modal, echoes the curator question, and requires an answer. **`/me/applications`** is a highlight: status pill ("ожидает ответа"), status-filter tabs (В ожидании / Принятые / Отклонённые / Отозванные), the question+answer echoed, a Moscow-time timestamp, and a withdraw action. Open-signup labels ("Записаться" → "Вы записаны" / "Отписаться") and follow ("Подписаться" → "Вы подписаны") are clear. The **calendar legend** (🔵 Вы участвуете / 🟡 От подписок / 🔵🟡 И то, и другое) is clear and the "today" highlight correctly uses Moscow time. The **star review widget** is touch-friendly and the **"В календарь" (.ics)** affordance is discoverable.

### Top 3 attendee improvements
1. **Fix R4** — the detail page forgetting your join/apply status on reload is the single most confusing attendee bug and invites duplicate sign-ups.
2. **Stabilise theming** — the desync + no-persistence + per-route inconsistency makes the product feel broken; it's cosmetic but pervasive.
3. **Fix the feed's first impression** — lead with upcoming events and fix the colliding verified badge; these are the first two things a newcomer sees.

---

# PASS 3 — ADMIN

**Journey:** Logged in as admin → organizer verification queue `/admin/moderation/organizers` (**verified** the pending "Студия «Лиа»") → complaints inbox `/admin/complaints` (filed a test complaint, exercised **dismiss**; inspected the **takedown** modal) → event moderation `/admin/moderation/events` (published & taken-down tabs) → dashboard `/admin`.

## Findings

| Screen | Issue | Severity | Suggested fix |
|---|---|---|---|
| Verified organizers | **No revoke / un-verify action** on confirmed organizers — the "Подтверждённые" cards are static with no controls, so verification can't be pulled back from here. | **Major** | Add a "Снять верификацию" action (with reason) on confirmed orgs. |
| Complaints inbox | **Russian pluralization bug:** "**1 жалоб**" (should be "1 жалоба"; "2 жалобы"; "5 жалоб"). | **Minor (copy)** | Use plural rules for the count. |
| Complaints inbox | **Dismiss ("Отклонить") is one-click with no guard/confirmation**, while takedown requires a reason — an accidental click resolves a complaint. | **Minor** | Lightweight confirm or an undo toast for dismiss. |
| Admin theming | Admin is **light-theme only** (no toggle in the admin nav) — inconsistent with the themed public app. | **Minor** | Support the same theme, or intentionally document admin as light-only. |
| Empty states | Very terse ("**Пусто.**", "Жалоб нет.") — functional but abrupt. | **Minor/Polish** | Friendlier empty copy. |
| Event moderation | Dates show **seconds** and numeric format ("03.07.2026, 11:00:00"), differing from the public "3 июля в 11:00". | **Polish** | Drop seconds; align formatting. |
| Data hygiene | The "Снятые" tab is full of **dev test junk** ("Парольное событие", "repro published", reasons like "wefqwefqwef"). | **Note** | Clean seed/test data before real organizers arrive. |

### ✔ What works well (admin)
The admin **feels part of the same product** (same rounded nav, card system) — not bolted on. The **verification queue is scannable** (name, site, description) and actions are clear: verify is one click, **reject requires a reason** via a styled modal with the confirm button disabled until filled. The **complaints inbox** groups by event with reason-category counts, the event's current status, and the reporter's comment; **takedown requires a reason** ("Снять и закрыть жалобы" is a nicely explicit label) and is **reversible** ("Вернуть" restores taken-down events, with the reason retained). The **dashboard** (KPIs + quick actions + "‹ События" back-to-app) gives good at-a-glance triage with **no dead-ends**.

> Not verified: the public **"Снято модератором"** badge — confirming it would have required executing a real takedown on a shared production event, which I intentionally did not do. The takedown *affordance* (required reason, reversible) is sound; the resulting public badge should be spot-checked by the team.

### Top 3 admin improvements
1. **Add revoke/un-verify** for confirmed organizers — the moderation lifecycle is otherwise one-directional.
2. **Guard the dismiss action** and fix "1 жалоб" — small changes that prevent misfires and read as unpolished.
3. **Purge dev test data** from the moderation tables before onboarding real organizers.

---

# CROSS-CUTTING — sitewide checklist

| Check | Verdict | Notes |
|---|---|---|
| **No email leaking on public/organizer surfaces** | ✅ **Pass** | Applications show applicant name only; no email anywhere in the DOM. |
| **Timezone / date consistency** | ✅ Mostly | App consistently uses Europe/Moscow (today = 15th past midnight MSK; timestamps in MSK). *But* multi-day never renders as a range, and admin dates differ in format (seconds, numeric). |
| **Light / dark parity** | ⚠️ **Systemic issue** | Light theme itself renders well, but the theme system is unstable: toggle desync, no persistence across loads, and **per-route theme inconsistency** in one session. |
| **No dead-ends / back-navigation** | ✅ Mostly | Admin and most pages link back to the app. One wrong target: `/me/organizer` back-link goes to the feed, not the organizer hub. |
| **No empty-flash on client-rendered pages** | ⚠️ | Plain "Загрузка…" text (no skeletons); combined with the theme flip, some routes visibly change appearance after first paint. |
| **Human Russian error/microcopy** | ✅ Mostly | Copy is generally warm and human ("Оставьте пустым — без ограничения", the curator-question hint). Defects: "**1 жалоб**" (plural), English "Choose File / No file chosen" on uploads, and terminology sprawl (события/практики/записи/заявки/мероприятия). |
| **Native controls vs. custom UI** | ⚠️ **Systemic** | Native file inputs (English), native datetime inputs (no TZ label), and a native `confirm()` on publish all break the otherwise-polished custom Russian UI. |
| **Mobile-first (tab bar, thumb-reach, touch targets)** | ❓ **Not verified** | Could not render true mobile width (tooling limitation). Re-test on device. |

## Systemic issues (fix once, fix everywhere)
1. **Theme system** — desync, no persistence, per-route inconsistency. Pervasive "feels broken" signal.
2. **Multi-day events** — never shown as a range (feed, detail, list, calendar).
3. **Detail-page status amnesia (R4)** — join/apply state forgotten on reload for both signup types.
4. **Verified-badge collision** — absolute-positioned badge overlaps card text when the organizer name wraps; both themes.
5. **Raw native controls** — file inputs / datetime / confirm() undermine the custom UI and Russian-only experience.
6. **Terminology sprawl** — one word per concept, please.

---

# Master ranked list (fix highest-impact first, before real organizers arrive)

1. **R4 — detail page forgets join/apply status on reload** (attendee) — *Blocker (UX)*, systemic.
2. **Theme instability** (toggle desync + no persistence + per-route flip) — *Major*, systemic, "looks broken".
3. **Multi-day events never show as a range** (feed/detail/list/calendar) — *Major*, systemic, loses core info.
4. **Publish-by-default + neutral "Сохранить"** → accidental publishing — *Major* (trust/safety).
5. **Native `confirm()` on the quick Publish** (blocking, unstyled, froze session) — *Major*, consistency.
6. **Verified badge collides with card text** when organizer name wraps — *Major*, systemic visual.
7. **Feed leads with past events** — *Major*, first-impression.
8. **Owner sees attendee view** on own event (no badge/edit/manage) — *Major*.
9. **Verification is unexplained** on `/me/organizer` (no "why/what next", hidden 2nd step) — *Major*.
10. **Native file inputs in English, no preview/progress** (cover + logo) — *Major*, consistency.
11. **Category filter exposes only 3 of 8 categories** — *Major*.
12. **Map has no markers until "Искать в этой области"** — *Major*.
13. **No revoke/un-verify for confirmed organizers** (admin) — *Major*.
14. **Venue not pre-filled on the edit form** (wipe risk) — *Major*.
15. **Pending application shown as confirmed on the calendar** — *Major*.
16. **Organizer hub hierarchy weak** (text-only tiles, dead space, duplicated status) — *Major*.
17. **No timezone label** on date/time inputs — *Minor*.
18. **Review gating only on submit** — *Minor*.
19. **Organizer bio/logo not shown** on public organizer page — *Minor*.
20. **"1 жалоб" plural bug** + terminology sprawl + terse empty states + dismiss has no guard + admin date seconds + auth-modal jump + `/me/organizer` back-link target — *Minor/Polish*.

---

*Not fully exercised (noted for completeness):* true mobile rendering (tooling); external-signup attendee clarity (no external-link event available); the post-event organizer "Отзывы" success state and the public "Снято модератором" badge (would have required destructive/finished-event actions I avoided on shared data).
