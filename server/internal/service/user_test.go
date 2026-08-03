package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// ---------- 新建用户 ----------

// TestCreateUser_AlwaysRoleUser 验证新建账号恒为 role=user（对应需求 #7 #8）：
// CreateUserInput 本身没有 role 字段，无法从调用方指定，这条测试确认落库结果确实是 user。
func TestCreateUser_AlwaysRoleUser(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "alice", Password: "Passw0rd!1"})
	if err != nil {
		t.Fatalf("创建应成功: %v", err)
	}
	if u.Role != model.RoleUser {
		t.Errorf("新账号应恒为 user 角色，实际 %s", u.Role)
	}
	if u.Protected {
		t.Error("普通用户不应标记为 protected")
	}
}

// TestCreateUser_DuplicateUsernameCaseInsensitive 验证用户名大小写不敏感唯一（对应需求 #9）。
func TestCreateUser_DuplicateUsernameCaseInsensitive(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, CreateUserInput{Username: "Carol", Password: "Passw0rd!1"}); err != nil {
		t.Fatalf("首次创建应成功: %v", err)
	}
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "carol", Password: "Passw0rd!1"})
	if err == nil {
		t.Fatal("大小写不同但同名应被拒绝")
	}
	if AsError(err).Code != 409 {
		t.Errorf("重名应返回 409，实际 %d", AsError(err).Code)
	}
}

// TestCreateUser_EmptyUsernameRejected 验证空用户名（含纯空白）被拒绝（对应需求 #10）。
func TestCreateUser_EmptyUsernameRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, err := svc.CreateUser(ctx, CreateUserInput{Username: "   ", Password: "Passw0rd!1"})
	if err == nil {
		t.Fatal("空用户名应被拒绝")
	}
	if AsError(err).Code != 400 {
		t.Errorf("应返回 400，实际 %d", AsError(err).Code)
	}
}

// TestCreateUser_PasswordHashedNotPlaintext 验证密码用 bcrypt 哈希、不落明文（对应需求 #11 #12）。
func TestCreateUser_PasswordHashedNotPlaintext(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	const plain = "S3cretPass!"
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "grace", Password: plain})
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

// TestCreateUser_CanLogin 验证新建用户可以正常登录（对应需求 #13）。
func TestCreateUser_CanLogin(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, CreateUserInput{Username: "henry", Password: "Passw0rd!1"}); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if _, err := svc.Login(ctx, "henry", "Passw0rd!1"); err != nil {
		t.Fatalf("新用户应能登录: %v", err)
	}
}

// ---------- 重置密码 ----------

// TestResetPassword_OldFailsNewSucceeds 验证重置密码后旧密码失效、新密码生效
// （对应需求 #14 #15 #16），全程走 svc.Login 而非直接比对哈希，验证端到端行为。
func TestResetPassword_OldFailsNewSucceeds(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	const oldPw = "OldPassw0rd!"
	const newPw = "NewPassw0rd!"
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "ivan", Password: oldPw})
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

// TestResetPassword_CannotResetAdmin 验证不能通过账号管理重置永久 admin 的密码（对应需求 #17）。
func TestResetPassword_CannotResetAdmin(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	admin := bootstrapAdmin(t, r)
	err := svc.ResetUserPassword(ctx, admin.ID, ResetPasswordInput{Password: "Whatever!1"})
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("重置 admin 密码应 409，实际 err=%v", err)
	}
}

// ---------- 删除用户 ----------

// TestDeleteUser_SoftDeleteRejectsLoginAndSelf 验证删除用户后无法登录（对应需求 #18 #19）。
func TestDeleteUser_SoftDeleteRejectsLoginAndSelf(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	u, err := svc.CreateUser(ctx, CreateUserInput{Username: "judy", Password: "Passw0rd!1"})
	if err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	if err := svc.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("删除应成功: %v", err)
	}
	if _, err := svc.Login(ctx, "judy", "Passw0rd!1"); err == nil {
		t.Fatal("已删除账号不应能登录")
	}
}

// TestDeleteUser_CannotDeleteAdmin 验证永久 admin 不可被删除（对应需求 #21）。
func TestDeleteUser_CannotDeleteAdmin(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	admin := bootstrapAdmin(t, r)
	err := svc.DeleteUser(ctx, admin.ID)
	if err == nil || AsError(err).Code != 409 {
		t.Fatalf("删除 admin 应 409，实际 err=%v", err)
	}
}

// bootstrapAdmin 直接用 repo 建一个 role=admin 账号，模拟系统里唯一的永久 admin
// （生产环境由 seed.EnsureBootstrapAdmin 创建；测试直接建，避免依赖 seed 包）。
func bootstrapAdmin(t *testing.T, r *repo.Repo) *model.AdminUser {
	t.Helper()
	hash, err := auth.HashPassword("AdminPass!1")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	u := &model.AdminUser{Username: "root-admin", PasswordHash: hash, Role: model.RoleAdmin}
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("建 admin 失败: %v", err)
	}
	return u
}
