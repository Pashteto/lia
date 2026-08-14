# QA 14-aug mobile fixes — Deploy + QA

_2026-08-14. Frontend-only (`lia-frontend:qa14aug-r1`, rollback
`lia-frontend-presence:rollback-qa14aug-20260814`); backend untouched.
`main` `123717d` pushed before deploy (two commits: `36bd9bc` UX fixes,
`123717d` city single-source refactor)._

## What shipped

Пакет фиксов по QA-прогону всех ролей 14.08 (мобильная версия):

- Шапка ленты: «МСК ↓» — переключатель города (Москва ✓ / СПб «Скоро», disabled), `ui/CityControl.tsx`
- `lib/city.ts` — единый источник города: копирайт ленты/логина/metadata, центр /map, список городов
- Страница черновика/на-модерации: owner-панель «Опубликовать / Редактировать / Мои события» (`OwnerEventActions.tsx`)
- «Мои события», мобайл: «···» → «Действия ···»
- `YandexMap`: плейсхолдер «Загружаем карту…» до инициализации ymaps
- Админ-очередь модерации работает <900px (гейт снят только с очереди; users/organizers остались desktop-only)
- Мастер события, шаг 2: «Указать на карте» — bordered-кнопка
- 404: нейтральная копия; календарь: легенда «■ есть события / □ выбранный день»; кабинет организатора: «← Лента» в шапке
- lint: MapBrowse initial load отложен на микротаск

## Procedure

Standard frontend swap: build (2 build-args: `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`,
`NEXT_PUBLIC_YANDEX_MAPS_KEY` — берётся из env работающего контейнера через
`docker inspect lia-frontend-presence`) → grep key+URL в бандле → save|gzip →
rsync --partial → rollback-тег с работающего контейнера → load → stop/rm →
tag latest → docker run :3002→3001. Tarball удалить, prune.

## Verified on production (browser, live session, mobile viewport)

| Проверка | Результат |
|---|---|
| Шапка ленты | «МСК ↓» — кнопка; клик открывает листбокс «Москва ✓ / Санкт-Петербург — Скоро (disabled)» |
| Черновик события | Owner-панель «Черновик — видно только вам» + Опубликовать / Редактировать / Мои события прямо на странице |
| /organizer | «← Лента» в шапке рядом с «Кабинет» |
| /admin/moderation/events <900px | Очередь работает: одна колонка, список (ограничен по высоте) → карточка → причины → Одобрить/Отклонить/На доработку |
| Контейнер | `lia-frontend-presence:latest` Up, `curl 127.0.0.1:3002` → 200 |

Диск после prune + trim старых образов (amd64-*, walk, mockfb, dr11, email-r1,
verif-r2 и дубли тегов): 76% → **67%** (6.3G свободно). Rollback-тегов фронта
осталось три: qa14aug-20260814, tiles-20260812, extreg-20260812.

## Rollback

```
ssh vdska2
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-qa14aug-20260814 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
```
