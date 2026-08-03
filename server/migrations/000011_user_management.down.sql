-- 回滚 000011：移除软删除标记列。
-- 不可逆警告：down 之后已软删除账号的「已删除」状态信息丢失——回滚后这些行会重新出现在
-- 全部查询里，视为「从未被删除过」（这是迁移前本就没有该列的自然结果，而非缺陷）。

ALTER TABLE admin_user
  DROP INDEX idx_admin_user_deleted_at,
  DROP COLUMN deleted_at;
