-- 回滚上架包模块。先删有外键指向 listing_app 的子表，再删主表。
DROP TABLE IF EXISTS listing_gate_log;
DROP TABLE IF EXISTS listing_gate;
DROP TABLE IF EXISTS listing_domain;
DROP TABLE IF EXISTS listing_app;
