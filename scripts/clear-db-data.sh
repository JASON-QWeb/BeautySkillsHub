#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "--- Cleaning up database ---"
cd "$ROOT_DIR/backend"
go run ./cmd/clear-db

echo "--- Cleaning up avatars, thumbnails, and uploads ---"
# Keep the directories but delete all contents
find "$ROOT_DIR/backend/avatars" -mindepth 1 -delete
find "$ROOT_DIR/backend/thumbnails" -mindepth 1 -delete
find "$ROOT_DIR/backend/uploads" -mindepth 1 -delete

echo "--- Cleanup complete ---"
