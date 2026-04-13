DROP INDEX IF EXISTS notified_problem_user_idx;
DROP TABLE IF EXISTS notified_problem;
ALTER TABLE tg_user DROP COLUMN IF EXISTS last_polled_submission_id;
