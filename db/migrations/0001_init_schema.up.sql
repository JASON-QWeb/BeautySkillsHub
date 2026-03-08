CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password TEXT NOT NULL,
    avatar_url VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);

CREATE TABLE IF NOT EXISTS skills (
    id BIGSERIAL PRIMARY KEY,
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
    downloads BIGINT NOT NULL DEFAULT 0,
    likes_count BIGINT NOT NULL DEFAULT 0,
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
    published BOOLEAN NOT NULL DEFAULT FALSE,
    github_path VARCHAR(1024) NOT NULL DEFAULT '',
    github_url VARCHAR(1024) NOT NULL DEFAULT '',
    github_files TEXT NOT NULL DEFAULT '',
    github_sync_status VARCHAR(32) NOT NULL DEFAULT 'disabled',
    github_sync_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skills_user_id ON skills (user_id);
CREATE INDEX IF NOT EXISTS idx_skills_resource_type ON skills (resource_type);
CREATE INDEX IF NOT EXISTS idx_skills_ai_review_status ON skills (ai_review_status);
CREATE INDEX IF NOT EXISTS idx_skills_human_review_status ON skills (human_review_status);
CREATE INDEX IF NOT EXISTS idx_skills_human_reviewer_id ON skills (human_reviewer_id);
CREATE INDEX IF NOT EXISTS idx_skills_published ON skills (published);

CREATE TABLE IF NOT EXISTS skill_likes (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_user_like ON skill_likes (skill_id, user_id);

CREATE TABLE IF NOT EXISTS skill_favorites (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_user_favorite ON skill_favorites (skill_id, user_id);
