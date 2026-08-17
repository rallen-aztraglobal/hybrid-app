package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/seed"
	"github.com/hybrid-app/server/internal/service"
	"github.com/hybrid-app/server/internal/storage"
)

const testRunnerToken = "test-runner-token"

// sanitizeDBName 把测试名转成合法的内存库名（去掉 / 等）。
func sanitizeDBName(s string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(s)
}

// testApp 聚合起一个可直接 ServeHTTP 的完整应用（品牌 + RBAC 已 seed），供 RequirePerm 中间件的
// 放行/拒绝行为做端到端验证。每个测试用独立命名的内存 sqlite 库，互不串扰（同 service_test.go 的做法）。
type testApp struct {
	e       *echo.Echo
	repo    *repo.Repo
	authMgr *auth.Manager
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeDBName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	r := repo.New(db)
	ctx := t.Context()
	if err := seed.EnsureBrands(ctx, db); err != nil {
		t.Fatalf("seed 品牌失败: %v", err)
	}
	if err := seed.EnsureRBAC(ctx, db); err != nil {
		t.Fatalf("seed RBAC 失败: %v", err)
	}

	st, err := storage.NewLocal(t.TempDir(), "http://test/static")
	if err != nil {
		t.Fatalf("创建本地存储失败: %v", err)
	}
	cfg := config.Load()
	cfg.AppConfigTTLSecs = 600
	cfg.DefaultProbePath = "/healthz"
	cfg.RunnerToken = testRunnerToken

	svc := service.New(cfg, r, st)
	authMgr := auth.NewManager("test-secret", "test", time.Hour, 24*time.Hour)
	authMgr.RunnerToken = cfg.RunnerToken

	h := New(cfg, svc, authMgr, r)
	e := echo.New()
	h.Register(e)

	return &testApp{e: e, repo: r, authMgr: authMgr}
}

// createUser 直接落库建一个挂指定角色名的账号，返回其 access token。
func (a *testApp) createUser(t *testing.T, username, roleName string) (uint64, string) {
	t.Helper()
	ctx := t.Context()
	role, err := a.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		t.Fatalf("角色 %q 不存在: %v", roleName, err)
	}
	hash, err := auth.HashPassword("pw123456")
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	u := &model.AdminUser{Username: username, PasswordHash: hash, Role: model.RoleOperator, RoleID: role.ID}
	if err := a.repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	access, _, err := a.authMgr.Issue(u)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	return u.ID, access
}

// do 发起一个请求，token 非空时带 Bearer。
func (a *testApp) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set(echo.HeaderContentType, "application/json")
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.e.ServeHTTP(rec, req)
	return rec
}

// TestRequirePermAllowsRoleDefaultPerms 验证只读/运营角色的默认权限边界符合契约。
func TestRequirePermAllowsRoleDefaultPerms(t *testing.T) {
	app := newTestApp(t)

	_, viewerTok := app.createUser(t, "viewer1", "只读")
	_, opTok := app.createUser(t, "op1", "运营")

	// 只读：GET /channels（page:channels）放行。
	if rec := app.do(http.MethodGet, "/api/channels", viewerTok, nil); rec.Code != http.StatusOK {
		t.Errorf("只读 GET /channels 应放行，实际 %d: %s", rec.Code, rec.Body.String())
	}
	// 只读：POST /channels（channel:create，写操作）应拒绝。
	if rec := app.do(http.MethodPost, "/api/channels", viewerTok, map[string]any{}); rec.Code != http.StatusForbidden {
		t.Errorf("只读 POST /channels 应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
	// 只读：POST /stores（store:manage）应拒绝。
	if rec := app.do(http.MethodPost, "/api/stores", viewerTok, map[string]any{"code": "xx", "name": "X"}); rec.Code != http.StatusForbidden {
		t.Errorf("只读 POST /stores 应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}

	// 运营：POST /channels（channel:create）放行（即便业务校验后续可能因缺字段失败，也不应是 403）。
	if rec := app.do(http.MethodPost, "/api/channels", opTok, map[string]any{}); rec.Code == http.StatusForbidden {
		t.Errorf("运营 POST /channels 不应被权限拦截，实际 403: %s", rec.Body.String())
	}
	// 运营：POST /stores（store:manage，系统设置类写操作）应拒绝——运营默认不含 store/user/role manage。
	if rec := app.do(http.MethodPost, "/api/stores", opTok, map[string]any{"code": "yy", "name": "Y"}); rec.Code != http.StatusForbidden {
		t.Errorf("运营 POST /stores 应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
	// 运营：GET /roles（role:manage）应拒绝。
	if rec := app.do(http.MethodGet, "/api/roles", opTok, nil); rec.Code != http.StatusForbidden {
		t.Errorf("运营 GET /roles 应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRequirePermSuperAdminHasFullAccess 验证超级管理员对任意 catalog 权限点放行（不含机器接口）。
func TestRequirePermSuperAdminHasFullAccess(t *testing.T) {
	app := newTestApp(t)
	_, tok := app.createUser(t, "root1", "超级管理员")

	if rec := app.do(http.MethodGet, "/api/roles", tok, nil); rec.Code != http.StatusOK {
		t.Errorf("超管 GET /roles 应放行，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodGet, "/api/users", tok, nil); rec.Code != http.StatusOK {
		t.Errorf("超管 GET /users 应放行，实际 %d: %s", rec.Code, rec.Body.String())
	}
	// 超管也不隐式持有机器权限 build:runner（不在 catalog 内，任何人类账号都不可能持有）。
	if rec := app.do(http.MethodPost, "/api/build/claim", tok, nil); rec.Code != http.StatusForbidden {
		t.Errorf("超管不应能碰机器接口 /build/claim，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRequirePermDeletedUserIs401 验证账号被删除后已签发的 token 立即失效为 401。
// 通过真实的 DELETE /api/users/:id 接口删除（而非直接操作 repo），因为权限缓存需要
// handler 侧在角色/用户写操作后显式 Invalidate（30s TTL 内，直接操作 repo 不会触发失效，
// 那是缓存的预期行为，不是本用例要测的东西）。
func TestRequirePermDeletedUserIs401(t *testing.T) {
	app := newTestApp(t)
	_, adminTok := app.createUser(t, "root2", "超级管理员")
	uid, tok := app.createUser(t, "ghost", "只读")

	if rec := app.do(http.MethodGet, "/api/channels", tok, nil); rec.Code != http.StatusOK {
		t.Fatalf("删除前应可访问，实际 %d", rec.Code)
	}
	if rec := app.do(http.MethodDelete, fmt.Sprintf("/api/users/%d", uid), adminTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("超管删除账号应成功，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodGet, "/api/channels", tok, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("账号被删除后应 401，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRunnerOnlyMachineEndpoints 验证构建机静态 token 能碰的接口边界：4 条机器接口 + 服务端打包
// 流水线必经的只读接口（manifest/records/res.zip，阻断1：claim 后 render.Pull 要拉这些，不放行
// 整条服务端打包流水线会 403 断掉），其余一律 403；且未携带任何 Bearer 时统一 401。
func TestRunnerOnlyMachineEndpoints(t *testing.T) {
	app := newTestApp(t)

	// 4 条机器接口 + manifest/records/res.zip 读接口：runner token 应放行
	// （即便因空 body/参数/记录不存在返回业务错误，也不应是 401/403）。
	allowedReqs := []struct {
		method, path string
	}{
		{http.MethodPost, "/api/build/claim"},
		{http.MethodPost, "/api/build/records/1/status"},
		{http.MethodPost, "/api/build/records/1/logs"},
		{http.MethodPost, "/api/build/records/1/artifacts"},
		{http.MethodGet, "/api/build/manifest?brand=ap"},
		{http.MethodGet, "/api/build/records"},
		{http.MethodGet, "/api/build/records/1"},
		{http.MethodGet, "/api/build/records/1/logs"},
		{http.MethodGet, "/api/channels/1/res.zip"},
	}
	for _, r := range allowedReqs {
		rec := app.do(r.method, r.path, testRunnerToken, map[string]any{})
		if rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
			t.Errorf("runner %s %s 应放行(非 401/403)，实际 %d: %s", r.method, r.path, rec.Code, rec.Body.String())
		}
	}

	// 非机器接口：runner token 一律 403（不再是可碰任意写接口的泛化 operator 角色；
	// 渠道列表/详情/写接口都不该放行，只有 res.zip 是例外）。
	otherReqs := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/channels"},
		{http.MethodPost, "/api/channels"},
		{http.MethodGet, "/api/channels/1"},
		{http.MethodPost, "/api/stores"},
	}
	for _, r := range otherReqs {
		rec := app.do(r.method, r.path, testRunnerToken, map[string]any{})
		if rec.Code != http.StatusForbidden {
			t.Errorf("runner %s %s 应 403，实际 %d: %s", r.method, r.path, rec.Code, rec.Body.String())
		}
	}

	// 无 Bearer：统一 401。
	if rec := app.do(http.MethodGet, "/api/channels", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("无 token 应 401，实际 %d", rec.Code)
	}
}

// TestPublicEndpointsNeedOnlyLogin 验证「登录即可」类端点：任意已登录角色都能访问，无需具体权限点。
func TestPublicEndpointsNeedOnlyLogin(t *testing.T) {
	app := newTestApp(t)
	_, tok := app.createUser(t, "viewer2", "只读")

	for _, path := range []string{"/api/brands", "/api/stores", "/api/perms/catalog", "/api/auth/me"} {
		if rec := app.do(http.MethodGet, path, tok, nil); rec.Code != http.StatusOK {
			t.Errorf("GET %s 登录即可应放行，实际 %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestActiveAccountEndpointsRejectDeletedUser 是应修4的回归测试：/auth/me、/perms/catalog、
// /brands、/stores 这四条「登录即可」端点不进 RequirePerm，必须靠 RequireActiveAccount 单独校验
// 账号仍存在——否则被删账号的旧 token 在其自然过期前会一直能读这几条。
func TestActiveAccountEndpointsRejectDeletedUser(t *testing.T) {
	app := newTestApp(t)
	_, adminTok := app.createUser(t, "root4", "超级管理员")
	uid, tok := app.createUser(t, "soon-deleted", "只读")

	for _, path := range []string{"/api/auth/me", "/api/perms/catalog", "/api/brands", "/api/stores"} {
		if rec := app.do(http.MethodGet, path, tok, nil); rec.Code != http.StatusOK {
			t.Fatalf("删除前 GET %s 应放行，实际 %d: %s", path, rec.Code, rec.Body.String())
		}
	}

	if rec := app.do(http.MethodDelete, fmt.Sprintf("/api/users/%d", uid), adminTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("超管删除账号应成功，实际 %d: %s", rec.Code, rec.Body.String())
	}

	for _, path := range []string{"/api/auth/me", "/api/perms/catalog", "/api/brands", "/api/stores"} {
		if rec := app.do(http.MethodGet, path, tok, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("账号被删除后 GET %s 应立刻 401，实际 %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestActiveAccountEndpointsRejectRunner 是应修4的回归测试：runner 静态 token 对这四条「登录即可」
// 端点没有正当用途（/perms/catalog、/auth/me 对机器身份无意义；/brands、/stores 是人工维护渠道用的
// 基础数据，构建机拉全量配置走 GET /build/manifest），一律 403。
func TestActiveAccountEndpointsRejectRunner(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{"/api/auth/me", "/api/perms/catalog", "/api/brands", "/api/stores"} {
		if rec := app.do(http.MethodGet, path, testRunnerToken, nil); rec.Code != http.StatusForbidden {
			t.Errorf("runner GET %s 应 403，实际 %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestRunnerUsernameCannotImpersonateMachine 是 B1 的回归测试：一个用户名恰好叫 "runner" 的
// 普通账号（模拟脏数据/绕过 service.CreateUser 保留名校验落库的历史账号），其正常签发的
// access token 类型是 auth.TokenAccess 而非 auth.TokenRunner，必须像任何其他普通账号一样
// 被权限系统对待——不能碰 4 条机器接口，也只能按其挂的角色访问普通接口。
func TestRunnerUsernameCannotImpersonateMachine(t *testing.T) {
	app := newTestApp(t)
	// 直接落库建一个用户名为 runner、挂「只读」角色的普通账号（绕开 service.CreateUser 的保留名拦截，
	// 模拟脏数据场景，验证鉴权层本身是否真的不信任 Username）。
	_, tok := app.createUser(t, "runner", "只读")

	// 不能碰机器接口：必须 403（若靠 Username 判定机器身份，这里会被误放行，B1 的实测复现点）。
	for _, r := range []struct{ method, path string }{
		{http.MethodPost, "/api/build/claim"},
		{http.MethodPost, "/api/build/records/1/status"},
		{http.MethodPost, "/api/build/records/1/logs"},
		{http.MethodPost, "/api/build/records/1/artifacts"},
	} {
		if rec := app.do(r.method, r.path, tok, map[string]any{}); rec.Code != http.StatusForbidden {
			t.Errorf("用户名=runner 的普通账号访问机器接口 %s %s 应 403，实际 %d: %s",
				r.method, r.path, rec.Code, rec.Body.String())
		}
	}
	// 仍按其挂的「只读」角色正常工作：能读、不能写。
	if rec := app.do(http.MethodGet, "/api/channels", tok, nil); rec.Code != http.StatusOK {
		t.Errorf("用户名=runner 的只读账号 GET /channels 应放行，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodPost, "/api/channels", tok, map[string]any{}); rec.Code != http.StatusForbidden {
		t.Errorf("用户名=runner 的只读账号 POST /channels 应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestCreateUserRejectsReservedUsername 验证走真实 API 新建账号时，用户名 "runner"（大小写不敏感）
// 被服务层拦截为保留名（B1，400），而非等到鉴权层才发现问题。
func TestCreateUserRejectsReservedUsername(t *testing.T) {
	app := newTestApp(t)
	_, adminTok := app.createUser(t, "root3", "超级管理员")

	for _, name := range []string{"runner", "Runner", "RUNNER"} {
		body := map[string]any{"username": name, "password": "pw123456", "roleId": 0}
		rec := app.do(http.MethodPost, "/api/users", adminTok, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("创建用户名=%q 应 400（保留名），实际 %d: %s", name, rec.Code, rec.Body.String())
		}
	}
}

// TestRequirePermRoleMissingIs403WithClearMessage 是 M2 的回归测试：账号存在但其 role_id 指向的
// 角色已不存在（脏数据/角色被误删），RequirePerm 应返回 403 + 明确文案，而不是与「账号不存在」
// 混为一谈的 401（那会误导运维去查一个其实存在的账号）。
func TestRequirePermRoleMissingIs403WithClearMessage(t *testing.T) {
	app := newTestApp(t)
	ctx := t.Context()

	dangling := &model.Role{Name: "临时角色", Builtin: false}
	if err := app.repo.CreateRole(ctx, dangling); err != nil {
		t.Fatalf("创建临时角色失败: %v", err)
	}
	hash, err := auth.HashPassword("pw123456")
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	u := &model.AdminUser{Username: "dangler", PasswordHash: hash, Role: model.RoleOperator, RoleID: dangling.ID}
	if err := app.repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	tok, _, err := app.authMgr.Issue(u)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	// 角色被直接删除（模拟误删/脏数据），账号本身仍然存在。
	if err := app.repo.DeleteRole(ctx, dangling.ID); err != nil {
		t.Fatalf("删除角色失败: %v", err)
	}

	rec := app.do(http.MethodGet, "/api/channels", tok, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("角色缺失应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "角色") {
		t.Errorf("403 文案应明确提及角色缺失，实际: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "账号不存在") {
		t.Errorf("不应把「角色缺失」误报成「账号不存在」，实际: %s", rec.Body.String())
	}
}
