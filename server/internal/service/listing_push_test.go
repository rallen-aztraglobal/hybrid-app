package service

import (
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

// 上架包推送最关键的安全线：只有判定为 B 面的设备进入推送受众，A 面/未知设备一律排除。
// 这直接防止「把 B 面促销推给可能是审核员的 A 面设备」。
func TestListingPushOnlyTargetsBModeDevices(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx) // android 端

	// 注册三台 android 设备，上报不同 AB 面判定结果；只有 B 面那台应入受众。
	devices := []struct{ token, mode string }{
		{"tok-b-android", "B"}, // 应入受众
		{"tok-a-android", "A"}, // A 面，必须排除
		{"tok-unknown", ""},    // 未上报，视为非 B，必须排除
	}
	for _, d := range devices {
		if err := svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
			Platform:    "android",
			BundleID:    "com.vividnest.colorstack5821",
			DeviceToken: d.token,
			GateMode:    d.mode,
		}); err != nil {
			t.Fatalf("注册 %s 失败: %v", d.token, err)
		}
	}

	byPlatform, err := svc.repo.ActiveListingTokensBMode(ctx, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, ts := range byPlatform {
		for _, tk := range ts {
			total++
			// 明确断言：A 面与未知设备的 token 绝不出现在受众里。
			if tk.DeviceToken != "tok-b-android" {
				t.Errorf("非 B 面设备 %s 不得进入推送受众", tk.DeviceToken)
			}
			if tk.LastGateMode != model.GateModeB {
				t.Errorf("受众内设备 %s 的 gate mode 应为 B，实际 %q", tk.DeviceToken, tk.LastGateMode)
			}
		}
	}
	if total != 1 {
		t.Fatalf("B 面受众应为 1 台，实际 %d", total)
	}

	n, err := svc.repo.CountActiveListingTokensBMode(ctx, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("B 面设备计数应为 1，实际 %d", n)
	}
}

// Flutter 双端同包名：android 与 ios 是两个独立 listing 行，设备各归其列，互不串台。
func TestListingPushFlutterDualPlatformIsolation(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	androidL := mustCreateListing(t, svc, ctx)
	iosL, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "ios", BundleID: "com.vividnest.colorstack5821",
		Name: "ColorStack iOS", PalCode: "1053259",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 两端各注册一台 B 面设备。
	for _, d := range []struct{ platform, token string }{
		{"android", "tok-and"}, {"ios", "tok-ios"},
	} {
		if err := svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
			Platform: d.platform, BundleID: "com.vividnest.colorstack5821",
			DeviceToken: d.token, GateMode: "B",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// android listing 的受众只含 android 设备；ios listing 只含 ios 设备。
	if n, _ := svc.repo.CountActiveListingTokensBMode(ctx, androidL.ID); n != 1 {
		t.Errorf("android listing 受众应为 1，实际 %d", n)
	}
	if n, _ := svc.repo.CountActiveListingTokensBMode(ctx, iosL.ID); n != 1 {
		t.Errorf("ios listing 受众应为 1，实际 %d", n)
	}
}

// 设备再次上报判定结果时，其 mode 应被更新（A→B 或 B→A），受众随之变化。
func TestListingDeviceGateModeUpdates(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	l := mustCreateListing(t, svc, ctx)

	reg := func(mode string) {
		if err := svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
			Platform: "android", BundleID: "com.vividnest.colorstack5821",
			DeviceToken: "tok-x", GateMode: mode,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// 初次 A 面 → 不在受众。
	reg("A")
	if n, _ := svc.repo.CountActiveListingTokensBMode(ctx, l.ID); n != 0 {
		t.Fatalf("A 面设备不应在受众，实际计数 %d", n)
	}
	// 同一设备后来判为 B 面 → 进入受众（upsert 覆盖，不新增行）。
	reg("B")
	if n, _ := svc.repo.CountActiveListingTokensBMode(ctx, l.ID); n != 1 {
		t.Fatalf("设备转 B 面后应入受众，实际计数 %d", n)
	}
	// 再次转回 A 面 → 退出受众。
	reg("A")
	if n, _ := svc.repo.CountActiveListingTokensBMode(ctx, l.ID); n != 0 {
		t.Fatalf("设备转回 A 面后应退出受众，实际计数 %d", n)
	}
}

// 注册到不存在的上架包应被拒绝（防滥用）。
func TestRegisterListingTokenUnknownListing(t *testing.T) {
	svc, ctx := newListingTestService(t, nil)
	err := svc.RegisterListingDeviceToken(ctx, RegisterListingTokenInput{
		Platform: "android", BundleID: "com.nonexistent.app", DeviceToken: "tok",
	})
	if err == nil {
		t.Error("注册到不存在的上架包应报错")
	}
}
