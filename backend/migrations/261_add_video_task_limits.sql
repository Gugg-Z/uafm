ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS video_max_pending_tasks INTEGER NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS video_task_timeout_hours INTEGER NOT NULL DEFAULT 2;

COMMENT ON COLUMN groups.video_max_pending_tasks IS '每个用户允许的待完成视频任务数';
COMMENT ON COLUMN groups.video_task_timeout_hours IS '视频任务未确认结果的最长等待小时数';
