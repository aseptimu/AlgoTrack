ALTER TABLE tg_user ADD COLUMN last_polled_submission_id TEXT;

CREATE TABLE notified_problem (
    user_id      BIGINT NOT NULL REFERENCES tg_user(user_id) ON DELETE CASCADE,
    title_slug   TEXT   NOT NULL,
    notified_day DATE   NOT NULL,
    PRIMARY KEY (user_id, title_slug, notified_day)
);

CREATE INDEX notified_problem_user_idx ON notified_problem(user_id);
