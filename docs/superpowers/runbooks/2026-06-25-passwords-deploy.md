# Runbook — Password auth + events-user-data deploy to vds-ru215 (2026-06-25)

Deploys branch `feat/passwords-gateguard-and-events-user-data`: real password
sign-up/sign-in (GateGuard vendored + extended), event organizer data +
`GET /events/mine`, default-published, cover-image host fix, password UI.

Live target: `https://lia.pashteto.com` / `https://api.lia.pashteto.com` on
**vds-ru215** (`193.32.188.7`, `ssh vdska2`, root). Hand-managed Docker (documented
demo exception to the Terraform standard).

## TL;DR of what bit us (read first)

1. **The box cannot pull `golang:1.25`/`1.26` over the AmneziaWG tunnel** — the
   large layers fail with `connection reset by peer` (small images like
   `golang:1.24`, postgis, redis pull fine). GateGuard's upgraded deps (grpc
   v1.81, protobuf v1.36, otel) require **Go ≥ 1.25**, so it can't be built on the
   box. → **Build the `linux/amd64` images on a well-connected machine (the Mac)
   and ship them over SSH.** This is the reliable path.
2. **grpc Unimplemented embed**: protoc-gen-go-grpc v1.6+ requires the handler to
   embed `UnimplementedGateguardServiceServer` **by value** (not pointer), or it
   nil-panics at `RegisterGateguardServiceServer`. Build compiles either way.
3. **genproto module split**: `go get -u ./...` can produce an ambiguous
   `googleapis/rpc/status` import; bump the monolith + split module to latest.
4. **Full-tunnel VPN freezes inbound SSH** while up; on a 1.9 GB box a big image
   pull + builds can OOM-thrash sshd into unresponsiveness. Prefer not to build on
   the box; if you must, free RAM first (stop `backend-app` + `lia-frontend`).

## Procedure (build-on-Mac, ship images)

All three images are tiny except the frontend (215 MB). Built on the Mac with
`--platform linux/amd64` (Mac pulls `golang:1.26` reliably), shipped via
`docker save | gzip | ssh | docker load`, retagged to the compose names.

```bash
# 0. (one-time, if not already) GateGuard DB migration on the box:
cat /opt/gateguard/db/000011_add_password_and_email_verification.up.sql \
  | ssh vdska2 'docker exec -i backend-postgres-1 psql -U lia_prod -d gateguard'

# 1. GateGuard
cd gateguard && docker build --platform linux/amd64 -t gateguard:local-amd64 .
docker save gateguard:local-amd64 | gzip | ssh vdska2 'gunzip | docker load'
ssh vdska2 'docker tag gateguard:local-amd64 gateguard:local'

# 2. Backend (image name compose expects: backend-app:latest)
cd ../backend && docker build --platform linux/amd64 -t backend-app:amd64 .
docker save backend-app:amd64 | gzip | ssh vdska2 'gunzip | docker load'
ssh vdska2 'docker tag backend-app:amd64 backend-app:latest'

# 3. Frontend
cd ../frontend && docker build --platform linux/amd64 \
  --build-arg NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com -t lia-frontend:amd64 .
docker save lia-frontend:amd64 | gzip | ssh vdska2 'gunzip | docker load'
ssh vdska2 'docker tag lia-frontend:amd64 lia-frontend:latest'

# 4. Bring the stack up on the new images (NO --build) + recreate frontend
ssh vdska2 'cd /opt/lia/backend && \
  docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml \
    -f docker-compose.gateguard.yml up -d --no-build'
ssh vdska2 'docker rm -f lia-frontend; \
  docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend:latest'
```

Keep the source on the box in sync too (for future on-box builds / reference):
`rsync -az --exclude=.git --exclude=data/ --exclude='*.pb.go' gateguard/ vdska2:/opt/gateguard/`
and likewise `backend/` → `/opt/lia/backend/` (exclude `.env*`, `data/`), `frontend/` → `/opt/lia/frontend/`.

## Verify (public API, real GateGuard, HTTP_MOCK_AUTH=false)

```bash
API=https://api.lia.pashteto.com/api/v1; EM="pwtest_$(date +%s)@presence.test"; PW=hunter2pass
curl -s -X POST $API/auth/register -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EM\",\"name\":\"Test\",\"password\":\"$PW\"}"          # → 200 {token}
curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/auth/login \
  -H 'Content-Type: application/json' -d "{\"email\":\"$EM\",\"password\":\"$PW\"}"      # → 200
curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/auth/login \
  -H 'Content-Type: application/json' -d "{\"email\":\"$EM\",\"password\":\"x\"}"        # → 401
# create with the token → 201 status=published; GET $API/events/mine → returns it w/ organizer
```

All of the above were confirmed green on 2026-06-25.

## Recovery

- Box wedged / SSH dead during a VPN build window: restart via the VPS panel
  (https://vdska.ru/control/). AmneziaWG is not enabled at boot, so it comes back
  with the tunnel DOWN and the `restart: unless-stopped` containers back on the
  old images (Postgres + `lia_uploads` volume persist). Then redeploy via the
  build-on-Mac path above.
- The GateGuard DB migration is idempotent and persists across reboots.

## Follow-ups / not done

- Email verification is a **stub** (no mailer) — wire GateGuard's SMTP notificator
  before any real prod use; surface `/auth/verify-email` in Lia + a verify page.
- `deploy/vpn-build-all.sh` (VPN-up → build-on-box → VPN-down) is committed and
  correct in shape, but the `golang:1.25/1.26` pull over the tunnel makes it
  unreliable for GateGuard — the build-on-Mac+transfer path above is what works.
- The rest of the `/me/*` suite (saved/follows/notifications/applications) remains
  per `docs/superpowers/plans/2026-06-25-passwords-and-me-suite.md` Phase 3.
- Pre-existing GateGuard `Test_ReactToInvitation_Success` failure is unrelated to
  this work (org-invitation logic vs its test mocks) — left as-is.
