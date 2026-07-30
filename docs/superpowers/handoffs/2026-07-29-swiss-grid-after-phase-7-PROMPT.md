# Prompt — after Swiss Grid Phase 7

**Save location:** `docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-PROMPT.md`  
**Handoff:** [`2026-07-29-swiss-grid-after-phase-7-HANDOFF.md`](./2026-07-29-swiss-grid-after-phase-7-HANDOFF.md)  
**Use:** paste the block below into a **new** agent chat to continue work.

---

## Paste this

```
You are continuing Presence.Tarski (repo Pashteto/lia) after Swiss Grid redesign is complete.

## Current state (trust the handoff, not stale memory)
Read first, in order:
1. docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md
2. docs/superpowers/reports/2026-07-29-swiss-grid-phase-7.md
3. docs/Redesign/5/design_handoff_presence_swiss_grid/README.md (only if the task is visual)

Facts:
- main @ 81514df is on origin/main and LIVE on https://presence.tarski.ru
- Swiss Grid master plan P1–P7 is DONE (A4 users + hygiene + P7.3 Liquid Glass sweep)
- Prod LIA DB level 020 — P7 was images-only (no migrate)
- Rollback tags: backend-app:rollback-swiss-p7-20260729-142227, lia-frontend-presence:rollback-swiss-p7-20260729-142227
- Do NOT reopen locked P1–P7 deviations without explicit product sign-off

## Design / hard constraints
- Swiss Grid: zero radius, zero shadows, 1px hairlines; admin data-surface="ink"
- Fonts: Golos Text / Manrope / JetBrains Mono; mono numerals; signal red only for attention/destructive
- A2–A4 desktop-only below 900px; no TanStack Query migration for admin
- UI copy Russian; code/commits English
- Deploy: Mac linux/amd64 build → docker save|gzip|ssh|load; recreate app with all 4 compose files; frontend needs BOTH NEXT_PUBLIC_API_URL and NEXT_PUBLIC_YANDEX_MAPS_KEY

## What to do next (pick with me — do not invent a new redesign phase)
Recommended order from the handoff:
1. Live-verify /admin/users as admin (desktop A4 fidelity + <900px gate). Report gaps only; fix if clearly broken vs plan.
2. Optional product items (ask which):
   - Wire A1 «Пользователей» tile (reopens locked P6 #5 — need OK)
   - A4 search UI (API q already exists)
   - Migrate remaining text-[Npx] toward named type utilities (~325 intentional sizes)
   - Formal Lighthouse/CLS pass
3. Ops (separate): YANDEX_PLACES_KEY provisioning; GATEGUARD_AUTH_SECRET rotation
4. Or: non–Swiss-Grid product backlog — brainstorm first (superpowers:brainstorming), then writing-plans, then worktree + implement

## Process
- Use superpowers skills (brainstorming before new creative work; writing-plans before multi-step impl; worktrees for feature branches; verification-before-completion before claiming done).
- Prefer subagent-driven-development for multi-task plans.
- Do not force-push main; do not migrate prod DB unless the plan explicitly requires it and I approve.
- Commit only when I ask; push only when I ask.

Start by confirming you read the handoff, summarizing remaining options in 5 bullets, and asking which option I want.
```

---

## One-liner (ultra-short)

```
Continue Pashteto/lia after Swiss Grid P1–P7 LIVE (main 81514df). Read docs/superpowers/handoffs/2026-07-29-swiss-grid-after-phase-7-HANDOFF.md + PROMPT sibling. Do not reopen locked deviations. Ask which follow-up: A4 live QA, A1 users tile, A4 search, typography pass, ops keys, or new product work (brainstorm first).
```
