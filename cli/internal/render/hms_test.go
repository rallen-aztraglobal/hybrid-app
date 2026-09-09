package render

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hybrid-app/cli/internal/manifest"
)

// TestRenderHMSChannelsWritesEffectiveFlags 验证 hms-channels.json 写的是**全量**渠道的有效值：
// 后端显式下发的（含显式 false）以下发为准；未下发（老后端）回落「品牌整体开 HMS 或 _hw 华为商店包」。
func TestRenderHMSChannelsWritesEffectiveFlags(t *testing.T) {
	on, off := true, false
	m := &manifest.Manifest{
		Brand:      "ap",
		HMSEnabled: false, // ap 品牌整体不开 HMS
		Channels: []manifest.Channel{
			// 显式开：上架华为商店但 flavor 不带 _hw 后缀的老渠道（本功能的由来）。
			{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "1", AppName: "AP", HMSEnabled: &on},
			// 显式关：即使名字像华为包也不集成。
			{Flavor: "johnc2010092_hw", ApplicationId: "com.arenaplus.johnc2010092.hw", PalCode: "2", AppName: "AP2", HMSEnabled: &off},
			// 未下发：回落默认规则——_hw 结尾 → true。
			{Flavor: "johnc2010097_hw", ApplicationId: "com.arenaplus.johnc2010097.hw", PalCode: "3", AppName: "AP3"},
			// 未下发且非 _hw、品牌也不开 → false。
			{Flavor: "ap01036", ApplicationId: "com.arenaplus.ap01036", PalCode: "4", AppName: "AP4"},
		},
	}
	r := fakeRepo(t)
	src := &fixtureSource{m: m}

	res, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.HMSEnabledCount != 2 {
		t.Fatalf("期望 2 个渠道集成 HMS，实得 %d", res.HMSEnabledCount)
	}

	data, err := os.ReadFile(r.AppHMSChannelsJSON())
	if err != nil {
		t.Fatalf("hms-channels.json 应已写出: %v", err)
	}
	var got map[string]bool
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("解析 hms-channels.json 失败: %v", err)
	}
	// 全量收录（含 false）：Gradle 侧「键缺失」的语义是回落默认规则，不是关闭。
	want := map[string]bool{
		"com.arenaplus.ap01018":         true,
		"com.arenaplus.johnc2010092.hw": false,
		"com.arenaplus.johnc2010097.hw": true,
		"com.arenaplus.ap01036":         false,
	}
	if len(got) != len(want) {
		t.Fatalf("期望 %d 个键（全量渠道），实得 %d: %+v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s 期望 %v，实得 %v", k, v, got[k])
		}
	}
}

// TestRenderHMSChannelsBrandWideFallback 验证品牌整体开 HMS（bp）且后端未下发渠道级值时，
// 全部渠道回落成 true——加渠道级开关不能悄悄关掉存量 bp 包的 OAID 采集。
func TestRenderHMSChannelsBrandWideFallback(t *testing.T) {
	m := &manifest.Manifest{
		Brand:      "bp",
		HMSEnabled: true,
		Channels: []manifest.Channel{
			{Flavor: "bp001", ApplicationId: "com.bingoplus.bp001", PalCode: "1", AppName: "BP"},
		},
	}
	r := fakeRepo(t)
	src := &fixtureSource{m: m}

	res, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.HMSEnabledCount != 1 {
		t.Fatalf("bp 渠道应回落为集成 HMS，实得 %d", res.HMSEnabledCount)
	}
	data, err := os.ReadFile(r.AppHMSChannelsJSON())
	if err != nil {
		t.Fatalf("hms-channels.json 应已写出: %v", err)
	}
	var got map[string]bool
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("解析 hms-channels.json 失败: %v", err)
	}
	if !got["com.bingoplus.bp001"] {
		t.Errorf("bp 存量渠道应为 true，实得 %+v", got)
	}
}

// TestRenderHMSChannelsMergesOtherBrands 验证跨品牌合并：hms-channels.json 单文件服务
// build.gradle 里三个品牌的 allChannels，`pull bp` 不能抹掉 ap/gp 已有的显式开关，
// 但本品牌自己的陈旧键（渠道已删）要随本次 manifest 一起清掉。
func TestRenderHMSChannelsMergesOtherBrands(t *testing.T) {
	r := fakeRepo(t)
	// 工作区里已有上一次 pull ap 的产物 + 一条本次 manifest 里不再存在的 bp 陈旧键。
	seed := map[string]bool{
		"com.arenaplus.ap01018":  true,  // 别的品牌，必须原样保留
		"com.gamezone.gzmkt031":  false, // 别的品牌，必须原样保留
		"com.bingoplus.bp_stale": true,  // 本品牌陈旧键，必须清掉
	}
	if err := os.MkdirAll(filepath.Dir(r.AppHMSChannelsJSON()), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(r.AppHMSChannelsJSON(), data, 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		Brand:      "bp",
		HMSEnabled: true,
		Channels: []manifest.Channel{
			{Flavor: "bp001", ApplicationId: "com.bingoplus.bp001", PalCode: "1", AppName: "BP"},
		},
	}
	if _, err := RenderManifest(context.Background(), r, &fixtureSource{m: m}, m, Options{SkipRes: true}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(r.AppHMSChannelsJSON())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("解析 hms-channels.json 失败: %v", err)
	}
	if !got["com.arenaplus.ap01018"] {
		t.Error("pull bp 抹掉了 ap 的显式开关——本地再打 ap01018 会静默漏掉 OAID")
	}
	if _, ok := got["com.gamezone.gzmkt031"]; !ok {
		t.Error("pull bp 抹掉了 gp 的显式配置")
	}
	if _, ok := got["com.bingoplus.bp_stale"]; ok {
		t.Error("本品牌已删渠道的陈旧键应被清掉")
	}
	if !got["com.bingoplus.bp001"] {
		t.Error("本次 manifest 的 bp 渠道应写入且为 true")
	}
}
