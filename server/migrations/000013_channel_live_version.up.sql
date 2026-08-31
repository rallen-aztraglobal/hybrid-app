-- 线上版本号（人工备忘）：包太多记不住上次上线的版本，后台记一笔。
-- 仅记录/展示，不参与打包、不下发 APK、不进 CLI 渲染的 CSV。
-- NOT NULL DEFAULT ''：存量行补列后自然为「未记录」；生产走 AutoMigrate 效果一致（只加列）。
ALTER TABLE channel
  ADD COLUMN live_version VARCHAR(32) NOT NULL DEFAULT '' AFTER remark;
