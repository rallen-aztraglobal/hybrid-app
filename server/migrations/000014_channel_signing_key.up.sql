-- 签名 key 选择：一批已上架商店的渠道当年用另一把 keystore（CN=empty-app）签名，商店按包名
-- 绑定证书、不能变；其余渠道用默认 key（release-key，CN=bingo）。构建机 runner 打包后按此
-- 列的值用 apksigner 重签，Gradle 构建逻辑不动。空串 = 默认 key。DB 只存 key ID，不存密钥材料。
ALTER TABLE channel
  ADD COLUMN signing_key VARCHAR(32) NOT NULL DEFAULT '' AFTER live_version;

-- 存量回填：2025-09~10 在 ap_xiaomi / gzmkt031 分支用 gzmkt031-key.jks（CN=empty-app）签名并
-- 上架的渠道。生产走 AutoMigrate 只会加列（不会自动回填），此 UPDATE 由运维在发版时手工执行。
UPDATE channel SET signing_key = 'emptyapp'
 WHERE signing_key = '' AND application_id IN (
   'com.arenaplus.ap01018','com.arenaplus.ap01019','com.arenaplus.ap01020',
   'com.arenaplus.ap01021','com.arenaplus.ap01022','com.gamezone.gzmkt031');
