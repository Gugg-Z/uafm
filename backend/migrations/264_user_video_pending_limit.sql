ALTER TABLE users ADD COLUMN IF NOT EXISTS video_max_pending_tasks INTEGER NOT NULL DEFAULT 0;
COMMENT ON COLUMN users.video_max_pending_tasks IS '用户所有分组视频待完成任务上限，0 沿用分组默认';
