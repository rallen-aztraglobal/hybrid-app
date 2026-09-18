-- 回滚周期定时推送扩展。
ALTER TABLE push_campaign
  DROP INDEX idx_campaign_parent,
  DROP COLUMN parent_id,
  DROP COLUMN last_run_at,
  DROP COLUMN run_count,
  DROP COLUMN repeat_max_runs,
  DROP COLUMN repeat_end_at,
  DROP COLUMN repeat_every_days;
