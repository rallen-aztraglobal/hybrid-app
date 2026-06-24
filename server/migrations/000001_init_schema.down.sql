-- 回滚初始表结构。按外键依赖倒序删除。
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS admin_user;
DROP TABLE IF EXISTS build_record;
DROP TABLE IF EXISTS domain_health;
DROP TABLE IF EXISTS channel_domain;
DROP TABLE IF EXISTS channel;
DROP TABLE IF EXISTS brand_domain;
DROP TABLE IF EXISTS brand;
