# HANDOFF — Email Verification (SendPulse) + Event Invitations

**Status as of 2026-07-16 (revised):** Feature is **code-complete and merged**, and `f4bdbeb`
is now **on `origin/main`**. The SendPulse account, the SMTP credentials, and the
`info@tarski.ru` sender all **already exist** — see
`docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md` (git-ignored,
holds the secrets + the exact env var names verified against the code).

**What actually remains:** (1) add the SPF/DKIM/DMARC records at nic.ru and confirm the
domain in SendPulse — deliverability only, not a blocker; (2) set the env vars; (3) apply
migration 020 + deploy. Nothing is blocked by code.

---

## 1. Where things stand

| Thing | State |
|---|---|
| Code (3 phases, 27 tasks) | ✅ Done, reviewed, merged to local `main` |
| Backend / gateguard build + tests | ✅ Pass on merged `main` |
| Frontend production build | ✅ Clean (routes `/auth/verify`, `/invite/[token]`, `/me/invitations`) |
| Migration `000020_event_invitations` | ✅ Written + validated on real Postgres 16 (not yet applied to prod) |
| `git push origin main` | ✅ **Done** — merge `f4bdbeb` is on `origin/main` (verified 2026-07-16) |
| SendPulse sending account | ✅ **Exists** — Free SMTP plan, 12k/mo, active until 2026-08-15 |
| SendPulse sender `info@tarski.ru` | ✅ **Active** — no click-confirmation outstanding |
| SendPulse SMTP credentials | ✅ **Captured** → `runbooks/2026-07-15-sendpulse-nicru-credentials.local.md` |
| SendPulse authenticated domain `tarski.ru` | 🟡 Added 2026-07-16 — "Awaiting confirmation" until DKIM propagates |
| nic.ru DNS — SPF | ✅ **Live + propagated 2026-07-16**, SendPulse checker green |
| nic.ru DNS — DKIM (`sign._domainkey`) | 🟡 **Added + published**, propagating (NXDOMAIN as of ~18:55) |
| nic.ru DNS — DMARC | ✅ Pre-existing `p=quarantine` — **deliberately left alone**, see §4 |
| Deploy (migration + env + images) | ❌ **Not done — now the only real work left** |

> **Status corrected 2026-07-16.** The three ❌ rows above (push / account / sender) were
> stale — all were already done. **Email can be sent today**: `info@tarski.ru` is an active
> sender and the SMTP creds work, signed by SendPulse's own domain. The remaining DNS work
> is a *deliverability* upgrade (SPF/DKIM aligned to `tarski.ru`, inbox vs. spam), **not a
> hard blocker**. You can wire env + deploy now and do DNS as a follow-up.

**What the feature does:** organizer invites people to an event by email; invitees (incl. people with no account) accept via an email link or in-app list; accepting requires a verified email; unverified users are blocked from creating/editing events, RSVP/applying, and complaints. Verification = a 6-digit code emailed on signup.

**Why email setup was thought to be the blocker:** the code builds/tests/deploys fine with **no** SMTP config (codes are generated but never delivered; invites still appear in-app). Reaching real inboxes needs a real sender — which now exists. See the revised framing above: sending works today; DNS is the inbox-vs-spam upgrade.

---

## 2. ~~TODO — Push~~ ✅ DONE

Merge `f4bdbeb` is on `origin/main` (verified 2026-07-16). Nothing to do.

---

## 3. ~~TODO — SendPulse sender account~~ ✅ DONE (2026-07-16)

All of this is complete. Captured values → `docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md` (git-ignored).

- [x] SendPulse account exists — Pavel D. / `dodonovpavel@gmail.com` / ID 9460079. **Free SMTP plan, 12,000 emails/mo, active until 2026-08-15.** No payment was needed, so the Unisender Go fallback is moot.
- [x] `tarski.ru` added as an authenticated domain (SMTP → Settings → Domain settings → Activate). Status: **"Awaiting confirmation"** until the DNS lands. One authenticated domain is allowed on the free plan; the paid upsell is only for *unlimited* domains.
- [x] DNS records captured verbatim — see §4 below for the actual values.
- [x] `info@tarski.ru` is an **Active** sender already. **No click-confirmation email is outstanding**, so mailbox access to that address is NOT needed.
- [x] SMTP credentials captured: host `smtp-pulse.com`, ports 2525 / 465 SSL / 587 TLS, login `dodonovpavel@gmail.com`. Password is in the `.local.md`.

**Free-tier limit worth knowing:** **50 emails/hour.** A burst of invitations larger than that will throttle. Not a problem at current scale, but it's the first ceiling you'll hit.

---

## 4. ~~TODO — nic.ru DNS~~ ✅ **ENTERED + PUBLISHED 2026-07-16 ~18:53** (DKIM propagating)

**Done.** SPF replaced, SendPulse DKIM added, both published. Verified by DoH lookup:
**SPF is live and SendPulse's checker shows it green**; **DKIM is still NXDOMAIN, i.e.
propagating** (new name; up to 24h, usually far less). Domain stays "Awaiting confirmation"
in SendPulse until DKIM lands. Exact before/after + how to finish → the `.local.md`.

> **The zone already had a complete nicmail (nic.ru mail) email stack** — SPF, DKIM
> (`nicmail20230706._domainkey`), DMARC and an `mx01.nicmail.ru` MX, all created
> 2026-07-16 ~00:36, hours before this work and unknown to every planning doc. That
> changed two decisions:
>
> - **DMARC was left alone at `p=quarantine`.** SendPulse suggests `p=none`; applying it
>   would have *weakened* a stronger policy that already existed. Kept `quarantine`.
>   Accepted trade-off: SendPulse mail sent before the DKIM propagates may be quarantined
>   (it can't DMARC-align yet). Transient.
> - **The two DKIM keys coexist** on different selectors (`nicmail20230706` vs `sign`), so
>   nic.ru mail and SendPulse each sign with their own. Nothing was displaced.
>
> The SPF *replaced* the existing record with a strict superset (both `nicmail.ru`
> includes preserved), so nic.ru mail keeps working. Rollback value is in the `.local.md`.

**Gotchas learned here:**
- **nic.ru DNS Master stages edits.** Changes sit behind a "Зона содержит неопубликованные
  изменения" banner and are NOT live until you click **"Опубликовать"**.
- **`dig`/`host` do not work from the Claude sandbox** (no resolver reachable). Verify with
  DNS-over-HTTPS instead: `https://dns.google/resolve?name=<name>&type=TXT`.
- **In SendPulse, reopen the records via "..." → "Show settings"**, NOT the "Activate"
  button — Activate reopens the add-domain dialog and may regenerate the DKIM key.
- The DKIM key was pasted straight from SendPulse's clipboard button, so the
  transcription risk flagged earlier never materialized.

<details>
<summary>Original instructions (superseded — kept for reference)</summary>

The three records SendPulse generated (exact values, captured 2026-07-16 — also in the
`.local.md`):

| Type | Name | Value |
|---|---|---|
| TXT | `sign._domainkey.tarski.ru` | `v=DKIM1; k=rsa; p=…` (long — **see below**) |
| TXT | `tarski.ru` (zone root / `@`) | `v=spf1 include:mxsspf.sendpulse.com include:dc1.nicmail.ru include:dc2.nicmail.ru ?all` |
| TXT | `_dmarc.tarski.ru` | `v=DMARC1; p=none;` |

Three things changed vs. what this doc originally assumed:

- **The SPF is already merged, and it REPLACES the existing record.** SendPulse detected the
  current `tarski.ru` SPF and folded the `dc1/dc2.nicmail.ru` includes into its generated
  value. So do **not** add a second `v=spf1` — overwrite the existing one with the value
  above. Capture the before-state first.
- **SPF qualifier is `?all` (neutral)**, not the `~all` (softfail) guessed above. `?all`
  asserts nothing, so it buys little anti-spoofing. Consider tightening to `~all` once
  delivery is confirmed.
- **DMARC: SendPulse suggests `p=none;`**, not the `p=quarantine` guessed above. Prefer
  `p=none` to start — monitor-only, nothing gets quarantined while SPF/DKIM settle. Add
  `rua=mailto:info@tarski.ru` if you want aggregate reports. Tighten later.
- **No verification TXT token is required.**

> ⚠️ **DKIM key — do NOT copy from any doc.** The full key is recorded in the `.local.md`,
> but it was transcribed **off a screenshot** (SendPulse's copy button and the DOM reader
> were both unusable under automation). One wrong base64 char makes DKIM fail *silently*.
> **Re-copy it from the SendPulse dialog's own "Copy to clipboard" button** and paste that.
> Dialog: SMTP → Settings → Domain settings → Activate.

- [ ] Log in → **Мои домены** → `tarski.ru` → **Управление зоной**.
- [ ] If nic.ru shows the domain on **external nameservers**, STOP — records must go wherever
      the authoritative NS actually are. (The `dc1/dc2.nicmail.ru` includes in the existing
      SPF suggest nic.ru mail is in play, but that does not by itself prove nic.ru holds the zone.)
- [ ] Add/replace the three records above. Save the zone.
- [ ] Propagation: 30 min – a few hours (SendPulse says DNS updates can take up to 24h).
- [ ] Back in SendPulse → same Activate dialog → **"Check DNS records"** until SPF + DKIM go green.
      Check once and wait; don't poll aggressively.

</details>

---

## 5. TODO — Wire env + deploy

**You do NOT have to wait for the domain to verify.** The creds work now; DNS only improves
inbox placement. The real values (password in the `.local.md`) — port 587/TLS chosen, with
2525 as the usual fallback if the prod network blocks it:

```
# Lia backend                       # GateGuard
SMTP_ADDRESS=smtp-pulse.com:587     NOTIFICATOR_ADDRESS=smtp-pulse.com:587
SMTP_USERNAME=dodonovpavel@gmail.com  NOTIFICATOR_USERNAME=dodonovpavel@gmail.com
SMTP_PASSWORD=<see .local.md>       NOTIFICATOR_PASSWORD=<see .local.md>
SMTP_FROM=info@tarski.ru            NOTIFICATOR_FROM=info@tarski.ru
PUBLIC_BASE_URL=https://presence.tarski.ru
```

> ⚠️ **Verify GateGuard's config actually landed.** Its env vars resolve via
> `viper.AutomaticEnv()` + Unmarshal, and viper is known to not always feed `Unmarshal()`
> from AutomaticEnv. Every `notificator.*` key here has a `SetDefault`, so it *should*
> resolve — but this was traced by **reading code, not proven at runtime**
> (`gateguard/cmd/root/root.go:52` does the Unmarshal). Since a miss falls back to
> Gateway.fm defaults **silently**, treat "a verification email actually arrived" as the
> only real confirmation.

Original steps:

- [ ] **GateGuard** (sends the verification codes) — set env:
  - `NOTIFICATOR_ADDRESS` = `<SendPulse SMTP host:port>`
  - `NOTIFICATOR_USERNAME` = `<SMTP login>`
  - `NOTIFICATOR_PASSWORD` = `<SMTP password>`
  - `NOTIFICATOR_FROM` = `info@tarski.ru`
  - ⚠️ **Corrected 2026-07-16.** This list previously read `notificator.address` etc.
    Those are internal *config keys*, not env var names — setting them as env vars does
    nothing. GateGuard uses `viper.AutomaticEnv()` + `SetEnvKeyReplacer(".", "_")`
    (`gateguard/cmd/root/root.go:36-37`), so the key `notificator.address` is fed by
    `NOTIFICATOR_ADDRESS`.
  - ⚠️ If unset, GateGuard does **not** fail — it falls back to defaults pointing at
    Gateway.fm infra (`smtp.gmail.com:587`, `infra@gateway.fm`, empty password;
    `gateguard/config/init.go:37-42`) and verification mail silently fails. Confirm the
    vars are really present in the deployed container.
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
