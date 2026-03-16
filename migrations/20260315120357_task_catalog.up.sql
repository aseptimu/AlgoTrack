CREATE TABLE task_catalog (
    id SERIAL PRIMARY KEY,

    platform TEXT NOT NULL,
    external_id TEXT,
    title TEXT,
    slug TEXT,
    url TEXT NOT NULL,

    difficulty TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT task_platform_external_unique UNIQUE(platform, external_id)
);

CREATE TABLE user_tasks (
    id SERIAL PRIMARY KEY
)