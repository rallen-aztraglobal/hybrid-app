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
// 超级管理员=全量 catalog；运营=全部 route+除系统设置三个 button 外的全部 button；只读=仅全部 route。
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

	op, ok := byName["运营"]
	if !ok {
		t.Fatal("应存在「运营」角色")
	}
	if op.Builtin {
		t.Error("运营不应 builtin")
	}
	wantOp := append([]string{}, perm.RouteCodes()...)
	excluded := map[string]bool{perm.StoreManage: true, perm.UserManage: true, perm.RoleManage: true}
	for _, c := range perm.ButtonCodes() {
		if !excluded[c] {
			wantOp = append(wantOp, c)
		}
	}
	assertSameCodes(t, "运营", op.PermCodes, wantOp)
	for _, c := range []string{perm.StoreManage, perm.UserManage, perm.RoleManage} {
		if contains(op.PermCodes, c) {
			t.Errorf("运营不应含系统设置权限 %q", c)
		}
	}

	viewer, ok := byName["只读"]
	if !ok {
		t.Fatal("应存在「只读」角色")
	}
	assertSameCodes(t, "只读", viewer.PermCodes, perm.RouteCodes())
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
