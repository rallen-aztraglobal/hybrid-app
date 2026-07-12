-- 回滚：去掉 channel 上的 Adjust 归因字段。
ALTER TABLE channel
  DROP COLUMN adjust_events,
  DROP COLUMN adjust_app_token;
