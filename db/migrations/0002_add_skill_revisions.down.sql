DROP INDEX IF EXISTS idx_skill_revisions_one_pending_per_skill;
DROP INDEX IF EXISTS idx_skill_revisions_human_review_status;
DROP INDEX IF EXISTS idx_skill_revisions_ai_review_status;
DROP INDEX IF EXISTS idx_skill_revisions_status;
DROP INDEX IF EXISTS idx_skill_revisions_resource_type;
DROP INDEX IF EXISTS idx_skill_revisions_user_id;
DROP INDEX IF EXISTS idx_skill_revisions_skill_id;

DROP TABLE IF EXISTS skill_revisions;
