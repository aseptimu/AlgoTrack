ALTER TABLE tg_user DROP COLUMN IF EXISTS recommend_mode;
DROP INDEX IF EXISTS recommended_problem_user_idx;
DROP TABLE IF EXISTS recommended_problem;
