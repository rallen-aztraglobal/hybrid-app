package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// maxPasswordBytes 是 bcrypt（auth.HashPassword 底层实现）本身的硬限制：超出即哈希失败。
// 这是当前鉴权体系里唯一「既有」的密码规则，此处显式前置校验，给出 400 而非哈希失败后的 500。
const maxPasswordBytes = 72

// UserView 是账号管理接口（列表/新建）下发的精简视图。
// Protected=true 表示该账号是系统里唯一的永久 admin：不可通过 User Management 改密/删除。
// V1 只有 admin 会是 protected；用显式字段而非让前端重复判断 role=="admin"，
// 是为将来「多 admin」演进预留的扩展点（届时 protected 的判定逻辑只需改这一处）。
type UserView struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	Protected bool      `json:"protected"`
}

func toUserView(u model.AdminUser) UserView {
	return UserView{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		Protected: u.Role == model.RoleAdmin,
	}
}

// ListUsers 返回全部账号（admin-only）。用 UserView 而非直接下发 model.AdminUser，
// 密码哈希从响应形状上就不存在，不依赖某个 json 标签不被误删。
func (s *Service) ListUsers(ctx context.Context) ([]UserView, error) {
	list, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserView, 0, len(list))
	for _, u := range list {
		out = append(out, toUserView(u))
	}
	return out, nil
}

// CreateUserInput 新建账号入参（POST /api/users，admin-only）。
// 没有 role 字段：V1 只支持创建普通用户，角色恒为 model.RoleUser，不接受调用方指定
// （不接受也就没有「创建 admin」这条路径，无需再校验/拒绝非法角色值）。
type CreateUserInput struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// CreateUser 新建一个普通用户账号（admin-only）。角色恒为 user；用户名大小写不敏感唯一。
// 密码用与登录同一套 bcrypt 机制哈希，绝不明文落库。
//
// 若该用户名曾经存在但已被删除（软删除），本次创建会原地复活那一行而不是报「已存在」——
// 见 repo.CreateOrReactivateUser 与 docs/admin/10-user-management.md「用户名复用」一节。
// 永久 admin 的用户名恒判定为「已被启用中的账号占用」（它从不会被删除），故不会被复活，
// 也不会被这条路径创建出第二个 admin。
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*UserView, error) {
	username := strings.TrimSpace(in.Username)
	if username == "" {
		return nil, errBadRequest("username 不能为空")
	}
	if in.Password == "" {
		return nil, errBadRequest("password 不能为空")
	}
	if len([]byte(in.Password)) > maxPasswordBytes {
		return nil, errBadRequest("password 过长")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, errBadRequest("密码哈希失败")
	}
	u, err := s.repo.CreateOrReactivateUser(ctx, username, hash)
	if err != nil {
		if errors.Is(err, repo.ErrActiveUsernameConflict) {
			return nil, errConflict("用户名已存在")
		}
		return nil, err
	}
	view := toUserView(*u)
	return &view, nil
}

// ResetPasswordInput 重置密码入参（POST /api/users/:id/reset-password，admin-only）。
type ResetPasswordInput struct {
	Password string `json:"password" validate:"required"`
}

// ResetUserPassword 管理员重置一个普通用户的密码：与登录同一套 bcrypt 哈希机制，
// 不记录、不返回明文；旧密码哈希被整行覆盖，旧密码此后必然失效。
// 永久 admin 的密码不能通过本接口重置——它不是 User Management 的管理对象。
func (s *Service) ResetUserPassword(ctx context.Context, targetID uint64, in ResetPasswordInput) error {
	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == model.RoleAdmin {
		return errConflict("不能通过账号管理重置管理员密码")
	}
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

// DeleteUser 删除一个普通用户（软删除，见 repo.DeleteUser / model.AdminUser 注释）。
// 永久 admin 不可删除——它是系统里唯一的管理员，删除会导致无人能管理账号，
// 且 V1 明确不支持多管理员，故没有「转移权限给另一个 admin」这个退路。
func (s *Service) DeleteUser(ctx context.Context, targetID uint64) error {
	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == model.RoleAdmin {
		return errConflict("不能删除管理员账号")
	}
	return s.repo.DeleteUser(ctx, targetID)
}
