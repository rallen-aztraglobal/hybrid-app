-- 回滚：先去掉 channel 上的外键/索引/列，再删 store 表。
ALTER TABLE channel DROP FOREIGN KEY fk_ch_store;
ALTER TABLE channel DROP INDEX idx_channel_store;
ALTER TABLE channel DROP COLUMN store_id;

DROP TABLE IF EXISTS store;
