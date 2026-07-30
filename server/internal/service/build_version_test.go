package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

// mustSucceedBuild 创建一条构建任务并直接跑到 success 终态，返回该记录（供测试摆好「当前版本」基线）。
func mustSucceedBuild(t *testing.T, svc *Service, brand, flavor, versionName string) *model.BuildRecord {
	t.Helper()
	ctx := context.Background()
	rec, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: brand, Flavors: []string{flavor}, VersionName: versionName,
	})
	if err != nil {
		t.Fatalf("创建构建任务(version=%s)失败: %v", versionName, err)
	}
	if _, err := svc.ClaimBuild(ctx, "runner"); err != nil {
		t.Fatalf("领取失败: %v", err)
	}
	if _, err := svc.ReportBuildStatus(ctx, rec.ID, ReportBuildStatusInput{Status: model.BuildSuccess}); err != nil {
		t.Fatalf("上报成功失败: %v", err)
	}
	got, err := svc.GetBuildRecord(ctx, rec.ID)
	if err != nil {
		t.Fatalf("回读构建记录失败: %v", err)
	}
	return got
}

// TestVersionValidation_NoPriorBuild 验证：品牌尚无成功构建时，任意合法版本都应放行。
func TestVersionValidation_NoPriorBuild(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.0.0",
	}); err != nil {
		t.Errorf("无历史成功构建时，1.0.0 应被放行，实际报错: %v", err)
	}
}

// TestVersionValidation_RejectLowerAllowEqualAndHigher 验证核心规则：
// 当前版本 1.3.8 时，低于它的一律拒绝；等于或高于它的一律放行（数值比较，非字符串比较）。
func TestVersionValidation_RejectLowerAllowEqualAndHigher(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	mustSucceedBuild(t, svc, "ap", "ap01018", "1.3.8")

	rejected := []string{"1.3.7", "1.2.99", "0.99.99"}
	for _, v := range rejected {
		if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
			Brand: "ap", Flavors: []string{"ap01018"}, VersionName: v,
		}); err == nil {
			t.Errorf("versionName=%s 应被拒绝（低于当前版本 1.3.8）", v)
		}
	}

	allowed := []string{"1.3.8", "1.3.9", "1.4.0", "2.0.0"}
	for _, v := range allowed {
		if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
			Brand: "ap", Flavors: []string{"ap01018"}, VersionName: v,
		}); err != nil {
			t.Errorf("versionName=%s 应被放行（等于/高于当前版本 1.3.8），实际报错: %v", v, err)
		}
	}
}

// TestVersionValidation_NumericNotLexicographic 验证数值比较：1.10.0 > 1.2.0（字符串比较会得出相反结论）。
func TestVersionValidation_NumericNotLexicographic(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	mustSucceedBuild(t, svc, "ap", "ap01018", "1.2.0")

	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.10.0",
	}); err != nil {
		t.Errorf("1.10.0 应大于 1.2.0 而被放行（数值比较），实际报错: %v", err)
	}
}

// TestVersionValidation_HighestNotNewestRow 验证「当前版本」取的是历史成功构建里语义版本最高的一条，
// 而不是按时间/自增 id 最新的一条。校验一旦生效，正常流程不可能再对同一品牌先成功一个高版本、
// 后成功一个更低版本（后者会被本校验直接拒绝）——「时间更晚但版本更低」只会出现在校验上线前的
// 历史脏数据里，因此这里绕过 service、直接用 repo 插入该历史记录来模拟这一场景。
func TestVersionValidation_HighestNotNewestRow(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	mustSucceedBuild(t, svc, "ap", "ap01018", "2.0.0")

	// 历史脏数据：时间更晚，但版本号更低（校验上线前遗留，绕过 service 直接写库模拟）。
	older := &model.BuildRecord{
		BrandCode: "ap", Flavors: `["ap01018"]`, Status: model.BuildSuccess,
		VersionName: "1.9.0", ApkURLs: "[]",
	}
	if err := r.CreateBuildRecord(ctx, older); err != nil {
		t.Fatalf("插入历史脏数据失败: %v", err)
	}

	current, found := svc.CurrentVersion(ctx, "ap")
	if !found || current != "2.0.0" {
		t.Fatalf("当前版本应仍是语义版本最高的 2.0.0，实际 found=%v current=%q", found, current)
	}

	// 提交一个高于「按时间最新一条」(1.9.0) 但低于真正当前版本(2.0.0)的版本，必须被拒绝。
	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.9.5",
	}); err == nil {
		t.Error("1.9.5 低于真正当前版本 2.0.0，应被拒绝（若实现误取「最新一条」1.9.0 则会被错误放行）")
	}
}

// TestVersionValidation_IgnoresInvalidLegacyVersion 验证：历史脏数据里存在无法解析的 versionName 时，
// CurrentVersion 不应 panic，且应忽略脏数据、仍能从合法版本中选出正确的当前版本。
func TestVersionValidation_IgnoresInvalidLegacyVersion(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	// 绕过 service 校验，直接在 repo 层插入一条历史脏数据（非法 versionName 的 success 记录）。
	dirty := &model.BuildRecord{
		BrandCode: "ap", Flavors: `["ap01018"]`, Status: model.BuildSuccess,
		VersionName: "not-a-version", ApkURLs: "[]",
	}
	if err := r.CreateBuildRecord(ctx, dirty); err != nil {
		t.Fatalf("插入历史脏数据失败: %v", err)
	}

	mustSucceedBuild(t, svc, "ap", "ap01018", "1.5.0")

	current, found := svc.CurrentVersion(ctx, "ap")
	if !found {
		t.Fatal("应能从合法版本中找出当前版本（不应因脏数据而 panic 或整体失败）")
	}
	if current != "1.5.0" {
		t.Errorf("当前版本应为合法版本里最高的 1.5.0，实际 %q", current)
	}

	// 提交更低版本仍应被拒绝，证明脏数据没有破坏正常校验路径。
	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.0.0",
	}); err == nil {
		t.Error("1.0.0 低于当前版本 1.5.0，应被拒绝")
	}
}

// TestVersionValidation_CurrentVersionScansFullHistoryNotJustRecentPage 验证 CurrentVersion
// （因而 GetCurrentVersion 端点与 CreateBuildJob 的强制校验）扫描的是全部 success 记录，
// 不受 ListBuildRecords 分页/上限（默认 50、上限 200）影响：最早一条就是全局最高版本，
// 之后插入远超 50 条更新但版本更低的记录，当前版本必须仍是最早那条。
func TestVersionValidation_CurrentVersionScansFullHistoryNotJustRecentPage(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	// 最早一条（绕过 service 直接写库，模拟早期数据）就是全局最高版本。
	oldest := &model.BuildRecord{
		BrandCode: "ap", Flavors: `["ap01018"]`, Status: model.BuildSuccess,
		VersionName: "9.9.9", ApkURLs: "[]",
	}
	if err := r.CreateBuildRecord(ctx, oldest); err != nil {
		t.Fatalf("插入最早记录失败: %v", err)
	}

	// 之后插入 60 条更新、但版本更低的记录（超过 /build/records 默认分页 50 的量级）。
	for i := 0; i < 60; i++ {
		rec := &model.BuildRecord{
			BrandCode: "ap", Flavors: `["ap01018"]`, Status: model.BuildSuccess,
			VersionName: fmt.Sprintf("1.0.%d", i), ApkURLs: "[]",
		}
		if err := r.CreateBuildRecord(ctx, rec); err != nil {
			t.Fatalf("插入第 %d 条记录失败: %v", i, err)
		}
	}

	current, found := svc.CurrentVersion(ctx, "ap")
	if !found || current != "9.9.9" {
		t.Fatalf("当前版本应仍是最早那条 9.9.9（不受任何分页/上限影响），实际 found=%v current=%q", found, current)
	}

	// 与强制校验联动：提交一个高于「最近 60 条里最高」(1.0.59) 但低于真正当前版本(9.9.9)
	// 的版本，必须被拒绝——证明 CreateBuildJob 用的是同一份不受分页影响的结果。
	if _, err := svc.CreateBuildJob(ctx, CreateBuildJobInput{
		Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "2.0.0",
	}); err == nil {
		t.Error("2.0.0 低于真正当前版本 9.9.9，应被拒绝")
	}
}
