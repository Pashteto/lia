# Admin Suite — Roadmap & Handoff

_Last updated: 2026-06-26. Status: **foundation (sub-projects 0 + 1) built, reviewed, and LIVE on prod.** The rest of the admin suite is unbuilt — this doc is the decomposition + handoff so the next session can pick up cold._

This is the umbrella tracker for the moderator/admin capability. The full vision
lives in `docs/design_agent_prompt.md` §4.3 (admin routes) and
`docs/event_discovery_mvp_technical_stack.md` (modules, tables, roles). That vision
was decomposed into independent sub-projects; each gets its own spec → plan →
implementation cycle (do NOT try to build the rest in one pass).

## Decomposition & status

| # | Sub-project | Status | Depends on |
|---|---|---|---|
| **0** | RBAC + admin shell foundation | ✅ **DONE + LIVE** (merged into 0+1) | — |
| **1** | Event moderation (take-down/reinstate) | ✅ **DONE + LIVE** | 0 |
| **2** | Organizer entity + verification | ⬜ TODO (heaviest — new domain) | 0 |
| **3** | Complaints / reports | ⬜ TODO | 0, 1 |
| **4** | Featured curation | ⬜ TODO | 0, 1 |
| **5** | Taxonomy admin (categories/interests CRUD) | ⬜ TODO | 0 |
| **R** | Moderator/admin role split (GateGuard `moderator`) | ⬜ TODO (cross-cutting) | 0 |

Spec for 0+1: [`../specs/2026-06-26-moderation-admin-foundation-design.md`](../specs/2026-06-26-moderation-admin-foundation-design.md).
Plan for 0+1: [`2026-06-26-moderation-admin-foundation.md`](2026-06-26-moderation-admin-foundation.md).
Deploy: [`../runbooks/2026-06-26-rsvp-moderation-fullstack-deploy.md`](../runbooks/2026-06-26-rsvp-moderation-fullstack-deploy.md).

## What's DONE (sub-projects 0 + 1) — live on https://lia.pashteto.com

- **RBAC via GateGuard's existing `admin` role** (Approach 1 — no GateGuard reship).
  Lia stopped discarding the role in `gatekeeper.go`; `Auth.Authenticate` returns the
  domain user with `.Role`, synced to `users.role` (migration `000014`) on **every**
  request (GateGuard is source of truth, `users.role` is a cache). A demoted admin
  loses access on the next request.
- **Admin gate** — `internal/http/admin` is a plain `net/http` handler mounted ahead
  of the go-swagger mux (mirrors `internal/http/uploads`). `requireStaff` allows
  `role == "admin"`; else 403 «Недостаточно прав»; 401 anon. No swagger-spec edits.
- **Endpoints** (all admin-gated except `/auth/me`): `GET /auth/me`,
  `GET /api/v1/admin/overview`, `GET /api/v1/admin/moderation/events?status=published|rejected`,
  `POST …/{id}/takedown` `{reason}`, `POST …/{id}/reinstate`.
- **Moderation domain** (`internal/moderation`): post-moderation take-down
  (`published → rejected`, reason required) / reinstate (`rejected → published`),
  each writing one `event_status_history` + one `audit_log` row in one tx;
  `ErrInvalidTransition` (409) when not in the expected status; `Counts` for overview.
- **Frontend**: gated `/admin` shell (4-state gate via `roleResolved`), overview,
  moderation queue (Опубликованные/Снятые tabs + reason modal + reinstate), «Админ»
  nav link in `AuthButton`, «Снято модератором» badge on `/events/mine`.
- **Bootstrap**: roles set manually in GateGuard's `users` table
  (`UPDATE users SET role='admin' WHERE email=…`). `poulissimo@gmail.com` is the
  current live admin (seeded 2026-06-26).

## Deferred follow-ups from 0+1 (small; do when convenient)

1. **`/events/mine` does not thread the take-down reason** — the organizer sees the
   «Снято модератором» badge but not WHY. Backend: add a `moderation: {reason}` block
   to the `/events/mine` event response (it already has `moderation.LatestReason`);
   frontend: render it under the badge. (Spec §10.1.)
2. **No in-app role-promotion UI** — promotion is manual SQL on the `gateguard` DB.
   A `/admin/users` page + a Lia endpoint proxying GateGuard's `UpdateUserRole` RPC
   would remove the manual step. (This is partly sub-project for "user management".)
3. **No audit-log viewer** — `audit_log` accrues rows but nothing surfaces them.
   A read-only `/admin/audit` list is a natural small add (and an ISO 27001 nicety).
4. **Overview has no loading skeleton**; **moderation queue has no pagination**
   (fine at demo volume; revisit if data grows).
5. **Moderation integration tests** (`//go:build integration`) are not run in this
   repo's local flow — wire them into CI with a migrated test DB.

## Remaining sub-projects (each needs its own brainstorm → spec → plan)

### #2 Organizer entity + verification  — heaviest, do first if continuing
Today "organizer" is just the user who created an event (events carry a derived
`organizer` read-model; there is NO `organizers` table). Real org verification needs:
- `organizers` + `organizer_members` tables (membership model — one org, many staff),
  `verification_status` (pending/verified/rejected). Events reference an organizer.
- Migration; `internal/organizers` domain; admin endpoints for the verification queue.
- `/admin/moderation/organizers` (verification queue) + `/admin/organizers` (search/detail).
- Ripples into event attribution (events → organizer FK instead of bare user).
**This is a domain-modelling project, not a thin slice — scope it carefully.**

### #3 Complaints / reports
- `complaints` table (target_type/target_id, reporter, reason, status, resolution).
- Public "Пожаловаться" action on event detail → `POST` complaint (behind auth).
- `/admin/complaints` inbox + resolve actions (which can trigger a take-down — reuse
  the moderation module's `Takedown`). Audit each resolution.

### #4 Featured curation
- A `featured` flag/ordering on events (migration), admin endpoints to set/clear/order.
- `/admin/featured` manual curation UI; surface a "featured" rail on Discovery.

### #5 Taxonomy admin
- CRUD UIs over the existing normalized `categories` (+ future `interests`) tables.
- `/admin/categories`, `/admin/interests`; admin-gated CRUD endpoints (the tables
  already exist from category/venue normalization — this is mostly UI + endpoints).

### #R Moderator/admin role split (cross-cutting)
0+1 deliberately reused GateGuard's existing `admin` role for ALL staff (Approach 1)
to avoid reshipping GateGuard. When you need a distinct, lower-privilege **moderator**:
- Add `moderator` to GateGuard's enum (`gateguard/internal/models/user_role.go`) +
  proto (value 5), regen **both** GateGuard's pb and Lia's vendored
  `protocols/gateguard` pb, and **rebuild + ship GateGuard to the box** (the painful
  part — Go 1.26 base, `docker save|ssh|load`; see the gateguard deploy runbook).
- Widen Lia's `IsStaff()` / gate to distinguish `moderator` (moderation only) from
  `admin` (everything). The single seam is `Authenticate`/`normalizeRole` + the gate.

## Recommended build order
0+1 (done) → **#2** (unblocks real organizer concept that #3/#4 lean on) → #3 →
#4 / #5 (independent, any order) → #R when the moderator/admin distinction is actually needed.

## Operational state (as of 2026-06-26)
- All of the above (RSVP + 0+1 + two hotfixes) is on **local `main`, UNPUSHED**
  (`origin/main` behind). Deployed to prod from locally-built images, not a pushed ref/PR.
- Two hotfixes landed during deploy: GateGuard wiring (recreate `app` with ALL THREE
  compose files incl. `-f docker-compose.gateguard.yml` — see deploy runbook) and
  `signup_mode` default `open` on event create.
- **Rotate `GATEGUARD_AUTH_SECRET`** — it was exposed in a session transcript on
  2026-06-26; rotating invalidates issued tokens (users re-login).
- One leftover prod draft event `c1dc7707-…` (owner `deploy-smoke@presence.test`,
  invisible) — smoke-test artifact, safe to delete.
