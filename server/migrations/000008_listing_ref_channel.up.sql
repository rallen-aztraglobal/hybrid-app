-- 上架包本轮三处 schema 调整（开发期由 GORM AutoMigrate 同步等价结构；
-- 生产 AutoMigrate 只会「加列/加索引」，删列/改列由本文件或发版脚本执行）。

-- 1) 新增「参考渠道包 PAL_CODE」：运营手填的纯数字串，上架包用它拼 B 面 /?palcode=。
--    不关联具体渠道记录，直接存值；必填由 Service 层强制。
ALTER TABLE listing_app
  ADD COLUMN pal_code VARCHAR(64) NOT NULL DEFAULT '' AFTER use_brand_domains,
  ADD INDEX idx_listing_pal_code (pal_code);

-- 2) 删除「技术栈」列：上架包是已在商店发布的外部 App，Console 不负责构建，tech 不驱动任何逻辑，纯冗余。
ALTER TABLE listing_app
  DROP COLUMN tech;

-- 3) 时区规则由「白名单」改为「黑名单」：命中即强制 A 面（与 IP 黑名单同类）。
ALTER TABLE listing_gate
  CHANGE COLUMN timezones timezone_deny TEXT NULL;
