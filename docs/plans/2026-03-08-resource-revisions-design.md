# Resource Revisions Review Flow Design

**Goal**: Keep one published resource record per asset while routing edits through a single pending revision that must pass review before replacing the published content.

## Problem

Current behavior is inconsistent across resource types:
- `mcp` and `tools` already submit edits via `PUT`, so they update the existing record.
- `skill` and `rules` use prefill on the upload page but still submit `POST`, so each edit creates a second database record.
- Published content and pending edits are stored in the same `skills` row, so there is no safe way to hold a reviewed draft without affecting the current public version.

This causes duplicate cards, ambiguous review state, and no guarantee that only one pending update exists for a resource.

## Requirements

1. Editing any resource type (`skill`, `rules`, `mcp`, `tools`) must never create a second published card.
2. Published content stays visible on detail pages while an update is under review.
3. Each published resource can have at most one active pending update.
4. Reviewers review the pending update, not the current published content.
5. When review passes, the pending update overwrites the published record in place and preserves the original resource ID.
6. Downloads, likes, and favorites remain attached to the published resource ID and are preserved across updates.
7. While an update is pending, authors cannot submit another update for the same resource.
8. Detail pages expose the pending-update state and provide a review entry for reviewers.

## Architecture

### Published Record vs Revision Record

Keep `skills` as the table for the currently published resource. Add a new `skill_revisions` table for pending updates.

`skills` remains the source of truth for:
- card lists
- detail pages
- likes/favorites/downloads
- final GitHub metadata

`skill_revisions` stores the candidate replacement content:
- edited metadata
- replacement file/folder/pasted markdown content
- replacement thumbnail
- review lifecycle state
- reviewer feedback for the revision

This cleanly separates public content from in-progress edits and avoids self-joins or duplicate card rows in `skills`.

### Revision Lifecycle

1. User clicks edit on a published resource.
2. Frontend checks whether an active revision already exists.
3. If none exists, edit form submits a `PUT`-style update request that creates or updates the single pending revision.
4. The published `skills` row remains unchanged.
5. Detail page shows a revision badge such as `更新中 / 待 Review` and a review CTA.
6. Review page loads the pending revision content.
7. On review approval:
   - copy revision content onto the `skills` row
   - preserve `skills.id`, `downloads`, `likes_count`, favorite relationships, like relationships
   - sync to GitHub using the new revision payload
   - mark revision as applied
8. On review rejection:
   - published row remains unchanged
   - revision stays available for the author to update and resubmit

### Single Active Revision Constraint

Enforce both application-level and database-level constraints:
- a service check prevents creating a second active revision
- a partial unique index guarantees at most one active revision per `skill_id`

Active means a revision is still in the review pipeline, for example:
- `draft`
- `queued`
- `running`
- `passed_ai_pending_human`

Terminal states such as `applied`, `rejected`, or `cancelled` are not active.

## Data Model

### New Table: `skill_revisions`

Proposed fields:
- `id`
- `skill_id` FK to `skills.id`
- `user_id`
- `resource_type`
- `name`
- `description`
- `category`
- `tags`
- `author`
- `file_name`
- `file_path`
- `file_size`
- `thumbnail_url`
- `github_path`
- `github_url`
- `github_files`
- `github_sync_status`
- `github_sync_error`
- `ai_approved`
- `ai_review_status`
- `ai_review_phase`
- `ai_review_attempts`
- `ai_review_max_attempts`
- `ai_review_started_at`
- `ai_review_completed_at`
- `ai_review_details`
- `ai_feedback`
- `ai_description`
- `human_review_status`
- `human_reviewer_id`
- `human_reviewer`
- `human_review_feedback`
- `human_reviewed_at`
- `status` (revision status: `pending`, `applied`, `rejected`, `cancelled`)
- `created_at`
- `updated_at`

Not copied into revisions:
- `downloads`
- `likes_count`
- `published`

Those remain on `skills`.

## API Changes

### Skills and Rules

Add true edit mode for `skill` and `rules`:
- frontend uses `?edit=:id`
- frontend fetches published resource plus pending revision summary
- submitting edit no longer calls upload `POST`
- submitting edit calls a revision endpoint that stores pending content instead of mutating `skills`

Recommended endpoints:
- `GET /api/skills/:id/revision`
- `PUT /api/skills/:id/revision`
- `GET /api/rules/:id/revision`
- `PUT /api/rules/:id/revision`

### MCP and Tools

These already use `?edit=` and `PUT`, but should move to the same revision mechanism for consistency.

That means:
- `PUT /api/mcps/:id` and `PUT /api/tools/:id` should no longer mutate the published row directly
- they should create or update the pending revision instead

This unifies all four resource types under the same review semantics.

### Detail and Review APIs

Detail responses should include revision summary fields when a pending update exists, for example:
- `has_pending_revision`
- `pending_revision_status`
- `pending_revision_id`
- `pending_revision_updated_at`

Review endpoints should load revision content for update reviews instead of only the base `skills` row.

## Frontend Behavior

### Card Click

Cards continue to open the detail page for all users.

### Detail Page

If a pending revision exists:
- keep showing the current published content
- show a visible update state badge
- replace edit action with a disabled or status-style `更新中 / 待 Review`
- show a `去 Review` or `帮忙 Review` action for eligible reviewers

### Edit Pages

For all resource types:
- if an active revision exists, load that revision into the edit form
- if no active revision exists, prefill from the published record
- on submit, update the revision rather than the published row

### Review Page

When reviewing an update revision:
- show the revision content being reviewed
- make clear that the current public version remains live until approval

## Error Handling

1. If a second update submission is attempted while a revision is active, return `409 Conflict`.
2. If the author tries to review their own pending revision, return `403`.
3. If revision apply fails after approval, keep the revision in a failure state with enough diagnostics to retry safely.
4. If GitHub sync fails during apply, do not silently mark the revision applied.

## Testing Strategy

### Backend

Add regression tests for:
- creating a revision instead of a new `skills` row on edit
- one active revision per resource
- rejecting a second concurrent update
- review approval copying revision content onto the published row without changing the resource ID
- preserving download/like/favorite state
- review rejection preserving the original published content

### Frontend

Add tests for:
- skill/rules edit mode using revision APIs instead of upload `POST`
- detail page showing pending revision state without replacing published content
- edit CTA disabled or status-tagged while revision is active
- mcp/tools edits using revision flow instead of direct mutation

## Migration Strategy

1. Add `skill_revisions` table and indexes.
2. Do not backfill historical duplicates automatically in this change.
3. New edits use the revision flow; old duplicate rows remain existing data hygiene work.

## Tradeoffs

### Why this is better than direct overwrite

Direct overwrite with review reset is simpler but makes the public version unstable during review. The revision table keeps production content stable.

### Why this is better than duplicate `skills` rows

Using a dedicated revision table avoids card duplication, simplifies published queries, and protects engagement metrics by keeping a single public ID.
