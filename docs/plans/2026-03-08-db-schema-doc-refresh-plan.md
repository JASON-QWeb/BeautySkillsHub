# DB Schema Doc Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refresh `db/SCHEMA.md` so it accurately documents the `db/` directory, every tracked SQL file, the current PostgreSQL schema, and safe migration behavior.

**Architecture:** Keep a single authoritative document in `db/SCHEMA.md` and remove stale statements instead of splitting the topic across multiple files. Use the existing migration SQL, local database scripts, and model/service code as the factual source for table shape, relationships, statuses, and execution flow.

**Tech Stack:** Markdown, PostgreSQL SQL migrations, Bash scripts, Go models/services

---

### Task 1: Capture the authoritative database facts

**Files:**
- Read: `db/migrations/0001_init_schema.up.sql`
- Read: `db/migrations/0001_init_schema.down.sql`
- Read: `db/migrations/0002_add_skill_revisions.up.sql`
- Read: `db/migrations/0002_add_skill_revisions.down.sql`
- Read: `db/init/001-extensions.sql`
- Read: `db/seed/local_seed.sql`
- Read: `scripts/db-local.sh`
- Read: `scripts/run-all-migrations.sh`
- Read: `scripts/seed-local.sh`
- Read: `backend/internal/model/skill.go`
- Read: `backend/internal/model/skill_revision.go`
- Read: `backend/internal/model/skill_like.go`
- Read: `backend/internal/model/skill_favorite.go`
- Read: `backend/internal/model/user.go`
- Read: `backend/internal/service/github_sync_service.go`
- Read: `backend/internal/service/skill_revision.go`

**Step 1: Verify the current file inventory**

Run: `find db -maxdepth 3 -type f | sort`
Expected: only the tracked database documentation and SQL files appear.

**Step 2: Confirm how each SQL file is used**

Run: `sed -n '1,220p' scripts/db-local.sh && sed -n '1,220p' scripts/run-all-migrations.sh && sed -n '1,220p' scripts/seed-local.sh`
Expected: local init uses `db/init`, migrations use `db/migrations`, and optional seed uses `db/seed/local_seed.sql`.

**Step 3: Confirm current schema and status constants**

Run: `sed -n '1,260p' backend/internal/model/skill.go && sed -n '1,240p' backend/internal/model/skill_revision.go`
Expected: current review status and revision status values match what the document will describe.

**Step 4: Commit**

```bash
git add docs/plans/2026-03-08-db-schema-doc-refresh-design.md docs/plans/2026-03-08-db-schema-doc-refresh-plan.md
git commit -m "docs: capture db schema doc refresh plan"
```

### Task 2: Rewrite the main database guide

**Files:**
- Modify: `db/SCHEMA.md`

**Step 1: Replace stale sections with a new document structure**

Write these sections:

- overview and source of truth
- `db/` directory map
- file-by-file explanation
- execution flow for init, migrations, and seed
- current table structure
- relationship and constraint summary
- status field reference
- migration safety rules

**Step 2: Ensure the table reference matches current schema**

Cover:

- `users`
- `skills`
- `skill_likes`
- `skill_favorites`
- `skill_revisions`

Expected: no claim remains that the schema only has four main business tables.

**Step 3: Include migration behavior clarification**

Document:

- `.up.sql` is used for normal forward deploys
- `.down.sql` is rollback-only and not run by `cmd/migrate`
- old deployed migrations should not be edited in place

**Step 4: Commit**

```bash
git add db/SCHEMA.md
git commit -m "docs: refresh database schema guide"
```

### Task 3: Remove junk and verify the new documentation

**Files:**
- Delete: `db/.DS_Store`
- Verify: `db/SCHEMA.md`

**Step 1: Remove the useless Finder artifact**

Delete: `db/.DS_Store`

Expected: `find db -maxdepth 3 -type f | sort` no longer lists `.DS_Store`.

**Step 2: Verify the rewritten guide references real files and current schema**

Run: `rg -n "skill_revisions|db/init|local_seed.sql|down migration|four" db/SCHEMA.md`
Expected:

- `skill_revisions` appears in the guide
- `db/init` and `local_seed.sql` are explained
- migration rollback guidance is present
- stale "four tables" wording is gone

**Step 3: Manually review the final document**

Run: `sed -n '1,260p' db/SCHEMA.md`
Expected: the document reads as a single coherent database guide without stale sections.

**Step 4: Commit**

```bash
git add db/SCHEMA.md docs/plans/2026-03-08-db-schema-doc-refresh-design.md docs/plans/2026-03-08-db-schema-doc-refresh-plan.md
git commit -m "docs: document database layout and schema"
```
