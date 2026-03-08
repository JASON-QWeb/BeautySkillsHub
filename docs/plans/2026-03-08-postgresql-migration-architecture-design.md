# PostgreSQL Migration Architecture Design

**Date:** 2026-03-08

**Status:** Approved for implementation

## Goal

Replace the current SQLite + GORM `AutoMigrate` startup behavior with a PostgreSQL-based, migration-driven architecture that is safe for repeat deployments, preserves existing data, and supports simple local development.

## Scope

- Standardize all environments on PostgreSQL.
- Move business schema ownership into versioned SQL migrations.
- Keep local development simple with project-managed Docker infrastructure.
- Keep shared and production environments database-independent from the app deployment package.
- Keep frontend configuration lightweight and public-only.

## Non-Goals

- No RBAC or reviewer/admin role redesign in this phase.
- No production HA topology redesign beyond enabling a cleaner database boundary.
- No frontend runtime-config platform rewrite in this phase.

## Current State

The backend opens SQLite directly and runs `db.AutoMigrate(...)` during server startup. Data is persisted in Docker volumes for the backend container, and deployment currently rebuilds frontend/backend containers through Compose.

This is convenient for local use but does not meet the desired standard for controlled schema evolution in shared or production environments.

## Target Architecture

### Repository Layout

```text
frontend/
backend/
db/
  init/
  migrations/
  seed/
infra/
  docker/
    compose.local.yml
scripts/
  db-local.sh
  run-all-migrations.sh
  seed-local.sh
docs/
  plans/
```

### Responsibilities

- `backend/`
  - Connects to PostgreSQL through `DATABASE_URL`.
  - Owns business logic, API handlers, and model serialization.
  - Does not mutate schema automatically on startup.
- `db/init/`
  - Local-only database bootstrap scripts.
  - Creates required extensions or database-level prerequisites.
  - Does not define business tables.
- `db/migrations/`
  - Single source of truth for schema creation and schema evolution.
  - Includes initial schema and all later changes.
- `db/seed/`
  - Optional local/test data only.
- `infra/docker/compose.local.yml`
  - Local developer infrastructure for PostgreSQL and Redis.
  - May optionally run app services for convenience.

## Migration Strategy

### Tooling

Use `golang-migrate` with SQL migration files.

Why:

- Widely used in PostgreSQL-backed Go services.
- Keeps schema changes explicit and reviewable.
- Decouples schema rollout from application startup.
- Works well in local scripts and CI/CD jobs.

### Rules

- All business table creation belongs in `db/migrations/`.
- `db/init/` is limited to local database bootstrap and extensions.
- Production and shared environments run migrations explicitly before backend rollout.
- Backend startup must fail fast on connection/config issues, not attempt schema repair.

### Change Safety

Schema evolution follows expand-and-contract:

1. Additive change first.
2. Backfill data if needed.
3. Update backend to read/write the new shape.
4. Remove old shape only in a later migration.

This avoids destructive deployments and preserves existing data across releases.

## Environment Configuration

### Backend

Committed:

- `backend/.env.example`

Local-only:

- `backend/.env.local`

Behavior:

- Local development may load `.env.local`.
- Shared and production environments receive real values from deployment/runtime secret injection.
- SQLite-specific settings are removed in favor of `DATABASE_URL`.

Representative variables:

```env
APP_ENV=local
PORT=8080
DATABASE_URL=postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable
JWT_SECRET=replace-me
REDIS_ADDR=localhost:6379
OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o-mini
GITHUB_SYNC_ENABLED=false
```

### Frontend

Committed:

- `frontend/.env.example`

Local-only:

- `frontend/.env.local`

Allowed values are public-only:

```env
VITE_APP_ENV=local
VITE_API_BASE_URL=http://localhost:8080
```

Secrets are never exposed to the frontend.

## Local Development Flow

### First-time setup

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
./scripts/seed-local.sh
```

### Normal daily flow

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go run cmd/server/main.go
cd frontend && npm run dev
```

Behavior:

- `db-local.sh`
  - Starts local PostgreSQL and Redis through local Compose.
  - Waits until PostgreSQL is ready.
  - Applies `db/init/` scripts.
- `run-all-migrations.sh`
  - Migrates the target database to the latest schema version.
- `seed-local.sh`
  - Optional developer/test dataset.

## Deployment Flow

Shared and production deployments follow this order:

1. Run schema migrations against target PostgreSQL.
2. Roll out backend.
3. Roll out frontend.

The app does not own schema rollout at startup.

## Testing and Verification

### Backend

- Update tests to use PostgreSQL-oriented configuration logic where relevant.
- Keep fast unit tests where DB access can remain mocked or isolated.
- Add migration verification steps:
  - migrate empty database to latest
  - migrate existing database forward

### Frontend

- Keep configuration lightweight.
- Verify build still succeeds.
- Verify API base handling works for local development.

## Implementation Notes

- Existing SQLite-specific scripts and docs need replacement or deprecation.
- Existing `AutoMigrate` calls and `DB_PATH` config must be removed.
- Existing deployment docs must be updated to mention PostgreSQL, migration execution, and volume responsibilities for local-only infrastructure.

## Success Criteria

- Backend no longer performs schema migrations on startup.
- Schema is fully defined by versioned SQL migration files.
- Local development can stand up PostgreSQL with a single script.
- Repeat deployments run only pending migrations and preserve existing data.
- Frontend and backend both remain usable for local development with environment-specific configuration.
