# Clickable dashboard tiles — Deploy + QA (as executed)

_2026-08-12, second deploy of the day. Frontend-only (`lia-frontend:tiles-r1`,
rollback `lia-frontend-presence:rollback-tiles-20260812`); backend untouched.
`main` `2dafeed` pushed before deploy. Spec
`docs/superpowers/specs/2026-08-12-clickable-dashboard-tiles-design.md`._

## What shipped

Язык аффордансов «стрелка в капшене → плитка кликабельна» + hover-инверсия:
`Cell` c `href`; `?status=` URL-фильтр в «Мои события»; проводка плиток
кабинета организатора, админ-обзора (плитки, мобильные, превью-строки) и /me.
Инертные без стрелки: «Всего записей», «Событий всего».

## Procedure

Standard frontend swap: build (2 build-args) → grep key+URL in bundle →
save|gzip → rsync --partial → rollback tag from running container →
load → stop/rm → tag latest → docker run :3002→3001. Tarball removed, prune.

## Verified on production (browser, live session)

| Проверка | Результат |
|---|---|
| /organizer | «Опубликовано →» «На модерации →» «Черновики →», «Всего записей» без стрелки |
| Клик «Черновики →» | /events/mine?status=draft, чип «Черновики·1» активен, черновик в списке |
| /me | «Посещено →» «Подписки →»; клик → /me?tab=past, чип переключился same-route без remount |
| /admin | «Событий всего» инертна; «Ждут модерации →» hover: бумага + красный текст (каскад cap/group-hover работает); «Организаторов →» «Пользователей →» |
| Клик «Организаторов →» | /admin/organizers открылся |

## Rollback

```
ssh vdska2
docker rm -f lia-frontend-presence
docker tag lia-frontend-presence:rollback-tiles-20260812 lia-frontend-presence:latest
docker run -d --restart unless-stopped --name lia-frontend-presence -p 127.0.0.1:3002:3001 lia-frontend-presence:latest
```
