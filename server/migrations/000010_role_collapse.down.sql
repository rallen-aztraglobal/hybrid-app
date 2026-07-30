-- 回滚 000010：把默认角色改回 operator，并把非 admin 账号一律改回 operator。
--
-- 不可逆警告：这次收敛把 operator 与 viewer 合并成了 user，合并动作本身丢失了
-- 「某个非 admin 账号原本是 operator 还是 viewer」这条信息。本 down 迁移只能把
-- 所有非 admin 账号统一还原为 operator（历史上权限更高的一档），不能重建 viewer——
-- 如确需精确恢复 viewer 账号，只能从迁移前的数据库备份手工核对恢复，本脚本不代为处理。

ALTER TABLE admin_user
  MODIFY COLUMN role VARCHAR(16) NOT NULL DEFAULT 'operator';

UPDATE admin_user SET role = 'operator' WHERE role = 'user';
