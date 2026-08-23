# СПб: запуск второго города — Deploy

_2026-08-23. Backend `lia-backend:spb-r2` + frontend `lia-frontend:spb-r2`
(rollback `backend-app:rollback-spb-20260823`, `lia-frontend-presence:rollback-spb-20260823`).
Lia DB 26 → **27**. Spec `docs/superpowers/specs/2026-08-23-spb-city-launch-design.md`,
plan `docs/superpowers/plans/2026-08-23-spb-city-launch.md`, `main` = `87cd7e4` (+фиксы ревью)._

## What shipped

Backend:
- Миграция 000027: `city text NOT NULL DEFAULT 'msk'` на `venues` и `events`;
  индексы `venue_city_name_lower_idx (city, lower(name))` (взамен старого) и
  `events_status_city_starts_at_idx (status, city, starts_at)` (взамен
  `event_status_starts_at_idx` из 000004). **Коррекция backfill: площадки в
  координатной рамке СПб (59.5–60.3 / 29.3–31.0) → `city='spb'`** — сиды
  batch 1–2 уже содержали 75 питерских площадок; 2 старых события при них
  тоже стали spb.
- Слаги `msk`/`spb` (`models.Cities`); `?city=` на listEvents (дефолт msk;
  с `organizer_id` без city — вся страница организатора), listVenues (city
  обязателен на уровне сервиса, дефолт msk); наследование города события от
  площадки (конфликт → 400; снятие площадки СОХРАНЯЕТ город); venue
  find-or-create скоупится городом (одноимённые площадки в двух городах —
  разные записи).
- `GET /api/v1/cities` → `[{code, available}]`; `spb.available` =
  app_settings **`cities.spb_available`** (settings.Service, как
  auto_verify_all). Включение/выключение — PUT `/api/v1/admin/settings`
  `{"key":"cities.spb_available","enabled":true|false}` под админом, без
  редеплоя.
- Геокодер/Places: viewport bias по городу (`?city=` на /geocode и /places).

Frontend:
- Cookie `lia_city` (год) + `?city=` override на `/`, `/map`, `/search`
  (CityCookieSync персистит override). CityProvider/useCity — доступность с
  сервера; CityControl реально переключает (недоступные — «Скоро»).
- Лента/Подбор/карта (центр!)/логин/тайтлы следуют городу; VenuePicker и
  форма события city-скоупные; чип-выбор города в форме виден при >1
  открытом городе; city шлётся только для событий без площадки; edit-форма
  сидит город ИЗ события (не из cookie посетителя).

## Procedure (as executed)

Стандартная схема (extreg/qa23): build amd64 на маке → static-check `/lia`
(statically linked) → frontend с 2 build-args → save|gzip → rsync → scp
000027\*.sql в `/opt/lia/backend/db/migrations` → pg_dump бэкап
(`/opt/lia/backup-pre-spb-lia-20260823-073546.sql.gz`) → rollback-теги с
РАБОТАЮЩИХ контейнеров → load → `docker tag lia-backend:spb-r2
backend-app:latest` → compose 4 файла `up -d --no-build --force-recreate app`
(migrate 27/u прошёл автоматически, not dirty) → frontend stop/rm → tag →
run :3002→3001 → QA → prune (диск 68%).

Code review (medium, 9 верификаций) до деплоя: 8 подтверждённых находок,
все исправлены — главные: `pgRepository.Update` не писал колонку `city`
(терялась при PATCH); /search игнорировал город; edit-форма могла молча
«переселить» событие в город посетителя; geocode bias не доезжал с фронта.

## Verified on production

| Проверка | Результат |
|---|---|
| `schema_migrations` | 27, not dirty; venues: msk 99 / spb 75; events: msk 38 / spb 2 |
| Московская афиша | 8 published до = 8 после; лента в браузере без изменений |
| `GET /cities` | msk true; spb false → **true после включения настройки** (без рестарта) |
| `?city=ekb` | 422 (enum go-swagger) |
| Venue search «Эрмитаж» | city=spb → находится; дефолт (msk) → пусто |
| Переключатель в шапке | СПб был «Скоро» при выключенной настройке; после включения кликается, СПб↔МСК в обе стороны, копирайт/счётчик меняются |
| `/map?city=spb` | Карта центрируется на Санкт-Петербурге |
| `?city=spb` линк | Открывает ленту СПб и персистит cookie |

## Состояние после деплоя

**`cities.spb_available` = ВКЛЮЧЕНО** (по прямой просьбе Павла 23.08,
раньше контентного критерия «≥15 живых событий» из спеки). Лента СПб пока
пустая: 75 площадок есть, живых событий нет — контентный трек (импорт
whitelist-платформ / редакция / организаторы) впереди. Выключить обратно:
тот же PUT с `enabled:false`.

## Rollback

```bash
ssh vdska2
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-spb-20260823 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
docker tag backend-app:rollback-spb-20260823 backend-app:latest
cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build --force-recreate app
# при необходимости: migrate down 1 (000027.down.sql на боксе) — вернёт старый индекс, дропнет city
```

## Gaps / follow-ups

- Гео-IP определение города нового посетителя — НЕ реализовано (нужен выбор
  провайдера: MaxMind GeoLite2 (оффлайн-БД, нужен аккаунт) vs внешний API).
  Сохранение выбора уже работает (cookie на год).
- На `?city=` ссылке шапка мигает старым городом до пост-гидрационного
  refresh (косметика, самолечится).
- Выбор города пока не хранится в профиле (только cookie) — кросс-девайс нет.
- go-swagger сгенерированный код НЕ в гите: перед сборкой из свежего чекаута
  обязателен `make generate-api`.
