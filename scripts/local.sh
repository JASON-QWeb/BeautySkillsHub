#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
COMPOSE_FILE="$ROOT_DIR/infra/docker/compose.local.yml"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT_DIR/db/migrations}"
SEED_FILE="${SEED_FILE:-$ROOT_DIR/db/seed/local_seed.sql}"

default_local_database_url() {
  local user="${POSTGRES_USER:-skillhub}"
  local password="${POSTGRES_PASSWORD:-skillhub}"
  local db="${POSTGRES_DB:-skillhub_local}"
  local port="${POSTGRES_PORT:-5432}"
  printf 'postgres://%s:%s@localhost:%s/%s?sslmode=disable' "$user" "$password" "$port" "$db"
}

compose_local() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

load_backend_env() {
  if [[ -f "$BACKEND_DIR/.env.local" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "$BACKEND_DIR/.env.local"
    set +a
  fi
}

apply_local_defaults() {
  load_backend_env
  export APP_ENV="${APP_ENV:-local}"
  export PORT="${PORT:-8080}"
  export DATABASE_URL="${DATABASE_URL:-$(default_local_database_url)}"
  export JWT_SECRET="${JWT_SECRET:-local-dev-secret}"
  export REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
  export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
  export REDIS_DB="${REDIS_DB:-0}"
}

ensure_port_free() {
  local port="$1"
  local name="$2"

  if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "$name port is already in use: $port" >&2
    exit 1
  fi
}

wait_for_postgres() {
  local user="${POSTGRES_USER:-skillhub}"
  local db="${POSTGRES_DB:-skillhub_local}"

  until compose_local exec -T postgres pg_isready -U "$user" -d "$db" >/dev/null 2>&1; do
    sleep 1
  done
}

apply_init_sql() {
  local user="${POSTGRES_USER:-skillhub}"
  local db="${POSTGRES_DB:-skillhub_local}"

  for sql_file in "$ROOT_DIR"/db/init/*.sql; do
    [[ -e "$sql_file" ]] || continue
    cat "$sql_file" | compose_local exec -T postgres \
      psql -v ON_ERROR_STOP=1 -U "$user" -d "$db"
  done
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local attempts="${3:-60}"

  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$name is ready: $url"
      return 0
    fi
    sleep 1
  done

  echo "$name did not become ready: $url" >&2
  return 1
}

db_up() {
  export POSTGRES_USER="${POSTGRES_USER:-skillhub}"
  export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-skillhub}"
  export POSTGRES_DB="${POSTGRES_DB:-skillhub_local}"
  export POSTGRES_PORT="${POSTGRES_PORT:-5432}"

  compose_local up -d postgres redis
  wait_for_postgres
  apply_init_sql

  echo "Local PostgreSQL and Redis are ready."
  echo "PostgreSQL: localhost:${POSTGRES_PORT}"
  echo "Redis: localhost:${REDIS_PORT:-6379}"
}

db_down() {
  local args=(down)
  if [[ "${1:-}" == "--volumes" || "${1:-}" == "-v" ]]; then
    args+=(-v)
  fi
  compose_local "${args[@]}"
}

db_logs() {
  if [[ "$#" -eq 0 ]]; then
    compose_local logs -f postgres redis
    return
  fi
  compose_local logs -f "$@"
}

run_migrate() {
  apply_local_defaults
  (
    cd "$BACKEND_DIR"
    go run ./cmd/migrate -database-url "$DATABASE_URL" -migrations-dir "$MIGRATIONS_DIR"
  )
}

run_seed() {
  local user="${POSTGRES_USER:-skillhub}"
  local db="${POSTGRES_DB:-skillhub_local}"

  if [[ ! -f "$SEED_FILE" ]]; then
    echo "Seed file not found: $SEED_FILE" >&2
    exit 1
  fi

  cat "$SEED_FILE" | compose_local exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U "$user" -d "$db"

  echo "Local seed data applied from $SEED_FILE"
}

run_reset() {
  apply_local_defaults

  echo "--- Cleaning up database ---"
  (
    cd "$BACKEND_DIR"
    go run ./cmd/clear-db
  )

  echo "--- Cleaning up avatars, thumbnails, and uploads ---"
  for dir in \
    "$BACKEND_DIR/avatars" \
    "$BACKEND_DIR/thumbnails" \
    "$BACKEND_DIR/uploads"; do
    if [[ -d "$dir" ]]; then
      find "$dir" -mindepth 1 -delete
    fi
  done

  echo "--- Cleanup complete ---"
}

run_dev() {
  local backend_pid=""
  local frontend_pid=""
  local frontend_port="${FRONTEND_PORT:-5173}"

  cleanup() {
    if [[ -n "$backend_pid" ]] && kill -0 "$backend_pid" >/dev/null 2>&1; then
      kill "$backend_pid" >/dev/null 2>&1 || true
      wait "$backend_pid" >/dev/null 2>&1 || true
    fi
    if [[ -n "$frontend_pid" ]] && kill -0 "$frontend_pid" >/dev/null 2>&1; then
      kill "$frontend_pid" >/dev/null 2>&1 || true
      wait "$frontend_pid" >/dev/null 2>&1 || true
    fi
  }

  trap cleanup EXIT INT TERM

  apply_local_defaults
  export VITE_APP_ENV="${VITE_APP_ENV:-local}"
  export VITE_API_BASE_URL="${VITE_API_BASE_URL:-http://127.0.0.1:${PORT}/api}"

  ensure_port_free "$PORT" "backend"
  ensure_port_free "$frontend_port" "frontend"

  db_up
  run_migrate

  if [[ "${SEED_LOCAL:-1}" == "1" ]]; then
    run_seed
  fi

  if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    (
      cd "$FRONTEND_DIR"
      npm ci
    )
  fi

  (
    cd "$BACKEND_DIR"
    go run ./cmd/server/ 2>&1 | sed -u 's/^/[backend] /'
  ) &
  backend_pid=$!

  wait_for_http "backend" "http://127.0.0.1:${PORT}/api/skills"

  (
    cd "$FRONTEND_DIR"
    npm run dev -- --host 0.0.0.0 --port "$frontend_port" 2>&1 | sed -u 's/^/[frontend] /'
  ) &
  frontend_pid=$!

  wait_for_http "frontend" "http://127.0.0.1:${frontend_port}"

  echo "Local stack is ready."
  echo "Backend:  http://127.0.0.1:${PORT}"
  echo "Frontend: http://127.0.0.1:${frontend_port}"
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
}

print_help() {
  cat <<'EOF'
Skill Hub local operations

Usage:
  ./scripts/local.sh db up
  ./scripts/local.sh db down [-v|--volumes]
  ./scripts/local.sh db logs [postgres|redis]
  ./scripts/local.sh migrate
  ./scripts/local.sh seed
  ./scripts/local.sh reset
  ./scripts/local.sh dev
  ./scripts/local.sh help

Commands:
  db up                Start local PostgreSQL and Redis, then apply db/init SQL
  db down              Stop local PostgreSQL and Redis
  db down -v           Stop local PostgreSQL and Redis and remove their volumes
  db logs              Tail PostgreSQL and Redis logs
  migrate              Run SQL migrations against DATABASE_URL
  seed                 Apply db/seed/local_seed.sql to local PostgreSQL
  reset                Clear business data and local uploaded/generated assets
  dev                  Start Docker PostgreSQL/Redis plus host backend/frontend
  help                 Show this help

Notes:
  - Commands prefer backend/.env.local when present.
  - Without backend/.env.local, local defaults are used automatically.
  - Override behavior with env vars such as DATABASE_URL, POSTGRES_PORT, FRONTEND_PORT, SEED_LOCAL=0.
EOF
}

main() {
  case "${1:-help}" in
    db)
      case "${2:-help}" in
        up)
          db_up
          ;;
        down)
          db_down "${3:-}"
          ;;
        logs)
          shift 2
          db_logs "$@"
          ;;
        help|"")
          print_help
          ;;
        *)
          echo "Unknown db subcommand: ${2}" >&2
          print_help
          exit 1
          ;;
      esac
      ;;
    migrate)
      run_migrate
      ;;
    seed)
      run_seed
      ;;
    reset)
      run_reset
      ;;
    dev)
      run_dev
      ;;
    help|-h|--help)
      print_help
      ;;
    *)
      echo "Unknown command: $1" >&2
      print_help
      exit 1
      ;;
  esac
}

main "$@"
