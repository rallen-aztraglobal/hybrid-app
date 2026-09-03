package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestManifestMapsSigningKey 验证 GET /api/build/manifest 响应里的 channel.signingKey
// （ADR-0016）被正确透传进 manifest.Channel.SigningKey——这一步是 serverManifest 手工转换，
// 不是直接 json.Unmarshal，漏拷会导致 runner 永远读到空值、把商店渠道当默认签名投递。
func TestManifestMapsSigningKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/build/manifest" || r.Method != http.MethodGet {
			t.Errorf("意外请求: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(envelopeJSON(t, map[string]any{
			"brand":         "ap",
			"brandDomains":  []string{"https://arenaplus.ph"},
			"configBaseUrl": "https://cdn.example.com/cfg",
			"channels": []map[string]any{
				{
					"flavorName":       "ap01018",
					"applicationId":    "com.arenaplus.ap01018",
					"palCode":          "111",
					"appName":          "AP",
					"effectiveDomains": []string{"https://arenaplus.ph"},
					"signingKey":       "emptyapp",
				},
				{
					"flavorName":       "ap02000",
					"applicationId":    "com.arenaplus.ap02000",
					"palCode":          "222",
					"appName":          "AP2",
					"effectiveDomains": []string{"https://arenaplus.ph"},
					// signingKey 缺省 = 空串 = 默认 key。
				},
			},
		}))
	}))
	defer srv.Close()

	cl := New(srv.URL, "tok")
	m, err := cl.Manifest(context.Background(), "ap")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Channels) != 2 {
		t.Fatalf("应解析出 2 个渠道，实得 %d", len(m.Channels))
	}
	if m.Channels[0].Flavor != "ap01018" || m.Channels[0].SigningKey != "emptyapp" {
		t.Errorf("ap01018 应带 signingKey=emptyapp，实得 %+v", m.Channels[0])
	}
	if m.Channels[1].Flavor != "ap02000" || m.Channels[1].SigningKey != "" {
		t.Errorf("ap02000 未设置 signingKey 时应为空串（默认 key），实得 %+v", m.Channels[1])
	}
}
