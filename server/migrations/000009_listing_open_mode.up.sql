-- 上架包新增「打开方式 open_mode」：B 面（web）的打开方式开关。
-- internal（默认）= 客户端原生 WebView 内开；external = 客户端唤起外部浏览器外开。
-- 仅在网关判为 B 面时对客户端有意义，不影响 A/B 判定逻辑本身（listing_gate 不变）。
--
-- 生产 schema 主要靠 GORM AutoMigrate 加列（见 internal/model/model.go AllModels），
-- 本文件是等价备份 / 发版脚本，字段定义须与 model.ListingApp.OpenMode 的 gorm tag 保持一致。
ALTER TABLE listing_app
  ADD COLUMN open_mode VARCHAR(16) NOT NULL DEFAULT 'internal' AFTER gate_enabled;
