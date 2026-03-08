# Multi-Resource Upload + Detail Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build independent upload/detail experiences for `skills`, `rules`, `mcp`, and `tools`, with route-driven behavior and cleaner modular frontend architecture.

**Architecture:** Keep one shared resource table and common API shape, but introduce per-resource policy (review vs no-review) and per-resource frontend feature modules. `skills` remains current UX, `rules` reuses review pipeline, `mcp/tools` become article-like publishing with long markdown body + embedded image assets + optional source link/package. Add compatibility redirects to avoid breaking existing URLs.

**Tech Stack:** React + TypeScript + React Router + Vite; Gin + GORM + SQLite.

---

### Task 1: Route Contract and Compatibility Layer

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/features/home/HomePage.tsx`
- Modify: `frontend/src/components/Navbar.tsx`

**Steps:**
1. Add new upload routes: `/resource/skill/upload`, `/resource/rules/upload`, `/resource/mcp/upload`, `/resource/tools/upload`.
2. Keep `/upload` as compatibility entry and redirect by query/default to `/resource/:type/upload`.
3. Keep existing detail route `/skill/:id` working; introduce typed detail route `/resource/:type/:id` and redirect old route after resolving resource type.
4. Verify navigation from each resource list page opens matching upload route.

**Verify:**
- `cd frontend && npm run build`
- Manual: from each `/resource/:type` click upload and verify URL type consistency.

---

### Task 2: Frontend Upload Module Restructure

**Files:**
- Create: `frontend/src/features/upload/shared/UploadShell.tsx`
- Create: `frontend/src/features/upload/shared/types.ts`
- Create: `frontend/src/features/upload/shared/RichMarkdownEditor.tsx`
- Create: `frontend/src/features/upload/shared/FileDropzone.tsx`
- Create: `frontend/src/features/upload/shared/upload.css`
- Create: `frontend/src/features/upload/index/ResourceUploadRouter.tsx`
- Modify: `frontend/src/pages/UploadPage.tsx`

**Steps:**
1. Split current monolithic upload page into shared shell + per-resource pages.
2. Move generic state/validation helpers into shared module.
3. Keep `skills` page visually unchanged by wrapping current implementation as `SkillUploadPage`.

**Verify:**
- `cd frontend && npm run build`
- Manual: visual parity for skills upload page.

---

### Task 3: Skills Upload/Detail Lockdown (No Behavior Drift)

**Files:**
- Create: `frontend/src/features/upload/skill/SkillUploadPage.tsx`
- Create: `frontend/src/features/detail/skill/SkillDetailPage.tsx`
- Modify: route bindings only

**Steps:**
1. Move current skills upload/detail implementation into `skill` modules.
2. Ensure styles and UX stay exactly as current baseline.

**Verify:**
- Regression checklist: upload folder, AI review polling, human review waiting, detail AI summary/sidebar.

---

### Task 4: Rules Backend Policy (AI + Human Review Required)

**Files:**
- Modify: `backend/internal/handler/resource_handler.go`
- Modify: `backend/cmd/server/main.go`
- Create: `backend/internal/handler/resource_review_handlers.go` (or extract reusable helpers)
- Modify: `backend/internal/service/skill.go` (if policy helper needed)
- Create/Modify tests: `backend/internal/handler/*_test.go`

**Steps:**
1. Introduce per-resource review policy map:
   - `skill`: AI review + human review
   - `rules`: AI review + human review
   - `mcp/tools`: no review
2. Expose rules review endpoints analogous to skills:
   - `GET /api/rules/:id/review-status`
   - `POST /api/rules/:id/review/retry`
   - `POST /api/rules/:id/human-review`
3. Ensure list visibility logic matches pending/approved states for rules.

**Verify:**
- `cd backend && go test ./internal/handler`
- Manual API: create rule -> queued/running/passed -> human review pending.

---

### Task 5: Rules Upload UX (File/Paste, Strict File Types)

**Files:**
- Create: `frontend/src/features/upload/rules/RulesUploadPage.tsx`
- Modify: `frontend/src/services/api/skills.ts` (or split upload API by resource)
- Modify: `backend/internal/handler/resource_handler.go` (rules upload validation)
- Modify: i18n keys

**Steps:**
1. Rules upload supports:
   - upload `.md` or `.txt`
   - or paste markdown text directly
2. If paste mode selected, backend stores generated markdown file in upload session.
3. Rules upload enters AI review + human review flow (same pattern as skills).

**Verify:**
- Backend tests for extension validation and paste-mode persistence.
- Frontend manual: file mode/paste mode both create reviewable rule.

---

### Task 6: Content Asset Pipeline for Long Article Body (MCP/Tools)

**Files:**
- Create: `backend/internal/handler/content_asset_handlers.go`
- Create: `backend/internal/model/content_asset.go` (if separate table needed)
- Modify: `backend/cmd/server/main.go`
- Create: `frontend/src/services/api/content-assets.ts`
- Modify: `frontend/src/features/upload/shared/RichMarkdownEditor.tsx`

**Steps:**
1. Add image upload endpoint for markdown content embedding (e.g. `POST /api/content-assets/images`).
2. Save uploaded images under dedicated directory and serve by URL.
3. Editor supports image upload and inserts markdown image syntax automatically.
4. Add size/type validation and ownership safety checks.

**Verify:**
- Upload image from editor -> markdown contains URL -> detail page renders image.
- `cd backend && go test ./...`

---

### Task 7: MCP Upload UX (Single Large Form, No Steps/Review)

**Files:**
- Create: `frontend/src/features/upload/mcp/McpUploadPage.tsx`
- Modify: backend upload handler for `mcp` metadata fields
- Modify: API types (`frontend/src/services/api/types.ts`, backend model if needed)

**Steps:**
1. Remove step/review UI for MCP upload.
2. Use long markdown body input with preview and embedded image support (via Task 6).
3. Add optional GitHub URL field with backend validation.
4. Publish directly (no AI/human review).

**Verify:**
- Upload MCP with long markdown + embedded image + github link, check direct visibility and rendered content.

---

### Task 8: Tools Upload UX (Single Form, Long Markdown, Archive Support, No Review)

**Files:**
- Create: `frontend/src/features/upload/tools/ToolsUploadPage.tsx`
- Modify: backend tools upload validation (zip/tar/gz etc)
- Modify: download metadata wiring

**Steps:**
1. Remove step/review UI for tools upload.
2. Long markdown body input with preview and embedded image support.
3. Support archive upload and direct publish.

**Verify:**
- Upload archive tool and confirm downloadable artifact from detail page.

---

### Task 9: Detail Page Refactor by Resource Type

**Files:**
- Create: `frontend/src/features/detail/shared/ResourceDetailShell.tsx`
- Create: `frontend/src/features/detail/rules/RulesDetailPage.tsx`
- Create: `frontend/src/features/detail/mcp/McpDetailPage.tsx`
- Create: `frontend/src/features/detail/tools/ToolsDetailPage.tsx`
- Modify: `frontend/src/features/skill-detail/SkillDetailPage.tsx` (move to skill module)

**Steps:**
1. Skills detail: unchanged.
2. Rules detail: remove terminal/paste-like box under title.
3. MCP detail: replace terminal area with GitHub link panel; main content renders long markdown body (images supported).
4. Tools detail: remove terminal area; main content renders long markdown body (images supported); show archive download if present.

**Verify:**
- Manual check each type on desktop/mobile.

---

### Task 10: API and Model Field Normalization

**Files:**
- Modify: `backend/internal/model/skill.go`
- Modify: `backend/internal/handler/*`
- Modify: `frontend/src/services/api/types.ts`

**Steps:**
1. Add/normalize optional fields used by MCP/Tools detail (e.g., `source_url`, `body_markdown`, `attachment_file_name`, `attachment_file_path`).
2. Ensure backward-compatible JSON fields for existing data.
3. Add migration/backfill logic for nullable new fields.

**Verify:**
- `cd backend && go test ./...`
- Existing skills data remains readable.

---

### Task 11: Styling and Folder Hygiene

**Files:**
- Create/Move: `frontend/src/styles/upload/{skill,rules,mcp,tools}.css`
- Create/Move: `frontend/src/styles/detail/{skill,rules,mcp,tools}.css`
- Modify: `frontend/src/styles/index.css`

**Steps:**
1. Split style concerns by feature and resource type.
2. Remove dead/legacy upload style files after migration.
3. Keep a shared token layer + shared component style layer.

**Verify:**
- `cd frontend && npm run build`
- no missing class regressions.

---

### Task 12: End-to-End Verification Matrix

**Files:**
- Create: `docs/testing/resource-upload-detail-matrix.md`

**Steps:**
1. Build test matrix across 4 types and all requested behaviors.
2. Include route, upload mode, review expectation, detail rendering, download behavior.

**Verify commands:**
- `cd backend && go test ./...`
- `cd frontend && npm run build`

---

### Task 13: Rollout and Risk Controls

**Files:**
- Update: `README.md` (route and behavior docs)
- Update: deployment notes if API contract changed

**Steps:**
1. Ship in 3 batches to reduce regression risk:
   - Batch A: route + frontend module split (no behavior changes)
   - Batch B: rules review backend + rules UI
   - Batch C: mcp/tools upload+detail specialization
2. Keep legacy routes during rollout; remove only after validation.
