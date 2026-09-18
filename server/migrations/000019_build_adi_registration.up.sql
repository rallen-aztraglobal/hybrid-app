-- Android 开发者验证（包名注册）标识：打包中心选填，构建机打包前写入
-- 各 flavor 的 assets/adi-registration.properties。任务级（同一 Google 账号下各包名共用一个标识）。
-- NOT NULL DEFAULT ''：空串 = 本次不带注册文件；生产走 AutoMigrate 效果一致（只加列）。
ALTER TABLE build_record
  ADD COLUMN adi_registration VARCHAR(128) NOT NULL DEFAULT '' AFTER test_events;
