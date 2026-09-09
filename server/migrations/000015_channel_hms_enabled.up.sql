-- 渠道级「是否集成华为 HMS/OAID」开关：原判据只有「品牌整体开 HMS（bp）」+「flavor 以 _hw
-- 结尾的华为商店包」，漏掉了「已上架华为商店但 flavor 不带 _hw 后缀」的老渠道（如 ap01018）——
-- 这类包在华为设备上没有 GAID 也没有 OAID，AppsFlyer 归因会丢事件。
--
-- 可空（三态）是刻意的：NULL = 未显式配置，构建时回落上述默认规则，存量行为与加列前一字不差；
-- TRUE/FALSE = Console 在渠道表单里显式指定，优先于默认规则。故此处**不做任何回填**，
-- 哪些渠道要开由运营在后台逐个勾选（勾选后该行才写入显式值）。
ALTER TABLE channel
  ADD COLUMN hms_enabled TINYINT(1) NULL DEFAULT NULL AFTER signing_key;
