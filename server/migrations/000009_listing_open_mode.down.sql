-- 回滚：删除上架包「打开方式 open_mode」列。
ALTER TABLE listing_app
  DROP COLUMN open_mode;
