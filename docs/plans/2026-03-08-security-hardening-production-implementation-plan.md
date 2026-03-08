# Security Hardening Production Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace wildcard CORS, add production-grade security headers, and enforce rate limiting for sensitive backend routes while documenting the review fixes.

**Architecture:** Introduce config-driven middleware for CORS and security headers, add a limiter abstraction with Redis-first and in-memory fallback implementations, and wire route-specific policies in the Gin server. Validate behavior with focused middleware and server tests before final full-suite verification.

**Tech Stack:** Go, Gin, Redis, GORM, Node/Vite docs tooling only for existing frontend build verification

---

### Task 1: Prepare the worktree baseline

**Files:**
- Modify: `docs/plans/2026-03-08-security-hardening-production-design.md`
- Modify: `docs/plans/2026-03-08-security-hardening-production-implementation-plan.md`

**Step 1: Verify worktree baseline**

Run: `git status --short --branch`
Expected: branch `codex/security-hardening-production`

**Step 2: Run baseline backend verification**

Run: `cd backend && go test ./...`
Expected: current backend baseline status captured before changes

**Step 3: Run baseline frontend build verification**

Run: `cd frontend && npm run build`
Expected: current frontend baseline status captured before changes

### Task 2: Write failing CORS and security-header tests

**Files:**
- Create: `backend/internal/middleware/cors_test.go`
- Create: `backend/internal/middleware/security_headers_test.go`
- Modify: `backend/internal/config/config.go`

**Step 1: Write CORS allow/deny tests**

Cover:

- allowed origin receives reflected `Access-Control-Allow-Origin`
- disallowed origin receives `403`
- no-origin requests are allowed without CORS headers
- preflight request receives expected headers and `204`

**Step 2: Write security-header tests**

Cover:

- baseline headers always present
- CSP report-only mode uses the report-only header only
- HSTS appears only on secure requests when enabled

**Step 3: Run focused tests and confirm red**

Run: `cd backend && go test ./internal/middleware -run 'TestCORS|TestSecurityHeaders' -v`
Expected: fail because the new behavior does not exist yet

### Task 3: Write failing rate-limit tests

**Files:**
- Create: `backend/internal/middleware/rate_limit.go`
- Create: `backend/internal/middleware/rate_limit_test.go`

**Step 1: Write in-memory limiter tests**

Cover:

- requests under the limit pass
- next request returns `429`
- `Retry-After` is set
- limiter resets after window expiry

**Step 2: Write middleware identity tests**

Cover:

- user-ID keyed limiter works when auth context is present
- IP fallback works when auth context is absent

**Step 3: Run focused tests and confirm red**

Run: `cd backend && go test ./internal/middleware -run 'TestRateLimit' -v`
Expected: fail before implementation

### Task 4: Implement config and middleware

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/middleware/cors.go`
- Create: `backend/internal/middleware/security_headers.go`
- Create: `backend/internal/middleware/rate_limit.go`

**Step 1: Add config fields and parsing helpers**

Implement origin list parsing, CSP config, HSTS settings, and per-route rate-limit values with development-safe defaults.

**Step 2: Implement CORS allowlist middleware**

Support reflected allowlist origins, `Vary` headers, configurable methods/headers, and `403` for invalid cross-origin requests.

**Step 3: Implement security-header middleware**

Set baseline security headers, configurable CSP, report-only mode, and conditional HSTS.

**Step 4: Implement Redis-first / memory-fallback rate limiter**

Expose middleware constructors for route-specific keys and budgets with JSON `429` responses.

**Step 5: Re-run focused middleware tests**

Run: `cd backend && go test ./internal/middleware -v`
Expected: pass

### Task 5: Wire middleware into the server and prove route coverage

**Files:**
- Modify: `backend/cmd/server/main.go`
- Create: `backend/cmd/server/main_test.go`

**Step 1: Refactor server setup into a testable router builder**

Extract router construction from `main()` into a function that accepts config, dependencies, and optional limiter store.

**Step 2: Attach global security middleware**

Use CORS and security-header middleware on the Gin engine.

**Step 3: Attach route-specific limiters**

Guard login, register, review-retry, and AI chat routes with the correct policies.

**Step 4: Write router-level tests**

Verify:

- guarded routes return `429` when the limit is exceeded
- ordinary read routes are not globally limited
- CORS/security headers remain present on limited routes

**Step 5: Run server-focused tests**

Run: `cd backend && go test ./cmd/server -v`
Expected: pass

### Task 6: Add review-fix documentation

**Files:**
- Create: `docs/review_fix/2026-03-08-architecture-review.md`

**Step 1: Document findings addressed**

Map review items 1.1, 1.2, and 1.3 to concrete code changes.

**Step 2: Document operational configuration**

List required environment variables, recommended production values, and rollout cautions.

**Step 3: Document verification**

Include the exact commands used to verify the fix.

### Task 7: Full verification

**Files:**
- Modify: any touched files from previous tasks

**Step 1: Run backend full test suite**

Run: `cd backend && go test ./...`
Expected: pass

**Step 2: Run frontend build**

Run: `cd frontend && npm run build`
Expected: pass

**Step 3: Run diff hygiene checks**

Run: `git diff --check`
Expected: pass

**Step 4: Review changed files**

Run: `git status --short`
Expected: only intended files changed
