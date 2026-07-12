package render

import (
	"context"
	"encoding/json"
	"os"
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
