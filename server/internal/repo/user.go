// Package repo — 账号管理（Admin-only User Management）的数据访问。
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

// ErrLastEnabledAdmin 表示这次变更会导致「启用中的 admin」归零，被拒绝。
var ErrLastEnabledAdmin = errors.New("不能移除最后一个启用中的管理员")

// ListUsers 返回全部账号（不分页；账号数量级不需要），按 id 升序。
func (r *Repo) ListUsers(ctx context.Context) ([]model.AdminUser, error) {
	var list []model.AdminUser
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询账号列表失败: %w", err)
	}
	return list, nil
}

// GetUserByID 按 id 取账号。
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

// ExistsUsernameCI 大小写不敏感地检查用户名是否已被占用（可选排除某 id，供编辑场景复用）。
// 用应用层 LOWER() 比较而非依赖库表 collation：sqlite（开发）默认大小写敏感、
// MySQL（生产）默认多为不敏感，两边行为不一致；显式比较保证跨环境结果一致。
func (r *Repo) ExistsUsernameCI(ctx context.Context, username string, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.AdminUser{}).Where("LOWER(username) = LOWER(?)", username)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, fmt.Errorf("校验用户名唯一性失败: %w", err)
	}
	return n > 0, nil
}

// CountEnabledAdmins 统计当前启用中的 admin 数量。
func (r *Repo) CountEnabledAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AdminUser{}).
		Where("role = ? AND enabled = ?", model.RoleAdmin, true).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("统计启用中管理员数失败: %w", err)
	}
	return n, nil
}

// UpdateUserRoleEnabled 在事务中修改角色和/或启用状态（nil = 不改该字段）。
//
// 「最后一个启用中的 admin」保护对并发生效：用 SELECT ... FOR UPDATE 锁住目标行
// （MySQL 生效；SQLite 无行锁但同一连接事务天然串行，足够开发期正确性），事务内
// 重新统计其余启用中的 admin 数，确保两个并发请求不会同时把最后一个 admin
// 降级/禁用成功（其中一个必然读到「其余启用 admin = 0」而被拒）。
//
// 返回 ErrLastEnabledAdmin 时，调用方（service 层）应映射为 409。
func (r *Repo) UpdateUserRoleEnabled(ctx context.Context, id uint64, newRole *string, newEnabled *bool) (*model.AdminUser, error) {
	var result model.AdminUser
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u model.AdminUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("查询账号失败: %w", err)
		}

		wasEnabledAdmin := u.Role == model.RoleAdmin && u.Enabled
		willBeAdmin := u.Role == model.RoleAdmin
		willBeEnabled := u.Enabled
		if newRole != nil {
			willBeAdmin = *newRole == model.RoleAdmin
		}
		if newEnabled != nil {
			willBeEnabled = *newEnabled
		}
		willBeEnabledAdmin := willBeAdmin && willBeEnabled

		if wasEnabledAdmin && !willBeEnabledAdmin {
			var others int64
			if err := tx.Model(&model.AdminUser{}).
				Where("role = ? AND enabled = ? AND id != ?", model.RoleAdmin, true, id).
				Count(&others).Error; err != nil {
				return fmt.Errorf("统计启用中管理员数失败: %w", err)
			}
			if others == 0 {
				return ErrLastEnabledAdmin
			}
		}

		fields := map[string]any{}
		if newRole != nil {
			fields["role"] = *newRole
		}
		if newEnabled != nil {
			fields["enabled"] = *newEnabled
		}
		if len(fields) > 0 {
			if err := tx.Model(&model.AdminUser{}).Where("id = ?", id).Updates(fields).Error; err != nil {
				return fmt.Errorf("更新账号失败: %w", err)
			}
		}
		if err := tx.First(&result, id).Error; err != nil {
			return fmt.Errorf("重新查询账号失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
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
