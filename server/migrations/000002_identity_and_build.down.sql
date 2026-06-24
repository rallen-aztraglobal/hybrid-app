-- 回滚 000002（按依赖倒序）。

-- ADR-0008
DROP TABLE IF EXISTS build_artifact;
ALTER TABLE build_record
  DROP COLUMN log,
  DROP COLUMN name;

-- ADR-0009：恢复 pal_code 全局唯一与去掉品牌包前缀。
-- 注意：若库内已存在重复 pal_code，恢复 UNIQUE 会失败——这是预期（数据已不满足旧约束）。
ALTER TABLE channel DROP INDEX idx_channel_pal_code;
ALTER TABLE channel ADD UNIQUE KEY pal_code (pal_code);

ALTER TABLE brand DROP COLUMN package_prefix;
