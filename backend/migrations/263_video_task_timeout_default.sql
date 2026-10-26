ALTER TABLE groups ALTER COLUMN video_task_timeout_hours SET DEFAULT 24;
COMMENT ON COLUMN groups.video_task_timeout_hours IS '视频任务未确认结果的最长等待小时数，0 表示永不超时';
