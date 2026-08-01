# QA — Full-site sweep across all roles (post Swiss Grid redesign)

**Date:** 2026-07-30 → 2026-07-31
**Target:** production — https://presence.tarski.ru (API https://api.presence.tarski.ru)
**Build under test:** frontend container recreated 2026-07-30 (`lia-frontend-presence:latest`); backend `backend-app` unchanged
**Repo state:** `main` == `origin/main` == `464fd22`, working tree clean
**Method:** live browser (Claude in Chrome) at 1440×900 / 1512×795, plus direct API probes and prod DB reads; root causes traced into the codebase where possible
**Status:** Complete for anonymous / user / organizer / admin. Four items unverified — see [Not covered](#not-covered).

---

## 1. Verdict

The redesign is in good shape. Swiss Grid is applied consistently across the public, organizer and admin surfaces, and **every core flow exercised end-to-end worked**: signup/login/logout, application submit, RSVP + `.ics`, event creation through the 4-step stepper, publish, and the full admin suite.

**API role gating is airtight** — verified on all 8 admin endpoints: admin `200`, common user `403`, anonymous `401`.

Thirteen defects found. None block the product, but **#1–#4 are user-visible or policy-level and should be fixed before any wider launch.**

---

## 2. Test accounts created

Three accounts were created on production (authorised by the user for this pass):

| Email | Password | Role | How the role was set |
|---|---|---|---|
| `qa.user.30jul@tarski.ru` | `QaPresence2026!` | common | — |
| `qa.organizer.30jul@tarski.ru` | `QaPresence2026!` | common (+organizer profile) | — |
| `qa.admin.30jul@tarski.ru` | `QaPresence2026!` | **admin** | `UPDATE users SET role='admin'` in the **`gateguard`** DB |

All three had `email_verified` set to `true` in `gateguard.users`.

> **Note for future QA:** the role is read from the **GateGuard** claim on every request and synced *into* `lia_prod.users` (`internal/http/auth/auth.go` — `ensureUser` overwrites a drifted role). Promoting a user by editing `lia_prod.users.role` therefore does nothing; it must be done in `gateguard`.

---

## 3. What works

| Area | Verified |
|---|---|
| **U1 feed** | Grid, covers, category numerals, time/category chips, text search, count pluralization («1 СОБЫТИЕ»), U8 empty state, «Сбросить фильтры» |
| **U2 detail** | Cover strip, fact cells, venue map (grayscale confirmed), organizer + «✓ Проверен» chip, «Пожаловаться» |
| **U3 Подбор** | Suggestion chips, free-text query, answer strip, `Совпало: …` reason, mandatory escape hatch, skeleton loading |
| **U4 calendar** | Month grid, today marker, agenda rail, empty-day copy |
| **U5 map** | Grayscale tiles, **square ink numbered markers matching list numerals**, stat strip, «Искать в этой области» |
| **U6 profile** | Identity strip, tab chips with counts, application row, empty states |
| **U7 auth** | Split-panel `/login`, login, logout (clears `lia.auth.token`), verification banner correctly absent for verified users |
| **RSVP / applications** | Application modal with curator question → «Заявка отправлена» + «Отозвать заявку»; open RSVP `201 going`; `calendar.ics` `200` |
| **O1–O5** | Organizer hub, 4-step create stepper with live feed preview, venue autocomplete, `/events/mine` table with seat-fill bars, publish confirm, invite-by-email panel, organizer profile |
| **A1–A4** | Ink-inverted admin surface, overview tiles, moderation queue + reason chips, organizer registry, user registry + content-hygiene rail |
| **Auth gating (API)** | `/admin/{overview,moderation/events,moderation/organizers,organizers,users,hygiene,settings,complaints}` → admin `200` / user `403` / anon `401` |
| **User endpoints** | `/me/{follows,calendar,invitations,practices}`, `/geocode`, `/auth/me` all `200` |

---

## 4. Defects

Ordered by severity.

### P1 — user-visible or policy

#### 4.1 «МЕСТА» is dead on every event detail page
**Confidence:** high — root cause in code.

`Cell caption="Места"` renders `attendanceShort(event)` (`components/EventDetailView.tsx:86`), which returns `"—"` whenever `attendeeCount == null` (`lib/format.ts:114`). The API adapter never populates `attendeeCount` — `lib/api.ts:72` maps only `seatsRemaining`.

Result: the capacity cell reads `—` on **every** event, while the sidebar immediately beside it says «Осталось мест: 11». Reproduced on a seeded event (capacity 12) and on a freshly created one (capacity 20). The organizer table on `/events/mine` computes `0 / 20` correctly from the same data, so this is an inconsistency inside one app, not missing data.

**Fix:** derive `attendeeCount = capacity − seats_remaining` in the adapter, or render `seats_remaining / capacity` directly.

#### 4.2 The feed leads with events that already happened
**Confidence:** high — root cause in code.

`fetchPublishedEvents()` (`lib/api.ts:163`) builds `?status=published` with no `from`. `from`/`to` are supported by the backend but are only passed by the today/weekend chips.

On 30.07 the home page showed 8 of 9 events in the past (15.07, 18.07, 20.07, 25.07, 26.07 …), sorted so that past events dominate the first screen.

**Fix:** default `from = now` for the unfiltered feed, or add explicit past/upcoming treatment.

#### 4.3 Moderation is effectively disabled in production
**Confidence:** high — observed live.

- The create form promises «Модерация занимает до 24 часов. Событие появится в ленте после одобрения.»
- Publishing a draft put it on the **public feed immediately** (`GET /events` count went 9 → 10, status `published`, `МОДЕРАЦИЯ · 0`).
- `/admin/settings` → **«Авто-подтверждение всех организаторов» is enabled**, auto-approving every organizer verification request and bypassing the moderation queue.

May be intentional for the demo, but the UI actively promises the opposite. Needs an explicit decision, and the copy should match whichever way it goes.

#### 4.4 `YANDEX_PLACES_KEY` still unprovisioned — venue search silently degraded
**Confidence:** high — reproduced.

`GET /api/v1/places?q=Гараж` → **`503 {"error":"places_failed"}`**.

The venue autocomplete still returns results, because it falls back to the local `venues` table — so the failure is invisible in the UI and easy to mistake for working. This is the open follow-up from the 2026-07-23 QA-20-jul deploy: the var is declared in compose but absent from `.env.prod`.

**Fix:** add `YANDEX_PLACES_KEY` to `/opt/lia/backend/.env.prod`, then `docker compose … up -d --no-build app`.

### P2 — correctness / spec

#### 4.5 Publish dialog copy is wrong
«После публикации изменить его будет нельзя» — but «Ред.» remains available after publishing, and `PATCH /events/{id}` with `{"status":"draft"}` on a published event returned `200`. Edit-published shipped in R2; the warning predates it.

#### 4.6 `/admin/complaints` and `/admin/settings` were missed by the redesign
Both render as plain unstyled text with no Swiss Grid treatment, no U8 empty-state pattern (`/admin/complaints` is just «Жалобы» / «Жалоб нет.»), and neither appears in the admin nav. `/admin/settings` uses a **native blue checkbox**, violating the token-sheet rule (colour outside the sheet; spec calls for a square ink control).

#### 4.7 `/me/applications` payload gap (backend)
The embedded `event` object carries only `venue_id` / `organizer_id` and an empty `categories: []` — no `venue`, no `organizer`. The U6 application row therefore renders `—` for both columns. Frontend is correct; the API needs the same enrichment `GET /events` already does.

#### 4.8 Numeric cells that can never populate (backend gaps)
`GET /api/v1/admin/overview` returns only `{complaints_open, events_published, events_removed, events_total, organizers_pending}` — no organizer or user totals, so **«ОРГАНИЗАТОРОВ —» and «ПОЛЬЗОВАТЕЛЕЙ —»** are permanently blank despite 28 users in the DB.

Same class elsewhere: `СОБЫТИЙ` / `ЖАЛОБ` per row on `/admin/organizers`, `ПОДПИСЧИКОВ` on `/me/organizer`, `ВСЕГО ЗАПИСЕЙ` on the organizer hub, `РАДИУС` on `/map`.

#### 4.9 Colliding admin short IDs
`lib/admin-id.ts` builds `EV-` + first 4 hex chars of the UUID. All seeded events are `b0000000-…`, so **seven different rows in the moderation queue all display `EV-B000`**; several users display `C000`. The unit test passes because it uses a random-looking UUID. IDs are unusable for identifying a row against the live dataset.

#### 4.10 Russian copy defects
- «**1 события** с тестовыми данными» on the admin overview — should be «1 событие» (no plural rule applied).
- «ПОДАНО **05.07 НАЗАД**» on the moderation detail — «назад» appended to an absolute date.
- The moderation list mixes absolute (`05.07`) and relative (`5 Д`) date formats in one column.

#### 4.11 Feedback form shown to non-attendees
The «Как прошла встреча?» form renders for any signed-in user on a past event, including one who never attended. The backend correctly rejects with `403 ErrNotParticipant` (`internal/feedback/service.go:85`), so this is a frontend gating issue — the form simply shouldn't render without an active RSVP.

### P3 — cosmetic

#### 4.12 Blank cover band on the detail page
An event with no cover renders a ~430px empty paper block on `/events/[id]`. The feed has a numeral-plate fallback for exactly this case; the detail page has none.

#### 4.13 Venue map marker not to spec
The `/events/[id]` map uses Yandex's default teardrop pin with Yandex's own controls visible («Как добраться», «Доехать на такси», «Создать свою карту»). The `/map` browse view does this correctly (square ink numbered markers) — the detail view wasn't brought along. *Map grayscale itself is correct and was confirmed via computed style.*

#### 4.14 Low confidence — admin dashboard first-load failure
`/admin` showed «Не удалось загрузить обзор» on first load and rendered correctly after «Повторить»; a subsequent full reload was clean, with `/admin/overview` returning `200`. The token had been injected via JS immediately before a client-side navigation, so **this may be an artefact of the test harness rather than a real race.** Worth one confirmation via a normal login → «Админ» click before filing.

---

## 5. Production data hygiene — Phase 2.5 purge never ran

The redesign master plan folded a QA-data purge into Phase 2.5. It has not happened. Live on production now:

- **Events:** `twix-mars` (junk title, visible on the public feed)
- **Users:** `QA Block8` (`qa-block8-714@presence.test`), `Мастерская «Таран»` (`fzamfhir@sharklasers.com` — disposable mail)
- **Organizers:** `QA Revoke-Fix Test Org` (`https://qa.test`), `Smoke Verification Org` (`https://smoke.test`)
- **Venues:** the `venues` table holds **St. Petersburg** locations — `LOFT 812`, `SMART-coworking на Коломяжском`, `SMART-coworking на Шпалерной`, `SOK Земляной Вал` — in a product whose feed header reads «МОСКВА»
- Venue records also surface as rows in the user registry with `—` for email

The admin content-hygiene rail already detects part of this, which is a good sign for the feature. `pg_dump` first; unpublish rather than delete, per the original plan.

---

## 6. Traces / evidence

```
# Role gating
admin  → /api/v1/admin/{overview,moderation/events,moderation/organizers,
                        organizers,users,hygiene,settings,complaints}   200 ×8
common → same                                                          403
anon   → /api/v1/admin/overview                                        401

# Places
GET /api/v1/places?q=Гараж        503 {"error":"places_failed"}

# Admin overview payload (no organizer/user totals)
{"complaints_open":0,"events_published":10,"events_removed":5,
 "events_total":35,"organizers_pending":0}

# Publish went straight public, no moderation
GET /api/v1/events   9 → 10 published, "QA проверка создания события 30.07"

# Edit-after-publish works (contradicting the dialog)
PATCH /api/v1/events/{id} {"status":"draft"}   200

# Feed dates on 30.07 — 8 of 9 in the past
2026-07-05, 07-10, 07-12, 07-15, 07-18, 07-20, 07-25, 07-25 | 2026-08-15
```

Prod DB at time of test: `lia_prod` — 2 admin / 28 common users; 9 published / 20 draft / 5 rejected events.

---

## 7. Cleanup performed / outstanding

**Done:** the test event created during this pass was reverted to `draft` — the public feed is back to 9 published events, and nothing QA-related is publicly visible.

**Left on production** (harmless, removable on request):

- 3 QA accounts (§2), one of them with `admin` role in `gateguard`
- 1 application (`qa.user` → «Лаборатория медиаций»)
- 1 RSVP (`qa.user` → the QA draft event)
- 1 draft event «QA проверка создания события 30.07» owned by `qa.organizer`

---

## 8. Not covered

The browser extension disconnected near the end of the session. These remain unverified:

1. **Mobile at 390px** — no responsive pass was completed on any surface
2. **404 page render** — `app/not-found.tsx` exists and the route returns HTTP `404`, but the inverted-ink U8 treatment was not seen
3. **Complaint submission** — the «Пожаловаться» control was seen but never exercised
4. **Invitation accept flow** — `/invite/[token]` and `/me/invitations` (endpoint returns `200`, UI unexercised)

---

## 9. Recommended order of work

1. `YANDEX_PLACES_KEY` into `.env.prod` (§4.4) — one-line config, currently failing silently
2. Decide the moderation policy and align the copy (§4.3, §4.5)
3. `attendeeCount` in the API adapter (§4.1) — one-line fix, visible on every event page
4. `from = now` on the default feed (§4.2)
5. Prod data purge (§5)
6. Redesign `/admin/complaints` + `/admin/settings` (§4.6)
7. Backend enrichment + counts (§4.7, §4.8)
8. Copy and short-ID fixes (§4.9, §4.10), feedback gating (§4.11), cosmetics (§4.12, §4.13)
9. Finish the untested surfaces (§8)
