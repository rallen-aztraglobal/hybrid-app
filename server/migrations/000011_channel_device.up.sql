-- 渠道设备上报表：APK 启动时上报安装设备信息（GAID/ADID/OAID），device_key 幂等去重，
-- 供投放/归因侧导出。见 internal/model/device.go。
CREATE TABLE IF NOT EXISTS channel_device (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_key     VARCHAR(64)  NOT NULL COMMENT '客户端安装 UUID',
  application_id VARCHAR(128) NOT NULL,
  brand_code     VARCHAR(16)  NOT NULL DEFAULT '',
  pal_code       VARCHAR(64)  NOT NULL DEFAULT '',
  app_name       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '注册时渠道名快照，导出免 JOIN',
  device_name    VARCHAR(128) NOT NULL DEFAULT '',
  gaid           VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '统一小写；opt-out/全0 存空串',
  adid           VARCHAR(64)  NOT NULL DEFAULT '',
  oaid           VARCHAR(64)  NOT NULL DEFAULT '',
  created_at     DATETIME(3)  NOT NULL,
  updated_at     DATETIME(3)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_device_key (device_key),
  KEY idx_app_created (application_id, created_at, id),
  KEY idx_created (created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
