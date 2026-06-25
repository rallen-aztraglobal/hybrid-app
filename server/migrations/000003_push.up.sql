-- 推送功能四张表（ADR-0012）。
-- 由 golang-migrate 在生产环境执行；开发期 GORM AutoMigrate 同步等价结构。

-- 设备 token 注册库（APK 上报 applicationId + token）
CREATE TABLE IF NOT EXISTS push_device_token (
  id              BIGINT       PRIMARY KEY AUTO_INCREMENT,
  application_id  VARCHAR(128) NOT NULL,
  brand_code      VARCHAR(16)  NOT NULL,
  device_token    VARCHAR(255) NOT NULL UNIQUE,
  pal_code        VARCHAR(64)  NOT NULL DEFAULT '',
  platform        VARCHAR(16)  NOT NULL DEFAULT 'android',
  model_info      VARCHAR(255) DEFAULT NULL,
  is_active       TINYINT(1)   NOT NULL DEFAULT 1,
  last_seen_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_token_app_active (application_id, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 推送活动（内容 + 目标 + 状态机）
CREATE TABLE IF NOT EXISTS push_campaign (
  id              BIGINT       PRIMARY KEY AUTO_INCREMENT,
  name            VARCHAR(128) NOT NULL,
  title           VARCHAR(255) NOT NULL,
  body            TEXT         NOT NULL,
  image_url       VARCHAR(512) DEFAULT NULL,
  deeplink_path   VARCHAR(512) DEFAULT NULL,
  extra_data      JSON         DEFAULT NULL,
  status          VARCHAR(16)  NOT NULL DEFAULT 'draft',
  scheduled_at    DATETIME     DEFAULT NULL,
  sent_at         DATETIME     DEFAULT NULL,
  total_devices   INT          NOT NULL DEFAULT 0,
  success_count   INT          NOT NULL DEFAULT 0,
  failure_count   INT          NOT NULL DEFAULT 0,
  created_by      VARCHAR(64)  DEFAULT NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_campaign_status_scheduled (status, scheduled_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 活动目标渠道（多对多：一个活动 → 多个 applicationId）
CREATE TABLE IF NOT EXISTS push_campaign_target (
  id              BIGINT       PRIMARY KEY AUTO_INCREMENT,
  campaign_id     BIGINT       NOT NULL,
  application_id  VARCHAR(128) NOT NULL,
  KEY idx_target_campaign (campaign_id),
  CONSTRAINT fk_target_campaign FOREIGN KEY (campaign_id) REFERENCES push_campaign(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 发送结果（按 application_id 汇总，不存逐 token 明细）
CREATE TABLE IF NOT EXISTS push_record (
  id              BIGINT       PRIMARY KEY AUTO_INCREMENT,
  campaign_id     BIGINT       NOT NULL,
  application_id  VARCHAR(128) NOT NULL,
  sent            INT          NOT NULL DEFAULT 0,
  failed          INT          NOT NULL DEFAULT 0,
  error_sample    TEXT         DEFAULT NULL,
  finished_at     DATETIME     DEFAULT NULL,
  KEY idx_record_campaign (campaign_id),
  CONSTRAINT fk_record_campaign FOREIGN KEY (campaign_id) REFERENCES push_campaign(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
