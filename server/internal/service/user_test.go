package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
)

// ---------- 新建账号 ----------

// TestCreateUser_AdminAndUser 验证可以新建 admin 与 user 两档角色（对应需求 #4 #5）。
func TestCreateUser_AdminAndUser(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "alice", Password: "Passw0rd!1", Role: model.RoleUser})
	if err != nil {
		t.Fatalf("创建 user 账号应成功: %v", err)
	}
	if u.Role != model.RoleUser || !u.Enabled {
		t.Errorf("新账号应为 user 角色且默认启用，实际 role=%s enabled=%v", u.Role, u.Enabled)
	}

	a, err := svc.CreateUser(ctx, CreateUserInput{Username: "bob", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("创建 admin 账号应成功: %v", err)
	}
	if a.Role != model.RoleAdmin {
		t.Errorf("新账号应为 admin 角色，实际 %s", a.Role)
	}
}

// TestCreateUser_DuplicateUsernameCaseInsensitive 验证用户名大小写不敏感唯一（对应需求 #6）。
func TestCreateUser_DuplicateUsernameCaseInsensitive(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, CreateUserInput{Username: "Carol", Password: "Passw0rd!1", Role: model.RoleUser}); err != nil {
		t.Fatalf("首次创建应成功: %v", err)
	}
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "carol", Password: "Passw0rd!1", Role: model.RoleUser})
	if err == nil {
		t.Fatal("大小写不同但同名应被拒绝")
	}
	se := AsError(err)
	if se.Code != 409 {
		t.Errorf("重名应返回 409，实际 %d", se.Code)
	}
}

// TestCreateUser_EmptyUsernameRejected 验证空用户名（含纯空白）被拒绝（对应需求 #7）。
func TestCreateUser_EmptyUsernameRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "   ", Password: "Passw0rd!1", Role: model.RoleUser})
	if err == nil {
		t.Fatal("空用户名应被拒绝")
	}
	if AsError(err).Code != 400 {
		t.Errorf("应返回 400，实际 %d", AsError(err).Code)
	}
}

// TestCreateUser_InvalidRoleRejected 验证未知角色被拒绝（对应需求 #8）。
func TestCreateUser_InvalidRoleRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "dave", Password: "Passw0rd!1", Role: "manager"})
	if err == nil || AsError(err).Code != 400 {
		t.Fatalf("非法角色应 400，实际 err=%v", err)
	}
}

// TestCreateUser_OperatorRoleRejected 验证历史角色 operator 不再被接受（对应需求 #9）。
func TestCreateUser_OperatorRoleRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "erin", Password: "Passw0rd!1", Role: "operator"})
	if err == nil || AsError(err).Code != 400 {
		t.Fatalf("operator 角色应被拒绝，实际 err=%v", err)
	}
}

// TestCreateUser_ViewerRoleRejected 验证历史角色 viewer 不再被接受（对应需求 #10）。
func TestCreateUser_ViewerRoleRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "frank", Password: "Passw0rd!1", Role: "viewer"})
	if err == nil || AsError(err).Code != 400 {
		t.Fatalf("viewer 角色应被拒绝，实际 err=%v", err)
	}
}

// TestCreateUser_PasswordHashedNotPlaintext 验证密码用 bcrypt 哈希、不落明文（对应需求 #11 #12）。
func TestCreateUser_PasswordHashedNotPlaintext(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	const plain = "S3cretPass!"
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "grace", Password: plain, Role: model.RoleUser})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	stored, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if stored.PasswordHash == plain {
		t.Fatal("密码不应以明文存储")
	}
	if !strings.HasPrefix(stored.PasswordHash, "$2") {
		t.Errorf("密码哈希应为 bcrypt 格式（$2 前缀），实际 %q", stored.PasswordHash)
	}
	if !auth.CheckPassword(stored.PasswordHash, plain) {
		t.Error("存储的哈希应能校验回原密码")
	}
}

// TestCreateUser_EnabledFalseAtCreation 验证「新建即禁用」不会被 GORM 的零值省略行为
// 静默改写成默认 true（repo.CreateUser 用 Select("*") 规避该问题）。
func TestCreateUser_EnabledFalseAtCreation(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	disabled := false
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "henry", Password: "Passw0rd!1", Role: model.RoleUser, Enabled: &disabled})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if u.Enabled {
		t.Fatal("显式 Enabled:false 创建的账号应保持禁用")
	}
	stored, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if stored.Enabled {
		t.Fatal("落库后仍应为禁用（未被静默改写成启用）")
	}
}

// ---------- 修改账号（角色 / 启停用） ----------

// TestUpdateUser_ChangeRoleAndToggleEnabled 验证 admin 可修改他人角色、禁用、再启用（对应需求 #14 #15 #16）。
func TestUpdateUser_ChangeRoleAndToggleEnabled(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	actor, err := svc.CreateUser(ctx, CreateUserInput{Username: "actor1", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("建 actor 失败: %v", err)
	}
	target, err := svc.CreateUser(ctx, CreateUserInput{Username: "target1", Password: "Passw0rd!1", Role: model.RoleUser})
	if err != nil {
		t.Fatalf("建 target 失败: %v", err)
	}

	newRole := model.RoleAdmin
	u, err := svc.UpdateUser(ctx, actor.ID, target.ID, UpdateUserInput{Role: &newRole})
	if err != nil {
		t.Fatalf("改角色应成功: %v", err)
	}
	if u.Role != model.RoleAdmin {
		t.Errorf("角色应变为 admin，实际 %s", u.Role)
	}

	disabled := false
	u, err = svc.UpdateUser(ctx, actor.ID, target.ID, UpdateUserInput{Enabled: &disabled})
	if err != nil {
		t.Fatalf("禁用应成功: %v", err)
	}
	if u.Enabled {
		t.Error("账号应已被禁用")
	}

	enabled := true
	u, err = svc.UpdateUser(ctx, actor.ID, target.ID, UpdateUserInput{Enabled: &enabled})
	if err != nil {
		t.Fatalf("重新启用应成功: %v", err)
	}
	if !u.Enabled {
		t.Error("账号应已重新启用")
	}
}

// TestUpdateUser_CannotDisableSelf 验证 admin 不能禁用自己（对应需求 #17）。
func TestUpdateUser_CannotDisableSelf(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	// 另建一个 admin，避免这里同时触发「最后一个启用 admin」规则，纯粹测试自禁用规则。
	if _, err := svc.CreateUser(ctx, CreateUserInput{Username: "other-admin", Password: "Passw0rd!1", Role: model.RoleAdmin}); err != nil {
		t.Fatalf("建第二个 admin 失败: %v", err)
	}
	self, err := svc.CreateUser(ctx, CreateUserInput{Username: "self1", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("建自身账号失败: %v", err)
	}
	disabled := false
	_, err = svc.UpdateUser(ctx, self.ID, self.ID, UpdateUserInput{Enabled: &disabled})
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("禁用自己应 409，实际 err=%v", err)
	}
}

// TestUpdateUser_CannotDemoteSelf 验证 admin 不能把自己降级为 user（对应需求 #18）。
func TestUpdateUser_CannotDemoteSelf(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, CreateUserInput{Username: "other-admin2", Password: "Passw0rd!1", Role: model.RoleAdmin}); err != nil {
		t.Fatalf("建第二个 admin 失败: %v", err)
	}
	self, err := svc.CreateUser(ctx, CreateUserInput{Username: "self2", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("建自身账号失败: %v", err)
	}
	newRole := model.RoleUser
	_, err = svc.UpdateUser(ctx, self.ID, self.ID, UpdateUserInput{Role: &newRole})
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("降级自己应 409，实际 err=%v", err)
	}
}

// TestUpdateUser_FinalEnabledAdminCannotBeDisabled 验证唯一启用中的 admin 不能被禁用
// （对应需求 #20）。用一个不等于目标的 actorID 模拟「由另一个身份发起」，隔离自禁用规则，
// 单独验证 repo.UpdateUserRoleEnabled 的「最后一个启用 admin」保护本身。
func TestUpdateUser_FinalEnabledAdminCannotBeDisabled(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	onlyAdmin, err := svc.CreateUser(ctx, CreateUserInput{Username: "lonely-admin", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("建唯一 admin 失败: %v", err)
	}
	const otherActorID = 999999 // 不存在的 id，代表「另一个发起者」，只为绕开 actorID==targetID 的自身规则
	disabled := false
	_, err = svc.UpdateUser(ctx, otherActorID, onlyAdmin.ID, UpdateUserInput{Enabled: &disabled})
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("禁用最后一个启用中的 admin 应 409，实际 err=%v", err)
	}
}

// TestUpdateUser_FinalEnabledAdminCannotBeDemoted 验证唯一启用中的 admin 不能被降级
// （对应需求 #21，隔离方式同上）。
func TestUpdateUser_FinalEnabledAdminCannotBeDemoted(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	onlyAdmin, err := svc.CreateUser(ctx, CreateUserInput{Username: "lonely-admin2", Password: "Passw0rd!1", Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("建唯一 admin 失败: %v", err)
	}
	const otherActorID = 999999
	newRole := model.RoleUser
	_, err = svc.UpdateUser(ctx, otherActorID, onlyAdmin.ID, UpdateUserInput{Role: &newRole})
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("降级最后一个启用中的 admin 应 409，实际 err=%v", err)
	}
}

// ---------- 重置密码 ----------

// TestResetPassword_OldFailsNewSucceeds 验证重置密码后旧密码失效、新密码生效
// （对应需求 #23 #24 #25），全程走 svc.Login 而非直接比对哈希，验证端到端行为。
func TestResetPassword_OldFailsNewSucceeds(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	const oldPw = "OldPassw0rd!"
	const newPw = "NewPassw0rd!"
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "ivan", Password: oldPw, Role: model.RoleUser})
	if err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	if _, err := svc.Login(ctx, "ivan", oldPw); err != nil {
		t.Fatalf("重置前旧密码应能登录: %v", err)
	}

	if err := svc.ResetUserPassword(ctx, u.ID, ResetPasswordInput{Password: newPw}); err != nil {
		t.Fatalf("重置密码应成功: %v", err)
	}

	if _, err := svc.Login(ctx, "ivan", oldPw); err == nil {
		t.Fatal("重置后旧密码应失效")
	}
	if _, err := svc.Login(ctx, "ivan", newPw); err != nil {
		t.Fatalf("重置后新密码应生效: %v", err)
	}
}

// TestLogin_DisabledUserRejected 验证禁用账号即使密码正确也无法登录（对应需求 #26）。
func TestLogin_DisabledUserRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	disabled := false
	if _, err := svc.CreateUser(ctx, CreateUserInput{
		Username: "julia", Password: "Passw0rd!1", Role: model.RoleUser, Enabled: &disabled,
	}); err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	if _, err := svc.Login(ctx, "julia", "Passw0rd!1"); err == nil {
		t.Fatal("禁用账号应拒绝登录，即使密码正确")
	}
}
