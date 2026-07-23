package service

import (
	"context"
	"net"
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

// fakeResolver 是可控的 IP→国家解析器，供编排测试注入。
type fakeResolver struct{ table map[string]string }

func (f fakeResolver) Country(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	c, ok := f.table[ip.String()]
	return c, ok
}

// newListingTestService 在标准测试 Service 上注入一个可控 resolver。
func newListingTestService(t *testing.T, table map[string]string) (*Service, context.Context) {
	t.Helper()
	svc, _ := newTestService(t)
	svc.SetCountryResolver(fakeResolver{table: table})
	return svc, context.Background()
}

// mustCreateListing 建一个 android 上架包，挂在 ap 品牌下（其种子域名 https://arenaplus.ph），
// 参考渠道包 PAL_CODE 手填一个合法纯数字串。
func mustCreateListing(t *testing.T, svc *Service, ctx context.Context) *model.ListingApp {
	t.Helper()
	l, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap",
		Platform:  "android",
		BundleID:  "com.vividnest.colorstack5821",
		Name:      "ColorStack",
		PalCode:   "1053259",
	})
	if err != nil {
		t.Fatalf("创建上架包失败: %v", err)
	}
	return l
}

func TestCreateListingDefaults(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	if l.GateEnabled {
		t.Error("新建上架包应默认 gate_enabled=false（只有 A 面）")
	}
	if !l.UseBrandDomains {
		t.Error("新建上架包应默认继承品牌域名")
	}
	if l.Status != model.ListingEnabled {
		t.Errorf("默认状态应为 enabled，实际 %s", l.Status)
	}
}

func TestCreateListingRejectsDuplicatePlatformBundle(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	mustCreateListing(t, svc, ctx)

	// 同平台同包名 → 冲突。
	_, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.vividnest.colorstack5821", Name: "dup", PalCode: "1053259",
	})
	if err == nil {
		t.Fatal("同 (platform,bundleId) 应冲突")
	}

	// 同包名但不同平台 → 允许（Flutter 双端共用包名，见 model 唯一键设计）。
	_, err = svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "ios", BundleID: "com.vividnest.colorstack5821", Name: "ios", PalCode: "1053259",
	})
	if err != nil {
		t.Fatalf("不同平台同包名应允许，却失败: %v", err)
	}
}

// 空 Adjust token 应规范化为 NULL（nil），而非落库成字面空串 ""。
// 下游按 IS NULL 判「是否绑定 Adjust」的消费者依赖这个口径。
func TestCreateListingEmptyAdjustTokenNormalizedToNil(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	empty := ""
	l, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.x.empty",
		Name: "x", AdjustAppToken: &empty, PalCode: "1053259",
	})
	if err != nil {
		t.Fatal(err)
	}
	if l.AdjustAppToken != nil {
		t.Errorf("空 token 应规范化为 nil(NULL)，实际 %q", *l.AdjustAppToken)
	}

	// 更新为纯空白也应变 nil。
	blank := "   "
	updated, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{AdjustAppToken: &blank})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AdjustAppToken != nil {
		t.Errorf("纯空白 token 应规范化为 nil，实际 %q", *updated.AdjustAppToken)
	}

	// 非空 token 应保留（去空白）。
	tok := "  abc123  "
	got, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{AdjustAppToken: &tok})
	if err != nil {
		t.Fatal(err)
	}
	if got.AdjustAppToken == nil || *got.AdjustAppToken != "abc123" {
		t.Errorf("非空 token 应保留并去空白为 abc123，实际 %v", got.AdjustAppToken)
	}
}

// 参考渠道包 PAL_CODE：手填，必填、纯数字串；成功后原样落库。
func TestCreateListingPalCodeRules(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)

	// 缺失 → 报错。
	if _, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.x.nopal", Name: "x",
	}); err == nil {
		t.Fatal("未填 PAL_CODE 应报错")
	}

	// 非纯数字 → 报错。
	if _, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.x.badpal", Name: "x", PalCode: "PAL-abc",
	}); err == nil {
		t.Fatal("非纯数字 PAL_CODE 应报错")
	}

	// 合法纯数字（含前后空白）→ 成功，去空白后原样落库。
	l, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.x.okpal", Name: "x", PalCode: "  1053259  ",
	})
	if err != nil {
		t.Fatalf("合法 PAL_CODE 应成功: %v", err)
	}
	if l.PalCode != "1053259" {
		t.Errorf("PalCode 应去空白后为 1053259，实际 %q", l.PalCode)
	}

	// 更新 PAL_CODE 生效。
	np := "2087431"
	got, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{PalCode: &np})
	if err != nil {
		t.Fatal(err)
	}
	if got.PalCode != "2087431" {
		t.Errorf("更新后 PalCode 应为 2087431，实际 %q", got.PalCode)
	}
}

func TestSetListingGateValidation(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	// 国家白名单为空 → 拒绝。
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{}); err == nil {
		t.Error("空国家白名单应被拒绝")
	}

	// 含 US → 拒绝（被硬编码强制 A 面，不允许入白名单）。
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH", "US"}}); err == nil {
		t.Error("国家白名单含 US 应被拒绝")
	}

	// 含 CN → 拒绝。
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"CN"}}); err == nil {
		t.Error("国家白名单含 CN 应被拒绝")
	}

	// 非法 CIDR → 拒绝。
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH"}, IPDenyCIDRs: []string{"乱写"}}); err == nil {
		t.Error("非法 CIDR 应被拒绝")
	}

	// 合法配置 → 成功，且国家码规范化为大写去重。
	gate, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"ph", "PH", "id"}})
	if err != nil {
		t.Fatalf("合法网关配置应成功: %v", err)
	}
	if len(gate.Countries) != 2 {
		t.Errorf("国家码应去重为 2 个，实际 %v", gate.Countries)
	}
}

func TestEnableGateRequiresCountries(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	// 未配规则就想开总开关 → 拒绝。
	on := true
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{GateEnabled: &on}); err == nil {
		t.Error("未配国家白名单时打开 gate_enabled 应被拒绝")
	}

	// 配好规则后再开 → 成功。
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH"}}); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{GateEnabled: &on})
	if err != nil {
		t.Fatalf("配好规则后打开开关应成功: %v", err)
	}
	if !updated.GateEnabled {
		t.Error("gate_enabled 应已打开")
	}
}

// 端到端编排：菲律宾 IP 进 B 面并拿到品牌域名；美国 IP 强制 A 面；总开关关时一律 A。
func TestEvaluateListingGateEndToEnd(t *testing.T) {
	svc, ctx := newListingTestService(t, map[string]string{
		"112.198.0.1": "PH",
		"8.8.8.8":     "US",
		"203.0.113.7": "JP",
	})
	l := mustCreateListing(t, svc, ctx)
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH"}}); err != nil {
		t.Fatal(err)
	}

	req := func(ipStr string) GateRequest {
		return GateRequest{Platform: "android", BundleID: "com.vividnest.colorstack5821", IP: net.ParseIP(ipStr)}
	}

	// 总开关未开：全 A。
	if res, _ := svc.EvaluateListingGate(ctx, req("112.198.0.1")); res.Mode != model.GateModeA {
		t.Errorf("总开关未开时菲律宾 IP 也应 A，实际 %s", res.Mode)
	}

	// 打开总开关。
	on := true
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{GateEnabled: &on}); err != nil {
		t.Fatal(err)
	}

	// 菲律宾 IP → B，且 URL 为继承的品牌域名 + 参考渠道包 palcode。
	res, err := svc.EvaluateListingGate(ctx, req("112.198.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != model.GateModeB {
		t.Fatalf("菲律宾 IP 应进 B 面，实际 %s", res.Mode)
	}
	if res.URL != "https://arenaplus.ph/?palcode=1053259" {
		t.Errorf("B 面 URL 应为品牌域名拼 palcode，实际 %q", res.URL)
	}

	// 美国 IP → 强制 A（即便不在白名单，硬编码闸也拦下）。
	if res, _ := svc.EvaluateListingGate(ctx, req("8.8.8.8")); res.Mode != model.GateModeA {
		t.Errorf("美国 IP 应强制 A，实际 %s", res.Mode)
	}
	// 美国 IP 的响应必须不含 URL（绝不泄露 B 面地址）。
	if res, _ := svc.EvaluateListingGate(ctx, req("8.8.8.8")); res.URL != "" {
		t.Errorf("A 面响应绝不能带 URL，实际 %q", res.URL)
	}

	// 日本 IP（不在白名单）→ A。
	if res, _ := svc.EvaluateListingGate(ctx, req("203.0.113.7")); res.Mode != model.GateModeA {
		t.Errorf("非白名单国家应 A，实际 %s", res.Mode)
	}

	// 未知包 → A。
	if res, _ := svc.EvaluateListingGate(ctx, GateRequest{Platform: "android", BundleID: "com.unknown.pkg", IP: net.ParseIP("112.198.0.1")}); res.Mode != model.GateModeA {
		t.Errorf("未知包应 A，实际 %s", res.Mode)
	}
}

// resolver 未注入时（nil）应安全降级：所有请求判 A 面，不 panic。
func TestEvaluateListingGateNilResolver(t *testing.T) {
	svc, _ := newTestService(t) // 不调用 SetCountryResolver
	ctx := context.Background()
	l := mustCreateListing(t, svc, ctx)
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH"}}); err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{GateEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.EvaluateListingGate(ctx, GateRequest{Platform: "android", BundleID: "com.vividnest.colorstack5821", IP: net.ParseIP("112.198.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != model.GateModeA {
		t.Errorf("无 GeoIP resolver 时应降级为 A 面，实际 %s", res.Mode)
	}
}
