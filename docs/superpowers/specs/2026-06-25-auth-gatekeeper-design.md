# Auth slice — Gatekeeper integration (design)

_Date: 2026-06-25. Status: design, pending Gatekeeper-contract confirmation (see §9)._

## 1. Goal

Replace the demo's `HTTP_MOCK_AUTH=true` bypass with real authentication so that
write actions (create event, and — in later slices — RSVP and image upload) are
attributable to a signed-in user. This is the prerequisite slice that currently
blocks the "Войти" button, RSVP, and image upload.

**Decision (confirmed):** authenticate via gateway.fm's **Gatekeeper** — the
path the template is already wired for. Lia is a **resource server**: Gatekeeper
owns login/signup/token issuance; Lia only *validates* the Bearer JWT per request
and maps it to a local user. **Login flow:** Gatekeeper **hosted redirect /
OIDC** — the frontend redirects to Gatekeeper's hosted login and receives a token
on callback.

This is a **security-surface change** touching access controls; it must be
documented for ISO 27001 / Vanta (see §7).

## 2. Scope

**In scope**
- Backend: finish `internal/http/auth.CheckAuth` to validate JWTs via Gatekeeper.
- Just-in-time (JIT) provisioning of a local `users` row per Gatekeeper identity.
- Protect `POST /events` (organizer = current principal); keep discovery reads open.
- Frontend: OIDC redirect login, callback handling, token storage, authed UI
  state, gated create-event page, sign-out.
- Config/rollout: turn `MockAuth` off in prod; secrets handling.

**Out of scope (YAGNI — each its own later slice)**
- RSVP module.
- Image upload (the original request that led here).
- Roles/permissions beyond the existing `AdminEmails` admin check.
- Profile editing UI.

## 3. Current state (verified in repo)

- `internal/http/module.go`: `api.JwtAuth = m.auth.CheckAuth` — swagger Bearer
  security handler seam already in place.
- `internal/http/auth/auth.go`: `CheckAuth` returns the mock user when
  `mocked==true`, otherwise has a `TODO: integrate gatekeeper` and returns 401.
  `IsAdmin(email)` checks `AdminEmails`.
- `config/scheme.go`: `GatekeeperConfig{ Address, Timeout }`; defaults
  `http.gatekeeper.address=localhost:9091`, `timeout=5s`. `MockAuth bool`.
- `internal/models/user.go` + a `users` go-pg table exist; `internal/users` is a
  doc-only skeleton. `protocols/userservice/user.proto` is the app's OWN user
  service (not Gatekeeper).
- **No Gatekeeper client, proto, or go.mod dependency exists yet.**
- Most operations declare `security: []` (open); create-event declares `jwt`.

## 4. Architecture & data flow

### 4.1 Login (frontend ↔ Gatekeeper, OIDC redirect)
```
User clicks "Войти"
  → frontend redirects to Gatekeeper hosted login (with redirect_uri back to Lia)
  → user authenticates at Gatekeeper
  → Gatekeeper redirects to Lia callback with an auth code / token
  → Next route handler (server-side) finalizes and sets an httpOnly session cookie
  → subsequent API calls carry the JWT (see §6.3)
```

### 4.2 Authenticated request (frontend → Lia backend)
```
Request with Authorization: Bearer <jwt>
  → go-swagger invokes JwtAuth = CheckAuth(token)
  → CheckAuth calls Gatekeeper validate (gRPC) → claims {subject,email,name,exp,...}
  → JIT-provision/lookup local users row by subject → attach local user_id
  → returns *models.User principal → handler runs with principal
  → invalid/expired/missing → 401
```

## 5. Backend design

### 5.1 `CheckAuth`
- Strip `Bearer ` prefix; empty → 401.
- Call Gatekeeper validation (gRPC client built from `GatekeeperConfig`,
  context with `Timeout`). On a valid response, build `models.User` from claims
  (subject, email, name; status "active").
- Map all Gatekeeper validation failures / transport errors to a 401
  `errors.New(401, ...)` (do not leak transport detail to the client; log
  server-side without the token).
- Keep `mocked` short-circuit for local dev/tests.

### 5.2 Gatekeeper client
- New `internal/gatekeeper` (or `pkg/gatekeeper`) thin wrapper around the
  Gatekeeper gRPC stub. Built once at module init from config; injected into
  `Auth`. TLS for the gRPC channel in prod.
- **Assumed contract (confirm — §9):** a unary `ValidateToken(token) →
  {subject, email, name, expires_at, roles?}`; non-OK status ⇒ invalid.

### 5.3 JIT user provisioning
- On a valid token, upsert a `users` row keyed by Gatekeeper **subject** (stable
  id), storing email/name, `status=active`. Returns the local user UUID.
- Uses existing repository/service layers. Honors the go-pg zero-UUID convention
  already used for `organizer_id`/`venue_id` (no nullable-UUID scans).
- `POST /events` sets `organizer_id = principal.localUserID`.

### 5.4 Protected routes
- Confirm create-event requires `jwt` in the spec; discovery/list/detail/
  categories/venues/nearby stay `security: []`.
- Pattern established here is reused by future RSVP/upload slices.

### 5.5 Config / rollout
- Local/dev/tests: `MockAuth=true` (unchanged).
- **Prod: `HTTP_MOCK_AUTH=false`** in `.env.prod` — retires the documented
  non-prod control. Set `http.gatekeeper.address` to the real instance + TLS.
- Secrets (any Gatekeeper service credential, OIDC client secret, cookie signing
  key): `.env.prod` chmod 600, gitignored on the demo box; **AWS Secrets Manager
  for real prod** (per org standard). Never logged, never committed.

## 6. Frontend design

### 6.1 Login (`Войти`)
- Replace the disabled stub with a control that initiates the Gatekeeper OIDC
  redirect (`redirect_uri` → Lia callback). Carry a CSRF `state` param.

### 6.2 Callback
- A Next **route handler** (server-side) receives the redirect, finalizes the
  token exchange with Gatekeeper, validates `state`, and sets an **httpOnly,
  Secure, SameSite=Lax** session cookie. Then redirects to the originating page.

### 6.3 Attaching the token
- Server components / route handlers read the cookie and forward
  `Authorization: Bearer` to the Lia API. Client components call same-origin Next
  route handlers that attach it server-side (token never exposed to JS — XSS-safe).
- Update `lib/api` write calls (e.g. `createEvent`) to go through the authed path.

### 6.4 UI state & gating
- Nav shows signed-in identity + "Выйти" when authed; "Войти" otherwise.
- `/events/new` redirects anonymous users to login, then back.
- Sign-out clears the cookie (route handler) and resets UI.

## 7. Security & compliance (ISO 27001 / Vanta)

- Document: what is protected, how tokens are validated (Gatekeeper), where
  secrets live, and that `MockAuth` is OFF in prod. This affects access reviews
  and change management — flag for the audit trail.
- Log auth **decisions** (allow/deny + subject id) for audit; **never** log
  tokens, cookies, or PII beyond the subject id.
- httpOnly + Secure + SameSite cookies; CSRF `state` on the OIDC round-trip.
- gRPC to Gatekeeper over TLS in prod.
- Demo box remains a documented hand-managed exception; real prod uses Secrets
  Manager + Terraform-managed config.

## 8. Testing

- Unit: `CheckAuth` with a faked Gatekeeper client — valid → principal;
  expired/invalid/missing/transport-error → 401. JIT provisioning upsert (new vs
  existing subject).
- Integration: protected route returns 401 anonymous, 200 with a valid (faked)
  token; reuse the existing `MockAuth` test harness pattern.
- Frontend: Playwright smoke of the redirect → callback → cookie-set → authed-nav
  path (Gatekeeper stubbed/mocked in the test env).
- CI parity: `go build/vet/test`, golangci-lint v1, `pnpm lint`/`build`.

## 9. Dependencies to confirm (Gatekeeper-side facts, not yet in repo)

These are external to this repo and must be confirmed before implementation; the
spec assumes a sensible default for each so design can proceed:

1. **Client/contract** — the Gatekeeper Go client / proto module path (or gRPC
   contract). _Assumption:_ a gRPC `ValidateToken` as in §5.2.
2. **Reachable instance** — a Gatekeeper instance the demo box can reach
   (address + TLS), or this stays design-for-later with prod wiring deferred.
   _Assumption:_ reachable at a configured `address`; if not, ship the code with
   `MockAuth=true` retained until an instance exists.
3. **OIDC details** — Gatekeeper's authorize/token endpoints, client id/secret,
   and redirect-uri registration for `lia.pashteto.com`. _Assumption:_ standard
   OIDC authorization-code flow.

If any assumption is wrong, the affected section (§5.2 / §6.1–6.2) is revised
before the implementation plan is finalized.

## 10. Rollout sequence

1. Land backend `CheckAuth` + Gatekeeper client + JIT provisioning behind tests
   (`MockAuth` still true everywhere).
2. Wire frontend OIDC redirect/callback/cookie against a confirmed Gatekeeper.
3. Register the redirect-uri; put secrets in `.env.prod`.
4. Flip `HTTP_MOCK_AUTH=false` in prod; verify protected route 401→200 end to end.
5. Update HANDOFF + deploy memory (MockAuth retired; auth = Gatekeeper).
