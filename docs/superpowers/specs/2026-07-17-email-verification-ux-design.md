# Email verification UX — design

_2026-07-17. Fixes the discovery hole in the email-verification feature shipped
2026-07-16 (runbook `runbooks/2026-07-16-email-verification-invitations-deploy.md`).
Target: `presence.tarski.ru`._

## Problem

Verification is **purely punitive**: nothing announces it, and you only learn it
exists by being blocked. Observed firsthand by the product owner, who registered on
prod, received the code, and saw no sign anywhere on the site that an email was coming.

Confirmed in code:

1. **Nothing proactively points to `/auth/verify`.** It is reachable only from
   `VerifyEmailInterstitial` (shown when a gated action is attempted — RSVP via
   `SignupCTA`, report via `ReportButton`, invite via `InviteByEmailPanel`), a redirect
   from `/me/invitations`, or an `/invite/[token]` link. Every path requires already
   being blocked.
2. **Signup says nothing.** `AuthButton.tsx` modal copy never mentions email; on
   success it calls `onClose()` (`AuthButton.tsx:104`) — no toast, redirect, or
   confirmation. The modal simply closes.
3. **No `/me` root page exists** (`app/me/` has `calendar`, `organizer`,
   `applications`, `practices`, `invitations` — no `page.tsx`), so there is nowhere a
   persistent status could live.
4. **`verificationCodeTTL = 15 * time.Minute`** starts ticking on a code the user was
   never told to expect. If they don't check mail immediately, it dies before they
   know it exists.

**Also found while specifying this:** `VerifyEmail`
(`gateguard/internal/service/email_verification.go:70-81`) has **no attempt limiter**.
It compares the code and returns; nothing counts or caps wrong guesses. A 6-digit code
is 1,000,000 combinations, so **the 15-minute TTL is currently the only thing bounding
brute force**. Extending the TTL without a cap would widen that window 96×.

## Decisions

| Decision | Choice | Why |
|---|---|---|
| Role of verification | **Soft gate + always-visible status** | Keeps the current permissive model; fixes only the discovery hole. Gates stay exactly where they are. |
| TTL | **15 min → 24 h**, paired with an attempt limiter | Generous for real users; the guess cap — not the clock — does the security work. |
| Attempt cap | **5 wrong guesses, then the code dies** | Brute-force odds become 5/1,000,000 regardless of TTL. |
| Profile page | **Not now** | The banner already answers "does the site tell me?". A profile page would say the same thing twice. Revisit when it has more than one status row to show. |
| Banner dismissible | **No** | It is the only proactive signal, it is one slim row, and it has a clear exit. Dismissible recreates the original problem. |

## Design

### 1. Signup confirmation (`components/AuthButton.tsx`)

On **register** success only, the modal switches to a confirmation state instead of
closing:

```
Проверьте почту
Мы отправили 6-значный код на pavel@gmail.com
[Ввести код]   [Позже]
```

- **[Ввести код]** → `/auth/verify`
- **[Позже]** → closes the modal (the banner then carries the signal)
- **Login is unchanged** — it still closes silently.

This is the core fix: it addresses the exact moment the user got nothing.

### 2. Persistent banner (`components/VerifyEmailBanner.tsx`, new)

Rendered globally in the app layout. Visible when `isAuthed && ready && !emailVerified`.

```
[!] Почта не подтверждена          [Подтвердить]
```

- **[Подтвердить]** → `/auth/verify`
- **Not dismissible** while unverified.
- Reads `emailVerified` from `lib/auth-context` — already exposed globally
  (`auth-context.tsx:34`), no new plumbing.
- **Disappears by itself**: `/auth/verify` already calls `refresh()` on success
  (`app/auth/verify/page.tsx:38`), which re-fetches `/auth/me` and flips
  `emailVerified`. No extra wiring, no manual invalidation.
- Must not render while `ready` is false, to avoid flashing on first paint before
  auth state resolves.

### 3. TTL + attempt limiter (`gateguard/internal/service/email_verification.go`)

- `verificationCodeTTL`: `15 * time.Minute` → `24 * time.Hour`
- New: `verificationMaxAttempts = 5`
- New error: `ErrVerificationTooManyAttempts`
- `verificationResendCooldown` stays at **60s** — it is what prevents resend-farming
  from refilling the attempt budget.

**`VerifyEmail` logic:**

| Case | Behaviour |
|---|---|
| Code matches, within TTL, attempts < 5 | Verify. Clear token. **Reset attempts to 0.** |
| Code wrong, attempts < 5 | **Increment attempts.** Return `ErrVerificationTokenInvalid`. |
| Code wrong, attempts reaches 5 | **Clear the token** (code dies). Return `ErrVerificationTooManyAttempts`. |
| Attempts already ≥ 5 | Return `ErrVerificationTooManyAttempts` — **even if the code is correct.** |
| Past 24 h | Return `ErrVerificationCodeExpired` (existing). |

**Check order is load-bearing — implement exactly this sequence:**

1. **`attempts >= verificationMaxAttempts` → return `ErrVerificationTooManyAttempts`.**
   This MUST come **before** the token comparison. Lockout clears the token, so once
   locked out `user.EmailVerificationToken == ""` and the existing guard
   (`if token == "" || user.EmailVerificationToken != token`,
   `email_verification.go:75`) would return `ErrVerificationTokenInvalid` instead —
   telling a locked-out user "wrong code" forever, with no hint that resending is the
   way out. That is the exact class of dead-end this spec exists to remove.
2. Token comparison → on mismatch, increment (and clear on reaching the cap).
3. TTL check.
4. Success.

Note the existing TTL check (`email_verification.go:78-81`) currently runs *after* the
token comparison, so an expired-but-correct code reports expired — that ordering is
already right and does not change.

**`RequestEmailVerification` (resend)** already mints a new code and resets
`sent_at`; it must additionally **reset attempts to 0**, so a locked-out user recovers
by resending. Ordering: the existing 60s cooldown check runs first and is unchanged.

### 4. Schema (`gateguard/db/000012_add_verification_attempts.{up,down}.sql`, new)

GateGuard migrations are **flat in `db/`** (not `db/migrations/`); latest is
`000011_add_password_and_email_verification`.

```sql
-- up
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_attempts int NOT NULL DEFAULT 0;
-- down
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_attempts;
```

Add `EmailVerificationAttempts int` with tag `pg:"email_verification_attempts,use_zero"`
to `internal/models/user.go` (mirroring `EmailVerified`). `use_zero` matters — without
it go-pg omits zero values and the counter will not persist a reset to 0.

Every `UpdateUserBy` call that changes the counter must name
`email_verification_attempts` in its column list — the repo updates only named columns.

### 5. Error surfacing (`app/auth/verify/page.tsx`)

Today the page renders whatever message comes back. It must distinguish three states,
or the lockout is invisible and reads as a broken form:

| Condition | Message |
|---|---|
| Expired (>24 h) | «Код истёк. Запросите новый.» |
| Invalid digits | «Неверный код.» |
| Too many attempts | «Код заблокирован после 5 попыток. Запросите новый.» |

This requires the gRPC/HTTP error to carry a distinguishable code through
`backend/internal/http/authverify/handler.go` → `lib/api.ts`, in the same shape as the
existing `email_not_verified` code used by `verified_gate.go:12`.

## Data flow

```
register → gateguard SignUpWithPassword → code emailed (SendPulse)
         → modal shows "код отправлен на <email>"   [NEW]
         → banner visible on every page              [NEW]
/auth/verify → verifyEmail() → refresh() → emailVerified=true
         → banner unmounts automatically
```

## Testing

**GateGuard (unit, run with `-vet=off` — pre-existing printf false-positive):**
- TTL boundary: just under 24 h verifies; just over returns expired.
- 5 wrong attempts → token cleared; 6th returns `ErrVerificationTooManyAttempts`
  **even with the correct code** — this test pins the check order in §3 and fails if
  the attempts check is placed after the token comparison.
- Resend resets attempts to 0 and issues a new code.
- Successful verify resets attempts to 0.
- Resend cooldown (60s) still enforced and unchanged.

**Frontend:**
- Banner renders when `emailVerified === false`, not when true, and **not while
  `ready === false`**.
- Register shows the confirmation state; login does not.
- The three error messages map to the three backend conditions.

## Scope

Two frontend components (one new, one modified), one gateguard migration + model
field, one gateguard service file, error plumbing, plus tests. **No profile page. No
change to which actions are gated.**

## Deploy notes

- The migration targets the **`gateguard` database**, not Lia's `020` chain:
  `docker run --rm --network backend_default -v /opt/gateguard/db:/db migrate/migrate:v4.17.1
  -path=/db/ -database "postgresql://…@postgres:5432/gateguard?sslmode=disable" up`
- **Sync `1-lia/gateguard/` → box `/opt/gateguard/` first** — the box's copy has gone
  stale before, and **build gateguard on the Mac**: building on the box fails on a
  github TLS error. Both traps are documented in
  `runbooks/2026-07-16-email-verification-invitations-deploy.md`.
- Frontend rebuild needs **both** build-args (`NEXT_PUBLIC_API_URL` +
  `NEXT_PUBLIC_YANDEX_MAPS_KEY`) or the site degrades and maps break.

## Out of scope / follow-ups

- `/me` root profile page (revisit when it has more to show).
- The two deferred invitation bugs (handoff §7): expiry not enforced on accept;
  re-inviting a pending email emails a dead token.
- Rate-limiting `RequestEmailVerification` per-IP (the 60s per-account cooldown is the
  only limiter today).
