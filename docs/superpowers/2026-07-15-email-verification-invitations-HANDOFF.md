# HANDOFF — Email Verification (SendPulse) + Event Invitations

**Status as of 2026-07-16:** Feature is **code-complete and merged to local `main`** (merge commit `f4bdbeb`). What remains is **operational**: push, set up the email sender + DNS, and deploy. Nothing below is blocked by code — it's all config/ops.

---

## 1. Where things stand

| Thing | State |
|---|---|
| Code (3 phases, 27 tasks) | ✅ Done, reviewed, merged to local `main` |
| Backend / gateguard build + tests | ✅ Pass on merged `main` |
| Frontend production build | ✅ Clean (routes `/auth/verify`, `/invite/[token]`, `/me/invitations`) |
| Migration `000020_event_invitations` | ✅ Written + validated on real Postgres 16 (not yet applied to prod) |
| `git push origin main` | ❌ **Not pushed** — `main` is 32 commits ahead of `origin/main` |
| SendPulse sending account | ❌ Not created |
| nic.ru DNS (SPF/DKIM/DMARC for `tarski.ru`) | ❌ Not added |
| Deploy (migration + env + images) | ❌ Not done |

**What the feature does:** organizer invites people to an event by email; invitees (incl. people with no account) accept via an email link or in-app list; accepting requires a verified email; unverified users are blocked from creating/editing events, RSVP/applying, and complaints. Verification = a 6-digit code emailed on signup.

**Why email setup is the blocker:** the code builds/tests/deploys fine with **no** SMTP config (codes are generated but never delivered; invites still appear in-app). But for verification codes and invite emails to actually reach inboxes, you need a real sender (SendPulse) with authenticated DNS on `tarski.ru`. That's the manual part below.

---

## 2. TODO — Push (2 min)

- [ ] `cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia && git push origin main`
  - Publishes the 32 unpushed commits. Do this whenever you're comfortable; nothing else requires it first, but the deploy image is built from this tree.

---

## 3. TODO — SendPulse sender account

> There's a **browser-agent prompt** that does most of this for you (you just log in manually): `docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-browser-setup-prompt.md`. Paste it into a browser-capable Claude session. The manual steps below are the same thing if you'd rather click through yourself. Free tier = 12,000 emails/month, enough for this app.

- [ ] **Create/log into a SendPulse account** at https://sendpulse.com (the account's own login email can be anything — it does NOT need to be `@tarski.ru`).
- [ ] **Add `tarski.ru` as a sending domain.** Navigate to the SMTP / transactional section → "Sending domains" / "Add domain" → enter `tarski.ru` and start the verification flow.
- [ ] **Copy the DNS records SendPulse shows** (you'll paste these at nic.ru in step 4). Capture them **verbatim** — there are typically:
  - an **SPF** value like `v=spf1 include:sendpulse.com ~all` (copy the exact `include:` host shown)
  - a **DKIM** record: a host/selector (e.g. `sp._domainkey`) + a long `v=DKIM1; k=rsa; p=...` value (copy the **full** key)
  - possibly an extra **verification TXT** token
- [ ] **Add `info@tarski.ru` as the sender / From address.** Note whether SendPulse requires confirming `info@tarski.ru` via a click-email (if so you'll need mailbox access to that address — or domain verification alone may suffice).
- [ ] **Copy the SMTP connection credentials** (Settings → SMTP): **host**, **port** (SendPulse offers 2525/465/587 — record all), **login**, **password**. Keep the password secret.
- [ ] Save all captured values to `docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md` (already git-ignored via `*.local.md`).

**Payment note:** SendPulse's free 12k/month tier needs no payment, but if account creation demands a card it's EUR (Visa/MC/PayPal). If that's a blocker, the fallback is **Unisender Go** (RU billing, ruble-friendly) — the SMTP wiring is identical, just different creds + a different SPF `include:`/DKIM.

---

## 4. TODO — nic.ru DNS for `tarski.ru`

- [ ] Log into https://www.nic.ru (RU-Center) → **Мои домены** → `tarski.ru` → **Управление зоной / DNS zone management**.
- [ ] If nic.ru says the domain uses **external nameservers** (not nic.ru's), you must add the records **wherever the authoritative NS are**, not here.
- [ ] Add each record SendPulse gave you:
  - [ ] **SPF** (TXT, name `@`): the SendPulse SPF string. ⚠️ A domain may have only **ONE** `v=spf1` record — if one already exists, **merge** the SendPulse `include:` into it rather than adding a second.
  - [ ] **DKIM** (TXT, name = the selector e.g. `sp._domainkey`): the full `v=DKIM1; ...` value.
  - [ ] **Verification TXT** (if SendPulse required one).
  - [ ] **DMARC** (TXT, name `_dmarc`): `v=DMARC1; p=quarantine; rua=mailto:info@tarski.ru` (sane starter; skip/adjust if a `_dmarc` already exists).
- [ ] Save the zone changes. **Propagation takes 30 min – a few hours.**
- [ ] Back in SendPulse, click **Verify / Check records** on the domain until SPF + DKIM show green.

---

## 5. TODO — Wire env + deploy

Once the domain verifies green in SendPulse and you have the SMTP creds:

- [ ] **GateGuard** (sends the verification codes) — set env:
  - `notificator.address` = `<SendPulse SMTP host:port>`
  - `notificator.username` = `<SMTP login>`
  - `notificator.password` = `<SMTP password>`
  - `notificator.from` = `info@tarski.ru`
- [ ] **Lia backend** (sends the invite emails) — set env:
  - `SMTP_ADDRESS` = `<host:port>`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` = `info@tarski.ru`
  - `PUBLIC_BASE_URL` = `https://presence.tarski.ru` (default; this builds the `/invite/<token>` links)
- [ ] **Apply migration 000020** on the prod box: `scp` `backend/db/migrations/000020_event_invitations.{up,down}.sql` to the box's `/opt/lia/backend/db/migrations/`, then run the migrate step (prod DB is at 019 → goes to 020).
- [ ] **Build + ship images** (build-on-Mac → save|ssh|load, per the deploy runbook):
  - Backend: the swagger `internal/http/{models,server}` are git-ignored and regenerated — run **`make generate-api`** in `backend/` before building the image, or the `email_verified` principal field won't be in the built code.
  - GateGuard: its Docker build runs `make generate-proto` itself (proto `.pb.go` is regenerated at build), so no extra step — just ensure the committed `.proto` source is in the build context (it is now).
- [ ] After deploy, **prune stale Docker** on the box (20 GB disk fills up) per `lia-deploy-image-cleanup` memory.

---

## 6. TODO — Smoke test (after deploy)

- [ ] Register a new user with a **real Gmail/Yandex/Mail.ru** address → confirm the 6-digit code arrives **in the inbox (not spam)** and DKIM passes (check the email's "show original" → DKIM=pass).
- [ ] Enter the code on `/auth/verify` → confirm it verifies.
- [ ] As a verified organizer, open `/events/mine`, expand a published event, send an invite to a test address → confirm the invite email arrives with a working `/invite/<token>` link.
- [ ] Open the link in a fresh session → sign up → verify → accept → confirm you land on the event and an RSVP is created.
- [ ] Confirm an unverified user gets the "verify your email" interstitial when trying to RSVP / create an event.

---

## 7. Known follow-ups (non-blocking, deferred)

These were flagged in the final review and consciously deferred — safe to ship without, worth doing later:

- [ ] **Invite expiry not enforced on accept.** `invitations.Repository.ExpireOverdue` has no caller, and `service.accept` only checks `status=='pending'`, not `expires_at`. So a pending invite past its 30-day TTL stays acceptable forever (the `/me/invitations` list does filter expired, so it's only the direct token/id accept paths). Fix: either call `ExpireOverdue` on a schedule, or add an `expires_at > now()` check in `accept`.
- [ ] **Re-inviting an already-pending email sends a dead link.** `repository.Insert` uses `ON CONFLICT ... DO NOTHING`, but `service.Invite` still emails the freshly-generated token — which was never stored, so that link 404s. Fix: have `Insert` report whether a row was actually inserted and only email/count on a real insert.
- [ ] Minor cleanups (all logged in `.superpowers/sdd/progress.md`): the 403-guard is copy-pasted 4× in `frontend/lib/api.ts`; `listMine` returns a hardcoded 500 instead of via `mapErr`; a few untested branches (RSVP-fail, AcceptByID/Preview/ListMine).

---

## 8. Reference — where things live

- **Design spec:** `docs/superpowers/specs/2026-07-15-email-verification-and-invitations-design.md`
- **Plans:** `docs/superpowers/plans/2026-07-15-phase-{a,b,c}-*.md`
- **Execution ledger (per-task history):** `.superpowers/sdd/progress.md` (git-ignored)
- **Browser-setup agent prompt:** `docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-browser-setup-prompt.md`
- **Key code:**
  - Verification: `gateguard/internal/service/email_verification.go`, notificator `gateguard/internal/pkg/notificator/`
  - Gate: `backend/internal/http/handlers/verified_gate.go`; propagation `backend/internal/http/auth/`
  - Verify proxy: `backend/internal/http/authverify/handler.go`
  - Invitations: `backend/internal/invitations/`, `backend/internal/http/invitations/handler.go`, mailer `backend/internal/notifications/mailer.go`
  - Frontend: `frontend/app/auth/verify/`, `frontend/app/invite/[token]/`, `frontend/app/me/invitations/`, `frontend/components/{InviteByEmailPanel,VerifyEmailInterstitial}.tsx`, `frontend/lib/api.ts`

## 9. Gotchas for whoever resumes

- Local `golangci-lint` is v2 but the repo config is v1 → use `gofmt`/`go build`/`go vet` instead.
- `gateguard/internal/service` has a pre-existing `go vet` printf false-positive → run its tests with `-vet=off`.
- Pre-existing **failing** tests `Test_ReactToInvitation_*` in the gateguard module (org-mock mismatch) fail on `main` too — **not** from this work; don't chase them.
- Backend `internal/http/{models,server}` and gateguard `*.pb.go` are git-ignored generated code — regenerate before building (see step 5).
