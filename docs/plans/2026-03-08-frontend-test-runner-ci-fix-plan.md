# Frontend Test Runner CI Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix GitHub Actions frontend verification by giving TypeScript lightweight tests a single supported runner that works on Node 20 locally and in CI.

**Architecture:** Keep the existing lightweight tests and Node test model, but run them through `tsx` so `.ts` test files are loaded explicitly. Remove cwd assumptions from the Docker regression test and make workflow/docs call the same script.

**Tech Stack:** Node.js, tsx, npm scripts, GitHub Actions

---

### Task 1: Lock the expected behavior with failing tests

**Files:**
- Modify: `frontend/docker-runtime.test.mjs`

**Step 1: Write the failing test**

Add assertions that:
- the frontend package declares `tsx`
- the frontend package exposes a `test:node` script
- the workflow runs `cd frontend && npm run test:node`

**Step 2: Run test to verify it fails**

Run: `node --test frontend/docker-runtime.test.mjs`

**Step 3: Write minimal implementation**

Update `frontend/package.json`, add a small runner script, and change the workflow.

**Step 4: Run test to verify it passes**

Run: `node --test frontend/docker-runtime.test.mjs`

### Task 2: Remove cwd dependence from the Docker regression test

**Files:**
- Modify: `frontend/docker-runtime.test.mjs`

**Step 1: Resolve repo root from file location**

Make the test locate repo files relative to itself instead of `process.cwd()`.

**Step 2: Verify from both repo root and frontend directory**

Run:
- `node --test frontend/docker-runtime.test.mjs`
- `cd frontend && npm run test:node`

### Task 3: Add the supported frontend node test runner

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/scripts/run-node-tests.mjs`

**Step 1: Add direct dependency and script**

Add `tsx` and a single `test:node` npm script.

**Step 2: Verify the runner**

Run: `cd frontend && npm run test:node`

### Task 4: Align CI and docs with the supported command

**Files:**
- Modify: `.github/workflows/verify.yml`
- Modify: `README.md`
- Modify: `DEVELOPMENT.md`

**Step 1: Replace inline file list**

Switch workflow and docs to `cd frontend && npm run test:node`.

**Step 2: Verify build still passes**

Run: `cd frontend && npm run build`

### Task 5: Full verification and integration

**Files:**
- No new production files

**Step 1: Run full checks**

Run:
- `node --test frontend/docker-runtime.test.mjs`
- `cd frontend && npm ci`
- `cd frontend && npm run test:node`
- `cd frontend && npm run build`
- `cd backend && go test ./...`
- `git diff --check`

**Step 2: Commit and push**

Commit only the intended CI/test-runner fix, then push `main`.
