CREATE TABLE recommended_problem (
    user_id        BIGINT      NOT NULL REFERENCES tg_user(user_id) ON DELETE CASCADE,
    task_number    INTEGER     NOT NULL,
    recommended_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, task_number)
);

CREATE INDEX recommended_problem_user_idx ON recommended_problem(user_id);

ALTER TABLE tg_user ADD COLUMN recommend_mode TEXT NOT NULL DEFAULT 'default';
