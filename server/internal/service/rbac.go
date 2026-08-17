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

// ---------- 调用者权限/数据范围解析（B2 最小权限约束 + 数据权限，docs/admin/10-rbac.md） ----------

// callerContext 一次性解析调用者（callerID）的角色摘要、权限集与有效数据范围，供角色/用户
// 管理的最小权限校验统一复用（权限集 ⊆ 自身、数据范围 ⊆ 自身双重约束）。都直接复用
// auth.RBAC.ResolveFresh/EffectiveScope（不经过 30s 缓存——角色/用户管理写操作本就低频，
// 直接查库保证即时一致），与 handler 层 RequirePerm 共享同一套算法，不各写一份。
func (s *Service) callerContext(ctx context.Context, callerID uint64) (auth.RoleInfo, map[string]bool, auth.Scope, error) {
	role, perms, err := s.rbac.ResolveFresh(ctx, callerID)
	if err != nil {
		return auth.RoleInfo{}, nil, auth.Scope{}, err
	}
	scope, err := s.rbac.EffectiveScope(ctx, callerID)
	if err != nil {
		return auth.RoleInfo{}, nil, auth.Scope{}, err
	}
	return role, perms, scope, nil
}

// errPrivilegeEscalation 是最小权限约束统一使用的拒绝文案。
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
// 调用者是 builtin（超级管理员）不受限；否则目标角色若是 builtin 一律拒绝；非 builtin 则要求
// 目标角色的权限集 ⊆ 调用者权限集**且**数据范围 ⊆ 调用者数据范围（两条都要满足，防止越权
// 挂号/接管权限或数据范围比自己大的角色/账号）。
func (s *Service) assertCanManageRole(ctx context.Context, callerRole auth.RoleInfo, callerPerms map[string]bool, callerScope auth.Scope, target *model.Role) error {
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
	if err := assertSubsetOf(targetCodes, callerPerms); err != nil {
		return err
	}
	targetScope, err := s.rbac.RoleEffectiveScope(ctx, target.ID)
	if err != nil {
		return err
	}
	if !targetScope.SubsetOf(callerScope) {
		return errPrivilegeEscalation()
	}
	return nil
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

// RoleInput 是创建/修改角色的入参（POST/PUT 共用同一形状，PUT 视为整体替换，
// 数据范围字段同 permCodes 一样是全量替换语义，不是增量 patch）。
//
// ScopeAllBrands/ScopeAllChannels 默认零值 false——这不是「默认全部范围」的陷阱：调用方
// （handler）永远从前端拿到的是显式勾选状态（checkbox），不存在「没传等于全部」的模糊地带；
// 与角色的默认权限集（PermCodes 同样要求调用方显式传全量）是同一套约定。
type RoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PermCodes   []string `json:"permCodes"`

	ScopeAllBrands   bool     `json:"scopeAllBrands"`
	BrandCodes       []string `json:"brandCodes"`
	ScopeAllChannels bool     `json:"scopeAllChannels"`
	ChannelIDs       []uint64 `json:"channelIds"`
}

// RoleView 是角色列表/详情的返回形状，数据范围字段形如 docs/admin/10-rbac.md 约定的
// {allBrands, brands, allChannels, channelIds}（与登录/刷新/me 响应体的 scope 字段同构）。
type RoleView struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Builtin     bool     `json:"builtin"`
	PermCodes   []string `json:"permCodes"`
	UserCount   int64    `json:"userCount"`

	ScopeAllBrands   bool     `json:"scopeAllBrands"`
	BrandCodes       []string `json:"brandCodes"`
	ScopeAllChannels bool     `json:"scopeAllChannels"`
	ChannelIDs       []uint64 `json:"channelIds"`
}

// roleView 组装单个角色的返回视图：builtin 角色的 permCodes/数据范围恒为全量（不依赖
// role_permission/role_brand/role_channel 落库，经 auth.RBAC.RolePermCodes/RoleEffectiveScope
// 与鉴权中间件共用同一份判定）。
func (s *Service) roleView(ctx context.Context, role *model.Role) (*RoleView, error) {
	codes, err := s.rbac.RolePermCodes(ctx, role)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountUsersByRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	scope, err := s.rbac.RoleEffectiveScope(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	return &RoleView{
		ID: role.ID, Name: role.Name, Description: role.Description,
		Builtin: role.Builtin, PermCodes: codes, UserCount: n,
		ScopeAllBrands: scope.AllBrands, BrandCodes: scope.BrandCodeList(),
		ScopeAllChannels: scope.AllChannels, ChannelIDs: scope.ChannelIDList(),
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

// resolveRoleScopeInput 校验并构造 RoleInput 里的数据范围字段：brandCodes 必须是真实存在的
// 品牌 code；channelIds 必须是真实存在的渠道 id。返回的 auth.Scope 已把渠道范围与品牌范围
// 求交（品牌不在允许范围内的渠道 id 静默剔除，不报错——与 auth.RBAC.RoleEffectiveScope 的
// 读取口径一致：这是「求交」而不是「校验渠道所属品牌也必须同时勾选」）。
func (s *Service) resolveRoleScopeInput(ctx context.Context, in RoleInput) (auth.Scope, error) {
	scope := auth.Scope{AllBrands: in.ScopeAllBrands, AllChannels: in.ScopeAllChannels}
	if !in.ScopeAllBrands {
		codes := dedupeStrings(in.BrandCodes)
		brands := make(map[string]bool, len(codes))
		for _, code := range codes {
			if _, err := s.repo.GetBrandByCode(ctx, code); err != nil {
				return auth.Scope{}, errBadRequest(fmt.Sprintf("品牌 %q 不存在", code))
			}
			brands[code] = true
		}
		scope.Brands = brands
	}
	if !in.ScopeAllChannels {
		ids := dedupeUint64(in.ChannelIDs)
		info, err := s.repo.ChannelBrandCodesByIDs(ctx, ids)
		if err != nil {
			return auth.Scope{}, err
		}
		channelIDs := make(map[uint64]bool, len(ids))
		for _, id := range ids {
			brandCode, ok := info[id]
			if !ok {
				return auth.Scope{}, errBadRequest(fmt.Sprintf("渠道 id=%d 不存在", id))
			}
			if !scope.BrandAllowed(brandCode) {
				continue // 求交：不在允许品牌内的渠道勾选自动失效
			}
			channelIDs[id] = true
		}
		scope.ChannelIDs = channelIDs
	}
	return scope, nil
}

// saveRoleScope 把 scope 写回 role_brand/role_channel（全量替换，AllBrands/AllChannels=true
// 时对应表清空——role.scope_all_* 已经是 true，这两张表内容不会被读取，清空只是保持数据整洁）。
func (s *Service) saveRoleScope(ctx context.Context, roleID uint64, scope auth.Scope) error {
	if err := s.repo.ReplaceRoleBrands(ctx, roleID, scope.BrandCodeList()); err != nil {
		return err
	}
	if err := s.repo.ReplaceRoleChannels(ctx, roleID, scope.ChannelIDList()); err != nil {
		return err
	}
	return nil
}

// CreateRole 新增角色（新建角色一律非 builtin）。callerID 是发起请求的账号 ID：
// 非超管调用者不能授出自己没有的权限（permCodes 必须 ⊆ 调用者自身权限集），数据范围也必须
// ⊆ 调用者自身范围；两个「全部」标志位非超管一律不能授出，除非自己就是「全部」
// （docs/admin/10-rbac.md「数据权限」与最小权限约束两节）。
func (s *Service) CreateRole(ctx context.Context, callerID uint64, in RoleInput) (*RoleView, error) {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
	scope, err := s.resolveRoleScopeInput(ctx, in)
	if err != nil {
		return nil, err
	}
	if !callerRole.Builtin {
		if err := assertSubsetOf(codes, callerPermSet); err != nil {
			return nil, err
		}
		if !scope.SubsetOf(callerScope) {
			return nil, errPrivilegeEscalation()
		}
	}
	if _, err := s.repo.GetRoleByName(ctx, name); err == nil {
		return nil, errConflict(fmt.Sprintf("角色名 %q 已存在", name))
	}
	role := &model.Role{
		Name: name, Description: strings.TrimSpace(in.Description), Builtin: false,
		ScopeAllBrands: in.ScopeAllBrands, ScopeAllChannels: in.ScopeAllChannels,
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, role.ID, codes); err != nil {
		return nil, err
	}
	if err := s.saveRoleScope(ctx, role.ID, scope); err != nil {
		return nil, err
	}
	return s.roleView(ctx, role)
}

// UpdateRole 修改角色（整体替换 name/description/permCodes/数据范围）。builtin 角色不可编辑；
// 非超管调用者的约束同 CreateRole。
func (s *Service) UpdateRole(ctx context.Context, callerID, id uint64, in RoleInput) (*RoleView, error) {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
	scope, err := s.resolveRoleScopeInput(ctx, in)
	if err != nil {
		return nil, err
	}
	if !callerRole.Builtin {
		if err := assertSubsetOf(codes, callerPermSet); err != nil {
			return nil, err
		}
		if !scope.SubsetOf(callerScope) {
			return nil, errPrivilegeEscalation()
		}
	}
	if err := s.repo.UpdateRoleFields(ctx, id, map[string]any{
		"name": name, "description": strings.TrimSpace(in.Description),
		"scope_all_brands": in.ScopeAllBrands, "scope_all_channels": in.ScopeAllChannels,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, id, codes); err != nil {
		return nil, err
	}
	if err := s.saveRoleScope(ctx, id, scope); err != nil {
		return nil, err
	}
	role.Name = name
	role.Description = strings.TrimSpace(in.Description)
	return s.roleView(ctx, role)
}

// DeleteRole 删除角色。builtin 不可删；仍有账号挂靠时拒绝（409）；callerID 非超管时受最小权限
// 约束：只能删除权限集与数据范围都 ⊆ 自身的角色（删除本身不构成「提权」，但仍是「操作超出
// 自身权限范围的角色」——不能删一个权限/数据范围比自己大、只是恰好 0 成员的角色）。
func (s *Service) DeleteRole(ctx context.Context, callerID, id uint64) error {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
	if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, role); err != nil {
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

func dedupeUint64(in []uint64) []uint64 {
	seen := make(map[uint64]bool, len(in))
	out := make([]uint64, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
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
// roleId 必须指向已存在角色。callerID 非超管时受最小权限约束：目标角色是 builtin 只有超管可挂，
// 非 builtin 也要求目标角色权限集与数据范围都 ⊆ 调用者自身。
func (s *Service) CreateUser(ctx context.Context, callerID uint64, in CreateUserInput) (*UserView, error) {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
	if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, role); err != nil {
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

// UpdateUser 修改账号角色；保护「最后一个超级管理员」不被改走；callerID 非超管时受双向约束
// （不仅新角色要 ⊆ 调用者权限集/数据范围，目标账号「改动前」的角色也必须是调用者有权管理的——
// 否则非超管可以先把超管的角色改成自己权限子集内的角色，此时新角色校验能通过，
// 再顺势 ResetUserPassword 接管，逐个把超管剪到只剩一个）。
// 目标账号当前角色查不到（role_id 悬挂/查库出错）一律 fail-closed 拒绝。
func (s *Service) UpdateUser(ctx context.Context, callerID, id uint64, in UpdateUserInput) (*UserView, error) {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, currentRole); err != nil {
			return nil, err
		}
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, newRole); err != nil {
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

// ResetUserPassword 重置账号密码（最小长度 6）；callerID 非超管时受最小权限约束：目标账号挂
// builtin 角色只有超管可重置，非 builtin 也要求目标角色权限集与数据范围都 ⊆ 调用者自身
// （防止越权接管更高权限/更大数据范围的账号）。目标角色查不到（role_id 悬挂/查库出错）一律
// fail-closed 拒绝，不能因为查不到就静默跳过校验。
func (s *Service) ResetUserPassword(ctx context.Context, callerID, id uint64, password string) error {
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, targetRole); err != nil {
			return err
		}
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.UpdateUserFields(ctx, id, map[string]any{"password_hash": hash})
}

// DeleteUser 删除账号：不能删除自己；不能删除「最后一个超级管理员」；callerID 非超管时受最小
// 权限约束同 ResetUserPassword。目标角色查不到一律 fail-closed 拒绝。
func (s *Service) DeleteUser(ctx context.Context, callerID, id uint64) error {
	if id == callerID {
		return errBadRequest("不能删除自己")
	}
	callerRole, callerPermSet, callerScope, err := s.callerContext(ctx, callerID)
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
		if err := s.assertCanManageRole(ctx, callerRole, callerPermSet, callerScope, targetRole); err != nil {
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
