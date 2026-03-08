# Upload Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Harden multipart upload handling against oversized request bodies, prevent thumbnail filename collisions, and keep local build artifacts out of git.

**Architecture:** Add a reusable multipart request-limiting middleware in the backend middleware layer, then attach it only to routes that may parse multipart forms. Keep business-level file and thumbnail validation inside handlers, but move request-body enforcement to pre-handler parsing. Thumbnail persistence will keep the current slug-based naming while adding a random suffix to avoid collisions across resources with the same title.

**Tech Stack:** Go, Gin, net/http, existing backend handler tests, git ignore rules

---

### Task 1: Multipart Request Limit Middleware

**Files:**
- Create: `backend/internal/middleware/request_size_test.go`
- Create: `backend/internal/middleware/request_size.go`

**Step 1: Write the failing test**

Add tests that:
- send an oversized multipart request through a test router using a new middleware function,
- expect `413 Request Entity Too Large`,
- verify a normal multipart request still reaches the handler and can read `FormFile`.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware -run 'TestLimitMultipartBody' -count=1`
Expected: FAIL because the middleware function does not exist yet.

**Step 3: Write minimal implementation**

Implement a Gin middleware that:
- wraps `c.Request.Body` with `http.MaxBytesReader`,
- parses multipart requests early,
- aborts with `413` on `MaxBytesError`,
- cleans up `MultipartForm` temp files after `c.Next()`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware -run 'TestLimitMultipartBody' -count=1`
Expected: PASS

### Task 2: Unique Thumbnail Storage

**Files:**
- Modify: `backend/internal/handler/skill_upload_handlers_test.go`
- Modify: `backend/internal/handler/skill_upload_handlers.go`

**Step 1: Write the failing test**

Add a test that saves two thumbnails for the same resource name and asserts:
- the stored filenames differ,
- both files exist on disk,
- filenames still end with `"_thumb.<ext>"`.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler -run 'TestSaveUploadedThumbnail' -count=1`
Expected: FAIL because the current implementation produces the same filename.

**Step 3: Write minimal implementation**

Update thumbnail naming so the persisted filename becomes `slug-random_thumb.ext`, reusing the existing random token helper.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler -run 'TestSaveUploadedThumbnail' -count=1`
Expected: PASS

### Task 3: Route Wiring And Ignore Rules

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handler/skill_upload_handlers.go`
- Modify: `backend/internal/handler/resource_handler.go`
- Modify: `.gitignore`

**Step 1: Write the failing test**

Use the middleware tests from Task 1 and thumbnail tests from Task 2 as the regression safety net for this wiring step.

**Step 2: Run test to verify baseline**

Run: `go test ./internal/middleware ./internal/handler -run 'TestLimitMultipartBody|TestSaveUploadedThumbnail' -count=1`
Expected: PASS once Tasks 1 and 2 are complete.

**Step 3: Write minimal implementation**

- Export route-level request size constants from the handler layer.
- Attach the multipart middleware to reviewed upload routes, resource create/update routes, and content image upload.
- Expand `.gitignore` to cover tracked local artifacts such as `backend/server` and `frontend/tsconfig.tsbuildinfo`.

**Step 4: Run targeted verification**

Run: `go test ./internal/middleware ./internal/handler -run 'TestLimitMultipartBody|TestSaveUploadedThumbnail' -count=1`
Expected: PASS

### Task 4: Full Verification

**Files:**
- Modify: none

**Step 1: Run backend tests**

Run: `go test ./...`
Expected: PASS

**Step 2: Run frontend build**

Run: `npm run build`
Workdir: `frontend`
Expected: PASS

**Step 3: Remove tracked local artifacts from git index**

Run: `git rm --cached -- backend/server frontend/tsconfig.tsbuildinfo`
Expected: staged index removal while files remain on disk
