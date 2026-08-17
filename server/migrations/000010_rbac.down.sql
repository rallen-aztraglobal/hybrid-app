ALTER TABLE admin_user DROP INDEX idx_admin_user_role;
ALTER TABLE admin_user DROP COLUMN role_id;

DROP TABLE IF EXISTS role_permission;
DROP TABLE IF EXISTS role;
