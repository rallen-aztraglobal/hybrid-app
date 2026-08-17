package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/model"
)

// ---------- Role ----------

// ListRoles 返回全部角色（按 id 升序）。
func (r *Repo) ListRoles(ctx context.Context) ([]model.Role, error) {
	var list []model.Role
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return list, nil
}

// GetRoleByID 按 id 取角色。
func (r *Repo) GetRoleByID(ctx context.Context, id uint64) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &role, nil
}

// GetRoleByName 按名称取角色。
func (r *Repo) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &role, nil
}

// CreateRole 新建角色。
//
// 注意（GORM 零值陷阱，比想象的更狠）：Role.ScopeAllBrands/ScopeAllChannels 带
// gorm:"default:true" 标签——这是为了让 AutoMigrate/迁移生成的列带 DEFAULT 1（"已有角色默认
// 全部范围"）。但这个标签也让 GORM 在结构体 Create 时，对「Go 零值」（bool 的零值正好是
// false）的字段整套改写：不但跳过写入 INSERT 语句、让数据库 DEFAULT 生效，**还会在 Create
// 成功后把数据库实际写入的值（true）回填进传入的 role 指针**——也就是说，调用方传
// ScopeAllBrands: false 进来，Create 返回后 role.ScopeAllBrands 会被 GORM 悄悄改写成 true，
// 连"读原始入参再纠正"都不成立，必须在调用 Create 之前就把调用方的真实意图存进局部变量。
func (r *Repo) CreateRole(ctx context.Context, role *model.Role) error {
	wantAllBrands, wantAllChannels := role.ScopeAllBrands, role.ScopeAllChannels
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		return fmt.Errorf("创建角色失败: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", role.ID).Updates(map[string]any{
		"scope_all_brands":   wantAllBrands,
		"scope_all_channels": wantAllChannels,
	}).Error; err != nil {
		return fmt.Errorf("修正角色数据范围标志位失败: %w", err)
	}
	role.ScopeAllBrands, role.ScopeAllChannels = wantAllBrands, wantAllChannels
	return nil
}

// UpdateRoleFields 局部更新角色字段。
func (r *Repo) UpdateRoleFields(ctx context.Context, id uint64, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("更新角色失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRole 删除角色（调用方需先确认非 builtin 且未被用户引用）。
func (r *Repo) DeleteRole(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Delete(&model.Role{}, id)
	if res.Error != nil {
		return fmt.Errorf("删除角色失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// RolePermCodes 返回某角色的权限 code 列表。
func (r *Repo) RolePermCodes(ctx context.Context, roleID uint64) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).Model(&model.RolePermission{}).
		Where("role_id = ?", roleID).Pluck("perm_code", &codes).Error; err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}
	return codes, nil
}

// ReplaceRolePermissions 事务性替换某角色的权限集。
func (r *Repo) ReplaceRolePermissions(ctx context.Context, roleID uint64, codes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return fmt.Errorf("清空角色权限失败: %w", err)
		}
		if len(codes) == 0 {
			return nil
		}
		rows := make([]model.RolePermission, 0, len(codes))
		for _, code := range codes {
			rows = append(rows, model.RolePermission{RoleID: roleID, PermCode: code})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("写入角色权限失败: %w", err)
		}
		return nil
	})
}

// CountUsersByRole 统计挂在某角色下的账号数（角色删除前校验 / 保护最后一个超级管理员）。
func (r *Repo) CountUsersByRole(ctx context.Context, roleID uint64) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.AdminUser{}).Where("role_id = ?", roleID).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("统计角色用户数失败: %w", err)
	}
	return n, nil
}

// DeleteStaleRolePermissions 删除 perm_code 不在 validCodes 内的 role_permission 行
// （catalog 演进时防悬挂，seed.EnsureRBAC 启动时调用）。
func (r *Repo) DeleteStaleRolePermissions(ctx context.Context, validCodes []string) error {
	q := r.db.WithContext(ctx)
	if len(validCodes) == 0 {
		// 理论不会发生（catalog 静态非空）；防御性地不做全表删除。
		return nil
	}
	if err := q.Where("perm_code NOT IN ?", validCodes).Delete(&model.RolePermission{}).Error; err != nil {
		return fmt.Errorf("清理失效权限点失败: %w", err)
	}
	return nil
}

// ---------- AdminUser（RBAC 补充） ----------

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

// ListUsers 返回全部账号（按 id 升序）。
func (r *Repo) ListUsers(ctx context.Context) ([]model.AdminUser, error) {
	var list []model.AdminUser
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询账号列表失败: %w", err)
	}
	return list, nil
}

// UpdateUserFields 局部更新账号字段。
//
// 注意（M4）：不按 RowsAffected==0 判定 ErrNotFound——AdminUser 没有 autoUpdateTime 字段，
// 值未变化的更新（如把角色重复设成当前已挂的角色）在 MySQL 下 RowsAffected 会是 0（sqlite 因实现
// 差异测不出这个问题），会被误判成「账号不存在」。调用方（service.UpdateUser/ResetUserPassword）
// 在此之前已用 GetUserByID 确认账号存在，这里无需也不应再重复判定存在性。
func (r *Repo) UpdateUserFields(ctx context.Context, id uint64, fields map[string]any) error {
	if err := r.db.WithContext(ctx).Model(&model.AdminUser{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("更新账号失败: %w", err)
	}
	return nil
}

// DeleteUser 删除账号。
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
