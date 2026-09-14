-- BP 原始事件开关（渠道级）+ 品牌 Adjust 短链 host（品牌级）。
--
-- adjust_bp_raw_events：BP 品牌小渠道包是否走「Adjust 原始事件」新逻辑（相对现有 adjust_app_token/
-- adjust_events 那套是编译期 feature gate，见 CLI adjust-tokens.json 渲染）。只有所属品牌为 bp 的
-- 渠道才允许置 true（业务侧校验，见 internal/service/channel.go），非 bp 品牌恒为 false。
-- NOT NULL DEFAULT 0：存量渠道（含非 bp 品牌）加列后自动为「不启用」，与加列前行为一致。
ALTER TABLE channel
  ADD COLUMN adjust_bp_raw_events TINYINT(1) NOT NULL DEFAULT 0 AFTER adjust_events;

-- adjust_deeplink_host：品牌级 Adjust 品牌短链 host（如 link.bingoplus.com），只存 host、不含
-- scheme/路径/端口。空串 = 未配置该品牌的 Adjust 短链。
-- NOT NULL DEFAULT ''：存量品牌加列后为空串，等价于「未配置」，行为与加列前一致。
ALTER TABLE brand
  ADD COLUMN adjust_deeplink_host VARCHAR(128) NOT NULL DEFAULT '';
