package model

import "testing"

// TestNormalizeOpenMode 验证 B 面打开方式的归一化：只认 internal/external（大小写、首尾空白
// 不敏感），其余（含空串）一律回落 internal——这是「默认内开」的向后兼容口径。
func TestNormalizeOpenMode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空串回落 internal", "", ListingOpenInternal},
		{"纯空白回落 internal", "   ", ListingOpenInternal},
		{"internal 原样保留", "internal", ListingOpenInternal},
		{"external 原样保留", "external", ListingOpenExternal},
		{"大写 EXTERNAL 归一化为小写", "EXTERNAL", ListingOpenExternal},
		{"混合大小写 Internal", "Internal", ListingOpenInternal},
		{"首尾空白 external 应被清理", "  external  ", ListingOpenExternal},
		{"未知值回落 internal", "popup", ListingOpenInternal},
		{"未知值回落 internal（非法枚举）", "webview", ListingOpenInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeOpenMode(tc.in); got != tc.want {
				t.Errorf("NormalizeOpenMode(%q) = %q，期望 %q", tc.in, got, tc.want)
			}
		})
	}
}
