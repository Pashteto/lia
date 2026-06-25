# Password Auth (vendored GateGuard) + Email-Verification Stub + `/me` Suite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add real password sign-up / sign-in (with a stubbed email-verification flow) by vendoring the GateGuard identity service into this repo and extending it, wire Lia + the frontend to it, then build the planned `/me/*` personal area.

**Architecture:** GateGuard stays the JWT authority. We vendor its source into `/gateguard` (done) and add credentialed RPCs (`SignUpWithPassword`, `SignInWithPassword`) plus stubbed `RequestEmailVerification` / `VerifyEmail`. Passwords are bcrypt-hashed and stored on GateGuard's `users` table (it owns identity; Lia never stores credentials). Lia gains `POST /auth/register` + `POST /auth/login` that proxy to GateGuard over gRPC and return the same JWT the rest of the app already validates via `CheckAuth`. The frontend login modal gains password + a register/login toggle. The `/me/*` suite is built last, in dependency order, as its own per-domain plans.

**Tech Stack:** Go 1.23/1.24 (both modules), go-pg (ORM + go-pg hooks), gofrs/uuid, golang-migrate (`db/*.sql`), protoc 34.1 + protoc-gen-go + protoc-gen-go-grpc + mockery (codegen), `golang.org/x/crypto/bcrypt`, gRPC; frontend Next.js App Router + TS + Tailwind v4 + pnpm + RHF/Zod.

## Global Constraints

- **Two proto copies must stay in sync.** GateGuard implements `/gateguard/protocols/gateguard/*.proto`; Lia *calls* the vendored copy at `/backend/protocols/gateguard/*.proto`. Every RPC/message added here MUST be added to BOTH and regenerated in BOTH. GateGuard's `*.pb.go` is gitignored + regenerated (`make generate-proto`); Lia's vendored `*.pb.go` is **force-committed** (overrides gitignore) — regenerate and `git add -f`.
- **go-pg + gofrs UUID cannot scan SQL NULL into a uuid field.** New nullable columns here are `text`/`bool`/`timestamp` (fine). Keep using zero-UUID convention elsewhere.
- **GateGuard sign-in panics on unset Status/Role.** Any code path that mints a JWT MUST set `Status: UserActive` and `Role: UserRoleCommon` on the user (see `gateguard/internal/service/sign_in.go:getOrCreateUser` and Lia `signer.go:52`). Do not regress this.
- **`FROM scratch` + tz:** keep `_ "time/tzdata"` imports in both `cmd/*` mains; do not remove.
- **golangci-lint v1** in CI for both modules — do not migrate configs to v2.
- **Never log or return the password / `password_hash`.** Exclude `PasswordHash` from `User.Proto()` and from any API response. `GET /files`, demo-login, etc. unchanged.
- **Compliance (ISO 27001 / Vanta):** adding a credential store is an auditable change. Passwords are bcrypt (cost ≥ 10). The email-verification flow is a **stub** in this phase (token generated + persisted + "send" logged, not emailed) — it must be clearly labelled non-production and tracked as a follow-up before any real prod use. `demo-login` remains a non-prod control and is unchanged.
- **RU UI copy, English code.** New frontend strings in Russian to match existing screens.

---

## Phase 0 — Vendor GateGuard + build wiring

### Task 0.1: Vendor the GateGuard source (DONE) + ignore generated artifacts

**Files:**
- Create: `/gateguard/**` (copied from `/Users/dodonovpavel/gateway_fm/appstore/gateguard`, excluding `.git/`, `.idea/`, `vendor/`) — **already done**
- Modify: `/.gitignore` (root)

- [ ] **Step 1: Confirm the copy is present and is a separate Go module**

Run: `head -1 gateguard/go.mod`
Expected: `module gateguard`

- [ ] **Step 2: Keep GateGuard's generated proto out of git (it regenerates), but DO track its source `.proto` and migrations**

GateGuard's own `.gitignore` already ignores `*.pb.go`. Confirm nothing generated is staged:

Run: `git -C . status --porcelain gateguard | grep -E '\.pb\.go$' || echo "no generated pb.go staged"`
Expected: `no generated pb.go staged`

- [ ] **Step 3: Commit the vendored source**

```bash
git add gateguard .gitignore
git commit -m "chore(gateguard): vendor GateGuard identity service into the repo"
```

### Task 0.2: Point the compose build at the vendored copy

**Files:**
- Modify: `/backend/docker-compose.gateguard.yml` (the `gateguard` service)

Currently `image: gateguard:local` assumes a pre-built image from `/opt/gateguard`. Add a build context so the image builds from the vendored source.

- [ ] **Step 1: Add `build` to the `gateguard` service**

```yaml
  gateguard:
    image: gateguard:local
    build:
      context: ../gateguard
      dockerfile: Dockerfile
    restart: unless-stopped
```

- [ ] **Step 2: Verify compose still parses**

Run: `cd backend && docker compose -f docker-compose.yml -f docker-compose.gateguard.yml config >/dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add backend/docker-compose.gateguard.yml
git commit -m "chore(gateguard): build the vendored image from /gateguard"
```

---

## Phase 1 — GateGuard: password auth + email-verification stub

All paths in this phase are under `/gateguard`.

### Task 1.1: DB migration — password + email-verification columns

**Files:**
- Create: `db/000011_add_password_and_email_verification.up.sql`
- Create: `db/000011_add_password_and_email_verification.down.sql`

**Interfaces:**
- Produces: columns `users.password_hash text`, `users.email_verified boolean NOT NULL DEFAULT false`, `users.email_verification_token text`, `users.email_verification_sent_at timestamp`.

- [ ] **Step 1: Write the up migration**

```sql
-- Password credentials + email-verification (stub) support.
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_sent_at timestamp without time zone;

CREATE INDEX IF NOT EXISTS user_email_verification_token_idx
    ON users USING btree(email_verification_token);
```

- [ ] **Step 2: Write the down migration**

```sql
DROP INDEX IF EXISTS user_email_verification_token_idx;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
```

- [ ] **Step 3: Apply against a local gateguard DB and verify columns exist**

Run: `migrate -path db -database "$GATEGUARD_DB_URL" up` then `psql "$GATEGUARD_DB_URL" -c '\d users'`
Expected: the four new columns are listed.

- [ ] **Step 4: Commit**

```bash
git add gateguard/db/000011_add_password_and_email_verification.up.sql gateguard/db/000011_add_password_and_email_verification.down.sql
git commit -m "feat(gateguard): migration for password_hash + email verification columns"
```

### Task 1.2: `password` package — bcrypt hash/compare (TDD)

**Files:**
- Create: `internal/pkg/password/password.go`
- Test: `internal/pkg/password/password_test.go`
- Modify: `go.mod` (add `golang.org/x/crypto`)

**Interfaces:**
- Produces: `password.Hash(plain string) (string, error)`, `password.Compare(hash, plain string) error` (returns `nil` on match, `ErrMismatch` otherwise).

- [ ] **Step 1: Write the failing test**

```go
package password_test

import (
	"testing"

	"gateguard/internal/pkg/password"
)

func TestHashAndCompare(t *testing.T) {
	h, err := password.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if h == "correct horse battery staple" {
		t.Fatal("hash must not equal plaintext")
	}
	if err := password.Compare(h, "correct horse battery staple"); err != nil {
		t.Fatalf("expected match, got %v", err)
	}
	if err := password.Compare(h, "wrong"); err == nil {
		t.Fatal("expected mismatch error for wrong password")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/pkg/password/...`
Expected: FAIL (package/function not defined)

- [ ] **Step 3: Implement**

```go
// Package password hashes and verifies user passwords with bcrypt.
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrMismatch is returned when a plaintext password does not match the hash.
var ErrMismatch = errors.New("password mismatch")

// Cost is the bcrypt cost factor (>=10 per the security baseline).
const Cost = 12

// Hash returns a bcrypt hash of the plaintext password.
func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), Cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil if plain matches hash, ErrMismatch otherwise.
func Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return ErrMismatch
	}
	return nil
}
```

- [ ] **Step 4: Add the dependency and run tests**

Run: `cd gateguard && go get golang.org/x/crypto/bcrypt && go mod tidy && go test ./internal/pkg/password/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/pkg/password gateguard/go.mod gateguard/go.sum
git commit -m "feat(gateguard): bcrypt password hash/compare package"
```

### Task 1.3: Extend the `User` model with credential + verification fields

**Files:**
- Modify: `internal/models/user.go`

**Interfaces:**
- Produces: `User.PasswordHash string`, `User.EmailVerified bool`, `User.EmailVerificationToken string`, `User.EmailVerificationSentAt time.Time`. `PasswordHash` is NOT included in `Proto()`. `User.Proto()` gains nothing that leaks the hash; add `email_verified` only after the proto field exists (Task 1.4).

- [ ] **Step 1: Add struct fields** (after `TrialUsed`, before `CreatedOrRestored`)

```go
	PasswordHash            string    `pg:"password_hash"`
	EmailVerified           bool      `pg:"email_verified,use_zero"`
	EmailVerificationToken  string    `pg:"email_verification_token"`
	EmailVerificationSentAt time.Time `pg:"email_verification_sent_at"`
```

- [ ] **Step 2: Confirm `Proto()` does NOT serialize `PasswordHash`** (leave `Proto()` untouched in this task — it already omits it).

- [ ] **Step 3: Build**

Run: `cd gateguard && go build ./internal/models/...`
Expected: builds clean

- [ ] **Step 4: Commit**

```bash
git add gateguard/internal/models/user.go
git commit -m "feat(gateguard): add password_hash + email verification fields to User"
```

### Task 1.4: Proto — add credentialed RPCs + email-verification stub RPCs

**Files:**
- Modify: `protocols/gateguard/service_gateguard.proto`
- Regenerate: `protocols/gateguard/*.pb.go` (gitignored)

**Interfaces:**
- Produces (gRPC): `SignUpWithPassword(SignUpRequest) → TokenResponse`, `SignInWithPassword(PasswordSignInRequest) → TokenResponse`, `RequestEmailVerification(EmailRequest) → Empty`, `VerifyEmail(VerifyEmailRequest) → Empty`. New messages `SignUpRequest{email,name,password}`, `PasswordSignInRequest{email,password}`, `VerifyEmailRequest{email,token}`. (`EmailRequest`, `Empty`, `TokenResponse` already exist.)

- [ ] **Step 1: Add the RPCs to `service GateguardService`** (after `CheckAuth`)

```proto
  // SignUpWithPassword creates a credentialed account and returns a session JWT.
  rpc SignUpWithPassword(SignUpRequest) returns(TokenResponse);

  // SignInWithPassword verifies a password and returns a session JWT.
  rpc SignInWithPassword(PasswordSignInRequest) returns(TokenResponse);

  // RequestEmailVerification (STUB) issues + persists a token; does not email yet.
  rpc RequestEmailVerification(EmailRequest) returns(Empty);

  // VerifyEmail (STUB) marks the account verified if the token matches.
  rpc VerifyEmail(VerifyEmailRequest) returns(Empty);
```

- [ ] **Step 2: Add the request messages** (near `EmailRequest`)

```proto
message SignUpRequest {
  string email    = 1;
  string name     = 2;
  string password = 3;
}

message PasswordSignInRequest {
  string email    = 1;
  string password = 2;
}

message VerifyEmailRequest {
  string email = 1;
  string token = 2;
}
```

- [ ] **Step 3: Install codegen plugins (once) and regenerate**

```bash
cd gateguard
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
PATH="$(go env GOPATH)/bin:$PATH" make generate-proto
```
Expected: `protocols/gateguard/*.pb.go` regenerated with `SignUpWithPassword` etc.

- [ ] **Step 4: Verify the generated server interface includes the new methods**

Run: `cd gateguard && grep -c "SignUpWithPassword" protocols/gateguard/*grpc.pb.go`
Expected: ≥ 1

- [ ] **Step 5: Commit** (generated files stay gitignored in GateGuard; commit only the `.proto`)

```bash
git add gateguard/protocols/gateguard/service_gateguard.proto
git commit -m "feat(gateguard): proto for password sign-up/in + email-verify stub"
```

### Task 1.5: Repository — fetch/update by verification token (TDD where logic exists)

**Files:**
- Modify: `internal/repository/*.go` (add a `GetUser` "by token" selector + ensure `UpdateUserBy` can set the new columns)

**Interfaces:**
- Consumes: existing `repository.Email`, `repository.GetUser`, `repository.UpdateUserBy(ctx, user, by, columns...)`.
- Produces: `repository.EmailVerificationToken` selector constant so the service can `GetUser(ctx, user, repository.EmailVerificationToken)`.

- [ ] **Step 1: Add the selector constant** next to the existing `Email` selector (match the file/idiom used there).

```go
// EmailVerificationToken selects a user by their pending verification token.
EmailVerificationToken Selector = "email_verification_token"
```

- [ ] **Step 2: Ensure the selector is honored** in the `WHERE` switch used by `GetUser`/`UpdateUserBy` (add a `case EmailVerificationToken:` mirroring `case Email:`).

- [ ] **Step 3: Build**

Run: `cd gateguard && go build ./internal/repository/...`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add gateguard/internal/repository
git commit -m "feat(gateguard): repository selector for email verification token"
```

### Task 1.6: Service — `SignUpWithPassword` / `SignInWithPassword` (TDD)

**Files:**
- Create: `internal/service/sign_in_password.go`
- Test: `internal/service/tests/sign_in_password_test.go` (follow the existing `internal/service/tests` mock style)

**Interfaces:**
- Consumes: `u.repository` (`GetUser`, `CreateUser`, `UpdateUserBy`), `u.createJWT(user)`, `password.Hash/Compare`, `models.User`.
- Produces (methods on `*UsersService`):
  - `SignUpWithPassword(ctx, email, name, plain string) (token []byte, user *models.User, err error)`
  - `SignInWithPassword(ctx, email, plain string) (token []byte, user *models.User, err error)`
  - Sentinels: `ErrUserAlreadyExists`, `ErrInvalidCredentials`.

- [ ] **Step 1: Write failing tests** (happy sign-up, duplicate sign-up, good sign-in, bad password)

```go
func TestSignUpWithPassword_CreatesHashedUser(t *testing.T) {
	// repo.GetUser → ErrUserNotFound; repo.CreateUser captures user; createJWT stubbed.
	// assert: returned user has non-empty PasswordHash != plaintext, Status=Active, Role=Common,
	//         EmailVerified=false, and a non-empty token is returned.
}
func TestSignUpWithPassword_DuplicateReturnsErr(t *testing.T) {
	// repo.GetUser returns an existing user with a PasswordHash → ErrUserAlreadyExists.
}
func TestSignInWithPassword_GoodPassword(t *testing.T) {
	// repo.GetUser returns user whose PasswordHash = bcrypt("pw"); SignIn("pw") → token, no err.
}
func TestSignInWithPassword_BadPassword(t *testing.T) {
	// same user; SignIn("nope") → ErrInvalidCredentials, no token.
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd gateguard && go test ./internal/service/...`
Expected: FAIL (methods undefined)

- [ ] **Step 3: Implement**

```go
package service

import (
	"context"
	"errors"
	"fmt"

	"gateguard/internal/models"
	"gateguard/internal/pkg/password"
	"gateguard/internal/repository"
)

var (
	// ErrUserAlreadyExists is returned when registering an email that already has a password.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidCredentials is returned for an unknown email or wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func (u *UsersService) SignUpWithPassword(ctx context.Context, email, name, plain string) ([]byte, *models.User, error) {
	existing := &models.User{Email: email}
	err := u.repository.GetUser(ctx, existing, repository.Email)
	if err == nil && existing.PasswordHash != "" {
		return nil, nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("lookup user %s: %w", email, err)
	}

	hash, err := password.Hash(plain)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	token := newVerificationToken() // see Task 1.7
	user := &models.User{
		Email:                  email,
		Name:                   name,
		Status:                 models.UserActive,
		Role:                   models.UserRoleCommon,
		PasswordHash:           hash,
		EmailVerified:          false,
		EmailVerificationToken: token,
	}

	if errors.Is(err, repository.ErrUserNotFound) {
		if err := u.repository.CreateUser(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("create user %s: %w", email, err)
		}
		user.CreatedOrRestored = true
	} else {
		// Pre-existing passwordless account (e.g. demo-login): attach credentials.
		user.UUID = existing.UUID
		if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
			"password_hash", "email_verification_token", "email_verified", "name"); err != nil {
			return nil, nil, fmt.Errorf("attach credentials %s: %w", email, err)
		}
	}

	u.sendVerificationStub(ctx, user) // Task 1.7

	jwt, err := u.createJWT(user)
	if err != nil {
		return nil, nil, fmt.Errorf("create session token: %w", err)
	}
	return jwt, user, nil
}

func (u *UsersService) SignInWithPassword(ctx context.Context, email, plain string) ([]byte, *models.User, error) {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("lookup user %s: %w", email, err)
	}
	if user.PasswordHash == "" || password.Compare(user.PasswordHash, plain) != nil {
		return nil, nil, ErrInvalidCredentials
	}
	jwt, err := u.createJWT(user)
	if err != nil {
		return nil, nil, fmt.Errorf("create session token: %w", err)
	}
	return jwt, user, nil
}
```

- [ ] **Step 4: Run tests**

Run: `cd gateguard && go test ./internal/service/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/sign_in_password.go gateguard/internal/service/tests
git commit -m "feat(gateguard): password sign-up/sign-in service methods"
```

### Task 1.7: Service — email-verification STUB (TDD)

**Files:**
- Create: `internal/service/email_verification.go`
- Test: `internal/service/tests/email_verification_test.go`

**Interfaces:**
- Produces on `*UsersService`:
  - `newVerificationToken() string` (random, URL-safe)
  - `sendVerificationStub(ctx, *models.User)` — logs "[STUB] would email verification link" (uses `u.log`); does NOT send.
  - `RequestEmailVerification(ctx, email string) error` — regenerates + persists token, calls stub.
  - `VerifyEmail(ctx, email, token string) error` — matches token, sets `EmailVerified=true`, clears token. Sentinel `ErrVerificationTokenInvalid`.

- [ ] **Step 1: Write failing tests**

```go
func TestVerifyEmail_MatchingToken_SetsVerified(t *testing.T) {
	// repo.GetUser(byEmail) returns user with EmailVerificationToken="tok";
	// VerifyEmail(email,"tok") → UpdateUserBy called with email_verified=true; no err.
}
func TestVerifyEmail_WrongToken_Errors(t *testing.T) {
	// token mismatch → ErrVerificationTokenInvalid; no update.
}
```

- [ ] **Step 2: Run to verify failure** — `cd gateguard && go test ./internal/service/...` → FAIL

- [ ] **Step 3: Implement**

```go
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

// ErrVerificationTokenInvalid is returned when an email/token pair does not match.
var ErrVerificationTokenInvalid = errors.New("verification token invalid")

func newVerificationToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// sendVerificationStub is a NON-PRODUCTION stub: it logs instead of emailing.
func (u *UsersService) sendVerificationStub(ctx context.Context, user *models.User) {
	u.log.WarnCtx(ctx, fmt.Sprintf(
		"[STUB] email verification not sent (no mailer wired). email=%s token=%s",
		user.Email, user.EmailVerificationToken))
}

func (u *UsersService) RequestEmailVerification(ctx context.Context, email string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	user.EmailVerificationToken = newVerificationToken()
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "email_verification_token"); err != nil {
		return fmt.Errorf("persist token %s: %w", email, err)
	}
	u.sendVerificationStub(ctx, user)
	return nil
}

func (u *UsersService) VerifyEmail(ctx context.Context, email, token string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	if token == "" || user.EmailVerificationToken != token {
		return ErrVerificationTokenInvalid
	}
	user.EmailVerified = true
	user.EmailVerificationToken = ""
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "email_verified", "email_verification_token"); err != nil {
		return fmt.Errorf("mark verified %s: %w", email, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests** — `cd gateguard && go test ./internal/service/...` → PASS

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/email_verification.go gateguard/internal/service/tests
git commit -m "feat(gateguard): email-verification STUB (token persisted, send logged)"
```

### Task 1.8: gRPC server handlers for the four new RPCs

**Files:**
- Create: `internal/server/auth_password.go`
- Test: `internal/server/tests/auth_password_test.go` (follow existing server-test style)

**Interfaces:**
- Consumes: `h.srv.SignUpWithPassword/SignInWithPassword/RequestEmailVerification/VerifyEmail`, generated `proto.SignUpRequest` etc.
- Produces: handler methods on `*GateguardHandlers` matching the generated server interface.

- [ ] **Step 1: Implement the handlers**

```go
package server

import (
	"context"
	"fmt"

	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) SignUpWithPassword(ctx context.Context, req *proto.SignUpRequest) (*proto.TokenResponse, error) {
	token, user, err := h.srv.SignUpWithPassword(ctx, req.Email, req.Name, req.Password)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "SignUpWithPassword failed")
		return nil, fmt.Errorf("sign up: %w", err)
	}
	return &proto.TokenResponse{Token: token, UserCreatedOrRestored: user.CreatedOrRestored}, nil
}

func (h *GateguardHandlers) SignInWithPassword(ctx context.Context, req *proto.PasswordSignInRequest) (*proto.TokenResponse, error) {
	token, _, err := h.srv.SignInWithPassword(ctx, req.Email, req.Password)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "SignInWithPassword failed")
		return nil, fmt.Errorf("sign in: %w", err)
	}
	return &proto.TokenResponse{Token: token}, nil
}

func (h *GateguardHandlers) RequestEmailVerification(ctx context.Context, req *proto.EmailRequest) (*proto.Empty, error) {
	if err := h.srv.RequestEmailVerification(ctx, req.Email); err != nil {
		h.log.ErrorCtx(ctx, err, "RequestEmailVerification failed")
		return nil, fmt.Errorf("request verification: %w", err)
	}
	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) VerifyEmail(ctx context.Context, req *proto.VerifyEmailRequest) (*proto.Empty, error) {
	if err := h.srv.VerifyEmail(ctx, req.Email, req.Token); err != nil {
		h.log.ErrorCtx(ctx, err, "VerifyEmail failed")
		return nil, fmt.Errorf("verify email: %w", err)
	}
	return &proto.Empty{}, nil
}
```

- [ ] **Step 2: Update the service interface** that `GateguardHandlers` depends on (the `IUsersService`/`srv` interface in `internal/server`) to include the four new methods, and regenerate its mock (`mockery`) if one exists.

- [ ] **Step 3: Build + test the module**

Run: `cd gateguard && go build ./... && go test ./...`
Expected: clean + PASS (mocks regenerated)

- [ ] **Step 4: Commit**

```bash
git add gateguard/internal/server gateguard/internal/service/mocks gateguard/internal/server/mocks
git commit -m "feat(gateguard): gRPC handlers for password auth + email verify"
```

### Task 1.9: Build the GateGuard image from the vendored source

- [ ] **Step 1: Build the image** — `docker build -t gateguard:local ./gateguard`
  Expected: succeeds (force IPv4 in the Dockerfile if the box's broken IPv6 bites — see HANDOFF).
- [ ] **Step 2: Commit any Dockerfile fix** if required.

---

## Phase 2 — Lia + frontend wiring

### Task 2.1: Sync + regenerate Lia's vendored proto

**Files:**
- Modify: `backend/protocols/gateguard/service_gateguard.proto` (mirror the RPCs/messages from Task 1.4)
- Regenerate + **force-add** `backend/protocols/gateguard/*.pb.go`

- [ ] **Step 1:** Copy the new RPCs + messages into Lia's `service_gateguard.proto`.
- [ ] **Step 2:** Regenerate: `cd backend && make generate-api` is for swagger — for the gateguard proto run the same `protoc` invocation as GateGuard's `generate-proto`, then `git add -f backend/protocols/gateguard/*.pb.go`.
- [ ] **Step 3:** `cd backend && go build ./...` → clean.
- [ ] **Step 4:** Commit.

### Task 2.2: Extend Lia's `Signer` to call the new RPCs

**Files:**
- Modify: `backend/internal/http/auth/signer.go` (the `ggClient` interface + `gatekeeperSigner`)

**Interfaces:**
- Produces: `Signer.SignUp(ctx, email, name, password string) (string, error)`, `Signer.SignIn(ctx, email, password string) (string, error)` — both return the JWT string. Keep the existing demo `SignIn(ctx, email, name)` or rename to avoid clash (use `SignInPassword`).

- [ ] Steps: add the two methods calling `client.SignUpWithPassword` / `client.SignInWithPassword`; map gRPC errors → typed errors (409 for exists, 401 for invalid creds). Build + unit test the signer seam with a fake `ggClient`. Commit.

### Task 2.3: Lia HTTP — `POST /auth/register` + `POST /auth/login`

**Files:**
- Modify: `backend/api/swagger.yaml` (add the two operations + `RegisterInput{email,name,password}`, `LoginInput{email,password}`, reuse `DemoLoginResponse`)
- Run: `cd backend && make generate-api`
- Create: `backend/internal/http/handlers/auth_password.go` (handlers)
- Modify: `backend/internal/http/module.go` (wire handlers, like `DemoLogin`)
- Test: handler tests mirroring `events_test.go` style

**Interfaces:**
- `POST /auth/register` → 200 `{token}` / 409 exists / 400 invalid. `POST /auth/login` → 200 `{token}` / 401.

- [ ] Steps: swagger first → regenerate → write handlers calling `Signer.SignUp/SignInPassword` → register in module → tests (200/409/401) → build/vet/test → commit. Password min length (e.g. ≥ 8) validated server-side; never logged.

### Task 2.4: Frontend — password + register/login in the login modal

**Files:**
- Modify: `frontend/components/AuthButton.tsx` (LoginModal: add password field + a "Регистрация / Вход" toggle)
- Modify: `frontend/lib/api.ts` (add `register(email,name,password)` + `login(email,password)`; keep `demoLogin` for the dev path)
- Modify: `frontend/lib/auth-context.tsx` (expose `register`/`login`)

- [ ] Steps: add controlled password input (`type="password"`, min 8, RU labels «Пароль», «Регистрация», «Вход»); on submit call the right endpoint; store the returned token via existing `setSession`; surface 409/401 errors in RU. `pnpm lint` + `pnpm build` clean. Commit.

### Task 2.5: Frontend — email-verification stub UX

**Files:**
- Create: `frontend/app/auth/verify/page.tsx` (reads `?email=&token=`, calls a Lia `POST /auth/verify-email` proxy → GateGuard `VerifyEmail`)
- Modify: a small banner component shown when the signed-in user is unverified: «Подтвердите эл. почту (демо: ссылка в логах сервера)».

- [ ] Steps: add the Lia `POST /auth/verify-email` + `POST /auth/request-verification` endpoints (swagger + handlers + signer methods) mirroring Task 2.3; build the page + banner; clearly label the stub. Commit.

---

## Phase 2.5 — Surface user/event data (request #2: "user data regarding the events created")

Grounded in the current Lia code (verified 2026-06-25): events carry `OrganizerID` = the authed user's UUID (`backend/internal/http/handlers/events.go:105`), but `EventToAPI` exposes only the bare UUID, and `events.ListFilter` has only `Status`+`Limit`. So two gaps: (a) no creator info on event responses, (b) no "my events" query.

> **Privacy (compliance):** event responses are public. Expose the organizer's **display name** (+ optional avatar) only — **never email** — matching `design_agent_prompt.md` §5.14.8 `/me/profile` (email is private). This keeps the public surface aligned with the documented privacy model.

### Task 2.5.1: Backend — embed organizer (creator) display data on events
- **Swagger** (`backend/api/swagger.yaml`): add an `Organizer` object `{uuid, name, avatar_url}` and an `organizer` field on `Event`. Run `make generate-all` first (generated `internal/http/models` is gitignored — must exist to build), then `make generate-api`.
- **Repository** (`backend/internal/events/repository.go`): add `loadOrganizers(events)` mirroring `loadVenues`/`loadCover` — batch-load `users` by the set of `organizer_id`s (single query, no N+1), populate a domain `Organizer` field.
- **Formatter** (`backend/internal/http/formatter/event.go`): in `EventToAPI`, set `out.Organizer` from the loaded user (name + avatar_url via storage; NO email).
- **Test**: list/detail include `organizer.name`; never include email.

### Task 2.5.2: Backend — "my events" query
- **Repository**: add `OrganizerID uuid.UUID` to `ListFilter`; when non-nil, `query.Where("organizer_id = ?", id)`.
- **Endpoint**: add authenticated `GET /events/mine` (swagger op `listMyEvents`, `security: [jwt]`, handler takes `principal *apimodels.User`, sets `filter.OrganizerID = principal.UUID`, returns all statuses incl. drafts so the owner sees their own unpublished events). Register in `module.go`.
- **Test**: authed → only the caller's events (incl. draft); anon → 401.

### Task 2.5.3: Frontend — creator display + "my events"
- **API client** (`frontend/lib/api.ts` + `lib/types.ts`): map `event.organizer` → `LiaEvent.organizer {uuid,name,avatarUrl}`; add `fetchMyEvents()` (Bearer) hitting `GET /events/mine`.
- **Event card + detail**: show «Организатор: {name}» (+ avatar if present).
- **"Мои события"**: a list (under `/me` once Phase 3.1 lands, or a `/events/mine` page in the interim) using `fetchMyEvents()`. This is the slice that closes the original "couldn't find my created event" complaint — owners see their drafts + published here regardless of the discovery feed's `status=published` filter.

## Phase 3 — `/me/*` personal area (roadmap → separate plans)

Per the design spec `docs/design_agent_prompt.md` §5.14.8 + §4.1. Build in dependency order. Each bullet becomes its own plan under `docs/superpowers/plans/` because each depends on a distinct backend domain. **Unblocked today** vs **blocked on a new domain** is called out so nothing is silently skipped.

1. **`/me` shell + nav + auth gating** *(unblocked)* — App Router layout under `frontend/app/me/`, avatar dropdown (header) per spec line 619, redirect `/profile`→`/me/profile`, `/saved`→`/me/saved`.
2. **`/me/profile` — public profile** *(mostly unblocked)* — display name + avatar (reuse `users.avatar_file_id`, migration 000011 in Lia already adds the column) + optional bio/topics columns (new Lia migration). Backend: `GET/PATCH /users/me`.
3. **"My events" list** *(unblocked — see note)* — Lia already stamps `events.organizer_id` from the authenticated principal (`backend/internal/http/handlers/events.go:103`). Add `GET /events?mine=true` (filter `organizer_id = principal`) + a list under `/me` ("Мои события"). **This is the slice that directly fixes "I couldn't find my created event."** Consider doing this first, even before Phase 1, as a quick win.
4. **`/me/settings`** *(partially blocked)* — account (name/email via `/users/me`), security (sessions need GateGuard session listing — blocked), notification matrix (blocked on notifications domain).
5. **`/me/saved`** *(blocked)* — needs a `saved_events` domain (table + `POST/DELETE /events/{id}/save` + list).
6. **`/me/follows`** *(blocked)* — needs `follows` + organizer/host entities.
7. **`/me/practices` + `/me/applications`** *(blocked)* — need the RSVP/applications domain (the next P0 slice in HANDOFF).
8. **`/me/notifications` + `/me/interests`** *(blocked)* — need notifications + interests domains.

> **Scope note (writing-plans scope-check):** items 5–8 each introduce an independent subsystem and must be brainstormed + planned separately before implementation. This plan fully specifies Phases 0–2 (passwords + email-verify stub) and items 1–3 of Phase 3.

---

## Implementation notes (discovered during the build, 2026-06-25)

- **GateGuard codegen pin:** GateGuard pins `google.golang.org/grpc v1.56.3`. `protoc-gen-go-grpc` **must be v1.3.0** (emits `SupportPackageIsVersion7`); the latest (v1.6.2) emits `grpc.StaticMethod`/`SupportPackageIsVersion9` which need grpc ≥ v1.64 and break the build. Install: `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0`. `protoc-gen-go@latest` is fine. `protoc 34.1` works.
- **Repository selector not needed:** `VerifyEmail`/`RequestEmailVerification` fetch by `repository.Email` and compare the token in Go, so planned Task 1.5 (a token `UserGetter`) was **skipped** — no repo change required.
- **bcrypt without dependency churn:** `golang.org/x/crypto` was already an (indirect) dep at v0.35.0, which ships `bcrypt`. It was promoted to direct and **pinned at v0.35.0** (a blind `go get` bumps it to v0.53 and drags net/sync/sys/text up — avoid).
- **Pre-existing test drift — RESOLVED enough to compile+run.** The vendored snapshot shipped with broken tests unrelated to passwords; fixed mechanically:
  - `i_users_service.go` mock was missing `SetExtendedSession` (pre-existing) + the 4 new password methods — **hand-added** (mockery v2.43 won't build under Go 1.26; the mock is marked with a "hand-added" comment, regen properly on a Go 1.23/1.24 box).
  - `NewUsersService(...)` test call passed 7 args, signature wants 9 → added `, 0, 0` (`maxWeeklyInvitesNum`, `invitesTTLHours`) in `internal/service/tests/user_test.go:61`.
  - `clog.NewCustomLogger(os.Stdout, slog.LevelDebug, false)` → `clog.LevelDebug` (the pinned `scriptorium@v0.0.25` `clog.Level` is a distinct type, not a `slog.Level` alias) in `internal/service/tests/user_test.go` + `internal/server/tests/suite_test.go`; dropped the now-unused `log/slog` import.
  - **Result:** `go build ./...` green; `go test -vet=off ./...` passes everything EXCEPT `internal/service/tests` `Test_ReactToInvitation_Success` — a **pre-existing** org-invitation logic/test mismatch (code calls `UpdateUserRoleInOrganization`, test only mocks `AddUserToOrganization`), unrelated to this work. Left as-is; flag to GateGuard owners.
- **`go vet` noise:** Go 1.24+ flags "non-constant format string" on the codebase's `clog.WarnCtx/ErrorCtx(... fmt.Sprintf(...))` idiom (pervasive in `user.go`/`sign_in.go`; new files match it). Run tests with `-vet=off`, or have GateGuard adopt constant-format logging repo-wide. Not introduced by this work.

## Self-Review

- **Spec coverage:** Password sign-up/in → Tasks 1.1–2.4. Email-verification stub (explicitly requested) → Tasks 1.7, 2.5. Vendoring GateGuard → Phase 0. `/me/*` suite (`design_agent_prompt.md` §5.14.8) → Phase 3 roadmap, with the unblocked "my events" slice (the user's missing-event complaint) called out at 3.3.
- **Two-proto-copy hazard:** encoded as a Global Constraint and split across Task 1.4 (GateGuard) + Task 2.1 (Lia, force-add).
- **Credential safety:** bcrypt cost 12, `PasswordHash` excluded from `Proto()` + responses, never logged — Global Constraints + Task 1.3/1.6.
- **Type consistency:** service methods `SignUpWithPassword`/`SignInWithPassword` and proto messages `SignUpRequest`/`PasswordSignInRequest`/`VerifyEmailRequest` are referenced identically in Tasks 1.4, 1.6, 1.8, 2.1, 2.2.
- **Known follow-up:** email-verification is a STUB (no mailer) — must be replaced before real prod; GateGuard's existing SMTP `notificator` is the natural wiring point.
