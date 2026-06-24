// Package repo 是数据访问层（GORM）。service 只依赖这里暴露的方法，不直接碰 *gorm.DB。
package repo

import (
	"fmt"

	"github.com/glebarez/sqlite" // 纯 Go 的 sqlite 驱动（无 cgo，呼应 ADR-0001 全静态）
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/model"
)

// Open 按配置打开数据库连接。
func Open(cfg *config.Config) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "mysql":
		dialector = mysql.Open(cfg.DBDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("不支持的 DB_DRIVER: %s", cfg.DBDriver)
	}
	db, err := gorm.Open(dialector, gcfg)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return db, nil
}

// AutoMigrate 用 GORM 建表（开发便利；生产建议用 golang-migrate 的 SQL）。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return fmt.Errorf("AutoMigrate 失败: %w", err)
	}
	return nil
}
