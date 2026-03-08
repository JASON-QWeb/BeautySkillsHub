# Root Docs Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refresh the root documentation set so it matches the current Skill Hub codebase and operational model.

**Architecture:** Keep `README.md` as the short landing page, move local development detail into a new `DEVELOPMENT.md`, and update architecture, deployment, and AI review docs from current code instead of historical assumptions.

**Tech Stack:** Markdown, repository shell scripts, Docker Compose, Go backend, React frontend.

---

### Task 1: Capture current behavior from code

**Files:**
- Read: `backend/cmd/server/main.go`
- Read: `backend/internal/config/config.go`
- Read: `backend/internal/service/ai/review.go`
- Read: `backend/internal/handler/resource_handler.go`
- Read: `scripts/*.sh`
- Read: `docker-compose.yml`

**Steps:**
1. Record local dev entrypoints, deployment ports, security constraints, and review flow differences by resource type.
2. Use those observations as the source of truth for all doc edits.

### Task 2: Rebuild README and add DEVELOPMENT.md

**Files:**
- Modify: `README.md`
- Create: `DEVELOPMENT.md`

**Steps:**
1. Reduce README to overview, quick-start, docs map, and key current facts.
2. Move script-driven local development and test guidance into DEVELOPMENT.md.
3. Ensure README points readers to DEVELOPMENT.md instead of duplicating operational detail.

### Task 3: Refresh architecture and deployment docs

**Files:**
- Modify: `ARCHITECTURE.md`
- Modify: `DEPLOYMENT.md`

**Steps:**
1. Update architecture doc to describe current runtime topology, resource lifecycle, security middleware, logging, and data boundaries.
2. Update deployment doc to reflect non-root containers, `/health`, production SSL requirements, JWT/CORS/security config, and current container ports.

### Task 4: Refresh AI review flow doc

**Files:**
- Modify: `ai-review流程.md`

**Steps:**
1. Document that `skill` and `rules` use AI + human review, while `mcp` and `tools` are auto-approved/published.
2. Include retry limits, reviewed revisions, and the `OPENAI_API_KEY` missing auto-approve fallback.

### Task 5: Verify and finalize

**Files:**
- Verify: `README.md`
- Verify: `DEVELOPMENT.md`
- Verify: `ARCHITECTURE.md`
- Verify: `DEPLOYMENT.md`
- Verify: `ai-review流程.md`

**Steps:**
1. Run text searches for stale deployment values like production `sslmode=disable` examples.
2. Run `git diff --check`.
3. Review the final doc set for broken internal references and consistency.
