package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/model"
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

// TestUsersAPI_UserCannotListUsers 验证 user 角色访问账号列表应 403（对应需求 #2）。
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

// TestUsersAPI_PasswordHashNeverReturned 验证列表/创建响应体都不含密码哈希（对应需求 #13）。
func TestUsersAPI_PasswordHashNeverReturned(t *testing.T) {
	e, mgr, _, r := testServer(t)
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)

	rec := doReq(e, http.MethodPost, "/api/users", adminTok,
		`{"username":"newbie","password":"Passw0rd!1","role":"user"}`)
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

// TestUsersAPI_NoHardDeleteEndpoint 验证账号管理不提供硬删除端点：DELETE /api/users/:id
// 恒 404（路由未注册）。这是刻意的设计——audit_log.user_id / channel.created_by /
// listing_app.created_by 均引用 admin_user.id 且无级联，硬删会破坏审计与归属追溯，
// 见 service/user.go、docs 用户管理文档。该设计使「不能删除自己」（需求 #19）与
// 「最后一个启用 admin 不能被删除」（需求 #22）天然成立：删除能力本身不存在。
func TestUsersAPI_NoHardDeleteEndpoint(t *testing.T) {
	e, mgr, _, r := testServer(t)
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodDelete, "/api/users/1", adminTok, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("DELETE /api/users/:id 应 404（未注册该路由），实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_DisabledUserCannotLogin 验证禁用账号无法登录（对应需求 #26）。
func TestUsersAPI_DisabledUserCannotLogin(t *testing.T) {
	e, _, svc, _ := testServer(t)
	ctx := context.Background()
	disabled := false
	if _, err := svc.CreateUser(ctx, service.CreateUserInput{
		Username: "blocked", Password: "Passw0rd!1", Role: model.RoleUser, Enabled: &disabled,
	}); err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	rec := doReq(e, http.MethodPost, "/api/auth/login", "", `{"username":"blocked","password":"Passw0rd!1"}`)
	if rec.Code == http.StatusOK {
		t.Errorf("禁用账号登录不应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestUsersAPI_DisabledUserSessionRevoked 验证「禁用后已签发的 token 立即失效」：
// 先签发 token 并确认可访问受保护接口，再由 admin 禁用该账号，复用同一个（未过期的）token
// 请求同一受保护接口，应变为 401——这是 handler.RequireEnabled 中间件要解决的核心场景
// （对应需求 #27：会话撤销，而不是等 JWT 自然过期）。
func TestUsersAPI_DisabledUserSessionRevoked(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()

	target, err := svc.CreateUser(ctx, service.CreateUserInput{Username: "revokeme", Password: "Passw0rd!1", Role: model.RoleUser})
	if err != nil {
		t.Fatalf("建账号失败: %v", err)
	}
	targetTok, _, err := mgr.Issue(target)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	// 禁用前：该 token 能正常访问受保护接口。
	rec := doReq(e, http.MethodGet, "/api/channels?pageSize=10", targetTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("禁用前应能正常访问，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// admin 禁用该账号。
	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	disabled := false
	rec = doReq(e, http.MethodPut, fmt.Sprintf("/api/users/%d", target.ID), adminTok,
		`{"enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin 禁用目标账号应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	_ = disabled

	// 复用同一个（未过期的）旧 token 再次请求：应被拒绝（对应需求 #27）。
	rec = doReq(e, http.MethodGet, "/api/channels?pageSize=10", targetTok, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("禁用后复用旧 token 应 401，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}
