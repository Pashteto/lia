# Phase A — Real Verification Email (GateGuard) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace GateGuard's log-only email-verification stub with a real 6-digit code emailed via SMTP (SendPulse), with a resend cooldown and code expiry.

**Architecture:** Signup and `RequestEmailVerification` generate a 6-digit numeric code, store it in the existing `users.email_verification_token` column, stamp `users.email_verification_sent_at`, and send it through GateGuard's existing `SMTPNotificator` using a new `email_verification` HTML template. `VerifyEmail` accepts the code within a 15-minute window. No schema change (all columns already exist from migration `000011`).

**Tech Stack:** Go, `net/smtp`, `html/template`, `crypto/rand`, testify suite + mockery mocks (`stretchr/testify`), go-pg.

## Global Constraints

- Module root for this phase: `gateguard/` (its own Go module, `package`-rooted at `gateguard`).
- Email copy is **Russian**; sender From address is `info@tarski.ru` (set via config, not hardcoded).
- Verification code is a **6-digit numeric string** (`000000`–`999999`), cryptographically random, stored as text in `email_verification_token`.
- Resend cooldown: **60 seconds**. Code validity window: **15 minutes** (both measured against `email_verification_sent_at`).
- Do NOT change DB schema — columns `email_verified`, `email_verification_token`, `email_verification_sent_at` already exist.
- Follow existing patterns: notificator method mirrors `InviteUserToOrganization`; template mirrors `UserInviteToOrg`; service tests use the `UseCaseSuite` in `gateguard/internal/service/tests/user_test.go` (fields `s.repo`, `s.nMock`, `s.service`, `s.ctx`).
- After adding a method to `INotificator`, regenerate its mockery mock (`go generate ./...` from `gateguard/`).
- Run all Go commands from the `gateguard/` directory.

---

### Task 1: 6-digit verification code generator

**Files:**
- Modify: `gateguard/internal/service/email_verification.go`
- Test: `gateguard/internal/service/email_verification_internal_test.go` (create; `package service` for white-box access)

**Interfaces:**
- Produces: `func newVerificationCode() string` — returns a 6-char string of digits, zero-padded, uniformly random in `[0, 999999]`.

- [ ] **Step 1: Write the failing test**

```go
// gateguard/internal/service/email_verification_internal_test.go
package service

import (
	"regexp"
	"testing"
)

func Test_newVerificationCode_format(t *testing.T) {
	re := regexp.MustCompile(`^[0-9]{6}$`)
	seen := map[string]int{}
	for i := 0; i < 1000; i++ {
		code := newVerificationCode()
		if !re.MatchString(code) {
			t.Fatalf("code %q is not 6 digits", code)
		}
		seen[code]++
	}
	// Sanity: not all identical (would indicate a broken RNG).
	if len(seen) < 100 {
		t.Fatalf("suspiciously low entropy: only %d distinct codes in 1000", len(seen))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/service/ -run Test_newVerificationCode_format -v`
Expected: FAIL — `undefined: newVerificationCode`.

- [ ] **Step 3: Write minimal implementation**

Add to `gateguard/internal/service/email_verification.go` (imports: add `crypto/rand`, `math/big`; keep existing):

```go
// newVerificationCode returns a cryptographically-random 6-digit numeric code
// as a zero-padded string (e.g. "042173").
func newVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// rand.Reader failure is catastrophic; fall back to a non-guessable-enough
		// value derived from the same reader via a smaller read.
		b := make([]byte, 3)
		_, _ = rand.Read(b)
		n = big.NewInt(int64(b[0])<<16 | int64(b[1])<<8 | int64(b[2]))
		n.Mod(n, big.NewInt(1000000))
	}
	return fmt.Sprintf("%06d", n.Int64())
}
```

Note: `encoding/base64` may become unused once the old `newVerificationToken` is removed in Task 3; leave it for now (Task 3 removes the import).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd gateguard && go test ./internal/service/ -run Test_newVerificationCode_format -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/email_verification.go gateguard/internal/service/email_verification_internal_test.go
git commit -m "feat(gateguard): 6-digit verification code generator"
```

---

### Task 2: Verification email template

**Files:**
- Create: `gateguard/internal/pkg/notificator/templates/html/email_verification.go`
- Create: `gateguard/internal/pkg/notificator/templates/email_verification.go`
- Modify: `gateguard/internal/pkg/notificator/templates/parser.go`
- Test: `gateguard/internal/pkg/notificator/templates/email_verification_test.go` (create)

**Interfaces:**
- Produces:
  - `html.VerificationEmailTemplate` (string constant, an `html/template` body referencing `{{ .Code }}`).
  - `func NewEmailVerification(log *clog.CustomLogger, code string) *EmailVerification`
  - `EmailVerification` implements `templates.ITemplate` (`GetTemplateAsString`, `TemplateName`, `Subject`).

- [ ] **Step 1: Write the failing test**

```go
// gateguard/internal/pkg/notificator/templates/email_verification_test.go
package templates_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gateway-fm/scriptorium/clog"

	"gateguard/internal/pkg/notificator/templates"
)

func Test_EmailVerification_RendersCode(t *testing.T) {
	log := clog.NewCustomLogger(nil)
	tmpl := templates.NewEmailVerification(log, "042173")

	body, err := tmpl.GetTemplateAsString(context.Background())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(body, "042173") {
		t.Fatalf("rendered body does not contain the code; got: %s", body)
	}
	if tmpl.TemplateName() != "email_verification" {
		t.Fatalf("unexpected template name %q", tmpl.TemplateName())
	}
	if !strings.HasPrefix(tmpl.Subject(), "Subject:") {
		t.Fatalf("subject must start with 'Subject:'; got %q", tmpl.Subject())
	}
}
```

> If `clog.NewCustomLogger(nil)` is not the correct constructor, copy the logger-construction line used at the top of `gateguard/internal/service/tests/user_test.go` (`s.log`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/pkg/notificator/templates/ -run Test_EmailVerification_RendersCode -v`
Expected: FAIL — `undefined: templates.NewEmailVerification`.

- [ ] **Step 3a: Create the HTML template constant**

```go
// gateguard/internal/pkg/notificator/templates/html/email_verification.go
package html

const VerificationEmailTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8" />
    <title>Подтверждение почты</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 20px; background-color: #f4f4f4; }
        p { margin: 8px 0; }
        .container { background:#fff; padding:24px; border-radius:8px; max-width:600px; margin:0 auto; line-height:1.5; }
        .code { font-size:32px; letter-spacing:8px; font-weight:700; color:#111; margin:16px 0; }
        .muted { color:#666; font-size:13px; }
    </style>
</head>
<body>
<div class="container">
    <h2>Подтвердите вашу электронную почту</h2>
    <p>Введите этот код, чтобы подтвердить адрес электронной почты:</p>
    <div class="code">{{ .Code }}</div>
    <p class="muted">Код действителен 15 минут. Если вы не запрашивали подтверждение, просто проигнорируйте это письмо.</p>
    <p>С уважением,<br/>Команда Presence</p>
</div>
</body>
</html>`
```

- [ ] **Step 3b: Create the template type**

```go
// gateguard/internal/pkg/notificator/templates/email_verification.go
package templates

import (
	"context"

	"github.com/gateway-fm/scriptorium/clog"
)

const emailVerificationTemplateName = "email_verification"

// EmailVerification is the transactional email carrying a 6-digit code.
type EmailVerification struct {
	Code string

	log *clog.CustomLogger
}

func NewEmailVerification(log *clog.CustomLogger, code string) *EmailVerification {
	return &EmailVerification{Code: code, log: log}
}

func (t EmailVerification) GetTemplateAsString(ctx context.Context) (string, error) {
	return parseNamedTemplate(ctx, emailVerificationTemplateName, htmlVerificationBody(), parseTemplateIn{
		log:      t.log,
		metadata: t,
	})
}

func (t EmailVerification) TemplateName() string { return emailVerificationTemplateName }

func (t EmailVerification) Subject() string {
	return "Subject: Presence: код подтверждения почты"
}
```

- [ ] **Step 3c: Generalize the parser to accept a template name + body**

The current `parseTemplate` is hardcoded to the org-invite template. Refactor `gateguard/internal/pkg/notificator/templates/parser.go` so both templates share it. Replace the file body with:

```go
package templates

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/gateway-fm/scriptorium/clog"

	htmlpkg "gateguard/internal/pkg/notificator/templates/html"
)

const templateName = "organization_invite"

type parseTemplateIn struct {
	log      *clog.CustomLogger
	metadata any
}

// parseTemplate renders the org-invite template (kept for the existing caller).
func parseTemplate(ctx context.Context, in parseTemplateIn) (string, error) {
	return parseNamedTemplate(ctx, templateName, htmlpkg.EmailTemplate, in)
}

// htmlVerificationBody returns the verification template body (indirection keeps
// the html import in one place).
func htmlVerificationBody() string { return htmlpkg.VerificationEmailTemplate }

// parseNamedTemplate parses and executes an arbitrary named html/template body.
func parseNamedTemplate(ctx context.Context, name, body string, in parseTemplateIn) (string, error) {
	ctx = in.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"template_name": name})

	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		in.log.ErrorCtx(ctx, err, "failed to parse template")
		return "", fmt.Errorf("failed to parse %s template: %w", name, err)
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, in.metadata); err != nil {
		in.log.ErrorCtx(ctx, err, "failed to execute template")
		return "", fmt.Errorf("failed to execute %s template: %w", name, err)
	}
	return buf.String(), nil
}
```

> Note the import alias `htmlpkg` avoids shadowing the stdlib `html/template`. The pre-existing `html` import in this package was named `html`; switching to `htmlpkg` is intentional.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd gateguard && go test ./internal/pkg/notificator/templates/ -run Test_EmailVerification_RendersCode -v`
Expected: PASS. Also run `cd gateguard && go build ./...` to confirm the parser refactor didn't break the existing org-invite caller.

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/pkg/notificator/templates/
git commit -m "feat(gateguard): email-verification HTML template (RU, 6-digit code)"
```

---

### Task 3: Notificator SendEmailVerification method

**Files:**
- Modify: `gateguard/internal/pkg/notificator/interface.go`
- Modify: `gateguard/internal/pkg/notificator/notificator.go`
- Regenerate: `gateguard/internal/pkg/notificator/mocks/` (via `go generate`)

**Interfaces:**
- Produces: `SendEmailVerification(ctx context.Context, to, code string) error` on `INotificator` and `SMTPNotificator`.

- [ ] **Step 1: Add the method to the interface**

```go
// gateguard/internal/pkg/notificator/interface.go
package notificator

import (
	"context"

	"gateguard/internal/pkg/notificator/templates"
)

//go:generate ../../../bin/mockery --name INotificator

type INotificator interface {
	InviteUserToOrganization(ctx context.Context, to string, tmpl *templates.UserInviteToOrg) error
	SendEmailVerification(ctx context.Context, to, code string) error
}
```

- [ ] **Step 2: Implement on SMTPNotificator**

Append to `gateguard/internal/pkg/notificator/notificator.go`:

```go
// SendEmailVerification emails a 6-digit verification code.
func (s *SMTPNotificator) SendEmailVerification(ctx context.Context, to, code string) error {
	tmpl := templates.NewEmailVerification(s.log, code)
	return s.sendTemplate(ctx, to, tmpl)
}
```

- [ ] **Step 3: Regenerate the mock, then verify build**

Run:
```bash
cd gateguard && go generate ./internal/pkg/notificator/... && go build ./...
```
Expected: `mocks/INotificator` now has `SendEmailVerification`; build succeeds.

> If `go generate` can't find `../../../bin/mockery`, run mockery however this repo installs it (check `gateguard/Makefile` for a `mocks`/`generate` target and use that).

- [ ] **Step 4: Commit**

```bash
git add gateguard/internal/pkg/notificator/
git commit -m "feat(gateguard): SendEmailVerification on notificator + mock"
```

---

### Task 4: Wire real send into signup + RequestEmailVerification (replace stub)

**Files:**
- Modify: `gateguard/internal/service/email_verification.go`
- Modify: `gateguard/internal/service/sign_in_password.go`
- Test: `gateguard/internal/service/tests/email_verification_test.go` (create)

**Interfaces:**
- Consumes: `newVerificationCode()` (Task 1), `INotificator.SendEmailVerification` (Task 3), `UseCaseSuite` (`s.repo`, `s.nMock`, `s.service`, `s.ctx`).
- Produces: `RequestEmailVerification` now generates a 6-digit code, stamps `EmailVerificationSentAt`, persists both columns, and calls `SendEmailVerification`. Signup uses `newVerificationCode()` + stamps sent-at and sends.

- [ ] **Step 1: Write the failing test**

```go
// gateguard/internal/service/tests/email_verification_test.go
package service_test

import (
	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

func (s *UseCaseSuite) Test_RequestEmailVerification_SendsCode() {
	email := "user@example.com"

	// GetUser loads a user with no prior send (zero sent-at → no cooldown).
	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Return(nil).Once()

	// Persist must include both the token and the sent-at columns.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
			"email_verification_token", "email_verification_sent_at").
		Return(nil).Once()

	// A real 6-digit code must be sent to the address.
	s.nMock.EXPECT().
		SendEmailVerification(mock.Anything, email, mock.MatchedBy(func(code string) bool {
			return len(code) == 6
		})).
		Return(nil).Once()

	err := s.service.RequestEmailVerification(s.ctx, email)
	s.Require().NoError(err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/service/tests/ -run Test_RequestEmailVerification_SendsCode -v`
Expected: FAIL — mock `SendEmailVerification` never called / `UpdateUserBy` column mismatch.

- [ ] **Step 3: Rewrite `RequestEmailVerification` + remove the stub**

Replace the stub and request function in `gateguard/internal/service/email_verification.go`. Remove `sendVerificationStub`, remove `newVerificationToken` and the now-unused `encoding/base64` import; add `time`. New body:

```go
// RequestEmailVerification regenerates a 6-digit code, stamps the send time, and
// emails it. Enforces a 60-second resend cooldown (Task 5 adds the check).
func (u *UsersService) RequestEmailVerification(ctx context.Context, email string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}

	user.EmailVerificationToken = newVerificationCode()
	user.EmailVerificationSentAt = time.Now()
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verification_token", "email_verification_sent_at"); err != nil {
		return fmt.Errorf("persist code %s: %w", email, err)
	}

	if err := u.notificator.SendEmailVerification(ctx, email, user.EmailVerificationToken); err != nil {
		return fmt.Errorf("send verification %s: %w", email, err)
	}
	return nil
}
```

- [ ] **Step 4: Update signup to use the code + stamp + send**

In `gateguard/internal/service/sign_in_password.go`, inside `SignUpWithPassword`:

1. Replace `token := newVerificationToken()` with `code := newVerificationCode()`.
2. In the `user := &models.User{...}` literal, replace `EmailVerificationToken: token,` with:
   ```go
   EmailVerificationToken:  code,
   EmailVerificationSentAt: time.Now(),
   ```
3. In the passwordless-attach `UpdateUserBy(...)` call, add `"email_verification_sent_at"` to the column list.
4. Replace `u.sendVerificationStub(ctx, user)` with:
   ```go
   if sErr := u.notificator.SendEmailVerification(ctx, email, code); sErr != nil {
       // Do not fail signup if the email send fails; the user can request a resend.
       u.log.ErrorCtx(ctx, sErr, fmt.Sprintf("send verification email %s", email))
   }
   ```
5. Add `"time"` to the imports.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd gateguard && go test ./internal/service/... -run Test_RequestEmailVerification_SendsCode -v && go build ./...`
Expected: PASS and clean build (no unused-import errors).

- [ ] **Step 6: Commit**

```bash
git add gateguard/internal/service/email_verification.go gateguard/internal/service/sign_in_password.go gateguard/internal/service/tests/email_verification_test.go
git commit -m "feat(gateguard): send real verification code on signup + request; drop stub"
```

---

### Task 5: Resend cooldown (60s)

**Files:**
- Modify: `gateguard/internal/service/email_verification.go`
- Test: `gateguard/internal/service/tests/email_verification_test.go`

**Interfaces:**
- Produces: `var ErrVerificationCooldown = errors.New("verification code recently sent")`; `RequestEmailVerification` returns it when the last send was under 60s ago.

- [ ] **Step 1: Write the failing test**

```go
func (s *UseCaseSuite) Test_RequestEmailVerification_Cooldown() {
	email := "user@example.com"

	// GetUser returns a user whose last send was 5 seconds ago → cooldown active.
	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ ...any) {
			u.EmailVerificationSentAt = time.Now().Add(-5 * time.Second)
		}).
		Return(nil).Once()

	err := s.service.RequestEmailVerification(s.ctx, email)
	s.Require().ErrorIs(err, service.ErrVerificationCooldown)
}
```

Add imports to the test file as needed: `context`, `time`, and `gateguard/internal/service`. Match the `GetUser` mock signature's variadic exactly — inspect `gateguard/internal/service/mocks` for the generated `Run` closure argument types and adjust `_ ...any` to the generated type (likely `...repository.GetUserOpt` or the field-selector type). Use the same closure shape the existing tests use if they set fields via `Run`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/service/tests/ -run Test_RequestEmailVerification_Cooldown -v`
Expected: FAIL — `undefined: service.ErrVerificationCooldown` (then, once defined, FAIL because no cooldown check yet).

- [ ] **Step 3: Add the cooldown constant + check**

In `gateguard/internal/service/email_verification.go` add near the top:

```go
const verificationResendCooldown = 60 * time.Second

// ErrVerificationCooldown is returned when a code was sent less than the cooldown ago.
var ErrVerificationCooldown = errors.New("verification code recently sent")
```

In `RequestEmailVerification`, immediately after the successful `GetUser`, before regenerating the code:

```go
	if !user.EmailVerificationSentAt.IsZero() &&
		time.Since(user.EmailVerificationSentAt) < verificationResendCooldown {
		return ErrVerificationCooldown
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd gateguard && go test ./internal/service/tests/ -run "Test_RequestEmailVerification" -v`
Expected: both `_SendsCode` and `_Cooldown` PASS.

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/email_verification.go gateguard/internal/service/tests/email_verification_test.go
git commit -m "feat(gateguard): 60s resend cooldown for verification codes"
```

---

### Task 6: Code expiry (15 min) in VerifyEmail

**Files:**
- Modify: `gateguard/internal/service/email_verification.go`
- Test: `gateguard/internal/service/tests/email_verification_test.go`

**Interfaces:**
- Produces: `var ErrVerificationCodeExpired = errors.New("verification code expired")`; `VerifyEmail` rejects a matching code older than 15 minutes.

- [ ] **Step 1: Write the failing test**

```go
func (s *UseCaseSuite) Test_VerifyEmail_Expired() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ ...any) {
			u.EmailVerificationToken = code
			u.EmailVerificationSentAt = time.Now().Add(-20 * time.Minute) // older than 15m
		}).
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, code)
	s.Require().ErrorIs(err, service.ErrVerificationCodeExpired)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/service/tests/ -run Test_VerifyEmail_Expired -v`
Expected: FAIL — code currently verifies successfully regardless of age.

- [ ] **Step 3: Add TTL constant + expiry check**

In `gateguard/internal/service/email_verification.go` add:

```go
const verificationCodeTTL = 15 * time.Minute

// ErrVerificationCodeExpired is returned when a matching code is older than the TTL.
var ErrVerificationCodeExpired = errors.New("verification code expired")
```

In `VerifyEmail`, after the token-match check (`user.EmailVerificationToken != token` returns `ErrVerificationTokenInvalid`) and before setting `EmailVerified = true`:

```go
	if user.EmailVerificationSentAt.IsZero() ||
		time.Since(user.EmailVerificationSentAt) > verificationCodeTTL {
		return ErrVerificationCodeExpired
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd gateguard && go test ./internal/service/... -v`
Expected: all verification tests PASS.

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/email_verification.go gateguard/internal/service/tests/email_verification_test.go
git commit -m "feat(gateguard): 15-minute expiry for verification codes"
```

---

### Task 7: Config + full build/lint verification

**Files:**
- Reference only: `gateguard/config/init.go` (defaults), deployment env.

- [ ] **Step 1: Confirm config keys exist**

The notificator already reads (`gateguard/config/init.go`): `notificator.username`, `notificator.password`, `notificator.from`, `notificator.address`. No code change needed. Document the production values to set via env (do NOT commit secrets):

```
notificator.address  = <SendPulse SMTP host:port, e.g. smtp-pulse.com:2525>
notificator.username = <SendPulse SMTP login>
notificator.password = <SendPulse SMTP password>
notificator.from     = info@tarski.ru
```

- [ ] **Step 2: Full module verification**

Run:
```bash
cd gateguard && go build ./... && go vet ./... && go test ./...
```
Expected: build + vet clean; all tests (including pre-existing) PASS.

- [ ] **Step 3: Lint (match repo tooling)**

Run the repo's linter the same way CI does (per project memory, golangci **v1**):
```bash
cd gateguard && golangci-lint run ./... 2>&1 | tail -20
```
Expected: no new findings in touched files. Fix any unused-import / naming issues introduced by the parser refactor.

- [ ] **Step 4: Commit any lint fixups**

```bash
git add -A
git commit -m "chore(gateguard): lint + build fixups for verification email"
```

---

## Self-Review

**Spec coverage (Phase A of `2026-07-15-email-verification-and-invitations-design.md` §4):**
- 6-digit code → Task 1. ✅
- Real send + template + interface method → Tasks 2, 3. ✅
- Replace stub in signup + request → Task 4. ✅
- `email_verification_sent_at` stamped (first real use) → Task 4. ✅
- 60s resend cooldown → Task 5. ✅
- 15-min expiry → Task 6. ✅
- Config via env (From `info@tarski.ru`) → Task 7. ✅
- Not in this plan (deferred to later plans): proto `email_verified` propagation, Lia gating, Lia proxy endpoints, frontend `/auth/verify` page — those are Plan 2 / Plan 3.

**Placeholder scan:** No TBD/TODO; all code blocks concrete. Two flagged verification points (logger constructor in Task 2 Step 1; `GetUser` mock `Run` closure variadic type in Task 5 Step 1) are explicit "inspect-and-match" notes, not placeholders — the generated mock signatures must be read to get the exact types.

**Type consistency:** `newVerificationCode()` (Tasks 1,3,4), `SendEmailVerification(ctx,to,code)` (Tasks 3,4), `EmailVerificationSentAt` / `EmailVerificationToken` columns (Tasks 4,5,6), error vars `ErrVerificationCooldown` / `ErrVerificationCodeExpired` used consistently. `parseNamedTemplate` introduced in Task 2 and used only there.

## Frontend / verify-page note

The user-facing `/auth/verify` page and Lia proxy endpoints (`POST /auth/request-verification`, `POST /auth/verify-email`) are **Plan 2**, because they depend on the proto `email_verified` propagation and the Lia-side wiring. Phase A is testable end-to-end at the GateGuard RPC layer (call `RequestEmailVerification` / `VerifyEmail` directly) plus a real-inbox smoke test once SendPulse DNS + creds are in place.
