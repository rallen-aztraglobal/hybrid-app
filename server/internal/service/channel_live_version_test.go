package service

import (
	"context"
	"strings"
	"testing"
)

// TestChannelLiveVersion 覆盖「线上版本号」（人工备忘字段）的读写语义：
// 新建时去空白落库；更新只传 liveVersion 不动其他字段；传空串清空；未传不改；超长拒绝。
func TestChannelLiveVersion(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "ArenaPlus",
		LiveVersion: "  1.2.3 ",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if ch.LiveVersion != "1.2.3" {
		t.Fatalf("新建应去空白落库为 1.2.3，实际 %q", ch.LiveVersion)
	}

	// 卡片就地编辑：只发 liveVersion，palCode/appName 等保持原值。
	v := "1.3.0"
	ch, err = svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{LiveVersion: &v})
	if err != nil {
		t.Fatalf("只改 liveVersion 失败: %v", err)
	}
	if ch.LiveVersion != "1.3.0" || ch.PalCode != "PAL1" || ch.AppName != "ArenaPlus" {
		t.Fatalf("只改 liveVersion 后其它字段应不变，实际 live=%q pal=%q name=%q", ch.LiveVersion, ch.PalCode, ch.AppName)
	}

	// 未传 liveVersion（nil）→ 不改动。
	name := "ArenaPlus 2"
	ch, err = svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{AppName: &name})
	if err != nil {
		t.Fatalf("改 appName 失败: %v", err)
	}
	if ch.LiveVersion != "1.3.0" {
		t.Fatalf("未传 liveVersion 不应被改动，实际 %q", ch.LiveVersion)
	}

	// 传空串 → 清空。
	empty := ""
	ch, err = svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{LiveVersion: &empty})
	if err != nil {
		t.Fatalf("清空 liveVersion 失败: %v", err)
	}
	if ch.LiveVersion != "" {
		t.Fatalf("传空串应清空，实际 %q", ch.LiveVersion)
	}

	// 超长（> VARCHAR(32)）拒绝：更新与新建两条路径。
	long := strings.Repeat("9", maxLiveVersionLen+1)
	if _, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{LiveVersion: &long}); err == nil {
		t.Fatalf("超长 liveVersion 更新应被拒绝")
	}
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01019", PalCode: "PAL2", AppName: "X", LiveVersion: long,
	}); err == nil {
		t.Fatalf("超长 liveVersion 新建应被拒绝")
	}
}
