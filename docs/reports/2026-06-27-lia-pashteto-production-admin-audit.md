# Presence.Tarski — Production Admin Frontend Audit

**Date:** 2026-06-27  
**Environment:** https://lia.pashteto.com (API: https://api.lia.pashteto.com)  
**Method:** Live browser walkthrough as admin + API verification against codebase  
**Test account:** `poulissimo@gmail.com` (role: `admin`, display name: poul)  
**Sessions:** Initial walkthrough over VPN; latency re-verified **without VPN** same day (see [Latency retest](#latency-retest-2026-06-27-no-vpn))  
**Companion report:** [User-facing audit](./2026-06-27-lia-pashteto-production-frontend-audit.md)

---

## Executive summary

The **admin suite (sub-projects 0–3)** is live on production and usable end-to-end for staff with the GateGuard `admin` role. Overview stats, event moderation, organizer verification, organizer search/detail, complaints inbox, and global settings all load against the real API — not mocks. RBAC gating: non-admin users are redirected from `/admin/*` (verified in the user audit).

The admin surface is clearly **MVP / ops tooling**: no pagination, no audit-log viewer, no featured-curation or taxonomy screens. **Destructive actions** (takedown, reinstate, verify/reject, complaints resolve) were **not executed** on prod to avoid altering live seed content; modal UX and read paths were verified instead.

**Latency note:** Initial audit over VPN reported slow «Загрузка…» gates and multi-second admin-link delays. Re-test **without VPN** showed admin API in **~27–360ms**, admin layout ready in **<50ms**, «Админ» nav link in **~150–200ms** (`getMe`-bound), and tab-switch stale flash in **~26ms**. Slowness was **VPN-dominated**; remaining delay is short `getMe` gating, not backend performance.

**Overall admin maturity:** ~75% of planned admin-suite scope is live; remaining items are documented in `docs/HANDOFF.md` (#4 featured, #5 taxonomy, role split, audit viewer, pagination).

---

## Infrastructure (admin-specific)

| Component | Status | Notes |
|-----------|--------|-------|
| RBAC gate (`/admin/*` layout) | ✅ | Unauthed → `/`; non-admin → `/` after `getMe` |
| Admin API mount (`internal/http/admin`) | ✅ | Plain `net/http`, ahead of swagger mux |
| `GET /auth/me` role sync | ✅ | Returns `role: admin` for test account |
| Admin nav link (main app) | ⚠️ | Appears after `roleResolved`; **~150–200ms** without VPN (was ~1–3s over VPN) |
| Mobile tab bar on admin routes | ✅ | Hidden (`TabBar` skips `/admin/*`) |
| Admin glass nav (section chrome) | ✅ | Sticky header on all `/admin/*` pages |

---

## Admin feature matrix

| Feature | Route | Status | Notes |
|---------|-------|--------|-------|
| Access control | `/admin/*` | ✅ | Admin session loads; common user blocked (see user audit) |
| Overview dashboard | `/admin` | ✅ | 5 stat cards + shortcuts to moderation & complaints |
| Event moderation queue | `/admin/moderation/events` | ✅ | Published / Снятые tabs; takedown modal; reinstate on rejected |
| Organizer moderation | `/admin/moderation/organizers` | ✅ | Pending / Подтверждённые / Отклонённые tabs |
| Organizer search & detail | `/admin/organizers` | ✅ | Search + detail work; **~29ms** API / **<100ms** UI without VPN |
| Complaints inbox | `/admin/complaints` | ✅ | Empty state; resolve UI present (not exercised — 0 open) |
| Global settings | `/admin/settings` | ✅ | `organizers.auto_verify_all` toggle (read; not toggled) |
| «Админ» link in main nav | `/` (GlassNav) | ⚠️ | Delayed until `getMe` (~150–200ms without VPN) |
| Featured curation | — | ❌ | Not built (#4) |
| Taxonomy admin | — | ❌ | Not built (#5) |
| Audit log viewer | — | ❌ | Not built |
| Role promotion UI | — | ❌ | Manual SQL in GateGuard DB per HANDOFF |
| Moderator vs admin split | — | ❌ | Single `admin` role only (Approach 1) |

---

## Detailed findings

### 1. Overview (`/admin`)

**Works:**
- Stat cards populated from live API:
  - **29** total events
  - **9** published (seed feed)
  - **3** removed (moderation takedowns)
  - **0** organizers pending review
  - **0** open complaints
- Shortcuts: «Открыть очередь модерации →», «Открыть жалобы →»
- «‹ События» back link to discovery feed
- Dedicated **Lia Admin** glass nav with 5 section links

**Gaps:**
- Brief «Загрузка…» while `AdminLayout` waits for `roleResolved` on cold navigation — **<50ms** without VPN (was several seconds over VPN)
- No drill-down from stat cards (counts are display-only)

### 2. Event moderation (`/admin/moderation/events`)

**Works:**
- **Опубликованные** tab lists all 9 seed events with datetime + «Снять»
- **Снятые** tab lists 3 rejected user/test events with organizer name, takedown reason, «Вернуть»
- Takedown modal: title confirmation, required reason textarea, disabled submit until reason entered, «Отмена» dismisses (tested on seed event, **not submitted**)

**Rejected events on prod:**

| Title | Organizer | Reason (excerpt) |
|-------|-----------|------------------|
| Парольное событие | Парольный Тест | wefqwefqwef |
| repro published | Debug Repro | eded |
| cover smoke | Smoke | wrvqwefqwef |

**Gaps:**
- **Tab switch flash:** switching to «Снятые» briefly shows published rows with «Вернуть» labels before API data replaces them — **~26ms** stale / **~52ms** correct without VPN (was ~1–2s over VPN)
- Published-tab rows omit `organizer_name` (API returns title + `starts_at` only for seed UUIDs)
- No pagination (fine at 9/3 today; will not scale)
- Reinstate / takedown mutations not exercised on prod

### 3. Organizer moderation (`/admin/moderation/organizers`)

**Works:**
- **На проверке:** empty («Пусто.») — matches API `[]`
- **Подтверждённые:** shows **Smoke Verification Org** (`https://smoke.test`, description «rollout test»)
- **Отклонённые:** empty
- Pending tab exposes «Подтвердить» / «Отклонить» (no pending orgs to test)

**Gaps:**
- Verified/rejected tabs are read-only (by design — actions live on pending tab and `/admin/organizers`)
- Verify/reject/revoke mutations not exercised on prod

### 4. Organizer search & detail (`/admin/organizers`)

**Works:**
- Search `Smoke Verification` → result **Smoke Verification Org · Подтверждён**
- Detail panel after click:
  - Status, description, website
  - **Авто-подтверждение** per-organizer checkbox (`auto_verify: false`)
  - **Отозвать подтверждение** section (reason required; button disabled until filled)
  - **История:** `draft → pending` (26 Jun 18:53), `pending → verified` (26 Jun 19:17)

**Gaps:**
- Search has **no loading indicator** — over VPN felt like **~2–5s**; without VPN API **~29ms**, results **<100ms** (still worth a spinner / empty state)
- Empty search state not surfaced when query returns `[]` (only silent no-op)
- Revoke / auto-verify toggle not exercised on prod

### 5. Complaints inbox (`/admin/complaints`)

**Works:**
- Page loads under admin layout
- «‹ Админ» back link
- Empty state: «Жалоб нет.» — matches `GET /admin/complaints` → `[]`
- `complaints_open: 0` on overview

**Not verified (no data):**
- Grouped complaint cards with category chips
- «Снять» takedown-from-complaint modal
- «Отклонить» dismiss action
- Known UX debt (per HANDOFF): takedown error may render behind modal; no double-click guard on dismiss

### 6. Settings (`/admin/settings`)

**Works:**
- **Авто-подтверждение всех организаторов** checkbox loads from API
- Current value: **`organizers.auto_verify_all: false`**
- Descriptive helper copy for when to enable

**Gaps:**
- Same brief «Загрузка…» on cold load as other admin routes — negligible without VPN
- Toggle mutation not exercised (left off intentionally)

### 7. Main-app integration (admin as user)

**Works:**
- After `getMe` resolves, GlassNav shows **Календарь**, **Мои события**, **Админ**, **Выйти**
- Admin can browse the public discovery feed normally while authenticated
- Bottom mobile tab bar hidden on all `/admin/*` routes

**Gaps:**
- **«Админ» link missing until `getMe` completes** — **~150–200ms** without VPN (was ~1–3s over VPN); Календарь/Мои события appear immediately from `localStorage`
- Email display in nav is `hidden` below `sm` breakpoint — admin identity not visible on mobile

---

## Navigation & layout

| Surface | Admin chrome | Mobile tab bar |
|---------|--------------|----------------|
| `/admin` | Lia Admin glass nav | Hidden |
| `/admin/moderation/*` | Same | Hidden |
| `/admin/organizers` | Same | Hidden |
| `/admin/complaints` | Same + «‹ Админ» back | Hidden |
| `/admin/settings` | Same | Hidden |
| `/` (logged-in admin) | Discovery GlassNav + «Админ» link | Visible (<640px) |

Admin section uses its **own** sticky glass nav (`frontend/app/admin/layout.tsx`), separate from the public GlassNav on `/`.

---

## Production admin data snapshot

**Events (moderation lens):**
- 29 total in DB; 9 published seed; 3 moderation-rejected test events; remainder drafts/other statuses

**Organizers:**
- 0 pending
- 1 verified: **Smoke Verification Org**
- 0 rejected

**Complaints:** 0 open

**Settings:** `organizers.auto_verify_all = false`

**Admin user (`poulissimo@gmail.com`):**
- User id `0fc7522f-a50d-49b3-8c16-cece9d97e1b4`
- Verified Smoke Verification Org (actor on `pending → verified` transition)

---

## Latency retest (2026-06-27, no VPN)

Re-tested after disconnecting VPN, same admin account (`poulissimo@gmail.com`). Admin slowness from the initial VPN session is **mostly gone**; what remains is short, expected `getMe` gating plus a minor tab-switch UI bug.

| Measurement | Initial audit (VPN) | No VPN (retest) |
|-------------|---------------------|-----------------|
| `GET /auth/me` (browser fetch, 5×) | — | **~102ms avg** (22ms / 156ms alternating) |
| `GET /auth/me` (curl, 5 samples) | — | **~153–329ms** (first request slower) |
| `GET /api/v1/admin/overview` (curl, 5 samples) | — | **~27–30ms** |
| Admin layout «Загрузка…» → content | several seconds perceived | **<50ms** after navigation (brief flash) |
| «Админ» link on `/` after reload | **~1–3s** | **~150–200ms** (`getMe`-bound) |
| Event moderation tab switch stale flash | **~1–2s** wrong rows | **~26ms** stale / **~52ms** correct data |
| Organizer search API + UI | **~2–5s** perceived | **~29ms** API / **<100ms** to results |
| Admin authenticated endpoints (browser fetch, parallel) | — | **~200–360ms** first batch |

**Conclusion:** VPN inflated perceived latency across admin and user surfaces (see companion user audit). Still worth caching `role` in `sessionStorage` to eliminate the ~150–200ms «Админ» link gap; tab-switch flash fix remains valid but is now sub-100ms and low priority.

---

## Priority issues (admin)

| # | Issue | Severity | Suggested fix |
|---|-------|----------|---------------|
| 1 | Admin layout «Загрузка…» on cold `/admin` visit | Low | Cache role after first `getMe`; skeleton instead of blank screen (~150–200ms today without VPN) |
| 2 | «Админ» nav link delayed until `getMe` | Low | Same role-cache fix in `auth-context` |
| 3 | Tab switch stale list flash (events moderation) | Low | Clear `items` + spinner on tab change (~26ms flash today) |
| 4 | Organizer search lacks loading / empty feedback | Low | Spinner on `searching`; «Ничего не найдено» for empty results |
| 5 | No queue pagination | Low (until scale) | Add cursor/limit to admin list endpoints + UI |
| 6 | Complaints resolve flow unverified on prod | Medium | Seed a test complaint in staging; run takedown + dismiss E2E |
| 7 | Missing admin features (#4, #5, audit viewer) | Planned | See HANDOFF admin-suite backlog |
| 8 | No in-app role promotion | Low | Document ops runbook or build promotion UI |

---

## What’s solid (admin)

- Full RBAC gate: API 403 for non-admin, frontend redirect for wrong role
- Real overview counts driving dashboard cards
- Event post-moderation queue with reason-required takedown UX
- Organizer verification lifecycle visible in moderation + search/detail
- Verification history with timestamps and status transitions
- Complaints inbox wired (empty but functional)
- Global + per-organizer auto-verify controls present
- Admin section has consistent glass nav and hides mobile tab bar
- All read-path admin API endpoints return 200 for admin JWT; **~27–360ms** on direct network path

---

## Admin API endpoints verified

```
GET  /auth/me                                              → 200 (role: admin)
GET  /api/v1/admin/overview                                → 200 (counts above)
GET  /api/v1/admin/moderation/events?status=published      → 200 (9 seed events)
GET  /api/v1/admin/moderation/events?status=rejected       → 200 (3 rejected)
GET  /api/v1/admin/moderation/organizers?status=pending    → 200 ([])
GET  /api/v1/admin/moderation/organizers?status=verified   → 200 (1 org)
GET  /api/v1/admin/moderation/organizers?status=rejected   → 200 ([])
GET  /api/v1/admin/organizers?q=Smoke                      → 200 (1 org)
GET  /api/v1/admin/organizers/{id}                         → 200 (detail + history)
GET  /api/v1/admin/complaints                              → 200 ([])
GET  /api/v1/admin/settings                                → 200 (auto_verify_all: false)

Not exercised (avoid prod mutations):
POST /api/v1/admin/moderation/events/{id}/takedown
POST /api/v1/admin/moderation/events/{id}/reinstate
POST /api/v1/admin/moderation/organizers/{id}/verify|reject|revoke
POST /api/v1/admin/organizers/{id}/auto-verify
PUT  /api/v1/admin/settings
POST /api/v1/admin/complaints/events/{id}/resolve
```

---

## References

- User-facing audit: `docs/reports/2026-06-27-lia-pashteto-production-frontend-audit.md`
- Admin frontend: `frontend/app/admin/`
- Admin API handler: `backend/internal/http/admin/handler.go`
- Deploy / ops: `docs/HANDOFF.md` (admin suite backlog, `poulissimo@gmail.com` promotion)
