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

## Verified on production

(заполняется по итогам re-QA)

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
