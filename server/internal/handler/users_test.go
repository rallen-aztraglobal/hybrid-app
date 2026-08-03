package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/service"
)

// TestUsersAPI_AdminCanListUsers 验证 admin 可以列出账号（对应需求 #1）。
func TestUsersAPI_AdminCanListUsers(t *testing.T) {
	e, mgr, _, r := testServer(t)
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodGet, "/api/users", adminTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin GET /api/users 应 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"username"`) {
		t.Errorf("列表应包含 username 字段，实际 body=%s", rec.Body.String())
	}
}

// TestUsersAPI_UserCannotListUsers 验证 user 角色访问账号列表应 403（对应需求 #2 #22）。
func TestUsersAPI_UserCannotListUsers(t *testing.T) {
	e, mgr, _, r := testServer(t)
	userTok := issueToken(t, mgr, r, model.RoleUser)
	rec := doReq(e, http.MethodGet, "/api/users", userTok, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("user 访问 /api/users 应 403，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_UnauthenticatedRejected 验证未带 token 访问应 401（对应需求 #3）。
func TestUsersAPI_UnauthenticatedRejected(t *testing.T) {
	e, _, _, _ := testServer(t)
	rec := doReq(e, http.MethodGet, "/api/users", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("未鉴权访问 /api/users 应 401，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_ListDoesNotExposePasswordHash 验证列表/创建响应体都不含密码哈希（对应需求 #4）。
func TestUsersAPI_ListDoesNotExposePasswordHash(t *testing.T) {
	e, mgr, _, r := testServer(t)
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)

	rec := doReq(e, http.MethodPost, "/api/users", adminTok, `{"username":"newbie","password":"Passw0rd!1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建账号应 201，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "passwordHash") || strings.Contains(body, "Passw0rd!1") || strings.Contains(body, "$2") {
		t.Errorf("响应不应包含密码哈希或明文，实际 body=%s", body)
	}

	rec = doReq(e, http.MethodGet, "/api/users", adminTok, "")
	body = rec.Body.String()
	if strings.Contains(body, "passwordHash") || strings.Contains(body, "$2") {
		t.Errorf("列表响应不应包含密码哈希，实际 body=%s", body)
	}
}

// createRealAdmin 直接用 repo 建一个真实的 role=admin 账号（模拟系统里唯一的永久 admin），
// 并用 mgr 签发其 token。与 issueToken 的区别：issueToken 建的账号角色任意、用户名随机，
// 这里需要一个「货真价实的 admin」来测试 admin 专属保护规则（不可删/不可重置密码）。
func createRealAdmin(t *testing.T, mgr *auth.Manager, r *repo.Repo, username string) (*model.AdminUser, string) {
	t.Helper()
	hash, err := auth.HashPassword("AdminPass!1")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	admin := &model.AdminUser{Username: username, PasswordHash: hash, Role: model.RoleAdmin}
	if err := r.CreateUser(context.Background(), admin); err != nil {
		t.Fatalf("建 admin 失败: %v", err)
	}
	access, _, err := mgr.Issue(admin)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	return admin, access
}

// TestUsersAPI_AdminRowIsProtected 验证列表中永久 admin 带 protected:true 标记（对应需求 #5）。
func TestUsersAPI_AdminRowIsProtected(t *testing.T) {
	e, mgr, _, r := testServer(t)
	_, access := createRealAdmin(t, mgr, r, "root-admin")
	rec := doReq(e, http.MethodGet, "/api/users", access, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/users 应 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"protected":true`) {
		t.Errorf("admin 行应带 protected:true，实际 body=%s", rec.Body.String())
	}
}

// TestUsersAPI_CreateRequestCannotCreateAdmin 验证请求体里多塞一个 role:"admin" 字段
// 也无法创建出 admin 账号（对应需求 #8）：CreateUserInput 没有 role 字段，解码器会
// 直接忽略这个多余字段，落库角色恒为 user。
func TestUsersAPI_CreateRequestCannotCreateAdmin(t *testing.T) {
	e, mgr, _, r := testServer(t)
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodPost, "/api/users", adminTok,
		`{"username":"sneaky","password":"Passw0rd!1","role":"admin"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建应 201，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"role":"user"`) {
		t.Errorf("即使请求体携带 role:admin，落库角色也应恒为 user，实际 body=%s", rec.Body.String())
	}
}

// TestUsersAPI_AdminCanResetUserPassword 验证 admin 可重置普通用户密码（对应需求 #14）。
func TestUsersAPI_AdminCanResetUserPassword(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()
	target, err := svc.CreateUser(ctx, service.CreateUserInput{Username: "kim", Password: "OldPassw0rd!"})
	if err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", target.ID), adminTok,
		`{"password":"NewPassw0rd!"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("重置密码应 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_CannotResetAdminPassword 验证不能通过 API 重置永久 admin 密码（对应需求 #17）。
func TestUsersAPI_CannotResetAdminPassword(t *testing.T) {
	e, mgr, _, r := testServer(t)
	admin, _ := createRealAdmin(t, mgr, r, "root-admin2")
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", admin.ID), adminTok,
		`{"password":"Whatever!1"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("重置 admin 密码应 409，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_AdminCanDeleteUser 验证需求 #18 #19 #20：删除后旧 token 立即失效
// （复用同一个未过期 token 二次请求应变 401），且删除后无法登录。
func TestUsersAPI_AdminCanDeleteUser(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()

	target, err := svc.CreateUser(ctx, service.CreateUserInput{Username: "deleteme", Password: "Passw0rd!1"})
	if err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	targetTok, _, err := mgr.Issue(&model.AdminUser{ID: target.ID, Username: target.Username, Role: target.Role})
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	// 删除前：该 token 能正常访问受保护接口。
	rec := doReq(e, http.MethodGet, "/api/channels?pageSize=10", targetTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除前应能正常访问，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec = doReq(e, http.MethodDelete, fmt.Sprintf("/api/users/%d", target.ID), adminTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin 删除用户应 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// 复用同一个（未过期的）旧 token 再次请求：应被拒绝（对应需求 #20）。
	rec = doReq(e, http.MethodGet, "/api/channels?pageSize=10", targetTok, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("删除后复用旧 token 应 401，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// 删除后不能再登录（对应需求 #19）。
	rec = doReq(e, http.MethodPost, "/api/auth/login", "", `{"username":"deleteme","password":"Passw0rd!1"}`)
	if rec.Code == http.StatusOK {
		t.Errorf("已删除账号不应能登录，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_CannotDeleteAdmin 验证永久 admin 不可被删除（对应需求 #21）。
func TestUsersAPI_CannotDeleteAdmin(t *testing.T) {
	e, mgr, _, r := testServer(t)
	admin, _ := createRealAdmin(t, mgr, r, "root-admin3")
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodDelete, fmt.Sprintf("/api/users/%d", admin.ID), adminTok, "")
	if rec.Code != http.StatusConflict {
		t.Errorf("删除 admin 应 409，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_UserCannotCallManagementEndpoints 验证 user 角色调用 create/delete/reset 均 403
// （对应需求 #22）。
func TestUsersAPI_UserCannotCallManagementEndpoints(t *testing.T) {
	e, mgr, _, r := testServer(t)
	userTok := issueToken(t, mgr, r, model.RoleUser)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/users", `{"username":"x","password":"Passw0rd!1"}`},
		{http.MethodDelete, "/api/users/1", ""},
		{http.MethodPost, "/api/users/1/reset-password", `{"password":"Passw0rd!1"}`},
	}
	for _, c := range cases {
		rec := doReq(e, c.method, c.path, userTok, c.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("user %s %s 应 403，实际 %d body=%s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}
