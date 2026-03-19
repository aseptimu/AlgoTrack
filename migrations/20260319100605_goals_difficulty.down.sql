ALTER TABLE tg_user
    DROP COLUMN IF EXISTS goal_hard;

ALTER TABLE tg_user
    DROP COLUMN IF EXISTS goal_medium;

ALTER TABLE tg_user
    DROP COLUMN IF EXISTS goal_easy;
