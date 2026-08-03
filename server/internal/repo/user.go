// Package repo — 账号管理（Admin-only User Management，V1 单管理员 MVP）的数据访问。
// 与 repo.go 里的登录/bootstrap 用途的 AdminUser 方法分开放，避免 repo.go 越滚越大。
package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hybrid-app/server/internal/model"
)

// ErrActiveUsernameConflict 表示该用户名已被一个「未删除」的账号占用（含永久 admin）。
var ErrActiveUsernameConflict = errors.New("用户名已被启用中的账号占用")

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

// CreateOrReactivateUser 新建一个普通用户账号；若该用户名（大小写不敏感）此前被一个
// 已软删除的账号占用，则原地复活那一行而非插入新行——保留原 id，清空 deleted_at，
// 角色强制改回 user（永久 admin 绝不会走到复活分支，因为它永远不会被删除），
// 密码哈希整行覆盖为本次提交的新密码。若该用户名当前被一个未删除的账号占用
// （含永久 admin），返回 ErrActiveUsernameConflict。
//
// 用 Unscoped() + SELECT ... FOR UPDATE 在同一个事务里完成「查找 → 判断 → 建/复活」：
// MySQL 下这会真正锁住命中的行（若已存在），阻止两个并发的「新建同用户名账号」请求
// 同时判断出「可以复活」而各自提交出不一致的结果（同一行被并发改两次，或一个复活
// 成功、一个误判成活跃冲突）；SQLite 没有行锁，但同一连接内的事务本身是串行执行的，
// 足够开发期的正确性。
func (r *Repo) CreateOrReactivateUser(ctx context.Context, username, passwordHash string) (*model.AdminUser, error) {
	var result model.AdminUser
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.AdminUser
		err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("LOWER(username) = LOWER(?)", username).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			result = model.AdminUser{Username: username, PasswordHash: passwordHash, Role: model.RoleUser}
			return tx.Create(&result).Error
		case err != nil:
			return fmt.Errorf("查询账号失败: %w", err)
		}

		if !existing.DeletedAt.Valid {
			return ErrActiveUsernameConflict
		}

		updates := map[string]any{
			"deleted_at":    nil,
			"role":          model.RoleUser,
			"password_hash": passwordHash,
			"username":      username, // 规范化为本次提交的写法，避免与旧大小写混淆
		}
		if err := tx.Unscoped().Model(&model.AdminUser{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("复活账号失败: %w", err)
		}
		return tx.Unscoped().First(&result, existing.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
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
