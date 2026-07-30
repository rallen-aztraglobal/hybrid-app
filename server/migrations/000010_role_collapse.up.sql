-- RBAC 收敛为两档角色（admin / user）：产品需求变更，只保留 admin（全部权限）与
-- user（除系统设置/商店管理与渠道归档删除外的全部日常操作）。原 operator/viewer 两档取消。
--
-- role 列本身是 varchar(16)，无 ENUM/CHECK 约束，不需要改列类型——这里只做两件事：
--   1) 把历史数据里的 operator/viewer 一律归一为 user（admin 不变）；
--   2) 把新建账号的默认角色由 operator 改为 user。
--
-- 不可逆说明：down 迁移只能把「非 admin」的账号统一改回 operator，无法恢复
-- operator 与 viewer 的原始区分——这两个角色被合并前各自是谁，这条信息在 up 迁移
-- 执行后即丢失，属于本次收敛角色的预期结果，而非迁移脚本的缺陷。

UPDATE admin_user SET role = 'user' WHERE role IN ('operator', 'viewer');

ALTER TABLE admin_user
  MODIFY COLUMN role VARCHAR(16) NOT NULL DEFAULT 'user';
