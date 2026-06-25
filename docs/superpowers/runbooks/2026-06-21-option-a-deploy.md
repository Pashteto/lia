# Option A deploy runbook (oracle-1)

Host: oracle-1 (129.146.183.89). Hand-managed; **do NOT touch** dollbuilder
(`:8080`) or any other existing vhost/container. Everything below is `lia-*` and
additive. Run each step on the box; report the `Verify` output back.

`<USER>` / `<PASS>` / `<DB>` below are the NON-DEFAULT values you put in
`backend/.env.prod` — never `dev/dev`.

## 1. DNS (Namecheap BasicDNS)

Add an A record:  `api.lia` → `129.146.183.89`

Verify (after propagation): `dig +short api.lia.pashteto.com` → `129.146.183.89`

## 2. Sync repo + create .env.prod (NOT in git)

rsync the repo's `backend/` to the box (same path convention as the existing
frontend deploy). On the box, in `backend/`:

```bash
cp .env.prod.example .env.prod
# edit .env.prod: DATABASE_USER / DATABASE_PASSWORD / DATABASE_NAME = non-default
chmod 600 .env.prod
```

## 3. Bring up the backend stack

```bash
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Verify Postgres is NOT reachable from the host and the app is loopback-only:

```bash
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml ps
ss -tlnp | grep 9080      # expect 127.0.0.1:9080 only
ss -tlnp | grep 5432      # MUST be empty (Postgres compose-internal)
curl -s http://127.0.0.1:9080/api/v1/health
```

## 4. Seed (once, after healthy)

```bash
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
  exec -T postgres psql -U "<USER>" -d "<DB>" < db/seed/seed.sql
```

Verify counts + a seed-unique title (proves real data, not mock fallback):

```bash
curl -s "http://127.0.0.1:9080/api/v1/events?status=published" \
  | grep -c "Летний фестиваль медиаискусства"     # expect 1
curl -s "http://127.0.0.1:9080/api/v1/events/nearby?lat=55.75&lon=37.62&limit=50" \
  | head -c 400                                    # expect distance_m, nearest-first
```

## 5. nginx vhost + TLS for api.lia.pashteto.com

Model on the existing `lia.pashteto.com` vhost; only the upstream port differs
(`proxy_pass http://127.0.0.1:9080;`). Then:

```bash
nginx -t && systemctl reload nginx
certbot --nginx -d api.lia.pashteto.com
```

## 6. Rebuild + restart the frontend pointed at the real API

From `frontend/` on the box:

```bash
docker build --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend .
docker rm -f lia-frontend
docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend
```

## 7. End-to-end verification (see plan Task 5)

```bash
curl -s "https://api.lia.pashteto.com/api/v1/events?status=published" \
  | grep -c "Летний фестиваль медиаискусства"     # expect 1
```

Then open `https://lia.pashteto.com`, an event detail page, and `/map` — confirm a
seed-unique title renders. Confirm dollbuilder + other vhosts still serve (200).

## Notes

- `HTTP_MOCK_AUTH=true` stays on for Option A — a **known non-production control**;
  call it out at review. Real auth is the next slice (HANDOFF "What's next").
- oracle-1 remains a documented hand-managed demo exception to the
  Terraform/Secrets-Manager standard — demo scope only, not a production precedent.
- The seed is idempotent (`ON CONFLICT DO NOTHING`); re-running it is safe and the
  named volume `lia_pgdata` persists data across restarts.
