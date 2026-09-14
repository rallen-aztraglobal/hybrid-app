package render

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hybrid-app/cli/internal/manifest"
)

// TestRenderAdjustTokensWritesBoundChannelsOnly 验证 ADR-0013 §3：
// 只有 AdjustAppToken 非空的渠道写入 adjust-tokens.json，键=派生后的 applicationId，
// events 原样透传（CLI 不解析 CSV，只消费后端已解析好的 {name:token}）。
func TestRenderAdjustTokensWritesBoundChannelsOnly(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand: "gp",
		Channels: []manifest.Channel{
			{
				Flavor: "gpgzmkk042", ApplicationId: "com.gamezone.gpgzmkk042", PalCode: "1", AppName: "GZ",
				AdjustAppToken: "abc123xyz",
				AdjustEvents: map[string]string{
					"Login":    "wzb3fb",
					"Purchase": "gyuu2f",
				},
			},
			{
				// 未绑定 Adjust（AdjustAppToken 为空）：不应出现在渲染产物中。
				Flavor: "gpgzmkk043", ApplicationId: "com.gamezone.gpgzmkk043", PalCode: "2", AppName: "GZ2",
			},
		},
	}
	src := &fixtureSource{m: m}

	res, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.AdjustBoundCount != 1 {
		t.Fatalf("期望 1 个已绑定 Adjust 的渠道，实得 %d", res.AdjustBoundCount)
	}

	data, err := os.ReadFile(r.AppAdjustTokensJSON())
	if err != nil {
		t.Fatalf("adjust-tokens.json 应已写出: %v", err)
	}
	var got map[string]manifest.AdjustTokenEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("解析 adjust-tokens.json 失败: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("期望仅 1 个键，实得 %d: %+v", len(got), got)
	}
	// 键必须是派生后的 applicationId（ADR-0009），而不是渠道自带的、也不是 flavor。
	entry, ok := got["com.gamezone.gpgzmkk042"]
	if !ok {
		t.Fatalf("缺少已绑定渠道的键，got=%+v", got)
	}
	if entry.AppToken != "abc123xyz" {
		t.Errorf("appToken 不符，实得 %q", entry.AppToken)
	}
	if entry.Events["Login"] != "wzb3fb" || entry.Events["Purchase"] != "gyuu2f" {
		t.Errorf("events 未原样透传，实得 %+v", entry.Events)
	}
	if _, ok := got["com.gamezone.gpgzmkk043"]; ok {
		t.Error("未绑定渠道不应出现在 adjust-tokens.json 中")
	}
}

// TestRenderAdjustTokensSkipsWhenNoneBound 验证「无渠道绑定 Adjust」时不生成
// adjust-tokens.json，且会清理工作区可能残留的旧文件（构建机跨任务持久化场景）。
func TestRenderAdjustTokensSkipsWhenNoneBound(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand: "ap",
		Channels: []manifest.Channel{
			{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "1", AppName: "A"},
		},
	}
	src := &fixtureSource{m: m}

	// 模拟工作区残留了上一次品牌切换前落地的 adjust-tokens.json。
	stale := r.AppAdjustTokensJSON()
	if err := os.MkdirAll(r.Root+"/app", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte(`{"stale":"data"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.AdjustBoundCount != 0 {
		t.Fatalf("期望 0 个已绑定渠道，实得 %d", res.AdjustBoundCount)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("无渠道绑定时应清理残留的 adjust-tokens.json，但文件仍存在")
	}
}

// TestRenderAdjustTokensBpRawEvents 验证 ADR-0013「BP 原始事件」旁支：
//   - bpRaw 渠道写出 bpRawEvents=true 与品牌级 deepLinkHost；
//   - 非 bpRaw 渠道（即使已绑定 Adjust）两键都不出现；
//   - bpRaw 但品牌 host 为空时只写 bpRawEvents，不写 deepLinkHost；
//   - 未绑定 App Token 的 bpRaw 渠道仍不写入条目（开关对它无意义）。
func TestRenderAdjustTokensBpRawEvents(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand:              "bp",
		AdjustDeepLinkHost: "link.bingoplus.com",
		Channels: []manifest.Channel{
			{
				// bpRaw 开启 + 已绑定 token → 两键都应写出。
				Flavor: "bp001", ApplicationId: "com.bingoplus.bp001", PalCode: "1", AppName: "BP1",
				AdjustAppToken:    "tok-bpraw",
				AdjustBpRawEvents: true,
			},
			{
				// 已绑定 token 但未开 bpRaw → 两键都不应出现。
				Flavor: "bp002", ApplicationId: "com.bingoplus.bp002", PalCode: "2", AppName: "BP2",
				AdjustAppToken: "tok-normal",
			},
			{
				// bpRaw 开启但未绑定 token → 不应出现在产物中。
				Flavor: "bp003", ApplicationId: "com.bingoplus.bp003", PalCode: "3", AppName: "BP3",
				AdjustBpRawEvents: true,
			},
		},
	}
	src := &fixtureSource{m: m}

	res, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.AdjustBoundCount != 2 {
		t.Fatalf("期望 2 个已绑定 Adjust 的渠道，实得 %d", res.AdjustBoundCount)
	}

	data, err := os.ReadFile(r.AppAdjustTokensJSON())
	if err != nil {
		t.Fatalf("adjust-tokens.json 应已写出: %v", err)
	}
	var got map[string]manifest.AdjustTokenEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("解析 adjust-tokens.json 失败: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("期望仅 2 个键，实得 %d: %+v", len(got), got)
	}

	bpraw, ok := got["com.bingoplus.bp001"]
	if !ok {
		t.Fatalf("bpRaw 渠道应写入，got=%+v", got)
	}
	if !bpraw.BpRawEvents {
		t.Errorf("bpRawEvents 应为 true，实得 %+v", bpraw)
	}
	if bpraw.DeepLinkHost != "link.bingoplus.com" {
		t.Errorf("deepLinkHost 应透传品牌级配置，实得 %q", bpraw.DeepLinkHost)
	}

	normal, ok := got["com.bingoplus.bp002"]
	if !ok {
		t.Fatalf("非 bpRaw 渠道仍应写入（已绑定 token），got=%+v", got)
	}
	if normal.BpRawEvents {
		t.Error("非 bpRaw 渠道不应写出 bpRawEvents=true")
	}
	if normal.DeepLinkHost != "" {
		t.Errorf("非 bpRaw 渠道不应写出 deepLinkHost，实得 %q", normal.DeepLinkHost)
	}

	if _, ok := got["com.bingoplus.bp003"]; ok {
		t.Error("未绑定 token 的 bpRaw 渠道不应出现在 adjust-tokens.json 中")
	}

	// 校验序列化产物本身：非 bpRaw 渠道两键在 JSON 文本中完全不出现（而非出现为 false/""）。
	raw := string(data)
	if strings.Contains(raw, `"bpRawEvents"`) == false {
		t.Error("bpRaw 渠道应在原始 JSON 中含 bpRawEvents 键")
	}
	var normalRaw map[string]json.RawMessage
	if err := json.Unmarshal(data, &normalRaw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(normalRaw["com.bingoplus.bp002"]), "bpRawEvents") {
		t.Errorf("非 bpRaw 渠道的原始 JSON 不应含 bpRawEvents 键，实得 %s", normalRaw["com.bingoplus.bp002"])
	}
}

// TestRenderAdjustTokensBpRawEventsWithoutBrandHost 验证 bpRaw 渠道在品牌未配置
// AdjustDeepLinkHost 时只写 bpRawEvents，不写 deepLinkHost（omitempty，不落空串）。
func TestRenderAdjustTokensBpRawEventsWithoutBrandHost(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand: "bp",
		Channels: []manifest.Channel{
			{
				Flavor: "bp001", ApplicationId: "com.bingoplus.bp001", PalCode: "1", AppName: "BP1",
				AdjustAppToken:    "tok-bpraw",
				AdjustBpRawEvents: true,
			},
		},
	}
	src := &fixtureSource{m: m}

	if _, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(r.AppAdjustTokensJSON())
	if err != nil {
		t.Fatalf("adjust-tokens.json 应已写出: %v", err)
	}
	var got map[string]manifest.AdjustTokenEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	entry := got["com.bingoplus.bp001"]
	if !entry.BpRawEvents {
		t.Error("bpRawEvents 应为 true")
	}
	if entry.DeepLinkHost != "" {
		t.Errorf("品牌未配置 host 时不应写出 deepLinkHost，实得 %q", entry.DeepLinkHost)
	}
	if strings.Contains(string(data), "deepLinkHost") {
		t.Error("品牌未配置 host 时原始 JSON 不应含 deepLinkHost 键")
	}
}

// TestRenderAdjustTokensDryRunWritesNothing 验证 dry-run 既不写新文件也不清理残留文件。
func TestRenderAdjustTokensDryRunWritesNothing(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand: "gp",
		Channels: []manifest.Channel{
			{Flavor: "gpgzmkk042", ApplicationId: "com.gamezone.gpgzmkk042", PalCode: "1", AppName: "GZ", AdjustAppToken: "tok"},
		},
	}
	src := &fixtureSource{m: m}

	if _, err := RenderManifest(context.Background(), r, src, m, Options{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.AppAdjustTokensJSON()); !os.IsNotExist(err) {
		t.Error("dry-run 不应写 adjust-tokens.json")
	}
}
