-- 周期定时推送（单次/每天/每 N 天）。
-- 父任务从不直接发送：每次触发都克隆一条 parent_id 指向自己的子活动去真发，子活动
-- repeat_every_days 恒为 0，拥有独立 status/统计/push_record 历史。开发期 GORM AutoMigrate
-- 同步等价结构，此文件供生产 golang-migrate 执行。
ALTER TABLE push_campaign
  ADD COLUMN repeat_every_days INT      NOT NULL DEFAULT 0 AFTER failure_count,
  ADD COLUMN repeat_end_at     DATETIME NULL     AFTER repeat_every_days,
  ADD COLUMN repeat_max_runs  INT      NOT NULL DEFAULT 0 AFTER repeat_end_at,
  ADD COLUMN run_count        INT      NOT NULL DEFAULT 0 AFTER repeat_max_runs,
  ADD COLUMN last_run_at      DATETIME NULL     AFTER run_count,
  ADD COLUMN parent_id        BIGINT   NULL     AFTER last_run_at,
  ADD INDEX idx_campaign_parent (parent_id);
