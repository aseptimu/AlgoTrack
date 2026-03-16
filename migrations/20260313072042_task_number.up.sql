ALTER TABLE algo_tasks
ADD COLUMN task_number INTEGER UNIQUE;

ALTER TABLE algo_tasks
DROP CONSTRAINT algo_tasks_user_link_uq;

ALTER TABLE algo_tasks
    DROP CONSTRAINT algo_tasks_link_key;

ALTER TABLE tg_user
    ADD COLUMN goal_total INTEGER;