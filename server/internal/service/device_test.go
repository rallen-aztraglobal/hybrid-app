package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// mustCreateDeviceChannel 建一个渠道，返回其 applicationId，供设备上报测试使用。
func mustCreateDeviceChannel(t *testing.T, svc *Service, ctx context.Context, brand, flavor, appName string) string {
	t.Helper()
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: brand, FlavorName: flavor, PalCode: "PAL-" + flavor, AppName: appName,
	})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	return ch.ApplicationID
}

// firstDevice 取某 applicationId 下唯一一条设备记录（测试专用便捷查询）。
func firstDevice(t *testing.T, r *repo.Repo, appID string) DeviceView {
	t.Helper()
	ctx := context.Background()
	list, total, err := r.ListChannelDevices(ctx, repo.DeviceFilter{ApplicationIDs: []string{appID}, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("查询设备失败: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("应恰好有 1 条设备记录，实际 total=%d len=%d", total, len(list))
	}
	return deviceView(list[0])
}

// TestRegisterDeviceUpsertPreservesCreatedAt 验证 upsert 语义：
// 首次上报落库 created_at；同 deviceKey 二次上报（换 adid）created_at 不变、adid 更新。
func TestRegisterDeviceUpsertPreservesCreatedAt(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")

	const deviceKey = "device-uuid-001"
	const gaid = "1a2b3c4d-0000-1111-2222-333344445555"
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: deviceKey, DeviceName: "Pixel 6", GAID: gaid, ADID: "adid-old",
	}); err != nil {
		t.Fatalf("首次上报失败: %v", err)
	}
	first := firstDevice(t, r, appID)
	if first.CreatedAt.IsZero() {
		t.Fatal("首次上报后 created_at 不应为空")
	}
	if first.ADID != "adid-old" {
		t.Errorf("adid 应为 adid-old，实际 %q", first.ADID)
	}

	// 二次上报：同 deviceKey，换 adid。
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: deviceKey, DeviceName: "Pixel 6", GAID: gaid, ADID: "adid-new",
	}); err != nil {
		t.Fatalf("二次上报失败: %v", err)
	}
	second := firstDevice(t, r, appID)
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("created_at 不应因二次上报改变：first=%v second=%v", first.CreatedAt, second.CreatedAt)
	}
	if second.ADID != "adid-new" {
		t.Errorf("adid 应已更新为 adid-new，实际 %q", second.ADID)
	}
	if second.ID != first.ID {
		t.Errorf("应是同一条记录（同 deviceKey upsert），first.ID=%d second.ID=%d", first.ID, second.ID)
	}
}

// TestRegisterDeviceEmptyFieldsDoNotOverwrite 验证 adid/deviceName 来值为空时不覆盖已有非空值。
func TestRegisterDeviceEmptyFieldsDoNotOverwrite(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")

	const deviceKey = "device-uuid-002"
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: deviceKey, DeviceName: "Pixel 6", ADID: "adid-keep", GAID: "gaid-keep", OAID: "oaid-keep",
	}); err != nil {
		t.Fatalf("首次上报失败: %v", err)
	}

	// 二次上报：adid/deviceName 为空值（gaid 必填，保持不变）。
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: deviceKey, GAID: "gaid-keep",
	}); err != nil {
		t.Fatalf("二次上报（空字段）失败: %v", err)
	}

	got := firstDevice(t, r, appID)
	if got.ADID != "adid-keep" {
		t.Errorf("adid 来值为空不应覆盖已有值，实际 %q", got.ADID)
	}
	if got.DeviceName != "Pixel 6" {
		t.Errorf("deviceName 来值为空不应覆盖已有值，实际 %q", got.DeviceName)
	}
	if got.OAID != "oaid-keep" {
		t.Errorf("oaid 来值为空不应覆盖已有值，实际 %q", got.OAID)
	}
}

// TestRegisterDeviceRejectsInvalidGAID 验证无有效 GAID 的上报被拒收：
// 空 GAID 与全 0 GAID（opt-out 占位值）都不落库。
func TestRegisterDeviceRejectsInvalidGAID(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")

	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: "device-uuid-003",
	}); err == nil {
		t.Error("gaid 为空应被拒绝")
	}
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: "device-uuid-004",
		GAID: "00000000-0000-0000-0000-000000000000",
	}); err == nil {
		t.Error("全 0 GAID 应被拒绝")
	}

	var count int64
	if err := r.DB().WithContext(ctx).Model(&model.ChannelDevice{}).Count(&count).Error; err != nil {
		t.Fatalf("查询设备数失败: %v", err)
	}
	if count != 0 {
		t.Errorf("被拒收的上报不应落库，实际 %d 条", count)
	}
}

// TestRegisterDeviceRejectsUnknownAppID 验证不存在的 appId 返回错误。
func TestRegisterDeviceRejectsUnknownAppID(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	err := svc.RegisterDevice(ctx, RegisterDeviceInput{AppID: "com.fake.app", DeviceKey: "k1"})
	if err == nil {
		t.Fatal("未知 applicationId 应被拒绝")
	}
}

// TestRegisterDeviceRejectsEmptyRequired 验证 appId/deviceKey 缺失时被拒绝。
func TestRegisterDeviceRejectsEmptyRequired(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{AppID: "", DeviceKey: "k1"}); err == nil {
		t.Error("appId 为空应被拒绝")
	}
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{AppID: "com.arenaplus.ap01018", DeviceKey: ""}); err == nil {
		t.Error("deviceKey 为空应被拒绝")
	}
}

// TestListDevicesPaginationClamp 验证分页 offset 钳制（page 过深报错）与 pageSize 钳 200。
func TestListDevicesPaginationClamp(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{AppID: appID, DeviceKey: "d1", GAID: "gaid-d1"}); err != nil {
		t.Fatalf("上报失败: %v", err)
	}

	// page 翻太深：offset = (502-1)*200 = 100200 > 100000，应报错。
	if _, err := svc.ListDevices(ctx, ListDevicesInput{Page: 502, PageSize: 200}); err == nil {
		t.Error("offset 过深应被拒绝")
	}

	// pageSize 应被钳到 200：请求 pageSize=1000、page=102 时，
	// 若真的钳到 200，offset=101*200=20200（不报错）；若未钳制仍用 1000，
	// offset=101*1000=101000>100000（应报错）。用这组边界值区分是否发生了钳制。
	if _, err := svc.ListDevices(ctx, ListDevicesInput{Page: 102, PageSize: 1000}); err != nil {
		t.Errorf("pageSize 应被钳到 200，不应因翻页过深报错: %v", err)
	}

	// 常规查询应正常返回（不报错）。
	res, err := svc.ListDevices(ctx, ListDevicesInput{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("常规分页查询失败: %v", err)
	}
	if res.Total != 1 || len(res.Items) != 1 {
		t.Errorf("应有 1 条记录，实际 total=%d len=%d", res.Total, len(res.Items))
	}
}

// TestListDevicesInvalidDateFormat 验证非法 from/to 日期格式返回错误。
func TestListDevicesInvalidDateFormat(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.ListDevices(ctx, ListDevicesInput{From: "2026/01/01"}); err == nil {
		t.Error("非法 from 格式应被拒绝")
	}
	if _, err := svc.ListDevices(ctx, ListDevicesInput{To: "not-a-date"}); err == nil {
		t.Error("非法 to 格式应被拒绝")
	}
}

// TestExportDevicesByIDsRejectsTooMany 验证 ids 数量超过 1000 应被拒绝，空 ids 也应被拒绝。
func TestExportDevicesByIDsRejectsTooMany(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	var buf bytes.Buffer
	if err := svc.ExportDevicesCSVByIDs(ctx, &buf, auth.FullScope(), nil, nil); err == nil {
		t.Error("空 ids 应被拒绝")
	}

	tooMany := make([]uint64, DeviceExportMaxIDs+1)
	for i := range tooMany {
		tooMany[i] = uint64(i + 1)
	}
	if err := svc.ExportDevicesCSVByIDs(ctx, &buf, auth.FullScope(), tooMany, nil); err == nil {
		t.Error("超过 1000 个 ids 应被拒绝")
	}

	okIDs := make([]uint64, DeviceExportMaxIDs)
	for i := range okIDs {
		okIDs[i] = uint64(i + 1)
	}
	if err := svc.ExportDevicesCSVByIDs(ctx, &buf, auth.FullScope(), okIDs, nil); err != nil {
		t.Errorf("恰好 1000 个 ids 不应被拒绝（即便查不到数据也应正常导出空表）: %v", err)
	}
}

// TestExportDevicesCSVFormat 验证 CSV 输出格式：BOM、表头、gaid_sha256 与手算一致。
func TestExportDevicesCSVFormat(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")

	const gaid = "1a2b3c4d-0000-1111-2222-333344445555"
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: "dev-with-gaid", DeviceName: "Pixel 7", GAID: gaid, ADID: "ad-1", OAID: "oa-1",
	}); err != nil {
		t.Fatalf("上报失败: %v", err)
	}

	var buf bytes.Buffer
	if err := svc.ExportDevicesCSV(ctx, &buf, ListDevicesInput{ApplicationIDs: []string{appID}}, nil); err != nil {
		t.Fatalf("导出 CSV 失败: %v", err)
	}

	raw := buf.Bytes()
	if len(raw) < 3 || raw[0] != 0xEF || raw[1] != 0xBB || raw[2] != 0xBF {
		t.Fatalf("CSV 前 3 字节应为 UTF-8 BOM，实际 % x", raw[:min3(len(raw), 3)])
	}

	cr := csv.NewReader(strings.NewReader(string(raw[3:])))
	records, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	if len(records) != 2 { // 表头 + 1 行
		t.Fatalf("应有 2 行（表头+1），实际 %d: %v", len(records), records)
	}
	wantHeader := []string{"device_name", "gaid", "gaid_sha256", "adid", "oaid", "app_name", "palcode", "application_id", "brand", "created_at", "last_active_at"}
	if !equalStrSlice(records[0], wantHeader) {
		t.Fatalf("表头不符，实际 %v，期望 %v", records[0], wantHeader)
	}

	sum := sha256.Sum256([]byte(gaid))
	wantSha := hex.EncodeToString(sum[:])

	row := records[1]
	if row[0] != "Pixel 7" {
		t.Fatalf("设备行不符，实际: %v", row)
	}
	if row[1] != gaid {
		t.Errorf("gaid 列应为 %q，实际 %q", gaid, row[1])
	}
	if row[2] != wantSha {
		t.Errorf("gaid_sha256 应为 %q，实际 %q", wantSha, row[2])
	}
}

// TestListDevicesMultiFilter 验证新筛选维度：渠道多选（applicationIds IN）、
// 设备关键字（设备名/GAID/ADID/deviceKey 模糊）、包名关键字（applicationId 模糊），三者 AND 叠加。
func TestListDevicesMultiFilter(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	app1 := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")
	app2 := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01036", "ArenaPlus")
	app3 := mustCreateDeviceChannel(t, svc, ctx, "gp", "gz001", "GameZone")

	seed := []struct {
		appID, key, name, gaid string
	}{
		{app1, "key-a1", "Pixel 7", "aaaa1111-0000-1111-2222-333344445555"},
		{app2, "key-a2", "OPPO CPH2819", "bbbb2222-0000-1111-2222-333344445555"},
		{app3, "key-g1", "Redmi Note 12", "cccc3333-0000-1111-2222-333344445555"},
	}
	for _, d := range seed {
		if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
			AppID: d.appID, DeviceKey: d.key, DeviceName: d.name, GAID: d.gaid,
		}); err != nil {
			t.Fatalf("上报 %s 失败: %v", d.key, err)
		}
	}

	// 渠道多选：app1 + app2 应命中 2 条，不含 app3。
	got, err := svc.ListDevices(ctx, ListDevicesInput{ApplicationIDs: []string{app1, app2}})
	if err != nil {
		t.Fatalf("渠道多选查询失败: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("渠道多选应命中 2 条，实际 %d", got.Total)
	}
	for _, it := range got.Items {
		if it.ApplicationID == app3 {
			t.Errorf("渠道多选不应包含 %s", app3)
		}
	}

	// 设备关键字：按设备名模糊。
	got, err = svc.ListDevices(ctx, ListDevicesInput{DeviceKw: "cph2819"})
	if err != nil {
		t.Fatalf("设备关键字查询失败: %v", err)
	}
	if got.Total != 1 || got.Items[0].DeviceName != "OPPO CPH2819" {
		t.Errorf("设备名关键字应恰好命中 OPPO CPH2819，实际 total=%d", got.Total)
	}

	// 设备关键字：按 GAID 片段模糊。
	got, err = svc.ListDevices(ctx, ListDevicesInput{DeviceKw: "cccc3333"})
	if err != nil {
		t.Fatalf("GAID 关键字查询失败: %v", err)
	}
	if got.Total != 1 || got.Items[0].ApplicationID != app3 {
		t.Errorf("GAID 关键字应恰好命中 app3 的设备，实际 total=%d", got.Total)
	}

	// 包名关键字：模糊命中 ap01018/ap01036 两条。
	got, err = svc.ListDevices(ctx, ListDevicesInput{PackageKw: "ap010"})
	if err != nil {
		t.Fatalf("包名关键字查询失败: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("包名关键字 ap010 应命中 2 条，实际 %d", got.Total)
	}

	// AND 叠加：渠道多选 + 设备关键字取交集。
	got, err = svc.ListDevices(ctx, ListDevicesInput{ApplicationIDs: []string{app1, app2}, DeviceKw: "pixel"})
	if err != nil {
		t.Fatalf("叠加筛选查询失败: %v", err)
	}
	if got.Total != 1 || got.Items[0].ApplicationID != app1 {
		t.Errorf("叠加筛选应恰好命中 app1 的 Pixel 7，实际 total=%d", got.Total)
	}
}

// TestListDevicesActiveTimeFilter 验证最后活跃时间（updated_at）筛选：
// 刚上报的记录活跃时间为「今天」，activeFrom=今天 应命中、activeFrom=明天 应为空、
// activeTo=昨天 应为空；updatedAt 也应随记录返回（非零值）。
func TestListDevicesActiveTimeFilter(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	appID := mustCreateDeviceChannel(t, svc, ctx, "ap", "ap01018", "ArenaPlus")
	if err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		AppID: appID, DeviceKey: "key-active", DeviceName: "Pixel 7",
		GAID: "dddd4444-0000-1111-2222-333344445555",
	}); err != nil {
		t.Fatalf("上报失败: %v", err)
	}

	day := func(offset int) string { return time.Now().AddDate(0, 0, offset).Format(deviceDateLayout) }

	got, err := svc.ListDevices(ctx, ListDevicesInput{ActiveFrom: day(0)})
	if err != nil {
		t.Fatalf("activeFrom=今天 查询失败: %v", err)
	}
	if got.Total != 1 {
		t.Errorf("activeFrom=今天 应命中 1 条，实际 %d", got.Total)
	}
	if len(got.Items) == 1 && got.Items[0].UpdatedAt.IsZero() {
		t.Error("updatedAt 应随记录返回，实际为零值")
	}

	if got, err = svc.ListDevices(ctx, ListDevicesInput{ActiveFrom: day(1)}); err != nil {
		t.Fatalf("activeFrom=明天 查询失败: %v", err)
	} else if got.Total != 0 {
		t.Errorf("activeFrom=明天 应为空，实际 %d", got.Total)
	}

	if got, err = svc.ListDevices(ctx, ListDevicesInput{ActiveTo: day(-1)}); err != nil {
		t.Fatalf("activeTo=昨天 查询失败: %v", err)
	} else if got.Total != 0 {
		t.Errorf("activeTo=昨天 应为空，实际 %d", got.Total)
	}

	if _, err = svc.ListDevices(ctx, ListDevicesInput{ActiveFrom: "bad-date"}); err == nil {
		t.Error("activeFrom 非法格式应报错")
	}
}

func min3(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func equalStrSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
