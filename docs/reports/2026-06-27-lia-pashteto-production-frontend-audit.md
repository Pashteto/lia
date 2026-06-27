# Presence.Tarski — Production Frontend Audit

**Date:** 2026-06-27  
**Environment:** https://lia.pashteto.com (API: https://api.lia.pashteto.com)  
**Method:** Live browser walkthrough + API verification against codebase  
**Test account:** `pashteto@inbox.ru` (role: `common`, display name: paapa)  
**Sessions:** Initial walkthrough over VPN; latency re-verified **without VPN** same day (see [Latency retest](#latency-retest-2026-06-27-no-vpn))

---

## Executive summary

The live demo is a **working event-discovery MVP** backed by a real Go API and PostGIS database — not the frontend mock fallback. Core flows (browse → detail → RSVP → calendar) work end-to-end. The product is clearly in **demo/MVP stage**: several filters are cosmetic and AI search is a stub. Auth via the UI hit a transient **503** once; the API itself responded normally on retry.

**Latency note:** Calendar/practices felt slow (**5–8s «Загрузка…»**) during the initial VPN session. Re-test **without VPN** showed API responses in **~25–165ms** and UI ready in **<100ms** — the slowness was **network-path (VPN) dominated**, not backend or React Query. Demo from a direct route when possible.

**Overall maturity:** ~70% of planned user-facing features are live and usable.

---

## Infrastructure

| Component | Status | Notes |
|-----------|--------|-------|
| Frontend (Next.js 16, SSR) | ✅ | `lia.pashteto.com` — HTTP 200, prerender cache ~30s |
| Backend API | ✅ | `api.lia.pashteto.com` — events, auth, RSVP, calendar; **~26–106ms** without VPN |
| Database | ✅ | 9 seeded Moscow events; prod at migration 018 |
| Auth (GateGuard + password) | ⚠️ | Login API 200; UI login had one 503 (transient) |
| File storage | ✅ | Cover images via `GET /api/v1/files/{key}` |

---

## Feature matrix

| Feature | Route | Status | Notes |
|---------|-------|--------|-------|
| Discovery feed | `/` | ✅ | 9 live seed events, category filters |
| Event detail | `/events/[id]` | ✅ | Cover, facts, map, RSVP, ICS, report |
| Auth (login/register) | modal | ⚠️ | Works; intermittent 503 on UI |
| Create event | `/events/new` | ⚠️ | Form complete; categories showed offline error |
| My events | `/events/mine` | ✅ | Draft + publish flow |
| Personal calendar | `/me/calendar` | ✅ | Fast without VPN (<100ms UI); was ~5–8s over VPN |
| My practices | `/me/practices` | ✅ | RSVP list with cancel; ~26ms content ready without VPN |
| My applications | `/me/applications` | ✅ | Empty for test user |
| Map browse | `/map` | ⚠️ | UI present; tiles black in test browser |
| AI search | `/search` | ❌ | Stub (`ComingSoon`) |
| Organizer profile | `/me/organizer` | ✅ | Form present; save gated on name |
| Admin | `/admin/*` | 🔒 | Redirect for non-admin (correct) |
| Follow/subscribe | `/organizers/[id]` | ⏳ | Not verifiable — no verified orgs in seed |
| Complaints | event detail | ✅ | Button present; modal for authed users |
| Theme toggle | nav | ✅ | Light/dark, persists in localStorage |
| Mobile tab bar | layout | ✅ | 5 tabs; hidden ≥640px and on detail/create |

---

## Detailed findings

### 1. Discovery (`/`)

**Works:**
- 9 published seed events with real titles, venues, prices, categories
- Mix of real cover photos and category gradient placeholders
- Category filter chips (e.g. «Медиации» → 2 events)
- Client-side search by title
- Theme toggle, glass nav, responsive tab bar
- Data from live API (not `lib/mock-events.ts`)

**Gaps:**
- **«Сегодня» / «Выходные»** chips visible but not wired — filter by slug `today`/`weekend`, which no event matches (`frontend/lib/mock-events.ts` documents this)
- Search placeholder says «месту, ведущему» but `DiscoveryFeed` only matches **title + organizer name**, not venue
- **«рядом со мной»** requires browser geolocation (not exercised in audit)
- Glass nav only on `/`; other tab routes use plain back links

### 2. Event detail (`/events/[id]`)

**Works:**
- SSR fetch, hero cover, category chips, 2×2 fact grid
- Description, venue + metro, Leaflet map with marker
- Sticky bottom bar: price, «Записаться», «В календарь» (ICS)
- «Пожаловаться» complaint button
- Tab bar hidden (signup CTA bar instead)

**RSVP tested:**
- «Записаться» on «Выставка: конструктивизм и хлеб» → «Отписаться» / «Вы записаны»
- API: `POST /events/{id}/rsvp` → 200; `me/practices` returns 1 event
- **After reload**, CTA resets to «Записаться» — `GET /events/{id}` does not populate `my_rsvp_status` for the caller

### 3. Authentication

**Works:**
- Email + password modal; register toggle
- Session in `localStorage` (`lia.auth.token`, `lia.auth.email`)
- Logged-in nav: Календарь, Мои события, email, Выйти
- `POST /api/v1/auth/login` → 200 + JWT
- `GET /auth/me` → role `common`

**Issues:**
- First UI login: **«Не удалось войти (503)»** — transient GateGuard/backend flake (see `docs/HANDOFF.md`)
- No admin link for test user (admin: `poulissimo@gmail.com` per HANDOFF)

### 4. Create event (`/events/new`)

**Works:**
- Cover upload, title, description, format, venue typeahead, dates, free/paid, draft/publish
- Auth gate for anonymous users
- «Сохранить» in top bar

**Issues:**
- Categories showed **«Категории недоступны (бэкенд офлайн)»** on first load while `GET /api/v1/categories` returns 8 items with 200 — client race or error-handling bug

### 5. My events (`/events/mine`)

- Lists user-created events
- Test user has **1 draft**: «смерть паши» (Медиации, ГМИИ, 10 Sep)
- «Черновик» badge + «Опубликовать» action

### 6. Personal calendar (`/me/calendar`)

**Works:**
- Month / Week / Day views, navigation, colour legend
- RSVP'd exhibition appears on **3 July** (blue = attending)
- API: `GET /me/calendar` returns event with `attending: true`

**Issues:**
- Over VPN: «Загрузка…» for **5–8 seconds** before grid renders
- **Without VPN (retest):** brief spinner then grid in **<100ms**; `GET /me/calendar` **165ms** cold → **~25–31ms** warm (curl)
- Root cause: VPN latency to `api.lia.pashteto.com`, not slow backend — optional skeleton UI still nice-to-have

### 7. My practices (`/me/practices`)

- Upcoming / Past tabs
- Shows RSVP'd exhibition with «Отписаться»
- Over VPN: same long delay as calendar; **without VPN: ~26ms** to content

### 8. My applications (`/me/applications`)

- Status filters: В ожидании / Принятые / Отклонённые / Отозванные
- «Пока ничего.» for test user

### 9. Map (`/map`)

- «События на карте» + «Искать в этой области»
- Map tiles black in automation browser; embedded venue map on detail **did** render

### 10. AI search (`/search`)

- Stub only — references `design/screens/ai-search.html`

### 11. Admin (`/admin/*`)

- Role `common` → loading then redirect to `/`
- API `GET /admin/overview` → 403

### 12. Follow / subscribe

- Code and API exist; seed list API returns `organizer: null`; no verified organizers to follow on prod

---

## Navigation & responsive behavior

| Viewport | Bottom glass tab bar | Top «Создать событие» |
|----------|----------------------|------------------------|
| **< 640px** (`sm:hidden`) | Visible | Hidden (tab «Создать») |
| **≥ 640px** | Hidden | Visible in glass nav |

Tab bar also hidden on: event detail, create form, admin (`frontend/components/ui/TabBar.tsx`).

Glass nav (`GlassNav`) is mounted **only** on `app/page.tsx`, not on Calendar, Map, Search, etc.

---

## Production data snapshot

**Published events (9):** e.g. Выставка: конструктивизм и хлеб (500 ₽, Центр «Зотов»), Кураторская экскурсия (бесплатно, Гараж), … through Летний фестиваль медиаискусства.

**Test user (`pashteto@inbox.ru`):**
- 1 draft event
- 1 active RSVP (exhibition, 3 Jul)
- 0 follows, 0 applications

---

## Latency retest (2026-06-27, no VPN)

Re-tested after disconnecting VPN, using the same production environment and an admin session (`poulissimo@gmail.com`) for authenticated endpoints. Previous 5–8s «Загрузка…» on calendar/practices was **not reproducible**.

| Measurement | With VPN (initial audit) | No VPN (retest) |
|-------------|--------------------------|-----------------|
| `GET /api/v1/events` (curl) | ~1s cited | **~106ms** total (ttfb ~105ms) |
| `GET /api/v1/categories` (curl) | — | **~26ms** total |
| `GET /api/v1/me/calendar` (curl, 5 samples) | — | **165ms** cold → **~25–31ms** warm |
| `GET /auth/me` (curl, 5 samples) | — | **~153–329ms** (first request slower) |
| Calendar UI: «Загрузка…» → grid | **5–8s** | **<100ms** (brief flash) |
| Practices UI: «Загрузка…» → content | **5–8s** | **~26ms** |
| `GET /auth/me` (browser fetch, 5×) | — | **~102ms avg** (22ms / 156ms alternating) |

**Conclusion:** API and UI are fast on a direct route. The initial audit's calendar slowness should **not** be filed as a backend bug. Remaining UX work is optional polish (skeleton instead of text spinner). **Keep VPN latency in mind when demoing from abroad.**

---

## Priority issues

| # | Issue | Severity | Suggested fix |
|---|-------|----------|---------------|
| 1 | «Сегодня»/«Выходные» filters non-functional | High | Implement date filtering or hide chips |
| 2 | Search ignores venue despite UI copy | Medium | Extend `DiscoveryFeed` filter to `event.venue?.name` |
| 3 | RSVP state lost on page reload | Medium | Return `my_rsvp_status` on authenticated `GET /events/{id}` |
| 4 | Calendar/practices «Загрузка…» (VPN-only slowness) | Low | Optional skeleton UI; no backend fix needed — retest without VPN |
| 5 | Login 503 intermittently | Medium | GateGuard stability on vds-ru215 (see HANDOFF) |
| 6 | Categories «бэкенд офлайн» on create | Medium | Fix categories query error/retry UX |
| 7 | AI Search stub | Low (planned) | Build or remove tab until ready |
| 8 | Inconsistent app chrome | Low | Shared `AppShell` with glass nav on all tab routes |

---

## What’s solid

- Real backend integration (not mock fallback on prod); sub-200ms API on direct network path
- Curated Russian seed content
- Apple HIG design system on main screens
- RSVP → practices → calendar pipeline
- Draft → publish for organizers
- Auth, complaints, admin gating, file uploads wired
- Event detail with map and sticky CTA is polished

---

## API endpoints verified

```
GET  /api/v1/events?status=published     → 200 (9 events)
POST /api/v1/auth/login                   → 200 + JWT
GET  /auth/me                            → 200 (role: common)
GET  /api/v1/events/mine                 → 200 (1 draft)
GET  /api/v1/me/calendar                 → 200 (1 after RSVP)
GET  /api/v1/me/practices?tab=upcoming   → 200 (1 after RSVP)
GET  /api/v1/categories                  → 200 (8 categories)
GET  /api/v1/events/nearby               → 200
GET  /admin/overview                     → 403 (non-admin)
POST /api/v1/events/{id}/rsvp            → 200 / 409 if duplicate
```

---

## References

- Frontend README: `frontend/README.md`
- Design system: `design/DESIGN.md`
- Deploy / ops: `docs/HANDOFF.md`
- Screen status table: `frontend/README.md` (Discovery ✅, Detail ✅, Create ✅, AI Search ⏳)
- Admin audit (same day): [Production Admin Frontend Audit](./2026-06-27-lia-pashteto-production-admin-audit.md)
