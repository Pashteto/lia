# Organizer Entity + Verification — Design (verification-only slice)

_Date: 2026-06-26. Sub-project **#2** of the admin suite ([roadmap](../plans/2026-06-26-admin-suite-roadmap.md)). Status: **design, awaiting review.**_

## 1. Goal & scope

Introduce a real `organizers` entity that a user can opt into, plus an admin
**verification** workflow and an **auto-approve** escape hatch for when no
moderators are available. This is the **verification-only slice** chosen during
brainstorming — it deliberately does NOT re-attribute events to organizations.

**In scope**
- `organizers` table: a **1:1** profile keyed by `owner_user_id` (one user → at most one organizer profile).
- Verification lifecycle `draft → pending → verified | rejected` with resubmit and admin revoke.
- **Auto-approve**, two independent layers:
  1. **Global** runtime toggle (admin-controlled, stored in a new `app_settings` store): when on, every submitted draft is auto-verified for everyone.
  2. **Per-organizer** `auto_verify` trust flag (admin-set on an org): that org's submissions skip the queue.
- User-facing profile management (`/me/organizer`), admin verification queue + search/detail, an admin **settings** page, a public **verified** badge on events, and a public organizer page.

**Out of scope (deferred to later slices)**
- `organizer_members` / teams, org switcher, the `/o/*` organizer dashboard.
- Event → organizer FK re-attribution (events keep `organizer_id = creator user id`).
- Slugs (`/organizers/[slug]`); we route by id. Cyrillic-slug generation is a later nicety.
- Retroactive sweep when the global toggle flips on (applies at submit-time only).

## 2. Current state (what we build on)

- `events.organizer_id` **already exists** (migration `000004`) and holds the **creator's user id**;
  it is forced server-side to the authenticated user on create (`handlers/events.go`).
- The `Organizer` read-model (`models/event.go`: `UUID, Name, AvatarURL`) is built by the batched
  `loadOrganizers` query (`events/repository.go`) joining `users` + `files` — no N+1, no DB column.
- `users.role` (migration `000014`) drives RBAC; the admin gate (`internal/http/admin`, `requireStaff`)
  allows `role == "admin"`, else 403 «Недостаточно прав» / 401 anon. Mounted ahead of the go-swagger mux.
- `moderation` domain is the template: a `transition()` helper runs the state change + one
  `event_status_history` row + one `audit_log` row **in a single `RunInTransaction`**, using a
  `WHERE status = <expected>` guard → `ErrInvalidTransition` (409) when not in the expected state.
- `audit_log` (`actor_user_id, action, target_type, target_id, metadata jsonb, created_at`) is reused.
- The `files` domain (upload via `POST /api/v1/uploads`, public read `GET /api/v1/files/{key}`) is reused for logos.
- Next migration number is **`000015`**.

## 3. Data model — migrations `000015` + `000016`

### 3.1 `000015_organizers`

```sql
CREATE TABLE organizers (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id       uuid NOT NULL UNIQUE,                 -- 1:1 with users(id)
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    website_url         TEXT NOT NULL DEFAULT '',
    logo_file_id        uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000', -- zero = unset
    verification_status TEXT NOT NULL DEFAULT 'draft'
                        CHECK (verification_status IN ('draft','pending','verified','rejected')),
    auto_verify         boolean NOT NULL DEFAULT false,       -- per-org trust flag (admin-set)
    verified_at         timestamptz,                          -- nullable
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizers_status_idx ON organizers (verification_status, created_at DESC);

CREATE TABLE organizer_verification_history (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id  uuid NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_user_id uuid NOT NULL,                              -- zero-uuid for system/auto actions
    reason        TEXT,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizer_verification_history_org_idx
    ON organizer_verification_history (organizer_id, created_at DESC);
```

Notes:
- **`logo_file_id` is `NOT NULL DEFAULT zero-uuid`** (not nullable) — follows the events/venues pattern because
  go-pg + gofrs UUID cannot scan SQL `NULL` into a uuid field (documented gotcha). Zero = "no logo".
- **No denormalized `latest_reason`** — the latest reject/revoke reason is read from
  `organizer_verification_history` via a `LatestReason(orgID)` repo func (mirrors moderation's `LatestReason`).
- `verified_at` is a nullable `timestamptz` (`*time.Time` in Go — NULL scan is fine for non-uuid columns).

### 3.2 `000016_app_settings`

```sql
CREATE TABLE app_settings (
    key            TEXT PRIMARY KEY,
    value          jsonb NOT NULL DEFAULT '{}',
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid                                       -- admin who last changed it
);
-- Seed the organizer auto-verify-all flag (default off).
INSERT INTO app_settings (key, value) VALUES
    ('organizers.auto_verify_all', '{"enabled": false}')
    ON CONFLICT (key) DO NOTHING;
```

A small generic key/value store so future global toggles reuse it. Reads are infrequent
(once per submit / once per settings page load), so no caching layer — a direct lookup per call.

## 4. State machine

```
draft ──submit──▶ pending ──verify──▶ verified
  ▲                 │                    │
  │                 └──reject(reason)──▶ rejected ──(edit + submit)──▶ pending
  └─(create)                              ▲
                    verified ──revoke(reason)┘

Auto-approve short-circuit (evaluated inside Submit):
   submit, when  global organizers.auto_verify_all == true  OR  org.auto_verify == true
            ──▶ verified   (instead of pending), with an audit row marked {auto:true, source}
```

- **Editing** is allowed in `draft`/`rejected` and re-submitting moves to `pending` (or `verified` if auto).
  Editing a `verified` profile is allowed but does **not** auto-unverify in this slice (admins revoke instead).
- Every transition writes **one `organizer_verification_history` row + one `audit_log` row in a single tx**,
  guarded by `WHERE verification_status = <expected>` → `ErrInvalidTransition` (409) on mismatch.
- `reject` and `revoke` require a non-empty trimmed reason → `ErrReasonRequired` (400).
- Auto-verifications record `actor_user_id = zero-uuid` (system) and `audit_log.metadata =
  {"auto": true, "source": "global" | "org"}` so the bypass is fully traceable.

## 5. Backend domains

### 5.1 `internal/organizers/` (mirrors `internal/moderation/`)

`service.go` — `type Service interface`:
- `GetByOwner(ctx, userID) (*Organizer, error)` — own profile; `ErrNotFound` if none.
- `Upsert(ctx, userID, OrganizerInput) (*Organizer, error)` — create (→ `draft`) or edit; `name` required (`ErrNameRequired`).
- `Submit(ctx, userID) (*Organizer, error)` — `draft|rejected → pending`, or `→ verified` when auto-approve applies.
- `Verify(ctx, orgID, actorID) error` — `pending → verified`.
- `Reject(ctx, orgID, actorID, reason) error` — `pending → rejected`.
- `Revoke(ctx, orgID, actorID, reason) error` — `verified → rejected`.
- `SetAutoVerify(ctx, orgID, actorID, enabled bool) error` — admin toggles the per-org trust flag (audited as `organizer.set_auto_verify`).
- `List(ctx, ListFilter{Status, Query}) ([]Organizer, error)` — admin queue (by status) + search (by name/owner email).
- `GetByID(ctx, orgID) (*Organizer, []HistoryEntry, error)` — admin detail + history.
- `Counts(ctx) (Counts, error)` — `{pending int}` for the admin overview.
- `VerifiedByOwners(ctx, []userID) (map[uuid.UUID]VerifiedOrg, error)` — batched lookup for the event badge (no N+1).

`repository.go` — pg repo with a `transition()` helper structurally copied from moderation;
sentinel errors `ErrInvalidTransition`, `ErrReasonRequired`, `ErrNameRequired`, `ErrNotFound`.
The `Submit` short-circuit reads the global flag via the settings service (injected) and the
row's `auto_verify` before choosing the target state, all inside the transaction.

Wired in `application.go` `registerModules()` only when the DB is available; injected into the HTTP module
alongside the other domains.

### 5.2 `internal/settings/` (new, minimal)

- `Service`: `Bool(ctx, key) (bool, error)`, `SetBool(ctx, key, actorID, val) error`, `All(ctx) (map[string]any, error)`.
- Backed by `app_settings`. `SetBool` writes the row **and** an `audit_log` row
  (`action='settings.update'`, `target_type='setting'`, `metadata={key, value}`) in one tx — toggling a
  detective control is itself an audited event (ISO 27001 / Vanta).
- The organizers service depends on `settings.Service` for the `organizers.auto_verify_all` read.

## 6. Event "verified" badge (read-model)

Extend the batched `loadOrganizers` query in `events/repository.go` with
`LEFT JOIN organizers o ON o.owner_user_id = u.id`. The `Organizer` read-model struct gains:
- `Verified bool` — `o.verification_status = 'verified'`.
- `ProfileID uuid.UUID` — `o.id` (lets the frontend link to `/organizers/{id}`; zero when no profile).

When verified, the read-model **prefers the org name + logo**; otherwise it falls back to the creator's
user name/avatar exactly as today. Still a single query — no N+1. The formatter
(`http/formatter/event.go`) maps `Verified`/`ProfileID` onto the API `Organizer` object (omitempty).

## 7. HTTP endpoints

No go-swagger spec edits — all new routes are plain `net/http`, consistent with the `uploads` and `admin`
handlers already mounted ahead of the swagger mux.

### 7.1 User-facing — new handler `internal/http/organizers` (bearer-required, like `uploads`)
- `GET  /api/v1/me/organizer` → own profile + latest reason (404 if none yet).
- `PUT  /api/v1/me/organizer` → upsert `{name, description, website_url, logo_file_id}` (name required).
- `POST /api/v1/me/organizer/submit` → submit for verification (returns the resulting status).

### 7.2 Admin — added to `internal/http/admin` (`requireStaff`-gated)
- `GET  /api/v1/admin/moderation/organizers?status=pending|verified|rejected` → queue.
- `GET  /api/v1/admin/organizers?q=<search>` → search list.
- `GET  /api/v1/admin/organizers/{id}` → detail + verification history.
- `POST /api/v1/admin/moderation/organizers/{id}/verify`
- `POST /api/v1/admin/moderation/organizers/{id}/reject`  `{reason}`
- `POST /api/v1/admin/moderation/organizers/{id}/revoke`  `{reason}`
- `POST /api/v1/admin/organizers/{id}/auto-verify` `{enabled bool}` → per-org trust flag.
- `GET  /api/v1/admin/settings` → all settings (currently `organizers.auto_verify_all`).
- `PUT  /api/v1/admin/settings` `{key, value}` → update a setting (audited).
- `GET  /api/v1/admin/overview` → Counts extended with `organizers_pending`.

### 7.3 Public — added to `internal/http/organizers`
- `GET /api/v1/organizers/{id}` → public profile **only if `verified`** (404 otherwise, so pending/rejected
  never leak) + the org's **published** events (reuse events list by `owner_user_id`, status=`published`).

### 7.4 Error → status mapping (in the handlers)
`ErrNameRequired`/`ErrReasonRequired` → 400; not found → 404; `ErrInvalidTransition` → 409; else 500.

## 8. Frontend (`frontend/`)

- **`/me/organizer`** — create/edit form (name, description, website, logo upload via the existing uploads
  flow), «Отправить на проверку» button, status badge (draft / на проверке / проверен / отклонён), and the
  rejection/revoke reason shown when `rejected`.
- **`/admin/moderation/organizers`** — pending queue with verify / reject(reason) modal (mirrors the event
  moderation queue). **`/admin/organizers`** — search + detail with revoke(reason), the per-org auto-verify
  toggle, and verification history.
- **`/admin/settings`** — global toggles page; the `organizers.auto_verify_all` switch with a short
  "no moderators? turn this on" explanation. New nav link in the admin `layout.tsx`; pending count on overview.
- **Verified badge** — a small `✓` component on event cards and event detail when `organizer.verified`,
  linking to `/organizers/{id}`.
- **`/organizers/[id]`** — public profile (name, logo, description, website, ✓ badge) + the org's published events.
- `lib/api.ts` client functions for all of the above; reuse the existing `roleResolved` admin gating and
  bearer-attach patterns from `auth-context`.

## 9. Testing

- **Service unit tests** for every transition incl. invalid-transition (409), reason-required (400),
  name-required, and the **auto-approve short-circuit** (global on, per-org on, both off) — mirrors the
  moderation service tests.
- **Repository transition tests** under `//go:build integration` (same convention as moderation; still not
  wired into local CI — a known deferred item from sub-project 0+1).
- **Settings** unit test: `SetBool` writes both the row and the audit entry.
- **Frontend**: `pnpm lint` + `pnpm build` clean; manual end-to-end verify — create profile → submit →
  (a) admin verify → badge appears on the org's events and the public page renders; (b) flip global
  auto-verify on → a fresh submit lands `verified` without admin action.

## 10. Compliance notes (ISO 27001 / Vanta)

- Auto-verify is a **bypass of a detective control** (moderation). Both the global toggle and the per-org
  flag are **audited** (`settings.update`, `organizer.set_auto_verify`), and auto-verifications are logged
  with `{auto:true, source}` — so an access review can always answer "who turned moderation off, when, and
  which orgs skipped it." This is a deliberate, documented control toggle, not a silent gap.
- Public `GET /organizers/{id}` returns only `verified` orgs — pending/rejected profiles (and their owners'
  intent) are not exposed.

## 11. Open calls confirmed during brainstorming
- Linking: 1:1 organizer profile on the user; event badge **derived** from the creator owning a verified org;
  no change to `events.organizer_id`.
- Lifecycle: `pending → verified/rejected` with resubmit + admin revoke.
- UI: all four surfaces in scope (user profile, admin queue/detail, public badge, public organizer page).
- Auto-approve: global runtime toggle **and** per-org flag; applies at submit-time (no retroactive sweep).
- Public page routed by **id**, not slug.
- Editing a verified org does **not** auto-unverify.
