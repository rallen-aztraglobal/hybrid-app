-- Adjust 归因集成（ADR-0013 / docs/admin/08-adjust.md）：channel 表加两个可空字段。
-- adjust_app_token：Adjust 面板 App Token，NULL/空 = 该渠道未绑定 Adjust，打包时不集成、不发事件。
-- adjust_events：{事件name: token} 的 JSON 对象，由 Web 前端解析运营上传的 Adjust 事件 CSV
-- （token,name,unique，unique 列丢弃）得到；server 不解析 CSV，只存储/校验（空 或 string->string
-- 的 JSON 对象）并原样读出，供 CLI 渲染 app/adjust-tokens.json。

ALTER TABLE channel
  ADD COLUMN adjust_app_token VARCHAR(64) NULL AFTER store_id,
  ADD COLUMN adjust_events    JSON        NULL AFTER adjust_app_token;
