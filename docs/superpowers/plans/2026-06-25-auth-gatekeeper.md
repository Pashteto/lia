# Auth (GateGuard) Implementation Plan

> **For agentic workers:** use superpowers:test-driven-development for code steps.
> Steps use checkbox (`- [ ]`) syntax. Spec: [`../specs/2026-06-25-auth-gatekeeper-design.md`](../specs/2026-06-25-auth-gatekeeper-design.md).

**Goal:** Real authentication for Lia via gateway.fm's **GateGuard** (self-hosted
on the box). Lia is a resource server: it validates GateGuard-issued JWTs and
maps them to local users; GateGuard owns token issuance. Demo login = **email →
`SignInOAuth` → cookie** (no Google).

**Architecture:** `internal/http/auth.CheckAuth` (the swagger `JwtAuth` seam)
calls `GateguardService.CheckAuth` over gRPC behind a `TokenValidator` interface,
JIT-provisions a local `users` row by email, returns the principal. `POST /events`
requires `jwt` (organizer = principal). GateGuard runs as its own container
(reusing Lia's Postgres) on Lia's compose network.

**Status:** Phase A ✅ done · Phase B ✅ done · Phase C ⏳ pending. All on branch
`feat/dark-mode-theme`.

---

## Phase A — Backend validation (DONE ✅, commits 11e0072 + b9a570b)

- [x] `TokenValidator` interface + `Claims`; `Auth` gains a validator via
  `WithValidator` option (existing 3-arg `NewAuth` callers unchanged).
- [x] `CheckAuth`: non-mock → validate token → JIT-provision local user by email
  (`GetUserByEmail`/`CreateUser`) → principal; invalid/missing/unconfigured → 401;
  logs decisions by subject only (no token/PII). TDD'd.
- [x] Vendor GateGuard protos → `backend/protocols/gateguard` (go_package repointed
  to this module); generate client with `protoc` (buf not needed); force-commit
  the `*.pb.go` (overrides `*.pb.go` gitignore — non-conforming third-party protos).
- [x] `gatekeeperValidator.Validate` calls `GateguardService.CheckAuth(TokenRequest)`
  → maps `User → Claims` (subject from uuid bytes, email, name). Lazy gRPC dial,
  insecure transport (on-box). TDD'd with a fake gRPC client.
- [x] `module.go` wires the validator from `http.gatekeeper` config when
  `MockAuth=false` (returns error on dial failure).
- [x] `POST /events` requires `jwt` (swagger `security`), regen'd; handler takes
  the principal and sets `organizer_id` from it, ignoring client-supplied value.
  TDD'd. `build/vet/test` green (22 pkgs).

## Phase B — Deploy GateGuard on vds-ru215 (DONE ✅, commits aa9b778 + af5858d)

- [x] `backend/docker-compose.gateguard.yml`: `gateguard` (gRPC :9090, internal)
  + `gateguard-redis` on Lia's `backend_default` network; reuses Lia's Postgres
  (separate `gateguard` DB); wires `app` `HTTP_GATEKEEPER_ADDRESS` (MockAuth stays true).
- [x] `GATEGUARD_AUTH_SECRET` (random) added to `.env.prod`.
- [x] rsync GateGuard source → `/opt/gateguard`; build `gateguard:local` on box
  (force IPv4 on the Dockerfile `curl`s — box IPv6 to github is broken).
- [x] Create `gateguard` DB in Lia's Postgres; run GateGuard's 10 migrations.
- [x] Bring up; verified: GateGuard connects to Postgres + Redis, gRPC serving on
  :9090, reachable as `gateguard:9090` from Lia's network. Live demo unchanged.
- Runbook: [`../runbooks/2026-06-25-gateguard-deploy.md`](../runbooks/2026-06-25-gateguard-deploy.md).

## Phase C — Demo-login + flip (PENDING ⏳)

### Task 1: Backend demo-login entrypoint — DONE (Lia side), BLOCKED on GateGuard bug
- [x] Swagger `POST /auth/demo-login` (open) + `auth.Signer` wrapping GateGuard
  `SignInOAuth` + `handlers.DemoLogin`. TDD'd, deployed. **Decided: swagger endpoint;
  token returned in body (frontend stores it) — not an httpOnly cookie (demo scope).**
- [ ] ⛔ **BLOCKED:** GateGuard's `SignInOAuth` panics (`index out of range [2]`),
  so demo-login returns 503 and **create-event is currently broken on the live demo**
  (auth is enforced but no token can be minted). Full triage + revert command +
  next steps: **[`../runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md`](../runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md)**.

### Task 2: Frontend login flow
- [ ] Replace the disabled "Войти" stub with a demo-login form (email[, name]).
- [ ] On submit → call the demo-login endpoint → store cookie → authed UI state
  (show identity + "Выйти" in the nav).
- [ ] Attach `Authorization: Bearer` (from the cookie, server-side) to write calls
  in `lib/api` (`createEvent`); gate `/events/new` (redirect anon → login).
- [ ] Sign-out clears the cookie.
- [ ] Verify with Playwright: login → create event → organizer attributed.

### Task 3: Flip to real auth
- [ ] Set `HTTP_MOCK_AUTH=false` in `.env.prod`, redeploy `app`.
- [ ] Verify end-to-end: `POST /events` 401 anonymous, 201 after demo-login;
  GET discovery still open.
- [ ] ⚠️ Document the access-control change (ISO 27001 / Vanta). Update HANDOFF +
  deploy memory (MockAuth retired).

## Global constraints / gotchas
- After any `backend/api/swagger.yaml` change run `make generate-api` (go-swagger
  code is gitignored). golangci-lint is **v1** in CI.
- Box: flaky SSH, 1.9 GB RAM, broken IPv6 (force IPv4 for github in Docker builds).
- GateGuard `organizations` client points at `127.0.0.1:9097` (lazy); demo-login by
  email (no refCode) shouldn't touch it — stub/disable if `SignInOAuth` calls it.
