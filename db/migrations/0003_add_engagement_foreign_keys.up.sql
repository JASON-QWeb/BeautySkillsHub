DELETE FROM skill_likes
WHERE skill_id NOT IN (SELECT id FROM skills)
   OR user_id NOT IN (SELECT id FROM users);

DELETE FROM skill_favorites
WHERE skill_id NOT IN (SELECT id FROM skills)
   OR user_id NOT IN (SELECT id FROM users);

ALTER TABLE skill_likes
    ADD CONSTRAINT fk_skill_likes_skill_id
        FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE;

ALTER TABLE skill_likes
    ADD CONSTRAINT fk_skill_likes_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE skill_favorites
    ADD CONSTRAINT fk_skill_favorites_skill_id
        FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE;

ALTER TABLE skill_favorites
    ADD CONSTRAINT fk_skill_favorites_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
