# GitHub Actions Merge-Back Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a real GitHub Actions verification workflow, complete the related documentation, then merge the PostgreSQL migration worktree back into `main` with full verification before and after merge.

**Architecture:** Keep CI narrowly focused on verification: a backend job with PostgreSQL service plus migration-first test execution, and a frontend job that builds static assets. Complete the docs so local workflow, GitHub Actions, and enterprise CI/CD template stay consistent, then merge the branch into `main` with non-destructive conflict resolution.

**Tech Stack:** GitHub Actions, PostgreSQL service containers, Go, Node.js, Docker Compose local scripts, Git

---

### Task 1: Add GitHub Actions verification workflow

**Files:**
- Create: `.github/workflows/verify.yml`
- Check: `backend/go.mod`
- Check: `frontend/package-lock.json`
- Check: `scripts/run-all-migrations.sh`

**Step 1: Write the failing test**

Create the workflow file reference target and plan for:
- `push` and `pull_request`
- backend job with PostgreSQL service, migration run, and `go test`
- frontend job with `npm ci` and `npm run build`

**Step 2: Run validation to verify initial failure/gap**

Run:
- `test -f .github/workflows/verify.yml && echo exists || echo missing`
Expected: `missing` before implementation.

**Step 3: Write minimal implementation**

Add workflow that:
- checks out code
- sets up Go from `backend/go.mod`
- sets up Node.js for frontend
- injects `DATABASE_URL`
- runs `./scripts/run-all-migrations.sh`
- runs `cd backend && go test ./...`
- runs `cd frontend && npm ci && npm run build`

**Step 4: Run validation to verify it passes**

Run: `actionlint .github/workflows/verify.yml`
Expected: PASS.

### Task 2: Complete GitHub Actions documentation

**Files:**
- Modify: `README.md`
- Modify: `DEPLOYMENT.md`
- Modify: `CI_CD_TEMPLATE.md`
- Create: `GITHUB_ACTIONS.md`

**Step 1: Write the failing test**

Identify the documentation gap: the repository will have a real workflow but the docs do not yet explain where it lives, what it verifies, or how it differs from enterprise deployment pipelines.

**Step 2: Run inspection to verify the gap**

Run:
- `rg -n 'GitHub Actions|verify.yml|workflow' README.md DEPLOYMENT.md CI_CD_TEMPLATE.md GITHUB_ACTIONS.md 2>/dev/null || true`
Expected: missing or incomplete coverage before implementation.

**Step 3: Write minimal implementation**

Document:
- local test flow versus GitHub Actions flow
- workflow location and triggers
- backend PostgreSQL requirement in CI
- distinction between verification workflow and later enterprise deployment pipelines

**Step 4: Run verification**

Run:
- `rg -n 'GitHub Actions|verify.yml|workflow' README.md DEPLOYMENT.md CI_CD_TEMPLATE.md GITHUB_ACTIONS.md`
Expected: relevant matches in all intended docs.

### Task 3: Re-run full worktree verification

**Files:**
- Check: `.github/workflows/verify.yml`
- Check: repo root docs
- Check: backend/frontend code already changed in worktree

**Step 1: Write the failing test**

Use existing verification commands as the required proof gate before merge.

**Step 2: Run full verification**

Run:
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`
- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `actionlint .github/workflows/verify.yml`

**Step 3: Fix any failures minimally**

Resolve only issues found by these commands.

**Step 4: Re-run the same commands**

Expected: all pass.

### Task 4: Commit the worktree branch

**Files:**
- All modified and added files in the worktree

**Step 1: Verify working tree contents**

Run: `git status --short`
Expected: only intended changes for PostgreSQL migration, docs, tests, and workflow.

**Step 2: Create commit**

Run non-interactively with a descriptive message covering PostgreSQL migration and CI verification.

**Step 3: Verify commit exists**

Run: `git log --oneline -1`
Expected: new commit at HEAD.

### Task 5: Merge branch back into main and re-verify

**Files:**
- Main repository checkout at `/Users/qianjianghao/Desktop/Skill_Hub`

**Step 1: Verify main is safe to merge into**

Run:
- `git status --short`
- `git branch --show-current`
Expected: clean `main` checkout.

**Step 2: Merge branch**

Run a non-fast-forward merge from `codex/postgres-migration` into `main`.
If conflicts occur:
- inspect conflicted files
- resolve conservatively without reverting user work
- ask the user only if intent is ambiguous

**Step 3: Re-run proof commands on main**

Run:
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`
- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `actionlint .github/workflows/verify.yml`

**Step 4: Report final state**

Include:
- merge result
- exact verification commands run
- any remaining operational notes
