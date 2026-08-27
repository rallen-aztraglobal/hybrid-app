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
	"github.com/hybrid-app/server/internal/perm"
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
	svc     *service.Service
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

	return &testApp{e: e, repo: r, svc: svc, authMgr: authMgr}
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

// createUserWithRoleID 同 createUser，但按角色 id（而不是内置角色名）挂角色——用于挂自定义
// 数据范围角色的测试账号。
func (a *testApp) createUserWithRoleID(t *testing.T, username string, roleID uint64) (uint64, string) {
	t.Helper()
	ctx := t.Context()
	hash, err := auth.HashPassword("pw123456")
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	u := &model.AdminUser{Username: username, PasswordHash: hash, Role: model.RoleOperator, RoleID: roleID}
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

// ---------- 数据权限（docs/admin/10-rbac.md「数据权限」节）端点级覆盖测试 ----------
//
// scopeFixture 建好一套「限定 brand=ap」的测试夹具：
//   - ap001/ap002（品牌 ap）、bp001（品牌 bp）三个渠道；
//   - apAllRole：permCodes=全量，scopeAllBrands=false（仅 ap），scopeAllChannels=true；
//   - apChannelRole：同上但 scopeAllChannels=false，只勾了 ap001（不含 ap002）——用来测
//     「渠道级」粒度的越界拒绝（POST /build/jobs 混入范围外渠道整单拒绝、POST /channels
//     在指定渠道范围下 403）。
//   - 分别给 ap/bp 各建一条构建记录、一条推送活动、一条设备上报、一个上架包，
//     供列表类端点验证"只返回 ap 的数据"。
type scopeFixture struct {
	app                        *testApp
	ap001, ap002, bp001        *model.Channel
	apAllTok, apChannelTok     string
	apAllRoleID, apChannelID   uint64
	apBuildID, bpBuildID       uint64
	apCampaignID, bpCampaignID uint64
}

func newScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()
	app := newTestApp(t)
	ctx := t.Context()
	rootID, _ := app.createUser(t, "scope-root", "超级管理员")

	ap001, err := app.svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap001", PalCode: "P1", AppName: "AP One",
	})
	if err != nil {
		t.Fatalf("建渠道 ap001 失败: %v", err)
	}
	ap002, err := app.svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap002", PalCode: "P2", AppName: "AP Two",
	})
	if err != nil {
		t.Fatalf("建渠道 ap002 失败: %v", err)
	}
	bp001, err := app.svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "bp", FlavorName: "bp001", PalCode: "P3", AppName: "BP One",
	})
	if err != nil {
		t.Fatalf("建渠道 bp001 失败: %v", err)
	}

	apAllRole, err := app.svc.CreateRole(ctx, rootID, service.RoleInput{
		Name: "ap全渠道专员", PermCodes: perm.AllCodes(),
		ScopeAllBrands: false, BrandCodes: []string{"ap"}, ScopeAllChannels: true,
	})
	if err != nil {
		t.Fatalf("建 ap 全渠道角色失败: %v", err)
	}
	apChannelRole, err := app.svc.CreateRole(ctx, rootID, service.RoleInput{
		Name: "ap单渠道专员", PermCodes: perm.AllCodes(),
		ScopeAllBrands: false, BrandCodes: []string{"ap"},
		ScopeAllChannels: false, ChannelIDs: []uint64{ap001.ID},
	})
	if err != nil {
		t.Fatalf("建 ap 单渠道角色失败: %v", err)
	}
	_, apAllTok := app.createUserWithRoleID(t, "ap-all-caller", apAllRole.ID)
	_, apChannelTok := app.createUserWithRoleID(t, "ap-channel-caller", apChannelRole.ID)

	// 构建记录：ap、bp 各一条（用于 GET /build/records 列表过滤）。
	apBuild, err := app.svc.CreateBuildJob(ctx, auth.FullScope(), service.CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap001"}, VersionName: "1.0.0",
	})
	if err != nil {
		t.Fatalf("建 ap 构建记录失败: %v", err)
	}
	bpBuild, err := app.svc.CreateBuildJob(ctx, auth.FullScope(), service.CreateBuildJobInput{
		Brand: "bp", Flavors: []string{"bp001"}, VersionName: "1.0.0",
	})
	if err != nil {
		t.Fatalf("建 bp 构建记录失败: %v", err)
	}

	// 推送活动：ap、bp 各一条目标（用于 GET /push/campaigns 列表过滤）。
	apCampaign, err := app.svc.CreateCampaign(ctx, auth.FullScope(), service.PushCampaignInput{
		Name: "ap活动", Title: "t", Body: "b", TargetAppIDs: []string{ap001.ApplicationID},
	}, "setup")
	if err != nil {
		t.Fatalf("建 ap 推送活动失败: %v", err)
	}
	bpCampaign, err := app.svc.CreateCampaign(ctx, auth.FullScope(), service.PushCampaignInput{
		Name: "bp活动", Title: "t", Body: "b", TargetAppIDs: []string{bp001.ApplicationID},
	}, "setup")
	if err != nil {
		t.Fatalf("建 bp 推送活动失败: %v", err)
	}

	// 设备：ap、bp 各一条上报（用于 GET /devices 列表过滤）。
	for _, d := range []*model.Channel{ap001, bp001} {
		dev := &model.ChannelDevice{
			DeviceKey: "device-" + d.FlavorName, ApplicationID: d.ApplicationID,
			BrandCode: d.FlavorName[:2], AppName: d.AppName, GAID: "gaid-" + d.FlavorName,
		}
		if err := app.repo.DB().WithContext(ctx).Create(dev).Error; err != nil {
			t.Fatalf("建设备上报 %s 失败: %v", d.FlavorName, err)
		}
	}

	// 上架包：ap、bp 各一个（用于 GET /listings 列表过滤）。
	if _, err := app.svc.CreateListing(ctx, service.CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.ap.listing", Name: "ap上架包", PalCode: "1",
	}); err != nil {
		t.Fatalf("建 ap 上架包失败: %v", err)
	}
	if _, err := app.svc.CreateListing(ctx, service.CreateListingInput{
		BrandCode: "bp", Platform: "android", BundleID: "com.bp.listing", Name: "bp上架包", PalCode: "1",
	}); err != nil {
		t.Fatalf("建 bp 上架包失败: %v", err)
	}

	return &scopeFixture{
		app: app, ap001: ap001, ap002: ap002, bp001: bp001,
		apAllTok: apAllTok, apChannelTok: apChannelTok,
		apAllRoleID: apAllRole.ID, apChannelID: apChannelRole.ID,
		apBuildID: apBuild.ID, bpBuildID: bpBuild.ID,
		apCampaignID: apCampaign.ID, bpCampaignID: bpCampaign.ID,
	}
}

// TestDataScopeSingleItemAndCreateGate 是数据权限的端点级覆盖测试核心：用一个限定 brand=ap 的
// 角色逐条打「强制点」清单里的单体类（越界 404）与新建渠道（scope_all_channels=false → 403）
// 端点，表驱动列全、避免以后加新端点漏掉范围过滤。
func TestDataScopeSingleItemAndCreateGate(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	cases := []struct {
		name     string
		method   string
		path     string
		token    string
		wantCode int
	}{
		// ---- 单体类：越界（bp001／brand=bp）一律 404，不泄露存在性 ----
		{"GET channel(bp) 越界 404", http.MethodGet, fmt.Sprintf("/api/channels/%d", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"PUT channel(bp) 越界 404", http.MethodPut, fmt.Sprintf("/api/channels/%d", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"DELETE channel(bp) 越界 404", http.MethodDelete, fmt.Sprintf("/api/channels/%d", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"PUT channel(bp)/domains 越界 404", http.MethodPut, fmt.Sprintf("/api/channels/%d/domains", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"POST channel(bp)/icon 越界 404", http.MethodPost, fmt.Sprintf("/api/channels/%d/icon", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"POST channel(bp)/splash 越界 404", http.MethodPost, fmt.Sprintf("/api/channels/%d/splash", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"GET channel(bp)/res.zip 越界 404", http.MethodGet, fmt.Sprintf("/api/channels/%d/res.zip", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"GET channel(bp)/latest-apk 越界 404", http.MethodGet, fmt.Sprintf("/api/channels/%d/latest-apk", f.bp001.ID), f.apAllTok, http.StatusNotFound},
		{"GET brand(bp)/domains 越界 404", http.MethodGet, "/api/brands/bp/domains", f.apAllTok, http.StatusNotFound},
		{"PUT brand(bp)/domains 越界 404", http.MethodPut, "/api/brands/bp/domains", f.apAllTok, http.StatusNotFound},
		// ---- 范围内（ap001／brand=ap）应正常放行（不是 404） ----
		{"GET channel(ap) 范围内 200", http.MethodGet, fmt.Sprintf("/api/channels/%d", f.ap001.ID), f.apAllTok, http.StatusOK},
		{"GET brand(ap)/domains 范围内 200", http.MethodGet, "/api/brands/ap/domains", f.apAllTok, http.StatusOK},
		// ---- 新建渠道：scope_all_channels=false 的角色一律 403（即便品牌在范围内） ----
		{"POST channels 品牌越界(bp) 403", http.MethodPost, "/api/channels", f.apAllTok, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body any
			if tc.path == "/api/channels" && tc.method == http.MethodPost {
				body = map[string]any{"brandCode": "bp", "flavorName": "bp999", "palCode": "1", "appName": "X"}
			}
			rec := app.do(tc.method, tc.path, tc.token, body)
			if rec.Code != tc.wantCode {
				t.Errorf("%s %s 应 %d，实际 %d: %s", tc.method, tc.path, tc.wantCode, rec.Code, rec.Body.String())
			}
		})
	}

	// POST /channels：scope_all_channels=false 的角色即便品牌在范围内也一律 403（不是「找不到品牌」
	// 那种模糊错误，是明确的「你不能新建」文案——新建完自己也看不到，是个死局）。
	rec := app.do(http.MethodPost, "/api/channels", f.apChannelTok,
		map[string]any{"brandCode": "ap", "flavorName": "ap998", "palCode": "1", "appName": "X"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("指定渠道范围的角色新建渠道应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}

	// 对照组：全渠道范围(apAllTok)在自己品牌(ap)内新建应该放行。
	rec = app.do(http.MethodPost, "/api/channels", f.apAllTok,
		map[string]any{"brandCode": "ap", "flavorName": "ap997", "palCode": "1", "appName": "X"})
	if rec.Code == http.StatusForbidden || rec.Code == http.StatusNotFound {
		t.Errorf("全渠道范围角色在自己品牌内新建渠道不应被数据权限拦截，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDataScopeListFiltering 覆盖「强制点」列表类端点：查询层过滤，限定 brand=ap 的角色
// 只应看到 ap 的数据，看不到 bp 的（GET /channels、/brands、/build/records、/push/campaigns、
// /devices、/listings）。
func TestDataScopeListFiltering(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	type listCheck struct {
		name           string
		path           string
		mustContain    string
		mustNotContain string
	}
	checks := []listCheck{
		{"GET /channels 只含 ap", "/api/channels", f.ap001.FlavorName, f.bp001.FlavorName},
		{"GET /brands 只含 ap", "/api/brands", `"code":"ap"`, `"code":"bp"`},
		{"GET /build/records 只含 ap", "/api/build/records", f.ap001.FlavorName, f.bp001.FlavorName},
		{"GET /push/campaigns 只含 ap", "/api/push/campaigns", "ap活动", "bp活动"},
		{"GET /devices 只含 ap", "/api/devices", f.ap001.ApplicationID, f.bp001.ApplicationID},
		{"GET /listings 只含 ap", "/api/listings", "com.ap.listing", "com.bp.listing"},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			rec := app.do(http.MethodGet, c.path, f.apAllTok, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s 应 200，实际 %d: %s", c.path, rec.Code, rec.Body.String())
			}
			bodyStr := rec.Body.String()
			if !strings.Contains(bodyStr, c.mustContain) {
				t.Errorf("%s 结果应包含 %q（ap 的数据），实际: %s", c.path, c.mustContain, bodyStr)
			}
			if strings.Contains(bodyStr, c.mustNotContain) {
				t.Errorf("%s 结果不应包含 %q（bp 的数据，越权泄露），实际: %s", c.path, c.mustNotContain, bodyStr)
			}
		})
	}
}

// TestDataScopeBuildJobRejectsOutOfRangeBrand 覆盖「强制点」打包一节：POST /build/jobs 的 brand
// 必须在调用者范围内；brand 越界直接拒绝（403），不像普通业务校验那样是 400。
func TestDataScopeBuildJobRejectsOutOfRangeBrand(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	rec := app.do(http.MethodPost, "/api/build/jobs", f.apAllTok, map[string]any{
		"brand": "bp", "flavors": []string{f.bp001.FlavorName}, "versionName": "1.0.0",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("brand=bp 越界应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDataScopeBuildJobRejectsOutOfRangeChannel 覆盖「有一个越界整单拒绝」的渠道粒度：
// apChannelTok 只勾了 ap001（不含 ap002），提交 brand=ap 但 flavors 混入 ap002 应整单被拒——
// 即便 ap002 是合法的、已启用的 ap 品牌渠道（品牌校验会通过），只是不在调用者的渠道范围内。
func TestDataScopeBuildJobRejectsOutOfRangeChannel(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	// 单独提交范围内的 ap001 应成功。
	rec := app.do(http.MethodPost, "/api/build/jobs", f.apChannelTok, map[string]any{
		"brand": "ap", "flavors": []string{f.ap001.FlavorName}, "versionName": "1.0.0",
	})
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Errorf("提交范围内的 ap001 应成功，实际 %d: %s", rec.Code, rec.Body.String())
	}

	// 混入范围外的 ap002 应整单拒绝。
	rec = app.do(http.MethodPost, "/api/build/jobs", f.apChannelTok, map[string]any{
		"brand": "ap", "flavors": []string{f.ap001.FlavorName, f.ap002.FlavorName}, "versionName": "1.0.0",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("混入范围外的 ap002 应整单被拒绝(403)，实际 %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDataScopeBuildManifestTrimsOutOfRangeBrand 覆盖「GET /build/manifest 按范围裁剪」：
// 请求一个不在范围内的品牌，返回不是硬 404（品牌本身不敏感），而是 channels 被裁剪为空。
func TestDataScopeBuildManifestTrimsOutOfRangeBrand(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	rec := app.do(http.MethodGet, "/api/build/manifest?brand=bp", f.apAllTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("manifest 应 200（裁剪不是拒绝），实际 %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), f.bp001.FlavorName) {
		t.Errorf("越界品牌的 manifest 不应带出任何渠道明细，实际: %s", rec.Body.String())
	}
}

// TestDataScopePushCreateAndAudience 覆盖「强制点」推送一节：活动 brand 必须在范围内才能建；
// GET /push/audience 按范围过滤越界的 appId（静默丢弃，不报错）。
func TestDataScopePushCreateAndAudience(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app

	// 建活动：目标是 bp 渠道 → 403。
	rec := app.do(http.MethodPost, "/api/push/campaigns", f.apAllTok, map[string]any{
		"name": "越权活动", "title": "t", "body": "b", "targetAppIds": []string{f.bp001.ApplicationID},
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("目标含 bp 渠道的活动应 403，实际 %d: %s", rec.Code, rec.Body.String())
	}

	// audience：appIds 混入 ap + bp，返回结果不应含 bp 的 appId。
	rec = app.do(http.MethodGet,
		"/api/push/audience?appIds="+f.ap001.ApplicationID+","+f.bp001.ApplicationID, f.apAllTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("audience 应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), f.bp001.ApplicationID) {
		t.Errorf("audience 结果不应包含范围外的 bp appId，实际: %s", rec.Body.String())
	}
}

// TestRoleScopeWireContract 锁死 /api/roles 上数据范围字段的**线上形状**（10-rbac.md「API」一节）：
// 平铺的 {scopeAllBrands, brandCodes, scopeAllChannels, channelIds}，请求与响应两侧都是。
//
// 背景：前端一度按登录/me 响应体里嵌套的 {scope:{allBrands,...}} 形状发角色请求，Echo Bind
// 静默丢弃未知字段 → 角色落库成「零品牌零渠道」，界面上却因为读不到平铺字段而回显「全部品牌」，
// 表现为「配了全部品牌的角色登录后什么包都看不到」。Go 侧的结构体测试抓不到这类纯 JSON 键名漂移，
// 所以这里直接用裸 map 走 HTTP 断言键名。
func TestRoleScopeWireContract(t *testing.T) {
	app := newTestApp(t)
	_, tok := app.createUser(t, "rootwire", "超级管理员")

	// 解析 Envelope.data 里的角色对象为裸 map，逐字断言键名。
	roleData := func(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var env struct {
			Code int            `json:"code"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("解析响应失败: %v (%s)", err, rec.Body.String())
		}
		return env.Data
	}

	// 1) 请求侧：平铺字段必须被真正读进去（brandCodes 落到 role_brand）。
	rec := app.do(http.MethodPost, "/api/roles", tok, map[string]any{
		"name": "开发", "description": "wire", "permCodes": []string{"page:channels"},
		"scopeAllBrands": false, "brandCodes": []string{"ap"},
		"scopeAllChannels": true, "channelIds": []uint64{},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("建角色应 201，实际 %d: %s", rec.Code, rec.Body.String())
	}
	data := roleData(t, rec)

	// 2) 响应侧：必须是平铺键名（前端据此回显；键名漂移过就是这个 bug 的另一半）。
	for _, k := range []string{"scopeAllBrands", "brandCodes", "scopeAllChannels", "channelIds"} {
		if _, ok := data[k]; !ok {
			t.Errorf("角色响应缺少数据范围字段 %q: %s", k, rec.Body.String())
		}
	}
	if _, nested := data["scope"]; nested {
		t.Errorf("角色响应不应改用嵌套 scope 对象（契约是平铺字段）: %s", rec.Body.String())
	}
	if data["scopeAllBrands"] != false || data["scopeAllChannels"] != true {
		t.Errorf("数据范围标志位未按入参落库: %s", rec.Body.String())
	}
	codes, _ := data["brandCodes"].([]any)
	if len(codes) != 1 || codes[0] != "ap" {
		t.Errorf("brandCodes 未落库，实际 %v: %s", data["brandCodes"], rec.Body.String())
	}

	// 3) PUT 整体替换：改回「全部品牌 + 全部渠道」同样按平铺字段生效。
	roleID := uint64(data["id"].(float64))
	rec = app.do(http.MethodPut, fmt.Sprintf("/api/roles/%d", roleID), tok, map[string]any{
		"name": "开发", "description": "wire", "permCodes": []string{"page:channels"},
		"scopeAllBrands": true, "brandCodes": []string{},
		"scopeAllChannels": true, "channelIds": []uint64{},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("改角色应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	if data = roleData(t, rec); data["scopeAllBrands"] != true || data["scopeAllChannels"] != true {
		t.Errorf("PUT 未把数据范围改成全部: %s", rec.Body.String())
	}
}
