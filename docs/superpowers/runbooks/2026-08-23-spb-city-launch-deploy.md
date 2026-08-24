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
раньше контентного критерия «≥15 живых событий» из спеки). Выключить обратно:
тот же PUT с `enabled:false`. (24.08 афиша наполнена — см. секции «Контент».)

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

## Hotfix spb-r3 (frontend)

QA Павла после открытия: в ДЕСКТОПНОЙ шапке переключателя города не было
вовсе — `CityControl` жил только в мобильном блоке (`mobileCaption`,
`sm:hidden`). Фикс: AppHeader рендерит `CityControl` в десктопной навигации
на каждой не-админской странице (`72abc91`). Задеплоен `lia-frontend:spb-r3`
(тот же rollback-тег), переключение на десктопе проверено live в обе стороны.

## Geo-IP (spb-r4 backend / spb-r5 frontend, deployed same day)

MaxMind GeoLite2-City (выбор Павла). `internal/geoip`: оффлайн-резолвер,
msk ← MOW/MOS, spb ← SPE/LEN (область → её метрополия), не-RU/прочие → "".
База НЕ в образе: `/opt/lia/geoip/GeoLite2-City.mmdb` (63M, зеркало
P3TERX/GeoLite.mmdb; обновление = замена файла + recreate app), bind-mount
`/opt/lia/geoip:/geoip:ro` + env `GEOIP_MMDB_PATH` в docker-compose.prod.yml;
нет файла → фича тихо выключена. `GET /cities` строки несут `suggested`
(IP из X-Real-IP). Фронт: `CityGeoDefault` при первом заходе БЕЗ cookie
спрашивает /cities ИЗ БРАУЗЕРА (SSR дал бы IP бокса), пишет cookie (включая
msk-fallback — geo-lookup один раз на посетителя), refresh только если
подсказка ≠ дефолту. Проверено на проде: X-Real-IP 31.134.191.1 → spb
suggested; 95.220.0.1 → msk; 8.8.8.8 → ничего; первый заход пишет cookie.
Атрибуция: «This product includes GeoLite2 data created by MaxMind».

## Контент СПб (2026-08-24)

Афиша наполнена: **17 реальных событий** с внешней регистрацией через
KudaGo (whitelist) — 3 бесплатные лекции (библиотеки Лермонтова / Охта-8),
8 выставок (Эрмитаж «Императорский Китай», Русский музей «Шишкин» и
Михайловский замок «Великая», Эрарта, Севкабель ×2 («Цой», «Траектории
интервалов»), Анненкирхе, усадьба Державина), 3 концерта (Анненкирхе,
Петрикирхе, планетарий), спектакль (Александринский), 2 прогулки
(Belle Époque в Эрмитаже, «730 шагов с Раскольниковым»). Данные — открытое
API KudaGo (kudago.com/public-api/v1.4, location=spb); +8 новых площадок с
координатами (venues 182). Обложки догружены позже (см. «Обложки»).

Организатор: профиль админа переименован в **«Редакция PRESENCE»**,
verified, daily-limit 0 (без дневного капа). EVENTS_MONTHLY_LIMIT на время
загрузки поднимался 10→100 на боксе и ВОЗВРАЩЁН к 10 (+recreate).
Ongoing-выставкам ставится starts_at = ближайший день 12:00 и ends_at =
дата закрытия (прошлое starts_at не проходит фильтр from= ленты).

## Контент МСК + снятие демо (2026-08-24)

Той же KudaGo-механикой наполнена Москва: **15 реальных событий** от
«Редакции PRESENCE» — выставки (Третьяковка ×2: Борисов-Мусатов и «Иконы
Третьякова», МАММ «Искусство будущего», Щусева «АЗС», Дали и Пикассо в
усадьбе Голицыных, «Королевский Копенгаген» в Кускове, Босх в Люмьер-Холле),
кино (Московская неделя кино в «ТАУ», «Докер» в «Октябре»), фестиваль садов
в Царицыне, «Чайка» в Вахтангова, лекции (ТОК в ГЭС-2, Котельническая
высотка), концерты (Миядзаки, «Джаз на воде»). +13 новых площадок с
координатами.

**8 демо-событий сидов (`b0000000-…` + af51a5be) сняты** через админский
takedown (status `rejected`, reason «Демо-контент…»; обратимо reinstate'ом).
EVENTS_MONTHLY_LIMIT снова поднимался 10→100→10 (+recreate ×2).

## Обложки (2026-08-24, вечер)

Все **32 события** (17 СПб + 15 МСК) получили реальные обложки из KudaGo
(events API, поле `images`). Механика: скачаны шеллом → прикреплены в
edit-форму каждого события через `file_upload` расширения (input «Обложка»;
форма сама грузит на /uploads и autosave'ом патчит `cover_file_id`).
Прямой путь «браузер ← localhost CORS-сервер» не работает: Chrome Private
Network Access блокирует fetch с публичного https на localhost.
Попутно исправлены **10 битых external_registration_url** московских
событий (при создании были вписаны домышленные KudaGo-slug'и; настоящие
взяты из API). Проверено: 32/32 cover_url в API, лента с картинками.

## Gaps / follow-ups

- GeoLite2-база обновляется вручную (замена файла); можно добавить cron.
  Для официальных обновлений завести аккаунт MaxMind (сейчас — зеркало).
- Разовые события (лекции/концерты) пройдут к началу сентября — афишу СПб
  надо пополнять (повторить KudaGo-подборку или автоматизировать импорт).
- На `?city=` ссылке шапка мигает старым городом до пост-гидрационного
  refresh (косметика, самолечится).
- Выбор города пока не хранится в профиле (только cookie) — кросс-девайс нет.
- go-swagger сгенерированный код НЕ в гите: перед сборкой из свежего чекаута
  обязателен `make generate-api`.
