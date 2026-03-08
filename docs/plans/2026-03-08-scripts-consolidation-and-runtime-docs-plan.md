# Scripts Consolidation And Runtime Docs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate local operations into one script interface, add runtime documentation for scripts/backend/frontend, and align top-level docs plus `docs/review` with the current codebase.

**Architecture:** Replace fragmented shell entrypoints with a single command router while preserving current behavior under the hood. Then document the new surface once and point all root docs at the same commands.

**Tech Stack:** Bash, Docker Compose, Go, Node.js, Markdown

---

### Task 1: Lock the expected script surface with a failing regression test

**Files:**
- Create: `scripts/local-script.test.mjs`

**Step 1: Write the failing test**

Assert that:
- `scripts/local.sh` exists
- old split scripts do not exist
- `README.md` and `DEVELOPMENT.md` reference `./scripts/local.sh`

**Step 2: Run test to verify it fails**

Run: `node --test scripts/local-script.test.mjs`

**Step 3: Write minimal implementation**

Create `local.sh`, delete obsolete scripts, and update docs.

**Step 4: Run test to verify it passes**

Run: `node --test scripts/local-script.test.mjs`

### Task 2: Implement the unified local operations script

**Files:**
- Create: `scripts/local.sh`
- Delete: `scripts/db-local.sh`
- Delete: `scripts/dev-all.sh`
- Delete: `scripts/run-all-migrations.sh`
- Delete: `scripts/seed-local.sh`
- Delete: `scripts/clear-db-data.sh`

**Step 1: Port behavior into subcommands**

Implement `db up/down/logs`, `migrate`, `seed`, `reset`, `dev`, and `help`.

**Step 2: Smoke-test each supported command**

Run:
- `./scripts/local.sh help`
- `./scripts/local.sh db up`
- `./scripts/local.sh migrate`
- `./scripts/local.sh db down`

### Task 3: Add runtime README files

**Files:**
- Create: `scripts/README.md`
- Create: `backend/README.md`
- Create: `frontend/README.md`

**Step 1: Document current structure and commands**

Explain what each area is for, which entrypoints matter, and how to run/build/test it.

**Step 2: Check that docs describe current commands**

Run: `rg -n "scripts/local.sh|npm run test:node|go test ./..." scripts/README.md backend/README.md frontend/README.md`

### Task 4: Align root docs and review notes

**Files:**
- Modify: `README.md`
- Modify: `DEVELOPMENT.md`
- Modify: `DEPLOYMENT.md`
- Modify: `docs/review/2026-03-08-architecture-review.md`

**Step 1: Replace stale script references**

Point local-development instructions to `./scripts/local.sh`.

**Step 2: Correct inaccurate review claims**

Only keep findings and fix statuses that match the real codebase.

### Task 5: Full verification and integration

**Files:**
- No new production files

**Step 1: Run verification**

Run:
- `node --test scripts/local-script.test.mjs`
- `./scripts/local.sh help`
- `./scripts/local.sh db up`
- `./scripts/local.sh migrate`
- `./scripts/local.sh db down`
- `cd backend && go test ./...`
- `cd frontend && npm run test:node`
- `cd frontend && npm run build`
- `git diff --check`

**Step 2: Commit and push**

Commit the consolidated scripts, docs, and `docs/review`, then push `main`.
