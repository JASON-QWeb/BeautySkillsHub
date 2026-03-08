INSERT INTO users (username, password, avatar_url)
SELECT 'local-admin', '$2a$10$hDvxOGlpNlbGaejU9pNObOSAriP9ba3TRAezaFbtfZ0JMIjr2SsQu', ''
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE username = 'local-admin'
);

INSERT INTO skills (
    name,
    description,
    resource_type,
    author,
    ai_approved,
    ai_review_status,
    ai_review_phase,
    human_review_status,
    published
)
SELECT
    'Local Demo Skill',
    'Seeded development record for local PostgreSQL workflow.',
    'skill',
    'local-admin',
    TRUE,
    'passed',
    'done',
    'approved',
    TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM skills WHERE name = 'Local Demo Skill' AND resource_type = 'skill'
);
