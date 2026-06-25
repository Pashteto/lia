# HANDOFF — GateGuard `SignInOAuth` panic blocks demo-login (auth Phase C)

_2026-06-25. For the next agent. Context: auth slice spec/plan =
[`../specs/2026-06-25-auth-gatekeeper-design.md`](../specs/2026-06-25-auth-gatekeeper-design.md) /
[`../plans/2026-06-25-auth-gatekeeper.md`](../plans/2026-06-25-auth-gatekeeper.md).
All auth work is on branch `feat/dark-mode-theme` (unmerged)._

## TL;DR — current LIVE state (⚠️ demo partially broken)

- On vds-ru215 (`ssh vdska2`), Lia backend runs with **`HTTP_MOCK_AUTH=false`** →
  **real auth is enforced**. Verified: `POST /events` without a valid token → **401**.
  Discovery GETs are open → browsing works.
- **`POST /auth/demo-login` → 503** because GateGuard's `SignInOAuth` **panics**.
  So there's no way to obtain a token → **"create event" on the live demo is broken.**
- **One-line revert to restore the demo** (do this if the demo must work now):
  ```bash
  ssh vdska2 'cd /opt/lia/backend && sed -i "s/^HTTP_MOCK_AUTH=.*/HTTP_MOCK_AUTH=true/" .env.prod && \
    docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d app'
  ```
  (Auth code + GateGuard stay deployed; this just bypasses validation again.)

## The bug

GateGuard (`backend-gateguard-1`) on a `SignInOAuth` call panics; its recovery
interceptor returns gRPC `Internal`. Logs:
- Lia app: `demo-login: gateguard signin: rpc error: code = Internal desc = runtime error: index out of range [2] with length 2`
- GateGuard: `SignInOAuth handler called {request_email: demo@lia.test}` then the same Internal error.

### Reproduce
```bash
curl -s -X POST https://api.lia.pashteto.com/api/v1/auth/demo-login \
  -H 'Content-Type: application/json' -d '{"email":"x@y.z","name":"X"}'
# → {"code":503,"message":"auth backend error"}; GateGuard log shows the panic
```

## Already traced / ruled out (don't redo)

The panic is **NOT** in any of these (all read, none index `[2]`):
- `internal/service/sign_in.go` `SignInOAuth` + `getOrCreateUser`
- `internal/service/user.go` `createJWT`; `internal/models/user.go` `ToJWT` (trivial), `UserFromProto` (uses `uuid.FromBytesOrNil`, safe)
- `github.com/andskur/gatekeeper` `jwt/jwt.go` `Create` + `formatStorageKey` (just `Sprintf`)

GateGuard's recovery interceptor does **not** log `debug.Stack()`, so the file:line
is unknown. **Getting the stack trace is the key next step.**

## Recommended next steps (in order)

1. **Get the stack trace.** Either:
   - Add `debug.Stack()` logging to GateGuard's gRPC recovery interceptor
     (`/opt/gateguard` or `~/gateway_fm/appstore/gateguard`), rebuild, repro; OR
   - Run GateGuard locally (`docker-compose.yaml` in its repo: postgres+redis+migrate+service)
     and call `SignInOAuth` via grpcurl with `{email}` — the panic prints a full trace.
2. **Cheap experiment first (might fix without the trace):** the panic may be a
   missing field on a minimal user. In Lia's signer try populating the proto user
   before the call — `backend/internal/http/auth/signer.go`, in `SignIn`:
   set `Status: gg.UserStatus_UserActive`, `Role: gg.UserRole_UserRoleCommon`,
   and a fresh `Uuid: uuid.NewV4().Bytes()`. Redeploy backend, repro. If the panic
   moves/clears, that localizes it.
3. Suspects to check once the trace points somewhere: the **redis** storage path
   (`go-redis` v6, `REDIS_ADDRESS` URL parsing), a **sentry/logging gRPC interceptor**,
   or `repository.CreateUser` (go-pg) building columns.

## What is DONE and correct (don't rebuild)

- **Phase A** (Lia backend auth): `internal/http/auth/{auth,gatekeeper,signer}.go` —
  `CheckAuth` validates via GateGuard gRPC, JIT-provisions a local user, `POST /events`
  requires jwt (organizer = principal). TDD'd. **Deployed + enforcing (401 verified).**
- **Phase B** (GateGuard deployed): `backend-gateguard-1` + `-redis-1`, gRPC
  `gateguard:9090`, reuses Lia's Postgres (`gateguard` DB). Runbook
  [`2026-06-25-gateguard-deploy.md`](2026-06-25-gateguard-deploy.md).
- **Phase C backend** (Lia side): `POST /auth/demo-login` handler + `auth.Signer`
  are **correct** — they just call GateGuard `SignInOAuth`. The fault is GateGuard's,
  not Lia's. Lia returns 503 on the GateGuard error (expected).

## Remaining once the panic is fixed

1. Verify the full round-trip: `demo-login` → token → `POST /events` with
   `Authorization: Bearer <token>` → **201** (proves CheckAuth too).
2. **Phase C frontend** (not started): "Войти" form (email) → call `/auth/demo-login`
   → store token → attach `Bearer` to `createEvent` in `frontend/lib/api.ts` → gate
   `/events/new` → sign-out. Deploy frontend.
3. Update HANDOFF + deploy memory; document the access-control change (ISO/Vanta).

## Notes
- GateGuard `SignInOAuth` is DEMO-ONLY (mints a token for any email). Keep it gated;
  never enable in real prod (known non-prod control, like the old `HTTP_MOCK_AUTH`).
- A leftover draft test event `x` (`id 2ae9911a-…`) exists from an earlier test;
  harmless (draft, not in published feed). Delete needs explicit approval (prod DB).
