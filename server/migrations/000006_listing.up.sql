-- 上架包（listing）模块：正式上架 Google Play / App Store 的合规应用 + AB 面放行网关。
-- 与 channel（小渠道 APK）是两条独立产线，故独立建表。
-- 注意：既有 store 表指「应用商店后缀（华为/小米/Oppo）」，与本模块无关，勿混淆。
-- 由 golang-migrate 在生产环境执行；开发期 GORM AutoMigrate 同步等价结构。

-- 上架包。
-- 唯一键是 (platform, bundle_id) 而非 bundle_id 单列：Flutter 包两端共用同一包名
-- （如 ColorStack 的 android 与 ios 都是 com.vividnest.colorstack5821），全局唯一会误杀。
-- brand_id + use_brand_domains 复刻 channel 的域名继承语义（ADR-0006）：B 面落到的 web 与小渠道同一套。
CREATE TABLE IF NOT EXISTS listing_app (
  id                BIGINT       PRIMARY KEY AUTO_INCREMENT,
  brand_id          BIGINT       NOT NULL,
  platform          VARCHAR(16)  NOT NULL,
  bundle_id         VARCHAR(128) NOT NULL,
  name              VARCHAR(64)  NOT NULL,
  display_name      VARCHAR(128) NULL,
  tech              VARCHAR(24)  NOT NULL,
  store_url         VARCHAR(512) NULL,
  status            VARCHAR(16)  NOT NULL DEFAULT 'enabled',
  use_brand_domains TINYINT(1)   NOT NULL DEFAULT 1,
  -- AB 面总开关，默认关闭：新建的上架包先只有 A 面，运营核对规则后再手动打开。
  gate_enabled      TINYINT(1)   NOT NULL DEFAULT 0,
  af_dev_key        VARCHAR(64)  NULL,
  af_app_id         VARCHAR(64)  NULL,
  adjust_app_token  VARCHAR(64)  NULL,
  adjust_events     JSON         NULL,
  remark            VARCHAR(255) NULL,
  created_by        BIGINT       NULL,
  created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_platform_bundle (platform, bundle_id),
  KEY idx_listing_brand (brand_id),
  CONSTRAINT fk_listing_brand FOREIGN KEY (brand_id) REFERENCES brand(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 上架包级 B 面域名覆盖（use_brand_domains=0 时生效）。position 0=主，1..n=备用。
CREATE TABLE IF NOT EXISTS listing_domain (
  id         BIGINT       PRIMARY KEY AUTO_INCREMENT,
  listing_id BIGINT       NOT NULL,
  position   INT          NOT NULL,
  url        VARCHAR(255) NOT NULL,
  enabled    TINYINT(1)   NOT NULL DEFAULT 1,
  UNIQUE KEY uk_listing_pos (listing_id, position),
  CONSTRAINT fk_ldomain_listing FOREIGN KEY (listing_id) REFERENCES listing_app(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- AB 面放行规则，与 listing_app 一对一。判定全在服务端做，客户端不内置任何 B 面地址。
-- 条件之间一律 AND，全部满足才进 B 面；任一不满足即 A 面。判定顺序与各字段语义
-- 详见 model/listing.go 的 ListingGate 文档（那里是唯一事实来源）。
-- 另有硬编码闸：国家为 CN/US 时强制 A 面，无视本表配置，故本表无需也无法配置这两国。
CREATE TABLE IF NOT EXISTS listing_gate (
  id             BIGINT      PRIMARY KEY AUTO_INCREMENT,
  listing_id     BIGINT      NOT NULL,
  -- 必填：ISO-3166-1 alpha-2 大写国家码白名单，服务端按请求 IP 查 GeoIP 得出。
  -- 非空由 Service 层校验强制（MySQL 无法约束 JSON 数组非空）；空清单 = 配置无效而非「不限国家」，
  -- 真要全关请把 listing_app.gate_enabled 置 0。
  countries      JSON        NULL,
  -- 选填：IANA 时区名白名单，客户端上报（可伪造，只作收紧条件叠加，不单独作准）。为空 = 不参与判定。
  timezones      JSON        NULL,
  -- 选填：非空时请求 IP 还必须落在其中之一（在国家白名单之上再收紧一层）。
  ip_allow_cidrs JSON        NULL,
  -- 选填：命中即强制 A 面；用于钉死已知审核机房网段。
  ip_deny_cidrs  JSON        NULL,
  updated_at     DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_gate_listing (listing_id),
  CONSTRAINT fk_gate_listing FOREIGN KEY (listing_id) REFERENCES listing_app(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 网关判定流水。上线后排查「为什么这台设备没进 B 面」全靠它，故连拒绝原因一并落库。
-- 仅作运维观测，不参与判定逻辑。
CREATE TABLE IF NOT EXISTS listing_gate_log (
  id         BIGINT       PRIMARY KEY AUTO_INCREMENT,
  listing_id BIGINT       NOT NULL,
  ip         VARCHAR(64)  NULL,
  country    VARCHAR(8)   NULL,
  timezone   VARCHAR(64)  NULL,
  decision   VARCHAR(4)   NOT NULL,
  reason     VARCHAR(128) NULL,
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_listing_time (listing_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
