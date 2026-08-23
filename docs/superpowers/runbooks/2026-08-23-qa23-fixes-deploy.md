# QA-23-aug fixes — Deploy + re-QA

_2026-08-23. Backend `lia-backend:qa23-r1` + frontend `lia-frontend:qa23-r1`
(rollback `backend-app:rollback-qa23-20260823`, `lia-frontend-presence:rollback-qa23-20260823`).
No DB migrations. Plan: `docs/superpowers/plans/2026-08-23-qa23-fixes.md`._

## What shipped

Backend:
- Пре-модерация: публикация неверифицированным организатором → `pending_review`
  (`events.SetOrganizerVerifier` ← `organizers.IsVerifiedOwner`); админ-approve публикует.
- Запрет самозаписи владельца: `rsvp.SignUp` → 409 «нельзя записаться на собственное событие».
- `/events/nearby` больше не отдаёт прошедшие (`starts_at >= москва-полночь`).
- Пейлоад: `is_owner` (detail + /events/mine), `pending_applications_count` (/events/mine).
  swagger.yaml + `make generate-api`.

Frontend:
- Owner-панель на опубликованном событии; «Записаться» скрыт у владельца («Это ваше событие»).
- Ошибки логина/регистрации — видимый `role=alert` блок.
- После кода верификации — возврат на исходную страницу (`?next=` из pathname).
- Баннер «почта не подтверждена» не показывается по сбойному `/auth/me`; 401 сносит сессию.
- Подписчики: оптимистичный счётчик + invalidate; «Вы подписаны ✓» читаемый.
- Мобильная шапка: без слипания, email truncate, «Выйти» не режется; чипы с fade-подсказкой скролла.
- «Я»-хаб: секция «Организую» (Кабинет / Мои события / Заявки·N) + «+ Создать событие» + Календарь.
- Честный копирайт публикации (verified → «сразу в ленте», иначе «на модерацию до 24 ч»).
- Статусы: «Вы идёте / Ждёт ответа», фильтры «Опубликованные / На проверке / …», «Действия ▾»,
  бейдж «Заявки · N»; админ-таб «На проверке · N».
- /map: карта получила высоту (mapPane `flex-1`, YandexMap `h-full`) — моб+десктоп.
- Редиректы `/auth/signup → /signup`, `/events/create → /events/new`.

## Procedure

По образцу extreg-раннбука 12.08: build из рабочего дерева (worktree qa23-fixes,
`make generate-api` уже прогнан) → static-check `/lia` → frontend с 2 build-args
(`NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`, `NEXT_PUBLIC_YANDEX_MAPS_KEY`
с работающего контейнера) → save|gzip → rsync → rollback-теги С РАБОТАЮЩИХ
контейнеров ДО load → load → tag latest → compose (4 файла, `--no-build
--force-recreate app`) → frontend stop/rm → run :3002→3001 → QA → prune.

## Verified on production (re-QA, browser + API + psql)

| № | Проверка | Результат |
|---|---|---|
| map | /map моб. 390px и десктоп | **Карта рендерится**: тайлы, 5 пинов; «Всего 5» = согласовано с лентой (прошедшие исключены) |
| 4 | Неверный пароль на /login | Видимый красный alert «Неверный email или пароль» |
| 3 | **`organizers.auto_verify_all` ВЫКЛЮЧЕН через /admin/settings** (был enabled с 26.07 — корень «Проверен у всех») | Новые организаторы больше не авто-verified |
| 3 | Публикация неверифицированным организатором | Диалог «Отправить на модерацию?» → статус `pending_review`, в ленте НЕТ, owner-панель «На модерации» |
| 3 | Админ: таб «На проверке · 1» → Опубликовать | Событие `published`, `published_at` установлен, появилось в публичной ленте |
| 5 | Владелец → POST /events/{id}/rsvp на своё событие | **409** «нельзя записаться на собственное событие»; `is_owner: true` в detail |
| 6 | Черновик/событие владельца | CTA заменён на «Это ваше событие»; owner-панель на месте |
| 7 | Регистрация из модалки события → код → возврат | `/auth/verify?next=/events/…` → после кода вернуло НА СОБЫТИЕ → «Вы записаны» |
| 8 | Вход админа | Ложного баннера «почта не подтверждена» нет |
| 9 | Подписка на организатора | Счётчик 00→01 мгновенно; «Вы подписаны ✓» чёрным по белому, читаемо |
| 10/11 | Моб. шапка + «Я»-хаб | Email truncate, «Выйти» виден; секция «Организую» (Кабинет/Мои события/Заявки) + «+ Создать событие» + «Календарь →» |
| 12 | «Вы идёте» вместо «ОК»; редиректы | Чип «ВЫ ИДЁТЕ» в профиле; `/auth/signup` → 307 → `/signup` |

**Hotfix qa23-r2 (frontend):** первый r1 содержал регрессию — `failMe` сносил
сессию по 401 от `/auth/me`, а бэкенд отвечает 401 сразу после регистрации
(до верификации) → свежезарегистрированного выкидывало посреди ввода кода.
Убран clearSession (баннер-фикс сохранён: roleResolved остаётся false).
Плюс десктопный рейл мастера «После отправки» стал условным.

**Продуктовое изменение конфигурации:** `organizers.auto_verify_all = ВЫКЛ.
Все существующие организаторы остались verified; новые проходят верификацию,
их события до этого публикуются через пре-модерацию (таб «На проверке»).
Организатор `dodonopavel+qa23o` переведён в `pending` для тестов.**

**Gaps, найденные при перетесте (не чинились):**
- В `/admin/organizers/{id}` нет действия «Отозвать верификацию» (транзиция
  есть в бэкенде, UI нет — для теста делали SQL-ом).
- Счётчик таба «На проверке · N» в очереди модерации = 0 до первого клика по табу.
- Деталь pending-события в очереди всегда пишет «ссылка не в белом списке»,
  даже когда причина — неверифицированный организатор.
- `/auth/me` отвечает 401 для неверифицированной сессии — фронт это терпит,
  но из-за этого у свежего пользователя не резолвится роль до верификации.

**Уборка:** QA-retest событие → `rejected`; тестовые rsvps ×3 и follow ×1
удалены (контроль 0/0); в ленте 0 QA-событий. Аккаунты qa23v/qa23o/qa23r
живы (тестовые). Браузер возвращён под админа.

## Rollback

```
ssh vdska2
# frontend
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-qa23-20260823 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
# backend
docker tag backend-app:rollback-qa23-20260823 backend-app:latest
cd /opt/lia/backend
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build --force-recreate app
```
