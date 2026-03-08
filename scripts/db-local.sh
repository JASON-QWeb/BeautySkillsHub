#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/compose.local.yml"
POSTGRES_USER="${POSTGRES_USER:-skillhub}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-skillhub}"
POSTGRES_DB="${POSTGRES_DB:-skillhub_local}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"

export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_PORT

docker compose -f "$COMPOSE_FILE" up -d postgres redis

until docker compose -f "$COMPOSE_FILE" exec -T postgres \
  pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
  sleep 1
done

for sql_file in "$ROOT_DIR"/db/init/*.sql; do
  [[ -e "$sql_file" ]] || continue
  cat "$sql_file" | docker compose -f "$COMPOSE_FILE" exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
done

echo "Local PostgreSQL is ready on port $POSTGRES_PORT"
