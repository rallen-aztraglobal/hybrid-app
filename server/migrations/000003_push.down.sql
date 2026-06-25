-- 回滚推送功能四张表（按依赖顺序逆序删除）。
DROP TABLE IF EXISTS push_record;
DROP TABLE IF EXISTS push_campaign_target;
DROP TABLE IF EXISTS push_campaign;
DROP TABLE IF EXISTS push_device_token;
