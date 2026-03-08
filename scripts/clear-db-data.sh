#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "--- Cleaning up database ---"
cd "$ROOT_DIR/backend"
go run ./cmd/clear-db

echo "--- Cleaning up avatars, thumbnails, and uploads ---"
# Keep the directories but delete all contents
for dir in \
  "$ROOT_DIR/backend/avatars" \
  "$ROOT_DIR/backend/thumbnails" \
  "$ROOT_DIR/backend/uploads"
do
  if [[ -d "$dir" ]]; then
    find "$dir" -mindepth 1 -delete
  fi
done

echo "--- Cleanup complete ---"
echo "Tip: run ./scripts/run-all-migrations.sh before restarting services if schema needs to be re-applied to a fresh database."
