# Backend Review Fixes Round 2 Design

## Goal

Address the next batch of backend review findings without changing established product behavior. The targeted fixes are:

- add foreign-key safety for `skill_likes` and `skill_favorites`
- stop silently swallowing download counter failures
- make returned like counts consistent with the update transaction
- add broad but finite input-length validation for user-controlled text fields

## Scope

This change set intentionally excludes:

- changing the existing like/favorite semantics
- making `/download` fail when analytics counting fails
- SSRF hardening for `source_url`
- index tuning beyond schema safety changes

## Approach Options

### Option A: Focused safety fixes with backward-compatible behavior

Add one migration for referential integrity, patch handlers/services for better error handling and transactional consistency, and add generous validation limits in create/update flows.

Pros:

- Lowest regression risk
- Directly answers the validated review items
- Keeps API behavior stable for existing clients

Cons:

- Does not tackle speculative follow-up optimizations

### Option B: Broader data-model cleanup

Bundle foreign keys, composite indexes, stricter schema-level constraints, and handler/service refactors into one round.

Pros:

- More comprehensive data-layer cleanup

Cons:

- Significantly larger blast radius
- Harder to attribute regressions
- Not required for the user’s confirmed scope

### Option C: Minimal patches without tests or docs expansion

Patch the specific lines with small fixes only.

Pros:

- Fastest initial change

Cons:

- Violates TDD expectations
- Leaves migration safety and review traceability weak

## Recommendation

Use Option A. It fixes the confirmed problems while keeping semantics stable and testable.

## Detailed Design

### Foreign Keys

Create a new migration that:

- deletes orphaned rows from `skill_likes` and `skill_favorites`
- adds foreign keys from both tables to `skills(id)` with `ON DELETE CASCADE`
- adds foreign keys from both tables to `users(id)` with `ON DELETE CASCADE`

The migration must be additive and safe to run on deployed databases.

### Download Counting

Two code paths exist:

- `/download` serves the file directly
- `/download-hit` exists for explicit analytics tracking

Desired behavior:

- `/download`: if incrementing the counter fails, log the error and continue serving the file
- `/download-hit`: if incrementing fails, return an error as it already does

This removes silent failure without degrading the primary download path.

### Like Count Consistency

Keep the existing additive toggle logic:

- first like inserts the row and increments `likes_count`
- duplicate like remains idempotent
- unlike decrements only if a like existed

Change only how the response count is produced:

- fetch the updated `likes_count` inside the same transaction before commit, or use an atomic `RETURNING` strategy if the dialect supports it cleanly

The key point is to avoid a post-commit read that can observe unrelated concurrent updates.

### Input Length Validation

Apply the same limits in create and update flows for reviewed uploads and resource uploads:

- `name <= 255`
- `author <= 100`
- `source_url <= 1024`
- `tags <= 1000`
- `description <= 5000`

These limits are intentionally generous. They should block pathological payloads, not normal content.

### Testing

Write failing tests first for:

- migration SQL content expectations
- download handler behavior when `IncrementDownload` fails
- like/unlike count consistency under controlled service behavior
- oversized input rejection for skill and resource create/update endpoints

## Risks

- migration failures on existing dirty data if orphan cleanup is incomplete
- validation changes rejecting existing automated clients with extremely large payloads
- handler tests needing targeted seams for download-count failure injection

## Mitigations

- clean orphans in the migration before adding constraints
- use generous limits and clear error messages
- add focused handler tests instead of broad integration rewrites
