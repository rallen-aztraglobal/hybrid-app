package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/seed"
	"github.com/hybrid-app/server/internal/storage"
)

// newTestService 起一个 sqlite 内存库 + 临时本地存储的 Service。
// 每个测试用独立命名的共享缓存内存库，避免跨用例数据串扰。
func newTestService(t *testing.T) (*Service, *repo.Repo) {
	t.Helper()
	// 用测试名作为唯一 DB 名：同一 *gorm.DB 连接池内共享 schema，不同测试互相隔离。
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeDBName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	// 限制为单连接，确保内存库不被连接池里多个连接看成不同实例。
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
	return New(cfg, r, st), r
}

// TestCreateChannelDerivesAppID 验证 applicationId 由 <品牌包前缀>.<flavor> 派生（ADR-0009）。
func TestCreateChannelDerivesAppID(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "ArenaPlus",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if ch.ApplicationID != "com.arenaplus.ap01018" {
		t.Errorf("appId 应派生为 com.arenaplus.ap01018，实际 %q", ch.ApplicationID)
	}
	// gp 品牌前缀不同。
	gch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "gp", FlavorName: "gp001", PalCode: "PAL2", AppName: "GameZone",
	})
	if err != nil {
		t.Fatalf("创建 gp 失败: %v", err)
	}
	if gch.ApplicationID != "com.gamezone.gp001" {
		t.Errorf("appId 应派生为 com.gamezone.gp001，实际 %q", gch.ApplicationID)
	}
}

// TestCreateChannelUniqueness 验证唯一性以 applicationId 与 (brand,flavor) 为准；pal_code 不再查重。
func TestCreateChannelUniqueness(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	base := CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "1053259232660520961", AppName: "ArenaPlus",
	}
	if _, err := svc.CreateChannel(ctx, base); err != nil {
		t.Fatalf("首次创建应成功: %v", err)
	}

	// 重复 (brand, flavor) → appId 也重复，应被拒。
	_, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "999", AppName: "X",
	})
	if err == nil {
		t.Fatal("重复 (brand,flavor) 应被唯一性校验拒绝")
	}
	if !strings.Contains(err.Error(), "flavor") && !strings.Contains(err.Error(), "applicationId") {
		t.Errorf("错误信息应指出 flavor/applicationId 冲突，实际: %v", err)
	}
}

// TestPalCodeNotUniqueAcrossBrands 验证 ADR-0009：同一 palcode 可跨品牌重复。
func TestPalCodeNotUniqueAcrossBrands(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	const sharedPal = "MKT-SHARED-001"

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: sharedPal, AppName: "A",
	}); err != nil {
		t.Fatalf("ap 创建失败: %v", err)
	}
	// 同 palcode、不同品牌 → 应允许（不再全局唯一）。
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "gp", FlavorName: "gp001", PalCode: sharedPal, AppName: "G",
	}); err != nil {
		t.Fatalf("跨品牌复用 palcode 应被允许，却失败: %v", err)
	}
	// 同品牌内复用 palcode 也允许（仅 appId/flavor 唯一）。
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01019", PalCode: sharedPal, AppName: "A2",
	}); err != nil {
		t.Fatalf("同品牌不同 flavor 复用 palcode 应被允许，却失败: %v", err)
	}
}

func TestCreateChannelValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bad := []CreateChannelInput{
		{BrandCode: "ap", FlavorName: "01ap", PalCode: "1", AppName: "A"},    // flavor 以数字开头（非法包名片段）
		{BrandCode: "ap", FlavorName: "ap_01ap", PalCode: "1", AppName: "A"}, // 分段以数字开头（商店后缀分段非法）
		{BrandCode: "ap", FlavorName: "ap01_", PalCode: "1", AppName: "A"},   // 尾随下划线（空段）
		{BrandCode: "ap", FlavorName: "_ap01", PalCode: "1", AppName: "A"},   // 前导下划线（空段）
		{BrandCode: "ap", FlavorName: "ap__01", PalCode: "1", AppName: "A"},  // 连续下划线（空段）
		{BrandCode: "ap", FlavorName: "", PalCode: "1", AppName: "A"},        // flavor 空
		{BrandCode: "zz", FlavorName: "zz01", PalCode: "1", AppName: "A"},    // 品牌不存在
	}
	for i, in := range bad {
		if _, err := svc.CreateChannel(ctx, in); err == nil {
			t.Errorf("用例 %d 应被拒绝: %+v", i, in)
		}
	}
}

// TestCSVImportDerivesAppIDAndCorrectsMismatch 验证 import 按 flavor 派生 appId，
// 历史 mismatch 行（ap01035|com.arenaplus.ap01034）被修正为派生值并作独立渠道导入（ADR-0009）。
func TestCSVImportDerivesAppIDAndCorrectsMismatch(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()

	csv := `# header
ap01034|com.arenaplus.ap01034|1053259243279695873|Arena Plus
ap01035|com.arenaplus.ap01034|1053259242391433216|Arena Plus
`
	rep, err := seed.ImportCSV(ctx, r, "ap", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	// 两行都应入库（ap01035 派生为 com.arenaplus.ap01035，不再与 ap01034 冲突）。
	if rep.Inserted != 2 {
		t.Errorf("应插入 2 条，实际 %d（skipped=%v）", rep.Inserted, rep.Skipped)
	}
	if len(rep.Skipped) != 0 {
		t.Errorf("不应跳过任何行，实际跳过 %d: %v", len(rep.Skipped), rep.Skipped)
	}
	// ap01035 的 CSV applicationId 与派生值不符 → 记一条修正。
	if len(rep.Corrected) != 1 {
		t.Errorf("应有 1 条修正（ap01035），实际 %d: %v", len(rep.Corrected), rep.Corrected)
	}

	// 验证库内 ap01035 的 appId 已被修正为派生值。
	list, total, err := svc.ListChannels(ctx, repo.ChannelFilter{BrandCode: "ap"})
	if err != nil {
		t.Fatalf("列渠道失败: %v", err)
	}
	if total != 2 {
		t.Fatalf("库内应有 2 条，实际 %d: %v", total, flavorNames(list))
	}
	for i := range list {
		if list[i].FlavorName == "ap01035" && list[i].ApplicationID != "com.arenaplus.ap01035" {
			t.Errorf("ap01035 的 appId 应被派生修正为 com.arenaplus.ap01035，实际 %q", list[i].ApplicationID)
		}
	}
}

func TestAppConfigInheritsBrandDomains(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// 设置品牌 ap 的域名：主 + 1 备用。
	_, err := svc.SetBrandDomains(ctx, "ap", []domainutil.DomainInput{
		{Position: 0, URL: "https://arenaplus.ph", Enabled: true},
		{Position: 1, URL: "https://ap-backup.net", Enabled: true},
	})
	if err != nil {
		t.Fatalf("设置品牌域名失败: %v", err)
	}

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL123", AppName: "A",
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}
	const appID = "com.arenaplus.ap01018"
	if ch.ApplicationID != appID {
		t.Fatalf("appId 应为 %s，实际 %q", appID, ch.ApplicationID)
	}

	// 默认继承品牌域名；解析键为 appId（ADR-0009）。
	cfg, err := svc.AppConfigForAppId(ctx, appID)
	if err != nil {
		t.Fatalf("组装 app config 失败: %v", err)
	}
	if cfg.AppID != appID {
		t.Errorf("config.AppID 应为 %s，实际 %q", appID, cfg.AppID)
	}
	if cfg.Palcode != "PAL123" {
		t.Errorf("config.Palcode 应回显 PAL123，实际 %q", cfg.Palcode)
	}
	if len(cfg.Domains) != 2 || cfg.Domains[0] != "https://arenaplus.ph" {
		t.Fatalf("应继承品牌 2 个域名，实际: %v", cfg.Domains)
	}
	if cfg.ProbePath != "/healthz" {
		t.Errorf("probePath 应为 /healthz，实际 %q", cfg.ProbePath)
	}

	// 渠道覆盖域名后，config 用覆盖值。
	_, err = svc.SetChannelDomains(ctx, ch.ID, false, []domainutil.DomainInput{
		{Position: 0, URL: "https://special-ap01018.com", Enabled: true},
	})
	if err != nil {
		t.Fatalf("设置渠道覆盖域名失败: %v", err)
	}
	cfg2, err := svc.AppConfigForAppId(ctx, appID)
	if err != nil {
		t.Fatalf("组装 app config(覆盖) 失败: %v", err)
	}
	if len(cfg2.Domains) != 1 || cfg2.Domains[0] != "https://special-ap01018.com" {
		t.Fatalf("应使用渠道覆盖域名，实际: %v", cfg2.Domains)
	}

	// 切回继承。
	_, err = svc.SetChannelDomains(ctx, ch.ID, true, nil)
	if err != nil {
		t.Fatalf("切回继承失败: %v", err)
	}
	cfg3, _ := svc.AppConfigForAppId(ctx, appID)
	if len(cfg3.Domains) != 2 {
		t.Fatalf("切回继承后应有 2 个品牌域名，实际: %v", cfg3.Domains)
	}
}

func TestAppConfigUnknownAppID(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.AppConfigForAppId(context.Background(), "com.does.notexist"); err == nil {
		t.Fatal("未知 appId 应返回错误")
	}
}

func TestArchivedChannelExcludedFromConfig(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, _ = svc.SetBrandDomains(ctx, "gp", []domainutil.DomainInput{{Position: 0, URL: "https://gzone.ph", Enabled: true}})
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "gp", FlavorName: "gp001", PalCode: "GPPAL", AppName: "G",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := svc.ArchiveChannel(ctx, ch.ID); err != nil {
		t.Fatalf("归档失败: %v", err)
	}
	if _, err := svc.AppConfigForAppId(ctx, "com.gamezone.gp001"); err == nil {
		t.Fatal("已归档渠道不应可被 app config 取到")
	}
}

func TestBuildManifestRendersCSVLine(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, _ = svc.SetBrandDomains(ctx, "ap", []domainutil.DomainInput{{Position: 0, URL: "https://arenaplus.ph", Enabled: true}})
	_, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PALX", AppName: "ArenaPlus:USA Basketball Live",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	m, err := svc.BuildManifestForBrand(ctx, "ap")
	if err != nil {
		t.Fatalf("manifest 失败: %v", err)
	}
	if len(m.Channels) != 1 {
		t.Fatalf("应有 1 个渠道，实际 %d", len(m.Channels))
	}
	want := "ap01018|com.arenaplus.ap01018|PALX|ArenaPlus:USA Basketball Live"
	if got := m.Channels[0].CSVLine(); got != want {
		t.Errorf("CSV 行渲染 = %q，期望 %q", got, want)
	}
	// effectiveDomains 应继承品牌域名。
	if len(m.Channels[0].EffectiveDomains) != 1 || m.Channels[0].EffectiveDomains[0] != "https://arenaplus.ph" {
		t.Errorf("manifest 渠道应继承品牌域名，实际: %v", m.Channels[0].EffectiveDomains)
	}
}

// TestVersionNameValidation 验证 versionName 必须形如 X.Y.Z（ADR-0008）。
func TestVersionNameValidation(t *testing.T) {
	ok := []string{"1.0.0", "10.20.30", "0.0.1"}
	bad := []string{"1.0", "1", "1.0.0.0", "v1.0.0", "1.0.x", "", "1.0.0-rc1"}
	for _, v := range ok {
		if !validVersionName(v) {
			t.Errorf("%q 应为合法 versionName", v)
		}
	}
	for _, v := range bad {
		if validVersionName(v) {
			t.Errorf("%q 应为非法 versionName", v)
		}
	}
}

// TestBuildJobLifecycle 跑通：入队 → 领取 → 上报日志 → 产物 → 成功 → 渠道最新包可取。
func TestBuildJobLifecycle(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}

	// 入队（未给 name → 用默认名）。
	rec, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.2.3",
	})
	if err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	if rec.Status != model.BuildQueued {
		t.Errorf("初始状态应为 queued，实际 %q", rec.Status)
	}
	if rec.Name == "" || !strings.HasPrefix(rec.Name, "ap-1.2.3-") {
		t.Errorf("默认任务名应形如 ap-1.2.3-YYYYMMDD-HHmm，实际 %q", rec.Name)
	}

	// 非法 versionName 应被拒。
	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.2",
	}); err == nil {
		t.Error("versionName=1.2 应被拒")
	}
	// 不属于品牌的 flavor 应被拒。
	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"nonexist"}, VersionName: "1.0.0",
	}); err == nil {
		t.Error("未知 flavor 应被拒")
	}

	// 领取。
	claimed, err := svc.ClaimBuild(ctx, "runner-1")
	if err != nil {
		t.Fatalf("领取失败: %v", err)
	}
	if claimed == nil || claimed.ID != rec.ID {
		t.Fatalf("应领取到刚入队的任务，实际 %+v", claimed)
	}
	if claimed.Status != model.BuildRunning {
		t.Errorf("领取后状态应为 running，实际 %q", claimed.Status)
	}
	// 队列已空，再领应得 nil。
	if again, err := svc.ClaimBuild(ctx, "runner-2"); err != nil || again != nil {
		t.Errorf("队列空时应返回 nil，实际 rec=%v err=%v", again, err)
	}

	// 上报日志（分两段），验证分段拉取。
	if err := svc.AppendBuildLog(ctx, rec.ID, "Task :app:assembleAp01018Release\n"); err != nil {
		t.Fatalf("追加日志失败: %v", err)
	}
	if err := svc.AppendBuildLog(ctx, rec.ID, "BUILD SUCCESSFUL\n"); err != nil {
		t.Fatalf("追加日志失败: %v", err)
	}
	seg, err := svc.BuildLog(ctx, rec.ID, 0)
	if err != nil {
		t.Fatalf("取日志失败: %v", err)
	}
	if !strings.Contains(seg.Log, "assembleAp01018Release") || !strings.Contains(seg.Log, "BUILD SUCCESSFUL") {
		t.Errorf("日志内容不完整: %q", seg.Log)
	}
	if seg.Done {
		t.Error("running 状态 done 应为 false")
	}
	// 增量拉取：从 next 起应为空。
	seg2, _ := svc.BuildLog(ctx, rec.ID, seg.Next)
	if seg2.Log != "" {
		t.Errorf("从游标 next 起应无新日志，实际 %q", seg2.Log)
	}

	// 上报产物。
	if _, err := svc.AddBuildArtifact(ctx, rec.ID, AddBuildArtifactInput{
		Flavor: "ap01018", ApkURL: "https://console/apks/ap/ap01018/1.2.3/app-ap01018-release.apk", Size: 12345,
	}); err != nil {
		t.Fatalf("上报产物失败: %v", err)
	}

	// 成功。
	done, err := svc.ReportBuildStatus(ctx, rec.ID, ReportBuildStatusInput{Status: model.BuildSuccess, LogExcerpt: "BUILD SUCCESSFUL"})
	if err != nil {
		t.Fatalf("上报成功失败: %v", err)
	}
	if done.Status != model.BuildSuccess || done.FinishedAt == nil {
		t.Errorf("终态应为 success 且记 finished_at，实际 status=%q finished=%v", done.Status, done.FinishedAt)
	}
	if len(done.Artifacts) != 1 {
		t.Errorf("应有 1 个产物，实际 %d", len(done.Artifacts))
	}

	// 日志 done 应为 true。
	segDone, _ := svc.BuildLog(ctx, rec.ID, 0)
	if !segDone.Done {
		t.Error("终态后日志 done 应为 true")
	}

	// 渠道最新包应取到该产物。
	latest, err := svc.LatestApkForChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("取最新包失败: %v", err)
	}
	if latest == nil || latest.Flavor != "ap01018" {
		t.Fatalf("应取到 ap01018 最新包，实际 %+v", latest)
	}

	// 列表也应带 latestApkUrl。
	list, _, err := svc.ListChannels(ctx, repo.ChannelFilter{BrandCode: "ap"})
	if err != nil {
		t.Fatalf("列渠道失败: %v", err)
	}
	if len(list) != 1 || list[0].LatestApkURL == "" {
		t.Errorf("渠道列表应填充 latestApkUrl，实际: %+v", list)
	}
}

// TestLatestApkOnlyFromSuccess 验证 latest-apk 只统计成功构建的产物。
func TestLatestApkOnlyFromSuccess(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}
	rec, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.0.0",
	})
	if err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	if _, err := svc.ClaimBuild(ctx, "r"); err != nil {
		t.Fatalf("领取失败: %v", err)
	}
	// 产物存在，但任务以 failed 收尾 → 不应被 latest-apk 取到。
	if _, err := svc.AddBuildArtifact(ctx, rec.ID, AddBuildArtifactInput{
		Flavor: "ap01018", ApkURL: "https://console/apks/x.apk",
	}); err != nil {
		t.Fatalf("上报产物失败: %v", err)
	}
	if _, err := svc.ReportBuildStatus(ctx, rec.ID, ReportBuildStatusInput{Status: model.BuildFailed}); err != nil {
		t.Fatalf("上报失败状态失败: %v", err)
	}
	latest, err := svc.LatestApkForChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("取最新包失败: %v", err)
	}
	if latest != nil {
		t.Errorf("failed 构建的产物不应被取到，实际 %+v", latest)
	}
}

// TestDeriveApplicationIDStoreSuffix 验证商店后缀 flavor（<base>_<storeCode>）派生出点号分段 appId，
// 且老数据（flavor 无下划线）派生结果不变（向后兼容）。
func TestDeriveApplicationIDStoreSuffix(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()

	store, err := svc.CreateStore(ctx, CreateStoreInput{Code: "hw", Name: "华为"})
	if err != nil {
		t.Fatalf("创建商店失败: %v", err)
	}

	// 老数据：flavor 无下划线，appId 不变。
	old, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "bp", FlavorName: "bpocmhuawei004", PalCode: "P1", AppName: "老渠道",
	})
	if err != nil {
		t.Fatalf("创建老渠道失败: %v", err)
	}
	if old.ApplicationID != "com.bingoplus.bpocmhuawei004" {
		t.Errorf("老数据 appId 应不变，实际 %q", old.ApplicationID)
	}

	// 商店包：flavor = <base>_<storeCode> → appId 点号分段。
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "bp", FlavorName: "bpocmhuawei004_hw", PalCode: "P2", AppName: "商店渠道", StoreID: &store.ID,
	})
	if err != nil {
		t.Fatalf("创建商店渠道失败: %v", err)
	}
	if ch.ApplicationID != "com.bingoplus.bpocmhuawei004.hw" {
		t.Errorf("商店渠道 appId 应为点号分段，实际 %q", ch.ApplicationID)
	}
	if ch.StoreID == nil || *ch.StoreID != store.ID {
		t.Errorf("渠道应记录 storeId=%d，实际 %+v", store.ID, ch.StoreID)
	}

	// flavor 后缀与所选商店不一致应被拒。
	xm, err := svc.CreateStore(ctx, CreateStoreInput{Code: "xm", Name: "小米"})
	if err != nil {
		t.Fatalf("创建商店失败: %v", err)
	}
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "bp", FlavorName: "bpocmhuawei004_hw2", PalCode: "P3", AppName: "X", StoreID: &xm.ID,
	}); err == nil {
		t.Error("flavor 后缀与所选商店不一致应被拒绝")
	}

	// 停用商店后不能新建渠道。
	if _, err := svc.UpdateStore(ctx, xm.ID, UpdateStoreInput{Status: strPtr(model.StoreDisabled)}); err != nil {
		t.Fatalf("停用商店失败: %v", err)
	}
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "bp", FlavorName: "bpocmhuawei005_xm", PalCode: "P4", AppName: "X", StoreID: &xm.ID,
	}); err == nil {
		t.Error("已停用商店不应允许新建渠道")
	}

	// 列表应带回 store 关联（Preload）。
	list, _, err := r.ListChannels(ctx, repo.ChannelFilter{BrandCode: "bp"})
	if err != nil {
		t.Fatalf("列渠道失败: %v", err)
	}
	found := false
	for i := range list {
		if list[i].FlavorName == "bpocmhuawei004_hw" {
			found = true
			if list[i].Store == nil || list[i].Store.Code != "hw" {
				t.Errorf("渠道列表应预加载 store，实际 %+v", list[i].Store)
			}
		}
	}
	if !found {
		t.Fatal("未找到商店渠道")
	}
}

// TestStoreCRUDAndUniqueness 验证应用商店 CRUD、code 唯一性与「被引用不可删除」。
func TestStoreCRUDAndUniqueness(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	st, err := svc.CreateStore(ctx, CreateStoreInput{Code: "HW", Name: "华为", Sort: 1})
	if err != nil {
		t.Fatalf("创建商店失败: %v", err)
	}
	if st.Code != "hw" {
		t.Errorf("code 应规范化为小写，实际 %q", st.Code)
	}

	// 重复 code（大小写不敏感）应被拒。
	if _, err := svc.CreateStore(ctx, CreateStoreInput{Code: "hw", Name: "华为2"}); err == nil {
		t.Error("重复 code 应被拒绝")
	}

	// 非法 code。
	bad := []string{"1hw", "H W", "", "hw-1"}
	for _, c := range bad {
		if _, err := svc.CreateStore(ctx, CreateStoreInput{Code: c, Name: "X"}); err == nil {
			t.Errorf("code %q 应被拒绝", c)
		}
	}

	// 更新 name/sort/status。
	updated, err := svc.UpdateStore(ctx, st.ID, UpdateStoreInput{Name: strPtr("华为应用市场"), Sort: intPtr(5)})
	if err != nil {
		t.Fatalf("更新商店失败: %v", err)
	}
	if updated.Name != "华为应用市场" || updated.Sort != 5 || updated.Code != "hw" {
		t.Errorf("更新结果不符: %+v", updated)
	}

	// 被渠道引用后不可删除。
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018_hw", PalCode: "P1", AppName: "A", StoreID: &st.ID,
	}); err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}
	if err := svc.DeleteStore(ctx, st.ID); err == nil {
		t.Error("已被引用的商店不应可删除")
	}

	// 未被引用的商店可以删除。
	other, err := svc.CreateStore(ctx, CreateStoreInput{Code: "op", Name: "Oppo"})
	if err != nil {
		t.Fatalf("创建商店失败: %v", err)
	}
	if err := svc.DeleteStore(ctx, other.ID); err != nil {
		t.Errorf("未被引用的商店应可删除: %v", err)
	}
}

// TestCreateChannelWithAdjustBinding 验证创建渠道时可一并绑定 Adjust App Token + 事件表，
// 且 CRUD 详情与 CLI 配置下发接口（BuildManifestForBrand）都原样带出（ADR-0013 §2/§5）。
func TestCreateChannelWithAdjustBinding(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode:      "gp",
		FlavorName:     "gpgzmkk042",
		PalCode:        "PAL1",
		AppName:        "GameZone",
		AdjustAppToken: " abc123xyz ", // 前后空白应被 trim
		AdjustEvents: map[string]string{
			"Login":    "wzb3fb",
			"Purchase": "gyuu2f",
		},
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}
	if ch.AdjustAppToken == nil || *ch.AdjustAppToken != "abc123xyz" {
		t.Fatalf("adjustAppToken 应为 abc123xyz（已 trim），实际 %+v", ch.AdjustAppToken)
	}
	if len(ch.AdjustEvents) != 2 || ch.AdjustEvents["Login"] != "wzb3fb" || ch.AdjustEvents["Purchase"] != "gyuu2f" {
		t.Fatalf("adjustEvents 应原样存储，实际 %+v", ch.AdjustEvents)
	}

	// 详情接口（DB 往返后）应仍能读出。
	got, err := svc.GetChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("取详情失败: %v", err)
	}
	if got.AdjustAppToken == nil || *got.AdjustAppToken != "abc123xyz" {
		t.Errorf("详情 adjustAppToken 应为 abc123xyz，实际 %+v", got.AdjustAppToken)
	}
	if len(got.AdjustEvents) != 2 {
		t.Errorf("详情 adjustEvents 应有 2 条，实际 %+v", got.AdjustEvents)
	}

	// CLI 配置下发接口（manifest）应原样带出。
	m, err := svc.BuildManifestForBrand(ctx, "gp")
	if err != nil {
		t.Fatalf("manifest 失败: %v", err)
	}
	if len(m.Channels) != 1 {
		t.Fatalf("应有 1 个渠道，实际 %d", len(m.Channels))
	}
	mc := m.Channels[0]
	if mc.AdjustAppToken != "abc123xyz" {
		t.Errorf("manifest adjustAppToken 应为 abc123xyz，实际 %q", mc.AdjustAppToken)
	}
	if len(mc.AdjustEvents) != 2 || mc.AdjustEvents["Login"] != "wzb3fb" {
		t.Errorf("manifest adjustEvents 应原样带出，实际 %+v", mc.AdjustEvents)
	}
}

// TestCreateChannelWithoutAdjustBinding 验证未绑定 Adjust 是合法状态：token/events 均为空。
func TestCreateChannelWithoutAdjustBinding(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}
	if ch.AdjustAppToken != nil {
		t.Errorf("未绑定时 adjustAppToken 应为空，实际 %+v", ch.AdjustAppToken)
	}
	if len(ch.AdjustEvents) != 0 {
		t.Errorf("未绑定时 adjustEvents 应为空，实际 %+v", ch.AdjustEvents)
	}

	m, err := svc.BuildManifestForBrand(ctx, "ap")
	if err != nil {
		t.Fatalf("manifest 失败: %v", err)
	}
	if len(m.Channels) != 1 || m.Channels[0].AdjustAppToken != "" {
		t.Errorf("未绑定渠道 manifest 的 adjustAppToken 应为空: %+v", m.Channels)
	}
}

// TestUpdateChannelAdjustBindAndUnbind 验证更新可绑定/解绑 Adjust；未传字段不应误改动（部分更新语义）。
func TestUpdateChannelAdjustBindAndUnbind(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	})
	if err != nil {
		t.Fatalf("创建渠道失败: %v", err)
	}

	// 绑定。
	token := "tok123"
	events := map[string]string{"Login": "wzb3fb"}
	updated, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{
		AdjustAppToken: &token,
		AdjustEvents:   &events,
	})
	if err != nil {
		t.Fatalf("绑定 Adjust 失败: %v", err)
	}
	if updated.AdjustAppToken == nil || *updated.AdjustAppToken != token {
		t.Fatalf("绑定后 adjustAppToken 应为 %q，实际 %+v", token, updated.AdjustAppToken)
	}
	if len(updated.AdjustEvents) != 1 {
		t.Fatalf("绑定后 adjustEvents 应有 1 条，实际 %+v", updated.AdjustEvents)
	}

	// 只改 remark，不传 adjust 字段：不应影响已绑定的 Adjust 配置。
	remark := "改个备注"
	untouched, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{Remark: &remark})
	if err != nil {
		t.Fatalf("更新备注失败: %v", err)
	}
	if untouched.AdjustAppToken == nil || *untouched.AdjustAppToken != token {
		t.Errorf("未传 adjust 字段时不应清空已绑定 token，实际 %+v", untouched.AdjustAppToken)
	}
	if len(untouched.AdjustEvents) != 1 {
		t.Errorf("未传 adjust 字段时不应清空已绑定事件表，实际 %+v", untouched.AdjustEvents)
	}

	// 显式传空字符串 / 空对象 → 解绑。
	empty := ""
	emptyEvents := map[string]string{}
	unbound, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{
		AdjustAppToken: &empty,
		AdjustEvents:   &emptyEvents,
	})
	if err != nil {
		t.Fatalf("解绑 Adjust 失败: %v", err)
	}
	if unbound.AdjustAppToken != nil {
		t.Errorf("解绑后 adjustAppToken 应为空，实际 %+v", unbound.AdjustAppToken)
	}
	if len(unbound.AdjustEvents) != 0 {
		t.Errorf("解绑后 adjustEvents 应为空，实际 %+v", unbound.AdjustEvents)
	}
}

// TestAdjustEventsValidationRejectsBadShape 验证 adjustEvents 的形状校验：
// 空 key / 空 value 均应被拒绝（要么整体为空，要么是干净的 string→string 对象）。
func TestAdjustEventsValidationRejectsBadShape(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	cases := []map[string]string{
		{"": "wzb3fb"},  // 空事件名
		{"Login": ""},   // 空 token
		{" ": "wzb3fb"}, // 空白事件名
	}
	for i, events := range cases {
		_, err := svc.CreateChannel(ctx, CreateChannelInput{
			BrandCode: "ap", FlavorName: "ap0101" + string(rune('a'+i)), PalCode: "P", AppName: "A",
			AdjustEvents: events,
		})
		if err == nil {
			t.Errorf("用例 %d（events=%v）应被拒绝", i, events)
		}
	}
}

// TestAdjustAppTokenTooLong 验证 adjustAppToken 超过列宽 VARCHAR(64) 应被拒绝。
func TestAdjustAppTokenTooLong(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	tooLong := strings.Repeat("a", 65)
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "P", AppName: "A",
		AdjustAppToken: tooLong,
	}); err == nil {
		t.Error("超长 adjustAppToken 应被拒绝")
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// sanitizeDBName 把测试名转成合法的内存库名（去掉 / 等）。
func sanitizeDBName(s string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(s)
}

func flavorNames(list []model.Channel) []string {
	out := make([]string, len(list))
	for i := range list {
		out[i] = list[i].FlavorName
	}
	return out
}
