package service

import (
	"net"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

func TestNormalizeListingExtraQuery(t *testing.T) {
	ok := []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"?", ""},
		{"?channel=fb&lang=en", "channel=fb&lang=en"},
		{"channel=fb&lang=en", "channel=fb&lang=en"},
		{"  ?&a=1&&b=2&  ", "a=1&b=2"},
		{"?z=1&a=2", "z=1&a=2"},                                   // 保留输入顺序，不按 key 排序
		{"?redirect=/home/index?x=1", "redirect=/home/index?x=1"}, // query 合法字符原样保留
		{"?flag", "flag"},                                         // 无值参数
		{"?name=a b&tag=中文", "name=a%20b&tag=%E4%B8%AD%E6%96%87"}, // 非法字符转义
		{"?ref=x#top", "ref=x%23top"},                             // # 不能截断 URL
		{"?v=%E4%B8%AD", "v=%E4%B8%AD"},                           // 已转义的保持不变
	}
	for _, c := range ok {
		got, err := normalizeListingExtraQuery(c.in)
		if err != nil {
			t.Errorf("%q 应合法，实际报错 %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q 规范化应为 %q，实际 %q", c.in, c.want, got)
		}
	}

	bad := []string{
		"?=1",              // 缺参数名
		"?a=1&palcode=123", // palcode 只能走 PAL_CODE 字段
		"?PalCode=123",     // 大小写不敏感
		"?pal%63ode=123",   // 转义绕过也要拦
		"?a=100%",          // 悬空 %
		"?a=%zz",           // 非法转义
		"?" + strings.Repeat("a", listingExtraQueryMaxLen+1),
	}
	for _, in := range bad {
		if _, err := normalizeListingExtraQuery(in); err == nil {
			t.Errorf("%q 应被拒绝", in)
		}
	}
}

func TestBuildListingBSideURL(t *testing.T) {
	cases := []struct{ domain, pal, extra, want string }{
		{"https://arenaplus.ph", "", "", "https://arenaplus.ph"},
		{"https://arenaplus.ph", "1053259", "", "https://arenaplus.ph/?palcode=1053259"},
		{"https://arenaplus.ph/", "1053259", "channel=fb&lang=en", "https://arenaplus.ph/?palcode=1053259&channel=fb&lang=en"},
		{"https://arenaplus.ph", "", "channel=fb", "https://arenaplus.ph/?channel=fb"},
	}
	for _, c := range cases {
		if got := buildListingBSideURL(c.domain, c.pal, c.extra); got != c.want {
			t.Errorf("buildListingBSideURL(%q,%q,%q) = %q，应为 %q", c.domain, c.pal, c.extra, got, c.want)
		}
	}
}

// TestListingExtraQueryEndToEnd：后台填附加参数 → 网关下发的 B 面 URL 立即带上，改/清空也即时生效。
func TestListingExtraQueryEndToEnd(t *testing.T) {
	svc, ctx := newListingTestService(t, map[string]string{"112.198.0.1": "PH"})
	l, err := svc.CreateListing(ctx, CreateListingInput{
		BrandCode: "ap", Platform: "android", BundleID: "com.x.extraquery", Name: "x",
		PalCode: "1053259", ExtraQuery: "?channel=fb&lang=en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if l.ExtraQuery != "channel=fb&lang=en" {
		t.Fatalf("新建后 ExtraQuery 应去掉前导 ?，实际 %q", l.ExtraQuery)
	}
	if _, err := svc.SetListingGate(ctx, l.ID, SetGateInput{Countries: []string{"PH"}}); err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{GateEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	gateURL := func() string {
		t.Helper()
		res, err := svc.EvaluateListingGate(ctx, GateRequest{Platform: "android", BundleID: "com.x.extraquery", IP: net.ParseIP("112.198.0.1")})
		if err != nil {
			t.Fatal(err)
		}
		if res.Mode != model.GateModeB {
			t.Fatalf("菲律宾 IP 应进 B 面，实际 %s", res.Mode)
		}
		return res.URL
	}

	if got := gateURL(); got != "https://arenaplus.ph/?palcode=1053259&channel=fb&lang=en" {
		t.Errorf("B 面 URL 应带附加参数，实际 %q", got)
	}

	next := "?channel=tt"
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{ExtraQuery: &next}); err != nil {
		t.Fatal(err)
	}
	if got := gateURL(); got != "https://arenaplus.ph/?palcode=1053259&channel=tt" {
		t.Errorf("改参数后 B 面 URL 应即时更新，实际 %q", got)
	}

	empty := ""
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{ExtraQuery: &empty}); err != nil {
		t.Fatal(err)
	}
	if got := gateURL(); got != "https://arenaplus.ph/?palcode=1053259" {
		t.Errorf("清空参数后 B 面 URL 应只剩 palcode，实际 %q", got)
	}

	dup := "?palcode=999"
	if _, err := svc.UpdateListing(ctx, l.ID, UpdateListingInput{ExtraQuery: &dup}); err == nil {
		t.Error("附加参数里写 palcode 应被拒绝")
	}
}
