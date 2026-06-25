# Runbook — deploy GateGuard (auth) on vds-ru215, reuse Lia's Postgres

Target: **vds-ru215** `193.32.188.7` (`ssh vdska2`). Self-host gateway.fm's
**GateGuard** (gRPC token store) so Lia can do real auth (Phase B of the auth
slice; spec `2026-06-25-auth-gatekeeper-design.md`). Login UX = **demo-login**
(no Google) in Phase C.

Decision: GateGuard reuses Lia's existing Postgres container (separate
`gateguard` DB) + a small Redis, on Lia's compose network. `HTTP_MOCK_AUTH`
stays **true** until Phase C — flipping it before the frontend sends tokens would
401 every `POST /events`.

> Box is flaky (1.9 GB RAM). Run the image build detached + poll. See
> [[lia-demo-deployment]].

## Step 0 — secret (laptop or box)
Add a random GateGuard JWT key to `/opt/lia/backend/.env.prod` (chmod 600, git-ignored):
```bash
ssh vdska2 'cd /opt/lia/backend && grep -q GATEGUARD_AUTH_SECRET .env.prod || echo "GATEGUARD_AUTH_SECRET=$(openssl rand -hex 32)" >> .env.prod'
```

## Step 1 — copy GateGuard source to the box
From laptop (exclude build cruft; `.git` IS needed — the Dockerfile copies it):
```bash
rsync -az --delete --exclude 'data' --exclude 'node_modules' \
  -e 'ssh -o ConnectTimeout=10' \
  /Users/dodonovpavel/gateway_fm/appstore/gateguard/ vdska2:/opt/gateguard/
```

## Step 2 — build the image (detached + poll; flaky box)
> ⚠️ GOTCHA (hit 2026-06-25): the box's Docker build can't reach github.com over
> IPv6 (broken on this box) — the Dockerfile's `curl -OL` of protoc/grpc-web hung
> 5 min → `curl: (28) SSL connection timeout`. Fix: force IPv4 + retries on the
> box's copy of the Dockerfile before building:
> ```bash
> ssh vdska2 'cd /opt/gateguard && sed -i "s#curl -OL#curl -4 --retry 5 --retry-delay 3 --connect-timeout 30 -OL#g" Dockerfile'
> ```
> (go mod download works — it goes via proxy.golang.org, not github directly.)
```bash
ssh vdska2 'cd /opt/gateguard && rm -f /tmp/gg-build.done && setsid bash -c "docker build -t gateguard:local . > /tmp/gg-build.log 2>&1; echo \$? > /tmp/gg-build.done" </dev/null >/dev/null 2>&1 &'
# poll /tmp/gg-build.done for rc=0
```

## Step 3 — create the gateguard database (in Lia's Postgres)
```bash
ssh vdska2 'cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
  psql -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "CREATE DATABASE gateguard;"'
# (idempotent: ignore "already exists")
```

## Step 4 — run GateGuard migrations against that DB
```bash
ssh vdska2 'set -a; . /opt/lia/backend/.env.prod; set +a; \
  docker run --rm --network backend_default -v /opt/gateguard/db:/db migrate/migrate:v4.17.1 \
  -path=/db/ -database "postgresql://$DATABASE_USER:$DATABASE_PASSWORD@postgres:5432/gateguard?sslmode=disable" up'
# NOTE: confirm the compose network name with `docker network ls | grep lia\|backend`.
```

## Step 5 — bring up GateGuard + Redis (joins Lia's stack)
```bash
ssh vdska2 'cd /opt/lia/backend && docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d gateguard-redis gateguard app'
```

## Step 6 — verify (Phase B done)
```bash
ssh vdska2 'docker compose --env-file /opt/lia/backend/.env.prod -f /opt/lia/backend/docker-compose.yml -f /opt/lia/backend/docker-compose.prod.yml -f /opt/lia/backend/docker-compose.gateguard.yml logs --tail=20 gateguard'
# expect: gRPC server listening on :9090, no DB/redis dial errors
ssh vdska2 'docker ps --filter name=gateguard --format "{{.Names}} {{.Status}}"'
```
`HTTP_MOCK_AUTH` is still true → live demo unchanged; Lia does not call GateGuard yet.

## Phase C (next) — demo-login + flip
- Backend: a demo-login entrypoint that calls `GateguardService.SignInOAuth(User{email,name})`
  → JWT → httpOnly cookie on the lia domain. Frontend `Войти` form + attach Bearer.
- Then set `HTTP_MOCK_AUTH=false`, redeploy `app`, verify `POST /events`:
  401 anonymous, 201 after demo-login. Update HANDOFF + [[lia-demo-deployment]].
- **Audit (ISO/Vanta):** flipping mock auth off is an access-control change — document it.
