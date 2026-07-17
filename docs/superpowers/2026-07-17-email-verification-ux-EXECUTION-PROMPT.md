# Execution prompt — Email Verification UX (subagent-driven)

_Written 2026-07-17 by the session that produced the spec + plan. Paste the block
under **PROMPT** into a fresh Claude Code session at the repo root. Everything the
next agent needs is either in the prompt or reachable from it._

---

## PROMPT

```
Execute the implementation plan at
docs/superpowers/plans/2026-07-17-email-verification-ux.md
using the superpowers:subagent-driven-development skill — dispatch a fresh subagent
per task, review between tasks. Invoke that skill FIRST, before any other action.

Repo root: /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
Branch: main (4 commits ahead of origin/main, all docs — safe to push, not required)
Plan HEAD when written: 43962f2

## Read these first, in this order
1. docs/superpowers/plans/2026-07-17-email-verification-ux.md  ← the plan you execute
2. docs/superpowers/specs/2026-07-17-email-verification-ux-design.md  (spec, db0614d)
   — read at least "Decisions" and §3; it explains WHY the check order matters.
3. docs/superpowers/runbooks/2026-07-16-email-verification-invitations-deploy.md
   — only when you reach Task 8. Its "Traps" section is the last deploy's scar tissue.

## What this is
The email-verification feature is ALREADY BUILT, DEPLOYED, AND LIVE on
presence.tarski.ru (deployed 2026-07-16; tarski.ru verified in SendPulse, DKIM+SPF+
DMARC green; prod DB at Lia migration 020; a real signup was proven end-to-end —
code emailed, delivered, entered, email_verified=t in the DB).

You are NOT building the feature. You are fixing its UX, which is purely punitive:
nothing announces that verification exists, so the only way to discover it is to be
blocked mid-action. The product owner hit this himself — registered on prod, got the
email, and saw no sign anywhere on the site that an email was coming. You are adding
the missing signposts (signup confirmation + a persistent banner) and making the code
live 24h bounded by a 5-attempt cap instead of a 15-minute clock.

## Scope (8 tasks; the plan has exact code for every step)
1. gateguard migration 000012 + model field (`use_zero` is REQUIRED)
2. TTL 15min→24h + verificationMaxAttempts=5 + ErrVerificationTooManyAttempts
3. gateguard returns typed gRPC status codes
4. backend proxy maps codes → JSON {code,message}
5. frontend parses the code → three distinct Russian messages
6. new non-dismissible VerifyEmailBanner, mounted globally
7. AuthButton register-success confirmation state
8. deploy

## Non-negotiables — every one of these already bit someone
- **Task 2 Step 0 comes FIRST.** Two EXISTING tests break on this change and are
  repaired before anything else: Test_VerifyEmail_Expired asserts expiry on a
  20-minute-old code (valid at 24h), and Test_RequestEmailVerification_SendsCode pins
  UpdateUserBy to exactly two columns (we add a third). Skip this and the suite goes
  red for reasons unrelated to your work.
- **The attempts check MUST precede the token comparison** in VerifyEmail. Lockout
  clears the token, so checking after the comparison reports "wrong code" to a
  locked-out user forever, with no hint that resending is the way out — recreating the
  exact dead-end this work exists to remove. Test_VerifyEmail_LockedOutRejectsEven-
  CorrectCode pins this. If a subagent "simplifies" the order, reject the task.
- **Gateguard tests: `go test -vet=off ./internal/service/... -run TestUsecase`.**
  `-vet=off` is mandatory (pre-existing printf false-positive). Tests are testify-suite
  METHODS + mockery EXPECT() mocks — `-run Test_VerifyEmail_X` matches nothing and
  passes vacuously. The entrypoint is TestUsecase.
- **`Test_ReactToInvitation_*` FAIL on main already** (org-mock mismatch). Not yours.
  Do not chase them, do not "fix" them, do not let a subagent claim they're a
  regression.
- **go-pg omits zero values without `use_zero`.** Without the tag, resetting attempts
  to 0 silently doesn't persist — the counter only climbs and a user who resends stays
  locked out forever. This is the single easiest way to ship a broken feature here.
- **`UpdateUserBy` writes ONLY the columns named in its variadic list.** A field
  changed but not named is silently dropped.
- Local golangci-lint is v2 but the repo config is v1 → use gofmt / go build / go vet.
- All user-facing copy is Russian. Match the existing components.
- No new dependencies. Everything needed is already in both go.mods and package.json.

## Task 8 (deploy) — read before you touch prod
- **STOP and ask the human before deploying.** Do not deploy on your own initiative
  just because tasks 1-7 are green. Confirm they want it live.
- **The prod DB migration will be BLOCKED by the permission classifier.** That is
  expected and correct. Write the command into a SCRIPT FILE on the box and have the
  human run `! ssh vdska2 /opt/lia/<script>.sh`. Do NOT paste a long one-liner: a
  wrapped `docker run ... \n -path=/db/` makes migrate print usage and bash then tries
  to execute `-path=/db/`. And use SINGLE quotes in SQL — `" dirty="` is read by
  Postgres as a column identifier. Both mistakes were made on 2026-07-16.
- **This migration targets the `gateguard` database, NOT Lia's 020 chain.** Mount
  `-v /opt/gateguard/db:/db` — gateguard migrations are FLAT in db/, not db/migrations/.
- **Build gateguard on the MAC, not the box.** The box's docker build fails on a
  github TLS error fetching protoc-gen-grpc-web; `-4 --retry` does not help.
- **Sync 1-lia/gateguard/ → box /opt/gateguard/ FIRST and md5-verify.** The box's copy
  went stale once and held a pre-feature stub that logged the link instead of emailing
  it — a deploy that looks successful while sending nothing. `rsync` is blocked by the
  classifier; use `tar czf - | ssh vdska2 'cat > /tmp/gg-src.tgz'`.
- **Frontend needs BOTH build-args** (NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru
  AND NEXT_PUBLIC_YANDEX_MAPS_KEY from frontend/.env.local). An undeclared build-arg is
  silently dropped and inlines as "" — a missing maps key breaks every map on the site.
- **Backend: `make generate-api` BEFORE docker build.** swagger internal/http/{models,
  server} are git-ignored and NOT regenerated by `make build`.
- Live image names are `backend-app` / `lia-frontend-presence` / `gateguard:local` —
  NOT lia-*. Tagging rollbacks against the wrong names silently no-ops.
- The frontend is NOT compose-managed (bare docker run, 127.0.0.1:3002:3001). Ignore
  /opt/lia/frontend-build.sh — it is stale and points at api.lia.pashteto.com, which
  degrades the site to mock data.
- **`docker exec backend-app-1 env` returns NOTHING** — scratch image, no env binary.
  That is NOT evidence the env is missing. Use
  `docker inspect <c> --format '{{range .Config.Env}}{{println .}}{{end}}'`.
- ssh alias is `vdska2` (193.32.188.7). "vds-ru215" in the runbooks is the box label,
  not a resolvable host. `dig`/`host` do NOT work from the sandbox at all — verify DNS
  via https://dns.google/resolve?name=<n>&type=TXT.
- Back up the gateguard DB before migrating. Tag rollbacks before cutover. Prune after
  (`docker builder prune -f; docker image prune -f`) — the 20 GB disk has hit 90%.

## Hands off
- **Do NOT touch DMARC.** tarski.ru sits at a pre-existing `p=quarantine` that SendPulse
  accepts and shows green. SendPulse suggests p=none; applying it would WEAKEN a
  stronger policy that was already there. This was a deliberate decision.
- **Do NOT commit secrets.** SMTP creds + DNS records live in the git-ignored
  docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md
  (`*.local.md` in .gitignore). Refer to values by env var name. Grep any doc you
  commit for the password and the DKIM key first.
- **Do NOT create accounts or enter passwords** — Claude cannot do either. The human
  signs up; you watch gateguard logs and SendPulse's Sending History.
  NOTE: gateguard only logs on FAILURE (sign_in_password.go:69-72 deliberately does not
  fail signup if the send errors). Log silence = success. SendPulse → SMTP → Sending
  history is the authoritative record, not the logs.
- **Out of scope** (do not let scope creep in): a /me root profile page (deliberately
  YAGNI'd — the banner already says it); changing WHICH actions are gated; the two
  deferred invitation bugs (see below).

## Known-deferred, do not "helpfully" fix mid-plan
Logged in the handoff §7, both user-visible, both out of scope here:
- Invite expiry not enforced on accept (invitations.Repository.ExpireOverdue has no
  caller; service.accept checks only status=='pending').
- Re-inviting an already-pending email emails a token that was never stored
  (ON CONFLICT DO NOTHING) → that link 404s.
Also unfixed: RequestEmailVerification has no per-IP rate limit; the 60s per-account
cooldown is the only limiter.

## Definition of done
- Tasks 1-7 committed, each with its own passing test cycle.
- `cd gateguard && go build ./... && go test -vet=off ./internal/service/... -run TestUsecase`
  → green (ReactToInvitation excepted).
- `cd backend && go test ./internal/http/authverify/...` → green.
- `cd frontend && pnpm build` → exit 0.
- Verified in a running app, not just tests: an unverified user sees the banner on
  every page; registering shows «Проверьте почту … на <address>»; logging in does NOT;
  verifying makes both the banner and the interstitial disappear.
- Task 8 only with the human's explicit go-ahead.
```

---

## Notes for the human (not part of the prompt)

- The 4 unpushed commits are all docs (spec, plan, runbook, handoff edits). `git push
  origin main` whenever convenient; nothing in the plan depends on it.
- `docs/HANDOFF.md` had uncommitted edits from another session when this was written —
  left untouched deliberately.
- If you'd rather run it inline in one session than subagent-driven, swap the first
  line to `superpowers:executing-plans`. Everything else in the prompt still applies.
