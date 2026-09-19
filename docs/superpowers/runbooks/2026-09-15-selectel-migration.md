# Переезд прода на Selectel — 2026-09-15

Причина: мобильный ТСПУ режет TLS к старому боксу (см.
`docs/superpowers/plans/2026-08-14-mobile-tls-block-diagnosis.md` §8–10). Тестовый
вход `m.presencehq.ru` через Selectel-VM открылся с телефона стабильно → переносим.

## Где что теперь

| | Старый бокс | Новый сервер |
|---|---|---|
| Хостер | vdska.ru (vds-ru215), `ssh vdska2` | Selectel, **личный** аккаунт 684546 |
| IP | 193.32.188.7 | **135.106.216.153** |
| Размер | 1 vCPU / 2 ГБ / 20 ГБ | Shared 1 vCPU 10% / 1 ГБ + 1 ГБ swap / SSD 10 ГБ, ru-7a, ≈708 ₽/мес |
| Вход | `ssh vdska2` | `ssh -i ~/.ssh/id_ed25519 root@135.106.216.153` (только ключ) |

Потребление на новом: ~340 МБ RAM + ~180 МБ swap при полном стеке (postgres,
app, gateguard+redis, prometheus+node_exporter, один фронт). 1 ГБ хватает, потому
что фронт теперь один (было три: presence/ptarski/phq).

## Что перенесено

- `/opt/lia/backend` (compose ×4, `.env.prod`, миграции, `monitoring/`) и `/opt/lia/geoip` — как было на боксе.
- Образы — ровно те, из которых были запущены контейнеры (тег по image id, не `:latest`):
  `backend-app:latest`, `gateguard:local`, `lia-frontend:phq-r2`; копии под `lia-migr/*:20260915`.
  Публичные (`postgis`, `redis`, `prometheus`, `node-exporter`, `migrate/migrate`) — `docker pull` с VM, Docker Hub доступен.
- БД `lia_prod` + `gateguard`: `pg_dump -Fc` → `pg_restore --clean --if-exists --no-owner`
  (миграция 27, 73 события, 196 площадок, 30 пользователей — сверено до и после).
  Redis gateguard не переносился → сессии могли сброситься, пользователи перелогинятся.
- Том `backend_lia_uploads` (38 файлов) — tar из `/var/lib/docker/volumes/.../_data`.
- `/etc/letsencrypt` целиком (presencehq.ru, presence.tarski.ru, p.tarski.ru) — renewal-конфиги на nginx-аутентификаторе заработают, когда DNS имён смотрит на новый IP.
- Фронт: `docker run -d --name lia-frontend-phq --restart unless-stopped -p 127.0.0.1:3004:3001 --env-file /opt/lia/frontend-phq.env lia-frontend:phq-r2` (env = `NEXT_PUBLIC_*` из старого контейнера).

Запуск стека на новом (как раньше, 4 файла):

```bash
cd /opt/lia/backend && docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.gateguard.yml -f docker-compose.monitoring.yml up -d --no-build
```

## nginx на новом (`/etc/nginx/sites-available/presencehq`)

- `presencehq.ru` — единственный рабочий хост (фронт на `/`, бэкенд на `/api/` и точные `location =` для `/auth/*`, `/complaints`, `/feedback`, `/invitations`).
- `presence.tarski.ru`, `api.presence.tarski.ru`, `p.tarski.ru`, `api.tarski.ru` → **301 на `https://presencehq.ru$request_uri`** (пути бэкенда те же, старые API-ссылки на обложки продолжают работать).
- `m.presencehq.ru` — бывший тестовый вход, теперь тоже 301.
- Реальный IP клиента приходит в `X-Real-IP` напрямую (нет промежуточного прокси) → rate-limit снова per-client.

## Переключение (18:28:25 → 18:28:54 UTC, простой API ~30 с)

1. Бэкап nginx старого бокса → `/etc/nginx/bak-20260915/`.
2. `docker stop backend-app-1 backend-gateguard-1` на старом (контейнеры целы).
3. Финальные дампы → Mac (`final-*-20260915.dump` в scratchpad сессии) → restore на VM; ресинк uploads; `up -d gateguard app` на VM.
4. Старый nginx → `cutover-to-selectel`: `presencehq.ru` **проксируется** на новый сервер (клиенты со старым DNS не теряются), tarski-имена → 301.
5. bind на старом: `@ A` → 135.106.216.153 (TTL заранее снижен до 300), `ns1/ns2` пока на старом IP.

## Ещё не сделано

- [x] **nic.ru, presencehq.ru:** NS `ns1/ns2.presencehq.ru` → `a/b/c/d.ns.selectel.ru` —
  заявка ~19:00 UTC, **реестр переключил 2026-09-15 20:17 UTC** (8.8.8.8 / 77.88.8.8 / 1.1.1.1
  сразу видят NS Selectel). Зона в Selectel: `A @`, `A m` → 135.106.216.153 (TTL 300).
  bind на старом боксе больше не авторитетен; **старый бокс можно гасить после 2026-09-16 20:17 UTC**.
- [x] **nic.ru, tarski.ru** (DNS-master): A `presence`, `api.presence`, `p`, `api` → 135.106.216.153,
  TTL 300, опубликовано; на NS nic.ru с 19:59 UTC. MX/SPF/DMARC/DKIM/`www`/`@` не тронуты.
  Грабли DNS-master: после правки записи список пересортировывается — редактировать через поиск
  по значению (`193.32.188.7`), а не по позиции строки; изменения применяются только после «Опубликовать».
- [x] `certbot renew --dry-run` на новом: presencehq.ru, presence.tarski.ru, p.tarski.ru — OK.
- [ ] `backend/docker-compose.prod.yml` в репо ≠ боевой (боевой — на сервере, от 2026-09-02).
- [ ] Старый бокс не продлевается (решение владельца). **vdska 185.5.75.80 — НЕ этот бокс:** на ней живой AmneziaWG-VPN для роутера, удалять только с явного подтверждения.

## Правила эксплуатации нового сервера (2026-09-16)

1. **Никогда не собирать образы на сервере.** `next build` берёт 1–2 ГБ RAM —
   на 1 ГБ он упадёт. Именно сборка на боксе (`/opt/lia/frontend-build.sh`,
   `vpn-build-all.sh`) и съедала память на старом сервере при каждом деплое.
   Собираем на Mac под `linux/amd64` → `docker save | ssh | docker load`.
2. **После каждого деплоя чистить образы:** `docker image prune -f` + удалять
   старые теги, оставляя последний rollback. Иначе каждый деплой фронта
   добавляет ещё один образ и диск кончается.
3. **Фронт собирается в режиме `standalone`** (`next.config.ts: output: "standalone"`
   + двухстадийный `frontend/Dockerfile`, запуск `node server.js`, не `next start`).
   **Измерено 2026-09-16: 106 МБ вместо 1.06 ГБ**; локально проверено, что контейнер
   стартует, отдаёт `/` и `/_next/static/*`, и что `sharp` внутри есть (иначе next/image
   ломается — он ставится явно в runner-стадии). При правке одного файла синхронно
   править второй. На прод ещё не задеплоено.
4. **Память:** рантайм ~350 МБ + ~220 МБ swap из 1 ГБ (фронт ~100 МБ, postgres ~60,
   backend ~48, prometheus ~43 при лимите 256 МБ, node_exporter ~20, gateguard+redis ~13).
   Prometheus «ест» не память, а диск: образ 386 МБ + TSDB до 512 МБ (`retention.size`).
5. **Диск:** 20 ГБ (расширен 2026-09-16 с 10 ГБ). Больше всего занимают образы
   (`/usr` ~1.3 ГБ — система). Проверка: `df -h /` и `docker system df`.

## Откат (пока старый бокс жив)

`docker start backend-app-1 backend-gateguard-1` на старом, вернуть `sites-enabled` из
`/etc/nginx/bak-20260915/`, bind из `db.presencehq.ru.bak-20260915-cutover` + `rndc reload`.
Данные, записанные на новом сервере после 18:28 UTC, при этом нужно перенести обратно дампом.
