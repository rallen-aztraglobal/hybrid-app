// Package migrations 内嵌 SQL 迁移文件，供 golang-migrate 的 iofs source 在生产环境执行。
// 开发期默认走 GORM AutoMigrate（见 internal/repo），生产期建议用这些版本化迁移。
package migrations

import "embed"

// FS 内嵌全部 *.up.sql / *.down.sql。
//
//go:embed *.sql
var FS embed.FS
