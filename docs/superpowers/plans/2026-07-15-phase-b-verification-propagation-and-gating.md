# Phase B + D — Verification Propagation, Action Gating & Verify UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Propagate `email_verified` from GateGuard into the Lia app, block unverified users from creating/editing events, RSVP/applying, and submitting complaints, expose Lia proxy endpoints + a `/auth/verify` page so users can verify, and add a defensive verification check to GateGuard's legacy invite-accept paths.

**Architecture:** GateGuard is the source of truth for `email_verified`; it rides the existing `CheckAuth` → `gg.User` proto (new field 13) into Lia's `Claims` → `domain.User` (treated as a cache like `role`, not persisted locally). A `RequireVerified` gate returns HTTP 403 `email_not_verified` on the gated write actions. The frontend adds a 6-digit verify page and shows an interstitial on 403.

**Tech Stack:** Go, protobuf/gRPC, go-swagger (go-openapi), go-pg; Next.js app-router, React Query, TypeScript.

## Global Constraints

- Two Go modules: `gateguard/` and `backend/`. Run each module's commands from its own directory.
- Proto `User` next free field tag is **13**; add `bool email_verified = 13;` to **both** copies: `backend/protocols/gateguard/models_gateguard.proto` and `gateguard/protocols/gateguard/models_gateguard.proto`.
- Regen commands: GateGuard `cd gateguard && make generate-proto`; Lia `cd backend && make proto-generate-all`; Lia swagger `cd backend && make generate-api`.
- `email_verified` is carried on `Claims` → `domain.User` only — **do NOT add a local DB migration/column** for it (GateGuard owns it, exactly as `role` is a cache).
- 403 body shape (verbatim): `{"code":"email_not_verified","message":"Подтвердите электронную почту, чтобы продолжить"}`.
- Gated actions: create event, update/publish event, RSVP/apply, submit complaint. NOT gated: follows, all reads, auth endpoints, the verify endpoints themselves.
- Mock-auth principal (`HTTP_MOCK_AUTH`) must be `EmailVerified=true` so local/mock runs are unaffected.
- Frontend copy is Russian; reuse `inputClass` and the `LoginModal` submit/busy/error pattern from `frontend/components/AuthButton.tsx`.

---

### Task 1: Add `email_verified` to proto + GateGuard mapping

**Files:**
- Modify: `backend/protocols/gateguard/models_gateguard.proto:12` and `gateguard/protocols/gateguard/models_gateguard.proto:12`
- Regenerate: both `*.pb.go`
- Modify: `gateguard/internal/models/user.go` (`Proto()` ~:89-109, `UserFromProto()` ~:111-129)

**Interfaces:**
- Produces: `proto.User.EmailVerified bool` (field 13); GateGuard `Proto()` sets it, `UserFromProto()` reads it.

- [ ] **Step 1: Add the proto field (both copies)**

In each `models_gateguard.proto`, inside `message User`, after `bool trial_used = 12;` add:

```proto
  bool           email_verified   = 13;
```

- [ ] **Step 2: Regenerate stubs**

Run:
```bash
cd gateguard && make generate-proto
cd ../backend && make proto-generate-all
```
Expected: both `models_gateguard.pb.go` now contain an `EmailVerified` field + `GetEmailVerified()` getter. Confirm:
```bash
grep -n "EmailVerified" gateguard/protocols/gateguard/models_gateguard.pb.go backend/protocols/gateguard/models_gateguard.pb.go | head
```

- [ ] **Step 3: Wire GateGuard model↔proto**

In `gateguard/internal/models/user.go`, in the `Proto()` return literal add `EmailVerified: u.EmailVerified,` and in `UserFromProto()` return literal add `EmailVerified: pb.EmailVerified,`.

- [ ] **Step 4: Build both modules**

Run: `cd gateguard && go build ./... && cd ../backend && go build ./...`
Expected: both succeed.

- [ ] **Step 5: Commit**

```bash
git add gateguard/protocols gateguard/internal/models/user.go backend/protocols
git commit -m "feat(proto): add User.email_verified (field 13) + gateguard mapping"
```

---

### Task 2: Propagate `email_verified` into Lia Claims + domain.User

**Files:**
- Modify: `backend/internal/http/auth/auth.go` (`Claims` :22-28, `ensureUser` :116-142, `mockDomainUser` :180-188)
- Modify: `backend/internal/http/auth/gatekeeper.go` (`Validate` :72)
- Modify: `backend/internal/models/user.go` (domain User :15-26)
- Test: `backend/internal/http/auth/auth_test.go` (extend)

**Interfaces:**
- Consumes: `gg.User.EmailVerified` (Task 1).
- Produces: `auth.Claims.EmailVerified bool`; `domain.User.EmailVerified bool`; both populated end-to-end.

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/http/auth/auth_test.go` (mirror the existing fake-validator test setup in that file — reuse its fake `TokenValidator`/`ggClient`; match the constructor the existing tests use):

```go
func TestAuthenticate_PropagatesEmailVerified(t *testing.T) {
	// Arrange a fake validator returning a verified user (copy the fake wiring
	// used by the existing Authenticate tests in this file).
	claims := &Claims{Subject: "s", Email: "v@example.com", Name: "V", Role: "common", EmailVerified: true}
	a := newAuthWithFakeValidator(t, claims) // helper pattern already present or add one mirroring existing tests

	u, err := a.Authenticate("Bearer tok")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !u.EmailVerified {
		t.Fatalf("expected domain user EmailVerified=true, got false")
	}
}
```

> If the existing test file builds `Auth` differently (e.g. via a fake that returns `*Claims`), match that exact construction. The assertion that matters is `domain.User.EmailVerified == claims.EmailVerified`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/auth/ -run TestAuthenticate_PropagatesEmailVerified -v`
Expected: FAIL — `Claims` has no `EmailVerified`, or `domain.User` has no `EmailVerified`.

- [ ] **Step 3: Add the fields + wiring**

1. `backend/internal/models/user.go` domain `User` struct — add:
   ```go
   EmailVerified bool `pg:"email_verified,use_zero"`
   ```
   (Field is claim-sourced; the `pg` tag is harmless — no column is selected/written for it because repo methods use explicit columns. If a `SELECT *` path errors on the missing column, drop the `pg` tag to `pg:"-"`.)

2. `backend/internal/http/auth/auth.go` `Claims` struct — add `EmailVerified bool`.

3. `backend/internal/http/auth/gatekeeper.go` `Validate` return — change to:
   ```go
   return &Claims{Subject: subject, Email: u.Email, Name: u.Name, Role: u.Role.String(), EmailVerified: u.EmailVerified}, nil
   ```

4. `backend/internal/http/auth/auth.go` `ensureUser` — its signature/body currently takes `(ctx, email, name, role)`. Add an `emailVerified bool` parameter, thread it from `Authenticate` (`claims.EmailVerified`), and set `u.EmailVerified = emailVerified` on the provisioned/synced user (mirror how `role` is set). Update the `Authenticate` call site (`:95`) accordingly.

5. `mockDomainUser` (`:180-188`) — add `EmailVerified: true` to the returned `domain.User`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/auth/ -v && go build ./...`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/auth/ backend/internal/models/user.go
git commit -m "feat(auth): propagate email_verified into Claims + domain.User"
```

---

### Task 3: Expose `email_verified` on the go-openapi principal

**Files:**
- Modify: `backend/api/swagger.yaml` (`User` definition ~:1197-1228)
- Regenerate: `backend/internal/http/models/user.go` etc. via `make generate-api`
- Modify: `backend/internal/http/auth/auth.go` `toPrincipal` (:156-166)

**Interfaces:**
- Produces: `apimodels.User.EmailVerified bool` populated from `domain.User.EmailVerified`.

- [ ] **Step 1: Add the swagger field**

In `backend/api/swagger.yaml`, in the `User` definition's `properties`, add:

```yaml
        email_verified:
          type: boolean
```

- [ ] **Step 2: Regenerate the API models**

Run: `cd backend && make swagger-validate && make generate-api`
Expected: `backend/internal/http/models/user.go` gains an `EmailVerified bool \`json:"email_verified,omitempty"\`` field.

- [ ] **Step 3: Populate it in toPrincipal**

In `backend/internal/http/auth/auth.go` `toPrincipal`, add `EmailVerified: u.EmailVerified,` to the returned `&models.User{...}`.

- [ ] **Step 4: Build**

Run: `cd backend && go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add backend/api/swagger.yaml backend/internal/http/models/ backend/internal/http/server/ backend/internal/http/auth/auth.go
git commit -m "feat(api): expose email_verified on the swagger User principal"
```

---

### Task 4: `RequireVerified` gate helpers

**Files:**
- Create: `backend/internal/http/handlers/verified_gate.go`
- Test: `backend/internal/http/handlers/verified_gate_test.go`

**Interfaces:**
- Produces:
  - `func UnverifiedResponder() middleware.Responder` — writes 403 + the standard JSON body (for go-openapi handlers).
  - `func IsVerified(p *apimodels.User) bool` — nil-safe.

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/http/handlers/verified_gate_test.go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"

	"github.com/Pashteto/lia/internal/http/handlers"
	apimodels "github.com/Pashteto/lia/internal/http/models"
)

func TestIsVerified(t *testing.T) {
	if handlers.IsVerified(nil) {
		t.Fatal("nil principal must be unverified")
	}
	if handlers.IsVerified(&apimodels.User{}) {
		t.Fatal("zero-value principal must be unverified")
	}
	v := true
	if !handlers.IsVerified(&apimodels.User{EmailVerified: v}) {
		t.Fatal("verified principal must pass")
	}
}

func TestUnverifiedResponder(t *testing.T) {
	rr := httptest.NewRecorder()
	handlers.UnverifiedResponder().WriteResponse(rr, runtime.JSONProducer())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "email_not_verified") {
		t.Fatalf("body missing code: %s", rr.Body.String())
	}
}
```

> Adjust `apimodels.User{EmailVerified: v}` to whatever type the regenerated field uses (`bool` vs `*bool`). If it's `*bool`, use `&v`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/handlers/ -run "TestIsVerified|TestUnverifiedResponder" -v`
Expected: FAIL — undefined `handlers.IsVerified` / `handlers.UnverifiedResponder`.

- [ ] **Step 3: Implement**

```go
// backend/internal/http/handlers/verified_gate.go
package handlers

import (
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	apimodels "github.com/Pashteto/lia/internal/http/models"
)

const unverifiedBody = `{"code":"email_not_verified","message":"Подтвердите электронную почту, чтобы продолжить"}`

// IsVerified reports whether the principal has a verified email (nil-safe).
func IsVerified(p *apimodels.User) bool {
	return p != nil && p.EmailVerified
}

// UnverifiedResponder writes a 403 email_not_verified response.
func UnverifiedResponder() middleware.Responder {
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(unverifiedBody))
	})
}
```

> If the regenerated `EmailVerified` is `*bool`, change the check to `p != nil && p.EmailVerified != nil && *p.EmailVerified`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/handlers/ -run "TestIsVerified|TestUnverifiedResponder" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/handlers/verified_gate.go backend/internal/http/handlers/verified_gate_test.go
git commit -m "feat(api): RequireVerified gate helpers (403 email_not_verified)"
```

---

### Task 5: Gate the go-openapi write actions (create/update event, RSVP)

**Files:**
- Modify: `backend/internal/http/handlers/events.go` (`CreateEvent.Handle` ~:181)
- Modify: `backend/internal/http/handlers/events_update.go` (`UpdateEvent.Handle` ~:28)
- Modify: `backend/internal/http/handlers/rsvp.go` (`SignUp.Handle` ~:37)
- Test: `backend/internal/http/handlers/events_test.go` (extend)

**Interfaces:**
- Consumes: `IsVerified`, `UnverifiedResponder` (Task 4).

- [ ] **Step 1: Write the failing test (create event)**

Add to `backend/internal/http/handlers/events_test.go` (reuse the existing create-event test scaffolding in that file — the fake events service + params builder):

```go
func TestCreateEvent_UnverifiedForbidden(t *testing.T) {
	h := newCreateEventHandlerForTest(t) // existing helper/pattern in this test file
	params := newValidCreateEventParams(t) // existing helper/pattern
	principal := &apimodels.User{ /* EmailVerified: false (zero) */ }

	resp := h.Handle(params, principal)

	rr := httptest.NewRecorder()
	resp.WriteResponse(rr, runtime.JSONProducer())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 for unverified, got %d", rr.Code)
	}
}
```

> Match the helper names to what `events_test.go` already uses to build the handler + params. If none exist, construct the handler with the fake service the other tests in the file use and build `eventsops.CreateEventParams` inline as they do.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/handlers/ -run TestCreateEvent_UnverifiedForbidden -v`
Expected: FAIL — create proceeds (non-403).

- [ ] **Step 3: Insert the gate in all three handlers**

In each handler, immediately after the existing `principal == nil` / `principal != nil` guard and before doing work, add:

`events.go` `CreateEvent.Handle` (after the `principal` nil check, ~:190):
```go
	if !IsVerified(principal) {
		return UnverifiedResponder()
	}
```

`events_update.go` `UpdateEvent.Handle` (after the nil guard, ~:32):
```go
	if !IsVerified(principal) {
		return UnverifiedResponder()
	}
```

`rsvp.go` `SignUp.Handle` (after the nil guard, ~:46):
```go
	if !IsVerified(principal) {
		return UnverifiedResponder()
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/http/handlers/ -v && go build ./...`
Expected: the new test PASSES; existing handler tests still PASS. If existing create/update/rsvp tests now fail with 403, update their fixtures to set the principal `EmailVerified=true` (they represent an authenticated, verified user).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/handlers/
git commit -m "feat(api): gate create/update event + RSVP on verified email"
```

---

### Task 6: Gate complaint submission (raw http)

**Files:**
- Modify: `backend/internal/http/complaints/handler.go` (`submit` ~:50-81)
- Test: `backend/internal/http/complaints/handler_test.go` (extend)

**Interfaces:**
- Consumes: `domain.User.EmailVerified` (Task 2). No swagger needed — this handler already has the domain user.

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/http/complaints/handler_test.go` (reuse the existing fake `Authenticate` dep + request builder in that file):

```go
func TestSubmit_UnverifiedForbidden(t *testing.T) {
	// Authenticate returns an unverified user.
	deps := Deps{
		Authenticate: func(string) (*domain.User, error) {
			return &domain.User{UUID: someUUID, EmailVerified: false}, nil
		},
		Complaints: fakeComplaints{}, // existing fake in this test file
	}
	h := NewHandler(deps)

	req := newComplaintRequest(t, "Bearer x") // existing helper/pattern
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "email_not_verified") {
		t.Fatalf("body missing code: %s", rr.Body.String())
	}
}
```

> Match `someUUID`, `fakeComplaints`, and `newComplaintRequest` to the identifiers the existing complaints tests use. The essential arrangement: `Authenticate` yields `EmailVerified:false`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/complaints/ -run TestSubmit_UnverifiedForbidden -v`
Expected: FAIL — submit proceeds.

- [ ] **Step 3: Insert the gate**

In `backend/internal/http/complaints/handler.go` `submit`, right after the `u := h.principal(r)` / `if u == nil { ... }` block (~:55), add:

```go
	if !u.EmailVerified {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"email_not_verified","message":"Подтвердите электронную почту, чтобы продолжить"}`))
		return
	}
```

> If this package already has a `writeErr`/`writeJSON` helper, use it instead of the inline write, matching the surrounding style.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/complaints/ -v`
Expected: PASS (existing tests may need their fake `Authenticate` to return `EmailVerified:true`).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/complaints/
git commit -m "feat(api): gate complaint submission on verified email"
```

---

### Task 7: Defensive verification check at GateGuard invite-accept (Phase D)

**Files:**
- Modify: `gateguard/internal/service/sign_in.go` (~:27, OAuth auto-accept)
- Modify: `gateguard/internal/service/react_to_invitation.go` (~:37-39 / `models.Accepted` case ~:68)
- Test: `gateguard/internal/service/tests/react_to_invitation_test.go` (extend)

**Interfaces:**
- Produces: `var ErrEmailNotVerified = errors.New("email not verified")`; accept transitions are blocked when the invitee is unverified.

- [ ] **Step 1: Write the failing test**

Add to `gateguard/internal/service/tests/react_to_invitation_test.go` (mirror the existing accept test in that file — same suite `s`, same mock wiring, but the invitee user has `EmailVerified:false`):

```go
func (s *UseCaseSuite) Test_ReactToInvitation_Accept_BlockedWhenUnverified() {
	// GetUser returns an active but unverified invitee.
	s.repo.EXPECT().
		GetUser(mock.Anything, mock.Anything, repository.Email).
		Run(func(_ context.Context, u *models.User, _ ...any) {
			u.Status = models.UserActive
			u.EmailVerified = false
		}).
		Return(nil).Once()

	_, err := s.service.ReactToInvitation(s.ctx, /* invitee email */ "x@example.com", models.Accepted, "REFCODE")
	s.Require().ErrorIs(err, service.ErrEmailNotVerified)
}
```

> Match `ReactToInvitation`'s exact signature and the surrounding mock expectations to the existing accept test (`react_to_invitation_test.go:40+`). Only the `EmailVerified:false` + expected `ErrEmailNotVerified` are new.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gateguard && go test ./internal/service/tests/ -run Test_ReactToInvitation_Accept_BlockedWhenUnverified -v`
Expected: FAIL — accept currently succeeds.

- [ ] **Step 3: Add the error + checks**

Add to `gateguard/internal/service/react_to_invitation.go` (near the other error vars) or a shared errors file:
```go
// ErrEmailNotVerified blocks invite acceptance by an unverified account.
var ErrEmailNotVerified = errors.New("email not verified")
```

In `react_to_invitation.go`, in the `models.Accepted` branch (before `UpdateUserRoleInOrganization`, ~:68), add:
```go
	if !inviteeUser.EmailVerified {
		return nil, ErrEmailNotVerified
	}
```
(Use the already-loaded invitee user variable name from the surrounding code.)

In `gateguard/internal/service/sign_in.go`, inside the `if user.RefCode != ""` acceptance block (after `getOrCreateUser`, before flipping the invitation to `Accepted`, ~:42), add:
```go
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}
```
(Return signature must match `SignInOAuth`'s — adjust the number of return values accordingly.)

- [ ] **Step 4: Run tests**

Run: `cd gateguard && go test ./internal/service/... -v && go build ./...`
Expected: new test PASSES; existing invite tests still PASS (set `EmailVerified:true` on their invitee fixtures where accept is expected to succeed).

- [ ] **Step 5: Commit**

```bash
git add gateguard/internal/service/
git commit -m "feat(gateguard): block invite acceptance for unverified emails (defensive)"
```

---

### Task 8: Lia verify proxy endpoints (`/auth/request-verification`, `/auth/verify-email`)

**Files:**
- Modify: `backend/internal/http/auth/gatekeeper.go` (`ggClient` interface :22-28)
- Modify: `backend/internal/http/auth/signer.go` (`Signer` interface + `gatekeeperSigner` impl)
- Create: `backend/internal/http/authverify/handler.go` (raw http mux, mounted ahead of swagger)
- Modify: `backend/internal/http/module.go` (build + dispatch, near complaints ~:290-395)
- Test: `backend/internal/http/auth/signer_test.go` (extend with a fake ggClient)

**Interfaces:**
- Consumes: generated `gg.EmailRequest{Email string}`, `gg.VerifyEmailRequest{Email, Token string}`, `gg.Empty`.
- Produces:
  - `Signer.RequestEmailVerification(ctx, email string) error`
  - `Signer.VerifyEmail(ctx, email, code string) error`
  - HTTP `POST /auth/request-verification` (body `{"email":"..."}`) and `POST /auth/verify-email` (body `{"email":"...","code":"..."}`).

- [ ] **Step 1: Extend the ggClient interface**

In `backend/internal/http/auth/gatekeeper.go`, add to the `ggClient` interface:
```go
	RequestEmailVerification(ctx context.Context, in *gg.EmailRequest, opts ...grpc.CallOption) (*gg.Empty, error)
	VerifyEmail(ctx context.Context, in *gg.VerifyEmailRequest, opts ...grpc.CallOption) (*gg.Empty, error)
```
(These already exist on the generated `gg.GateguardServiceClient`, so the concrete client satisfies the interface.)

- [ ] **Step 2: Write the failing test**

Add to `backend/internal/http/auth/signer_test.go` (reuse the fake `ggClient` used by existing signer tests; add the two methods to the fake):

```go
func TestSigner_RequestEmailVerification(t *testing.T) {
	fake := &fakeGGClient{} // existing fake in this test file; add fields to capture the call
	s := newSignerWithClient(fake)

	if err := s.RequestEmailVerification(context.Background(), "u@example.com"); err != nil {
		t.Fatalf("request: %v", err)
	}
	if fake.lastVerifyEmailReq != "u@example.com" {
		t.Fatalf("expected RPC called with email, got %q", fake.lastVerifyEmailReq)
	}
}
```

> Extend the existing `fakeGGClient` to implement the two new interface methods and record their inputs. Match the fake's existing style.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/auth/ -run TestSigner_RequestEmailVerification -v`
Expected: FAIL — `Signer` has no `RequestEmailVerification`.

- [ ] **Step 4: Implement on Signer**

Add to the `Signer` interface (`signer.go`):
```go
	// RequestEmailVerification asks GateGuard to (re)send a verification code.
	RequestEmailVerification(ctx context.Context, email string) error
	// VerifyEmail submits a code to mark the address verified.
	VerifyEmail(ctx context.Context, email, code string) error
```

Implement on `gatekeeperSigner`:
```go
func (s *gatekeeperSigner) RequestEmailVerification(ctx context.Context, email string) error {
	if s.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
		defer cancel()
	}
	if _, err := s.client.RequestEmailVerification(ctx, &gg.EmailRequest{Email: email}); err != nil {
		return fmt.Errorf("gateguard request verification: %w", err)
	}
	return nil
}

func (s *gatekeeperSigner) VerifyEmail(ctx context.Context, email, code string) error {
	if s.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
		defer cancel()
	}
	if _, err := s.client.VerifyEmail(ctx, &gg.VerifyEmailRequest{Email: email, Token: code}); err != nil {
		return fmt.Errorf("gateguard verify email: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/auth/ -v`
Expected: PASS.

- [ ] **Step 6: Create the HTTP handler (raw mux, like `/auth/me`)**

```go
// backend/internal/http/authverify/handler.go
package authverify

import (
	"encoding/json"
	"net/http"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
)

// Deps carries the signer used to reach GateGuard.
type Deps struct {
	Signer authpkg.Signer
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the verify endpoints (mounted ahead of the swagger mux).
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /auth/request-verification", h.request)
	h.mux.HandleFunc("POST /auth/verify-email", h.verify)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func (h *handler) request(w http.ResponseWriter, r *http.Request) {
	if h.deps.Signer == nil {
		writeErr(w, http.StatusServiceUnavailable, "auth backend not configured")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		writeErr(w, http.StatusBadRequest, "email is required")
		return
	}
	if err := h.deps.Signer.RequestEmailVerification(r.Context(), body.Email); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "could not send verification")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) verify(w http.ResponseWriter, r *http.Request) {
	if h.deps.Signer == nil {
		writeErr(w, http.StatusServiceUnavailable, "auth backend not configured")
		return
	}
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Code == "" {
		writeErr(w, http.StatusBadRequest, "email and code are required")
		return
	}
	if err := h.deps.Signer.VerifyEmail(r.Context(), body.Email, body.Code); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 7: Mount it ahead of the swagger mux**

In `backend/internal/http/module.go`, where custom handlers are built and dispatched (near complaints/follows, ~:290-395):
1. Build it: `authVerifyH := authverify.NewHandler(authverify.Deps{Signer: m.signer})` (use whatever field holds the signer on the module; if none, add one wired from `application.go` where `NewSigner` is constructed).
2. Add a dispatch branch before the swagger fall-through:
   ```go
   if authVerifyH != nil && (p == "/auth/request-verification" || p == "/auth/verify-email") {
       authVerifyH.ServeHTTP(w, r); return
   }
   ```
3. Ensure `/auth/request-verification` and `/auth/verify-email` are treated as **public** (no bearer required) — they take the email in the body. They must be in the unauthenticated path list alongside `/auth/login` etc.

- [ ] **Step 8: Build + test**

Run: `cd backend && go build ./... && go test ./internal/http/... -v`
Expected: success.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/http/auth/ backend/internal/http/authverify/ backend/internal/http/module.go
git commit -m "feat(api): /auth/request-verification + /auth/verify-email proxy endpoints"
```

---

### Task 9: Frontend — verify page, API funcs, verified state, 403 interstitial

**Files:**
- Modify: `frontend/lib/api.ts` (add `requestVerification`, `verifyEmail`, extend `getMe`)
- Modify: `frontend/lib/auth-context.tsx` (`AuthState` + populate `emailVerified`)
- Create: `frontend/app/auth/verify/page.tsx`
- Create: `frontend/components/VerifyEmailInterstitial.tsx`
- Modify call sites that can 403 (event create/edit, RSVP, complaint) to show the interstitial

**Interfaces:**
- Consumes: `POST /auth/request-verification`, `POST /auth/verify-email`, `GET /auth/me` (now returns `email_verified`).
- Produces: `useAuth().emailVerified: boolean`; `EMAIL_NOT_VERIFIED` sentinel thrown by gated calls.

- [ ] **Step 1: Add API functions**

In `frontend/lib/api.ts` add (mirroring `loginWithPassword` at :405 and `authHeaders()` at :576):

```ts
export async function requestVerification(email: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/request-verification`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!res.ok && res.status !== 204) throw new Error(`request verification failed: ${res.status}`);
}

export async function verifyEmail(email: string, code: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/verify-email`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  });
  if (!res.ok && res.status !== 204) throw new Error("Неверный или просроченный код");
}
```

Extend `getMe` (:581) to read `email_verified` from the response and include it in the returned object (`{ id, email, name, role, emailVerified: !!data.email_verified }`). Update its TypeScript return type.

- [ ] **Step 2: Add a helper to detect the 403 sentinel**

In `frontend/lib/api.ts`, add a shared checker used by write calls:
```ts
export const EMAIL_NOT_VERIFIED = "EMAIL_NOT_VERIFIED";
// In each write call (createEvent, updateEvent, rsvp signUp, submitComplaint),
// after `const res = await fetch(...)`, before the generic error throw, add:
//   if (res.status === 403) {
//     const b = await res.clone().json().catch(() => ({}));
//     if (b?.code === "email_not_verified") throw new Error(EMAIL_NOT_VERIFIED);
//   }
```
Apply that 4-line guard to the write functions that call gated endpoints (find them in `lib/api.ts`: the create-event, update-event, RSVP sign-up, and complaint-submit functions).

- [ ] **Step 3: Extend auth state**

In `frontend/lib/auth-context.tsx`, add `emailVerified: boolean` to `AuthState` (default `false`), and set it from `getMe()` wherever `role` is populated (on mount + after login/register).

- [ ] **Step 4: Build the verify page**

```tsx
// frontend/app/auth/verify/page.tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { requestVerification, verifyEmail } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

const inputClass =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export default function VerifyPage() {
  const { email } = useAuth();
  const router = useRouter();
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);

  const addr = email ?? "";

  async function onResend() {
    setError("");
    try {
      await requestVerification(addr);
      setSent(true);
    } catch {
      setError("Не удалось отправить код. Попробуйте позже.");
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (code.length !== 6) return;
    setBusy(true);
    setError("");
    try {
      await verifyEmail(addr, code);
      router.push("/"); // verified; caller flows resume
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка проверки кода");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-md px-4 py-10">
      <h1 className="mb-2 text-[28px] font-bold tracking-[-0.022em]">Подтвердите почту</h1>
      <p className="mb-4 text-[15px] text-label-secondary">
        Мы отправили 6-значный код на {addr || "вашу почту"}.
      </p>
      <form onSubmit={onSubmit}>
        <input
          className={inputClass + " mb-3 text-center text-[22px] tracking-[10px]"}
          inputMode="numeric"
          maxLength={6}
          placeholder="000000"
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
        />
        {error && <p className="mb-3 text-[14px] text-red-500">{error}</p>}
        {sent && <p className="mb-3 text-[14px] text-green-600">Код отправлен.</p>}
        <button
          type="submit"
          disabled={busy || code.length !== 6}
          className="w-full rounded-capsule bg-accent px-4 py-2.5 text-white disabled:opacity-50"
        >
          Подтвердить
        </button>
      </form>
      <button onClick={onResend} className="mt-4 text-[15px] text-accent">
        Отправить код ещё раз
      </button>
    </main>
  );
}
```

- [ ] **Step 5: Build the interstitial component**

```tsx
// frontend/components/VerifyEmailInterstitial.tsx
"use client";

import Link from "next/link";

export function VerifyEmailInterstitial({ onClose }: { onClose?: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="rounded-card bg-bg p-5 max-w-sm">
        <h2 className="mb-2 text-[19px] font-semibold text-label">Подтвердите почту</h2>
        <p className="mb-4 text-[15px] text-label-secondary">
          Чтобы выполнить это действие, подтвердите свою электронную почту.
        </p>
        <div className="flex gap-2">
          <Link href="/auth/verify" className="rounded-capsule bg-accent px-4 py-2 text-white">
            Подтвердить сейчас
          </Link>
          {onClose && (
            <button onClick={onClose} className="rounded-capsule bg-fill px-4 py-2 text-label">
              Позже
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Wire the interstitial at gated call sites**

At each place that calls a gated write (create/edit event submit, RSVP button, complaint submit), catch `EMAIL_NOT_VERIFIED` and render `<VerifyEmailInterstitial />`. Example pattern in the RSVP button handler:
```tsx
try {
  await rsvpSignUp(eventId, answer);
} catch (e) {
  if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) { setShowVerify(true); return; }
  throw e;
}
```

- [ ] **Step 7: Build the frontend**

Run: `cd frontend && npm run build` (or `pnpm build` / `yarn build` — match the repo's lockfile).
Expected: type-checks + builds clean.

- [ ] **Step 8: Commit**

```bash
git add frontend/lib/api.ts frontend/lib/auth-context.tsx frontend/app/auth/verify/ frontend/components/VerifyEmailInterstitial.tsx
git commit -m "feat(frontend): email verify page + 403 interstitial + verified state"
```

---

## Self-Review

**Spec coverage (design §5 Phase B, §7 Phase D):**
- Proto `email_verified` + regen + mapping → Task 1. ✅
- Claims → domain.User propagation + mock=true → Task 2. ✅
- Swagger principal field → Task 3. ✅
- `RequireVerified` 403 helper → Task 4. ✅
- Gate create/update event + RSVP → Task 5. ✅
- Gate complaints → Task 6. ✅
- Defensive GateGuard invite-accept check → Task 7. ✅
- Lia verify proxy endpoints → Task 8. ✅
- Frontend verify page + interstitial + verified state → Task 9. ✅
- Follows intentionally NOT gated (confirmed with user). ✅

**Placeholder scan:** Code blocks are concrete. "Match the existing helper/fake" notes (Tasks 2,5,6,7,8 tests) are explicit instructions to reuse in-file scaffolding whose exact identifiers must be read from the test file — not placeholders for logic. The `*bool` vs `bool` note in Tasks 4/5 is a real regen-dependent branch with both variants given.

**Type consistency:** `IsVerified(*apimodels.User) bool` + `UnverifiedResponder() middleware.Responder` (Tasks 4,5); `domain.User.EmailVerified` (Tasks 2,6,7); `Signer.RequestEmailVerification/VerifyEmail` + `gg.EmailRequest`/`gg.VerifyEmailRequest{...Token}` (Task 8); `EMAIL_NOT_VERIFIED` sentinel + `emailVerified` state (Task 9). Consistent throughout.

**Dependency note:** Phase B assumes Phase A (Plan 1) is merged (real codes are actually sent), but Phase B is independently buildable/testable — gating and propagation don't require emails to actually send. Deploy A and B together so users have a working way to verify before gates go live.
