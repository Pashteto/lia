# Swiss Grid Phase 4 — U3 Подбор (smart-filter)

**Branch:** `redesign/swiss-grid-p4`  
**Base:** `main` @ `85d40b2` (Phase 3 merge)  
**Date:** 2026-07-29  
**Status:** Ready for merge (no deploy required this phase)

---

## Shipped

- **`/search` U3 · AI-подбор** replaces the `ComingSoon` stub with a deterministic smart-filter: four suggestion chips, free-text input, templated one-sentence answer, and ≤3 `EventModule` results with `Совпало: …` match reasons.
- **Pure helpers** — `discover-intent` (chip + free-text → `DiscoverIntent`) and `discover-rank` (catalogue → ranked picks + answer sentence) — covered by Vitest TDD.
- **Client shell** — `DiscoverBrowse` applies intents over TanStack Query (`fetchPublishedEvents`, optional `fetchNearbyEvents` for geo); loading uses `Skeleton`, empty/error use `EmptyState`; escape hatch `Точные фильтры →` always visible.
- **Deferred:** real `POST /discover` / LLM integration (`lia-ai-provider-constraint`). Caption stays `AI-подбор` per handoff; answers are templated, not model-generated.

---

## Heuristic limitations (known)

| Intent | Limitation |
|---|---|
| **Тихое в выходные** | Filters by category + capacity heuristics, not real crowd/attendance data. Weekend window is calendar-based (Sat–Sun). |
| **Бесплатно рядом** | Without geolocation → city-wide free events; answer must not claim distance. With geo → ≤5 km filter. |
| **Для двоих вечером** | Evening = Moscow hour ≥18 within next 7 days; no “pair” capacity signal in catalogue. |
| **С детьми** | No kids category in taxonomy; title/description keyword heuristics. Weak match → ≤3 upcoming with honest `ближайшие события`. |

---

## Deliberate deviations (pre-decided — do not reopen P1–P3)

1. **Non-AI brain.** Caption `AI-подбор`; templated Russian one-liners. No honesty disclaimer. Real `POST /discover` / LLM deferred.
2. **Route stays `/search`.** Handoff conceptual `/discover` not renamed; nav already points at `/search`.
3. **«Бесплатно рядом» without geo** → free events city-wide; answer «Бесплатные события в городе.» (no distance claim). With geo → ≤5 km.
4. **«С детьми»** — keyword heuristics; fallback `ближайшие события`, never invent fitness.
5. **No chat history / multi-turn.** One submit → one answer + results; new submit replaces.
6. **Mobile chips keep full labels** with horizontal scroll (U1 filter-bar pattern); optional `text-[8px]` on `max-sm:`.
7. **Match-reason strings omit `Совпало:` prefix** — `EventModule` prepends it.

---

## Verification

```bash
cd frontend && pnpm build && pnpm test && pnpm lint
```

| Command | Result |
|---|---|
| `pnpm build` | PASS |
| `pnpm test` | PASS — **120** tests (26 files) |
| `pnpm lint` | PASS |

### Browser fidelity (Task 4)

| Surface | Result |
|---|---|
| Desktop ≥1024 | **PASS** — caption, 34px title, input+→, four chips, escape footer, vs handoff U3 |
| Mobile 390 | **PASS** — header `ПОДБОР`, 20px title, stacked modules, tab «Подбор» active, tap targets ≥44px |
| Behaviour | **PASS** (2 chip empties **env-only** — June mocks + offline API; empty UI correct) |
| `rounded-` / `shadow-` grep | **PASS** — none on new surface |

**Concerns (non-blocking):** API offline during verification; `quiet_weekend` / `evening_pair` chips empty against stale June `MOCK_EVENTS` on 2026-07-29. Unit tests use fixed July fixtures. Geo deny waits ~10s before city-wide free results.

---

## Commits (since `85d40b2`)

| SHA | Message |
|---|---|
| `8ac3944` | checkpoint before checking out main (plan file) |
| `1124dcf` | feat(frontend): add discover intent parser for U3 chips and free text |
| `209ff72` | feat(frontend): add discover ranker and templated U3 answer sentences |
| `407016d` | feat(frontend): ship U3 Podbor smart-filter on /search |
| `9c15086` | fix(frontend): U3 empty CTAs 44px and retry after refetch |

---

## Deploy

Optional — may bundle with a later phase per product preference. Phase 3 deploy follow-up remains open.

---

## Merge notes

- **PR title (when opened):** `Swiss Grid Phase 4 — U3 Подбор (smart-filter)` — base `main`.
- **No force-push.** Rebase onto latest `main` if drifted.
- **Post-merge:** refresh mock event dates or ensure backend is up for full four-chip browser demo; wire `POST /discover` only after product + `lia-ai-provider-constraint` sign-off.
