-- 数据权限：角色可见的品牌 / 渠道范围（docs/admin/10-rbac.md「数据权限」节）。
-- 只建表加列，不做数据搬运——DEFAULT 1 保证存量角色回填后仍是「全部范围」，不改变现状行为
-- （生产走 AutoMigrate，与 000010/000011 的路径一致；这里没有 Go seed 数据迁移路径，
-- 因为「全部」是标志位，DEFAULT 1 本身就是正确的初始值，不需要额外回填逻辑）。

ALTER TABLE role
  ADD COLUMN scope_all_brands   TINYINT(1) NOT NULL DEFAULT 1 AFTER builtin,
  ADD COLUMN scope_all_channels TINYINT(1) NOT NULL DEFAULT 1 AFTER scope_all_brands;

-- 角色-品牌数据范围关联。仅当 role.scope_all_brands=0 时生效。
CREATE TABLE IF NOT EXISTS role_brand (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id    BIGINT      NOT NULL,
  brand_code VARCHAR(16) NOT NULL,
  UNIQUE KEY uk_role_brand (role_id, brand_code),
  CONSTRAINT fk_role_brand_role FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 角色-渠道数据范围关联。仅当 role.scope_all_channels=0 时生效；有效范围计算时还会与品牌范围
-- 求交（品牌范围收窄后，这里勾选的渠道若不在允许品牌内自动失效，不需要清理本表，见
-- internal/auth.RBAC.RoleEffectiveScope）。
CREATE TABLE IF NOT EXISTS role_channel (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id    BIGINT NOT NULL,
  channel_id BIGINT NOT NULL,
  UNIQUE KEY uk_role_channel (role_id, channel_id),
  CONSTRAINT fk_role_channel_role FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE,
  CONSTRAINT fk_role_channel_channel FOREIGN KEY (channel_id) REFERENCES channel(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
