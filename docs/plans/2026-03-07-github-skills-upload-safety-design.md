# GitHub Skills Upload Safety And Deletion Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Keep current GitHub UX (`skills/<skill-name>/...`) while fixing path traversal and deletion safety, and preserving md/folder upload behavior.

**Architecture:** Use a canonical GitHub root dir per skill based on slugified skill name. Build and validate every uploaded path against that root. Persist uploaded GitHub file list (manifest) on each skill record and delete by exact manifest entries only.

**Tech Stack:** Go (Gin, GORM), SQLite model migration via AutoMigrate, existing GitHub Contents client.

---

### Task 1: Lock behavior with failing tests
- Add tests for path normalization to assert `skills/<skill-name>/<file>` and Chinese name support.
- Add tests for sync service conflict behavior when root dir belongs to another skill name.
- Add tests for delete behavior using manifest list rather than directory scan.

### Task 2: Implement safe GitHub path model
- Modify `BuildSkillRepoPath`/sync logic to always produce `skills/<slug>/<...>`.
- Add strict relative path normalization (`..`, absolute paths, empty segments rejected).
- Ensure folder upload remaps client top folder to skill-name root.

### Task 3: Persist GitHub manifest and harden delete
- Add `GitHubFiles` (`text`) in model for JSON file list.
- Save uploaded file paths to manifest on successful sync.
- Delete only manifest paths; remove parent-directory delete behavior.
- Keep backward-compatible fallback for legacy records without manifest.

### Task 4: Improve upload error feedback
- Return precise backend errors for conflict/path/sync failures.
- Frontend upload page maps common backend messages to clearer UI text and preserves raw error details.

### Task 5: Verify
- Run focused tests for service/handler changes.
- Run full backend tests and frontend build.
- Report remaining known failures not related to this change scope.
