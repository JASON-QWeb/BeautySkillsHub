#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT_DIR/db/migrations}"

if [[ -f "$BACKEND_DIR/.env.local" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$BACKEND_DIR/.env.local"
  set +a
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

cd "$BACKEND_DIR"
go run ./cmd/migrate -database-url "$DATABASE_URL" -migrations-dir "$MIGRATIONS_DIR"
