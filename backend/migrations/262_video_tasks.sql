CREATE TABLE IF NOT EXISTS video_tasks (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    upstream_id TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    next_poll_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until TIMESTAMPTZ,
    lease_token TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS video_tasks_due ON video_tasks(next_poll_at) WHERE active;
CREATE INDEX IF NOT EXISTS video_tasks_owner ON video_tasks(user_id, group_id) WHERE active;
CREATE UNIQUE INDEX IF NOT EXISTS video_tasks_upstream ON video_tasks(user_id, api_key_id, group_id, upstream_id) WHERE upstream_id <> '';
