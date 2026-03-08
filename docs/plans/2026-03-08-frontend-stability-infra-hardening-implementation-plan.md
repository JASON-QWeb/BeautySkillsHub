# Frontend Stability And Infrastructure Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the reviewed frontend stability issues and backend runtime hardening issues in one isolated branch without regressing existing behavior.

**Architecture:** Introduce a shared frontend request/session layer, move Profile onto `me`-scoped paginated data, add bounded cache and UI resilience primitives, and harden backend startup/runtime requirements for production environments.

**Tech Stack:** React 18, TypeScript, Vite, Gin, Gorm, PostgreSQL, Docker.

---

### Task 1: Add frontend request/session red tests

**Files:**
- Create: `frontend/src/services/api/request.test.ts`
- Create: `frontend/src/services/api/request.ts`
- Modify: `frontend/src/contexts/AuthContext.tsx`

**Step 1: Write the failing test**

Cover:

- token header generation
- expired JWT detection
- unauthorized handler invocation on `401`
- no unauthorized callback for anonymous requests

**Step 2: Run test to verify it fails**

Run: `cd frontend && node --test src/services/api/request.test.ts`

Expected: failures for missing request helper exports

**Step 3: Write minimal implementation**

- Add shared request helper
- Add token parsing helper
- Add unauthorized handler registration

**Step 4: Run test to verify it passes**

Run: `cd frontend && node --test src/services/api/request.test.ts`

**Step 5: Integrate into auth and API modules**

- Replace duplicated `getAuthHeaders()`
- Update `AuthContext` to use token expiry pre-check and unauthorized logout

---

### Task 2: Add failing test for bounded README cache

**Files:**
- Create: `frontend/src/features/skill-detail/readmeCache.test.ts`
- Create: `frontend/src/features/skill-detail/readmeCache.ts`
- Modify: `frontend/src/features/skill-detail/SkillDetailPage.tsx`

**Step 1: Write the failing test**

Cover:

- cache stores by id
- hit refreshes recency
- cache evicts least-recently-used item after limit

**Step 2: Run test to verify it fails**

Run: `cd frontend && node --test src/features/skill-detail/readmeCache.test.ts`

**Step 3: Write minimal implementation**

- Introduce LRU cache helper
- Rewire detail page to use helper instead of raw `Map`

**Step 4: Run test to verify it passes**

Run: `cd frontend && node --test src/features/skill-detail/readmeCache.test.ts`

---

### Task 3: Add failing frontend behavior tests/helpers for dialog and AI mouse scheduling

**Files:**
- Create: `frontend/src/contexts/dialogKeyboard.test.ts`
- Create: `frontend/src/components/aiMouseTracking.test.ts`
- Create or modify helper files near `DialogContext` / `AIChatCharacter`
- Modify: `frontend/src/contexts/DialogContext.tsx`
- Modify: `frontend/src/components/AIChatCharacter.tsx`

**Step 1: Write failing tests**

Cover:

- `Escape` closes dialog with cancel semantics
- mouse scheduler coalesces many events into one frame update

**Step 2: Run failing tests**

Run: `cd frontend && node --test src/contexts/dialogKeyboard.test.ts src/components/aiMouseTracking.test.ts`

**Step 3: Implement minimal helpers and wire components**

**Step 4: Re-run tests**

Run: `cd frontend && node --test src/contexts/dialogKeyboard.test.ts src/components/aiMouseTracking.test.ts`

---

### Task 4: Add failing backend tests for secure config and health endpoint

**Files:**
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/internal/handler/auth_test.go`
- Modify: `backend/cmd/server/security_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/handler/auth.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: Write the failing test**

Cover:

- non-local environment rejects missing or insecure `DATABASE_URL`
- non-local environment rejects missing `JWT_SECRET`
- `/health` responds successfully when router is wired

**Step 2: Run failing tests**

Run: `cd backend && go test ./internal/config ./internal/handler ./cmd/server -run 'Security|Health|JWT' -v`

**Step 3: Write minimal implementation**

- Add config validation helpers
- Enforce runtime checks
- Add `/health`

**Step 4: Re-run tests**

Run: `cd backend && go test ./internal/config ./internal/handler ./cmd/server -run 'Security|Health|JWT' -v`

---

### Task 5: Move profile to paginated me-scoped data

**Files:**
- Modify: `backend/internal/service/skill.go`
- Add/Modify: `backend/internal/handler/*profile*` or existing handlers
- Modify: `frontend/src/services/api/skills.ts`
- Modify: `frontend/src/services/api/types.ts`
- Modify: `frontend/src/features/profile/ProfilePage.tsx`
- Add tests in backend service/handler as needed

**Step 1: Write failing tests**

Cover:

- backend returns only current user uploads
- response includes totals for downloads/likes/items
- recent activity list is limited and ordered

**Step 2: Run tests to verify failures**

Run: targeted `go test` commands for new handler/service tests

**Step 3: Implement backend endpoint and frontend pagination state**

- add `me` endpoint
- page through uploads
- keep stats accurate without scanning 500 public rows on frontend

**Step 4: Re-run tests**

Run: targeted backend tests, then `cd frontend && npm run build`

---

### Task 6: Wire abortable requests through key pages

**Files:**
- Modify: `frontend/src/services/api/skills.ts`
- Modify: `frontend/src/services/api/content-assets.ts`
- Modify: `frontend/src/services/api/ai.ts`
- Modify: `frontend/src/contexts/AuthContext.tsx`
- Modify: `frontend/src/features/skill-detail/SkillDetailPage.tsx`
- Modify: `frontend/src/features/profile/ProfilePage.tsx`
- Modify: `frontend/src/features/upload/UploadPage.tsx`
- Modify: `frontend/src/features/upload/rules/RulesUploadPage.tsx`
- Modify: `frontend/src/features/upload/mcp/McpUploadPage.tsx`
- Modify: `frontend/src/features/upload/tools/ToolsUploadPage.tsx`
- Modify: `frontend/src/features/home/HomePage.tsx`
- Modify: `frontend/src/components/RightSidebar.tsx`

**Step 1: Add/extend failing tests for request helper signal propagation where feasible**

**Step 2: Implement minimal signal support in API functions**

**Step 3: Attach `AbortController` to lifecycle-driven requests**

**Step 4: Verify**

Run:

- targeted frontend tests
- `cd frontend && npm run build`

---

### Task 7: Add global error boundary

**Files:**
- Create: `frontend/src/components/AppErrorBoundary.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Write a failing test if a lightweight harness is practical; otherwise document why build-level verification is used**

**Step 2: Implement minimal error boundary**

**Step 3: Verify**

Run: `cd frontend && npm run build`

---

### Task 8: Harden Docker images to run as non-root

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `frontend/Dockerfile`

**Step 1: Update images**

- create non-root user/group
- ensure copied files and runtime directories are owned correctly
- switch final stage to `USER`

**Step 2: Verify**

If Docker is available:

```bash
docker build -f backend/Dockerfile backend
docker build -f frontend/Dockerfile frontend
```

If not, record that runtime image build verification could not be executed.

---

### Task 9: Write review fix documentation

**Files:**
- Create: `docs/review_fix/2026-03-08-frontend-and-infra-review-fixes.md`

**Step 1: Summarize each fixed review item**

**Step 2: Record commands actually used for verification**

---

### Task 10: Final verification

**Files:**
- Verify whole tree only

**Step 1: Run frontend tests**

Run the explicit node tests added in this branch.

**Step 2: Run backend full tests**

Run: `cd backend && go test ./...`

**Step 3: Run frontend build**

Run: `cd frontend && npm run build`

**Step 4: Run diff hygiene**

Run: `git diff --check`

**Step 5: Commit**

Commit in logical chunks after verification passes.
