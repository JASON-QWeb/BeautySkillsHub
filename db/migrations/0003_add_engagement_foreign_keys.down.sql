ALTER TABLE skill_favorites
    DROP CONSTRAINT IF EXISTS fk_skill_favorites_user_id;

ALTER TABLE skill_favorites
    DROP CONSTRAINT IF EXISTS fk_skill_favorites_skill_id;

ALTER TABLE skill_likes
    DROP CONSTRAINT IF EXISTS fk_skill_likes_user_id;

ALTER TABLE skill_likes
    DROP CONSTRAINT IF EXISTS fk_skill_likes_skill_id;
