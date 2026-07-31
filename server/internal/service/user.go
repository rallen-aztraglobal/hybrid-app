package service

import (
	"context"
	"errors"
	"strings"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// maxPasswordBytes 是 bcrypt（auth.HashPassword 底层实现）本身的硬限制：超出即哈希失败。
// 这是当前鉴权体系里唯一「既有」的密码规则，此处显式前置校验，给出 400 而非哈希失败后的 500。
const maxPasswordBytes = 72

// validRole 只认 admin/user 两档（ADR：RBAC 收敛，见 model.go 角色枚举注释）。
func validRole(role string) bool {
	return role == model.RoleAdmin || role == model.RoleUser
}

// ListUsers 返回全部账号（admin-only）。AdminUser.PasswordHash 的 json 标签是 "-"，
// 直接下发该结构体天然不会带出密码哈希，无需额外脱敏步骤。
func (s *Service) ListUsers(ctx context.Context) ([]model.AdminUser, error) {
	return s.repo.ListUsers(ctx)
}

// CreateUserInput 新建账号入参（POST /api/users，admin-only）。
type CreateUserInput struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Role     string `json:"role" validate:"required"`
	// Enabled 缺省 true（新建账号默认可登录）。
	Enabled *bool `json:"enabled"`
}

// CreateUser 新建账号（admin-only）。角色只允许 admin/user（拒绝 operator/viewer 等历史角色）；
// 用户名大小写不敏感唯一；密码用与登录同一套 bcrypt 机制哈希，绝不明文落库。
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*model.AdminUser, error) {
	username := strings.TrimSpace(in.Username)
	if username == "" {
		return nil, errBadRequest("username 不能为空")
	}
	if !validRole(in.Role) {
		return nil, errBadRequest("role 非法（仅允许 admin/user）")
	}
	if in.Password == "" {
		return nil, errBadRequest("password 不能为空")
	}
	if len([]byte(in.Password)) > maxPasswordBytes {
		return nil, errBadRequest("password 过长")
	}
	exists, err := s.repo.ExistsUsernameCI(ctx, username, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errConflict("用户名已存在")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, errBadRequest("密码哈希失败")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	u := &model.AdminUser{Username: username, PasswordHash: hash, Role: in.Role, Enabled: enabled}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUserInput 修改账号入参（PUT /api/users/:id，admin-only）：仅 role / enabled 可改。
type UpdateUserInput struct {
	Role    *string `json:"role"`
	Enabled *bool   `json:"enabled"`
}

// UpdateUser 修改账号角色/启用状态（admin-only）。安全规则：
//   - 不能修改自己的角色、不能禁用自己的账号（actorID==targetID 时拒绝）；
//   - 不能让「启用中的 admin」归零——由 repo.UpdateUserRoleEnabled 在事务里对
//     并发请求做强一致校验（见该方法注释），这里只把 ErrLastEnabledAdmin 映射成 409。
func (s *Service) UpdateUser(ctx context.Context, actorID, targetID uint64, in UpdateUserInput) (*model.AdminUser, error) {
	if in.Role != nil && !validRole(*in.Role) {
		return nil, errBadRequest("role 非法（仅允许 admin/user）")
	}
	if actorID == targetID {
		if in.Role != nil && *in.Role != model.RoleAdmin {
			return nil, errConflict("不能修改自己的角色")
		}
		if in.Enabled != nil && !*in.Enabled {
			return nil, errConflict("不能禁用自己的账号")
		}
	}
	u, err := s.repo.UpdateUserRoleEnabled(ctx, targetID, in.Role, in.Enabled)
	if err != nil {
		if errors.Is(err, repo.ErrLastEnabledAdmin) {
			return nil, errConflict("不能移除最后一个启用中的管理员")
		}
		return nil, err
	}
	return u, nil
}

// ResetPasswordInput 重置密码入参（POST /api/users/:id/reset-password，admin-only）。
type ResetPasswordInput struct {
	Password string `json:"password" validate:"required"`
}

// ResetUserPassword 管理员重置指定账号密码：与登录同一套 bcrypt 哈希机制，
// 不记录、不返回明文；旧密码哈希被整行覆盖，旧密码此后必然失效。
func (s *Service) ResetUserPassword(ctx context.Context, targetID uint64, in ResetPasswordInput) error {
	if in.Password == "" {
		return errBadRequest("password 不能为空")
	}
	if len([]byte(in.Password)) > maxPasswordBytes {
		return errBadRequest("password 过长")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return errBadRequest("密码哈希失败")
	}
	return s.repo.UpdatePasswordHash(ctx, targetID, hash)
}
