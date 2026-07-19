-- 上架包推送扩展（ADR-0012 增补 / docs/admin/09-listing.md §6）。
-- 让推送四表既能服务小渠道（application_id 键，原样不动），也能服务上架包（listing_id 键）。
-- 由 golang-migrate 在生产执行；开发期 GORM AutoMigrate 同步等价结构。

-- 设备 token 表：加 listing_id 与「最近一次 AB 面判定结果」。
--   listing_id：非空 = 该 token 属于某上架包（与 application_id 二选一；Flutter 双端同包名，
--               仅靠 application_id 无法区分平台，故上架包一律用 listing_id + platform）。
--   last_gate_mode / last_gate_at：客户端注册 token 时带上它最近一次网关判定结果（A/B）。
--     上架包推送**强制只发 last_gate_mode='B' 的设备**（绝不把 B 面内容推给可能是审核员的 A 面设备）。
ALTER TABLE push_device_token
  ADD COLUMN listing_id     BIGINT      NULL AFTER application_id,
  ADD COLUMN last_gate_mode VARCHAR(4)  NULL AFTER platform,
  ADD COLUMN last_gate_at   DATETIME    NULL AFTER last_gate_mode,
  ADD INDEX idx_token_listing_gate (listing_id, last_gate_mode, is_active);

-- 活动目标表：加 listing_id（与 application_id 二选一）。
ALTER TABLE push_campaign_target
  ADD COLUMN listing_id BIGINT NULL AFTER application_id,
  ADD INDEX idx_target_listing (listing_id);

-- 发送结果表：加 listing_id 便于按上架包汇总。
ALTER TABLE push_record
  ADD COLUMN listing_id BIGINT NULL AFTER application_id;

-- 活动表：加 kind 区分「渠道推送」与「上架包推送」，默认 channel（存量活动语义不变）。
ALTER TABLE push_campaign
  ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT 'channel' AFTER name;
