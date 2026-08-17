// Package service — 角色 + 用户管理业务逻辑（唯一契约 docs/admin/10-rbac.md）。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/perm"
	"github.com/hybrid-app/server/internal/repo"
)

// minPasswordLen 是账号密码的最小强度要求（与前端一致，M5）。
const minPasswordLen = 6

// reservedUsernames 是不可注册为普通账号用户名的保留名单：runner 是构建机静态 token 的机器身份
// 标识（auth.RunnerUsername），虽然 B1 修复后机器身份已改用不可伪造的 Claims.Type 判定、
// 不再信任 Username，但仍在此拦截以避免运营侧混淆（一个叫 runner 的人类账号在审计日志/前端里
// 与机器身份撞名，容易误判）。
var reservedUsernames = map[string]bool{
	strings.ToLower(auth.RunnerUsername): true,
}

// ---------- 调用者权限解析（B2：最小权限约束） ----------

// callerPerms 解析调用者（callerID）的角色摘要与权限 code 集合，直接复用 auth.RBAC.ResolveFresh
// （不经过 30s 缓存——角色/用户管理写操作本就低频，直接查库保证即时一致；与 handler 层 RequirePerm
// 共享同一套「角色 → 权限集」算法，见 auth/rbac.go 注释）。
func (s *Service) callerPerms(ctx context.Context, callerID uint64) (auth.RoleInfo, map[string]bool, error) {
	return s.rbac.ResolveFresh(ctx, callerID)
}

// errPrivilegeEscalation 是 B2 约束统一使用的拒绝文案。
func errPrivilegeEscalation() *Error {
	return errForbidden("不能操作超出自身权限的角色/用户")
}

// assertSubsetOf 校验 codes 里的每个 code 都在 allowed 集合内；不满足则视为越权。
func assertSubsetOf(codes []string, allowed map[string]bool) error {
	for _, c := range codes {
		if !allowed[c] {
			return errPrivilegeEscalation()
		}
	}
	return nil
}

// assertCanManageRole 校验调用者是否可以对某个「目标角色」（新建/挂给用户/操作其成员/删除）生效：
// 调用者是 builtin（超级管理员）不受限；否则目标角色若是 builtin 一律拒绝，
// 非 builtin 则要求目标角色权限集 ⊆ 调用者权限集（防止越权挂号/接管更高权限账号）。
func (s *Service) assertCanManageRole(ctx context.Context, callerRole auth.RoleInfo, callerPerms map[string]bool, target *model.Role) error {
	if callerRole.Builtin {
		return nil
	}
	if target.Builtin {
		return errPrivilegeEscalation()
	}
	targetCodes, err := s.rbac.RolePermCodes(ctx, target)
	if err != nil {
		return err
	}
	return assertSubsetOf(targetCodes, callerPerms)
}

// mustGetRole 按 id 查角色，查不到（含 role_id 悬挂）一律当错误处理，不允许调用方静默跳过——
// fail-closed（应修3）：GetRoleByID 出错就直接返回错误，角色缺失归一化为 auth.ErrRoleMissing
// （经 service.AsError 映射为 403 + 明确文案），不要「查不到就当作不用校验」。
func (s *Service) mustGetRole(ctx context.Context, roleID uint64) (*model.Role, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, auth.ErrRoleMissing
		}
		return nil, err
	}
	return role, nil
}

// ---------- 角色 ----------

// RoleInput 是创建/修改角色的入参（POST/PUT 共用同一形状，PUT 视为整体替换）。
type RoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PermCodes   []string `json:"permCodes"`
}

// RoleView 是角色列表/详情的返回形状。
type RoleView struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Builtin     bool     `json:"builtin"`
	PermCodes   []string `json:"permCodes"`
	UserCount   int64    `json:"userCount"`
}

// roleView 组装单个角色的返回视图：builtin 角色的 permCodes 恒为 perm.AllCodes()
// （不依赖 role_permission 落库，经 auth.RBAC.RolePermCodes 与鉴权中间件共用同一份判定）。
func (s *Service) roleView(ctx context.Context, role *model.Role) (*RoleView, error) {
	codes, err := s.rbac.RolePermCodes(ctx, role)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountUsersByRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	return &RoleView{
		ID: role.ID, Name: role.Name, Description: role.Description,
		Builtin: role.Builtin, PermCodes: codes, UserCount: n,
	}, nil
}

// ListRoles 返回全部角色。
func (s *Service) ListRoles(ctx context.Context) ([]RoleView, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleView, 0, len(roles))
	for i := range roles {
		v, err := s.roleView(ctx, &roles[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

// CreateRole 新增角色（新建角色一律非 builtin）。callerID 是发起请求的账号 ID：
// 非超管调用者不能授出自己没有的权限（B2：permCodes 必须 ⊆ 调用者自身权限集）。
func (s *Service) CreateRole(ctx context.Context, callerID uint64, in RoleInput) (*RoleView, error) {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errBadRequest("name 不能为空")
	}
	if len(name) > 64 {
		return nil, errBadRequest("name 长度不能超过 64")
	}
	codes := dedupeStrings(in.PermCodes)
	if err := validatePermCodes(codes); err != nil {
		return nil, err
	}
	if !callerRole.Builtin {
		if err := assertSubsetOf(codes, callerPermSet); err != nil {
			return nil, err
		}
	}
	if _, err := s.repo.GetRoleByName(ctx, name); err == nil {
		return nil, errConflict(fmt.Sprintf("角色名 %q 已存在", name))
	}
	role := &model.Role{Name: name, Description: strings.TrimSpace(in.Description), Builtin: false}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, role.ID, codes); err != nil {
		return nil, err
	}
	return s.roleView(ctx, role)
}

// UpdateRole 修改角色（整体替换 name/description/permCodes）。builtin 角色不可编辑；
// 非超管调用者不能把 permCodes 改成超出自身权限集的集合（B2）。
func (s *Service) UpdateRole(ctx context.Context, callerID, id uint64, in RoleInput) (*RoleView, error) {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return nil, err
	}
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role.Builtin {
		return nil, errForbidden("内置角色不可编辑")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errBadRequest("name 不能为空")
	}
	if len(name) > 64 {
		return nil, errBadRequest("name 长度不能超过 64")
	}
	if other, err := s.repo.GetRoleByName(ctx, name); err == nil && other.ID != id {
		return nil, errConflict(fmt.Sprintf("角色名 %q 已存在", name))
	}
	codes := dedupeStrings(in.PermCodes)
	if err := validatePermCodes(codes); err != nil {
		return nil, err
	}
	if !callerRole.Builtin {
		if err := assertSubsetOf(codes, callerPermSet); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateRoleFields(ctx, id, map[string]any{
		"name": name, "description": strings.TrimSpace(in.Description),
	}); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, id, codes); err != nil {
		return nil, err
	}
	role.Name = name
	role.Description = strings.TrimSpace(in.Description)
	return s.roleView(ctx, role)
}

// DeleteRole 删除角色。builtin 不可删；仍有账号挂靠时拒绝（409）；callerID 非超管时受 B2 约束：
// 只能删除权限集 ⊆ 自身权限集的角色（应修5：防止仅持 role:manage 的账号删掉权限比自己大、
// 但恰好 0 成员的角色——这本身虽不是「提权」，但仍是「操作超出自身权限范围的角色」）。
func (s *Service) DeleteRole(ctx context.Context, callerID, id uint64) error {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return err
	}
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role.Builtin {
		return errForbidden("内置角色不可删除")
	}
	if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, role); err != nil {
		return err
	}
	n, err := s.repo.CountUsersByRole(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errConflict("该角色下仍有账号，无法删除")
	}
	return s.repo.DeleteRole(ctx, id)
}

// validatePermCodes 校验每个 code 都在 catalog 内（perm.RunnerPerm 等机器权限不算合法可分配 code）。
func validatePermCodes(codes []string) error {
	for _, c := range codes {
		if !perm.IsValid(c) {
			return errBadRequest(fmt.Sprintf("非法权限 code: %q", c))
		}
	}
	return nil
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ---------- 用户 ----------

// UserView 是账号列表/详情的返回形状。
type UserView struct {
	ID          uint64    `json:"id"`
	Username    string    `json:"username"`
	RoleID      uint64    `json:"roleId"`
	RoleName    string    `json:"roleName"`
	BuiltinRole bool      `json:"builtinRole"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateUserInput 新增账号入参。
type CreateUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   uint64 `json:"roleId"`
}

// UpdateUserInput 修改账号角色入参。
type UpdateUserInput struct {
	RoleID uint64 `json:"roleId"`
}

func (s *Service) userView(u *model.AdminUser, role *model.Role) UserView {
	return UserView{
		ID: u.ID, Username: u.Username, RoleID: u.RoleID,
		RoleName: role.Name, BuiltinRole: role.Builtin, CreatedAt: u.CreatedAt,
	}
}

// ListUsers 返回全部账号（含角色摘要）。
func (s *Service) ListUsers(ctx context.Context) ([]UserView, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	roleByID := make(map[uint64]model.Role, len(roles))
	for _, r := range roles {
		roleByID[r.ID] = r
	}
	out := make([]UserView, 0, len(users))
	for i := range users {
		role := roleByID[users[i].RoleID] // 角色缺失（异常数据）时零值 Role{}，前端展示为空角色名
		out = append(out, s.userView(&users[i], &role))
	}
	return out, nil
}

// validateUsername 校验用户名：必填、≤64 字节、非保留名。
func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", errBadRequest("username 不能为空")
	}
	if len(username) > 64 {
		return "", errBadRequest("username 长度不能超过 64 字节")
	}
	if reservedUsernames[strings.ToLower(username)] {
		return "", errBadRequest(fmt.Sprintf("username %q 是保留名，不可注册", username))
	}
	return username, nil
}

// validatePassword 校验密码最小强度（M5：与前端一致，最小长度 6）。
func validatePassword(password string) error {
	if len([]rune(password)) < minPasswordLen {
		return errBadRequest(fmt.Sprintf("密码长度不能少于 %d 位", minPasswordLen))
	}
	return nil
}

// CreateUser 新增账号：username 必填、唯一、≤64 字节、非保留名；password 最小长度 6；
// roleId 必须指向已存在角色。callerID 非超管时受 B2 约束：目标角色是 builtin 只有超管可挂，
// 非 builtin 也要求目标角色权限集 ⊆ 调用者权限集。
func (s *Service) CreateUser(ctx context.Context, callerID uint64, in CreateUserInput) (*UserView, error) {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return nil, err
	}
	username, err := validateUsername(in.Username)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(in.Password); err != nil {
		return nil, err
	}
	role, err := s.repo.GetRoleByID(ctx, in.RoleID)
	if err != nil {
		return nil, errBadRequest("角色不存在")
	}
	if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, role); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetUserByUsername(ctx, username); err == nil {
		return nil, errConflict(fmt.Sprintf("用户名 %q 已存在", username))
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	// Role 字符串列已停用（仅 seed 一次性回填读取），这里填个满足 NOT NULL 约束的占位值。
	u := &model.AdminUser{Username: username, PasswordHash: hash, Role: model.RoleOperator, RoleID: in.RoleID}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	v := s.userView(u, role)
	return &v, nil
}

// UpdateUser 修改账号角色；保护「最后一个超级管理员」不被改走；callerID 非超管时受 B2 双向约束
// （应修2：不仅新角色要 ⊆ 调用者权限集，目标账号「改动前」的角色也必须是调用者有权管理的——
// 否则非超管可以先把超管的角色改成自己权限子集内的角色，此时新角色校验能通过，
// 再顺势 ResetUserPassword 接管，逐个把超管剪到只剩一个）。
// 目标账号当前角色查不到（role_id 悬挂/查库出错）一律 fail-closed 拒绝（应修3的同款问题）。
func (s *Service) UpdateUser(ctx context.Context, callerID, id uint64, in UpdateUserInput) (*UserView, error) {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	newRole, err := s.repo.GetRoleByID(ctx, in.RoleID)
	if err != nil {
		return nil, errBadRequest("角色不存在")
	}
	currentRole, err := s.mustGetRole(ctx, u.RoleID)
	if err != nil {
		return nil, err
	}
	if !callerRole.Builtin {
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, currentRole); err != nil {
			return nil, err
		}
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, newRole); err != nil {
			return nil, err
		}
	}
	if u.RoleID != in.RoleID && currentRole.Builtin && !newRole.Builtin {
		if err := s.assertNotLastBuiltinUser(ctx, u.RoleID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateUserFields(ctx, id, map[string]any{"role_id": in.RoleID}); err != nil {
		return nil, err
	}
	u.RoleID = in.RoleID
	v := s.userView(u, newRole)
	return &v, nil
}

// ResetUserPassword 重置账号密码（最小长度 6）；callerID 非超管时受 B2 约束：目标账号挂 builtin
// 角色只有超管可重置，非 builtin 也要求目标角色权限集 ⊆ 调用者权限集（防止越权接管更高权限账号）。
// 目标角色查不到（role_id 悬挂/查库出错）一律 fail-closed 拒绝，不能因为查不到就静默跳过校验
// （应修3：以前用 `if targetRole, err := ...; err == nil` 包裹校验，出错时两道护栏全被跳过）。
func (s *Service) ResetUserPassword(ctx context.Context, callerID, id uint64, password string) error {
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	target, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	targetRole, err := s.mustGetRole(ctx, target.RoleID)
	if err != nil {
		return err
	}
	if !callerRole.Builtin {
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, targetRole); err != nil {
			return err
		}
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.UpdateUserFields(ctx, id, map[string]any{"password_hash": hash})
}

// DeleteUser 删除账号：不能删除自己；不能删除「最后一个超级管理员」；callerID 非超管时受 B2 约束
// （同 ResetUserPassword：目标账号挂 builtin 角色只有超管可删，非 builtin 也要求目标角色权限集
// ⊆ 调用者权限集）。目标角色查不到一律 fail-closed 拒绝（应修3，同 ResetUserPassword）。
func (s *Service) DeleteUser(ctx context.Context, callerID, id uint64) error {
	if id == callerID {
		return errBadRequest("不能删除自己")
	}
	callerRole, callerPermSet, err := s.callerPerms(ctx, callerID)
	if err != nil {
		return err
	}
	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	targetRole, err := s.mustGetRole(ctx, u.RoleID)
	if err != nil {
		return err
	}
	if !callerRole.Builtin {
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, targetRole); err != nil {
			return err
		}
	}
	if targetRole.Builtin {
		if err := s.assertNotLastBuiltinUser(ctx, u.RoleID); err != nil {
			return err
		}
	}
	return s.repo.DeleteUser(ctx, id)
}

// assertNotLastBuiltinUser 若某 builtin 角色下只剩 1 个账号，拒绝对其做角色变更/删除
// （保护最后一个超级管理员：调用方已确认目标角色 builtin=true 才会走到这里）。
func (s *Service) assertNotLastBuiltinUser(ctx context.Context, roleID uint64) error {
	n, err := s.repo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return errConflict("不能修改或删除最后一个超级管理员")
	}
	return nil
}
