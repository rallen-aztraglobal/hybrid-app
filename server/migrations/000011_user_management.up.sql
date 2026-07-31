-- 后台账号管理（Admin-only User Management）：账号列表/新建/改角色/启停用/重置密码。
-- 硬删除会破坏引用（audit_log.user_id / channel.created_by / listing_app.created_by
-- 均指向 admin_user.id，且都没有 FK 级联），故本模块只做启停用（enabled），不提供硬删除接口。
--
-- 生产由 golang-migrate 执行；开发期 GORM AutoMigrate 同步等价结构（见 model.AdminUser）。

ALTER TABLE admin_user
  ADD COLUMN enabled TINYINT(1) NOT NULL DEFAULT 1 AFTER role,
  ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at;
