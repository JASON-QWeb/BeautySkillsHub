# Docs Refresh And Push Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refresh the project documentation so README, architecture, and deployment docs are directly actionable, then verify and push the changes to `origin/main`.

**Architecture:** Rewrite the root docs around three user journeys: understand the system, start it locally, and deploy it safely. Keep the repo's PostgreSQL migration-first model central in all docs, and align deployment instructions with the code that actually exists in this repository.

**Tech Stack:** Markdown, Git, GitHub, Go, Vite, PostgreSQL, Docker, Nginx

---

### Task 1: Refresh README for local startup and navigation

**Files:**
- Modify: `README.md`
- Check: `backend/.env.local.example`
- Check: `frontend/.env.local.example`
- Check: `scripts/db-local.sh`
- Check: `scripts/run-all-migrations.sh`

**Step 1: Write the failing test**

Identify gaps where README is technically correct but not optimized for fast onboarding.

**Step 2: Run inspection to verify the gap**

Run: `sed -n '1,260p' README.md`
Expected: content exists but needs stronger quick-start and doc navigation structure.

**Step 3: Write minimal implementation**

Rewrite README to highlight:
- project overview
- quickest local startup path
- startup commands for db/backend/frontend
- testing path
- where to find architecture, deployment, and GitHub Actions docs

**Step 4: Verify result**

Run: `sed -n '1,260p' README.md`
Expected: clearer quick-start and navigation.

### Task 2: Refresh architecture doc

**Files:**
- Modify: `ARCHITECTURE.md`
- Check: `infra/docker/compose.local.yml`
- Check: `backend/cmd/server/main.go`
- Check: `db/migrations/0001_init_schema.up.sql`

**Step 1: Write the failing test**

Identify where architecture doc lacks sufficient detail about boundaries, runtime topology, and startup flows.

**Step 2: Run inspection to verify the gap**

Run: `sed -n '1,260p' ARCHITECTURE.md`
Expected: content exists but can be expanded into a more operational architecture guide.

**Step 3: Write minimal implementation**

Update architecture doc to explain:
- repository layout and responsibilities
- local vs shared/prod topology
- request/data flow
- configuration model
- testing boundary and CI boundary

**Step 4: Verify result**

Run: `sed -n '1,260p' ARCHITECTURE.md`
Expected: complete project architecture explanation.

### Task 3: Refresh deployment doc with actionable steps

**Files:**
- Modify: `DEPLOYMENT.md`
- Check: `backend/Dockerfile`
- Check: `frontend/Dockerfile`
- Check: `frontend/nginx.conf`
- Check: `.github/workflows/verify.yml`

**Step 1: Write the failing test**

Identify where deployment doc explains principles but still needs more concrete runbook-style instructions.

**Step 2: Run inspection to verify the gap**

Run: `sed -n '1,320p' DEPLOYMENT.md`
Expected: principle-focused doc that can be made more directly actionable.

**Step 3: Write minimal implementation**

Rewrite deployment doc to include:
- prerequisites
- environment preparation
- PostgreSQL setup assumptions
- migration commands
- backend startup options
- frontend delivery options
- verification checklist
- upgrade path and rollback notes

**Step 4: Verify result**

Run: `sed -n '1,320p' DEPLOYMENT.md`
Expected: actionable deployment runbook.

### Task 4: Re-run repository verification and push

**Files:**
- Check: root docs and current `main`

**Step 1: Run verification**

Run:
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`
- `cd backend && go test ./...`
- `cd frontend && npm run build`

**Step 2: Commit docs update**

Create a non-interactive commit for doc refresh.

**Step 3: Push to origin**

Run: `git push origin main`

**Step 4: Verify push result**

Run: `git log --oneline -1`
Expected: pushed commit at HEAD.
