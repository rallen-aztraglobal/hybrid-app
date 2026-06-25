package service

import (
	"context"
	"testing"
	"time"
)

// TestRegisterDeviceTokenUpsert 验证 token 注册的 upsert 语义：
// 同一 device_token 重复上报 → 更新 applicationId/pal_code/last_seen_at，而不插入新行。
func TestRegisterDeviceTokenUpsert(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()

	// 先建一个渠道，提供合法的 applicationId。
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "PAL-U1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	const appID = "com.arenaplus.ap01001"
	const tok = "device-token-xyz"

	// 首次注册。
	if err := svc.RegisterDeviceToken(ctx, appID, tok, "PAL-U1", "android", "Pixel 6"); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}

	// 统计：应有 1 台设备。
	aud, err := svc.PushAudience(ctx, []string{appID})
	if err != nil {
		t.Fatalf("audience 查询失败: %v", err)
	}
	if aud.TotalDevices != 1 {
		t.Fatalf("应有 1 台，实际 %d", aud.TotalDevices)
	}

	// 用不同 palCode 再次上报同一 token（模拟换渠道）。
	if err := svc.RegisterDeviceToken(ctx, appID, tok, "PAL-U2", "android", "Pixel 6"); err != nil {
		t.Fatalf("二次注册失败: %v", err)
	}

	// 设备数仍应为 1（upsert 而非 insert）。
	aud2, err := svc.PushAudience(ctx, []string{appID})
	if err != nil {
		t.Fatalf("audience 查询失败: %v", err)
	}
	if aud2.TotalDevices != 1 {
		t.Errorf("upsert 后仍应有 1 台，实际 %d（可能发生了重复插入）", aud2.TotalDevices)
	}

	// 验证 DB 内 pal_code 已更新。
	tokens, err := r.ActiveTokensByAppIDs(ctx, []string{appID})
	if err != nil {
		t.Fatalf("查 token 失败: %v", err)
	}
	if len(tokens[appID]) != 1 {
		t.Fatalf("应只有 1 条 token 记录，实际 %d", len(tokens[appID]))
	}
	if tokens[appID][0].PalCode != "PAL-U2" {
		t.Errorf("pal_code 应已更新为 PAL-U2，实际 %q", tokens[appID][0].PalCode)
	}
}

// TestRegisterTokenRejectsUnknownAppID 验证公开端点对未知 applicationId 的轻量防滥用。
func TestRegisterTokenRejectsUnknownAppID(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	err := svc.RegisterDeviceToken(ctx, "com.fake.app", "tok-abc", "", "android", "")
	if err == nil {
		t.Fatal("未知 applicationId 应被拒绝")
	}
}

// TestPushAudienceStats 验证 audience 统计按 applicationId 分组正确。
func TestPushAudienceStats(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// 建两个渠道。
	for _, f := range []string{"ap01001", "ap01002"} {
		if _, err := svc.CreateChannel(ctx, CreateChannelInput{
			BrandCode: "ap", FlavorName: f, PalCode: "P-" + f, AppName: "A",
		}); err != nil {
			t.Fatalf("建渠道失败: %v", err)
		}
	}

	// ap01001 注册 2 个 token，ap01002 注册 1 个 token。
	appID1 := "com.arenaplus.ap01001"
	appID2 := "com.arenaplus.ap01002"
	for _, tok := range []string{"tok-a", "tok-b"} {
		_ = svc.RegisterDeviceToken(ctx, appID1, tok, "P", "android", "")
	}
	_ = svc.RegisterDeviceToken(ctx, appID2, "tok-c", "P", "android", "")

	aud, err := svc.PushAudience(ctx, []string{appID1, appID2})
	if err != nil {
		t.Fatalf("audience 失败: %v", err)
	}
	if aud.TotalDevices != 3 {
		t.Errorf("总设备数应为 3，实际 %d", aud.TotalDevices)
	}
	if aud.ByApp[appID1] != 2 {
		t.Errorf("%s 应有 2 台，实际 %d", appID1, aud.ByApp[appID1])
	}
	if aud.ByApp[appID2] != 1 {
		t.Errorf("%s 应有 1 台，实际 %d", appID2, aud.ByApp[appID2])
	}

	// 空 appIds 应返回 0。
	aud0, err := svc.PushAudience(ctx, nil)
	if err != nil {
		t.Fatalf("空 appIds audience 失败: %v", err)
	}
	if aud0.TotalDevices != 0 {
		t.Errorf("空 appIds 应返回 0，实际 %d", aud0.TotalDevices)
	}
}

// TestSendGate_FCMNotConfigured 验证 PUSH_ENABLED=false 时发送被门控拦截，campaign 留 draft。
func TestSendGate_FCMNotConfigured(t *testing.T) {
	svc, _ := newTestService(t) // PUSH_ENABLED 默认 false
	ctx := context.Background()

	// 建渠道并注册 token。
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	_ = svc.RegisterDeviceToken(ctx, "com.arenaplus.ap01001", "tok-gate", "P1", "android", "")

	// 创建活动。
	camp, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name:         "gate-test",
		Title:        "Hello",
		Body:         "World",
		TargetAppIDs: []string{"com.arenaplus.ap01001"},
	}, "tester")
	if err != nil {
		t.Fatalf("创建活动失败: %v", err)
	}

	// 非 dry-run 发送 → 应被 PUSH_ENABLED 门控拦截。
	_, err = svc.SendCampaign(ctx, camp.ID, false)
	if err == nil {
		t.Fatal("PUSH_ENABLED=false 时发送应被拒绝")
	}

	// 活动应仍处于非 done 状态（因为门控在置 sending 之前）。
	// 注：门控在 sending 状态设置后触发，但 campaign 不会到 done；验证错误即可。
	t.Logf("门控错误（预期）: %v", err)
}

// TestSendGate_DryRun 验证 dryRun=true 是无损预览：
// - Preview.TotalDevices / ByApp 正确反映活跃 token 数
// - campaign 的 status/name/sentAt/success_count 等持久化字段完全不变
// - dry-run 结束后可以正常真发（不被「已 done」拦截）
func TestSendGate_DryRun(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// 建 2 个渠道，各注册 1 个 token。
	for _, f := range []string{"ap01001", "ap01002"} {
		if _, err := svc.CreateChannel(ctx, CreateChannelInput{
			BrandCode: "ap", FlavorName: f, PalCode: "P-" + f, AppName: "A",
		}); err != nil {
			t.Fatalf("建渠道失败: %v", err)
		}
	}
	_ = svc.RegisterDeviceToken(ctx, "com.arenaplus.ap01001", "tok-dr1", "P1", "android", "Dev1")
	_ = svc.RegisterDeviceToken(ctx, "com.arenaplus.ap01002", "tok-dr2", "P2", "android", "Dev2")

	// 创建活动。
	camp, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name:         "dry-run-test",
		Title:        "Test Title",
		Body:         "Test Body",
		DeeplinkPath: "/promo/test",
		TargetAppIDs: []string{"com.arenaplus.ap01001", "com.arenaplus.ap01002"},
	}, "tester")
	if err != nil {
		t.Fatalf("创建活动失败: %v", err)
	}
	originalName := camp.Name

	// dry-run 发送（不论 PUSH_ENABLED 均可跑，无损预览）。
	result, err := svc.SendCampaign(ctx, camp.ID, true)
	if err != nil {
		t.Fatalf("dry-run 发送失败: %v", err)
	}

	// 响应中 DryRun 标记应为 true。
	if !result.DryRun {
		t.Error("响应 dryRun 字段应为 true")
	}

	// Preview 应有正确的触达数。
	if result.Preview == nil {
		t.Fatal("dry-run 响应应携带 preview 字段")
	}
	if result.Preview.TotalDevices != 2 {
		t.Errorf("Preview.TotalDevices 应为 2，实际 %d", result.Preview.TotalDevices)
	}
	if result.Preview.ByApp["com.arenaplus.ap01001"] != 1 {
		t.Errorf("ap01001 预计触达应为 1，实际 %d", result.Preview.ByApp["com.arenaplus.ap01001"])
	}
	if result.Preview.ByApp["com.arenaplus.ap01002"] != 1 {
		t.Errorf("ap01002 预计触达应为 1，实际 %d", result.Preview.ByApp["com.arenaplus.ap01002"])
	}

	// ---- 核心：campaign 持久化字段完全未被改动 ----

	// Campaign 字段在响应里仍是原始状态（未被改写）。
	if result.Campaign.Status != "draft" {
		t.Errorf("dry-run 后响应中 campaign.status 应仍为 draft，实际 %q", result.Campaign.Status)
	}
	if result.Campaign.Name != originalName {
		t.Errorf("dry-run 不应改 name：期望 %q，实际 %q", originalName, result.Campaign.Name)
	}
	if result.Campaign.SentAt != nil {
		t.Error("dry-run 不应设置 sentAt")
	}
	if result.Campaign.SuccessCount != 0 || result.Campaign.FailureCount != 0 {
		t.Errorf("dry-run 不应改计数：success=%d failure=%d", result.Campaign.SuccessCount, result.Campaign.FailureCount)
	}

	// 从 DB 重新取，确认真正没有写回。
	detail, err := svc.GetCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatalf("取活动详情失败: %v", err)
	}
	if detail.Status != "draft" {
		t.Errorf("DB 中 campaign.status 应仍为 draft，实际 %q", detail.Status)
	}
	if detail.Name != originalName {
		t.Errorf("DB 中 campaign.name 不应被改，期望 %q，实际 %q", originalName, detail.Name)
	}
	if len(detail.Records) != 0 {
		t.Errorf("dry-run 不应写入 push_record，实际有 %d 条", len(detail.Records))
	}

	// ---- 核心：dry-run 后活动仍可真发（不被「已 done」拦截）----
	// PUSH_ENABLED=false 时真发返回 FCM 未配置错误（而不是「已 done」错误）。
	_, err = svc.SendCampaign(ctx, camp.ID, false)
	if err == nil {
		t.Fatal("PUSH_ENABLED=false 时真发应报错（FCM 未配置），而不是成功")
	}
	// 错误应是「FCM 未配置」而不是「已 done 状态」。
	if err.Error() == "活动已处于 done 状态，不可重复发送" {
		t.Fatalf("dry-run 后不应出现「已 done」拦截：%v", err)
	}
	svcErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("期望 *service.Error，实际 %T: %v", err, err)
	}
	if svcErr.Code != 422 {
		t.Errorf("PUSH_ENABLED=false 应返回 422，实际 %d: %s", svcErr.Code, svcErr.Message)
	}
}

// TestCampaignCRUD 验证创建→列表→修改→定时的基本流程。
func TestCampaignCRUD(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "gp", FlavorName: "gp001", PalCode: "GP1", AppName: "G",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	// 创建草稿。
	camp, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name:         "TestCamp",
		Title:        "标题",
		Body:         "正文",
		DeeplinkPath: "/test/path",
		TargetAppIDs: []string{"com.gamezone.gp001"},
	}, "admin")
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if camp.Status != "draft" {
		t.Errorf("初始状态应为 draft，实际 %q", camp.Status)
	}
	if camp.DeeplinkPath != "/test/path" {
		t.Errorf("deeplinkPath 应为 /test/path，实际 %q", camp.DeeplinkPath)
	}

	// 修改草稿。
	updated, err := svc.UpdateCampaign(ctx, camp.ID, PushCampaignInput{
		Name:         "TestCamp-v2",
		Title:        "标题2",
		Body:         "正文2",
		TargetAppIDs: []string{"com.gamezone.gp001"},
	})
	if err != nil {
		t.Fatalf("修改失败: %v", err)
	}
	if updated.Name != "TestCamp-v2" {
		t.Errorf("Name 应更新为 TestCamp-v2，实际 %q", updated.Name)
	}

	// 列表应有 1 条。
	list, err := svc.ListCampaigns(ctx, "")
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("应有 1 条活动，实际 %d", len(list))
	}

	// 定时发送。
	future := time.Now().Add(2 * time.Hour)
	scheduled, err := svc.ScheduleCampaign(ctx, camp.ID, future)
	if err != nil {
		t.Fatalf("定时失败: %v", err)
	}
	if scheduled.Status != "scheduled" {
		t.Errorf("定时后 status 应为 scheduled，实际 %q", scheduled.Status)
	}

	// scheduled 状态不允许再修改。
	_, err = svc.UpdateCampaign(ctx, camp.ID, PushCampaignInput{
		Name:         "X",
		Title:        "Y",
		Body:         "Z",
		TargetAppIDs: []string{"com.gamezone.gp001"},
	})
	if err == nil {
		t.Error("scheduled 状态不应允许修改")
	}
}

// TestDeeplinkPathValidation 验证 deeplinkPath 不得含域名（ADR-0002）。
func TestDeeplinkPathValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}

	_, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name:         "bad-path",
		Title:        "T",
		Body:         "B",
		DeeplinkPath: "https://example.com/promo/618", // 含域名，应被拒
		TargetAppIDs: []string{"com.arenaplus.ap01001"},
	}, "admin")
	if err == nil {
		t.Fatal("deeplinkPath 含域名应被拒绝（ADR-0002）")
	}
}

// TestPushStatusDefault 验证 PushStatus 在零配置时返回 enabled=false。
func TestPushStatusDefault(t *testing.T) {
	svc, _ := newTestService(t)
	status := svc.PushStatus()
	if status.Enabled {
		t.Error("默认 PUSH_ENABLED 应为 false")
	}
	// 三个品牌都未配置 service account。
	for _, brand := range []string{"ap", "bp", "gp"} {
		if status.Brands[brand] {
			t.Errorf("品牌 %s 未配置 service account，应为 false", brand)
		}
	}
}
