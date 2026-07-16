# Browser-Agent Prompt — Set up SendPulse sending + nic.ru DNS for `tarski.ru`

> **How to use this file:** Start a new Claude session (or a browser-capable agent) and paste the block under **"PROMPT TO HAND THE AGENT"** below. You (the human) will do every login manually — the agent pauses and waits for you at each auth step. The agent's job is to navigate, read the exact values off each dashboard, enter the DNS records, and write everything it captured to a local file.

## Background (context for you, the operator)

- **Goal:** make the Lia app able to send transactional email (email-verification codes + event invitations) from **`info@tarski.ru`** via **SendPulse** SMTP, with proper **SPF / DKIM / DMARC** on the `tarski.ru` domain so mail lands in inboxes (Gmail/Yandex/Mail.ru), not spam.
- **Domain registrar / DNS host:** **nic.ru** (RU-Center), where `tarski.ru` is managed. Live site is `presence.tarski.ru`, API `api.presence.tarski.ru`.
- **Email provider:** **SendPulse** (free tier: 12,000 emails/month; includes SMTP + DKIM/SPF). Chosen in the design.
- **What the code needs back from this:** SMTP host, SMTP port, SMTP login, SMTP password, and confirmation that the sending domain `tarski.ru` verifies green in SendPulse. These get set as env vars on the two backend services.
- **Output file the agent must produce:** `docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md` (this file is git-ignored — see the note at the very bottom; it will contain secrets).

---

## PROMPT TO HAND THE AGENT

```
You are a browser-automation assistant using the Claude-in-Chrome tools. Your task
is to configure SendPulse (transactional email) and nic.ru (DNS) so the domain
`tarski.ru` can send authenticated email from `info@tarski.ru`. The HUMAN will
perform all logins manually — you must PAUSE and ask them to log in whenever an
auth wall appears, and never attempt to enter passwords yourself.

## Ground rules
- Load the Chrome tools first with ONE ToolSearch call:
  select:mcp__claude-in-chrome__tabs_context_mcp,mcp__claude-in-chrome__navigate,mcp__claude-in-chrome__computer,mcp__claude-in-chrome__read_page,mcp__claude-in-chrome__tabs_create_mcp,mcp__claude-in-chrome__get_page_text
- Call tabs_context_mcp first. Open a NEW tab for this work (don't hijack the human's tabs).
- NEVER click anything that pops a native confirm/alert dialog (e.g. "Delete"). If a
  destructive action seems required, stop and ask the human.
- When you hit a login screen, STOP and say exactly: "Please log in manually in the
  browser tab, then tell me to continue." Wait for the human. Do not proceed until they say so.
- After every important screen, use read_page / get_page_text to CAPTURE exact values
  (never paraphrase secrets or DNS record values — copy them verbatim).
- Take a screenshot (computer screenshot) at each milestone so the human can verify.
- If you get stuck or a page behaves unexpectedly after 2-3 attempts, STOP and ask the
  human rather than guessing.
- As you capture values, keep appending them to the output file (see "OUTPUT" at the end)
  so nothing is lost if the session is interrupted.

## PHASE 1 — SendPulse: create/verify the sending domain and get SMTP creds

1. Navigate to https://sendpulse.com (or https://login.sendpulse.com). If not logged in,
   PAUSE for manual login/registration. (If registering: the human enters their own email +
   password. The account's own login email can be anything; it does NOT have to be @tarski.ru.)

2. Once logged in, find the SMTP / transactional email section. It is usually under:
   "SMTP" in the top nav, or Settings → "SMTP" → "Sending" / "Senders" / "Service settings".
   The exact menu wording changes over time — navigate by meaning, and read_page to confirm
   you're on the SMTP settings area. Report the URL you landed on.

3. Add `tarski.ru` as a sending DOMAIN (not just a single sender address):
   - Look for "Sending domains", "Domains", "Add domain", or "Verify your domain".
   - Enter `tarski.ru` and start the domain-verification / DNS-setup flow.
   - SendPulse will now DISPLAY the DNS records you must add (SPF and DKIM, sometimes a
     verification TXT). CAPTURE THESE EXACTLY. There are typically:
       * an SPF value, e.g. `v=spf1 include:sendpulse.com ~all` (copy the EXACT include host shown)
       * a DKIM record: a host/name (a selector like `sp._domainkey` or similar) and a long
         TXT value starting `v=DKIM1; k=rsa; p=...` — copy the FULL key, it is long.
       * possibly an extra TXT verification token.
     Write each record's TYPE, NAME/HOST, and full VALUE to the output file under
     "SendPulse-provided DNS records".

4. Add `info@tarski.ru` as a verified SENDER address if SendPulse asks for a specific
   sender (some flows send a confirmation email to info@tarski.ru — note this; the human may
   need mailbox access to that address later, OR domain verification alone may suffice).
   Record whether a sender-confirmation email to info@tarski.ru is required.

5. Find the SMTP CONNECTION credentials (usually Settings → SMTP → "SMTP" tab, labeled
   "SMTP server settings" or "Configure your SMTP"). CAPTURE VERBATIM:
       * SMTP server / host (e.g. smtp-pulse.com)
       * SMTP port (SendPulse commonly offers 2525, 465, 587 — record ALL offered)
       * SMTP login / username
       * SMTP password (this may be shown, or need to be generated/revealed — if a button
         reveals or generates it, click it, then capture; do NOT regenerate an existing one
         without asking the human)
   Write these to the output file under "SMTP credentials (SECRET)".

Do NOT click any final "Verify" button on the domain yet — first the DNS records must exist
at nic.ru (Phase 2). You'll return to verify in Phase 3.

## PHASE 2 — nic.ru: add the DNS records to tarski.ru

6. Open a new tab to https://www.nic.ru (RU-Center). PAUSE for manual login.

7. Navigate to DNS management for the `tarski.ru` zone. Path is typically:
   "Мои домены / My domains" → select `tarski.ru` → "DNS-серверы и управление зоной" /
   "Управление зоной" / "DNS zone management" / "Web-forwarding & DNS". Read_page to confirm
   you're editing the `tarski.ru` zone. If nic.ru shows the domain uses external DNS servers
   (not nic.ru's own), STOP and tell the human — the records must be added wherever the
   authoritative nameservers are, which might not be nic.ru's panel.

8. Add each record SendPulse gave you in step 3. For a TXT record nic.ru usually asks for:
   Subdomain/Name, Type=TXT, and the Value. Add:
     a. SPF — Name `@` (or blank / root), Type TXT, Value = the SendPulse SPF string.
        IMPORTANT: a domain may have only ONE SPF (`v=spf1 ...`) record. First CHECK the
        existing zone for an existing `v=spf1` TXT. If one exists, do NOT add a second —
        instead MERGE the SendPulse `include:` into the existing record and tell the human
        what you changed (capture the before/after). If none exists, add the SendPulse SPF as-is.
     b. DKIM — Name = the selector host SendPulse showed (e.g. `sp._domainkey`), Type TXT,
        Value = the full `v=DKIM1; ...` string. nic.ru may auto-append the domain to the name;
        enter only the selector part if the UI shows the domain suffix separately.
     c. Verification TXT (if SendPulse required one) — Name `@`, Type TXT, Value = the token.
     d. DMARC — Name `_dmarc`, Type TXT, Value: `v=DMARC1; p=quarantine; rua=mailto:info@tarski.ru`
        (a sane starter policy). If a `_dmarc` record already exists, capture it and ask the
        human before changing.
   Save/apply the zone changes (nic.ru may have an explicit "Save" / "Применить" / "Add record"
   per row). Screenshot the final record list. Capture the exact final state of each record
   into the output file.

9. Tell the human DNS propagation typically takes 30 min – a few hours.

## PHASE 3 — verify

10. After the human confirms some time has passed (or immediately, to check), return to the
    SendPulse domain page and click its "Verify" / "Check records" button. Report which records
    show green/verified and which are still pending. If DKIM/SPF are still pending, tell the
    human to wait and re-run this step later. Do NOT loop-poll aggressively — check once,
    report, and let the human decide when to re-check.

11. When (and only when) SendPulse shows the domain verified, record "DOMAIN VERIFIED: yes"
    plus the timestamp in the output file.

## OUTPUT — write everything to this local file
Write/append your captured results to:
  /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/docs/superpowers/runbooks/2026-07-15-sendpulse-nicru-credentials.local.md

Use this structure (fill in real values; keep secrets exactly as shown by the dashboards):

  # SendPulse + nic.ru setup results — tarski.ru
  Date/time started:
  Operator confirmed logins: SendPulse [ ]  nic.ru [ ]

  ## SMTP credentials (SECRET — do not commit)
  SMTP host:
  SMTP port(s) offered:
  SMTP login:
  SMTP password:
  From address to use: info@tarski.ru

  ## SendPulse-provided DNS records (as shown)
  SPF:   name=__  type=TXT  value=__
  DKIM:  name=__  type=TXT  value=__
  Verify TXT (if any): name=__ value=__

  ## nic.ru zone — records actually added (final state)
  SPF (added or merged? note before/after):
  DKIM:
  Verify TXT:
  DMARC: name=_dmarc value=v=DMARC1; p=quarantine; rua=mailto:info@tarski.ru

  ## Sender confirmation
  Did SendPulse require confirming info@tarski.ru by email? yes/no. If yes, what's needed:

  ## Verification status
  SendPulse domain verified: pending/yes (timestamp):
  Records still pending:

  ## Notes / anything that needed a human decision

## Final report back to the human (chat, <15 lines)
- What got done vs. what's pending (esp. DNS propagation / domain verification).
- The SMTP host/port/login (you may show these in chat; treat the PASSWORD as sensitive —
  say it's saved in the local file rather than pasting it in chat).
- Any step where you needed the human and what you still need from them.
- Confirm the output file path you wrote.
```

---

## After the agent finishes (what I — Claude Code — will do with the results)

Once `2026-07-15-sendpulse-nicru-credentials.local.md` exists and the domain verifies, hand me the SMTP host/port/login/password and I'll wire them as env on both services:

- **GateGuard** (verification emails): `notificator.address` = `<host:port>`, `notificator.username`, `notificator.password`, `notificator.from = info@tarski.ru`.
- **Lia backend** (invitation emails, added in Plan 3): the same SMTP settings via the new mailer config + `PublicBaseURL=https://presence.tarski.ru`.

Then the real-inbox delivery smoke-test (send a verification code + an invite to a Gmail address, confirm inbox placement + DKIM pass) and deploy.

---

> **git-ignore note:** the credentials output file ends in `.local.md` and holds secrets — it must NOT be committed. Before the agent runs, ensure `*.local.md` (or that specific path) is in `.gitignore`. If it isn't, add it. This prompt file itself contains no secrets and is safe to commit.
