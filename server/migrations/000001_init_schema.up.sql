-- 渠道中台初始表结构（MySQL）。对应 docs/admin/01-architecture.md §4。
-- 由 golang-migrate 在生产环境执行；开发期也可用 GORM AutoMigrate（两者列名保持一致）。

-- 大渠道（品牌）
CREATE TABLE IF NOT EXISTS brand (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  code         VARCHAR(16)  NOT NULL UNIQUE,
  name         VARCHAR(64)  NOT NULL,
  scheme       VARCHAR(32)  NOT NULL,
  hms_enabled  TINYINT(1)   NOT NULL DEFAULT 0,
  accent_color VARCHAR(16)  DEFAULT NULL,
  sort         INT          NOT NULL DEFAULT 0,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 品牌级默认域名（小渠道默认继承）
CREATE TABLE IF NOT EXISTS brand_domain (
  id        BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_id  BIGINT NOT NULL,
  position  TINYINT NOT NULL,
  url       VARCHAR(255) NOT NULL,
  enabled   TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_brand_pos (brand_id, position),
  CONSTRAINT fk_bd_brand FOREIGN KEY (brand_id) REFERENCES brand(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 小渠道
CREATE TABLE IF NOT EXISTS channel (
  id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_id           BIGINT       NOT NULL,
  flavor_name        VARCHAR(64)  NOT NULL,
  application_id     VARCHAR(128) NOT NULL UNIQUE,
  pal_code           VARCHAR(64)  NOT NULL UNIQUE,
  app_name           VARCHAR(128) NOT NULL,
  status             VARCHAR(16)  NOT NULL DEFAULT 'enabled',
  use_brand_domains  TINYINT(1)   NOT NULL DEFAULT 1,
  icon_master_url    VARCHAR(255) DEFAULT NULL,
  icon_set_url       VARCHAR(255) DEFAULT NULL,
  splash_url         VARCHAR(255) DEFAULT NULL,
  remark             VARCHAR(255) DEFAULT NULL,
  created_by         BIGINT       DEFAULT NULL,
  created_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_brand_flavor (brand_id, flavor_name),
  CONSTRAINT fk_ch_brand FOREIGN KEY (brand_id) REFERENCES brand(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 小渠道级域名覆盖（use_brand_domains=0 时生效）
CREATE TABLE IF NOT EXISTS channel_domain (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  channel_id BIGINT NOT NULL,
  position   TINYINT NOT NULL,
  url        VARCHAR(255) NOT NULL,
  enabled    TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_ch_pos (channel_id, position),
  CONSTRAINT fk_cd_channel FOREIGN KEY (channel_id) REFERENCES channel(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 域名健康巡检结果
CREATE TABLE IF NOT EXISTS domain_health (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  url          VARCHAR(255) NOT NULL,
  status       VARCHAR(16)  NOT NULL DEFAULT 'unknown',
  http_code    INT          DEFAULT NULL,
  latency_ms   INT          DEFAULT NULL,
  checked_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_url_time (url, checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 构建记录（CLI 回传）
CREATE TABLE IF NOT EXISTS build_record (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_code   VARCHAR(16)  NOT NULL,
  flavors      JSON         NOT NULL,
  test_events  TINYINT(1)   NOT NULL DEFAULT 0,
  status       VARCHAR(16)  NOT NULL,
  operator     VARCHAR(64)  DEFAULT NULL,
  version_name VARCHAR(32)  DEFAULT NULL,
  apk_urls     JSON         DEFAULT NULL,
  log_excerpt  TEXT         DEFAULT NULL,
  started_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at  DATETIME     DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 后台账号
CREATE TABLE IF NOT EXISTS admin_user (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role          VARCHAR(16) NOT NULL DEFAULT 'operator',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 审计日志
CREATE TABLE IF NOT EXISTS audit_log (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id    BIGINT,
  action     VARCHAR(64) NOT NULL,
  target     VARCHAR(128),
  detail     JSON,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
