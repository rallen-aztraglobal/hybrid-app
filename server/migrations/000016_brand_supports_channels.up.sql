-- 品牌产线标记（ADR-0017）：区分「渠道 APK 品牌」与「只做上架包的品牌」。
--   1 = 走渠道 APK 产线（ap/bp/gp），可建小渠道包；
--   0 = 只做上架包（wp/WavePlay），listing_app.brand_id 指向它、继承其品牌域名作 B 面，
--       后端拒绝给它建渠道，Console 渠道页/打包中心也不展示它。
--
-- 默认 1：存量三个品牌加列后自动为 1，行为与加列前一字不差，无需回填。
-- 新品牌（wp）由 server 启动时的 seed.EnsureBrands 建行并置 0，不在此处 INSERT。
ALTER TABLE brand
  ADD COLUMN supports_channels TINYINT(1) NOT NULL DEFAULT 1 AFTER hms_enabled;
