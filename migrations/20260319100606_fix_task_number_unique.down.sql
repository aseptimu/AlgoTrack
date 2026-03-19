ALTER TABLE algo_tasks
    DROP CONSTRAINT IF EXISTS algo_tasks_user_task_number_uq;

ALTER TABLE algo_tasks
    ADD CONSTRAINT algo_tasks_task_number_key UNIQUE (task_number);
