# СПб: запуск второго города — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить понятие «город» (msk/spb) во всю систему — БД, API, фронтенд — задеплоить с СПб в состоянии «Скоро», засеять питерские площадки; открытие города — одна настройка в app_settings.

**Architecture:** Денормализованный `city` slug на `venues` и `events` (событие наследует город площадки); фильтр `?city=` с дефолтом `msk` на публичных выборках; `GET /api/v1/cities` отдаёт доступность (`cities.spb_available` в app_settings); фронт хранит выбор в cookie `lia_city` + `?city=` override.

**Tech Stack:** Go (go-pg v10, go-swagger codegen из `backend/api/swagger.yaml`, `make generate-api`), PostgreSQL (golang-migrate .sql), Next.js App Router + TypeScript + vitest.

**Spec:** `docs/superpowers/specs/2026-08-23-spb-city-launch-design.md`

## Global Constraints

- Slug города: `msk`, `spb` — латиница, нижний регистр; невалидный → 400.
- Отсутствие `?city=` на публичных выборках = `msk` (обратная совместимость).
- `/me/*` и админские выборки по городу НЕ фильтруются.
- Событие с `venue_id` всегда несёт город площадки; передать другой = 400.
- Backend codegen: после правки `swagger.yaml` — `make generate-api` (гоняется из `backend/`); руками сгенерённые файлы не редактировать.
- go-pg: NULL uuid не сканится — колонки NOT NULL DEFAULT, `use_zero`.
- Коммиты частые, тесты перед каждым коммитом: backend `go test ./...` из `backend/`, frontend `pnpm vitest run` из `frontend/`.

---

### Task 1: миграция 000027 — city на venues и events

**Files:**
- Create: `backend/db/migrations/000027_city_column.up.sql`
- Create: `backend/db/migrations/000027_city_column.down.sql`

**Interfaces:**
- Produces: колонки `venues.city`, `events.city` (text NOT NULL DEFAULT 'msk'), индексы `venue_city_name_lower_idx`, `events_city_starts_at_idx`.

- [ ] **Step 1: up-миграция**

```sql
-- Second city (СПб). City lives on BOTH venues and events (denormalized):
-- listings filter events without a join, and venue-less events (external,
-- drafts) still carry a city. Slug values ('msk','spb') are validated by the
-- backend against its constant whitelist — no lookup table (YAGNI for 2 rows).
-- See docs/superpowers/specs/2026-08-23-spb-city-launch-design.md.
ALTER TABLE venues ADD COLUMN IF NOT EXISTS city text NOT NULL DEFAULT 'msk';
ALTER TABLE events ADD COLUMN IF NOT EXISTS city text NOT NULL DEFAULT 'msk';

-- Venue search is always city-scoped now; replaces the plain name index.
DROP INDEX IF EXISTS venue_name_lower_idx;
CREATE INDEX IF NOT EXISTS venue_city_name_lower_idx
    ON venues (city, lower(name));

-- Public listing filters by city and orders by start time.
CREATE INDEX IF NOT EXISTS events_city_starts_at_idx
    ON events (city, starts_at);
```

- [ ] **Step 2: down-миграция**

```sql
DROP INDEX IF EXISTS events_city_starts_at_idx;
DROP INDEX IF EXISTS venue_city_name_lower_idx;
CREATE INDEX IF NOT EXISTS venue_name_lower_idx ON venues (lower(name));
ALTER TABLE events DROP COLUMN IF EXISTS city;
ALTER TABLE venues DROP COLUMN IF EXISTS city;
```

- [ ] **Step 3: прогнать локально up и down** (локальный Docker флаки — при отказе использовать host-run workaround из lia-dev-gotchas). Ожидаемо: обе применяются чисто.

- [ ] **Step 4: Commit** `feat(db): migration 000027 — city column on venues and events`

---

### Task 2: доменные константы города + поля моделей

**Files:**
- Create: `backend/internal/models/city.go`
- Create: `backend/internal/models/city_test.go`
- Modify: `backend/internal/models/event.go` (поле City)
- Modify: `backend/internal/models/venue.go` (поле City)

**Interfaces:**
- Produces: `models.CityMSK = "msk"`, `models.CitySPB = "spb"`, `models.ValidCity(s string) bool`, `models.DefaultCity = CityMSK`; `Event.City string pg:"city,use_zero"`, `Venue.City string pg:"city,use_zero"`.

- [ ] **Step 1: тест** (`city_test.go`)

```go
package models

import "testing"

func TestValidCity(t *testing.T) {
	for _, ok := range []string{"msk", "spb"} {
		if !ValidCity(ok) {
			t.Errorf("ValidCity(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "МСК", "SPB", "ekb", "moscow"} {
		if ValidCity(bad) {
			t.Errorf("ValidCity(%q) = true, want false", bad)
		}
	}
}
```

- [ ] **Step 2: убедиться, что тест падает** (`go test ./internal/models/ -run TestValidCity`) — undefined: ValidCity.

- [ ] **Step 3: реализация** (`city.go`)

```go
package models

// City slugs. Hardcoded whitelist — the city list is code, only availability
// (app_settings "cities.spb_available") is runtime. Latin lowercase slugs,
// NOT the display codes («МСК») the frontend renders.
const (
	CityMSK = "msk"
	CitySPB = "spb"
	// DefaultCity is assumed when a request carries no city (legacy clients).
	DefaultCity = CityMSK
)

// Cities lists every known city slug in display order.
var Cities = []string{CityMSK, CitySPB}

// ValidCity reports whether s is a known city slug.
func ValidCity(s string) bool {
	for _, c := range Cities {
		if s == c {
			return true
		}
	}
	return false
}
```

В `event.go` после `ExternalURLVerified`: `City string \`pg:"city,use_zero"\``; в `venue.go` после `District`: `City string \`pg:"city,use_zero"\``.

- [ ] **Step 4: тесты зелёные, Commit** `feat(models): city slug constants + city field on Event/Venue`

---

### Task 3: фильтр города в events-репозитории и сервисе

**Files:**
- Modify: `backend/internal/events/repository.go` (ListFilter + List)
- Modify: `backend/internal/events/service.go` (List подпись/валидация; создание/PATCH — наследование)
- Modify: `backend/internal/events/service_test.go` (или соседний тестовый файл по месту)

**Interfaces:**
- Consumes: `models.ValidCity`, `models.DefaultCity` (Task 2).
- Produces: `ListFilter.City string`; `Service.List(ctx, status, from, to, organizerOwnerID, city)` — city="" значит без фильтра (внутренние вызовы), хендлер передаёт явно; create/update проставляют `event.City`.

- [ ] **Step 1: тесты.** В стиле существующих unit-тестов events (fake repo): (a) `List` с city="spb" кладёт City в фильтр; (b) невалидный city → `ErrInvalidInput`; (c) Create с venue_id наследует город площадки, а явный конфликтующий city → ошибка валидации; (d) Create без площадки с city="" → `msk`.

- [ ] **Step 2: убедиться, что падают.**

- [ ] **Step 3: реализация.** В `ListFilter`:

```go
	// City, when non-empty, restricts to events in that city (slug, see
	// models.Cities). Public handlers always pass one; internal callers
	// (my-events, calendar re-enrich) leave it empty = all cities.
	City string
```

В `pgRepository.List` рядом с остальными Where: `if filter.City != "" { query = query.Where("city = ?", filter.City) }`.

В `service.List` — новый параметр `city string`, валидация: непустой и не `models.ValidCity(city)` → `ErrInvalidInput`; прокинуть в фильтр. Все существующие вызовы `List` внутри сервиса/других модулей передают `""`.

Наследование в Create/Update (там, где резолвится venue): если `event.VenueID != uuid.Nil` — загрузить venue (уже загружается для валидации) и `event.City = venue.City`; если пришёл явный city, отличный от города площадки → validation error `"city", "город события определяется площадкой"`. Без площадки: пустой city → `models.DefaultCity`; невалидный → validation error.

- [ ] **Step 4: `go test ./internal/events/...` зелёные, Commit** `feat(events): city filter in listing + city inheritance from venue`

---

### Task 4: swagger — city в API + GET /cities, codegen

**Files:**
- Modify: `backend/api/swagger.yaml`
- Regenerate: `make generate-api` (внутри `backend/`)

**Interfaces:**
- Produces: `?city=` на listEvents/listVenues; поле `city` в definitions `Event`, `Venue`, `EventInput`, `VenueInput` (если есть); новый public `GET /cities` → массив `City {code, available}`; операция `listCities` (tag `cities`, `security: []`).

- [ ] **Step 1: параметры.** В `listEvents.parameters` и `listVenues.parameters`:

```yaml
        - name: city
          in: query
          description: City slug (msk|spb). Defaults to msk when omitted.
          required: false
          type: string
          enum: [msk, spb]
```

- [ ] **Step 2: definitions.** В `Event`, `Venue` properties: `city: {type: string}`; в `EventInput`: `city: {type: string, enum: [msk, spb]}`. Новое:

```yaml
  City:
    type: object
    required: [code, available]
    properties:
      code:
        type: string
      available:
        type: boolean
```

- [ ] **Step 3: путь.**

```yaml
  /cities:
    get:
      summary: List cities
      description: Known cities and whether each is open for discovery.
      operationId: listCities
      tags: [cities]
      security: []
      responses:
        200:
          description: City list
          schema:
            type: array
            items:
              $ref: "#/definitions/City"
```

- [ ] **Step 4:** `make swagger-validate && make generate-api`; `go build ./...` — компиляция сломается на несогласованных хендлерах, чинится в Tasks 5–6.

- [ ] **Step 5: Commit** `feat(api): city param + /cities endpoint in swagger, regen`

---

### Task 5: хендлер ListCities + настройка cities.spb_available + wiring

**Files:**
- Create: `backend/internal/http/handlers/cities.go`
- Create: `backend/internal/http/handlers/cities_test.go`
- Modify: `backend/internal/settings/settings.go` (константа ключа)
- Modify: `backend/internal/http/module.go` + `backend/internal/application.go` (wiring по образцу существующих хендлеров)
- Modify: `backend/internal/http/handlers/events.go`, `venues.go` (city из params)

**Interfaces:**
- Consumes: `settings.Service.Bool(ctx, settings.KeyCitySPBAvailable)`; `models.Cities`; codegen `citiesops.ListCitiesParams` (Task 4); events `Service.List(..., city)` (Task 3).
- Produces: `GET /api/v1/cities` → `[{"code":"msk","available":true},{"code":"spb","available":false}]`.

- [ ] **Step 1:** в settings.go: `// KeyCitySPBAvailable, when true, opens СПб for discovery (city switcher).` `const KeyCitySPBAvailable = "cities.spb_available"`.

- [ ] **Step 2: тест** (fake settings service): флаг false → spb.available=false; true → true; msk всегда true; ошибка настроек → msk-only fallback (доступность деградирует, 200 не ломаем).

- [ ] **Step 3: реализация** `cities.go`:

```go
// ListCities handler returns known cities and their availability. msk is
// always available; spb is gated by app_settings cities.spb_available so the
// launch is a settings flip, not a deploy.
type ListCities struct{ settings settingsdomain.Service }

func NewListCities(svc settingsdomain.Service) *ListCities { return &ListCities{settings: svc} }

func (h *ListCities) Handle(params citiesops.ListCitiesParams) middleware.Responder {
	spb := false
	if h.settings != nil {
		if v, err := h.settings.Bool(params.HTTPRequest.Context(), settings.KeyCitySPBAvailable); err == nil {
			spb = v
		} else {
			logger.Log().Errorf("read cities.spb_available: %s", err.Error())
		}
	}
	avail := map[string]bool{models.CityMSK: true, models.CitySPB: spb}
	payload := make([]*apimodels.City, 0, len(models.Cities))
	for _, c := range models.Cities {
		code, a := c, avail[c]
		payload = append(payload, &apimodels.City{Code: &code, Available: &a})
	}
	return citiesops.NewListCitiesOK().WithPayload(payload)
}
```

(точные имена citiesops/apimodels сверить с сгенерённым кодом; wiring — по образцу ListCategories в module.go/application.go.)

- [ ] **Step 4: city в существующих хендлерах.** `events.go` Handle: `city := models.DefaultCity; if params.City != nil && *params.City != "" { city = *params.City }` → передать в `h.events.List(...)`. `venues.go` аналогично → в `Search` (Task 6).

- [ ] **Step 5: `go build ./... && go test ./...` зелёные, Commit** `feat(http): /cities endpoint + city param wired into events/venues listing`

---

### Task 6: venue-поиск по городу + bbox для Places

**Files:**
- Modify: `backend/internal/venues/repository.go` (+ `Search(city, q, limit)`), `service.go`
- Modify: `backend/internal/venues/repository_test.go`, `service_test.go`
- Modify: `backend/internal/http/handlers/venues.go`
- Modify: geocode/places прокси (`backend/internal/geocode/` — найти место сборки Places-запроса) — bbox по городу

**Interfaces:**
- Consumes: `models.CityMSK/CitySPB` (Task 2).
- Produces: `Repository.Search(city, q string, limit int)` — city обязателен, фильтр `Where("city = ?", city)`; bbox-константы `models.CityBBox(city) (ll, spn string)` рядом с городскими константами.

- [ ] **Step 1: тесты** — Search всегда добавляет city WHERE; Places-запрос для spb несёт питерский bbox.
- [ ] **Step 2: реализация.** В `Search` первой строкой после `query := r.db.Model(&list)`: `query = query.Where("city = ?", city)`. bbox: msk `ll=37.618,55.751&spn=0.9,0.6`, spb `ll=30.315,59.939&spn=0.9,0.5` (формат под фактический Places-клиент — сверить на месте).
- [ ] **Step 3: тесты зелёные, Commit** `feat(venues): city-scoped search + city bbox for Places suggest`

---

### Task 7: frontend — city как состояние (cookie + ?city= + контекст)

**Files:**
- Modify: `frontend/lib/city.ts` (slug, поиск по slug, чтение cookie)
- Create: `frontend/lib/city-context.tsx`
- Modify: `frontend/lib/api.ts` (city в fetchPublishedEvents/searchVenues + fetchCities)
- Modify: `frontend/components/ui/CityControl.tsx`
- Modify: `frontend/app/layout.tsx`, `frontend/app/page.tsx`, `frontend/app/map/page.tsx`, `frontend/app/login/page.tsx`
- Modify: `frontend/components/DiscoveryFeed.tsx`, `frontend/components/MapBrowse.tsx`, `frontend/components/VenuePicker.tsx`, `frontend/components/CreateEventForm.tsx`
- Tests: `frontend/lib/__tests__/city.test.ts`, `frontend/components/__tests__/city-control.test.tsx`

**Interfaces:**
- Consumes: `GET /api/v1/cities` (Task 5), `?city=` (Tasks 4–6).
- Produces: `city.ts`: `City.slug` ("msk"/"spb"), `cityBySlug(slug: string | undefined): City` (fallback msk), `CITY_COOKIE = "lia_city"`; `city-context.tsx`: `CityProvider({city, availability, children})`, `useCity(): {city: City, available: (slug: string) => boolean, setCity: (slug: string) => void}` (setCity пишет cookie на год + `router.refresh()`); `api.ts`: `fetchPublishedEvents(from?, to?, citySlug?)` (append `city=`), `searchVenues(q, limit, citySlug)`, `fetchCities(): Promise<{code: string; available: boolean}[]>`.

- [ ] **Step 1: тесты.** `city.test.ts`: `cityBySlug("spb").code === "СПБ"`, мусор/undefined → msk. `city-control.test.tsx`: доступный город кликабелен и зовёт setCity, недоступный — disabled со «Скоро»; признак берётся из контекста, не из константы.
- [ ] **Step 2: реализация.** В `CITIES` добавить `slug: "msk" | "spb"`; `available` из объекта убрать НЕ надо (остаётся как SSR-fallback: msk true, spb false — сервер уточняет). Layout: `const cookieCity = cityBySlug(cookies().get(CITY_COOKIE)?.value)`, `const cities = await fetchCities().catch(() => null)` → `<CityProvider city={cookieCity} availability={...}>` вокруг детей; тайтл из `cookieCity.genitive`. `page.tsx`/`map/page.tsx`: `searchParams.city` override (`cityBySlug(searchParams.city ?? cookies()...)`), SSR-фетч с city, маленький клиентский `<CityCookieSync slug=... />` пишет cookie при наличии `?city=`. `DiscoveryFeed`/`MapBrowse`: `const {city} = useCity()` вместо `CURRENT_CITY` (центр карты `city.center`, queryKey включает slug). `CityControl`: `const {city, available, setCity} = useCity()`; клик по доступному → `setCity(slug)`. `VenuePicker`/`CreateEventForm`: селектор города виден при >1 доступном; `searchVenues(q, 20, city.slug)`; POST/PATCH события передаёт `city` только когда площадка не выбрана.
- [ ] **Step 3: `pnpm vitest run` зелёные; ручной smoke `pnpm dev`** — переключатель работает, `?city=spb` даёт пустую ленту с копирайтом Петербурга и центром карты СПб.
- [ ] **Step 4: Commit** `feat(city): live city switcher — cookie + ?city= + server availability`

---

### Task 8: полный прогон + ревью

- [ ] Backend: `go test ./... && golangci-lint run` (v1 — см. gotchas). Frontend: `pnpm vitest run && pnpm lint && pnpm build`.
- [ ] Self-review диффа против спеки; `superpowers:requesting-code-review`.
- [ ] Commit остатков, всё на `main`.

---

### Task 9: деплой шаг 1 (тех-деплой, СПб остаётся «Скоро»)

По схеме lia-demo-deployment / раннбука qa23:

- [ ] scp `000027_city_column.{up,down}.sql` на бокс в `/opt/lia/backend/db/migrations`; прогнать migrate up (Lia DB 26 → 27).
- [ ] Build-on-Mac (linux/amd64, backend static — проверить `file`, тег образов `spb-r1`) → `docker save | ssh vdska2 docker load`.
- [ ] На боксе: tag rollback от РАБОТАЮЩИХ контейнеров (`rollback-spb-20260823`), затем `docker tag <новый> backend-app:latest` (+frontend), `up -d --force-recreate --no-build` со всеми 4 compose-файлами. Frontend build-args: `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` + `NEXT_PUBLIC_YANDEX_MAPS_KEY`.
- [ ] После верификации — прюнинг образов (диск 20 GB).

### Task 10: проверка на проде

- [ ] `curl https://api.presence.tarski.ru/api/v1/cities` → msk true / spb false.
- [ ] `curl '…/api/v1/events?status=published'` без city — счёт совпадает с до-деплоя (московская афиша не изменилась); `?city=spb` → `[]`.
- [ ] Браузером (claude-in-chrome): лента как раньше; в переключателе СПб — «Скоро»; `?city=spb` → пустая лента «Петербурга», карта с центром СПб; создание события и venue-поиск работают (msk).

### Task 11: сид площадок СПб

- [ ] Переиспользовать pipeline из `docs/research` (seed+geocode, batch-схема из lia-venue-catalogue): собрать список ~50–80 ключевых площадок СПб, `city='spb'`, геокодинг Яндексом (лимит 1000/день — ок), вставка в prod БД тем же методом, что batch 2 от 2026-08-01.
- [ ] Проверка: `venues` count по city='spb'; venue-поиск с city=spb находит «Эрмитаж».

### Task 12: открытие (вне этого прогона)

Контент: импорт whitelist-платформ + редакция + организаторы → при ≥15 живых событий: `INSERT INTO app_settings … 'cities.spb_available'` (или админ-ручка) → проверить, что переключатель ожил. Откат — выключить настройку.
