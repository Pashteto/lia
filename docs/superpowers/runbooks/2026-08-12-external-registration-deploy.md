# External-registration whitelist — Deploy + QA (as executed)

_2026-08-12. Target: `presence.tarski.ru` / `api.presence.tarski.ru` on
vds-ru215 (`ssh vdska2`). `main` `0edaa5f` (16 feature commits), pushed to
origin before deploy. Images `lia-backend:extreg-r1` + `lia-frontend:extreg-r1`.
Prod schema 25 → **26**._

## What shipped

Спека `docs/superpowers/specs/2026-08-12-external-registration-design.md`:

1. Таблица `trusted_platforms` (миграция 026) + seed 22 домена; suffix-матчинг
   в punycode.
2. Публикация external-события: известный домен → live сразу; неизвестный →
   `pending_review` + `moderation_required`; 422 только за не-https URL.
   Смена URL в любом статусе сбрасывает `external_url_verified`.
3. `moderation.Approve` + `POST /api/v1/admin/moderation/events/{id}/approve`.
   Владелец может править pending_review-событие (re-check при публикации).
4. Публичный `GET /api/v1/trusted-platforms`; admin CRUD
   `/api/v1/admin/trusted-platforms` (деактивация, не удаление).
5. UI: «Билеты на {платформа}», «Оплата на месте», «Места: Ограничено ·
   наличие на сайте регистрации», живая подсказка домена в форме, админ-таб
   «Ссылки», управление whitelist на /admin/settings; CTA-колонка 200→248px.

## Procedure (as executed)

1. Prune on box FIRST (builder+dangling; disk 72%). `pg_dump` обеих БД →
   `/opt/lia/backup-pre-extreg-{lia,gateguard}-20260812-084849.sql.gz`.
   **GateGuard DB называется `gateguard`** (не `presto`).
2. Backend build прямо из рабочего дерева (генерированный swagger-код в нём
   актуален после `make generate-api` в ветке): `docker build --platform
   linux/amd64 -t lia-backend:extreg-r1 backend/`. Проверка статической
   линковки (обязательная): `docker create` + `file /lia` → statically linked.
3. Frontend: оба build-args (`NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`,
   `NEXT_PUBLIC_YANDEX_MAPS_KEY` — берётся `docker inspect lia-frontend-presence
   … | grep -i yandex` с работающего контейнера). Проверка: grep ключа и URL в
   `/app/.next` бандле образа.
4. `docker save | gzip` → `rsync --partial` (10M backend + 215M frontend).
   Миграция 026 up+down → rsync в `/opt/lia/backend/db/migrations/`.
5. На боксе: rollback-теги `*:rollback-extreg-20260812` С РАБОТАЮЩИХ
   контейнеров ДО load; `docker load`; `docker tag lia-backend:extreg-r1
   backend-app:latest`.
6. Backend cutover: compose (4 файла, `--no-build --force-recreate app`) —
   `backend-migrate-1` применил `26/u trusted_platforms` автоматически;
   `schema_migrations` = 26, not dirty; seed = 22 rows.
7. Frontend swap: stop/rm → tag latest → `docker run -d --restart
   unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001`.
8. QA (ниже), затем удаление тарболов + prune. Диск 73%.

## Verified on production

| Проверка | Результат |
|---|---|
| `GET /api/v1/trusted-platforms` (anon) | 200, 22 платформы |
| Событие с `https://qa-probe.timepad.ru/…` (fixed 700₽, capacity_limited) | `published` сразу; GET отдаёт `external_platform_name: TimePad`, `capacity_limited: true` |
| Событие с неизвестным доменом | `pending_review` + `moderation_required: true`; anon GET → 404; в фиде отсутствует |
| Админ-очередь `?status=pending_review` | 200, строка содержит `external_registration_url` |
| Approve (anon / non-admin / admin) | 401 / 403 / **204** → событие `published`, появилось в фиде |
| Admin add platform `qa-probe-платформа.рф` | 201, сохранён как `xn--qa-probe--8yha4koa1amt6a0b.xn--p1ai` (punycode-нормализация живая) |
| Admin deactivate | 204; публичный список снова 22 |
| Страница external-события (браузер) | «БИЛЕТЫ НА TIMEPAD» + подпись `qa-probe.timepad.ru`, «Места: Ограничено · наличие на сайте регистрации», CTA-колонка 248px — ничего не пересекает линии |
| Страница платного open-события (браузер) | «ОПЛАТА НА МЕСТЕ» под ценой, обычная «Записаться» |
| Daily-limit 3/день | 4-е событие → 429 (обход для QA через `organizers.daily_event_limit`) |

**Не проверено в браузере:** админ-вкладка «Ссылки» и /admin/settings-секция
whitelist (вход в админку требует пароля; API-слой обеих проверен полностью).
Ручная проверка: зайти админом → /admin/moderation/events → чип «Ссылки».

## QA-данные

Ретроактивный бэкфилл не понадобился: external-событий в проде было 0, новые
колонки — `NOT NULL DEFAULT false`. Тестовые артефакты (аккаунт
`qa-extreg-20260812@tarski.ru` с временной ролью admin, 4 события, платформа
QA Probe) созданы, проверены и **полностью удалены** из обеих БД.

Gotchas этого прогона:
- `POST /api/v1/auth/register` работает на проде (не завис на SMTP, в отличие
  от локального смоука); `price_type` — `free|fixed|from`, не `paid`.
- Роль пользователя живёт в **GateGuard** `users.role` и синкается в Lia при
  каждом запросе — менять роль надо в `gateguard`, не в `lia_prod`.
- Создание события требует `email_verified` (флаг тоже в GateGuard).

## Rollback

```
ssh vdska2
# frontend
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-extreg-20260812 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
# backend
cd /opt/lia/backend
docker tag backend-app:rollback-extreg-20260812 backend-app:latest
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build --force-recreate app
```

Старый бэкенд работает со схемой 26 (явные списки колонок). Если надо снять
миграцию — `000026_trusted_platforms.down.sql` на боксе.
