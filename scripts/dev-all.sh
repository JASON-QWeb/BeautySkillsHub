#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ENV_FILE="$ROOT_DIR/backend/.env.local"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
SEED_LOCAL="${SEED_LOCAL:-1}"

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "Required file not found: $path" >&2
    exit 1
  fi
}

ensure_port_free() {
  local port="$1"
  local name="$2"
  if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "$name port is already in use: $port" >&2
    exit 1
  fi
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local attempts="${3:-60}"

  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$name is ready: $url"
      return 0
    fi
    sleep 1
  done

  echo "$name did not become ready: $url" >&2
  return 1
}

cleanup() {
  local exit_code=$?

  if [[ -n "${backend_pid:-}" ]] && kill -0 "$backend_pid" >/dev/null 2>&1; then
    kill "$backend_pid" >/dev/null 2>&1 || true
    wait "$backend_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "${frontend_pid:-}" ]] && kill -0 "$frontend_pid" >/dev/null 2>&1; then
    kill "$frontend_pid" >/dev/null 2>&1 || true
    wait "$frontend_pid" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

require_file "$BACKEND_ENV_FILE"
ensure_port_free "$BACKEND_PORT" "backend"
ensure_port_free "$FRONTEND_PORT" "frontend"

cd "$ROOT_DIR"

./scripts/db-local.sh
./scripts/run-all-migrations.sh

if [[ "$SEED_LOCAL" == "1" ]]; then
  ./scripts/seed-local.sh
fi

if [[ ! -d "$ROOT_DIR/frontend/node_modules" ]]; then
  (
    cd "$ROOT_DIR/frontend"
    npm ci
  )
fi

(
  cd "$ROOT_DIR/backend"
  go run ./cmd/server/main.go 2>&1 | sed -u 's/^/[backend] /'
) &
backend_pid=$!

wait_for_http "backend" "http://127.0.0.1:${BACKEND_PORT}/api/skills"

(
  cd "$ROOT_DIR/frontend"
  npm run dev -- --host 0.0.0.0 2>&1 | sed -u 's/^/[frontend] /'
) &
frontend_pid=$!

wait_for_http "frontend" "http://127.0.0.1:${FRONTEND_PORT}"

echo "Local stack is ready."
echo "Backend:  http://127.0.0.1:${BACKEND_PORT}"
echo "Frontend: http://127.0.0.1:${FRONTEND_PORT}"
echo "Press Ctrl+C to stop backend/frontend. PostgreSQL and Redis remain in Docker."

while true; do
  if ! kill -0 "$backend_pid" >/dev/null 2>&1; then
    wait "$backend_pid"
    exit $?
  fi
  if ! kill -0 "$frontend_pid" >/dev/null 2>&1; then
    wait "$frontend_pid"
    exit $?
  fi
  sleep 1
done
