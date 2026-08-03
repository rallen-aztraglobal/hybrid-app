-- 账号管理（Admin-only User Management，V1 单管理员 MVP）：
-- 账号列表 / 新建普通用户（role 恒 user）/ 重置密码 / 删除用户。
--
-- 删除用户用软删除（deleted_at 非空 = 已删除），不物理删除行：channel.created_by /
-- listing_app.created_by / audit_log.user_id 均引用 admin_user.id 且无级联，物理删除
-- 会让这些外键字段悬空、破坏审计与归属追溯。软删除保留行、只是对常规查询不可见
-- （GORM AdminUser.DeletedAt 字段语义，见 model.go 注释）。
--
-- 生产由 golang-migrate 执行；开发期 GORM AutoMigrate 同步等价结构。

ALTER TABLE admin_user
  ADD COLUMN deleted_at DATETIME NULL AFTER created_at,
  ADD INDEX idx_admin_user_deleted_at (deleted_at);
