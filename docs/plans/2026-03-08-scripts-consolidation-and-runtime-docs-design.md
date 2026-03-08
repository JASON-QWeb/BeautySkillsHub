# Scripts Consolidation And Runtime Docs Design

**Context**

The repository currently has five shell scripts under `scripts/`, but they form a split interface rather than independent tools. `dev-all.sh` already orchestrates the others, while `db-local.sh`, `run-all-migrations.sh`, `seed-local.sh`, and `clear-db-data.sh` expose individual steps. This works, but it is harder to remember, harder to document, and more likely to drift as the local development workflow evolves.

The user wants a stronger, easier-to-use interface, plus dedicated documentation for `scripts/`, `backend/`, and `frontend`. They also want `docs/review/` committed, which means it should reflect the real project state rather than stale or inaccurate findings.

**Goal**

Reduce local operational scripts to one obvious entrypoint, document the runtime surfaces clearly, and keep top-level docs plus `docs/review` aligned with the current codebase.

**Approaches**

1. Keep all existing scripts and only add README files.
   - Lowest implementation risk.
   - Rejected because it preserves command sprawl and does not meet the strong-consolidation requirement.

2. Replace the current five scripts with one `scripts/local.sh` command router and document that as the only supported local operations interface.
   - Easiest to teach and remember.
   - Keeps the underlying behavior but removes duplicated entrypoints.
   - Recommended.

3. Move script orchestration into a Makefile or npm task runner.
   - Viable, but adds another layer and would split shell, Go, and Node concerns.
   - Not needed for this repository size.

**Recommended Design**

- Introduce a single supported local-operations entrypoint: `scripts/local.sh`.
- Supported subcommands:
  - `db up`
  - `db down`
  - `db logs`
  - `migrate`
  - `seed`
  - `reset`
  - `dev`
  - `help`
- Keep the same underlying behavior:
  - `db up/down/logs` manage `infra/docker/compose.local.yml`
  - `migrate` still runs the Go migrator against `DATABASE_URL`
  - `seed` still applies `db/seed/local_seed.sql`
  - `reset` still clears business tables and local uploaded assets
  - `dev` still boots the host-process development flow with Docker Postgres/Redis
- Delete obsolete split-entry scripts after `local.sh` reaches parity.
- Add documentation:
  - `scripts/README.md`
  - `backend/README.md`
  - `frontend/README.md`
- Update `README.md`, `DEVELOPMENT.md`, and `DEPLOYMENT.md` to reference the new interface.
- Review and correct `docs/review/2026-03-08-architecture-review.md` before committing it.

**Validation**

- `./scripts/local.sh help`
- `./scripts/local.sh db up`
- `./scripts/local.sh migrate`
- `./scripts/local.sh seed`
- `./scripts/local.sh reset`
- `./scripts/local.sh db down`
- `cd backend && go test ./...`
- `cd frontend && npm run test:node`
- `cd frontend && npm run build`
- `git diff --check`
