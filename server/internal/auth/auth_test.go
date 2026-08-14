package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/model"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	if !CheckPassword(hash, "s3cret-pw") {
		t.Error("正确密码应校验通过")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("错误密码不应通过")
	}
}

func TestIssueAndParse(t *testing.T) {
	m := NewManager("test-secret", "test", time.Hour, 24*time.Hour)
	u := &model.AdminUser{ID: 7, Username: "alice", Role: model.RoleUser}

	access, refresh, err := m.Issue(u)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	ac, err := m.Parse(access)
	if err != nil {
		t.Fatalf("解析 access 失败: %v", err)
	}
	if ac.UserID != 7 || ac.Username != "alice" || ac.Role != model.RoleUser {
		t.Errorf("access claims 不符: %+v", ac)
	}
	if ac.Type != TokenAccess {
		t.Errorf("access token type 应为 %q，实际 %q", TokenAccess, ac.Type)
	}

	rc, err := m.Parse(refresh)
	if err != nil {
		t.Fatalf("解析 refresh 失败: %v", err)
	}
	if rc.Type != TokenRefresh {
		t.Errorf("refresh token type 应为 %q，实际 %q", TokenRefresh, rc.Type)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	m1 := NewManager("secret-a", "test", time.Hour, time.Hour)
	m2 := NewManager("secret-b", "test", time.Hour, time.Hour)
	access, _, _ := m1.Issue(&model.AdminUser{ID: 1, Username: "x", Role: model.RoleUser})
	if _, err := m2.Parse(access); err == nil {
		t.Error("不同 secret 应解析失败")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	m := NewManager("secret", "test", -time.Minute, time.Hour) // 立即过期
	access, _, _ := m.Issue(&model.AdminUser{ID: 1, Username: "x", Role: model.RoleUser})
	if _, err := m.Parse(access); err == nil {
		t.Error("过期 token 应解析失败")
	}
}

// TestRoleRanking 验证两档角色的排名关系：admin 高于 user。
func TestRoleRanking(t *testing.T) {
	if roleRank[model.RoleAdmin] <= roleRank[model.RoleUser] {
		t.Error("admin 应高于 user")
	}
}

// TestRequireRoleAdminGate 验证 RequireRole(RoleAdmin) 的实际放行/拒绝行为：
// admin 满足 admin 门槛；user 不满足 admin 门槛；两者都满足 user 门槛。
func TestRequireRoleAdminGate(t *testing.T) {
	e := echo.New()
	admin := RequireRole(model.RoleAdmin)
	user := RequireRole(model.RoleUser)
	ok := func(c echo.Context) error { return c.NoContent(http.StatusOK) }

	withClaims := func(role string) echo.Context {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ctxClaims, &Claims{Username: "t", Role: role, Type: TokenAccess})
		return c
	}

	// admin 满足 admin 门槛。
	c := withClaims(model.RoleAdmin)
	if err := admin(ok)(c); err != nil {
		t.Errorf("admin 应通过 admin 门槛: %v", err)
	}

	// user 不满足 admin 门槛 → 403。
	c = withClaims(model.RoleUser)
	err := admin(ok)(c)
	httpErr, isHTTPErr := err.(*echo.HTTPError)
	if !isHTTPErr || httpErr.Code != http.StatusForbidden {
		t.Errorf("user 访问 admin-only 路由应返回 403，实际: %v", err)
	}

	// admin 与 user 都应满足 user 门槛。
	for _, role := range []string{model.RoleAdmin, model.RoleUser} {
		c = withClaims(role)
		if err := user(ok)(c); err != nil {
			t.Errorf("角色 %q 应通过 user 门槛: %v", role, err)
		}
	}
}

// TestRunnerTokenMapsToUserNotAdmin 验证构建机静态令牌注入的是 user 身份，
// 不能满足 admin 门槛（不能碰 Store 管理 / 渠道归档删除）。
func TestRunnerTokenMapsToUserNotAdmin(t *testing.T) {
	m := NewManager("secret", "test", time.Hour, time.Hour)
	m.RunnerToken = "runner-static-token"

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/build/claim", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+m.RunnerToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var gotClaims *Claims
	next := func(cc echo.Context) error {
		gotClaims = FromContext(cc)
		return cc.NoContent(http.StatusOK)
	}
	if err := m.Middleware()(next)(c); err != nil {
		t.Fatalf("runner token 应放行: %v", err)
	}
	if gotClaims == nil || gotClaims.Role != model.RoleUser {
		t.Fatalf("runner 身份应为 RoleUser，实际: %+v", gotClaims)
	}

	// 用同样的 claims 过 admin 门槛应被拒绝。
	admin := RequireRole(model.RoleAdmin)
	req2 := httptest.NewRequest(http.MethodDelete, "/api/stores/1", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.Set(ctxClaims, gotClaims)
	err := admin(func(cc echo.Context) error { return cc.NoContent(http.StatusOK) })(c2)
	httpErr, isHTTPErr := err.(*echo.HTTPError)
	if !isHTTPErr || httpErr.Code != http.StatusForbidden {
		t.Errorf("runner 身份不应满足 admin 门槛，实际: %v", err)
	}
}
