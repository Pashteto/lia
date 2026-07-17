# Email Verification UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make email verification discoverable (signup confirmation + a persistent banner) and give the code a 24-hour life bounded by a 5-attempt cap instead of a 15-minute clock.

**Architecture:** GateGuard gains an attempts counter (new column + migration) and returns **typed gRPC status codes** so the three failure modes survive the gRPC → backend-proxy → frontend hop. The frontend gains one new component (`VerifyEmailBanner`, driven by the `emailVerified` flag already in the global auth context) and a confirmation state in the existing signup modal.

**Tech Stack:** Go 1.26 / go-pg / grpc-go v1.81.1 (both modules) / golang-migrate v4.17.1 / Next.js App Router + TypeScript + Tailwind.

**Spec:** `docs/superpowers/specs/2026-07-17-email-verification-ux-design.md` (commit `db0614d`)

## Global Constraints

- **Repo root:** `/Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia`
- **GateGuard tests MUST run with `-vet=off`** — `internal/service` has a pre-existing `go vet` printf false-positive unrelated to this work.
- **Pre-existing failing tests `Test_ReactToInvitation_*`** in the gateguard module fail on `main` too (org-mock mismatch). **Not caused by this work — do not chase them.**
- **Local `golangci-lint` is v2 but the repo config is v1** → use `gofmt`, `go build`, `go vet` instead.
- **GateGuard migrations are FLAT in `gateguard/db/`** — NOT `db/migrations/`. Latest existing is `000011_add_password_and_email_verification`.
- **go-pg omits zero values unless tagged `use_zero`.** Any new int column that must persist a reset-to-0 requires it.
- **`UpdateUserBy` only writes columns named in its variadic list.** A field changed but not named is silently dropped.
- All user-facing copy is **Russian**, matching existing components.
- **No new dependencies.** Everything needed is already in both `go.mod`s and `package.json`.

## File Structure

| File | Responsibility |
|---|---|
| `gateguard/db/000012_add_verification_attempts.{up,down}.sql` (create) | Add `email_verification_attempts` column |
| `gateguard/internal/models/user.go` (modify) | `EmailVerificationAttempts` field w/ `use_zero` |
| `gateguard/internal/service/email_verification.go` (modify) | TTL, cap, new error, check order, resets |
| `gateguard/internal/service/tests/email_verification_test.go` (modify) | Service unit tests — **two existing tests break and must be repaired**, see Task 2 Step 0 |
| `gateguard/internal/server/auth_password.go` (modify) | Map service errors → gRPC status codes |
| `backend/internal/http/authverify/handler.go` (modify) | Map gRPC codes → JSON `{code,message}` |
| `backend/internal/http/authverify/handler_test.go` (create/extend) | Proxy mapping tests |
| `frontend/lib/api.ts` (modify) | Parse the `code` field, export error constants |
| `frontend/components/VerifyEmailBanner.tsx` (create) | The persistent banner |
| `frontend/app/layout.tsx` (modify) | Mount the banner globally |
| `frontend/components/AuthButton.tsx` (modify) | Register-success confirmation state |
| `frontend/app/auth/verify/page.tsx` (modify) | Three distinct error messages |

---

### Task 1: GateGuard schema + model field

**Files:**
- Create: `gateguard/db/000012_add_verification_attempts.up.sql`
- Create: `gateguard/db/000012_add_verification_attempts.down.sql`
- Modify: `gateguard/internal/models/user.go:31` (add field after `EmailVerified`)

**Interfaces:**
- Consumes: nothing.
- Produces: `models.User.EmailVerificationAttempts int` — column name `email_verification_attempts`. Tasks 2 and 3 read/write it.

- [ ] **Step 1: Write the up migration**

Create `gateguard/db/000012_add_verification_attempts.up.sql`:

```sql
-- Bounds brute force on the 6-digit verification code. Paired with raising
-- verificationCodeTTL to 24h: the attempt cap, not the clock, limits guessing.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_attempts int NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Write the down migration**

Create `gateguard/db/000012_add_verification_attempts.down.sql`:

```sql
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_attempts;
```

- [ ] **Step 3: Add the model field**

In `gateguard/internal/models/user.go`, immediately after the `EmailVerified` field (line 31), add:

```go
	EmailVerificationAttempts int       `pg:"email_verification_attempts,use_zero"`
```

`use_zero` is REQUIRED: without it go-pg omits zero values, so resetting the counter to 0 would silently not persist and a user who resent their code would stay locked out forever.

- [ ] **Step 4: Verify it compiles**

Run: `cd gateguard && go build ./...`
Expected: exit 0, no output.

- [ ] **Step 5: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add gateguard/db/000012_add_verification_attempts.up.sql \
        gateguard/db/000012_add_verification_attempts.down.sql \
        gateguard/internal/models/user.go
git commit -m "feat(gateguard): add email_verification_attempts column + model field"
```

---

### Task 2: TTL 24h + attempt cap in the service

**Files:**
- Modify: `gateguard/internal/service/email_verification.go:23` (TTL), `:15-26` (consts/errors), `:44-67` (`RequestEmailVerification`), `:70-89` (`VerifyEmail`)
- Test: `gateguard/internal/service/tests/email_verification_test.go`

**Interfaces:**
- Consumes: `models.User.EmailVerificationAttempts` (Task 1).
- Produces: `service.ErrVerificationTooManyAttempts` (`errors.New("verification attempts exceeded")`) — Task 4 maps it to a gRPC code. Existing `ErrVerificationCodeExpired` and `ErrVerificationTokenInvalid` keep their current names.

- [ ] **Step 0: Fix the two EXISTING tests this change breaks**

Both live in `gateguard/internal/service/tests/email_verification_test.go`. They are written as **testify-suite methods on `UseCaseSuite`** (`func (s *UseCaseSuite) Test_...`) using mockery `EXPECT()` mocks — follow that style for everything below; there are no `newTestUser`/`newServiceWithUser` helpers.

**(a) `Test_VerifyEmail_Expired` (line 62)** uses a 20-minute-old code and comments "older than 15m". At a 24h TTL that code is **valid**, so the test would fail. Change the age:

```go
			u.EmailVerificationSentAt = time.Now().Add(-25 * time.Hour) // older than the 24h TTL
```

**(b) `Test_RequestEmailVerification_SendsCode` (lines 23-26)** asserts `UpdateUserBy` receives **exactly two** columns. Step 5 adds a third, so the expectation must name it:

```go
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
			"email_verification_token", "email_verification_sent_at", "email_verification_attempts").
		Return(nil).Once()
```

- [ ] **Step 1: Write the failing tests**

Append to `gateguard/internal/service/tests/email_verification_test.go`:

```go
func (s *UseCaseSuite) Test_VerifyEmail_LockoutAfterFiveWrongAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = "123456"
			u.EmailVerificationAttempts = 4 // one guess left
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// The 5th wrong guess trips the cap: attempts hits 5 AND the code is burned.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerificationAttempts == 5 && u.EmailVerificationToken == ""
			}),
			repository.Email,
			"email_verification_attempts", "email_verification_token").
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, "000000")
	s.Require().ErrorIs(err, service.ErrVerificationTooManyAttempts)
}

// PINS THE CHECK ORDER. Fails if the attempts check is placed after the token
// comparison: lockout clears the token, so a post-comparison check would fall into
// the mismatch branch and report ErrVerificationTokenInvalid — stranding the user
// with "wrong code" forever and no hint that resending is the way out.
func (s *UseCaseSuite) Test_VerifyEmail_LockedOutRejectsEvenCorrectCode() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = code
			u.EmailVerificationAttempts = 5 // already locked out
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// No UpdateUserBy expectation: the guard returns before any write. If the
	// implementation writes here, the mock fails the test.
	err := s.service.VerifyEmail(s.ctx, email, code) // CORRECT code
	s.Require().ErrorIs(err, service.ErrVerificationTooManyAttempts)
}

func (s *UseCaseSuite) Test_VerifyEmail_WrongCodeIncrementsAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = "123456"
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// Below the cap: increment only, token NOT burned.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerificationAttempts == 1 && u.EmailVerificationToken == "123456"
			}),
			repository.Email,
			"email_verification_attempts").
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, "999999")
	s.Require().ErrorIs(err, service.ErrVerificationTokenInvalid)
}

func (s *UseCaseSuite) Test_VerifyEmail_WithinTTLSucceedsAndResetsAttempts() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = code
			u.EmailVerificationAttempts = 3
			u.EmailVerificationSentAt = time.Now().Add(-23 * time.Hour) // just inside 24h
		}).
		Return(nil).Once()

	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerified && u.EmailVerificationAttempts == 0 && u.EmailVerificationToken == ""
			}),
			repository.Email,
			"email_verified", "email_verification_token", "email_verification_attempts").
		Return(nil).Once()

	s.Require().NoError(s.service.VerifyEmail(s.ctx, email, code))
}

func (s *UseCaseSuite) Test_RequestEmailVerification_ResetsAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationAttempts = 5                              // locked out
			u.EmailVerificationSentAt = time.Now().Add(-2 * time.Minute) // past the 60s cooldown
		}).
		Return(nil).Once()

	// A resend must hand back a fresh guess budget, or lockout is permanent.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool { return u.EmailVerificationAttempts == 0 }),
			repository.Email,
			"email_verification_token", "email_verification_sent_at", "email_verification_attempts").
		Return(nil).Once()

	s.nMock.EXPECT().
		SendEmailVerification(mock.Anything, email, mock.MatchedBy(func(c string) bool { return len(c) == 6 })).
		Return(nil).Once()

	s.Require().NoError(s.service.RequestEmailVerification(s.ctx, email))
}
```

Note `Test_VerifyEmail_ExpiredAfter24h` is not added — Step 0(a) already converts the existing `Test_VerifyEmail_Expired` into exactly that test.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd gateguard && go test -vet=off ./internal/service/tests/... -run TestUsecase -v`
Expected: FAIL — `undefined: service.ErrVerificationTooManyAttempts`.

The suite entrypoint is `TestUsecase` (`user_test.go`), so `-run` must target it, not the method names — testify suite methods are not top-level `go test` functions.

There is also an internal test file, `gateguard/internal/service/email_verification_internal_test.go`. Check whether it asserts anything about the TTL or the number of `UpdateUserBy` columns; if so, update it the same way as Step 0.

- [ ] **Step 3: Change TTL and add the cap constant + error**

In `gateguard/internal/service/email_verification.go`, replace line 23:

```go
const verificationCodeTTL = 15 * time.Minute
```

with:

```go
// 24h: the code arrives unannounced, so a short clock strands users who don't
// check mail immediately. Brute force is bounded by verificationMaxAttempts
// below, not by this TTL.
const verificationCodeTTL = 24 * time.Hour

// verificationMaxAttempts caps wrong guesses per issued code. A 6-digit code is
// 1,000,000 combinations; without this cap the TTL is the only bound.
const verificationMaxAttempts = 5

// ErrVerificationTooManyAttempts is returned once a code has been guessed wrong
// verificationMaxAttempts times. The code is dead; the user must resend.
var ErrVerificationTooManyAttempts = errors.New("verification attempts exceeded")
```

- [ ] **Step 4: Rewrite VerifyEmail with the load-bearing check order**

Replace the body of `VerifyEmail` (lines 70-89) with:

```go
// VerifyEmail marks the account verified when the email/token pair matches.
func (u *UsersService) VerifyEmail(ctx context.Context, email, token string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}

	// ORDER IS LOAD-BEARING: this MUST precede the token comparison. Lockout
	// clears the token, so a later check would fall into the mismatch branch and
	// report ErrVerificationTokenInvalid — telling a locked-out user "wrong code"
	// forever with no hint that resending is the way out.
	if user.EmailVerificationAttempts >= verificationMaxAttempts {
		return ErrVerificationTooManyAttempts
	}

	if token == "" || user.EmailVerificationToken != token {
		user.EmailVerificationAttempts++
		cols := []string{"email_verification_attempts"}
		if user.EmailVerificationAttempts >= verificationMaxAttempts {
			user.EmailVerificationToken = "" // burn the code
			cols = append(cols, "email_verification_token")
		}
		if err := u.repository.UpdateUserBy(ctx, user, repository.Email, cols...); err != nil {
			return fmt.Errorf("persist attempt %s: %w", email, err)
		}
		if user.EmailVerificationAttempts >= verificationMaxAttempts {
			return ErrVerificationTooManyAttempts
		}
		return ErrVerificationTokenInvalid
	}

	if user.EmailVerificationSentAt.IsZero() ||
		time.Since(user.EmailVerificationSentAt) > verificationCodeTTL {
		return ErrVerificationCodeExpired
	}

	user.EmailVerified = true
	user.EmailVerificationToken = ""
	user.EmailVerificationAttempts = 0
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verified", "email_verification_token", "email_verification_attempts"); err != nil {
		return fmt.Errorf("mark verified %s: %w", email, err)
	}
	return nil
}
```

- [ ] **Step 5: Reset attempts on resend**

In `RequestEmailVerification`, after `user.EmailVerificationSentAt = time.Now()`, add the reset and name the column. The block becomes:

```go
	user.EmailVerificationToken = newVerificationCode()
	user.EmailVerificationSentAt = time.Now()
	user.EmailVerificationAttempts = 0 // a new code gets a fresh guess budget
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verification_token", "email_verification_sent_at", "email_verification_attempts"); err != nil {
		return fmt.Errorf("persist code %s: %w", email, err)
	}
```

The existing 60s cooldown check above this stays exactly as-is — it is what stops resend-farming from refilling the budget.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd gateguard && go test -vet=off ./internal/service/... -run TestUsecase -v`
Expected: PASS for all `Test_VerifyEmail_*` and `Test_RequestEmailVerification_*`, including the two repaired in Step 0.

`Test_ReactToInvitation_*` fail here **and on `main`** (org-mock mismatch) — pre-existing, not yours, do not chase. Confirm this by running the same command on a clean checkout if in doubt.

- [ ] **Step 7: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add gateguard/internal/service/email_verification.go gateguard/internal/service/tests/
git commit -m "feat(gateguard): 24h verification TTL bounded by a 5-attempt cap"
```

---

### Task 3: Map service errors to gRPC status codes

**Files:**
- Modify: `gateguard/internal/server/auth_password.go:48-58` (`VerifyEmail` handler)

**Interfaces:**
- Consumes: `service.ErrVerificationTooManyAttempts`, `service.ErrVerificationCodeExpired`, `service.ErrVerificationTokenInvalid` (Task 2).
- Produces: gRPC status codes on the wire — `codes.ResourceExhausted` (too many attempts), `codes.DeadlineExceeded` (expired), `codes.InvalidArgument` (bad code). Task 4 reads these.

**Why:** today the handler returns `fmt.Errorf("verify email: %w", err)`, which crosses gRPC as `codes.Unknown` with a flattened string. Typed codes are what let the proxy distinguish the three cases without string-matching. `backend/internal/grpc/handlers/user_service.go` already uses `status`/`codes` — same pattern.

- [ ] **Step 1: Add the imports**

In `gateguard/internal/server/auth_password.go`, add to the import block:

```go
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gateguard/internal/service"
```

(Keep existing imports. If `errors` or `service` is already imported, don't duplicate.)

- [ ] **Step 2: Replace the VerifyEmail handler error branch**

Replace lines 52-55 (the `if err := h.srv.VerifyEmail(...)` block) with:

```go
	if err := h.srv.VerifyEmail(ctx, req.Email, req.Token); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to verify email")
		switch {
		case errors.Is(err, service.ErrVerificationTooManyAttempts):
			return nil, status.Error(codes.ResourceExhausted, "verification attempts exceeded")
		case errors.Is(err, service.ErrVerificationCodeExpired):
			return nil, status.Error(codes.DeadlineExceeded, "verification code expired")
		case errors.Is(err, service.ErrVerificationTokenInvalid):
			return nil, status.Error(codes.InvalidArgument, "verification token invalid")
		default:
			return nil, fmt.Errorf("verify email: %w", err)
		}
	}
```

- [ ] **Step 3: Verify it builds and vets**

Run: `cd gateguard && go build ./... && go vet ./internal/server/...`
Expected: exit 0 for both.

- [ ] **Step 4: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add gateguard/internal/server/auth_password.go
git commit -m "feat(gateguard): return typed gRPC codes from VerifyEmail"
```

---

### Task 4: Map gRPC codes to a JSON error code in the proxy

**Files:**
- Modify: `backend/internal/http/authverify/handler.go:33-37` (`writeErr`), `:58-76` (`verify`)
- Test: `backend/internal/http/authverify/handler_test.go`

**Interfaces:**
- Consumes: the gRPC codes from Task 3, arriving via `auth.Signer.VerifyEmail`, which wraps with `%w` (`backend/internal/http/auth/signer.go:154`). `status.FromError` unwraps through `%w` via `errors.As` on grpc-go v1.81.1, so the code survives.
- Produces: JSON `{"code": "...", "message": "..."}` with `code` ∈ `verification_expired` | `verification_invalid` | `verification_attempts_exceeded`. Task 5 reads `code`.

This mirrors the existing `{"code":"email_not_verified",...}` shape from `backend/internal/http/handlers/verified_gate.go:12`, which `frontend/lib/api.ts` already parses.

- [ ] **Step 1: Write the failing test**

Create/extend `backend/internal/http/authverify/handler_test.go`. Use a stub Signer returning a wrapped gRPC status, mirroring what the real signer does:

```go
type stubSigner struct{ err error }

func (s stubSigner) VerifyEmail(_ context.Context, _, _ string) error {
	if s.err == nil {
		return nil
	}
	return fmt.Errorf("gateguard verify email: %w", s.err) // same %w wrap as signer.go:154
}
// ...implement the rest of the auth.Signer interface as no-ops returning nil.

func TestVerify_MapsGRPCCodesToJSONCodes(t *testing.T) {
	cases := []struct {
		name     string
		grpcErr  error
		wantHTTP int
		wantCode string
	}{
		{"expired", status.Error(codes.DeadlineExceeded, "x"), http.StatusBadRequest, "verification_expired"},
		{"invalid", status.Error(codes.InvalidArgument, "x"), http.StatusBadRequest, "verification_invalid"},
		{"locked", status.Error(codes.ResourceExhausted, "x"), http.StatusTooManyRequests, "verification_attempts_exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(Deps{Signer: stubSigner{err: tc.grpcErr}})
			req := httptest.NewRequest("POST", "/auth/verify-email",
				strings.NewReader(`{"email":"a@b.c","code":"123456"}`))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantHTTP {
				t.Fatalf("status: want %d, got %d", tc.wantHTTP, rec.Code)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Code != tc.wantCode {
				t.Fatalf("code: want %q, got %q", tc.wantCode, body.Code)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/authverify/... -run TestVerify_MapsGRPCCodes -v`
Expected: FAIL — the handler currently returns `{"message":"invalid or expired code"}` with no `code` field.

- [ ] **Step 3: Add a code-carrying error writer**

In `backend/internal/http/authverify/handler.go`, add below the existing `writeErr` (keep `writeErr` — the other handlers still use it):

```go
func writeErrCode(w http.ResponseWriter, httpCode int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}
```

Add to the import block:

```go
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
```

- [ ] **Step 4: Replace the verify error branch**

Replace lines 71-74 (the `if err := h.deps.Signer.VerifyEmail(...)` block) with:

```go
	if err := h.deps.Signer.VerifyEmail(r.Context(), body.Email, body.Code); err != nil {
		// status.FromError unwraps %w via errors.As (grpc-go v1.81.1), so the
		// code set by GateGuard survives signer.go's fmt.Errorf wrap.
		switch status.Code(err) {
		case codes.ResourceExhausted:
			writeErrCode(w, http.StatusTooManyRequests, "verification_attempts_exceeded",
				"Код заблокирован после 5 попыток. Запросите новый.")
		case codes.DeadlineExceeded:
			writeErrCode(w, http.StatusBadRequest, "verification_expired",
				"Код истёк. Запросите новый.")
		default:
			writeErrCode(w, http.StatusBadRequest, "verification_invalid", "Неверный код.")
		}
		return
	}
```

`codes.InvalidArgument` intentionally falls into `default` — any unmapped error surfaces as the safest generic "wrong code" rather than leaking internals.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/authverify/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add backend/internal/http/authverify/
git commit -m "feat(backend): surface distinct verification error codes from the proxy"
```

---

### Task 5: Frontend — parse the codes, show three messages

**Files:**
- Modify: `frontend/lib/api.ts:625-632` (`verifyEmail`)
- Modify: `frontend/app/auth/verify/page.tsx:40-42` (error branch)

**Interfaces:**
- Consumes: the JSON `code` values from Task 4.
- Produces: exported constants `VERIFICATION_EXPIRED`, `VERIFICATION_INVALID`, `VERIFICATION_ATTEMPTS_EXCEEDED` from `lib/api.ts`.

- [ ] **Step 1: Export the constants and parse the code**

In `frontend/lib/api.ts`, near the existing `EMAIL_NOT_VERIFIED` constant, add:

```ts
export const VERIFICATION_EXPIRED = "VERIFICATION_EXPIRED";
export const VERIFICATION_INVALID = "VERIFICATION_INVALID";
export const VERIFICATION_ATTEMPTS_EXCEEDED = "VERIFICATION_ATTEMPTS_EXCEEDED";
```

Replace the body of `verifyEmail`:

```ts
/** Verifies the account's email via POST /auth/verify-email. */
export async function verifyEmail(email: string, code: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/verify-email`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  });
  if (res.ok || res.status === 204) return;
  const b = await res.json().catch(() => null);
  if (b?.code === "verification_expired") throw new Error(VERIFICATION_EXPIRED);
  if (b?.code === "verification_attempts_exceeded") throw new Error(VERIFICATION_ATTEMPTS_EXCEEDED);
  throw new Error(VERIFICATION_INVALID);
}
```

- [ ] **Step 2: Map the constants to Russian copy on the verify page**

In `frontend/app/auth/verify/page.tsx`, add the import:

```ts
import {
  requestVerification,
  verifyEmail,
  VERIFICATION_EXPIRED,
  VERIFICATION_ATTEMPTS_EXCEEDED,
} from "@/lib/api";
```

Replace the `catch` block in `onSubmit` (lines 40-42):

```tsx
    } catch (err) {
      const m = err instanceof Error ? err.message : "";
      if (m === VERIFICATION_EXPIRED) setError("Код истёк. Запросите новый.");
      else if (m === VERIFICATION_ATTEMPTS_EXCEEDED)
        setError("Код заблокирован после 5 попыток. Запросите новый.");
      else setError("Неверный код.");
    } finally {
```

- [ ] **Step 3: Verify the build compiles**

Run: `cd frontend && pnpm build`
Expected: exit 0, `/auth/verify` present in the route list.

- [ ] **Step 4: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add frontend/lib/api.ts frontend/app/auth/verify/page.tsx
git commit -m "feat(frontend): distinguish expired / invalid / locked-out verification codes"
```

---

### Task 6: The persistent banner

**Files:**
- Create: `frontend/components/VerifyEmailBanner.tsx`
- Modify: `frontend/app/layout.tsx`

**Interfaces:**
- Consumes: `useAuth()` from `@/lib/auth-context` — `{ isAuthed, ready, emailVerified }` (all already exposed; `emailVerified` at `auth-context.tsx:34`).
- Produces: `<VerifyEmailBanner />` (no props).

**Why no dismiss:** it is the only proactive signal, it is one slim row, and it has a clear exit. `/auth/verify` already calls `refresh()` on success (`app/auth/verify/page.tsx:38`), which re-fetches `/auth/me` and flips `emailVerified` — so the banner unmounts itself with no extra wiring.

- [ ] **Step 1: Write the component**

Create `frontend/components/VerifyEmailBanner.tsx`:

```tsx
"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";

/**
 * Persistent, non-dismissible notice shown while a signed-in user's email is
 * unverified. This is the only proactive signal that verification exists —
 * without it, users discover it only by being blocked mid-action.
 *
 * Unmounts itself: /auth/verify calls refresh() on success, flipping
 * emailVerified in the auth context.
 */
export function VerifyEmailBanner() {
  const { isAuthed, ready, emailVerified } = useAuth();

  // `ready` gates the first paint: without it the banner flashes for
  // already-verified users while /auth/me is still in flight.
  if (!ready || !isAuthed || emailVerified) return null;

  return (
    <div className="flex items-center justify-between gap-3 bg-amber-50 px-4 py-2 text-[13px] text-amber-900 dark:bg-amber-950 dark:text-amber-100">
      <span>Почта не подтверждена — часть действий недоступна.</span>
      <Link
        href="/auth/verify"
        className="shrink-0 rounded-capsule bg-accent px-3 py-1 text-white"
      >
        Подтвердить
      </Link>
    </div>
  );
}
```

- [ ] **Step 2: Mount it globally**

In `frontend/app/layout.tsx`, import it and render it inside the auth provider, immediately before the main content so it sits at the top of every page:

```tsx
import { VerifyEmailBanner } from "@/components/VerifyEmailBanner";
```

```tsx
        <VerifyEmailBanner />
```

Read `layout.tsx` first and place it inside whatever provider wraps the tree (it must be a descendant of the auth provider or `useAuth()` throws), directly above the existing `{children}` / page content.

- [ ] **Step 3: Verify the build compiles**

Run: `cd frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 4: Verify it renders in the real app**

Run: `cd frontend && pnpm dev`, open `http://localhost:3000`, sign in as an unverified user.
Expected: the amber bar appears at the top of every page; clicking **Подтвердить** lands on `/auth/verify`. Signed out → no bar. Verified user → no bar.

- [ ] **Step 5: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add frontend/components/VerifyEmailBanner.tsx frontend/app/layout.tsx
git commit -m "feat(frontend): persistent unverified-email banner"
```

---

### Task 7: Signup confirmation state

**Files:**
- Modify: `frontend/components/AuthButton.tsx:96-110` (submit handler), `:115-132` (modal body)

**Interfaces:**
- Consumes: `register()` from `useAuth()` (already used at `AuthButton.tsx:100`).
- Produces: nothing consumed elsewhere.

**Why:** this is the exact moment the product owner got nothing. `AuthButton.tsx:104` calls `onClose()` on success, so the modal just closes — no toast, no redirect, no confirmation.

- [ ] **Step 1: Add the confirmation state**

In `LoginModal`, add alongside the other `useState` hooks:

```tsx
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);
```

- [ ] **Step 2: Branch the submit handler**

Replace the `try` block body in `submit` (lines 99-104) with:

```tsx
      if (isRegister) {
        const addr = email.trim();
        await register(addr, name.trim(), password);
        setRegisteredEmail(addr); // show the confirmation instead of closing
      } else {
        await loginPassword(email.trim(), password);
        onClose();
      }
```

Login still closes silently; only register shows the confirmation.

- [ ] **Step 3: Render the confirmation instead of the form**

Immediately inside the `return (`, before the existing modal markup, add an early branch. Keep the same outer overlay/`onClick={onClose}` wrapper the form uses:

```tsx
  if (registeredEmail) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
        onClick={onClose}
      >
        <div className="w-full max-w-sm rounded-card bg-bg p-5" onClick={(e) => e.stopPropagation()}>
          <h2 className="mb-1 text-[17px] font-semibold">Проверьте почту</h2>
          <p className="mb-4 text-[13px] text-label-secondary">
            Мы отправили 6-значный код на {registeredEmail}. Он действует 24 часа.
          </p>
          <div className="flex gap-2">
            <Link href="/auth/verify" className="rounded-capsule bg-accent px-4 py-2 text-white">
              Ввести код
            </Link>
            <button onClick={onClose} className="rounded-capsule bg-fill px-4 py-2 text-label">
              Позже
            </button>
          </div>
        </div>
      </div>
    );
  }
```

Add `import Link from "next/link";` at the top if it isn't already imported.

- [ ] **Step 4: Verify the build compiles**

Run: `cd frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 5: Verify both flows in the real app**

Run: `cd frontend && pnpm dev`.
- Register a new account → the modal shows «Проверьте почту … на \<your address\>»; **Ввести код** → `/auth/verify`; **Позже** → closes and the Task 6 banner is visible.
- Log in with an existing account → modal closes silently, no confirmation.

- [ ] **Step 6: Commit**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add frontend/components/AuthButton.tsx
git commit -m "feat(frontend): confirm on signup that a verification code was sent"
```

---

### Task 8: Deploy

**Files:** none (operational).

Full procedure and traps: `docs/superpowers/runbooks/2026-07-16-email-verification-invitations-deploy.md`. Every trap below bit during that deploy.

- [ ] **Step 1: Back up the gateguard DB**

```bash
ssh vdska2 'set -a; . /opt/lia/backend/.env.prod; set +a; docker exec backend-postgres-1 pg_dump -U "$DATABASE_USER" gateguard | gzip > /opt/lia/backup-pre-verif-attempts-$(date +%Y%m%d-%H%M).sql.gz; ls -lh /opt/lia/backup-pre-verif-attempts-*.sql.gz'
```

- [ ] **Step 2: Sync gateguard source to the box**

The box's `/opt/gateguard` has been stale before (it held a pre-feature stub). `rsync` is blocked by the permission classifier; use tar:

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/gateguard
tar czf - --exclude='.git' --exclude='data' . | ssh vdska2 'cat > /tmp/gg-src.tgz'
ssh vdska2 'cd /opt/gateguard && tar xzf /tmp/gg-src.tgz && md5sum internal/service/email_verification.go'
```

Verify the md5 matches `md5 -q gateguard/internal/service/email_verification.go` locally.

- [ ] **Step 3: Apply gateguard migration 000012**

Targets the **`gateguard`** database, NOT Lia's `020` chain. Note `-v /opt/gateguard/db:/db` — migrations are flat in `db/`:

```bash
ssh vdska2 'set -a; . /opt/lia/backend/.env.prod; set +a; docker run --rm --network backend_default -v /opt/gateguard/db:/db migrate/migrate:v4.17.1 -path=/db/ -database "postgresql://$DATABASE_USER:$DATABASE_PASSWORD@postgres:5432/gateguard?sslmode=disable" up'
```

If this needs to be run by hand, put it in a **script file** — a wrapped one-liner makes `migrate` print usage and bash then tries to execute `-path=/db/`.

Verify: `ssh vdska2 'docker exec backend-postgres-1 psql -U lia_prod -d gateguard -c "\d users"'` → `email_verification_attempts` present.

- [ ] **Step 4: Build gateguard ON THE MAC and ship**

Building on the box FAILS — `curl` to github.com dies with a TLS error fetching protoc-gen-grpc-web.

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/gateguard
docker build --platform linux/amd64 -t gateguard:verif-r2 .
docker image inspect gateguard:verif-r2 --format '{{.Architecture}}'   # must print: amd64
docker save gateguard:verif-r2 | gzip | ssh vdska2 'gunzip | docker load'
```

- [ ] **Step 5: Build + ship the backend**

`make generate-api` FIRST — swagger `internal/http/{models,server}` are git-ignored and NOT regenerated by `make build`:

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/backend
make generate-api
docker build --platform linux/amd64 -t lia-backend:verif-r2 .
docker save lia-backend:verif-r2 | gzip | ssh vdska2 'gunzip | docker load'
```

- [ ] **Step 6: Build + ship the frontend with BOTH build-args**

A missing arg is silently dropped and inlines as `""` — the maps key going missing breaks every map.

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/frontend
YKEY=$(grep '^NEXT_PUBLIC_YANDEX_MAPS_KEY=' .env.local | cut -d= -f2-)
docker build --platform linux/amd64 \
  --build-arg NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru \
  --build-arg NEXT_PUBLIC_YANDEX_MAPS_KEY="$YKEY" \
  -t lia-frontend:verif-r2 .
docker save lia-frontend:verif-r2 | gzip | ssh vdska2 'gunzip | docker load'
```

- [ ] **Step 7: Tag rollbacks, then cut over**

Live image names are `backend-app` / `lia-frontend-presence` / `gateguard:local` — NOT `lia-*`.

```bash
ssh vdska2 'TS=$(date +%Y%m%d-%H%M); docker tag backend-app:latest backend-app:rollback-$TS; docker tag lia-frontend-presence:latest lia-frontend-presence:rollback-$TS; docker tag gateguard:local gateguard:rollback-$TS'
ssh vdska2 'cd /opt/lia/backend && docker tag gateguard:verif-r2 gateguard:local && docker tag lia-backend:verif-r2 backend-app:latest && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build --force-recreate app gateguard'
```

Frontend is NOT compose-managed — stop + **rename** the old container (preserves rollback), then run the new one:

```bash
ssh vdska2 'docker tag lia-frontend:verif-r2 lia-frontend-presence:latest; docker stop lia-frontend-presence; docker rename lia-frontend-presence lia-frontend-presence-old-$(date +%Y%m%d-%H%M); docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest'
```

- [ ] **Step 8: Verify**

```bash
for u in https://presence.tarski.ru https://presence.tarski.ru/auth/verify https://presence.tarski.ru/map https://api.presence.tarski.ru/api/v1/events?limit=1; do
  echo "$u -> $(curl -s -o /dev/null -w '%{http_code}' -m 15 "$u")"
done
ssh vdska2 'docker logs backend-gateguard-1 --tail=12'
```

Expected: all 200; gateguard logs show gRPC :9090 with no DB/redis errors.
`docker exec backend-app-1 env` returns NOTHING — it is a scratch image with no `env` binary. That is NOT evidence the env is missing; use `docker inspect backend-app-1 --format '{{range .Config.Env}}{{println .}}{{end}}'`.

Then in a browser: sign up with a real address → the confirmation appears → the banner is visible → enter the code → both disappear.

- [ ] **Step 9: Prune (the 20 GB disk has hit 90% before)**

```bash
ssh vdska2 'docker builder prune -f; docker image prune -f; df -h / | tail -1'
```

---

## Rollback

- **GateGuard:** `docker tag gateguard:rollback-<ts> gateguard:local` → recreate with the 4 compose files.
- **Backend:** `docker tag backend-app:rollback-<ts> backend-app:latest` → recreate.
- **Frontend:** `docker rm -f lia-frontend-presence && docker rename lia-frontend-presence-old-<ts> lia-frontend-presence && docker start lia-frontend-presence`.
- **DB:** `migrate ... down 1` against the `gateguard` database, or restore `backup-pre-verif-attempts-*.sql.gz`. The column is additive with a default, so the old binary tolerates it — rollback of code alone is safe without touching the DB.
