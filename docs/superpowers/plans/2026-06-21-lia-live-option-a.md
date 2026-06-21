# Lia → live (Option A) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve `lia.pashteto.com` from a real Go backend + PostgreSQL/PostGIS with seeded Moscow events, instead of the built-in mock data.

**Architecture:** A committed `docker-compose.prod.yml` override runs the existing `app` + `postgres` + `migrate` stack on oracle-1 with the app host-bound to `127.0.0.1:9080` (Postgres compose-internal only). A committed idempotent `seed.sql` loads real venues/events. The `lia-frontend` image is rebuilt with `NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com`. nginx + certbot expose `api.lia.pashteto.com` → `:9080`. Mock auth stays on (Option A scope).

**Tech Stack:** Docker Compose, PostgreSQL 16 + PostGIS 3.4, Go modular monolith, Next.js, nginx, certbot.

## Global Constraints

- **Do not touch** dollbuilder (`:8080`) or any existing oracle-1 vhost/container. All additions are `lia-*`-prefixed and additive.
- **No secrets in git.** `backend/.env.prod` (real DB creds) is uncommitted and git-ignored; only an `.env.prod.example` with placeholders is committed.
- **Postgres is never published** off the compose network in prod.
- **CORS** is narrowed to `https://lia.pashteto.com` in prod (from the `*` default).
- `HTTP_MOCK_AUTH=true` is intentional for Option A — a known non-production control, to be called out at review.
- oracle-1 stays a documented hand-managed demo exception to the org Terraform/Secrets-Manager standard — not a production precedent.
- Migrations run through `000009_venue_geo` (already on `main`). Schema is fixed; the seed must match it.
- Box-side steps (DNS, nginx, certbot, `docker compose` on the host) require host access — the **user** runs these via `! <cmd>`; the agent prepares exact commands and committed files only.

### Schema reference (verbatim, for the seed)

`events`: `id uuid PK`, `organizer_id uuid NOT NULL DEFAULT zero-uuid`, `venue_id uuid NOT NULL DEFAULT zero-uuid`, `title text`, `description text DEFAULT ''`, `status event_status DEFAULT 'draft'` (enum: draft/pending_review/published/rejected/cancelled), `format text DEFAULT 'offline'`, `price_type text DEFAULT 'free'`, `price_min int`, `price_max int`, `external_ticket_url text DEFAULT ''`, `starts_at timestamptz NOT NULL`, `ends_at timestamptz`, `published_at timestamptz`, `created_at`, `updated_at`.

`venues`: `id uuid PK`, `name text`, `address text DEFAULT ''`, `metro text DEFAULT ''`, `district text DEFAULT ''`, `lat double precision NULL`, `lon double precision NULL`, `geog` (generated — never insert), `created_at`, `updated_at`.

`categories`: seeded slugs — `lecture, workshop, mediation, concert, exhibition, performance, film, festival`. UUIDs are `gen_random_uuid()` → unknowable; resolve by slug.

`event_categories`: `(event_id uuid FK→events, category_id uuid FK→categories)` PK both. FK to events is `ON DELETE CASCADE`.

**Mock titles to NOT reuse** (so verification can prove real data): "Память и архив: разговор у работ", "Бумага ручного отлива", "Что значит смотреть вместе", "Читаем Зебальда".

---

### Task 1: Production compose override + env scaffold

**Files:**
- Create: `backend/docker-compose.prod.yml`
- Create: `backend/.env.prod.example`
- Modify: `backend/.gitignore` (add `.env.prod`)

**Interfaces:**
- Consumes: existing `backend/docker-compose.yml` services `postgres`, `migrate`, `app`.
- Produces: a prod override applied as `-f docker-compose.yml -f docker-compose.prod.yml`; env vars `DATABASE_USER`, `DATABASE_PASSWORD`, `DATABASE_NAME` sourced from `backend/.env.prod`.

- [ ] **Step 1: Add `.env.prod` to backend `.gitignore`**

The existing `.gitignore` line `.env` does NOT match `.env.prod`. Add an explicit entry under the existing `.env`:

```
.env
.env.prod
```

- [ ] **Step 2: Create `backend/.env.prod.example`** (committed; placeholders only)

```dotenv
# Copy to backend/.env.prod on oracle-1 and fill with NON-DEFAULT values.
# .env.prod is git-ignored — never commit real credentials.
DATABASE_USER=lia_prod
DATABASE_PASSWORD=<REDACTED>
DATABASE_NAME=lia_prod
```

- [ ] **Step 3: Create `backend/docker-compose.prod.yml`**

```yaml
# Production override for oracle-1. Apply with:
#   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
# Adjusts only what differs from local dev. Credentials come from .env.prod
# (git-ignored) — never the committed dev/dev defaults.

services:
  postgres:
    # Remove the host port mapping: Postgres is compose-internal only in prod.
    ports: !reset []
    environment:
      POSTGRES_USER: ${DATABASE_USER}
      POSTGRES_PASSWORD: ${DATABASE_PASSWORD}
      POSTGRES_DB: ${DATABASE_NAME}
    volumes: !reset
      - lia_pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DATABASE_USER} -d ${DATABASE_NAME}"]

  migrate:
    command:
      - "-path=/migrations"
      - "-database"
      - "postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@postgres:5432/${DATABASE_NAME}?sslmode=disable"
      - "up"

  app:
    # Host-bind to 127.0.0.1:9080 (nginx fronts it). Do NOT publish gRPC.
    ports: !reset
      - "127.0.0.1:9080:8080"
    environment:
      DATABASE_USER: ${DATABASE_USER}
      DATABASE_PASSWORD: ${DATABASE_PASSWORD}
      DATABASE_NAME: ${DATABASE_NAME}
      HTTP_CORS_ALLOWED_ORIGINS: "https://lia.pashteto.com"
      GRPC_ENABLED: "false"

volumes:
  lia_pgdata:
```

Notes: `!reset` is Compose's merge-reset tag — it replaces (not appends to) the base list, so the dev `5432:5432` mapping and the `./data/db/postgres` bind-mount are dropped rather than merged. `env_file` is not used; vars are interpolated from the shell environment, so the apply command must `set -a; . ./.env.prod; set +a` first (see Task 4). The base `HTTP_MOCK_AUTH: "true"` is inherited unchanged (intentional for Option A).

- [ ] **Step 4: Validate the merged config locally (no containers started)**

Run from `backend/`:
```bash
docker compose --env-file .env.prod.example -f docker-compose.yml -f docker-compose.prod.yml config
```
Expected: prints the resolved config with NO error; `postgres` has **no** published ports; `app` publishes only `127.0.0.1:9080→8080` (no 9090); `${DATABASE_*}` substituted; a top-level `lia_pgdata` volume exists. Use Compose's `--env-file` (parses `KEY=VALUE` literally) rather than `set -a; . file` — shell-sourcing breaks on `<>`/special chars in passwords. List-replacing keys use the `!override` merge tag (not `!reset`, which *empties* the list and would drop the replacement).

- [ ] **Step 5: Verify `.env.prod` is ignored**

```bash
cd backend && touch .env.prod && git check-ignore .env.prod && rm .env.prod
```
Expected: prints `.env.prod` (confirming it is ignored), then removes the stray file.

- [ ] **Step 6: Commit**

```bash
git add backend/docker-compose.prod.yml backend/.env.prod.example backend/.gitignore
git commit -m "feat(deploy): prod compose override for oracle-1 (Option A)"
```

---

### Task 2: Seed data — real Moscow venues + events

**Files:**
- Create: `backend/db/seed/seed.sql`

**Interfaces:**
- Consumes: tables `venues`, `events`, `categories`, `event_categories` (schema above).
- Produces: ~8 venues + ~9 published events with fixed UUIDs, queryable via `GET /events?status=published` and `GET /events/nearby`.

- [ ] **Step 1: Create `backend/db/seed/seed.sql`** (idempotent; fixed UUIDs)

```sql
-- Lia demo seed (Option A). Idempotent: safe to re-run. NOT wired as a compose
-- service — run manually once after the stack is healthy. Public venue data
-- only: no PII, no secrets. Coordinates are real; geog is a generated column
-- (never inserted). organizer_id left as the zero UUID (loose ref, scaffold
-- convention). Titles are deliberately distinct from frontend/lib/mock-events.ts
-- so a successful render proves real data, not the mock fallback.

-- ── Venues ──────────────────────────────────────────────────────────────────
INSERT INTO venues (id, name, address, metro, district, lat, lon) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'Музей «Гараж»',            'ул. Крымский Вал, 9, стр. 32', 'Парк культуры',    'ЦАО', 55.728990, 37.601510),
  ('a0000000-0000-0000-0000-000000000002', 'Дом культуры «ГЭС-2»',      'Болотная наб., 15',            'Кропоткинская',    'ЦАО', 55.740280, 37.609540),
  ('a0000000-0000-0000-0000-000000000003', 'Новая Третьяковка',         'ул. Крымский Вал, 10',         'Октябрьская',      'ЦАО', 55.734400, 37.604900),
  ('a0000000-0000-0000-0000-000000000004', 'ГМИИ им. Пушкина',          'ул. Волхонка, 12',             'Кропоткинская',    'ЦАО', 55.744700, 37.605900),
  ('a0000000-0000-0000-0000-000000000005', 'Центр «Зотов»',             'ул. Ходынская, 2, стр. 1',     'Улица 1905 года',  'ЦАО', 55.766600, 37.560400),
  ('a0000000-0000-0000-0000-000000000006', 'Центр «Винзавод»',          '4-й Сыромятнический пер., 1',  'Чкаловская',       'ЦАО', 55.754700, 37.664200),
  ('a0000000-0000-0000-0000-000000000007', 'Электротеатр «Станиславский»', 'ул. Тверская, 23',          'Тверская',         'ЦАО', 55.766100, 37.601900),
  ('a0000000-0000-0000-0000-000000000008', 'Еврейский музей',           'ул. Образцова, 11, стр. 1А',   'Марьина Роща',     'СВАО', 55.792200, 37.604800)
ON CONFLICT (id) DO NOTHING;

-- ── Events ──────────────────────────────────────────────────────────────────
-- starts_at/published_at are absolute (no now()) so re-runs are deterministic.
INSERT INTO events
  (id, venue_id, title, description, status, format, price_type, price_min, price_max, starts_at, ends_at, published_at) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
   'Кураторская экскурсия по новой экспозиции',
   'Прогулка с куратором по главному выставочному залу: ключевые работы и их контекст.',
   'published', 'offline', 'free', NULL, NULL,
   '2026-07-05 18:00:00+03', '2026-07-05 19:30:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
   'Мастерская керамики: ручная лепка',
   'Практическое занятие для начинающих: основы ручной лепки и работа с глазурью.',
   'published', 'offline', 'paid', 1500, 2500,
   '2026-07-08 12:00:00+03', '2026-07-08 15:00:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
   'Лекция: русский авангард и его наследие',
   'Обзорная лекция о художниках начала XX века и их влиянии на современное искусство.',
   'published', 'offline', 'free', NULL, NULL,
   '2026-07-10 19:00:00+03', '2026-07-10 20:30:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
   'Медиация в залах старых мастеров',
   'Совместное медленное рассматривание нескольких картин в сопровождении медиатора.',
   'published', 'offline', 'free', NULL, NULL,
   '2026-07-12 16:00:00+03', '2026-07-12 17:00:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000005',
   'Выставка: конструктивизм и хлеб',
   'Постоянная экспозиция о конструктивизме; вход по сеансам.',
   'published', 'offline', 'paid', 500, 500,
   '2026-07-03 11:00:00+03', '2026-07-03 21:00:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000006',
   'Концерт современной академической музыки',
   'Вечер новой музыки: произведения современных композиторов в исполнении ансамбля.',
   'published', 'offline', 'paid', 1200, 2000,
   '2026-07-15 20:00:00+03', '2026-07-15 22:00:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000007',
   'Спектакль: лабораторная сцена',
   'Экспериментальная постановка молодой режиссёрской группы.',
   'published', 'offline', 'paid', 2000, 3500,
   '2026-07-18 19:30:00+03', '2026-07-18 21:30:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000008',
   'Кинопоказ с обсуждением',
   'Показ документального фильма и разговор со зрителями после сеанса.',
   'published', 'offline', 'free', NULL, NULL,
   '2026-07-20 18:30:00+03', '2026-07-20 21:00:00+03', '2026-06-21 10:00:00+03'),
  ('b0000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000001',
   'Летний фестиваль медиаискусства',
   'Программа выходного дня: инсталляции, перформансы и встречи с художниками.',
   'published', 'offline', 'free', NULL, NULL,
   '2026-07-25 12:00:00+03', '2026-07-26 22:00:00+03', '2026-06-21 10:00:00+03')
ON CONFLICT (id) DO NOTHING;

-- ── Event ↔ category (resolve category UUID by slug) ─────────────────────────
INSERT INTO event_categories (event_id, category_id)
SELECT e.event_id, c.id
FROM (VALUES
  ('b0000000-0000-0000-0000-000000000001'::uuid, 'mediation'),
  ('b0000000-0000-0000-0000-000000000002'::uuid, 'workshop'),
  ('b0000000-0000-0000-0000-000000000003'::uuid, 'lecture'),
  ('b0000000-0000-0000-0000-000000000004'::uuid, 'mediation'),
  ('b0000000-0000-0000-0000-000000000005'::uuid, 'exhibition'),
  ('b0000000-0000-0000-0000-000000000006'::uuid, 'concert'),
  ('b0000000-0000-0000-0000-000000000007'::uuid, 'performance'),
  ('b0000000-0000-0000-0000-000000000008'::uuid, 'film'),
  ('b0000000-0000-0000-0000-000000000009'::uuid, 'festival')
) AS e(event_id, slug)
JOIN categories c ON c.slug = e.slug
ON CONFLICT DO NOTHING;
```

- [ ] **Step 2: Validate SQL syntax against a throwaway local PostGIS** (proves the seed loads against the real schema)

Run from `backend/` (uses the existing local dev compose, which migrates to 000009):
```bash
docker compose up -d --build
# wait for healthy, then:
docker compose exec -T postgres psql -U dev -d lia_dev < db/seed/seed.sql
docker compose exec -T postgres psql -U dev -d lia_dev -c \
  "SELECT count(*) AS events FROM events WHERE status='published';
   SELECT count(*) AS venues FROM venues WHERE lat IS NOT NULL;
   SELECT count(*) AS links FROM event_categories;"
```
Expected: no SQL errors; `events`=9, `venues`=8 (≥8), `links`=9.

> If local Docker is flaky (see HANDOFF gotcha), run the app binary on the host against the containerized Postgres instead; the `psql` seed step is unaffected.

- [ ] **Step 3: Verify idempotency (re-run must not error or duplicate)**

```bash
docker compose exec -T postgres psql -U dev -d lia_dev < db/seed/seed.sql
docker compose exec -T postgres psql -U dev -d lia_dev -c "SELECT count(*) FROM events;"
```
Expected: no error; event count unchanged from Step 2.

- [ ] **Step 4: Verify `/events/nearby` returns seeded distances** (exercises the geog column)

```bash
curl -s "http://localhost:8080/api/v1/events/nearby?lat=55.75&lon=37.62&limit=50" | head -c 800
```
Expected: JSON array of events, each with a numeric `distance_m`, nearest-first.

- [ ] **Step 5: Tear down the throwaway stack**

```bash
docker compose down -v
```

- [ ] **Step 6: Commit**

```bash
git add backend/db/seed/seed.sql
git commit -m "feat(deploy): idempotent Moscow seed data (Option A)"
```

---

### Task 3: Frontend re-point (build arg confirmation)

**Files:**
- Verify only: `frontend/Dockerfile` (already exposes `ARG NEXT_PUBLIC_API_URL`).

**Interfaces:**
- Consumes: `frontend/Dockerfile` build arg `NEXT_PUBLIC_API_URL` (default `http://127.0.0.1:9`).
- Produces: a documented build command baking `https://api.lia.pashteto.com` into the image.

No code change is required — the Dockerfile already accepts the build arg. This task only confirms that and records the exact build command (executed on the box in Task 4).

- [ ] **Step 1: Confirm the build arg exists**

```bash
grep -n "ARG NEXT_PUBLIC_API_URL" frontend/Dockerfile
```
Expected: matches the `ARG NEXT_PUBLIC_API_URL=...` line. (If absent, add `ARG NEXT_PUBLIC_API_URL=http://127.0.0.1:9` and `ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL` before `pnpm build`, then commit.)

No commit unless the Dockerfile had to change.

---

### Task 4: Box deployment runbook (user-executed on oracle-1)

**Files:**
- Create: `docs/superpowers/runbooks/2026-06-21-option-a-deploy.md` (committed runbook; the box itself stays out of git).

**Interfaces:**
- Consumes: committed `docker-compose.prod.yml`, `seed.sql`, frontend `Dockerfile`.
- Produces: live `https://api.lia.pashteto.com` + re-pointed `https://lia.pashteto.com`.

> These commands run **on oracle-1** with host access. The agent does not have it — the user runs each via `! <cmd>` (or over SSH) and reports output back. Replace `<...>` placeholders with the non-default values chosen for `.env.prod`.

- [ ] **Step 1: Write the runbook file** with the contents below.

````markdown
# Option A deploy runbook (oracle-1)

Host: oracle-1 (129.146.183.89). Hand-managed; do NOT touch dollbuilder (:8080)
or other vhosts. All additions are lia-* and additive.

## 1. DNS (Namecheap BasicDNS)
Add an A record:  api.lia → 129.146.183.89
Verify: `dig +short api.lia.pashteto.com` → 129.146.183.89

## 2. Sync repo + create .env.prod (NOT in git)
rsync the repo's `backend/` to the box (same path convention as the frontend deploy).
On the box, in backend/:
  cp .env.prod.example .env.prod
  # edit .env.prod: set DATABASE_USER / DATABASE_PASSWORD / DATABASE_NAME to
  # NON-DEFAULT values. Never dev/dev.
  chmod 600 .env.prod

## 3. Bring up the backend stack
From backend/:
  set -a; . ./.env.prod; set +a
  docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
Verify Postgres is NOT published to the host:
  docker compose -f docker-compose.yml -f docker-compose.prod.yml ps   # no 0.0.0.0:5432
  ss -tlnp | grep 9080   # app bound to 127.0.0.1:9080 only
  ss -tlnp | grep 5432   # MUST be empty
Health:
  curl -s http://127.0.0.1:9080/api/v1/health

## 4. Seed (once, after healthy)
From backend/ (env still exported):
  docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    exec -T postgres psql -U "$DATABASE_USER" -d "$DATABASE_NAME" < db/seed/seed.sql
Verify:
  curl -s "http://127.0.0.1:9080/api/v1/events?status=published" | head -c 400

## 5. nginx vhost + TLS for api.lia.pashteto.com
Create /etc/nginx/sites-available/api.lia.pashteto.com (proxy_pass to 127.0.0.1:9080),
enable it, `nginx -t`, reload. Then:
  certbot --nginx -d api.lia.pashteto.com
(model the vhost on the existing lia.pashteto.com one; only the upstream port differs)

## 6. Rebuild + restart the frontend pointed at the real API
From frontend/ on the box:
  docker build --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend .
  docker rm -f lia-frontend
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend
````

- [ ] **Step 2: Commit the runbook**

```bash
git add docs/superpowers/runbooks/2026-06-21-option-a-deploy.md
git commit -m "docs(deploy): Option A box runbook (oracle-1)"
```

- [ ] **Step 3: User executes the runbook on the box** (DNS → env → compose up → seed → nginx/certbot → frontend rebuild), reporting output of each `Verify` line.

---

### Task 5: End-to-end verification (post-deploy)

**Files:** none (verification only).

> A broken backend makes the frontend **silently** fall back to mock data, so every check below targets data that exists ONLY in the seed.

- [ ] **Step 1: API returns seeded events**

```bash
curl -s "https://api.lia.pashteto.com/api/v1/events?status=published" | grep -c "Летний фестиваль медиаискусства"
```
Expected: `1` (a seed-unique title; absent from `lib/mock-events.ts`).

- [ ] **Step 2: `/events/nearby` returns real distances, nearest-first**

```bash
curl -s "https://api.lia.pashteto.com/api/v1/events/nearby?lat=55.75&lon=37.62&limit=50" | head -c 800
```
Expected: events with numeric `distance_m`, ascending; no `distance_m` on coordless data.

- [ ] **Step 3: Public site renders real (not mock) data**

Open `https://lia.pashteto.com`, an event detail page, and `/map`. Confirm a seed-unique title (e.g. "Летний фестиваль медиаискусства" or "Лекция: русский авангард и его наследие") appears. Its presence proves real data, not the mock fallback.

- [ ] **Step 4: Neighbors untouched**

```bash
curl -sI https://lia.pashteto.com | head -1   # still 200/OK
# confirm dollbuilder + other vhosts/containers still up (docker ps on box)
```
Expected: existing vhosts and the dollbuilder container unchanged and serving.

- [ ] **Step 5: Update HANDOFF**

Edit `docs/HANDOFF.md`: move "Option A — real backend behind the demo" from *What's next* to *Recently done*, note the live `api.lia.pashteto.com`, and flag `HTTP_MOCK_AUTH=true` as the known non-production control. Commit:
```bash
git add docs/HANDOFF.md && git commit -m "docs: Option A live; demo now serves real data"
```

---

## Self-Review

**Spec coverage:** Component A (prod compose) → Task 1. Component B (seed) → Task 2. Component C (frontend re-point) → Task 3 + runbook Step 6. Data flow / nginx / DNS → Task 4. Verification (curl events, nearby, seed-unique title, neighbors untouched) → Task 5. Security notes (no Postgres host port, non-default creds, narrowed CORS, mock-auth callout) → Task 1 + Global Constraints + Task 5 Step 4. All spec sections mapped.

**Placeholder scan:** Credentials are intentional `<REDACTED>`/`<...>` placeholders in committed example/runbook files (per org policy — no real secrets in git), not plan gaps. All code/SQL/YAML steps contain full content.

**Type/identity consistency:** Venue UUIDs `a0000000-…-0001..0008`; event UUIDs `b0000000-…-0001..0009`; every event `venue_id` references a defined venue id; every `event_categories` slug is one of the seeded eight. `geog` is never inserted (generated). App host-bind port `9080` consistent across compose override, runbook, and verification.
