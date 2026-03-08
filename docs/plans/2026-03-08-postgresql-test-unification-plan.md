# PostgreSQL Test Unification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the remaining SQLite-based backend tests with PostgreSQL-backed tests that use the same migration-first schema model as production.

**Architecture:** Add a shared backend test helper that provisions an isolated PostgreSQL schema per test, runs SQL migrations into that schema, and returns a GORM connection bound to that schema via `search_path`. Migrate all remaining handler/service tests from SQLite + `AutoMigrate` to this helper, then remove the SQLite dependency if no code still requires it.

**Tech Stack:** Go, GORM, PostgreSQL, golang-migrate, Docker Compose local PostgreSQL, Go testing

---

### Task 1: Add shared PostgreSQL test helper

**Files:**
- Create: `backend/internal/testutil/postgres.go`
- Create: `backend/internal/testutil/postgres_test.go`
- Check: `backend/internal/config/config.go`
- Check: `db/migrations/0001_init_schema.up.sql`

**Step 1: Write the failing test**

Add helper tests that require:
- a test DB can be created from `DATABASE_URL`
- helper creates an isolated schema and applies migrations
- helper cleanup drops the schema after the test

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/testutil -run TestOpenPostgresTestDB -v`
Expected: FAIL because helper does not exist yet.

**Step 3: Write minimal implementation**

Implement a helper that:
- loads `DATABASE_URL` from env/config
- creates a unique schema name from `t.Name()`
- opens an admin connection to create/drop the schema
- runs migrations against a connection string with `search_path=<schema>`
- returns `*gorm.DB` connected to that schema
- registers cleanup with `t.Cleanup`

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/testutil -run TestOpenPostgresTestDB -v`
Expected: PASS.

### Task 2: Migrate handler tests from SQLite to PostgreSQL helper

**Files:**
- Modify: `backend/internal/handler/auth_test.go`
- Modify: `backend/internal/handler/security_p0_test.go`
- Check: `backend/internal/model/*.go`
- Use: `backend/internal/testutil/postgres.go`

**Step 1: Write the failing test**

Switch one handler test file at a time to call the new PostgreSQL helper and remove SQLite setup. Keep assertions unchanged.

**Step 2: Run targeted tests to verify failures are schema/setup related**

Run:
- `cd backend && go test ./internal/handler -run 'TestRegister|TestNewAuthHandler' -v`
- `cd backend && go test ./internal/handler -run 'TestSkillHandler|TestResourceHandler' -v`
Expected: FAIL until helper wiring and seed setup are correct.

**Step 3: Write minimal implementation**

Refactor local test setup functions to:
- request a PostgreSQL test DB from helper
- keep per-test temp directories for filesystem concerns
- keep existing assertions and routes unchanged

**Step 4: Run targeted tests to verify they pass**

Run the same commands above.
Expected: PASS.

### Task 3: Migrate service tests from SQLite to PostgreSQL helper

**Files:**
- Modify: `backend/internal/service/skill_like_test.go`
- Modify: `backend/internal/service/skill_favorite_test.go`
- Modify: `backend/internal/service/skill_summary_test.go`
- Modify: `backend/internal/service/skill_trending_test.go`
- Modify: `backend/internal/service/skill_context_provider_test.go`
- Use: `backend/internal/testutil/postgres.go`

**Step 1: Write the failing test**

Convert each service test file to use the helper and keep current assertions intact.

**Step 2: Run targeted tests to verify failures**

Run:
- `cd backend && go test ./internal/service -run 'TestLike|TestUnlike' -v`
- `cd backend && go test ./internal/service -run 'TestAddFavorite|TestGetUserFavorites' -v`
- `cd backend && go test ./internal/service -run TestGetResourceSummary -v`
- `cd backend && go test ./internal/service -run TestGetTrending_UsesSameVisibilityRulesAsList -v`
- `cd backend && go test ./internal/service -run 'TestSkillContextProvider' -v`
Expected: FAIL until setup is fully migrated.

**Step 3: Write minimal implementation**

Refactor test setup helpers so each test file gets a migrated PostgreSQL schema and uses the same service methods against real PostgreSQL semantics.

**Step 4: Run targeted tests to verify they pass**

Run the same commands above.
Expected: PASS.

### Task 4: Remove stale SQLite test dependency and verify whole backend suite

**Files:**
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`
- Check: all backend `*_test.go`

**Step 1: Write the failing test**

Confirm whether any test files still import SQLite.

**Step 2: Run search to verify current failure condition**

Run: `rg -n 'glebarez/sqlite|AutoMigrate' backend/internal backend/cmd -g'*_test.go'`
Expected: no remaining SQLite test imports/setup after migration.

**Step 3: Write minimal implementation**

If SQLite is no longer used anywhere in backend code or tests, run `go mod tidy` to drop it.

**Step 4: Run full verification**

Run:
- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`

Expected: all pass.

### Task 5: Update docs for unified PostgreSQL testing

**Files:**
- Modify: `README.md`
- Modify: `ARCHITECTURE.md`
- Modify: `CI_CD_TEMPLATE.md`

**Step 1: Write the failing test**

Identify places where docs still imply tests use SQLite or do not mention the PostgreSQL requirement.

**Step 2: Run inspection to verify the gap**

Run:
- `rg -n 'SQLite|go test' README.md ARCHITECTURE.md CI_CD_TEMPLATE.md`

**Step 3: Write minimal implementation**

Update docs to state:
- backend tests now require accessible PostgreSQL
- local test path expects `./scripts/db-local.sh` first
- tests do not require frontend/backend dev servers to be running

**Step 4: Run final verification**

Run:
- `cd backend && go test ./...`
- `cd frontend && npm run build`

Expected: PASS.
