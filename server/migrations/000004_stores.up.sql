-- 应用商店后缀功能：新增 store 表，channel 增 store_id 外键（可空）。
-- 由 golang-migrate 在生产环境执行；开发期 GORM AutoMigrate 同步等价结构。

-- 应用商店（华为/小米/Oppo 等）。code 一旦被渠道引用即不可改（已进 applicationId 分段）。
CREATE TABLE IF NOT EXISTS store (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  code       VARCHAR(16)  NOT NULL UNIQUE,
  name       VARCHAR(64)  NOT NULL,
  sort       INT          NOT NULL DEFAULT 0,
  status     VARCHAR(16)  NOT NULL DEFAULT 'enabled',
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 预置三个商店（存在则忽略；与 seed.go 幂等 seed 保持一致，供直接执行迁移的环境也有初始数据）。
INSERT IGNORE INTO store (code, name, sort, status) VALUES
  ('hw', '华为', 1, 'enabled'),
  ('xm', '小米', 2, 'enabled'),
  ('op', 'Oppo', 3, 'enabled');

-- 渠道可选归属某个应用商店。
ALTER TABLE channel
  ADD COLUMN store_id BIGINT NULL AFTER created_by;

ALTER TABLE channel
  ADD INDEX idx_channel_store (store_id);

ALTER TABLE channel
  ADD CONSTRAINT fk_ch_store FOREIGN KEY (store_id) REFERENCES store(id) ON DELETE RESTRICT;
