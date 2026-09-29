# Раннбук — ребрендинг в «Сообща» и переезд на soobshcha.ru

**Дата:** 2026-09-29
**Прод:** https://soobshcha.ru — Selectel VM `135.106.216.153` (аккаунт 684546)
**Код:** `origin/main` `984e615` (домен в `BRAND`), `bd1a0be` (хост обложек в `next.config.ts`)
**Образы:** `lia-frontend:soobshcha-r2`, `backend-app:soobshcha-r1`, `gateguard:soobshcha-r1`
**Откат:** `*:rollback-soobshcha-20260929` (до ребрендинга), `lia-frontend:rollback-domain-20260929`
(ребрендинг на presencehq.ru, до переезда домена)
**Бэкапы конфигов на боксе:** `docker-compose.prod.yml.bak-20260929`, `.env.prod.bak-20260929`,
`/opt/lia/frontend-phq.env.bak-20260929`, `/etc/nginx/sites-available/presencehq` (старый конфиг, не включён)

## Что сделано

1. **Ребрендинг (16:25 UTC).** Фронт, бэкенд (тема письма-приглашения), GateGuard (тема и подпись
   письма с кодом) — образы `soobshcha-r1`. Миграций нет. Организатор «Редакция PRESENCE» →
   «Редакция Сообща» (`organizers.id = b9f7a9aa-…`), UPDATE в транзакции.
2. **Домен.** `soobshcha.ru` куплен на nic.ru (договор 5753153/NIC-D) 2026-09-29, оплачен до
   2027-09-29. Из заказа убраны все навязанные услуги (страховка ТЗ, «повышенная безопасность»,
   SSL, почта, менеджер, антивирус). `soobscha.ru` из хэндоффа занят третьим лицом.
3. **DNS.** Зона `soobshcha.ru` в Selectel (аккаунт **684546**, не рабочий 640845):
   `A @` и `A www` → 135.106.216.153, TTL 300. NS у регистратора → `a–d.ns.selectel.ru`.
4. **TLS.** `certbot certonly --webroot -w /var/www/html -d soobshcha.ru -d www.soobshcha.ru
   --account 6407368ebe6898a2299f96fffbaf1b4b` (на боксе два ACME-аккаунта — без `--account`
   certbot спрашивает интерактивно). Продление: dry-run OK.
5. **Бэкенд** (`docker-compose.prod.yml` на боксе): `HTTP_CORS_ALLOWED_ORIGINS` += soobshcha.ru,
   www.soobshcha.ru; `STORAGE_PUBLIC_BASE` и `PUBLIC_BASE_URL` → soobshcha.ru; `.env.prod`
   `PUBLIC_BASE_URL` тоже. `up -d --no-build --force-recreate app`.
6. **Фронт:** собран с `NEXT_PUBLIC_API_URL=https://soobshcha.ru` (+ ключ карт из
   `/opt/lia/frontend-phq.env`), env-файл обновлён.
7. **nginx:** `sites-enabled/soobshcha` — soobshcha.ru основной; www → 301; presencehq.ru
   проксирует только `/api/` (старые ссылки), остальное 301 на soobshcha.ru; tarski-имена → 301
   на soobshcha.ru; `m-presencehq` оставлен (301 на presencehq → ещё один 301).

## Проверка

- `https://soobshcha.ru/` → «Сообща — События», API 200, иконка 200, сертификат до 2026-12-28.
- 301: www, presencehq.ru (кроме `/api/`), presence.tarski.ru.
- CORS: `Access-Control-Allow-Origin: https://soobshcha.ru`.
- Обложки: ссылки строятся из `STORAGE_PUBLIC_BASE`, все отдаются через `/_next/image`.

## Грабли этого захода

1. **DNS с Mac врёт.** VPN перехватывает :53 и отдаёт закэшированную заглушку nic.ru
   (178.210.92.188, TTL не тот, без authority). Проверять DNS только с бокса.
2. **`dig +short NS … @a.dns.ripn.net` пуст даже после делегирования** — NS приходят в
   authority-секции referral. Проверять публичные резолверы с бокса (8.8.8.8, 77.88.8.8).
3. **Карта с Mac «не загружается»:** Яндекс отвечает `limited` на VPN-IP. С российского IP ключ
   работает для soobshcha.ru — ограничения по домену нет.
4. **Одна обложка дала 500 в `/_next/image`** в момент перезагрузки nginx (TLS handshake failure
   при фетче изнутри контейнера) — повтор 200, кэш оптимизатора перезаполнился сам.
5. **Автоклассификатор** один раз заблокировал `docker load` на прод; прошло после явного
   разрешения пользователя в чате.

## Осталось

- **DNS-master nic.ru для tarski.ru оплачен до 2026-10-01**, автопродление включено, но на
  балансе 264,8 ₽ при цене 1 017 ₽ — нужно пополнить, иначе ляжет DNS tarski.ru (почта info@tarski.ru).
- **Баланс Selectel ≈ 14 дней** (377 ₽ на 2026-09-29).
- Имя отправителя «Сообща» в письмах — нужен код (FROM используется и как envelope MAIL FROM).
- `backend/docker-compose.prod.yml` в репо по-прежнему не совпадает с боевым.
- Проверить заставку и карту с телефона в РФ.
