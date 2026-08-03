// Package repo — 账号管理（Admin-only User Management，V1 单管理员 MVP）的数据访问。
// 与 repo.go 里的登录/bootstrap 用途的 AdminUser 方法分开放，避免 repo.go 越滚越大。
package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/model"
)

// ListUsers 返回全部账号（不分页；账号数量级不需要），按 id 升序。已软删除的账号
// 天然不出现在结果里（GORM 默认查询会过滤 deleted_at 非空的行）。
func (r *Repo) ListUsers(ctx context.Context) ([]model.AdminUser, error) {
	var list []model.AdminUser
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询账号列表失败: %w", err)
	}
	return list, nil
}

// GetUserByID 按 id 取账号。已软删除的账号查不到（返回 ErrNotFound），
// 这正是 handler.RequireActiveAccount 用它做「账号是否仍存在」检查的基础。
func (r *Repo) GetUserByID(ctx context.Context, id uint64) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询账号失败: %w", err)
	}
	return &u, nil
}

// ExistsUsernameCI 大小写不敏感地检查用户名是否已被占用（含已软删除的账号）。
// 用应用层 LOWER() 比较而非依赖库表 collation：sqlite（开发）默认大小写敏感、
// MySQL（生产）默认多为不敏感，两边行为不一致；显式比较保证跨环境结果一致。
// 必须用 Unscoped()：admin_user.username 是单列全局唯一索引，即使某账号已被软删除，
// 其用户名仍占着这个唯一值——不用 Unscoped 会出现「应用层说可用，DB 唯一索引却拒绝插入」
// 的不一致。V1 接受「用户名删除后不可再复用」这个简化限制，不做局部唯一索引之类的复杂方案。
func (r *Repo) ExistsUsernameCI(ctx context.Context, username string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Unscoped().Model(&model.AdminUser{}).
		Where("LOWER(username) = LOWER(?)", username).Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("校验用户名唯一性失败: %w", err)
	}
	return n > 0, nil
}

// DeleteUser 软删除账号（GORM 对 DeletedAt 字段的默认行为：UPDATE ... SET deleted_at=now()，
// 不是物理 DELETE）。保留行本身，channel.created_by / listing_app.created_by /
// audit_log.user_id 等外键引用不会因此悬空——只是该账号从此对登录、鉴权、列表等常规查询
// 全部不可见，等效于「已删除」（见 model.AdminUser 的字段注释）。
func (r *Repo) DeleteUser(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Delete(&model.AdminUser{}, id)
	if res.Error != nil {
		return fmt.Errorf("删除账号失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePasswordHash 覆盖账号的密码哈希（重置密码用）。
func (r *Repo) UpdatePasswordHash(ctx context.Context, id uint64, hash string) error {
	res := r.db.WithContext(ctx).Model(&model.AdminUser{}).Where("id = ?", id).Update("password_hash", hash)
	if res.Error != nil {
		return fmt.Errorf("重置密码失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
