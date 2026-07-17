#!/usr/bin/env bash
# Inserts the 115 real venues from venues_seed.sql into the prod `venues` table
# on vds-ru215 (presence.tarski.ru). Idempotent (see venues_seed.sql header) —
# safe to re-run, but this still WRITES to prod, so it is not run automatically.
#
# Usage:
#   ./run_venues_seed.sh --dry-run   # counts how many rows would insert, no writes
#   ./run_venues_seed.sh --apply     # backs up the venues table, then inserts
#
# Requires: ssh alias `vdska2` configured (per docs/superpowers/runbooks), the
# box's Lia stack running in /opt/lia/backend.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="$SCRIPT_DIR/venues_seed.sql"
BOX_HOST="vdska2"
BOX_DIR="/opt/lia/backend"
REMOTE_SQL_PATH="$BOX_DIR/db/seed/venues_seed.sql"
COMPOSE_CMD="docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml"

mode="${1:-}"
if [[ "$mode" != "--dry-run" && "$mode" != "--apply" ]]; then
  echo "Usage: $0 --dry-run | --apply" >&2
  exit 1
fi

if [[ ! -f "$SQL_FILE" ]]; then
  echo "Missing $SQL_FILE" >&2
  exit 1
fi

echo "==> Copying $SQL_FILE to $BOX_HOST:$REMOTE_SQL_PATH"
scp -o ConnectTimeout=15 "$SQL_FILE" "$BOX_HOST:$REMOTE_SQL_PATH"

# The SQL file lives on the box's filesystem, which is NOT mounted into the
# postgres container — feed it to psql via stdin (same technique as seed.sql).
if [[ "$mode" == "--dry-run" ]]; then
  echo "==> Dry run: inserting inside a transaction, then ROLLBACK (no writes)"
  ssh -o ConnectTimeout=15 "$BOX_HOST" "cd $BOX_DIR && { echo 'BEGIN;'; cat $REMOTE_SQL_PATH; echo 'SELECT count(*) AS venues_after_dry_run FROM venues;'; echo 'ROLLBACK;'; } | $COMPOSE_CMD exec -T postgres sh -c 'psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1' | grep -Ev '^INSERT 0 ' "
  echo "==> Dry run complete (ROLLBACK issued, nothing written)."
  exit 0
fi

echo "==> Backing up the venues table before inserting"
BACKUP_NAME="venues-pre-seed-$(date -u +%Y%m%d-%H%M%S 2>/dev/null || echo manual).sql.gz"
ssh -o ConnectTimeout=15 "$BOX_HOST" "cd $BOX_DIR && $COMPOSE_CMD exec -T postgres sh -c '
  pg_dump -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -t venues
' | gzip > /tmp/$BACKUP_NAME"
scp -o ConnectTimeout=15 "$BOX_HOST:/tmp/$BACKUP_NAME" "$SCRIPT_DIR/$BACKUP_NAME" 2>/dev/null || true
echo "    backup saved on box at /tmp/$BACKUP_NAME"

echo "==> Applying insert"
ssh -o ConnectTimeout=15 "$BOX_HOST" "cd $BOX_DIR && $COMPOSE_CMD exec -T postgres sh -c 'psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1' < $REMOTE_SQL_PATH | sort | uniq -c | sort -rn"

echo "==> Verifying new row count"
ssh -o ConnectTimeout=15 "$BOX_HOST" "cd $BOX_DIR && $COMPOSE_CMD exec -T postgres sh -c '
  psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"SELECT count(*) FROM venues;\"
'"

echo "==> Done."
