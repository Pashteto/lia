#!/usr/bin/env bash
#
# vpn-build-all.sh  —  run ON vds-ru215 (the lia worker box).
#
# Brings the AmneziaWG full tunnel UP (so the box has reliable internet via
# vds-amnezia), pulls + builds EVERY network-dependent dep for the lia deploy
# (GateGuard, backend, frontend), then ALWAYS brings the tunnel back DOWN.
#
# This supersedes vpn-install-deps.sh, which predated the vendored GateGuard
# service. New here:
#   * pulls the Go toolchain base images the builds need (golang:1.24 for the
#     backend, golang:1.25 for GateGuard) — the old script only pulled
#     postgis/migrate, so GateGuard's `go mod download` failed with
#     GOTOOLCHAIN=local on the box's stale builder;
#   * builds the GateGuard image (gateguard:local) from /opt/gateguard;
#   * longer watchdog (60 min) to cover three sequential builds on 1 vCPU.
#
# Safety (unchanged): self-detaches (setsid) to survive SSH drops; an EXIT trap
# AND a hard watchdog both force `awg-quick down`, so the VPN can never be left
# on; connmark rules best-effort keep inbound SSH alive during the window.
#
# Usage (as root on the box):   bash /opt/lia/vpn-build-all.sh
# Follow progress:              tail -f /var/log/lia-vpn-build.log

set -uo pipefail

WG_IFACE="awg0"
PUB_IFACE="ens18"
MARK="0x10000"                 # high bit, no overlap with wg's fwmark 0xca6c
WATCHDOG_SECS=3600             # 60 min hard cap (3 builds on 1 vCPU), then force VPN down
LOG="/var/log/lia-vpn-build.log"
BACKEND="/opt/lia/backend"
FRONTEND="/opt/lia/frontend"
GATEGUARD="/opt/gateguard"
API_URL="https://api.lia.pashteto.com"

# Toolchain + base images pulled while the tunnel is up so the multi-stage
# builds never have to reach the registry on the box's broken native network.
PULL_IMAGES=(
  "golang:1.26"            # GateGuard builder (go.mod go 1.26 + grpc v1.81 latest)
  "golang:1.24"            # backend builder
  "postgis/postgis:16-3.4"
  "migrate/migrate:latest"
  "redis:7.2.4"            # GateGuard sidecar
)

log(){ echo "[$(date -u +%H:%M:%S)] $*"; }

if [[ $EUID -ne 0 ]]; then echo "must run as root"; exit 1; fi

# ---- self-detach so a frozen SSH session can't kill the run ----------------
if [[ "${LIA_DETACHED:-}" != "1" ]]; then
  export LIA_DETACHED=1
  : > "$LOG"
  setsid bash "$0" "$@" >>"$LOG" 2>&1 < /dev/null &
  echo "Started detached (pid $!).  The VPN will go UP, deps build, VPN goes DOWN."
  echo "Your SSH may briefly freeze while the tunnel is up — that's expected; it self-heals."
  echo "Follow:  tail -f $LOG"
  exit 0
fi

# ---------------------------- from here: detached ---------------------------
add_preserve_rules(){
  iptables -t mangle -C PREROUTING -i "$PUB_IFACE" -m conntrack --ctstate NEW \
      -j CONNMARK --set-xmark "$MARK/$MARK" 2>/dev/null || \
  iptables -t mangle -A PREROUTING -i "$PUB_IFACE" -m conntrack --ctstate NEW \
      -j CONNMARK --set-xmark "$MARK/$MARK"
  iptables -t mangle -C OUTPUT -j CONNMARK --restore-mark --nfmask "$MARK" --ctmask "$MARK" 2>/dev/null || \
  iptables -t mangle -A OUTPUT -j CONNMARK --restore-mark --nfmask "$MARK" --ctmask "$MARK"
  ip rule list | grep -q "fwmark $MARK lookup main" || \
      ip rule add fwmark "$MARK" lookup main pref 100
}
del_preserve_rules(){
  ip rule del fwmark "$MARK" lookup main pref 100 2>/dev/null || true
  iptables -t mangle -D OUTPUT -j CONNMARK --restore-mark --nfmask "$MARK" --ctmask "$MARK" 2>/dev/null || true
  iptables -t mangle -D PREROUTING -i "$PUB_IFACE" -m conntrack --ctstate NEW \
      -j CONNMARK --set-xmark "$MARK/$MARK" 2>/dev/null || true
  iptables -t mangle -D POSTROUTING -o "$WG_IFACE" -p tcp --tcp-flags SYN,RST SYN \
      -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true
}

# AmneziaWG obfuscation (S1/S2) + path overhead means 1280 is too big: large TLS
# transfers (image layers) get dropped while tiny requests pass. Lower the inner
# MTU and clamp TCP MSS on the tunnel egress so segments actually fit the path.
apply_mtu_fix(){
  ip link set dev "$WG_IFACE" mtu 1200 2>/dev/null || true
  iptables -t mangle -C POSTROUTING -o "$WG_IFACE" -p tcp --tcp-flags SYN,RST SYN \
      -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || \
  iptables -t mangle -A POSTROUTING -o "$WG_IFACE" -p tcp --tcp-flags SYN,RST SYN \
      -j TCPMSS --clamp-mss-to-pmtu
  log "MTU lowered to 1200 on $WG_IFACE + MSS clamped to path MTU on egress"
}
vpn_down(){
  awg-quick down "$WG_IFACE" >/dev/null 2>&1 || true
  del_preserve_rules
  log "=== VPN DOWN + rules removed.  default route: $(ip route show default) ==="
}

# always tear down, whatever happens
trap 'rc=$?; log "trap: tearing down (rc=$rc)"; vpn_down; exit $rc' EXIT INT TERM

# watchdog: force teardown + kill the run if it overruns
( sleep "$WATCHDOG_SECS"; echo "[watchdog] $WATCHDOG_SECS s elapsed — forcing VPN down" >>"$LOG"; \
  awg-quick down "$WG_IFACE" >/dev/null 2>&1; \
  ip rule del fwmark "$MARK" lookup main pref 100 2>/dev/null; \
  kill -TERM -$$ 2>/dev/null ) &
WATCHDOG=$!
disown "$WATCHDOG" 2>/dev/null || true

log "=== bringing VPN UP (full tunnel -> vds-amnezia) ==="
add_preserve_rules
if ! awg-quick up "$WG_IFACE"; then log "FATAL: awg-quick up failed"; exit 1; fi
sleep 3
apply_mtu_fix

EGRESS_IP="$(curl -4 -s --max-time 20 https://api.ipify.org || echo '?')"
log "egress IP via tunnel: $EGRESS_IP   (expect 185.5.75.80 = vds-amnezia)"
log "registry reachable: $(curl -4 -s -o /dev/null -w '%{http_code}' --max-time 25 https://registry-1.docker.io/v2/ || echo timeout)"

# ----------------------------- pull base images ----------------------------
log "=== pulling base/toolchain images ==="
for img in "${PULL_IMAGES[@]}"; do
  ok=0
  for i in 1 2 3 4 5; do
    if docker pull "$img"; then ok=1; break; fi
    log "retry pull $img ($i/5)"; sleep 3
  done
  [[ "$ok" == 1 ]] || { log "ERROR: failed to pull $img"; exit 1; }
done

# ----------------------------- build images --------------------------------
# GateGuard first: heaviest + most network-dependent (go mod download, go
# install protoc plugins, curl protoc from github). Its Dockerfile forces IPv4
# on the github curls and uses golang:1.25 to satisfy the go.mod directive.
log "=== building GateGuard image (gateguard:local) ==="
( docker build -t gateguard:local "$GATEGUARD" ) || \
  { log "ERROR: gateguard build failed"; exit 1; }

log "=== building backend image (go mod download etc.) ==="
( cd "$BACKEND" && docker compose --env-file .env.prod \
    -f docker-compose.yml -f docker-compose.prod.yml build ) || \
  { log "ERROR: backend build failed"; exit 1; }

log "=== building frontend image (pnpm install + next build) ==="
( cd "$FRONTEND" && docker build --build-arg NEXT_PUBLIC_API_URL="$API_URL" -t lia-frontend . ) || \
  { log "ERROR: frontend build failed"; exit 1; }

log "=== images present ==="
docker images --format '  {{.Repository}}:{{.Tag}}  {{.Size}}' | grep -E 'lia-frontend|gateguard|postgis|migrate|backend|redis' || true

# stop the watchdog; the EXIT trap will bring the VPN down cleanly
kill "$WATCHDOG" 2>/dev/null || true
log "=== ALL IMAGES BUILT — VPN will now be turned OFF by trap ==="
# (trap on EXIT runs vpn_down here)
{
  echo "NEXT (no internet needed):"
  echo "  # GateGuard DB migration (one-time, if not already applied):"
  echo "  cat /opt/gateguard/db/000011_add_password_and_email_verification.up.sql | docker exec -i backend-postgres-1 psql -U <DB_USER> -d gateguard"
  echo "  # bring the stack up (recreates with the freshly built images):"
  echo "  cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d"
  echo "  docker rm -f lia-frontend 2>/dev/null; docker run -d --restart unless-stopped --name lia-frontend -p 127.0.0.1:3001:3001 lia-frontend"
} >>"$LOG"
echo "MARKER:BUILD-ALL-DONE" >>"$LOG"
