# Local Docker Stack Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `docker compose up -d --build` start the full local stack and document both Docker and host-process workflows.

**Architecture:** Keep `infra/docker/compose.local.yml` as the db-only path used by existing scripts, while upgrading the root `docker-compose.yml` into a full local stack with `postgres`, `redis`, `migrate`, `backend`, and `frontend`. Frontend stays behind the existing Nginx proxy, backend keeps runtime env injection, and seed remains manual.

**Tech Stack:** Docker Compose, PostgreSQL, Redis, Go backend, React/Vite frontend, Nginx

---

### Task 1: Full Local Compose Stack

**Files:**
- Modify: `docker-compose.yml`
- Modify: `frontend/Dockerfile`

**Step 1: Validate current failure**
Run: `docker build -f backend/Dockerfile backend`
Expected: PASS after the earlier `.env.example` fix, confirming the image base is stable before compose changes.

**Step 2: Add full-stack services**
- Extend root compose to include `migrate`, `backend`, and `frontend`.
- Keep `postgres` and `redis` with health checks.
- Make `backend` wait for successful migration completion.
- Make `frontend` build with `/api` as its base URL so Nginx proxying works inside the container path.

**Step 3: Validate compose wiring**
Run: `docker compose config`
Expected: PASS with all five services rendered.

### Task 2: Documentation And Manual Seed Flow

**Files:**
- Modify: `README.md`

**Step 1: Rewrite local startup section**
- Document Docker Compose full-stack startup.
- Keep the existing db-only and host-process flows.
- Add the manual seed command for the compose path.

**Step 2: Verify docs references**
Run: `rg -n 'backend/.env.example|docker compose up -d --build|seed' README.md`
Expected: no stale `backend/.env.example` references, and the new startup flow documented.

### Task 3: End-To-End Verification

**Files:**
- Modify: none

**Step 1: Run backend and frontend verification**
Run: `cd backend && go test ./...` and `cd frontend && npm run build`
Expected: PASS

**Step 2: Run compose stack**
Run: `docker compose up -d --build`
Expected: `postgres`, `redis`, `backend`, `frontend` running and `migrate` completed successfully.

**Step 3: Verify runtime**
Run: `docker compose ps` and `curl -fsS http://127.0.0.1:8080/api/skills` and `curl -I http://127.0.0.1:5173`
Expected: backend returns `200`, frontend returns `200`, services are healthy.
