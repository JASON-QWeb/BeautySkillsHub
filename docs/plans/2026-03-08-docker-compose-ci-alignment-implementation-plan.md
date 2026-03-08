# Docker Compose And CI Alignment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align local Docker, runtime images, and CI verification so the full stack builds and starts cleanly from a fresh checkout without private local env files.

**Architecture:** Keep runtime hardening inside Docker artifacts, keep local compose zero-config by embedding safe defaults, and extend the existing Docker regression tests and GitHub Actions workflow so future runtime breakage is caught before merge.

**Tech Stack:** Docker Compose, nginx, GitHub Actions, Node test runner, Go, Vite

---

### Task 1: Lock down expected Docker behavior with failing tests

**Files:**
- Modify: `frontend/docker-runtime.test.mjs`

**Step 1: Write the failing test**

Add assertions for:
- compose does not reference `env_file`
- compose includes safe backend defaults for local boot
- workflow contains Docker verification steps

**Step 2: Run test to verify it fails**

Run: `node --test frontend/docker-runtime.test.mjs`

**Step 3: Write minimal implementation**

Adjust Docker and workflow files to satisfy the new assertions.

**Step 4: Run test to verify it passes**

Run: `node --test frontend/docker-runtime.test.mjs`

### Task 2: Remove private-file dependency from Docker Compose

**Files:**
- Modify: `docker-compose.yml`

**Step 1: Replace backend env file usage**

Remove `env_file` and provide explicit local-safe environment defaults with shell overrides.

**Step 2: Verify compose resolves cleanly**

Run: `docker compose config`

### Task 3: Align CI with current Docker expectations

**Files:**
- Modify: `.github/workflows/verify.yml`

**Step 1: Add Docker verification**

Add steps that build backend/frontend images and run a compose smoke test matching the local stack.

**Step 2: Verify workflow syntax through local commands**

Run:
- `docker build -f backend/Dockerfile backend`
- `docker build -f frontend/Dockerfile frontend`
- `docker compose up -d --build`

### Task 4: Update docs if Docker commands or assumptions changed

**Files:**
- Modify as needed: `README.md`, `DEVELOPMENT.md`, `DEPLOYMENT.md`

**Step 1: Sync docs**

Document that local compose is zero-config and no longer depends on `backend/.env.local`.

**Step 2: Verify links and examples**

Run: `git diff --check`

### Task 5: Full verification and integration

**Files:**
- No new code files

**Step 1: Run full verification**

Run:
- `node --test frontend/docker-runtime.test.mjs`
- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `docker build -f backend/Dockerfile backend`
- `docker build -f frontend/Dockerfile frontend`
- `docker compose up -d --build`
- `docker compose ps`
- `curl -sf http://127.0.0.1:8080/health`
- `curl -I http://127.0.0.1:5173`

**Step 2: Commit and push**

Commit only the intended Docker, workflow, and docs changes on `main`, then push to `origin/main`.
