# Email Verification (SendPulse) + Event Invitations — Design

**Date:** 2026-07-15
**Status:** Approved design — ready for implementation planning
**Branch:** `feat/email-verification-and-invitations`

## 1. Problem & Goals

Registration currently stores an email but never proves the user controls it. Email
verification exists in the vendored **GateGuard** auth service only as an explicitly
non-production **STUB** (`gateguard/internal/service/email_verification.go` —
`sendVerificationStub` logs the token, sends nothing). This design replaces the stub
with real email delivery via **SendPulse**, propagates verification status into the Lia
app, gates sensitive Lia actions on it, and adds a new **event-attendance invitation**
feature (organizers invite people by email; unregistered people can be invited; accepting
requires a verified email).

Goals:
1. Send a real **6-digit verification code** on signup / on request, via SendPulse SMTP,
   From `info@tarski.ru`.
2. Make `email_verified` visible to the Lia backend and **block** unverified users from:
   creating/publishing/editing events, RSVP/applying to events, and submitting
   complaints/reports. (Following organizers and all read/browse stay open.)
3. Add **Lia-native event invitations**: organizer invites by email; invitee accepts via
   an email link **or** an in-app pending list; **unregistered** invitees can be invited
   and onboarded; accepting an invite creates/confirms an RSVP.
4. Add a cheap defensive `EmailVerified` check at GateGuard's legacy org-invite accept
   chokepoints (future-proofing only; no UI).

Non-goals: GateGuard org/team invitations UI, Google-OAuth invite flow, SMS/2FA,
personal (non-org) GateGuard referrals.

## 2. Current State (verified)

- **Signup:** `gateguard/internal/service/sign_in_password.go` creates the user with
  `email_verified=false`, generates a random link-style token, calls `sendVerificationStub`
  (log-only), and returns a session JWT immediately. Login does **not** check verification.
- **Verify RPCs already exist:** `RequestEmailVerification` / `VerifyEmail`
  (`gateguard/internal/server/auth_password.go:36-56`); `VerifyEmail` sets
  `email_verified=true` and clears the token on match.
- **Schema present:** migration `gateguard/db/000011_*` added `email_verified`,
  `email_verification_token`, `email_verification_sent_at` to `users`.
  `email_verification_sent_at` exists but is never written today.
- **Working SMTP mailer exists in GateGuard only:** `gateguard/internal/pkg/notificator/`
  (`SMTPNotificator`, Go `net/smtp`, HTML templates). Interface exposes only
  `InviteUserToOrganization`. Config via Viper: `notificator.username/password/from/address`
  (default `smtp.gmail.com:587`). Lia's `/backend/` has **no mailer**.
- **Auth path Lia→GateGuard:** Lia calls GateGuard `CheckAuth`, which returns a `gg.User`
  proto → mapped to `auth.Claims{Subject,Email,Name,Role}` in
  `backend/internal/http/auth/gatekeeper.go`. **The proto `User` has no `email_verified`
  field.** `backend/internal/http/auth/auth.go` provisions/loads a local `domain.User`.
- **Invitations today:** exist **only** inside GateGuard (org-membership, email link →
  Google OAuth auto-accept). Lia backend never calls the invite RPCs; frontend has no
  invite UI. Effectively dormant relative to Lia's password auth.

## 3. Architecture Overview

Three layers, built in three phases:

```
Phase A — Verification email (GateGuard)
  signup / request  ──► SMTPNotificator.SendEmailVerification(email, code)  ──► SendPulse ──► inbox
  user enters code  ──► VerifyEmail  ──► users.email_verified = true

Phase B — Status propagation + action gating (proto + Lia)
  gg.User.email_verified (new proto field)
    └─► CheckAuth ─► auth.Claims.EmailVerified ─► domain.User.EmailVerified
          └─► RequireVerified guard ─► 403 on gated write actions

Phase C — Event invitations (Lia-native, new feature)
  organizer ─► POST /events/{id}/invitations ─► event_invitations row + Lia SMTP email
  invitee (link)   ─► /invite/{token} ─► signup/login ─► verify ─► accept ─► RSVP
  invitee (in-app) ─► GET /me/invitations ─► accept/decline ─► RSVP
```

### 3.1 Mailers

- **Verification email** is sent by **GateGuard** (signup lives there) via its existing
  `notificator`.
- **Invitation email** is a **Lia** feature; Lia backend gains its **own** small SMTP
  sender (mirroring GateGuard's `notificator` shape) under `backend/internal/pkg/mailer/`.
- Both use the **same SendPulse account**, From `info@tarski.ru`. Config via env on each
  service. No cross-service RPC for email.

## 4. Phase A — Verification email (GateGuard)

**Code generation.** Replace the random link-token with a **6-digit numeric code**
(`000000`–`999999`, cryptographically random). Stored in the existing
`email_verification_token` column (string). Stamp `email_verification_sent_at = now()` on
send.

**Sending.**
- Add `SendEmailVerification(ctx, toEmail, code string) error` to the `INotificator`
  interface (`gateguard/internal/pkg/notificator/interface.go`) and implement on
  `SMTPNotificator`.
- New template `email_verification` under
  `gateguard/internal/pkg/notificator/templates/` (Russian copy; shows the 6-digit code
  prominently; states it expires; From `info@tarski.ru`).
- Replace `sendVerificationStub` body to call `SendEmailVerification` (keep the method name
  or rename to `sendVerification`; remove the "STUB" log-only behavior).

**Resend / rate-limit.** `RequestEmailVerification` regenerates a code, sends, and stamps
`email_verification_sent_at`. Enforce a **60-second cooldown** (reject if
`now - email_verification_sent_at < 60s` with a clear error). This is the first real use
of that column.

**Expiry.** A submitted code is valid for **15 minutes** (checked in `VerifyEmail` against
`email_verification_sent_at`); expired codes are rejected and the user can request a new one.

**Config.** Set GateGuard `notificator.address/username/password/from` to SendPulse SMTP +
`info@tarski.ru` via env (no code default change required; documented in runbook).

## 5. Phase B — Status propagation + action gating

**Proto.** Add `bool email_verified = <n>;` to `message User`
(`backend/protocols/gateguard/models_gateguard.proto` and GateGuard's copy). Regenerate
stubs for **both** modules. Set the field in GateGuard's model→proto mapping
(`gateguard/internal/models/user.go`).

**Lia plumbing.**
- Add `EmailVerified bool` to `auth.Claims` and populate it from `u.EmailVerified` in
  `gatekeeper.go:Validate`.
- Add `EmailVerified` to Lia `domain.User` and set it in `auth.go:ensureUser` (kept in sync
  from the claim, same pattern as `role`).
- Add a `RequireVerified(principal) error` guard returning HTTP **403** with body
  `{"code":"email_not_verified"}` when false.

**Gated actions (403 when unverified):**
- Create / publish / edit event (event write handlers, `backend/internal/http/handlers/events.go`)
- RSVP / apply to event (RSVP handlers)
- Submit complaint / report (`backend/internal/http/complaints/handler.go`)

**Not gated:** follow organizers; all read/list/detail/browse; auth endpoints; the verify
and invitation-accept endpoints themselves.

**Mock mode.** In `a.mocked` mode the mock principal is `EmailVerified=true` so local/mock
runs are unaffected.

## 6. Phase C — Event invitations (Lia-native)

### 6.1 Data model (new Lia migration)

Table `event_invitations`:

| column | type | notes |
|---|---|---|
| `id` | uuid PK | |
| `event_id` | uuid FK → events | |
| `inviter_user_id` | uuid FK → users | organizer who sent it |
| `invitee_email` | text NOT NULL | lower-cased; invite keyed by email, not user id |
| `token` | text UNIQUE NOT NULL | random, used by the email link |
| `status` | text | `pending` / `accepted` / `declined` / `revoked` / `expired` |
| `created_at` | timestamptz | |
| `responded_at` | timestamptz NULL | |
| `expires_at` | timestamptz | e.g. 30 days |

Index on `(invitee_email, status)` for the in-app list, and unique on `token`. Optional
partial-unique on `(event_id, invitee_email)` where status = pending to avoid duplicates.

### 6.2 Backend endpoints (Lia)

- `POST /events/{id}/invitations` — organizer sends invite(s) by email. Requires: caller
  owns the event **and** is verified. Creates rows, sends emails via Lia mailer. Skips
  emails already RSVP'd or already pending for that event.
- `GET /invitations/{token}` — **public** preview: returns event summary + invite status,
  for the `/invite/[token]` landing page (no auth; does not leak private data beyond the
  event's public detail).
- `POST /invitations/{token}/accept` / `POST /invitations/{token}/decline` — link-based.
  Requires auth **and** verified email **and** the authenticated user's email matches
  `invitee_email`. Accept → set `accepted`, `responded_at`, and **create/confirm an RSVP**
  for the event via the existing RSVP domain.
- `GET /me/invitations` — pending invitations for the **authenticated user's email**
  (surfaces invites sent before they registered).
- `POST /me/invitations/{id}/accept` / `decline` — in-app accept/decline (same
  verified-email requirement; same RSVP effect).

**Accept effect:** reuse the existing RSVP domain/service — accepting is equivalent to the
user RSVPing to that event. No parallel guest-list table.

**Revoke / expiry:** organizer may `revoke` a pending invite (optional endpoint
`DELETE /events/{id}/invitations/{id}`); a background/expiry check flips overdue `pending`
→ `expired` (mirror GateGuard's `expire_invitations` pattern, or lazy-expire on read).

### 6.3 Onboarding unregistered invitees

- Invite exists by email before any account.
- Link flow: `/invite/{token}` shows the event → user signs up with password (GateGuard
  password signup) → must enter the 6-digit code (Phase A) → once verified, the invite
  (matched by email) auto-accepts and RSVPs.
- Existing logged-in users: invite appears in `/me/invitations` (matched by email);
  accepting is verify-gated.

### 6.4 Frontend

- `frontend/app/auth/verify/page.tsx` — 6-digit code input + resend (cooldown), plus a
  reusable "verify your email" interstitial shown when any gated action returns
  `email_not_verified` (403).
- Lia proxy endpoints: `POST /auth/request-verification`, `POST /auth/verify-email`
  (already planned in the passwords spec; wire to GateGuard RPCs).
- Organizer "Invite by email" panel on the event manage page (enter one or more emails,
  send, see pending/accepted status).
- `/invite/[token]` landing page — event preview → signup/login → verify → accept.
- `/me/invitations` — pending-invites list with accept/decline.

## 7. Phase D — Defensive GateGuard gate (future-proofing)

Add `if !user.EmailVerified { return <error> }` before the Pending→Accepted transition in:
- `gateguard/internal/service/sign_in.go` (OAuth auto-accept path), and
- `gateguard/internal/service/react_to_invitation.go` (explicit accept, `models.Accepted` case).

No UI; guards the legacy org-invite path only.

## 8. Your (operator) manual steps — not automatable

1. Create SendPulse account; add `tarski.ru` as a sending domain.
2. Copy SendPulse SMTP host/port/login/password and the SPF/DKIM values it generates.
3. Add SPF, DKIM, and DMARC records for `tarski.ru` at nic.ru.
4. Provide the SMTP creds to set as env on both GateGuard and Lia (From `info@tarski.ru`).

Verification of delivery (code arrives in Gmail/Yandex/Mail.ru inboxes, not spam) depends
on steps 2–3 being complete.

## 9. Testing

- **GateGuard:** unit tests for 6-digit code generation, `SendEmailVerification` (fake
  notificator), resend cooldown, code expiry, `VerifyEmail` happy/expired/mismatch paths.
- **Lia:** unit tests for `RequireVerified` guard (403 vs pass), claims/domain propagation
  of `EmailVerified`, invitation create/list/accept/decline, unregistered-email matching,
  accept→RSVP effect, ownership + verified enforcement on send.
- **Frontend:** verify page (enter/resend), invite landing accept, in-app pending list.
- **Delivery smoke test:** after DNS+creds, send a real verification + invitation email to
  a Gmail address and confirm inbox placement + DKIM pass.

## 10. Rollout

1. New Lia migration for `event_invitations` (scp `.sql` to box
   `/opt/lia/backend/db/migrations` before migrate, per deploy runbook).
2. Regenerated proto stubs (GateGuard + Lia).
3. Env: SendPulse SMTP creds on GateGuard `notificator.*` and Lia mailer; From
   `info@tarski.ru`.
4. Build-on-Mac → save|ssh|load for both containers; migrate; verify; prune stale Docker.
5. Post-deploy: send test verification + invite; confirm inbox + accept→RSVP.

A dedicated deploy runbook will be written under `docs/superpowers/runbooks/`.

## 11. Open follow-ups / risks

- SendPulse payment is EUR (Visa/MC/PayPal); the free 12k/mo tier needs no payment but
  confirm the account can be created without a foreign-card block. If blocked, fall back to
  Unisender Go (RU billing) — the SMTP wiring is identical.
- Decide whether the organizer sending invites must themselves be verified (design says
  **yes**, since "create/edit event" is already gated).
- `event_invitations` expiry job vs lazy-expire — pick during planning.
