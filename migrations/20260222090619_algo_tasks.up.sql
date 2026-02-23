CREATE TABLE IF NOT EXISTS algo_tasks(
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT NOT NULL REFERENCES tg_user(user_id) ON DELETE CASCADE,

    link TEXT NOT NULL UNIQUE,
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT algo_tasks_user_link_uq UNIQUE (user_id, link)
);

