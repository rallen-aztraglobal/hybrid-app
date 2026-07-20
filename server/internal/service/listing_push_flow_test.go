package service

import (
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

// 端到端（到 dry-run 为止，真发需 FCM 私钥）：建上架包推送活动 → dry-run 只统计 B 面设备。
func TestListingCampaignCreateAndDryRun(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	// 注册两台 B 面 + 一台 A 面设备。
	for _, d := range []struct{ token, mode string }{
		{"b1", "B"}, {"b2", "B"}, {"a1", "A"},
	} {
		if err := svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
			Platform: "android", BundleID: "com.vividnest.colorstack5821",
			DeviceToken: d.token, GateMode: d.mode,
		}); err != nil {
			t.Fatal(err)
		}
	}

	campaign, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "上架包活动1", Title: "新关卡", Body: "快来玩",
		ListingIDs: []uint64{l.ID},
	})
	if err != nil {
		t.Fatalf("创建上架包活动失败: %v", err)
	}
	if campaign.Kind != model.CampaignKindListing {
		t.Errorf("活动 kind 应为 listing，实际 %s", campaign.Kind)
	}

	// dry-run：受众应只含 2 台 B 面设备，不含 A 面那台。
	res, err := svc.SendListingCampaign(ctx, campaign.ID, true)
	if err != nil {
		t.Fatalf("dry-run 失败: %v", err)
	}
	if !res.DryRun || res.Preview == nil {
		t.Fatal("dry-run 应返回 Preview")
	}
	if res.Preview.TotalDevices != 2 {
		t.Errorf("B 面受众应为 2，实际 %d", res.Preview.TotalDevices)
	}
}

// 预估受众接口应只数 B 面设备。
func TestListingCampaignAudienceOnlyBMode(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)
	for _, d := range []struct{ token, mode string }{{"b", "B"}, {"a", "A"}, {"x", ""}} {
		_ = svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
			Platform: "android", BundleID: "com.vividnest.colorstack5821",
			DeviceToken: d.token, GateMode: d.mode,
		})
	}
	aud, err := svc.ListingCampaignAudienceFor(ctx, []uint64{l.ID})
	if err != nil {
		t.Fatal(err)
	}
	if aud.TotalDevices != 1 || aud.ByListing[l.ID] != 1 {
		t.Errorf("受众应只含 1 台 B 面设备，实际 total=%d byListing=%v", aud.TotalDevices, aud.ByListing)
	}
}

// 渠道活动与上架包活动在各自列表里互不串台。
func TestListingAndChannelCampaignListsSeparate(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	// 一条上架包活动 + 一条渠道活动。
	lc, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "上架包", Title: "t", Body: "b", ListingIDs: []uint64{l.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lc.Kind != model.CampaignKindListing {
		t.Errorf("上架包活动 kind 应为 listing，实际 %s", lc.Kind)
	}
	if len(lc.ListingIDs) != 1 || lc.ListingIDs[0] != l.ID {
		t.Errorf("上架包活动应回带 listingIds=[%d]，实际 %v", l.ID, lc.ListingIDs)
	}
	if _, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name: "渠道", Title: "t", Body: "b", TargetAppIDs: []string{"com.x.y"},
	}, "tester"); err != nil {
		t.Fatal(err)
	}

	// 上架包列表只含上架包活动。
	listing, err := svc.ListListingCampaigns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing) != 1 || listing[0].Kind != model.CampaignKindListing {
		t.Errorf("上架包列表应只含 1 条 listing 活动，实际 %d 条", len(listing))
	}

	// 渠道列表只含渠道活动（不含上架包）。
	channel, err := svc.ListCampaigns(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range channel {
		if c.Kind == model.CampaignKindListing {
			t.Error("渠道推送列表不应出现上架包活动")
		}
	}
}

// 校验：无目标、目标不存在、把渠道活动误发给上架包端点，都应被拒。
func TestListingCampaignValidation(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)

	// 无目标。
	if _, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "x", Title: "t", Body: "b", ListingIDs: nil,
	}); err == nil {
		t.Error("无目标上架包应被拒绝")
	}

	// 目标不存在。
	if _, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "x", Title: "t", Body: "b", ListingIDs: []uint64{99999},
	}); err == nil {
		t.Error("不存在的上架包 id 应被拒绝")
	}

	// 渠道活动不能走上架包发送端点。
	ch, err := svc.CreateCampaign(ctx, PushCampaignInput{
		Name: "渠道活动", Title: "t", Body: "b", TargetAppIDs: []string{"com.x.y"},
	}, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendListingCampaign(ctx, ch.ID, true); err == nil {
		t.Error("kind=channel 的活动不应能走上架包发送")
	}
}

// PUSH_ENABLED=false 时真发（非 dry-run）应被门控拦截，活动不进 sending。
func TestListingCampaignSendGatedWhenPushDisabled(t *testing.T) {
	svc, ctx := newListingTestService(t, nil) // PUSH_ENABLED 默认 false
	l := mustCreateListing(t, svc, ctx)
	_ = svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
		Platform: "android", BundleID: "com.vividnest.colorstack5821",
		DeviceToken: "b", GateMode: "B",
	})
	campaign, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "x", Title: "t", Body: "b", ListingIDs: []uint64{l.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendListingCampaign(ctx, campaign.ID, false); err == nil {
		t.Error("PUSH_ENABLED=false 时真发应被拦截")
	}
}
