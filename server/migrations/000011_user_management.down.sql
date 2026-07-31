-- 回滚 000011：移除账号启停用状态与更新时间列。
-- 不可逆警告：down 之后所有账号的启停用历史丢失（回滚后视为「无此概念」，而非恢复到
-- 迁移前状态——迁移前本就没有这两列）。

ALTER TABLE admin_user
  DROP COLUMN enabled,
  DROP COLUMN updated_at;
