# Backend Review Fixes Round 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the next confirmed backend review issues while preserving current like/favorite business semantics and the non-blocking download experience.

**Architecture:** Add a safe follow-up migration for referential integrity, tighten handler validation with broad limits, and patch service/handler flows so download counting no longer fails silently and like counts are returned from the same transactional state. Keep the work test-driven and update the existing review-fix documentation for the shared architecture review.

**Tech Stack:** Go, Gin, GORM, PostgreSQL SQL migrations

---

### Task 1: Prepare baseline in the new worktree

**Files:**
- Create: `docs/plans/2026-03-08-backend-review-fixes-round2-design.md`
- Create: `docs/plans/2026-03-08-backend-review-fixes-round2-implementation-plan.md`

**Step 1: Verify branch and worktree**

Run: `git status --short --branch`
Expected: branch `codex/backend-review-fixes-round2`

**Step 2: Run backend baseline verification**

Run: `cd backend && go test ./...`
Expected: pass before changes

**Step 3: Run frontend build baseline verification**

Run: `cd frontend && npm run build`
Expected: pass before changes

### Task 2: Write failing tests for download counting and length validation

**Files:**
- Modify: `backend/internal/handler/skill_read_handlers_test.go`
- Modify: `backend/internal/handler/resource_handler_test.go`
- Modify: `backend/internal/handler/skill_upload_handlers_test.go`
- Modify: `backend/internal/handler/resource_revision_handlers_test.go`
- Modify: `backend/internal/handler/skill_revision_handlers_test.go`

**Step 1: Add `/download` counting failure test**

Show that the direct download path still serves the file when `IncrementDownload` fails, but no longer swallows the failure silently.

**Step 2: Add oversized payload validation tests**

Cover create and update flows for reviewed uploads and resource uploads.

**Step 3: Run focused tests to verify red**

Run: `cd backend && go test ./internal/handler -run 'Download|Upload|Revision' -v`
Expected: fail for missing behavior

### Task 3: Write failing tests for like-count consistency and migration SQL

**Files:**
- Modify: `backend/internal/service/skill_like_test.go`
- Create: `db/migrations/0003_add_engagement_foreign_keys.up.sql`
- Create: `db/migrations/0003_add_engagement_foreign_keys.down.sql`

**Step 1: Add like-count consistency regression test**

Verify the returned count is sourced from the transactionally updated row, not a later external read.

**Step 2: Add migration-content test or assertion**

Check that the new migration cleans orphans and adds foreign keys with cascade semantics.

**Step 3: Run focused tests to verify red**

Run: `cd backend && go test ./internal/service -run 'Like' -v`
Expected: fail before implementation

### Task 4: Implement the handler and service fixes

**Files:**
- Modify: `backend/internal/handler/skill_read_handlers.go`
- Modify: `backend/internal/handler/resource_handler.go`
- Modify: `backend/internal/handler/skill_upload_handlers.go`
- Modify: `backend/internal/service/skill.go`

**Step 1: Patch `/download` counting handling**

Log failed increments and continue serving the file.

**Step 2: Add shared broad input-length validation**

Apply the same limits in create and update paths with clear errors.

**Step 3: Make like/unlike return counts from the same transaction**

Preserve additive semantics and idempotency.

**Step 4: Re-run focused handler/service tests**

Run: `cd backend && go test ./internal/handler ./internal/service -v`
Expected: pass

### Task 5: Implement the foreign-key migration

**Files:**
- Create: `db/migrations/0003_add_engagement_foreign_keys.up.sql`
- Create: `db/migrations/0003_add_engagement_foreign_keys.down.sql`
- Modify: `db/SCHEMA.md`

**Step 1: Add orphan cleanup and FK constraints**

Ensure the migration is safe on existing deployed data.

**Step 2: Document the new migration in schema docs**

Explain why it exists and what it protects.

### Task 6: Update the shared review-fix document

**Files:**
- Modify: `docs/review_fix/2026-03-08-architecture-review.md`

**Step 1: Extend the document with the newly fixed review items**

Map foreign keys, download counting, like consistency, and length validation to their code changes.

### Task 7: Final verification

**Files:**
- Modify: all changed files from previous tasks

**Step 1: Run backend full test suite**

Run: `cd backend && go test ./...`
Expected: pass

**Step 2: Run frontend build**

Run: `cd frontend && npm run build`
Expected: pass

**Step 3: Run diff hygiene**

Run: `git diff --check`
Expected: pass

**Step 4: Confirm intended file set**

Run: `git status --short`
Expected: only intended files changed
