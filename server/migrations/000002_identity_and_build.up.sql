-- ADR-0009 身份重构 + ADR-0008 服务器端构建（后端侧）的表结构变更。
-- 由 golang-migrate 在生产环境执行；开发期 GORM AutoMigrate 同步等价结构。

-- ---------- ADR-0009：品牌包前缀 + applicationId 派生 + pal_code 不再全局唯一 ----------

-- 品牌增包前缀（applicationId = package_prefix + "." + flavor 的事实来源）。
ALTER TABLE brand
  ADD COLUMN package_prefix VARCHAR(128) NOT NULL DEFAULT '' AFTER name;

-- 回填三个内置品牌的包前缀。
UPDATE brand SET package_prefix = 'com.arenaplus' WHERE code = 'ap';
UPDATE brand SET package_prefix = 'com.bingoplus' WHERE code = 'bp';
UPDATE brand SET package_prefix = 'com.gamezone' WHERE code = 'gp';

-- 删除 pal_code 的全局唯一约束（允许跨品牌/同品牌复用 palcode），改为普通索引便于查询。
-- 注：000001 用 `pal_code ... UNIQUE` 生成的隐式索引名为 pal_code。
ALTER TABLE channel DROP INDEX pal_code;
ALTER TABLE channel ADD INDEX idx_channel_pal_code (pal_code);

-- ---------- ADR-0008：build_record 增 name/log + 新增 build_artifact ----------

-- 构建记录增「任务名」与「完整日志」字段；状态机扩展为 queued/running/success/failed（值层面，无需改列）。
ALTER TABLE build_record
  ADD COLUMN name VARCHAR(128) DEFAULT NULL AFTER id,
  ADD COLUMN log  LONGTEXT     DEFAULT NULL AFTER log_excerpt;

-- 单个 APK 产物（每个 flavor 一条；apk_url 指向 nginx 静态目录，APK 不走对象存储）。
CREATE TABLE IF NOT EXISTS build_artifact (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  build_record_id BIGINT       NOT NULL,
  flavor          VARCHAR(64)  NOT NULL,
  version_name    VARCHAR(32)  DEFAULT NULL,
  apk_url         VARCHAR(512) NOT NULL,
  size            BIGINT       NOT NULL DEFAULT 0,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_artifact_record (build_record_id),
  KEY idx_flavor_time (flavor, created_at),
  CONSTRAINT fk_artifact_record FOREIGN KEY (build_record_id) REFERENCES build_record(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
