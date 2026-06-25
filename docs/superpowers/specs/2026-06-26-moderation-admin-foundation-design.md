# Moderation/Admin foundation — RBAC + event moderation (design)

_Date: 2026-06-26. Status: **design, pending review.** First slice of the larger admin suite (sub-projects 0 + 1 combined). Plan: [`../plans/2026-06-26-moderation-admin-foundation.md`](../plans/2026-06-26-moderation-admin-foundation.md)._

## 1. Goal

Give Lia a real moderation capability: staff can review events and **take down**
(and reinstate) ones that don't meet the curatorial bar, with every action
attributed and logged. This is the foundation slice of the admin suite from the
vision docs (`docs/design_agent_prompt.md` §4.3, `docs/event_discovery_mvp_technical_stack.md`)
— it stands up role-based access (RBAC), an audit trail, and the `/admin` shell
that all later admin sub-projects hang off.

This is a **security-surface change** (introduces privileged actions over
user-generated content). It must be documented for ISO 27001 / Vanta — see §9.

## 2. Decisions (confirmed during brainstorming)

- **RBAC source — GateGuard-propagated, reusing the existing `admin` role.**
  GateGuard stays the source of truth for roles. We do **not** add a `moderator`
  role to GateGuard in this slice (that would force a proto regen + a rebuild and
  `docker save|ssh|load` reship of GateGuard to the flaky box). Instead, Lia
  treats GateGuard's existing `admin` role as "staff" (may moderate + administer).
  The moderator/admin *distinction* is deferred to a follow-up when org-verification
  / complaints need the finer split.
- **Moderation model — post-moderation.** New events keep publishing on create
  (the current, deliberate behavior — an omitted status must not create an
  invisible draft). Moderators act **after** the fact: take down a published
  event (reason required) so it leaves Discovery, or reinstate a removed one.
  `pending_review` remains in the status enum but is unused here.

## 3. Scope

**In scope**
- Backend: stop discarding the GateGuard role; cache it on the local user; gate
  `/admin/*` routes behind it.
- New `internal/moderation` module: take-down / reinstate transitions, status
  history, audit log.
- `GET /auth/me` so the frontend (which holds an opaque GateGuard JWT) can learn
  the caller's role.
- Frontend `/admin` shell: gate, home/overview, event moderation queue; an
  «Админ» header link; take-down badge on the organizer's own event views.
- Migration `000012`: `users.role`, `event_status_history`, `audit_log`.

**Out of scope (later sub-projects, each its own spec)**
- A `moderator` role distinct from `admin` (needs the GateGuard proto change).
- Organizer entity + verification queue (`/admin/moderation/organizers`).
- Complaints inbox, featured curation, categories/interests admin, user/org search.
- An in-app role-promotion UI — bootstrap is manual (§8).
- Pre-moderation / a `pending_review` gate.

## 4. Background — what already exists

- **Role already crosses the wire.** `gg.User` (vendored proto) has a `Role`
  field (`UserRole`: `common/viewer/billing/admin`), and GateGuard's `CheckAuth`
  returns it via `user.Proto()`. Lia's `internal/http/auth/gatekeeper.go:72`
  currently **drops it** — it builds `Claims{Subject, Email, Name}` only. So
  consuming the role is mostly "stop throwing it away."
- **`Claims`** (`internal/http/auth/auth.go`) is the seam: `TokenValidator.Validate`
  → `Claims` → `Auth.CheckAuth` → JIT-provisions a local `models.User` by email,
  which is the principal handlers receive.
- **GateGuard** has an `UpdateUserRole` RPC and an `admin` role today
  (`gateguard/internal/models/user_role.go`).
- **Discovery already filters by status.** `events/repository.go:186` applies a
  `status = ?` filter when set; `Nearby` hardcodes `status = 'published'`. So a
  take-down = flip status off `published` and the event disappears from public
  surfaces with no extra filtering work.
- **`/events/mine`** already returns the organizer's events in all statuses — the
  natural place to surface a take-down badge.

## 5. Data model — migration `000012`

```sql
-- 1. Local cache of the GateGuard role (source of truth stays GateGuard).
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'common';

-- 2. Append-only moderation timeline for an event.
CREATE TABLE event_status_history (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  from_status   TEXT NOT NULL,
  to_status     TEXT NOT NULL,
  actor_user_id uuid NOT NULL,           -- the staff member who acted
  reason        TEXT,                    -- required on take-down, null otherwise
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX event_status_history_event_idx ON event_status_history (event_id, created_at DESC);

-- 3. Generic admin-action audit log (the vision's admin_actions).
CREATE TABLE audit_log (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id uuid NOT NULL,
  action        TEXT NOT NULL,           -- e.g. 'event.takedown', 'event.reinstate'
  target_type   TEXT NOT NULL,           -- e.g. 'event'
  target_id     uuid NOT NULL,
  metadata      jsonb NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_target_idx ON audit_log (target_type, target_id, created_at DESC);
```

Notes:
- `actor_user_id` is intentionally **not** a hard FK (audit rows must survive a
  user delete; same rationale as keeping the log append-only). `event_status_history.event_id`
  *is* a cascade FK — history dies with the event it describes.
- `gen_random_uuid()` requires `pgcrypto`; PostGIS images include it, but the
  migration will `CREATE EXTENSION IF NOT EXISTS pgcrypto` to be safe.
- Watch the go-pg + gofrs-UUID NULL gotcha (HANDOFF): all uuid columns here are
  `NOT NULL`, so no NULL-into-uuid scans.

## 6. Backend design

### 6.1 Role propagation (RBAC)
- `Claims` gains a `Role string` field. `gatekeeper.go` maps `gg.User.Role`
  (`UserRole.String()` → `"admin"|"common"|…`) into it instead of discarding it.
- `Auth.ensureUser` (JIT-provision) **syncs** the role onto the local `users.role`
  on every `CheckAuth`: GateGuard is authoritative, the column is a fresh cache.
  The returned `*models.User` principal carries `Role`.
- A thin `IsStaff()` helper on the principal: `role == "admin"` (this slice). The
  single place to widen later when `moderator` lands.

### 6.2 Admin gate
- Middleware `requireStaff` wraps the `/api/v1/admin/*` group: runs the existing
  `CheckAuth`, then `403` (RU copy «Недостаточно прав») unless `principal.IsStaff()`.
  `401` for missing/invalid token (unchanged path). Anonymous → 401, common → 403,
  admin → through.

### 6.3 `internal/moderation` module
New module (matches the vision's `/internal/moderation`), depends on the events
repository for the transition + on the two new tables.

- `Takedown(ctx, eventID, actorID, reason)`:
  - require non-empty `reason` (400 «Укажите причину» if empty),
  - load event; require current `status == published` (409 if not),
  - set `status = rejected`, write an `event_status_history` row
    (`published → rejected`, reason), write an `audit_log` row (`event.takedown`),
  - all in one transaction.
- `Reinstate(ctx, eventID, actorID)`:
  - require current `status == rejected` (409 if not),
  - set `status = published`, history (`rejected → published`, no reason),
    audit (`event.reinstate`), one transaction.
- Reuses the existing `rejected` status for "removed by moderator" — no new enum
  value (the reason in the history row distinguishes intent; a dedicated
  `removed` status is a possible later refinement, called out, not built).

### 6.4 HTTP endpoints
All under `requireStaff` except `/auth/me` (any authed user):

| Method | Path | Purpose |
|---|---|---|
| GET | `/auth/me` | `{id, email, name, role}` for the caller (frontend role gate; sits under `/auth` like the other auth routes, not `/api/v1`) |
| GET | `/api/v1/admin/overview` | counts: `{events_total, events_published, events_removed}` |
| GET | `/api/v1/admin/moderation/events?status=published\|rejected&limit&offset` | review queue (any status, includes `organizer`) |
| POST | `/api/v1/admin/moderation/events/{id}/takedown` | body `{reason}` → published→rejected |
| POST | `/api/v1/admin/moderation/events/{id}/reinstate` | rejected→published |

`/auth/me` exists because the GateGuard JWT is **opaque to Lia and the frontend** —
the UI cannot decode the role from the token, so it must ask the server.

## 7. Frontend design (Next.js App Router, liquid-glass styling)

- `lib/api`: add `getMe()`, `getAdminOverview()`, `listModerationEvents(status)`,
  `takedownEvent(id, reason)`, `reinstateEvent(id)`.
- Auth/session store gains `role` (populated from `getMe()` after login / on load).
- `app/admin/layout.tsx` — fetches `me`; if `role !== 'admin'` redirect to `/`.
  Renders the admin chrome (nav: Обзор, Модерация событий; later items as
  disabled «скоро» tiles).
- `app/admin/page.tsx` — overview cards (total / published / removed) + link into
  the queue.
- `app/admin/moderation/events/page.tsx` — tabs **Опубликованные / Снятые**;
  rows show cover, title, organizer name, date, status badge; **Снять** opens a
  reason modal (required) → `takedownEvent`; **Вернуть** → `reinstateEvent`;
  list refreshes after each action. Loading skeleton + empty state.
- Header (`GlassNav`): show an **«Админ»** link when `role === 'admin'`.
- `/events/mine` + event-detail (organizer's own view): when `status === rejected`,
  show a **«Снято модератором»** badge + the reason (latest history row, fetched
  with the event or via a small `history` include — see §10).

## 8. Bootstrap (no UI in this slice)

Roles live in GateGuard. To create the first admin, set their role in GateGuard:
- **SQL** on the `gateguard` DB: `UPDATE users SET role = 'admin' WHERE email = '<you>';`
  (column/enum per `gateguard/internal/models/user_role.go`), **or**
- call GateGuard's `UpdateUserRole` RPC.

On the next request, Lia's `CheckAuth` reads the propagated `admin` role and syncs
`users.role`. This is documented in the plan's deploy section; no GateGuard
rebuild/reship is required (Approach 1).

## 9. Compliance note (ISO 27001 / Vanta)

- Introduces **privileged actions** (take-down / reinstate) over user content.
  Every action is attributed (`actor_user_id`) and **append-only logged**
  (`audit_log` + `event_status_history`) — an auditable change-management trail.
- **Least privilege:** the `/admin/*` group is gated; only GateGuard-`admin`
  users pass. Anonymous → 401, authenticated non-admin → 403.
- **Take-down is reversible** (reinstate) and **non-destructive** (status flip, no
  row delete) — blast radius is "event hidden from Discovery," fully recoverable.
- Role is sourced from GateGuard (central identity), cached read-only in Lia;
  Lia never mints or elevates roles.
- This affects access reviews + change management; flag for documentation per the
  org audit-awareness standard.

## 10. Open questions / minor decisions

1. **Take-down reason on the organizer view** — fetch the latest
   `event_status_history` row inline with the event response (an optional
   `moderation` block) vs a separate `GET /admin/events/{id}/history`. Lean: inline
   `moderation: {status, reason}` on `/events/mine` rows only (cheapest, no extra
   round-trip); resolve in the plan.
2. **Overview counts** — exact set (`total/published/removed`) is enough for the
   home; add `recent` later if the dashboard needs it.
3. **Queue pagination** — simple `limit/offset`; demo data volume is tiny.

## 11. Testing strategy

- **Backend:** `moderation` service tests — take-down requires `published` +
  non-empty reason, writes one history + one audit row, flips status; reinstate
  requires `rejected`; transactional (no partial writes). Gate tests — 401
  anon / 403 common / 200 admin. `/auth/me` returns the propagated role.
  `go build/vet/test ./...` + `golangci-lint` (v1) clean.
- **Frontend:** `pnpm lint` + `pnpm build`. Playwright/manual: admin sees the
  queue; take-down removes the event from Discovery; reinstate restores it;
  non-admin is redirected from `/admin` and sees no «Админ» link.
- **Live:** migration `000012` is the only schema delta; **no GateGuard reship**.
  Seed one admin (§8) and smoke-test. Full live verification deferred per the
  flaky-box norm (build images on the Mac, `docker save|ssh|load`).
