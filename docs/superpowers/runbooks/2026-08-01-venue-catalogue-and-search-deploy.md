# 2026-08-01 — venue catalogue batch 2 + venue search deploy

Closes the QA finding that popular places ("Дом Радио", "Ton Center", "NoDom")
could not be found. That was two independent causes, fixed separately:

1. **Data.** The original 114-row hand-picked seed skewed to yoga studios,
   coworkings and buddhist centres — almost no concert halls, philharmonics,
   drama theatres, libraries-as-lecture-venues or clubs. ТОН-Центр did not
   exist yet when that seed was built (it opened in the Круговое депо in July
   2026).
2. **Search.** `venues.Search` was `name ILIKE '%q%'` and nothing else. Typing a
   street found none of the venues on it, and «Noôdome» was unreachable for
   anyone who did not type the circumflex.

Identifications: "Ton Center" = **ТОН-Центр**, Москва, Комсомольская пл.,
3/30с1. "NoDom" = **Noôdome**, Москва, Романов пер., 2с1. Neither is in SPb.

## Part 1 — 50 venues seeded (applied 08:20 UTC)

Artifacts in `docs/research/`: `venues_seed_2_candidates.json` →
`venues_seed_2.json` → `venues_seed_2.sql` → `run_venues_seed_2.sh`. Same shape
as batch 1: idempotent `WHERE NOT EXISTS (lower(name))`, deterministic uuid5
ids, runner backs the table up before writing.

25 Moscow + 25 SPb, deduped against a live dump of prod — zero exact and zero
fuzzy name collisions. Coordinates from the **Yandex Geocoder run on the box**
(the only place `YANDEX_GEOCODER_KEY` lives):

```
ssh vdska2 'set -a; . /opt/lia/backend/.env.prod; set +a;
  curl -s --get https://geocode-maps.yandex.ru/1.x/ \
    --data-urlencode "apikey=$YANDEX_GEOCODER_KEY" \
    --data-urlencode "geocode=Санкт-Петербург, Итальянская улица, 27" \
    --data-urlencode format=json --data-urlencode results=1'
```

49 came back `precision=exact`, «Современник» `precision=number`. **Gotcha:** a
bare `"Москва, Лесная улица, 20с3"` resolves to **Пушкино, Московская область**
— Депо. Лесная was re-run with `ll=37.62,55.75&spn=0.8,0.5&rspn=1` to pin the
city. That override is recorded in `venues_seed_2.json`.

Dry run (BEGIN … ROLLBACK) showed 174; `--apply` wrote `50 INSERT 0 1`.
**venues 124 → 174.** Backup at `/tmp/venues-pre-seed-20260801-082014.sql.gz`
on the box and committed alongside the seed.

## Part 2 — search fix (commit `2a5f5d4`)

Search now matches **name, address and metro** with diacritics folded
(`unaccent`), plus a **`pg_trgm` similarity fallback on the name** so a
near-miss still lands — "NoDom" is not a substring of "noodome", so substring
matching alone could never have satisfied the report. `similarity` is 0.400,
above the 0.3 default threshold. Ranking: name hits, then closer trigram
matches, then name order. LIKE metacharacters in the query are escaped, so
"50%" searches for the character instead of matching every row.

Tests: 4 unit (escaping helper) + 7 integration (`-tags=integration` against
`TEST_DATABASE_URL`) covering address match, metro match, diacritic folding,
the "NoDom" near-miss, name-first ranking, wildcard literals, empty query.

Note `golangci-lint` cannot run in this checkout at all — installed binary is
v2, `.golangci.yml` is v1 format. Pre-existing, unrelated. `go vet` + `gofmt`
clean, full `go test ./...` green.

## Procedure (as executed) — migration first, then image

1. `scp` migrations 021 + 022 to `/opt/lia/backend/db/migrations/`.
2. Wrote `/opt/lia/migrate-022.sh` (same `migrate/migrate:v4.17.1` +
   `--network backend_default` pattern as `migrate-020.sh`, creds sourced from
   `.env.prod`) and ran it. **Lia DB 20 → 22, dirty=false.**
   - 021 = «Читательские группы» category (was pending, unrelated; a `migrate
     up` cannot skip it). Verified present.
   - 022 = `CREATE EXTENSION unaccent, pg_trgm` + `venues_name_trgm_idx`.
     Verified: both extensions installed, index exists,
     `unaccent('Noôdome')='Noodome'`, `similarity('noodome','nodom')=0.400`.
3. Build on Mac: `docker build --platform linux/amd64 -t
   backend-app:venuesearch-r1 .` → `Architecture=amd64` confirmed →
   `docker save | gzip` (8.8M) → `scp` to `/opt/lia/` → `docker load`.
4. Rollback tag `backend-app:rollback-venuesearch-20260801-084715`, then
   `docker tag backend-app:venuesearch-r1 backend-app:latest`.
5. Recreate with **all four** compose files + `--no-build`:
   `docker compose --env-file .env.prod -f docker-compose.yml -f
   docker-compose.prod.yml -f docker-compose.gateguard.yml -f
   docker-compose.monitoring.yml up -d --no-build --force-recreate app`.
   `backend-migrate-1` ran as a dependency and reported "no change".
6. Verified live (`api.presence.tarski.ru/api/v1/venues?q=…`):

   | query | result |
   |---|---|
   | `NoDom` | Noôdome — the literal query from the report |
   | `Noodome` | Noôdome |
   | `Фонтанки` | 6 venues on the Fontanka (was 0) |
   | `Технологический` | Клуб «Космонавт» via metro (was 0) |
   | `Тон` | ТОН-Центр first, then «16 Тонн», then Товстоногова |
   | `Дом Радио` | Дом Радио |
   | `50%` | 0 rows — wildcard escaped, did not match everything |

   Smoke: `/api/v1/events?status=published`, `/api/v1/venues`,
   `/api/v1/categories`, `/`, `/search`, `/map` — all 200.
7. Prune: `docker builder prune -f` + `docker image prune -f`, removed the
   uploaded tarball, trimmed `backend-app` rollback tags to the newest 3
   (dropped `rollback-qa20-*`, `rollback-r5`, `rollback-r4`). Disk 75%.

## Rollback

```
ssh vdska2
cd /opt/lia/backend
docker tag backend-app:rollback-venuesearch-20260801-084715 backend-app:latest
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml \
  up -d --no-build --force-recreate app
```

DB rollback is only needed if the image is reverted **and** something depends on
022 being absent — unlikely, since the extensions are additive. `migrate … down
1` twice takes 22 → 20; the 022 down drops only `venues_name_trgm_idx` and
deliberately leaves the extensions installed (they are database-wide and
another module may adopt them).

## Follow-ups

- **Latin↔Cyrillic transliteration is still unhandled** — "Erarta" will not find
  «Эрарта». Trigram similarity does not bridge alphabets. Needs either an alias
  column or a transliteration map.
- `YANDEX_PLACES_KEY` deliberately left unprovisioned (explicit decision), so
  the Places fallback for venues outside the DB stays inert. Every new venue
  therefore still arrives via a seed batch or user input.
- The batch-2 seed artifacts landed in commit `aba6739` ("feat(admin): bring the
  complaints inbox and settings onto the Swiss Grid") rather than their own
  commit — a concurrent Claude session on the same checkout swept them up with a
  broad `git add`. Content is intact; the commit message is just wrong for those
  four files. See the shared-worktree hazard note in project memory.
