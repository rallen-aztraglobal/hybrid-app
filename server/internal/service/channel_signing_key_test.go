package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
)

// TestChannelSigningKey 覆盖「按渠道选择签名 key」的读写语义：未注册的 key 一律拒绝；
// 已注册的 key（如 emptyapp）可落库；更新传空串恢复默认；未传不改动；manifest 原样带出。
func TestChannelSigningKey(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// 未注册的 signingKey → 400，新建应被拒绝。
	_, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "ArenaPlus",
		SigningKey: "nosuchkey",
	})
	if err == nil {
		t.Fatalf("未注册的 signingKey 应被拒绝")
	}
	if svcErr, ok := err.(*Error); !ok || svcErr.Code != http.StatusBadRequest {
		t.Fatalf("未注册 signingKey 应返回 400，实际 %+v", err)
	}

	// 已注册的 signingKey（emptyapp）→ 正常落库。
	ch, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01018", PalCode: "PAL1", AppName: "ArenaPlus",
		SigningKey: "emptyapp",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if ch.SigningKey != "emptyapp" {
		t.Fatalf("signingKey 应落库为 emptyapp，实际 %q", ch.SigningKey)
	}

	// 未传 signingKey（nil）→ 不改动。
	name := "ArenaPlus 2"
	ch, err = svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{AppName: &name})
	if err != nil {
		t.Fatalf("改 appName 失败: %v", err)
	}
	if ch.SigningKey != "emptyapp" {
		t.Fatalf("未传 signingKey 不应被改动，实际 %q", ch.SigningKey)
	}

	// 更新为未注册的 key → 拒绝，且不改动原值。
	unknown := "nosuchkey"
	if _, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{SigningKey: &unknown}); err == nil {
		t.Fatalf("更新为未注册的 signingKey 应被拒绝")
	}

	// 传空串 → 恢复默认（清空）。
	empty := ""
	ch, err = svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{SigningKey: &empty})
	if err != nil {
		t.Fatalf("恢复默认 signingKey 失败: %v", err)
	}
	if ch.SigningKey != "" {
		t.Fatalf("传空串应恢复默认（清空），实际 %q", ch.SigningKey)
	}

	// manifest（CLI 配置下发）应原样带出 signingKey。
	other := "emptyapp"
	if _, err := svc.UpdateChannel(ctx, ch.ID, UpdateChannelInput{SigningKey: &other}); err != nil {
		t.Fatalf("重新设置 signingKey 失败: %v", err)
	}
	m, err := svc.BuildManifestForBrand(ctx, auth.FullScope(), "ap")
	if err != nil {
		t.Fatalf("manifest 失败: %v", err)
	}
	if len(m.Channels) != 1 || m.Channels[0].SigningKey != "emptyapp" {
		t.Fatalf("manifest 应带出 signingKey=emptyapp，实际 %+v", m.Channels)
	}
}

// TestSigningKeyRegistry 覆盖注册表的基本契约：默认项存在且合法；未知 ID 不合法。
func TestSigningKeyRegistry(t *testing.T) {
	if !model.IsKnownSigningKey("") {
		t.Fatalf("空串（默认 key）应始终合法")
	}
	if !model.IsKnownSigningKey("emptyapp") {
		t.Fatalf("emptyapp 应已注册")
	}
	if model.IsKnownSigningKey("nosuchkey") {
		t.Fatalf("未注册的 key 不应合法")
	}
	keys := model.SigningKeys()
	if len(keys) < 2 {
		t.Fatalf("注册表应至少含默认 + emptyapp 两项，实际 %d", len(keys))
	}
	var hasDefault bool
	for _, k := range keys {
		if k.ID == "" && k.IsDefault {
			hasDefault = true
		}
	}
	if !hasDefault {
		t.Fatalf("注册表应含标记为 isDefault 的默认项")
	}
}
