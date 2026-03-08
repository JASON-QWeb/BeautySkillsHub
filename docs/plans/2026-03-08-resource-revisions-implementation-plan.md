# Resource Revisions Review Flow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a separate revision model so edits for `skill`, `rules`, `mcp`, and `tools` create or update a single pending revision and only overwrite the published resource after review approval.

**Architecture:** Keep `skills` as the published record and add `skill_revisions` for in-flight updates. Route all edit submissions through revision APIs, surface pending revision state on detail pages, and apply revision content back onto `skills` only after approval.

**Tech Stack:** Go, Gin, GORM, PostgreSQL, React, TypeScript, Vite

---

### Task 1: Add revision schema and model

**Files:**
- Create: `backend/internal/model/skill_revision.go`
- Modify: `db/migrations/0002_add_skill_revisions.up.sql`
- Modify: `db/migrations/0002_add_skill_revisions.down.sql`
- Test: `backend/internal/model` via integration usage in service tests

**Step 1: Write the failing test**
- Add a backend service test that expects creating an edit to write a row into `skill_revisions` and not create a second row in `skills`.

**Step 2: Run test to verify it fails**
Run: `cd backend && go test ./internal/service -run TestSkillRevision`  
Expected: FAIL because revision model/table does not exist.

**Step 3: Write minimal implementation**
- Add `SkillRevision` model.
- Add SQL migration creating `skill_revisions` and a partial unique index for one active revision per `skill_id`.

**Step 4: Run test to verify it passes**
Run: `cd backend && go test ./internal/service -run TestSkillRevision`
Expected: PASS.

**Step 5: Commit**
```bash
git add backend/internal/model/skill_revision.go db/migrations/0002_add_skill_revisions.up.sql db/migrations/0002_add_skill_revisions.down.sql
git commit -m "feat: add skill revision schema"
```

### Task 2: Add backend revision service behavior

**Files:**
- Modify: `backend/internal/service/skill.go`
- Create: `backend/internal/service/skill_revision_test.go`
- Modify: `backend/internal/model/skill.go` if revision summary fields are exposed via DTO helpers

**Step 1: Write the failing test**
- Test that editing a published skill creates or updates one pending revision.
- Test that a second concurrent update attempt returns a conflict.
- Test that approval applies the revision onto the published row and preserves counters.

**Step 2: Run test to verify it fails**
Run: `cd backend && go test ./internal/service -run 'TestSkillRevision|TestApplySkillRevision'`
Expected: FAIL because service methods do not exist.

**Step 3: Write minimal implementation**
- Add service methods to create/update active revisions.
- Add methods to fetch revision summaries and apply approved revisions.
- Preserve downloads/likes/favorites by mutating only content fields on `skills`.

**Step 4: Run test to verify it passes**
Run: `cd backend && go test ./internal/service -run 'TestSkillRevision|TestApplySkillRevision'`
Expected: PASS.

**Step 5: Commit**
```bash
git add backend/internal/service/skill.go backend/internal/service/skill_revision_test.go backend/internal/model/skill.go
git commit -m "feat: add revision service flow"
```

### Task 3: Route review pipeline through revisions

**Files:**
- Modify: `backend/internal/handler/skill_review_handlers.go`
- Modify: `backend/internal/handler/skill_human_review_handlers.go`
- Modify: `backend/internal/handler/resource_handler.go`
- Modify: `backend/internal/handler/skill_handler.go`
- Test: `backend/internal/handler/skill_review_handlers_test.go`
- Test: `backend/internal/handler/skill_permissions_test.go`

**Step 1: Write the failing test**
- Add handler tests that approve a pending revision and expect the published row to be overwritten without changing ID.
- Add tests that detail/review endpoints target the revision when one exists.

**Step 2: Run test to verify it fails**
Run: `cd backend && go test ./internal/handler -run 'TestSkillRevision|TestReviewAppliesRevision'`
Expected: FAIL because handlers still operate on `skills` directly.

**Step 3: Write minimal implementation**
- Load pending revision state for review flows.
- On approval, call the apply service.
- On rejection, keep the published resource unchanged and mark the revision rejected.

**Step 4: Run test to verify it passes**
Run: `cd backend && go test ./internal/handler -run 'TestSkillRevision|TestReviewAppliesRevision'`
Expected: PASS.

**Step 5: Commit**
```bash
git add backend/internal/handler/skill_review_handlers.go backend/internal/handler/skill_human_review_handlers.go backend/internal/handler/resource_handler.go backend/internal/handler/skill_handler.go backend/internal/handler/skill_review_handlers_test.go backend/internal/handler/skill_permissions_test.go
git commit -m "feat: review pending revisions"
```

### Task 4: Add revision endpoints for edit forms

**Files:**
- Modify: `backend/internal/handler/skill_update_handlers.go`
- Modify: `backend/internal/handler/resource_handler.go`
- Test: `backend/internal/handler/skill_upload_handlers_test.go`
- Test: `backend/internal/handler/resource_handler_test.go`

**Step 1: Write the failing test**
- Add tests showing `PUT` for `skill/rules/mcp/tools` creates or updates a revision instead of mutating or inserting the published row.

**Step 2: Run test to verify it fails**
Run: `cd backend && go test ./internal/handler -run 'TestUpdateCreatesRevision|TestResourceUpdateCreatesRevision'`
Expected: FAIL because current handlers write directly to the published row or still create a new row.

**Step 3: Write minimal implementation**
- Change update handlers to write to revisions.
- Support multipart edit flows for skill/rules file, folder, paste, and thumbnail updates.
- Return conflict when an existing active revision belongs to the same resource and a second update creation is attempted.

**Step 4: Run test to verify it passes**
Run: `cd backend && go test ./internal/handler -run 'TestUpdateCreatesRevision|TestResourceUpdateCreatesRevision'`
Expected: PASS.

**Step 5: Commit**
```bash
git add backend/internal/handler/skill_update_handlers.go backend/internal/handler/resource_handler.go backend/internal/handler/skill_upload_handlers_test.go backend/internal/handler/resource_handler_test.go
git commit -m "feat: route edits into revisions"
```

### Task 5: Update frontend API and edit pages

**Files:**
- Modify: `frontend/src/services/api/skills.ts`
- Modify: `frontend/src/services/api/types.ts`
- Modify: `frontend/src/features/upload/UploadPage.tsx`
- Modify: `frontend/src/features/upload/rules/RulesUploadPage.tsx`
- Modify: `frontend/src/features/upload/mcp/McpUploadPage.tsx`
- Modify: `frontend/src/features/upload/tools/ToolsUploadPage.tsx`
- Test: `frontend/src/features/upload` tests or targeted new tests

**Step 1: Write the failing test**
- Add frontend tests that entering `?edit=` for `skill/rules` uses edit mode.
- Add tests that edit submission calls revision/update APIs, not upload `POST`.
- Add regression tests for `mcp/tools` to ensure they still use update flow.

**Step 2: Run test to verify it fails**
Run: `cd frontend && npm test -- --runInBand revision` or the repo’s chosen frontend test command
Expected: FAIL because edit mode is missing or wrong.

**Step 3: Write minimal implementation**
- Add revision-aware API helpers.
- Add `useSearchParams` edit mode to skill/rules pages.
- Load pending revision when present, else published content.
- Submit edits through update endpoints for all resource types.

**Step 4: Run test to verify it passes**
Run: `cd frontend && npm test -- --runInBand revision`
Expected: PASS.

**Step 5: Commit**
```bash
git add frontend/src/services/api/skills.ts frontend/src/services/api/types.ts frontend/src/features/upload/UploadPage.tsx frontend/src/features/upload/rules/RulesUploadPage.tsx frontend/src/features/upload/mcp/McpUploadPage.tsx frontend/src/features/upload/tools/ToolsUploadPage.tsx
git commit -m "feat: add revision-aware edit pages"
```

### Task 6: Update detail and card state for pending revisions

**Files:**
- Modify: `frontend/src/features/skill-detail/SkillDetailPage.tsx`
- Modify: `frontend/src/components/SkillCard.tsx`
- Modify: `frontend/src/features/review/ReviewPage.tsx`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: corresponding frontend tests

**Step 1: Write the failing test**
- Add tests that detail pages continue showing published content but surface pending revision status.
- Add tests that the edit button becomes `更新中 / 待 Review` and that review CTA is present.

**Step 2: Run test to verify it fails**
Run: `cd frontend && npm test -- --runInBand revision-detail`
Expected: FAIL because UI state does not exist.

**Step 3: Write minimal implementation**
- Extend detail payload handling with pending revision summary.
- Update card/detail buttons and labels.
- Ensure card click still goes to detail.

**Step 4: Run test to verify it passes**
Run: `cd frontend && npm test -- --runInBand revision-detail`
Expected: PASS.

**Step 5: Commit**
```bash
git add frontend/src/features/skill-detail/SkillDetailPage.tsx frontend/src/components/SkillCard.tsx frontend/src/features/review/ReviewPage.tsx frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: show pending revision state"
```

### Task 7: Refresh docs and run full verification

**Files:**
- Modify: `README.md`
- Modify: `ARCHITECTURE.md`
- Modify: `db/SCHEMA.md`
- Modify: `DEPLOYMENT.md` if review flow behavior changes operationally

**Step 1: Update docs**
- Document published resource vs pending revision model.
- Document one-active-revision behavior.

**Step 2: Run backend verification**
Run: `./scripts/db-local.sh && ./scripts/run-all-migrations.sh && cd backend && go test ./...`
Expected: all backend tests pass.

**Step 3: Run frontend verification**
Run: `cd frontend && npm run build`
Expected: build passes.

**Step 4: Run compose verification**
Run: `docker compose up -d --build && docker compose ps && docker compose down`
Expected: all services healthy.

**Step 5: Commit**
```bash
git add README.md ARCHITECTURE.md db/SCHEMA.md DEPLOYMENT.md
git commit -m "docs: describe revision review flow"
```
