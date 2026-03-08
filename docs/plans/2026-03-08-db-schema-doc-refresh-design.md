# DB Schema Doc Refresh Design

**Goal:** Replace the outdated `db/SCHEMA.md` with a single accurate database guide that explains the `db/` directory, every SQL file, the current PostgreSQL schema, and the migration rules for deployed environments.

## Problem

The existing `db/SCHEMA.md` is partially stale:

- it still describes only four main business tables even though `skill_revisions` now exists
- it explains table structure, but not the purpose of each file under `db/`
- it does not clearly separate local bootstrap SQL, business schema migrations, and optional seed data
- it leaves room for confusion about `up` versus `down` migrations in deployed environments

For maintainers, this means the most obvious database document is no longer a reliable source of truth.

## Requirements

1. Keep a single primary database document in `db/SCHEMA.md`.
2. Remove stale or redundant content instead of layering more text on top.
3. Explain the purpose of each folder and each tracked file under `db/`.
4. Document the current schema as defined by `0001` and `0002` migrations.
5. Clarify which relationships are enforced by PostgreSQL and which are only application conventions.
6. Explain how `db/init`, `db/migrations`, and `db/seed` are actually used by local scripts and deployment flow.
7. Clarify `up`/`down` migration behavior so operators do not assume deploys will run destructive rollback SQL.

## Options Considered

### Option A: Rewrite `db/SCHEMA.md` into a full database guide

Pros:

- one obvious place to look
- minimal maintenance overhead
- directly fixes the stale document the team already references

Cons:

- the file becomes broader than pure table schema

### Option B: Keep `db/SCHEMA.md` narrow and add `db/DATABASE.md`

Pros:

- clearer separation between schema reference and operational guide

Cons:

- two documents can drift again
- maintainers now have to guess which file is authoritative

## Decision

Use Option A.

`db/SCHEMA.md` will become the canonical database guide and include:

- `db/` directory map
- folder-by-folder and file-by-file explanation
- execution flow for init, migrations, and seed
- current table structure and relationships
- status field reference
- migration and deployment rules

## Expected Outcome

After the refresh, a developer should be able to answer all of these from `db/SCHEMA.md` alone:

- what lives under `db/`
- which SQL runs in local bootstrap versus normal deploy
- what each migration file does
- what each table stores
- which constraints are enforced by PostgreSQL
- how to safely add future migrations without damaging deployed data
