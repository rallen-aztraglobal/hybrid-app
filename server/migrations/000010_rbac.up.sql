-- RBAC：角色 + 角色权限表，admin_user 加 role_id 列（docs/admin/10-rbac.md）。
-- 只建表加列，不做数据搬运——生产唯一会跑的数据初始化路径是 Go 侧 seed.EnsureRBAC
-- （建三个内置初始角色 + 按旧 role 字符串回填存量账号 role_id + 清理失效权限点）。

CREATE TABLE IF NOT EXISTS role (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  name        VARCHAR(64)  NOT NULL,
  description VARCHAR(255) DEFAULT NULL,
  builtin     TINYINT(1)   NOT NULL DEFAULT 0,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_role_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS role_permission (
  id        BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id   BIGINT      NOT NULL,
  perm_code VARCHAR(64) NOT NULL,
  UNIQUE KEY uk_role_perm (role_id, perm_code),
  CONSTRAINT fk_role_permission_role FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- admin_user 增 role_id（鉴权依据）；旧 role 字符串列保留但不再参与鉴权。
ALTER TABLE admin_user
  ADD COLUMN role_id BIGINT NOT NULL DEFAULT 0 AFTER role,
  ADD INDEX idx_admin_user_role (role_id);
