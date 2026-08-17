DROP TABLE IF EXISTS role_channel;
DROP TABLE IF EXISTS role_brand;

ALTER TABLE role
  DROP COLUMN scope_all_channels,
  DROP COLUMN scope_all_brands;
