CREATE TABLE IF NOT EXISTS skill_revisions (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    resource_type VARCHAR(50) NOT NULL DEFAULT 'skill',
    author VARCHAR(100) NOT NULL DEFAULT '',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    file_path VARCHAR(512) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    thumbnail_url VARCHAR(512) NOT NULL DEFAULT '',
    ai_approved BOOLEAN NOT NULL DEFAULT FALSE,
    ai_review_status VARCHAR(32) NOT NULL DEFAULT 'queued',
    ai_review_phase VARCHAR(32) NOT NULL DEFAULT 'queued',
    ai_review_attempts INTEGER NOT NULL DEFAULT 0,
    ai_review_max_attempts INTEGER NOT NULL DEFAULT 3,
    ai_review_started_at TIMESTAMPTZ NULL,
    ai_review_completed_at TIMESTAMPTZ NULL,
    ai_review_details TEXT NOT NULL DEFAULT '',
    ai_feedback TEXT NOT NULL DEFAULT '',
    ai_description TEXT NOT NULL DEFAULT '',
    human_review_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    human_reviewer_id BIGINT NULL,
    human_reviewer VARCHAR(100) NOT NULL DEFAULT '',
    human_review_feedback TEXT NOT NULL DEFAULT '',
    human_reviewed_at TIMESTAMPTZ NULL,
    github_path VARCHAR(1024) NOT NULL DEFAULT '',
    github_url VARCHAR(1024) NOT NULL DEFAULT '',
    github_files TEXT NOT NULL DEFAULT '',
    github_sync_status VARCHAR(32) NOT NULL DEFAULT 'disabled',
    github_sync_error TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skill_revisions_skill_id ON skill_revisions (skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_revisions_user_id ON skill_revisions (user_id);
CREATE INDEX IF NOT EXISTS idx_skill_revisions_resource_type ON skill_revisions (resource_type);
CREATE INDEX IF NOT EXISTS idx_skill_revisions_status ON skill_revisions (status);
CREATE INDEX IF NOT EXISTS idx_skill_revisions_ai_review_status ON skill_revisions (ai_review_status);
CREATE INDEX IF NOT EXISTS idx_skill_revisions_human_review_status ON skill_revisions (human_review_status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_revisions_one_pending_per_skill
    ON skill_revisions (skill_id)
    WHERE status = 'pending';
