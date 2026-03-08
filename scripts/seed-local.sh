#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/compose.local.yml"
POSTGRES_USER="${POSTGRES_USER:-skillhub}"
POSTGRES_DB="${POSTGRES_DB:-skillhub_local}"
SEED_FILE="${SEED_FILE:-$ROOT_DIR/db/seed/local_seed.sql}"

if [[ ! -f "$SEED_FILE" ]]; then
  echo "Seed file not found: $SEED_FILE" >&2
  exit 1
fi

cat "$SEED_FILE" | docker compose -f "$COMPOSE_FILE" exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"

echo "Local seed data applied from $SEED_FILE"
