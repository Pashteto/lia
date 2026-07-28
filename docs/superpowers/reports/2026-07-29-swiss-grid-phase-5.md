# Swiss Grid Phase 5 — Organizer suite (O1–O5)

**Branch:** `redesign/swiss-grid-p5`  
**Base:** `8c3137f` — Phase 4 handoff tip pin  
**HEAD:** `32280a5`  
**Date:** 2026-07-29  
**Status:** Ready for merge (deploy **not** run this task)

---

## Shipped

| Screen | Route | Summary |
|---|---|---|
| **O1 Кабинет** | `/organizer` | `OrganizerHub` — identity + CTA, 4-cell status strip (3 on mobile), signal pending-applications banner, next-event card + progress bar, activity stub rail |
| **O3 Мои события** | `/events/mine` | `MyEventsBrowse` — filter chips with counts, desktop 5-col table, mobile stacked rows, `Ред.`/`Копия`/`···` overflow (invite/apps/feedback/publish), client duplicate-as-draft |
| **O2 Создание** | `/events/new`, `/events/[id]/edit` | `CreateEventForm` wizard — 4 visual steps, inclusive stepper, autosave chip (edit), ghost draft (create), `EventModule` preview rail, cross-step validation |
| **O4 Заявки** | `/organizer/applications` | `OrganizerApplications` picker + `EventApplicationsPanel` — tabs, checkboxes, bulk accept loop, optimistic seats; mobile hides bulk bar |
| **O5 Профиль** | `/me/organizer`, `/organizers/[id]` | `OrganizerProfileEdit` (exclusive verification stepper + preview card) + `PublicOrganizerView` (identity, stats, follow CTA) |

**Shared infrastructure (Task 1–2):** pure helpers (`org-event-status`, `org-seats`, `org-dashboard`, `org-verification`, `relative-time`, `org-applications`, `org-duplicate-event`) with Vitest TDD; `Stepper` `fillMode` inclusive/exclusive; `TabBarGate` hides tab bar on `/me/organizer`.

All organizer routes use `AppHeader nav={ORG_NAV}` with active underline per screen.

---

## Deliberate deviations (pre-decided — do not reopen P1–P4)

1. Routes stay `/organizer`, `/events/mine`, `/events/new`, `/organizer/applications`, `/me/organizer` — not `/org/*`.
2. No `GET /org/summary`; O1 tiles from `fetchMyEvents` (+ application fetches). Activity log stub only — no fake rows.
3. «Всего записей» = sum of known filled seats; unknown → `—`.
4. O4 on `/organizer/applications` with event picker + `?event=` deep link.
5. Bulk accept/reject = sequential client `decideMany` loop.
6. Application meta from trimmed `applicationAnswer` or «Первая заявка» — no invented attendance history.
7. O2 autosave: edit = debounced blur `patchEvent`; create = ghost `ЧЕРНОВИК` → draft + navigate. Stepper visual only; one RHF form.
8. O2 four-step field remap (01–04) over existing `eventFormSchema`.
9. Invite/feedback expanders behind row `···` on O3 (not in mock columns).
10. «Копия» = `createEvent` draft duplicate + edit nav (cover not copied).
11. O5 contacts: `website_url` + optional read-only auth email; no telegram field.
12. Public follower count → `—` (no API).
13. Public logo missing → ink square placeholder.
14. O1 logged-out → `AuthGate` (not link-card hub).
15. O2 stepper inclusive; O5 verification stepper exclusive.
16. Tab bar hidden on `/me/organizer`; kept on public `/organizers/[id]`.

**Locked from earlier phases:** favorites deferred, home `/`, U4 personal calendar, Yandex not OSM, Golos/Manrope/JB Mono, U3 non-AI `/search`, etc.

---

## Parked / deferred (needs human browser verify before pixel-complete claim)

| Item | Source | Notes |
|---|---|---|
| **Live pixel QA O1–O5** | Tasks 3–7 ledger | No full 390px + desktop pass against HTML badges `:568–837` in this phase close |
| **O1 banner when pending = 0** | Task 8 checklist | Unit logic hides banner at 0; not browser-verified live |
| **O4 optimistic accept + seat bump** | Task 6 | Backend offline during dev; local state only |
| **O5 exclusive stepper + preview proportions** | Task 7 | Automated gates green; pixel compare deferred |
| **O2 `ImageUpload` / `VenuePicker` internal chrome** | Task 5 | Swiss-wrapped; inner liquid-glass not fully rewritten |
| **O5 edit vs public event-count semantics** | Task 7 | Preview counts published; public counts all organizer events |
| **Production deploy** | Task 8 Step 3 | Skipped pending user request |

---

## Verification

```bash
cd frontend && pnpm build && pnpm test && pnpm lint
```

| Command | Result |
|---|---|
| `pnpm build` | PASS |
| `pnpm test` | PASS — **140** tests (33 files) |
| `pnpm lint` | PASS |

### Browser fidelity (Task 8)

| Check | Result |
|---|---|
| Full O1–O5 checklist @ 390 + desktop | **DEFERRED** — see parked items |
| `rounded-` / `shadow-` on new surfaces | PASS (grep + task reviews) |
| Tab bar hidden on `/me/organizer` | PASS (code + Task 2) |
| ORG_NAV active underline | PASS (code review; live verify deferred) |

---

## Commits (since `8c3137f`)

| SHA | Message |
|---|---|
| `6e86627` | feat(frontend): add Phase 5 organizer pure helpers |
| `867b128` | fix(frontend): Stepper fillMode + hide org profile tab bar |
| `d9b9bb1` | feat(frontend): Swiss Grid O1 organizer hub |
| `fdc91a8` | feat(frontend): Swiss Grid O3 my events table |
| `bbbe09d` | fix(frontend): Swiss Grid PublishEventButton in O3 overflow |
| `526b461` | feat(frontend): Swiss Grid O2 create-event wizard chrome |
| `eb74a2f` | fix(frontend): jump O2 wizard to first invalid step |
| `266d820` | feat(frontend): Swiss Grid O4 applications suite |
| `fd7dd61` | fix(frontend): hide O4 bulk bar on mobile |
| `32280a5` | feat(frontend): Swiss Grid O5 organizer profile |

---

## Deploy

**Skipped** — per task scope (no production deploy unless user asks). Runbook: `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`.

---

## Merge notes

- **PR title (when opened):** `Swiss Grid Phase 5 — Organizer suite (O1–O5)` — base `main` (must include Phase 4).
- **No force-push.** Rebase onto latest `main` if drifted.
- **Before claiming pixel-complete:** run live browser checklist against HTML badges; verify O4 bulk accept against running API.
