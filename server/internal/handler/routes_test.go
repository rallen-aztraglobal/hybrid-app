package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

// testServer 起一个内存 sqlite + 本地临时存储的完整 Echo 实例，挂载 routes.go 里真实注册的路由与中间件。
// 这样验证的是「实际生效的权限矩阵」，而不是重新写一遍规则去校验规则本身。
// 额外返回 *repo.Repo：RequireActiveAccount 中间件对每个请求都会按 claims.UserID 查库确认账号仍存在，
// issueToken 因此需要真实写入 admin_user 表的账号，而不能只签一个不存在于库里的 token。
func testServer(t *testing.T) (*echo.Echo, *auth.Manager, *service.Service, *repo.Repo) {
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
	ctx := context.Background()
	if err := seed.EnsureBrands(ctx, db); err != nil {
		t.Fatalf("seed 品牌失败: %v", err)
	}
	st, err := storage.NewLocal(t.TempDir(), "http://test/static")
	if err != nil {
		t.Fatalf("创建本地存储失败: %v", err)
	}
	cfg := config.Load()
	cfg.AppConfigTTLSecs = 600
	cfg.DefaultProbePath = "/healthz"
	cfg.RunnerToken = "test-runner-token"
	svc := service.New(cfg, r, st)

	authMgr := auth.NewManager("test-secret", "test", time.Hour, 24*time.Hour)
	authMgr.RunnerToken = cfg.RunnerToken
	h := New(cfg, svc, authMgr, r)

	e := echo.New()
	h.Register(e)
	return e, authMgr, svc, r
}

func sanitizeDBName(s string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(s)
}

// testUserSeq 保证同一测试内多次 issueToken 调用得到不重名的账号（不同测试各用独立内存库，
// 无需跨测试唯一，只需测试内唯一）。
var testUserSeq int64

// issueToken 建一个真实账号并签发其 access token，供请求头 Authorization: Bearer 使用。
// 必须真实建账号（而非只签一个不存在于库里的 token）：RequireActiveAccount 中间件按 UserID
// 查库确认账号仍存在，查不到（含已被软删除）一律 401。
func issueToken(t *testing.T, mgr *auth.Manager, r *repo.Repo, role string) string {
	t.Helper()
	n := atomic.AddInt64(&testUserSeq, 1)
	hash, err := auth.HashPassword("Passw0rd!1")
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	u := &model.AdminUser{
		Username:     fmt.Sprintf("t-%s-%d", role, n),
		PasswordHash: hash,
		Role:         role,
	}
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("创建测试账号失败: %v", err)
	}
	access, _, err := mgr.Issue(u)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	return access
}

func doReq(e *echo.Echo, method, path, bearer string, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	if bearer != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestPermission_ChannelArchiveIsAdminOnly 验证渠道归档/删除（DELETE /api/channels/:id）：
// user 应 403；admin 应能成功执行。
func TestPermission_ChannelArchiveIsAdminOnly(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()
	ch, err := svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	path := fmt.Sprintf("/api/channels/%d", ch.ID)

	userTok := issueToken(t, mgr, r, model.RoleUser)
	rec := doReq(e, http.MethodDelete, path, userTok, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("user 删除/归档渠道应 403，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec = doReq(e, http.MethodDelete, path, adminTok, "")
	if rec.Code != http.StatusOK {
		t.Errorf("admin 删除/归档渠道应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestPermission_StoreRoutesAreAdminOnly 验证系统设置的商店管理全部接口都是 admin-only：
// user 在 GET/POST/PUT/DELETE 上均应 403；admin 应能正常读写。
func TestPermission_StoreRoutesAreAdminOnly(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()
	st, err := svc.CreateStore(ctx, service.CreateStoreInput{Code: "hw", Name: "华为", Sort: 1})
	if err != nil {
		t.Fatalf("建商店失败: %v", err)
	}
	updatePath := fmt.Sprintf("/api/stores/%d", st.ID)

	userTok := issueToken(t, mgr, r, model.RoleUser)
	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/stores", ""},
		{http.MethodPost, "/api/stores", `{"code":"xm","name":"小米","sort":2}`},
		{http.MethodPut, updatePath, `{"name":"华为应用市场"}`},
		{http.MethodDelete, updatePath, ""},
	}
	for _, c := range cases {
		rec := doReq(e, c.method, c.path, userTok, c.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("user %s %s 应 403，实际 %d body=%s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}

	adminTok := issueToken(t, mgr, r, model.RoleAdmin)
	rec := doReq(e, http.MethodGet, "/api/stores", adminTok, "")
	if rec.Code != http.StatusOK {
		t.Errorf("admin GET /api/stores 应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	rec = doReq(e, http.MethodPut, updatePath, adminTok, `{"name":"华为应用市场"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("admin PUT /api/stores/:id 应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	rec = doReq(e, http.MethodDelete, updatePath, adminTok, "")
	if rec.Code != http.StatusOK {
		t.Errorf("admin DELETE /api/stores/:id 应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestPermission_UserCanDoNormalBusinessOps 验证 user 仍能完成日常业务操作：
// 新增/编辑渠道、创建构建任务——这些不是 admin-only。
func TestPermission_UserCanDoNormalBusinessOps(t *testing.T) {
	e, mgr, _, r := testServer(t)
	userTok := issueToken(t, mgr, r, model.RoleUser)

	rec := doReq(e, http.MethodPost, "/api/channels", userTok,
		`{"brandCode":"ap","flavorName":"ap01099","palCode":"PAL99","appName":"A99"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("user 新增渠道应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	rec = doReq(e, http.MethodPost, "/api/build/jobs", userTok,
		`{"brand":"ap","flavors":["ap01099"],"versionName":"1.0.0"}`)
	if rec.Code != http.StatusCreated {
		t.Errorf("user 创建构建任务应成功，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestCurrentVersionEndpoint_ReturnsHighestSuccessfulVersion 验证 GET /api/build/current-version：
// 缺 brand 参数 400；user（非 admin-only）可访问；无成功构建时 versionName=null；
// 有成功构建后返回该品牌语义版本最高的一条——这条路径与 CreateBuildJob 的强制校验
// 共用同一个 Service.CurrentVersion 实现，两者结果按设计不可能不一致。
func TestCurrentVersionEndpoint_ReturnsHighestSuccessfulVersion(t *testing.T) {
	e, mgr, svc, r := testServer(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	userTok := issueToken(t, mgr, r, model.RoleUser)

	// 缺 brand 参数应 400。
	rec := doReq(e, http.MethodGet, "/api/build/current-version", userTok, "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("缺少 brand 参数应 400，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// 尚无成功构建：user 可访问（非 admin-only），versionName 应为 null。
	rec = doReq(e, http.MethodGet, "/api/build/current-version?brand=ap", userTok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("user 应能访问该只读端点，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"versionName":null`) {
		t.Errorf("尚无成功构建时 versionName 应为 null，实际 body=%s", rec.Body.String())
	}

	// 造一条成功构建，再次请求应返回该版本。
	created, err := svc.CreateBuildJob(ctx, service.CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.3.8",
	})
	if err != nil {
		t.Fatalf("创建构建任务失败: %v", err)
	}
	claimed, err := svc.ClaimBuild(ctx, "runner")
	if err != nil || claimed == nil || claimed.ID != created.ID {
		t.Fatalf("领取失败: err=%v claimed=%+v", err, claimed)
	}
	if _, err := svc.ReportBuildStatus(ctx, claimed.ID, service.ReportBuildStatusInput{Status: model.BuildSuccess}); err != nil {
		t.Fatalf("上报成功失败: %v", err)
	}

	rec = doReq(e, http.MethodGet, "/api/build/current-version?brand=ap", userTok, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"versionName":"1.3.8"`) {
		t.Errorf("应返回当前版本 1.3.8，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestPermission_RunnerReachesBuildRoutesButNotAdminOnly 验证构建机静态令牌：
// 能正常到达 /api/build/claim 等机器接口，但碰不到 admin-only 的 Store / 渠道归档路由。
func TestPermission_RunnerReachesBuildRoutesButNotAdminOnly(t *testing.T) {
	e, _, svc, _ := testServer(t)
	ctx := context.Background()
	ch, err := svc.CreateChannel(ctx, service.CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	const runnerTok = "test-runner-token"

	// runner 能正常领取任务（即便队列为空也应 200，data=null）。
	rec := doReq(e, http.MethodPost, "/api/build/claim", runnerTok, `{"runner":"r1"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("runner 领取任务应 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// runner 碰不到 admin-only 路由：Store 管理。
	rec = doReq(e, http.MethodGet, "/api/stores", runnerTok, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("runner 访问 /api/stores 应 403，实际 %d body=%s", rec.Code, rec.Body.String())
	}

	// runner 碰不到 admin-only 路由：渠道归档/删除。
	rec = doReq(e, http.MethodDelete, fmt.Sprintf("/api/channels/%d", ch.ID), runnerTok, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("runner 删除/归档渠道应 403，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}
