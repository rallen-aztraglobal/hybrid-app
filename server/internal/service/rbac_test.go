package service

import (
	"context"
	"sort"
	"testing"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/perm"
	"github.com/hybrid-app/server/internal/repo"
)

// mustDirectUser 绕开 service.CreateUser（它需要一个已存在的 callerID，无法用来建「第一个」账号）
// 直接落库建一个挂指定角色的账号，返回其 ID。测试专用，等价于生产 seed.EnsureBootstrapAdmin 的落库方式。
func mustDirectUser(t *testing.T, r *repo.Repo, username string, roleID uint64) uint64 {
	t.Helper()
	hash, err := auth.HashPassword("pw123456")
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	u := &model.AdminUser{Username: username, PasswordHash: hash, Role: model.RoleOperator, RoleID: roleID}
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("直接建账号 %s 失败: %v", username, err)
	}
	return u.ID
}

// roleIDsByName 取 EnsureRBAC 建好的三个内置角色 id。
func roleIDsByName(t *testing.T, svc *Service, ctx context.Context) (superID, opID, viewID uint64) {
	t.Helper()
	roles, err := svc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("列角色失败: %v", err)
	}
	for _, r := range roles {
		switch r.Name {
		case "超级管理员":
			superID = r.ID
		case "运营":
			opID = r.ID
		case "只读":
			viewID = r.ID
		}
	}
	if superID == 0 || opID == 0 || viewID == 0 {
		t.Fatalf("内置角色未齐全: super=%d op=%d view=%d", superID, opID, viewID)
	}
	return
}

// TestSeededRolePermCodes 验证 EnsureRBAC 建的三个初始角色权限集符合契约：
// 超级管理员=全量 catalog；运营=全部 route+全部 button 减去 perm.SystemManageCodes()；
// 只读=全部 route 减去 perm.SystemManageCodes()。
//
// 陷阱回归：role:manage/user:manage 的 kind 是 route，会被 perm.RouteCodes() 收进去——
// 只读角色若直接等于 perm.RouteCodes() 会自动白捡这两个敏感管理权限，必须显式排除
// （见 perm.SystemManageCodes 与 seed.viewerDefaultPerms 的注释）。
func TestSeededRolePermCodes(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	roles, err := svc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("列角色失败: %v", err)
	}
	byName := map[string]RoleView{}
	for _, r := range roles {
		byName[r.Name] = r
	}

	super, ok := byName["超级管理员"]
	if !ok {
		t.Fatal("应存在「超级管理员」角色")
	}
	if !super.Builtin {
		t.Error("超级管理员应 builtin=true")
	}
	assertSameCodes(t, "超级管理员", super.PermCodes, perm.AllCodes())
	// 超管不受陷阱影响：应仍能看到 role:manage / user:manage（否则超管自己都进不去角色/用户管理）。
	for _, c := range []string{perm.RoleManage, perm.UserManage, perm.StoreManage} {
		if !contains(super.PermCodes, c) {
			t.Errorf("超级管理员应包含 %q", c)
		}
	}

	systemManage := perm.SystemManageCodes()

	op, ok := byName["运营"]
	if !ok {
		t.Fatal("应存在「运营」角色")
	}
	if op.Builtin {
		t.Error("运营不应 builtin")
	}
	excluded := map[string]bool{}
	for _, c := range systemManage {
		excluded[c] = true
	}
	var wantOp []string
	for _, c := range perm.RouteCodes() {
		if !excluded[c] {
			wantOp = append(wantOp, c)
		}
	}
	for _, c := range perm.ButtonCodes() {
		if !excluded[c] {
			wantOp = append(wantOp, c)
		}
	}
	assertSameCodes(t, "运营", op.PermCodes, wantOp)

	viewer, ok := byName["只读"]
	if !ok {
		t.Fatal("应存在「只读」角色")
	}
	var wantViewer []string
	for _, c := range perm.RouteCodes() {
		if !excluded[c] {
			wantViewer = append(wantViewer, c)
		}
	}
	assertSameCodes(t, "只读", viewer.PermCodes, wantViewer)

	// 核心断言（陷阱回归）：运营、只读的默认权限集都不含 role:manage / user:manage / store:manage。
	for _, roleView := range []RoleView{op, viewer} {
		for _, c := range []string{perm.RoleManage, perm.UserManage, perm.StoreManage} {
			if contains(roleView.PermCodes, c) {
				t.Errorf("%s 不应包含敏感管理权限 %q", roleView.Name, c)
			}
		}
	}
}

// TestSystemManageCodesCatalogShape 验证 role:manage / user:manage 在 catalog 里的形状：
// kind=route（各自一个独立菜单的进入权，不再是「系统设置」页面内的按钮），且分别落在
// 「角色管理」「用户管理」独立模块下，而不是 settings 模块（前端把这两个从系统设置拆成
// 独立侧边栏菜单）；store:manage 仍留在 settings 模块、kind=button。
func TestSystemManageCodesCatalogShape(t *testing.T) {
	byCode := map[string]perm.Perm{}
	moduleByCode := map[string]string{}
	for _, m := range perm.Catalog() {
		for _, p := range m.Perms {
			byCode[p.Code] = p
			moduleByCode[p.Code] = m.Module
		}
	}

	roleManage, ok := byCode[perm.RoleManage]
	if !ok {
		t.Fatal("catalog 应含 role:manage")
	}
	if roleManage.Kind != perm.KindRoute {
		t.Errorf("role:manage 的 kind 应为 route，实际 %q", roleManage.Kind)
	}
	if moduleByCode[perm.RoleManage] == "settings" {
		t.Error("role:manage 不应再归在 settings 模块下（已拆成独立菜单）")
	}

	userManage, ok := byCode[perm.UserManage]
	if !ok {
		t.Fatal("catalog 应含 user:manage")
	}
	if userManage.Kind != perm.KindRoute {
		t.Errorf("user:manage 的 kind 应为 route，实际 %q", userManage.Kind)
	}
	if moduleByCode[perm.UserManage] == "settings" {
		t.Error("user:manage 不应再归在 settings 模块下（已拆成独立菜单）")
	}
	if moduleByCode[perm.RoleManage] == moduleByCode[perm.UserManage] {
		t.Error("role:manage 与 user:manage 应分属两个不同的独立模块")
	}

	storeManage, ok := byCode[perm.StoreManage]
	if !ok {
		t.Fatal("catalog 应含 store:manage")
	}
	if storeManage.Kind != perm.KindButton {
		t.Errorf("store:manage 的 kind 应仍为 button，实际 %q", storeManage.Kind)
	}
	if moduleByCode[perm.StoreManage] != "settings" {
		t.Errorf("store:manage 应仍归在 settings 模块下，实际 %q", moduleByCode[perm.StoreManage])
	}

	// code 字符串本身不应因这次拆分而改变（存量 role_permission 数据不能失效）。
	if perm.RoleManage != "role:manage" {
		t.Errorf("role:manage 的 code 字符串不应变化，实际 %q", perm.RoleManage)
	}
	if perm.UserManage != "user:manage" {
		t.Errorf("user:manage 的 code 字符串不应变化，实际 %q", perm.UserManage)
	}
}

// TestRoleCRUDValidation 覆盖角色 CRUD 的校验规则（唯一契约 docs/admin/10-rbac.md）。
// 调用者用超管账号，排除 B2 最小权限约束的干扰（B2 单独用 TestLeastPrivilegeConstraints 覆盖）。
func TestRoleCRUDValidation(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	caller := mustDirectUser(t, r, "root", superID)

	// 创建：name 必填。
	if _, err := svc.CreateRole(ctx, caller, RoleInput{Name: "  ", PermCodes: nil}); err == nil {
		t.Error("空 name 应被拒绝")
	}
	// 创建：permCodes 必须在 catalog 内。
	if _, err := svc.CreateRole(ctx, caller, RoleInput{Name: "测试角色A", PermCodes: []string{"not:a:real:code"}}); err == nil {
		t.Error("非法权限 code 应被拒绝")
	}
	// 机器权限不可分配。
	if _, err := svc.CreateRole(ctx, caller, RoleInput{Name: "测试角色B", PermCodes: []string{perm.RunnerPerm}}); err == nil {
		t.Error("build:runner 不应可分配给角色")
	}

	// 正常创建。
	role, err := svc.CreateRole(ctx, caller, RoleInput{
		Name: "客服", Description: "只看渠道和推送", PermCodes: []string{perm.PageChannels, perm.PagePush, perm.PageChannels},
	})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if role.Builtin {
		t.Error("新建角色不应 builtin")
	}
	assertSameCodes(t, "客服", role.PermCodes, []string{perm.PageChannels, perm.PagePush}) // 去重

	// 重名应拒绝（409）。
	if _, err := svc.CreateRole(ctx, caller, RoleInput{Name: "客服", PermCodes: nil}); err == nil {
		t.Error("重名角色应被拒绝")
	}

	// 更新：改名 + 换权限集。
	updated, err := svc.UpdateRole(ctx, caller, role.ID, RoleInput{Name: "客服2", Description: "改过", PermCodes: []string{perm.PageDomains}})
	if err != nil {
		t.Fatalf("更新角色失败: %v", err)
	}
	if updated.Name != "客服2" {
		t.Errorf("角色名应更新为 客服2，实际 %q", updated.Name)
	}
	assertSameCodes(t, "客服2", updated.PermCodes, []string{perm.PageDomains})

	// 更新：改名撞到已存在角色名应 409。
	if _, err := svc.CreateRole(ctx, caller, RoleInput{Name: "占位角色", PermCodes: nil}); err != nil {
		t.Fatalf("创建占位角色失败: %v", err)
	}
	if _, err := svc.UpdateRole(ctx, caller, role.ID, RoleInput{Name: "占位角色", PermCodes: nil}); err == nil {
		t.Error("改名撞到已存在角色名应被拒绝")
	}

	// builtin 角色不可改/删。
	if _, err := svc.UpdateRole(ctx, caller, superID, RoleInput{Name: "改个名", PermCodes: nil}); err == nil {
		t.Error("builtin 角色不可编辑")
	}
	if err := svc.DeleteRole(ctx, caller, superID); err == nil {
		t.Error("builtin 角色不可删除")
	}

	// 挂了用户的角色不可删（409）；未挂用户的角色可删。
	if _, err := svc.CreateUser(ctx, caller, CreateUserInput{Username: "kf1", Password: "pw123456", RoleID: role.ID}); err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	if err := svc.DeleteRole(ctx, caller, role.ID); err == nil {
		t.Error("挂了账号的角色不应可删除")
	}
	unused, err := svc.CreateRole(ctx, caller, RoleInput{Name: "闲置角色", PermCodes: nil})
	if err != nil {
		t.Fatalf("创建闲置角色失败: %v", err)
	}
	if err := svc.DeleteRole(ctx, caller, unused.ID); err != nil {
		t.Errorf("未挂账号的角色应可删除: %v", err)
	}
}

// TestUserProtectionRules 覆盖用户管理的保护规则：不能删自己、不能删/改走最后一个超级管理员、
// username 唯一 /≤64 字节 / 保留名、密码最小长度。调用者用超管账号，排除 B2 干扰。
func TestUserProtectionRules(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, viewerID := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	// username 必填 / 长度限制 / 保留名（B1）。
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "", Password: "pw123456", RoleID: viewerID}); err == nil {
		t.Error("空 username 应被拒绝")
	}
	tooLong := make([]byte, 65)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: string(tooLong), Password: "pw123456", RoleID: viewerID}); err == nil {
		t.Error("超过 64 字节的 username 应被拒绝")
	}
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "runner", Password: "pw123456", RoleID: viewerID}); err == nil {
		t.Error("username=runner 是保留名，应被拒绝（B1）")
	}
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "RUNNER", Password: "pw123456", RoleID: viewerID}); err == nil {
		t.Error("保留名校验应大小写不敏感")
	}
	// 密码最小长度（M5）。
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "shortpw", Password: "abc12", RoleID: viewerID}); err == nil {
		t.Error("密码短于 6 位应被拒绝")
	}
	// roleId 必须指向已存在角色。
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "u1", Password: "pw123456", RoleID: 999999}); err == nil {
		t.Error("不存在的 roleId 应被拒绝")
	}

	// 建第一个超级管理员账号。
	admin1, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "admin1", Password: "pw123456", RoleID: superID})
	if err != nil {
		t.Fatalf("创建超管账号失败: %v", err)
	}
	// username 唯一。
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "admin1", Password: "pw123456", RoleID: viewerID}); err == nil {
		t.Error("重复 username 应被拒绝")
	}

	// 不能删除自己（以 admin1 自己的身份删自己）。
	if err := svc.DeleteUser(ctx, admin1.ID, admin1.ID); err == nil {
		t.Error("不能删除自己")
	}

	// 建第二个账号（只读）。
	viewerUser, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "viewer1", Password: "pw123456", RoleID: viewerID})
	if err != nil {
		t.Fatalf("创建只读账号失败: %v", err)
	}

	// 此时超管有 root + admin1 两个，改走 admin1 的角色应允许（非最后一个）。
	if _, err := svc.UpdateUser(ctx, root, admin1.ID, UpdateUserInput{RoleID: viewerID}); err != nil {
		t.Errorf("非最后一个超管时应可改角色: %v", err)
	}
	// 现在只剩 root 一个超管：改走 root 的角色应被拒绝。
	if _, err := svc.UpdateUser(ctx, admin1.ID, root, UpdateUserInput{RoleID: viewerID}); err == nil {
		t.Error("不能把最后一个超级管理员改成其他角色")
	}
	// 删除唯一超管（root）也应被拒绝。
	if err := svc.DeleteUser(ctx, viewerUser.ID, root); err == nil {
		t.Error("不能删除最后一个超级管理员")
	}

	// 重置密码：账号不存在应报错；密码太短应报错；正常应成功。
	if err := svc.ResetUserPassword(ctx, root, 999999, "newpw12345"); err == nil {
		t.Error("重置不存在账号的密码应报错")
	}
	if err := svc.ResetUserPassword(ctx, root, viewerUser.ID, "123"); err == nil {
		t.Error("重置密码短于 6 位应报错")
	}
	if err := svc.ResetUserPassword(ctx, root, viewerUser.ID, "newpw12345"); err != nil {
		t.Errorf("重置密码应成功: %v", err)
	}
}

// TestLeastPrivilegeConstraints 覆盖 B2「最小权限约束」的每条拒绝路径 + 超管放行。
func TestLeastPrivilegeConstraints(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, opID, viewID := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	// 一个仅有 page:channels + page:push 的受限角色，及挂靠它的调用者账号。
	limitedRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "受限角色", PermCodes: []string{perm.PageChannels, perm.PagePush},
	})
	if err != nil {
		t.Fatalf("创建受限角色失败: %v", err)
	}
	limitedCaller := mustDirectUser(t, r, "limited", limitedRole.ID)

	// ---- CreateRole/UpdateRole：permCodes 必须 ⊆ 调用者自身权限集 ----
	if _, err := svc.CreateRole(ctx, limitedCaller, RoleInput{
		Name: "越权角色", PermCodes: []string{perm.PageDomains}, // 调用者没有 page:domains
	}); err == nil {
		t.Error("非超管创建角色时 permCodes 超出自身权限集应被拒绝")
	}
	okRole, err := svc.CreateRole(ctx, limitedCaller, RoleInput{
		Name: "合规角色", PermCodes: []string{perm.PageChannels},
	})
	if err != nil {
		t.Fatalf("非超管创建角色（权限集是自身子集）应成功: %v", err)
	}
	if _, err := svc.UpdateRole(ctx, limitedCaller, okRole.ID, RoleInput{
		Name: "合规角色", PermCodes: []string{perm.PageDomains},
	}); err == nil {
		t.Error("非超管修改角色时 permCodes 超出自身权限集应被拒绝")
	}
	// 超管不受限。
	if _, err := svc.CreateRole(ctx, root, RoleInput{Name: "超管建的角色", PermCodes: []string{perm.RoleManage}}); err != nil {
		t.Errorf("超管创建角色不应受权限子集约束: %v", err)
	}

	// ---- CreateUser/UpdateUser：目标 roleId 是 builtin → 仅超管；非 builtin 也要求 ⊆ 调用者权限集 ----
	if _, err := svc.CreateUser(ctx, limitedCaller, CreateUserInput{
		Username: "wannabe-admin", Password: "pw123456", RoleID: superID, // builtin
	}); err == nil {
		t.Error("非超管把新账号挂成 builtin 角色应被拒绝")
	}
	if _, err := svc.CreateUser(ctx, limitedCaller, CreateUserInput{
		Username: "wannabe-op", Password: "pw123456", RoleID: opID, // 运营权限集超出 limitedCaller
	}); err == nil {
		t.Error("非超管把新账号挂成权限超出自身的角色应被拒绝")
	}
	compliantUser, err := svc.CreateUser(ctx, limitedCaller, CreateUserInput{
		Username: "compliant", Password: "pw123456", RoleID: okRole.ID, // okRole ⊆ limitedCaller 权限集
	})
	if err != nil {
		t.Fatalf("非超管创建权限集是自身子集的账号应成功: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, limitedCaller, compliantUser.ID, UpdateUserInput{RoleID: superID}); err == nil {
		t.Error("非超管把账号改成 builtin 角色应被拒绝")
	}
	if _, err := svc.UpdateUser(ctx, limitedCaller, compliantUser.ID, UpdateUserInput{RoleID: viewID}); err == nil {
		t.Error("非超管把账号改成权限超出自身的角色（只读含 page:domains 等）应被拒绝")
	}
	// 超管不受限：可以把账号挂成 builtin。
	if _, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "root2", Password: "pw123456", RoleID: superID}); err != nil {
		t.Errorf("超管创建 builtin 账号不应受限: %v", err)
	}

	// ---- ResetUserPassword/DeleteUser：目标用户 builtin 角色 → 仅超管；非 builtin 也要求 ⊆ 调用者权限集 ----
	if err := svc.ResetUserPassword(ctx, limitedCaller, root, "newpw12345"); err == nil {
		t.Error("非超管重置 builtin 角色账号的密码应被拒绝")
	}
	if err := svc.DeleteUser(ctx, limitedCaller, root); err == nil {
		t.Error("非超管删除 builtin 角色账号应被拒绝")
	}
	// 建一个挂「只读」角色（权限超出 limitedCaller：只读含 page:domains 等 limitedCaller 没有的权限）的账号。
	viewerUser := mustDirectUser(t, r, "viewer-target", viewID)
	if err := svc.ResetUserPassword(ctx, limitedCaller, viewerUser, "newpw12345"); err == nil {
		t.Error("非超管重置权限超出自身的账号密码应被拒绝")
	}
	if err := svc.DeleteUser(ctx, limitedCaller, viewerUser); err == nil {
		t.Error("非超管删除权限超出自身的账号应被拒绝")
	}
	// 合规目标（compliantUser 的角色 ⊆ limitedCaller）：应放行。
	if err := svc.ResetUserPassword(ctx, limitedCaller, compliantUser.ID, "newpw12345"); err != nil {
		t.Errorf("非超管重置权限集是自身子集的账号密码应成功: %v", err)
	}
	if err := svc.DeleteUser(ctx, limitedCaller, compliantUser.ID); err != nil {
		t.Errorf("非超管删除权限集是自身子集的账号应成功: %v", err)
	}
	// 超管不受限：可以重置/删除 builtin 账号（用另一个超管账号操作，避免碰到「最后一个超管」保护）。
	root2, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "root3", Password: "pw123456", RoleID: superID})
	if err != nil {
		t.Fatalf("创建第三个超管账号失败: %v", err)
	}
	if err := svc.ResetUserPassword(ctx, root, root2.ID, "newpw12345"); err != nil {
		t.Errorf("超管重置 builtin 账号密码不应受限: %v", err)
	}
}

// TestUpdateUserRejectsChangingBuiltinTargetEvenWithinOwnScope 是应修2的回归测试：非超管调用者
// 仅持 user:manage 时,不能把一个超管账号改成「自己权限子集内」的角色——必须同时校验目标账号
// 「改动前」的角色是否在调用者管理范围内,不能只查「改动后」的新角色。
func TestUpdateUserRejectsChangingBuiltinTargetEvenWithinOwnScope(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	// 仅持 user:manage（+ page:settings 才能进设置页）的受限角色，不是 builtin。
	limitedRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "账号管理员", PermCodes: []string{perm.PageSettings, perm.UserManage},
	})
	if err != nil {
		t.Fatalf("创建受限角色失败: %v", err)
	}
	caller := mustDirectUser(t, r, "acct-admin", limitedRole.ID)

	// 第二个超管账号作为攻击目标（避免触发「最后一个超管」保护，聚焦测双向校验本身）。
	victim, err := svc.CreateUser(ctx, root, CreateUserInput{Username: "victim-admin", Password: "pw123456", RoleID: superID})
	if err != nil {
		t.Fatalf("创建目标超管账号失败: %v", err)
	}

	// 把目标改成 caller 自己的角色：新角色 ⊆ caller 自身权限（若只查新角色会放行），
	// 但目标当前是 builtin，非超管调用者不能动它——应被拒绝。
	if _, err := svc.UpdateUser(ctx, caller, victim.ID, UpdateUserInput{RoleID: limitedRole.ID}); err == nil {
		t.Error("非超管把超管账号改成自己权限子集内的角色也应被拒绝（B2 双向校验）")
	}

	// 佐证：目标账号角色应仍是原来的超管角色，未被越权更改。
	list, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("列账号失败: %v", err)
	}
	for _, u := range list {
		if u.ID == victim.ID && u.RoleID != superID {
			t.Errorf("越权更新应被完全拒绝，目标角色不应变化，实际 roleId=%d", u.RoleID)
		}
	}

	// 反向：超管本身不受此限制。
	if _, err := svc.UpdateUser(ctx, root, victim.ID, UpdateUserInput{RoleID: limitedRole.ID}); err != nil {
		t.Errorf("超管修改任意账号角色不应受限: %v", err)
	}
}

// TestResetAndDeleteFailClosedOnDanglingRole 是应修3的回归测试：目标账号的 role_id 查不到对应角色
// （悬挂/脏数据）时，ResetUserPassword/DeleteUser 必须直接拒绝，不能因为查不到就静默跳过校验、
// 放行操作——即便调用者是超管，也应统一 fail-closed（避免不确定的数据状态被当成「可以放行」）。
func TestResetAndDeleteFailClosedOnDanglingRole(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	dangling, err := svc.CreateRole(ctx, root, RoleInput{Name: "临时角色", PermCodes: nil})
	if err != nil {
		t.Fatalf("创建临时角色失败: %v", err)
	}
	victim := mustDirectUser(t, r, "victim", dangling.ID)

	// 直接删掉角色行制造 role_id 悬挂（正常业务流程里 DeleteRole 会因 0 成员之外的挂靠而受阻，
	// 这里用 repo 直接删模拟脏数据/异常场景，验证 service 层的 fail-closed 兜底）。
	if err := r.DeleteRole(ctx, dangling.ID); err != nil {
		t.Fatalf("删除角色失败: %v", err)
	}

	if err := svc.ResetUserPassword(ctx, root, victim, "newpw12345"); err == nil {
		t.Error("目标角色悬挂时重置密码应 fail-closed 拒绝（应修3）")
	}
	if err := svc.DeleteUser(ctx, root, victim); err == nil {
		t.Error("目标角色悬挂时删除账号应 fail-closed 拒绝（应修3）")
	}
}

// TestDeleteRoleLeastPrivilege 是应修5的回归测试：非超管仅持 role:manage 时，不能删除权限集
// 超出自身的角色（即便该角色 0 成员、删除本身不构成「提权」）；权限集 ⊆ 自身的角色可以删；
// 超管不受限。
func TestDeleteRoleLeastPrivilege(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	limitedRole, err := svc.CreateRole(ctx, root, RoleInput{Name: "受限角色2", PermCodes: []string{perm.PageChannels}})
	if err != nil {
		t.Fatalf("创建受限角色失败: %v", err)
	}
	limitedCaller := mustDirectUser(t, r, "limited2", limitedRole.ID)

	biggerRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "更大权限角色", PermCodes: []string{perm.PageChannels, perm.PageDomains},
	})
	if err != nil {
		t.Fatalf("创建更大权限角色失败: %v", err)
	}
	if err := svc.DeleteRole(ctx, limitedCaller, biggerRole.ID); err == nil {
		t.Error("非超管不能删除权限超出自身的角色，即便 0 成员")
	}

	subRole, err := svc.CreateRole(ctx, limitedCaller, RoleInput{Name: "子集角色", PermCodes: nil})
	if err != nil {
		t.Fatalf("非超管创建权限集是自身子集(空集)的角色应成功: %v", err)
	}
	if err := svc.DeleteRole(ctx, limitedCaller, subRole.ID); err != nil {
		t.Errorf("非超管应能删除权限集 ⊆ 自身的角色: %v", err)
	}

	if err := svc.DeleteRole(ctx, root, biggerRole.ID); err != nil {
		t.Errorf("超管删除角色不应受权限子集约束: %v", err)
	}
}

// TestRoleScopeCRUD 覆盖角色数据范围的创建/更新/持久化正确性（含 GORM 零值陷阱回归：
// Role.ScopeAllBrands/ScopeAllChannels 带 gorm:"default:true"，结构体 Create 对 false 零值会
// 静默回填成 true，repo.CreateRole 必须堵住这个陷阱，见该函数注释）。
func TestRoleScopeCRUD(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	ap001, err := svc.CreateChannel(ctx, CreateChannelInput{BrandCode: "ap", FlavorName: "ap001", PalCode: "P1", AppName: "A"})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	// 创建：指定品牌范围(仅 ap) + 指定渠道范围(仅 ap001)。
	role, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "范围角色", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: false, BrandCodes: []string{"ap"},
		ScopeAllChannels: false, ChannelIDs: []uint64{ap001.ID},
	})
	if err != nil {
		t.Fatalf("创建限定范围角色失败: %v", err)
	}
	if role.ScopeAllBrands {
		t.Error("scopeAllBrands 应为 false（GORM 零值陷阱回归：不应被悄悄存成 true）")
	}
	if role.ScopeAllChannels {
		t.Error("scopeAllChannels 应为 false（同上）")
	}
	if len(role.BrandCodes) != 1 || role.BrandCodes[0] != "ap" {
		t.Errorf("brandCodes 应为 [ap]，实际 %v", role.BrandCodes)
	}
	if len(role.ChannelIDs) != 1 || role.ChannelIDs[0] != ap001.ID {
		t.Errorf("channelIds 应为 [%d]，实际 %v", ap001.ID, role.ChannelIDs)
	}

	// ListRoles 返回的视图也应带上同样的范围（不是只有单条 CreateRole 的返回值对，列表也要对）。
	list, err := svc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("列角色失败: %v", err)
	}
	found := false
	for _, v := range list {
		if v.ID == role.ID {
			found = true
			if v.ScopeAllBrands || len(v.BrandCodes) != 1 || v.BrandCodes[0] != "ap" {
				t.Errorf("列表里的角色范围不符: %+v", v)
			}
		}
	}
	if !found {
		t.Fatal("列表里应能找到刚创建的角色")
	}

	// 更新：改成全量品牌范围 + 指定渠道范围清空（切回「全部」）。
	updated, err := svc.UpdateRole(ctx, root, role.ID, RoleInput{
		Name: "范围角色", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: true, ScopeAllChannels: true,
	})
	if err != nil {
		t.Fatalf("更新角色范围失败: %v", err)
	}
	if !updated.ScopeAllBrands || !updated.ScopeAllChannels {
		t.Errorf("更新后应为全量范围，实际 %+v", updated)
	}
	if len(updated.BrandCodes) != 0 || len(updated.ChannelIDs) != 0 {
		t.Errorf("全量范围下 brandCodes/channelIds 应为空，实际 %v / %v", updated.BrandCodes, updated.ChannelIDs)
	}
}

// TestRoleScopeInputValidation 验证角色数据范围入参校验：不存在的品牌 code / 渠道 id 应被拒绝；
// 渠道范围与品牌范围的求交语义（品牌未勾选时，该品牌下的渠道 id 静默从有效范围里剔除，不报错）。
func TestRoleScopeInputValidation(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	if _, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "坏品牌角色", ScopeAllBrands: false, BrandCodes: []string{"zz-not-exist"},
	}); err == nil {
		t.Error("不存在的品牌 code 应被拒绝")
	}
	if _, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "坏渠道角色", ScopeAllChannels: false, ChannelIDs: []uint64{999999},
	}); err == nil {
		t.Error("不存在的渠道 id 应被拒绝")
	}

	// 求交：渠道属于 bp，但品牌范围只给了 ap —— 不报错，渠道范围里这条静默失效。
	bpCh, err := svc.CreateChannel(ctx, CreateChannelInput{BrandCode: "bp", FlavorName: "bp001", PalCode: "P1", AppName: "B"})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	role, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "求交角色", ScopeAllBrands: false, BrandCodes: []string{"ap"},
		ScopeAllChannels: false, ChannelIDs: []uint64{bpCh.ID},
	})
	if err != nil {
		t.Fatalf("求交场景不应报错: %v", err)
	}
	if len(role.ChannelIDs) != 0 {
		t.Errorf("bp 渠道不在允许品牌(ap)内，求交后应被剔除，实际 channelIds=%v", role.ChannelIDs)
	}
}

// TestRoleScopeLeastPrivilege 覆盖数据权限的最小权限约束（docs/admin/10-rbac.md）：
// 非超管建/改角色时数据范围必须 ⊆ 自身范围；两个「全部」标志位非超管不能授出。
func TestRoleScopeLeastPrivilege(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	apCh, err := svc.CreateChannel(ctx, CreateChannelInput{BrandCode: "ap", FlavorName: "ap777", PalCode: "P1", AppName: "A"})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	// 调用者角色：仅 ap 品牌范围（全部渠道）。
	apOnlyRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "ap专员", PermCodes: []string{perm.PageChannels, perm.RoleManage},
		ScopeAllBrands: false, BrandCodes: []string{"ap"}, ScopeAllChannels: true,
	})
	if err != nil {
		t.Fatalf("创建 ap 专员角色失败: %v", err)
	}
	apCaller := mustDirectUser(t, r, "ap-caller", apOnlyRole.ID)

	// 不能建一个 ap+bp 的角色（超出自身品牌范围）。
	if _, err := svc.CreateRole(ctx, apCaller, RoleInput{
		Name: "越权ap+bp", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: false, BrandCodes: []string{"ap", "bp"}, ScopeAllChannels: true,
	}); err == nil {
		t.Error("非超管不能建一个数据范围超出自身的角色（多了 bp）")
	}
	// 不能建一个 scopeAllBrands=true 的角色（自己不是「全部」，不能把「全部」标志位授出去）。
	if _, err := svc.CreateRole(ctx, apCaller, RoleInput{
		Name: "越权全量品牌", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: true, ScopeAllChannels: true,
	}); err == nil {
		t.Error("非超管(非全量品牌范围)不能建一个 scopeAllBrands=true 的角色")
	}
	// 不能建一个 scopeAllChannels=true 但品牌超出自身的角色。
	if _, err := svc.CreateRole(ctx, apCaller, RoleInput{
		Name: "越权渠道全量但品牌超出", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: false, BrandCodes: []string{"bp"}, ScopeAllChannels: true,
	}); err == nil {
		t.Error("非超管不能建一个品牌超出自身范围的角色（即便渠道范围是全量）")
	}
	// 可以建一个 ap 子集（甚至更窄：仅 ap777 一个渠道）的角色。
	subRole, err := svc.CreateRole(ctx, apCaller, RoleInput{
		Name: "ap子集角色", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: false, BrandCodes: []string{"ap"}, ScopeAllChannels: false, ChannelIDs: []uint64{apCh.ID},
	})
	if err != nil {
		t.Fatalf("非超管创建数据范围是自身子集的角色应成功: %v", err)
	}
	if subRole.ScopeAllBrands || len(subRole.BrandCodes) != 1 || subRole.BrandCodes[0] != "ap" {
		t.Errorf("子集角色范围不符: %+v", subRole)
	}

	// 反向：把 apOnlyRole 改成全量品牌范围，应被拒绝（改角色同样受约束，不止创建）。
	if _, err := svc.UpdateRole(ctx, apCaller, apOnlyRole.ID, RoleInput{
		Name: "ap专员", PermCodes: []string{perm.PageChannels, perm.RoleManage},
		ScopeAllBrands: true, ScopeAllChannels: true,
	}); err == nil {
		t.Error("非超管不能把角色范围改成超出自身的全量范围")
	}

	// 超管不受限：可以建 scopeAllBrands=true 的角色。
	if _, err := svc.CreateRole(ctx, root, RoleInput{Name: "超管建的全量角色", ScopeAllBrands: true, ScopeAllChannels: true}); err != nil {
		t.Errorf("超管创建全量范围角色不应受限: %v", err)
	}
}

// TestUserScopeLeastPrivilege 验证最小权限约束同样通过 assertCanManageRole 落到用户管理上：
// 非超管不能把账号挂到一个数据范围超出自身的角色。
func TestUserScopeLeastPrivilege(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	superID, _, _ := roleIDsByName(t, svc, ctx)
	root := mustDirectUser(t, r, "root", superID)

	apOnlyRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "ap专员2", PermCodes: []string{perm.PageChannels, perm.UserManage},
		ScopeAllBrands: false, BrandCodes: []string{"ap"}, ScopeAllChannels: true,
	})
	if err != nil {
		t.Fatalf("创建 ap 专员角色失败: %v", err)
	}
	apCaller := mustDirectUser(t, r, "ap-caller2", apOnlyRole.ID)

	widerRole, err := svc.CreateRole(ctx, root, RoleInput{
		Name: "ap+bp角色", PermCodes: []string{perm.PageChannels},
		ScopeAllBrands: false, BrandCodes: []string{"ap", "bp"}, ScopeAllChannels: true,
	})
	if err != nil {
		t.Fatalf("创建 ap+bp 角色失败: %v", err)
	}

	// 非超管不能把新账号挂到一个数据范围比自己大的角色上。
	if _, err := svc.CreateUser(ctx, apCaller, CreateUserInput{
		Username: "wannabe-wide", Password: "pw123456", RoleID: widerRole.ID,
	}); err == nil {
		t.Error("非超管不能把账号挂到数据范围超出自身的角色（ap+bp 超出 ap-only）")
	}

	// 挂到自身范围的子集角色应成功。
	if _, err := svc.CreateUser(ctx, apCaller, CreateUserInput{
		Username: "compliant-scope", Password: "pw123456", RoleID: apOnlyRole.ID,
	}); err != nil {
		t.Errorf("非超管创建数据范围是自身子集的账号应成功: %v", err)
	}
}

func assertSameCodes(t *testing.T, label string, got, want []string) {
	t.Helper()
	g := append([]string{}, got...)
	w := append([]string{}, want...)
	sort.Strings(g)
	sort.Strings(w)
	if len(g) != len(w) {
		t.Errorf("%s 权限集数量不符：got=%v want=%v", label, g, w)
		return
	}
	for i := range g {
		if g[i] != w[i] {
			t.Errorf("%s 权限集不符：got=%v want=%v", label, g, w)
			return
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
