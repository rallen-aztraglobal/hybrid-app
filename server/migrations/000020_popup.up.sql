-- 马甲包弹窗模块（ADR-0019 / docs/admin/12-popup.md §2）。
-- 开发期与生产实际均由 GORM AutoMigrate 同步等价结构，此文件供 golang-migrate 路径使用。
CREATE TABLE IF NOT EXISTS popup (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name               VARCHAR(64)  NOT NULL DEFAULT '',
  position           VARCHAR(8)   NOT NULL DEFAULT '',
  enabled            TINYINT(1)   NOT NULL DEFAULT 0,
  priority           INT          NOT NULL DEFAULT 0,
  start_at           DATETIME(3)  NULL,
  end_at             DATETIME(3)  NULL,
  brand_codes        TEXT         NULL COMMENT 'JSON 数组，空=全部品牌',
  app_ids            TEXT         NULL COMMENT 'JSON 数组 applicationId，空=全部',
  min_version_code   INT          NOT NULL DEFAULT 0,
  max_version_code   INT          NOT NULL DEFAULT 0,
  user_type          VARCHAR(8)   NOT NULL DEFAULT 'all',
  countries          TEXT         NULL COMMENT 'JSON 数组 ISO2 大写，空=全部地区',
  tab_enabled        TINYINT(1)   NOT NULL DEFAULT 0,
  tab_icon_url       VARCHAR(512) NOT NULL DEFAULT '',
  tab_text           VARCHAR(32)  NOT NULL DEFAULT '',
  countdown          TINYINT(1)   NOT NULL DEFAULT 0,
  mask_closable      TINYINT(1)   NOT NULL DEFAULT 0,
  closable           TINYINT(1)   NOT NULL DEFAULT 1,
  open_mode          VARCHAR(16)  NOT NULL DEFAULT 'webview',
  autoplay_seconds   INT          NOT NULL DEFAULT 5,
  resume_gap_minutes INT          NOT NULL DEFAULT 30,
  badge              TINYINT(1)   NOT NULL DEFAULT 0,
  created_by         VARCHAR(64)  NOT NULL DEFAULT '',
  created_at         DATETIME(3)  NULL,
  updated_at         DATETIME(3)  NULL,
  deleted_at         DATETIME(3)  NULL,
  PRIMARY KEY (id),
  KEY idx_popup_position (position),
  KEY idx_popup_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS popup_card (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  popup_id    BIGINT UNSIGNED NOT NULL,
  sort        INT          NOT NULL DEFAULT 0,
  image_url   VARCHAR(512) NOT NULL DEFAULT '',
  link_url    VARCHAR(512) NOT NULL DEFAULT '',
  button_text VARCHAR(32)  NOT NULL DEFAULT '',
  title       VARCHAR(64)  NOT NULL DEFAULT '',
  description VARCHAR(255) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  KEY idx_popup_card_popup (popup_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS popup_position (
  code       VARCHAR(8)  NOT NULL,
  enabled    TINYINT(1)  NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NULL,
  updated_by VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 唯一键字节数：10*4 + 8 + 8 + 4 + 128*4 + 32*4 + 32*4 + 16*4 = 904 < 3072。
CREATE TABLE IF NOT EXISTS popup_stat_daily (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  stat_date      VARCHAR(10)  NOT NULL,
  popup_id       BIGINT UNSIGNED NOT NULL,
  card_id        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  card_index     INT          NOT NULL DEFAULT 0,
  application_id VARCHAR(128) NOT NULL,
  app_version    VARCHAR(32)  NOT NULL DEFAULT '',
  event          VARCHAR(32)  NOT NULL,
  dim            VARCHAR(16)  NOT NULL DEFAULT '',
  count          BIGINT       NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_popup_stat (stat_date, popup_id, card_id, card_index, application_id, app_version, event, dim)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS popup_event_batch (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  batch_id   VARCHAR(64)  NOT NULL,
  created_at DATETIME(3)  NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_popup_event_batch_batch_id (batch_id),
  KEY idx_popup_event_batch_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
