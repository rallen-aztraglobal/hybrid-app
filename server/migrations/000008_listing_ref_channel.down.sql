-- 回滚本轮上架包 schema 调整。
ALTER TABLE listing_gate
  CHANGE COLUMN timezone_deny timezones TEXT NULL;

ALTER TABLE listing_app
  ADD COLUMN tech VARCHAR(24) NOT NULL DEFAULT 'flutter' AFTER display_name;

ALTER TABLE listing_app
  DROP INDEX idx_listing_pal_code,
  DROP COLUMN pal_code;
