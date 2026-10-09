package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/seed"
)

// ---------- 辅助 ----------

func newPopupTestService(t *testing.T) (*Service, *repo.Repo) {
	t.Helper()
	svc, r := newTestService(t)
	if err := seed.EnsurePopupPositions(context.Background(), r.DB()); err != nil {
		t.Fatalf("seed 弹窗位置失败: %v", err)
	}
	return svc, r
}

func mustCreateChannel(t *testing.T, svc *Service, brand, flavor string) string {
	t.Helper()
	ch, err := svc.CreateChannel(context.Background(), CreateChannelInput{
		BrandCode: brand, FlavorName: flavor, PalCode: "PAL-" + flavor, AppName: flavor,
	})
	if err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	return ch.ApplicationID
}

func card(img, title string) PopupCardInput { return PopupCardInput{ImageURL: img, Title: title} }

const imgURL = "https://cdn.example.com/popup/images/a.png"

// validInput 返回某位置下一份合法入参。
func validInput(pos string) PopupInput {
	in := PopupInput{Name: "测试弹窗", Position: pos, Enabled: true, OpenMode: "webview"}
	switch pos {
	case "P3", "P5", "P8":
		in.Cards = []PopupCardInput{card(imgURL, "Hello")}
		if pos == "P5" {
			in.Cards = []PopupCardInput{{Title: "Hello"}}
		}
	default:
		in.Cards = []PopupCardInput{card(imgURL, "")}
	}
	return in
}

func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return AsError(err).Message
}

func bptr(b bool) *bool { return &b }

type fakeGeo map[string]string

func (g fakeGeo) Country(ip net.IP) (string, bool) {
	c, ok := g[ip.String()]
	return c, ok
}

// ---------- 版本号 ----------

func TestPopupVersionConversion(t *testing.T) {
	cases := []struct {
		in   string
		want int
		bad  bool
	}{
		{"", 0, false}, {"1.0.3", 10003, false}, {"2.15.99", 21599, false}, {"0.0.1", 1, false},
		{"1.0", 0, true}, {"1.0.100", 0, true}, {"a.b.c", 0, true}, {"1.0.3.4", 0, true},
	}
	for _, c := range cases {
		got, err := ParsePopupVersion(c.in)
		if (err != nil) != c.bad || got != c.want {
			t.Errorf("ParsePopupVersion(%q)=%d,%v want %d bad=%v", c.in, got, err, c.want, c.bad)
		}
	}
	if FormatPopupVersion(10003) != "1.0.3" || FormatPopupVersion(0) != "" {
		t.Errorf("FormatPopupVersion 反向换算错误")
	}
}

// ---------- 保存校验与归一化 ----------

func TestNormalizePopupInputRules(t *testing.T) {
	five := func(pos string) PopupInput {
		in := validInput(pos)
		in.Cards = nil
		for i := 0; i < 5; i++ {
			in.Cards = append(in.Cards, card(imgURL, "t"))
		}
		return in
	}
	six := func(pos string) PopupInput {
		in := five(pos)
		in.Cards = append(in.Cards, card(imgURL, "t"))
		return in
	}
	mod := func(pos string, f func(*PopupInput)) PopupInput {
		in := validInput(pos)
		f(&in)
		return in
	}
	cases := []struct {
		name    string
		in      PopupInput
		wantErr string // 非空 = 期望 400 且包含该子串
	}{
		{"P1 合法", validInput("P1"), ""},
		{"P1 五张合法", five("P1"), ""},
		{"P1 六张超限", six("P1"), "最多 5 张"},
		{"P3 六张超限", six("P3"), "最多 5 张"},
		{"P7 六张超限", six("P7"), "最多 5 张"},
		{"P2 两张超限", mod("P2", func(in *PopupInput) { in.Cards = append(in.Cards, card(imgURL, "")) }), "最多 1 张"},
		{"P4 两张超限", mod("P4", func(in *PopupInput) { in.Cards = append(in.Cards, card(imgURL, "")) }), "最多 1 张"},
		{"无卡片", mod("P1", func(in *PopupInput) { in.Cards = nil }), "至少需要 1 张"},
		{"位置非法", mod("P1", func(in *PopupInput) { in.Position = "P9" }), "位置"},
		{"名称为空", mod("P1", func(in *PopupInput) { in.Name = "  " }), "名称"},
		{"名称过长", mod("P1", func(in *PopupInput) { in.Name = strings.Repeat("名", 65) }), "64"},
		{"优先级越界", mod("P1", func(in *PopupInput) { in.Priority = 1000 }), "0~999"},
		{"P1 缺图", mod("P1", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "图片必填"},
		{"P2 缺图", mod("P2", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "图片必填"},
		{"P6 缺图", mod("P6", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "图片必填"},
		{"P7 缺图", mod("P7", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "图片必填"},
		{"P8 缺图", mod("P8", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "图片必填"},
		{"图片非 http", mod("P1", func(in *PopupInput) { in.Cards[0].ImageURL = "ftp://x/a.png" }), "http(s)"},
		{"P3 无图合法", mod("P3", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), ""},
		{"P3 缺文案", mod("P3", func(in *PopupInput) { in.Cards[0].Title = "" }), "文案必填"},
		{"P4 无图无文案", mod("P4", func(in *PopupInput) { in.Cards[0].ImageURL = "" }), "无图时文案必填"},
		{"P4 无图有文案合法", mod("P4", func(in *PopupInput) { in.Cards[0].ImageURL = ""; in.Cards[0].Title = "更新" }), ""},
		{"P5 缺文案", mod("P5", func(in *PopupInput) { in.Cards[0].Title = "" }), "文案必填"},
		{"P8 文案超 8 字符", mod("P8", func(in *PopupInput) { in.Cards[0].Title = "123456789" }), "8"},
		{"P8 中文 8 字合法（按 rune）", mod("P8", func(in *PopupInput) { in.Cards[0].Title = "一二三四五六七八" }), ""},
		{"tab 开启缺文案", mod("P1", func(in *PopupInput) { in.TabEnabled = true }), "便条文案必填"},
		{"tab 文案超 8", mod("P1", func(in *PopupInput) { in.TabEnabled = true; in.TabText = "123456789" }), "8"},
		{"tab 合法", mod("P1", func(in *PopupInput) { in.TabEnabled = true; in.TabText = "Bonus" }), ""},
		{"autoplay 越界", mod("P3", func(in *PopupInput) { in.AutoplaySeconds = 31 }), "2~30"},
		{"autoplay 过小", mod("P3", func(in *PopupInput) { in.AutoplaySeconds = 1 }), "2~30"},
		{"resumeGap 越界", mod("P1", func(in *PopupInput) { in.ResumeGapMinutes = 4 }), "5~1440"},
		{"resumeGap 越界2", mod("P1", func(in *PopupInput) { in.ResumeGapMinutes = 1441 }), "5~1440"},
		{"版本格式非法", mod("P1", func(in *PopupInput) { in.MinVersion = "1.0" }), "X.Y.Z"},
		{"版本区间反了", mod("P1", func(in *PopupInput) { in.MinVersion = "2.0.0"; in.MaxVersion = "1.0.0" }), "小于"},
		{"userType 非法", mod("P1", func(in *PopupInput) { in.UserType = "vip" }), "userType"},
		{"openMode 非法", mod("P1", func(in *PopupInput) { in.OpenMode = "tab" }), "openMode"},
		{"国家码非法", mod("P1", func(in *PopupInput) { in.Countries = []string{"PHL"} }), "国家码"},
		{"结束早于开始", mod("P1", func(in *PopupInput) {
			s := time.Now()
			e := s.Add(-time.Hour)
			in.StartAt, in.EndAt = &s, &e
		}), "晚于"},
		{"相对路径合法(webview)", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "/promo/deposit" }), ""},
		{"相对路径 browser 非法", mod("P1", func(in *PopupInput) { in.OpenMode = "browser"; in.Cards[0].LinkURL = "/promo" }), "webview"},
		{"协议相对 // 非法", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "//evil.com/x" }), "非法"},
		{"market webview 非法", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "market://details?id=a" }), "store"},
		{"market store 合法", mod("P1", func(in *PopupInput) { in.OpenMode = "store"; in.Cards[0].LinkURL = "market://details?id=a" }), ""},
		{"javascript 非法", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "javascript:alert(1)" }), "非法"},
		{"链接过长", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "https://a.com/" + strings.Repeat("x", 520) }), "512"},
		{"https 链接合法", mod("P1", func(in *PopupInput) { in.Cards[0].LinkURL = "https://a.com/x" }), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := normalizePopupInput(c.in)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("不应报错: %v", err)
				}
				if n == nil {
					t.Fatal("结果为空")
				}
				return
			}
			if err == nil {
				t.Fatalf("应报错包含 %q", c.wantErr)
			}
			if AsError(err).Code != 400 || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("错误 %v 应为 400 且包含 %q", err, c.wantErr)
			}
		})
	}
}

func TestNormalizePopupBehaviorMatrix(t *testing.T) {
	// 对每个位置塞满所有行为开关，看归一化后哪些被保留。
	type exp struct{ tab, countdown, mask, closable, badge bool }
	want := map[string]exp{
		"P1": {true, true, true, true, false},
		"P2": {false, false, false, true, true},
		"P3": {true, false, false, true, false},
		"P4": {true, false, true, false, false}, // 非强制时的保留见下方单独用例；此处 closable=false 即强制
		"P5": {false, false, false, true, false},
		"P6": {false, false, true, true, false},
		"P7": {true, true, false, true, false},
		"P8": {false, false, false, true, false},
	}
	for pos, w := range want {
		in := validInput(pos)
		in.TabEnabled, in.TabText = true, "Bonus"
		in.Countdown, in.MaskClosable, in.Badge = true, true, true
		in.Closable = bptr(false)
		n, err := normalizePopupInput(in)
		if err != nil {
			t.Fatalf("%s: %v", pos, err)
		}
		p := n.Popup
		got := exp{p.TabEnabled, p.Countdown, p.MaskClosable, p.Closable, p.Badge}
		if pos == "P4" {
			// 强制模式：tab / mask / countdown 全部强制 false。
			w = exp{false, false, false, false, false}
		}
		if got != w {
			t.Errorf("%s 归一化 got=%+v want=%+v", pos, got, w)
		}
		if !p.TabEnabled && (p.TabText != "" || p.TabIconURL != "") {
			t.Errorf("%s 便条关闭时文案/图标应被清空", pos)
		}
	}
	// P4 非强制：保留 tab / mask。
	in := validInput("P4")
	in.TabEnabled, in.TabText, in.MaskClosable = true, "Upd", true
	n, err := normalizePopupInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if !n.Popup.TabEnabled || !n.Popup.MaskClosable || !n.Popup.Closable {
		t.Errorf("P4 非强制应保留 tab/mask 且 closable=true: %+v", n.Popup)
	}
}

func TestNormalizePopupDefaultsAndFields(t *testing.T) {
	in := validInput("P3")
	in.Countries = []string{"ph", " PH ", "id"}
	in.BrandCodes = []string{"AP", "ap"}
	in.MinVersion, in.MaxVersion = "1.0.3", ""
	n, err := normalizePopupInput(in)
	if err != nil {
		t.Fatal(err)
	}
	p := n.Popup
	if p.AutoplaySeconds != 5 || p.ResumeGapMinutes != 30 {
		t.Errorf("默认值应为 5 / 30: %d %d", p.AutoplaySeconds, p.ResumeGapMinutes)
	}
	if len(p.Countries) != 2 || p.Countries[0] != "PH" || p.Countries[1] != "ID" {
		t.Errorf("国家码应大写去重: %v", p.Countries)
	}
	if len(p.BrandCodes) != 1 || p.BrandCodes[0] != "ap" {
		t.Errorf("品牌应小写去重: %v", p.BrandCodes)
	}
	if p.MinVersionCode != 10003 || p.MaxVersionCode != 0 {
		t.Errorf("版本换算错误: %d %d", p.MinVersionCode, p.MaxVersionCode)
	}
	if p.UserType != "all" || p.OpenMode != "webview" || !p.Closable {
		t.Errorf("默认 userType/openMode/closable 错误: %+v", p)
	}
	// 非 P3 的 autoplay 被归一化回 5，P1 的 resume 保留，非 P1 回 30。
	in2 := validInput("P1")
	in2.AutoplaySeconds, in2.ResumeGapMinutes = 20, 60
	n2, _ := normalizePopupInput(in2)
	if n2.Popup.AutoplaySeconds != 5 || n2.Popup.ResumeGapMinutes != 60 {
		t.Errorf("P1: autoplay 应回 5、resume 保留 60: %+v", n2.Popup)
	}
	// P5 不用图片：传了也清空。
	in5 := validInput("P5")
	in5.Cards[0].ImageURL = imgURL
	n5, _ := normalizePopupInput(in5)
	if n5.Cards[0].ImageURL != "" {
		t.Errorf("P5 应清空图片")
	}
}

// ---------- 卡片 id 合并 ----------

func TestMergePopupCards(t *testing.T) {
	existing := []model.PopupCard{{ID: 10}, {ID: 11}, {ID: 12}}
	in := []PopupCardInput{
		{ID: 12, ImageURL: "a"},   // 原地更新，顺序变第 0 位
		{ImageURL: "b"},           // 新建
		{ID: 10, ImageURL: "c"},   // 原地更新
		{ID: 999, ImageURL: "d"},  // 不属于该弹窗 → 当新建
		{ID: 12, ImageURL: "dup"}, // 重复 id → 第二次当新建
	}
	up, del := mergePopupCards(existing, in)
	if len(up) != 5 {
		t.Fatalf("upsert 数量 %d", len(up))
	}
	wantIDs := []uint64{12, 0, 10, 0, 0}
	for i, c := range up {
		if c.ID != wantIDs[i] || c.Sort != i {
			t.Errorf("第 %d 项 id=%d sort=%d，期望 id=%d sort=%d", i, c.ID, c.Sort, wantIDs[i], i)
		}
	}
	if len(del) != 1 || del[0] != 11 {
		t.Errorf("缺席的卡片 11 应被删除: %v", del)
	}
}

func TestUpdatePopupKeepsCardIDs(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	in := validInput("P1")
	in.Cards = []PopupCardInput{card(imgURL, ""), card(imgURL+"2", "")}
	v, err := svc.CreatePopup(ctx, full, in, "tester")
	if err != nil {
		t.Fatal(err)
	}
	id0, id1 := v.Cards[0].ID, v.Cards[1].ID
	// 交换顺序 + 删掉第 2 张 + 新增 1 张。
	in.Cards = []PopupCardInput{{ID: id1, ImageURL: imgURL + "2x"}, {ID: id0, ImageURL: imgURL}, card(imgURL+"3", "")}
	v2, err := svc.UpdatePopup(ctx, full, v.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(v2.Cards) != 3 || v2.Cards[0].ID != id1 || v2.Cards[1].ID != id0 || v2.Cards[2].ID == 0 {
		t.Fatalf("卡片应原地更新保持 id 并按新顺序: %+v", v2.Cards)
	}
	if v2.Cards[0].ImageURL != imgURL+"2x" {
		t.Errorf("卡片内容应已更新")
	}
	in.Cards = in.Cards[:1] // 只留 id1
	in.Cards[0].ID = id1
	v3, err := svc.UpdatePopup(ctx, full, v.ID, in)
	if err != nil || len(v3.Cards) != 1 || v3.Cards[0].ID != id1 {
		t.Fatalf("缺席的卡片应被删除: %v %+v", err, v3)
	}
}

// ---------- 状态 ----------

func TestPopupStatus(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	cases := []struct {
		name string
		p    model.Popup
		want string
	}{
		{"禁用优先于过期", model.Popup{Enabled: false, EndAt: &past}, "disabled"},
		{"已结束", model.Popup{Enabled: true, EndAt: &past}, "ended"},
		{"未开始", model.Popup{Enabled: true, StartAt: &future}, "scheduled"},
		{"生效", model.Popup{Enabled: true, StartAt: &past, EndAt: &future}, "active"},
		{"不限期", model.Popup{Enabled: true}, "active"},
	}
	for _, c := range cases {
		if got := popupStatus(&c.p, now); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

// ---------- 定向过滤 + configVersion ----------

func TestSelectPopupsForApp(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	posOn := map[string]bool{"P1": true, "P2": true, "P3": false}
	base := func(id uint64, pos string) model.Popup {
		return model.Popup{ID: id, Position: pos, Enabled: true, UserType: "all", OpenMode: "webview", Closable: true,
			Cards: []model.PopupCard{{ID: id * 10, Sort: 1, ImageURL: "img2"}, {ID: id*10 + 1, Sort: 0, ImageURL: "img1"}}}
	}
	mk := func(id uint64, pos string, f func(*model.Popup)) model.Popup {
		p := base(id, pos)
		f(&p)
		return p
	}
	popups := []model.Popup{
		base(1, "P1"),
		mk(2, "P1", func(p *model.Popup) { p.Enabled = false }),
		mk(3, "P3", func(p *model.Popup) {}), // 位置关
		mk(4, "P1", func(p *model.Popup) { p.EndAt = &past }),
		mk(5, "P1", func(p *model.Popup) { p.StartAt = &future }), // 未到 startAt 照常下发
		mk(6, "P1", func(p *model.Popup) { p.BrandCodes = model.StringList{"bp"} }),
		mk(7, "P1", func(p *model.Popup) { p.BrandCodes = model.StringList{"ap"} }),
		mk(8, "P1", func(p *model.Popup) { p.AppIDs = model.StringList{"com.other"} }),
		mk(9, "P1", func(p *model.Popup) { p.AppIDs = model.StringList{"com.arenaplus.ap01001"} }),
		mk(10, "P2", func(p *model.Popup) { p.Countries = model.StringList{"PH"} }),
		mk(11, "P2", func(p *model.Popup) { p.Countries = model.StringList{"ID"}; p.Priority = 50 }),
	}
	ids := func(items []PopupAppItem) []uint64 {
		var out []uint64
		for _, it := range items {
			out = append(out, it.ID)
		}
		return out
	}
	eq := func(a, b []uint64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	got := ids(selectPopupsForApp(popups, posOn, "ap", "com.arenaplus.ap01001", "PH", now))
	if want := []uint64{1, 5, 7, 9, 10}; !eq(got, want) {
		t.Errorf("PH: got %v want %v", got, want)
	}
	// 判不出国家：设了地区定向的弹窗不命中。
	got = ids(selectPopupsForApp(popups, posOn, "ap", "com.arenaplus.ap01001", "", now))
	if want := []uint64{1, 5, 7, 9}; !eq(got, want) {
		t.Errorf("未知国家: got %v want %v", got, want)
	}
	// 优先级降序：11 (50) 排最前。
	got = ids(selectPopupsForApp(popups, posOn, "bp", "x", "ID", now))
	if want := []uint64{11, 1, 5, 6}; !eq(got, want) {
		t.Errorf("bp/ID: got %v want %v", got, want)
	}
	// 卡片按 sort 排序、便条图标空则回填第一张卡片图。
	items := selectPopupsForApp(popups[:1], posOn, "ap", "a", "PH", now)
	if items[0].Cards[0].ImageURL != "img1" || items[0].TabIconURL != "img1" {
		t.Errorf("卡片应按 sort 排序且回填便条图标: %+v", items[0])
	}
}

func TestPopupConfigVersionStable(t *testing.T) {
	now := time.Now()
	mk := func() []model.Popup {
		return []model.Popup{{ID: 1, Position: "P1", Enabled: true, UserType: "all", OpenMode: "webview", Closable: true,
			Cards: []model.PopupCard{{ID: 1, ImageURL: "a", Title: "t"}}}}
	}
	pos := map[string]bool{"P1": true}
	v := func(ps []model.Popup) string {
		return popupConfigVersion(selectPopupsForApp(ps, pos, "ap", "x", "PH", now))
	}
	base := v(mk())
	if len(base) != 16 || base != v(mk()) {
		t.Fatalf("同内容应同值且 16 位: %s", base)
	}
	mutations := map[string]func(*model.Popup){
		"priority":  func(p *model.Popup) { p.Priority = 1 },
		"card":      func(p *model.Popup) { p.Cards[0].ImageURL = "b" },
		"title":     func(p *model.Popup) { p.Cards[0].Title = "u" },
		"tab":       func(p *model.Popup) { p.TabEnabled = true },
		"countdown": func(p *model.Popup) { p.Countdown = true },
		"version":   func(p *model.Popup) { p.MinVersionCode = 10003 },
		"closable":  func(p *model.Popup) { p.Closable = false },
	}
	for name, f := range mutations {
		ps := mk()
		f(&ps[0])
		if v(ps) == base {
			t.Errorf("改 %s 后 configVersion 应变化", name)
		}
	}
}

func TestAssemblePopupConfigEndToEnd(t *testing.T) {
	svc, r := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	appID := mustCreateChannel(t, svc, "ap", "ap01001")
	svc.SetCountryResolver(fakeGeo{"1.1.1.1": "PH", "2.2.2.2": "US"})

	in := validInput("P1")
	in.Countries = []string{"PH"}
	pv, err := svc.CreatePopup(ctx, full, in, "t")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := svc.PopupConfigForIP(ctx, appID, net.ParseIP("1.1.1.1"))
	if err != nil || len(cfg.Popups) != 1 || cfg.AppID != appID {
		t.Fatalf("PH 应命中: %v %+v", err, cfg)
	}
	if cfg.TzOffsetMinutes != 480 || cfg.ServerTime == 0 {
		t.Errorf("tzOffsetMinutes 应为 480（Asia/Manila）: %d", cfg.TzOffsetMinutes)
	}
	ver := cfg.ConfigVersion
	cfg2, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("1.1.1.1"))
	if cfg2.ConfigVersion != ver {
		t.Errorf("同内容 configVersion 应稳定")
	}
	if c, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("2.2.2.2")); len(c.Popups) != 0 {
		t.Errorf("US 不应命中 PH 定向")
	}
	if c, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("9.9.9.9")); len(c.Popups) != 0 {
		t.Errorf("GeoIP 判不出国家时地区定向弹窗不命中")
	}
	// 位置关闭 → 不下发；runtime-preview 与 App 端同 payload。
	if _, err := svc.SetPopupPosition(ctx, full, "P1", false, "t"); err != nil {
		t.Fatal(err)
	}
	if c, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("1.1.1.1")); len(c.Popups) != 0 {
		t.Errorf("位置关闭后不应下发")
	}
	got, _ := svc.GetPopup(ctx, full, pv.ID)
	if got.PositionEnabled {
		t.Errorf("positionEnabled 应回显 false")
	}
	_, _ = svc.SetPopupPosition(ctx, full, "P1", true, "t")
	pre, err := svc.PopupRuntimePreview(ctx, full, appID, "ph")
	if err != nil || len(pre.Popups) != 1 || pre.ConfigVersion != ver {
		t.Fatalf("runtime-preview 应与 App 端一致: %v %+v", err, pre)
	}
	// 弹窗关 / 软删 → 不下发。
	if _, err := svc.SetPopupEnabled(ctx, full, pv.ID, false); err != nil {
		t.Fatal(err)
	}
	if c, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("1.1.1.1")); len(c.Popups) != 0 {
		t.Errorf("弹窗关后不应下发")
	}
	_, _ = svc.SetPopupEnabled(ctx, full, pv.ID, true)
	if err := svc.DeletePopup(ctx, full, pv.ID); err != nil {
		t.Fatal(err)
	}
	if c, _ := svc.PopupConfigForIP(ctx, appID, net.ParseIP("1.1.1.1")); len(c.Popups) != 0 {
		t.Errorf("软删后不应下发")
	}
	// 未知 / 归档 appId → 404。
	if _, err := svc.PopupConfigForIP(ctx, "com.no.such", nil); AsError(err) == nil || AsError(err).Code != 404 {
		t.Errorf("未知 appId 应 404: %v", err)
	}
	if err := r.DB().Model(&model.Channel{}).Where("application_id = ?", appID).Update("status", model.ChannelArchived).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PopupConfigForIP(ctx, appID, nil); AsError(err) == nil || AsError(err).Code != 404 {
		t.Errorf("归档 appId 应 404: %v", err)
	}
}

// ---------- 埋点上报 ----------

func TestPopupStatKeyMapping(t *testing.T) {
	now := time.Now()
	loc := time.FixedZone("m", 8*3600)
	ms := func(t time.Time) int64 { return t.UnixMilli() }
	cases := []struct {
		name string
		ev   PopupEventIn
		want repo.PopupStatKey
		ok   bool
	}{
		{"impression auto", PopupEventIn{Event: "popup_impression", PopupID: 1, CardID: 34, CardIndex: 2, Display: "auto", TS: ms(now)},
			repo.PopupStatKey{PopupID: 1, CardID: 34, CardIndex: 2, Event: "popup_impression", Dim: "auto"}, true},
		{"click tab", PopupEventIn{Event: "popup_click", PopupID: 1, CardID: 34, CardIndex: 0, Display: "tab"},
			repo.PopupStatKey{PopupID: 1, CardID: 34, Event: "popup_click", Dim: "tab"}, true},
		{"filtered reason", PopupEventIn{Event: "popup_filtered", PopupID: 2, Reason: "mutex", CardID: 5},
			repo.PopupStatKey{PopupID: 2, Event: "popup_filtered", Dim: "mutex"}, true},
		{"close method", PopupEventIn{Event: "popup_close", PopupID: 1, Method: "back", CardID: 7},
			repo.PopupStatKey{PopupID: 1, Event: "popup_close", Dim: "back"}, true},
		{"slide card_index=to", PopupEventIn{Event: "popup_slide", PopupID: 1, From: 0, To: 3},
			repo.PopupStatKey{PopupID: 1, CardIndex: 3, Event: "popup_slide"}, true},
		{"trigger 无维度", PopupEventIn{Event: "popup_trigger", PopupID: 1, CardID: 9, CardIndex: 4},
			repo.PopupStatKey{PopupID: 1, Event: "popup_trigger"}, true},
		{"tab_click 无维度", PopupEventIn{Event: "tab_click", PopupID: 1},
			repo.PopupStatKey{PopupID: 1, Event: "tab_click"}, true},
		{"未知事件丢弃", PopupEventIn{Event: "popup_whatever", PopupID: 1}, repo.PopupStatKey{}, false},
		{"无 popupId 丢弃", PopupEventIn{Event: "popup_trigger"}, repo.PopupStatKey{}, false},
		{"未知 reason 归 other", PopupEventIn{Event: "popup_filtered", PopupID: 1, Reason: "zzz"},
			repo.PopupStatKey{PopupID: 1, Event: "popup_filtered", Dim: "other"}, true},
	}
	today := now.In(loc).Format("2006-01-02")
	for _, c := range cases {
		got, ok := popupStatKeyOf(c.ev, "1.0.3", now, loc)
		if ok != c.ok {
			t.Errorf("%s: ok=%v", c.name, ok)
			continue
		}
		if !ok {
			continue
		}
		c.want.StatDate, c.want.AppVersion = today, "1.0.3"
		if got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

func TestPopupStatDateByTZAndClamp(t *testing.T) {
	loc := time.FixedZone("m", 8*3600)
	now := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC) // 马尼拉已是 10-10 04:00
	ev := func(ts int64) PopupEventIn { return PopupEventIn{Event: "popup_trigger", PopupID: 1, TS: ts} }
	// ts 在 UTC 10-09 17:00 = 马尼拉 10-10 01:00 → 按时区换算为 10-10。
	k, _ := popupStatKeyOf(ev(time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC).UnixMilli()), "", now, loc)
	if k.StatDate != "2026-10-10" {
		t.Errorf("应按 POPUP_TZ 换算: %s", k.StatDate)
	}
	// 太旧 / 太新 → 用服务端接收时间（马尼拉 10-10）。
	for name, ts := range map[string]int64{
		"8天前":  now.Add(-8 * 24 * time.Hour).UnixMilli(),
		"2小时后": now.Add(2 * time.Hour).UnixMilli(),
		"0":    0,
	} {
		k, _ := popupStatKeyOf(ev(ts), "", now, loc)
		if k.StatDate != "2026-10-10" {
			t.Errorf("%s 应回落服务端时间: %s", name, k.StatDate)
		}
	}
	// 边界内的 6 天前保留。
	k, _ = popupStatKeyOf(ev(now.Add(-6*24*time.Hour).UnixMilli()), "", now, loc)
	if k.StatDate != "2026-10-04" {
		t.Errorf("6 天前应保留原日期: %s", k.StatDate)
	}
}

func sumStat(t *testing.T, r *repo.Repo, event, dim string) int64 {
	t.Helper()
	var n int64
	if err := r.DB().Model(&model.PopupStatDaily{}).Where("event = ? AND dim = ?", event, dim).
		Select("COALESCE(SUM(`count`),0)").Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIngestPopupEventsIdempotentAndMerge(t *testing.T) {
	svc, r := newPopupTestService(t)
	ctx := context.Background()
	appID := mustCreateChannel(t, svc, "ap", "ap01001")
	now := time.Now().UnixMilli()
	pv, err := svc.CreatePopup(ctx, auth.FullScope(), validInput("P1"), "t")
	if err != nil {
		t.Fatal(err)
	}
	cardID := pv.Cards[0].ID
	req := PopupEventsRequest{
		BatchID: "batch-1", AppID: appID, AppVersion: "1.0.3",
		Events: []PopupEventIn{
			{Event: "popup_impression", PopupID: pv.ID, CardID: cardID, CardIndex: 0, Display: "auto", TS: now},
			{Event: "popup_impression", PopupID: pv.ID, CardID: cardID, CardIndex: 0, Display: "auto", TS: now},
			{Event: "popup_close", PopupID: pv.ID, Method: "button", TS: now},
			{Event: "totally_unknown", PopupID: pv.ID, TS: now},
		},
	}
	res, err := svc.IngestPopupEvents(ctx, req)
	if err != nil || res.Accepted != 3 || res.Duplicate {
		t.Fatalf("首次上报: %v %+v", err, res)
	}
	if n := sumStat(t, r, "popup_impression", "auto"); n != 2 {
		t.Fatalf("同键应在内存合并后累加为 2: %d", n)
	}
	var rows int64
	r.DB().Model(&model.PopupStatDaily{}).Where("event = ?", "popup_impression").Count(&rows)
	if rows != 1 {
		t.Errorf("同键应只有一行: %d", rows)
	}
	// 同 batchId 再来一次：直接成功、不累加。
	res, err = svc.IngestPopupEvents(ctx, req)
	if err != nil || !res.Duplicate || res.Accepted != 0 {
		t.Fatalf("重复批次: %v %+v", err, res)
	}
	if n := sumStat(t, r, "popup_impression", "auto"); n != 2 {
		t.Errorf("重复批次不应累加: %d", n)
	}
	// 不同 batchId 同键：累加（count = count + ?）。
	req.BatchID = "batch-2"
	if _, err := svc.IngestPopupEvents(ctx, req); err != nil {
		t.Fatal(err)
	}
	if n := sumStat(t, r, "popup_impression", "auto"); n != 4 {
		t.Errorf("新批次应累加到 4: %d", n)
	}
	if n := sumStat(t, r, "popup_close", "button"); n != 2 {
		t.Errorf("close/button 应为 2: %d", n)
	}
	// 清理：7 天前的批次被删，新的保留。
	r.DB().Model(&model.PopupEventBatch{}).Where("batch_id = ?", "batch-1").
		Update("created_at", time.Now().Add(-8*24*time.Hour))
	if n, err := svc.PurgePopupEventBatches(ctx); err != nil || n != 1 {
		t.Errorf("应清理 1 条: %d %v", n, err)
	}
}

func TestIngestPopupEventsValidation(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	appID := mustCreateChannel(t, svc, "ap", "ap01001")
	many := make([]PopupEventIn, 101)
	cases := []struct {
		name string
		req  PopupEventsRequest
	}{
		{"缺 batchId", PopupEventsRequest{AppID: appID}},
		{"未知 appId", PopupEventsRequest{BatchID: "b", AppID: "com.no.such"}},
		{"超过 100 条", PopupEventsRequest{BatchID: "b", AppID: appID, Events: many}},
	}
	for _, c := range cases {
		if _, err := svc.IngestPopupEvents(ctx, c.req); AsError(err) == nil || AsError(err).Code != 400 {
			t.Errorf("%s 应 400: %v", c.name, err)
		}
	}
}

// ---------- 统计公式 ----------

func TestComputePopupStatsFormulas(t *testing.T) {
	row := func(date, ev, dim string, cardID uint64, idx int, n int64) repo.PopupStatRow {
		return repo.PopupStatRow{StatDate: date, PopupID: 12, CardID: cardID, CardIndex: idx, Event: ev, Dim: dim, Count: n}
	}
	rows := []repo.PopupStatRow{
		row("2026-10-01", "popup_trigger", "", 0, 0, 100),
		row("2026-10-02", "popup_trigger", "", 0, 0, 100),
		row("2026-10-01", "popup_impression", "auto", 34, 0, 80),
		row("2026-10-02", "popup_impression", "auto", 34, 0, 40),
		row("2026-10-01", "popup_impression", "tab", 34, 0, 20), // 便条重开：计入 idx0，不计 displays
		row("2026-10-01", "popup_impression", "auto", 35, 1, 90),
		row("2026-10-01", "popup_click", "auto", 34, 0, 12),
		row("2026-10-01", "popup_click", "auto", 35, 1, 6),
		row("2026-10-01", "popup_close", "button", 0, 0, 50),
		row("2026-10-01", "popup_close", "mask", 0, 0, 10),
		row("2026-10-01", "popup_close", "back", 0, 0, 20),
		row("2026-10-01", "popup_load_fail", "", 0, 0, 10),
		row("2026-10-01", "popup_collapse", "", 0, 0, 40),
		row("2026-10-01", "tab_impression", "", 0, 0, 50),
		row("2026-10-01", "tab_click", "", 0, 0, 10),
		row("2026-10-01", "tab_dismiss", "", 0, 0, 5),
		row("2026-10-01", "popup_slide", "", 0, 1, 7),
		row("2026-10-01", "popup_filtered", "frequency", 0, 0, 30),
		row("2026-10-01", "popup_filtered", "mutex", 0, 0, 3),
	}
	del := model.Popup{ID: 12, Name: "已删弹窗", Position: "P1", Cards: []model.PopupCard{{ID: 34, ImageURL: "u34"}}}
	del.DeletedAt.Valid = true
	items := computePopupStats(rows, map[uint64]model.Popup{12: del}, nil)
	if len(items) != 1 {
		t.Fatalf("应有 1 行: %d", len(items))
	}
	it := items[0]
	if it.Name != "已删弹窗" || !it.Deleted || it.Position != "P1" {
		t.Errorf("已删弹窗应带名字并标 deleted: %+v", it)
	}
	// displays = auto & idx0 = 80+40 = 120；impressions = 80+40+20+90 = 230；idx0 总 = 140。
	if it.Trigger != 200 || it.Displays != 120 || it.Impressions != 230 || it.Clicks != 18 || it.Closes != 80 {
		t.Errorf("汇总计数错误: %+v", it)
	}
	if it.Filtered.Frequency != 30 || it.Filtered.Mutex != 3 || it.CloseByMethod.Button != 50 || it.CloseByMethod.Back != 20 {
		t.Errorf("分布错误: %+v %+v", it.Filtered, it.CloseByMethod)
	}
	approx := func(name string, got, want float64) {
		if d := got - want; d > 1e-9 || d < -1e-9 {
			t.Errorf("%s got %v want %v", name, got, want)
		}
	}
	r := it.Rates
	approx("showRate", r.ShowRate, 120.0/200)
	approx("ctr", r.CTR, 18.0/230)
	approx("closeRate", r.CloseRate, 80.0/140)
	approx("loadFailRate", r.LoadFailRate, 10.0/200)
	approx("carouselDepth", r.CarouselDepth, 230.0/140)
	approx("recoveryRate", r.RecoveryRate, 10.0/40)
	approx("tabAbandonRate", r.TabAbandonRate, 5.0/50)
	approx("totalCtr", r.TotalCTR, 28.0/230)
	if len(it.Cards) != 2 || it.Cards[0].CardID != 34 || it.Cards[0].Impressions != 140 || it.Cards[0].Clicks != 12 ||
		it.Cards[0].ImageURL != "u34" || it.Cards[1].CardID != 35 {
		t.Errorf("卡片统计错误: %+v", it.Cards)
	}
	approx("card ctr", it.Cards[0].CTR, 12.0/140)
	if len(it.Daily) != 2 || it.Daily[0].Date != "2026-10-01" || it.Daily[0].Trigger != 100 || it.Daily[0].Displays != 80 ||
		it.Daily[0].Impressions != 190 || it.Daily[0].Clicks != 18 || it.Daily[1].Displays != 40 {
		t.Errorf("按日趋势错误: %+v", it.Daily)
	}
	// 分母为 0：全部比率为 0，不出现 NaN。
	z := computePopupStats(nil, map[uint64]model.Popup{7: {ID: 7, Name: "n"}}, []uint64{7})
	if len(z) != 1 || z[0].Rates != (PopupRates{}) {
		t.Errorf("分母为 0 时比率应全 0: %+v", z)
	}
}

func TestPopupStatsEndToEndWithScopeAndSoftDelete(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	apApp := mustCreateChannel(t, svc, "ap", "ap01001")
	bpApp := mustCreateChannel(t, svc, "bp", "bp01001")
	pv, err := svc.CreatePopup(ctx, full, validInput("P1"), "t")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	for i, app := range []string{apApp, bpApp, bpApp} {
		_, err := svc.IngestPopupEvents(ctx, PopupEventsRequest{
			BatchID: "b" + string(rune('a'+i)), AppID: app,
			Events: []PopupEventIn{{Event: "popup_trigger", PopupID: pv.ID, TS: now}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	get := func(scope auth.Scope, q PopupStatsQuery) PopupStatsItem {
		res, err := svc.PopupStats(ctx, scope, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Popups) != 1 {
			t.Fatalf("应有 1 行: %+v", res.Popups)
		}
		return res.Popups[0]
	}
	if it := get(full, PopupStatsQuery{}); it.Trigger != 3 {
		t.Errorf("全量 trigger=3: %d", it.Trigger)
	}
	if it := get(full, PopupStatsQuery{Brand: "bp"}); it.Trigger != 2 {
		t.Errorf("brand=bp trigger=2: %d", it.Trigger)
	}
	if it := get(full, PopupStatsQuery{AppID: apApp, Position: "P1"}); it.Trigger != 1 {
		t.Errorf("appId=ap trigger=1: %d", it.Trigger)
	}
	apScope := auth.Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: true}
	if it := get(apScope, PopupStatsQuery{}); it.Trigger != 1 {
		t.Errorf("ap 受限账号只统计 ap 的行: %d", it.Trigger)
	}
	// bp 数据对 ap 受限账号：带 appId=bp 过滤 → 交集为空 → 无行。
	res, _ := svc.PopupStats(ctx, apScope, PopupStatsQuery{AppID: bpApp})
	if len(res.Popups) != 0 {
		t.Errorf("越界 appId 不应有数据: %+v", res.Popups)
	}
	if res, _ := svc.PopupStats(ctx, full, PopupStatsQuery{Position: "P2"}); len(res.Popups) != 0 {
		t.Errorf("position 过滤应排除 P1 弹窗")
	}
	// 软删后统计仍能显示名称。
	if err := svc.DeletePopup(ctx, full, pv.ID); err != nil {
		t.Fatal(err)
	}
	it := get(full, PopupStatsQuery{})
	if it.Name != pv.Name || !it.Deleted {
		t.Errorf("软删弹窗统计应带名称并标 deleted: %+v", it)
	}
	// 跨度校验。
	if _, err := svc.PopupStats(ctx, full, PopupStatsQuery{From: "2026-01-01", To: "2026-12-31"}); AsError(err) == nil || AsError(err).Code != 400 {
		t.Errorf("跨度 >92 天应 400: %v", err)
	}
}

// ---------- 数据权限 ----------

func TestPopupDataScope(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	apApp := mustCreateChannel(t, svc, "ap", "ap01001")
	apApp2 := mustCreateChannel(t, svc, "ap", "ap01002")
	bpApp := mustCreateChannel(t, svc, "bp", "bp01001")

	brandAP := auth.Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: true}

	mk := func(scope auth.Scope, name string, brands, apps []string) (*PopupView, error) {
		in := validInput("P1")
		in.Name, in.BrandCodes, in.AppIDs = name, brands, apps
		return svc.CreatePopup(ctx, scope, in, "t")
	}
	allBrand, err := mk(full, "全品牌", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	apOnly, _ := mk(full, "仅ap", []string{"ap"}, nil)
	apBp, _ := mk(full, "ap+bp", []string{"ap", "bp"}, nil)
	_, _ = mk(full, "ap某包", []string{"ap"}, []string{apApp})
	bpPkg, _ := mk(full, "bp某包", nil, []string{bpApp})

	names := func(scope auth.Scope) map[string]bool {
		list, err := svc.ListPopups(ctx, scope, PopupListQuery{})
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]bool{}
		for _, v := range list {
			m[v.Name] = true
		}
		return m
	}
	// 品牌受限（ap）：ALL-match，只看到定向完全落在 ap 内的；空品牌（全品牌）与含 bp 的不可见。
	got := names(brandAP)
	if !got["仅ap"] || !got["ap某包"] || got["全品牌"] || got["ap+bp"] || got["bp某包"] {
		t.Errorf("品牌受限列表可见性错误: %v", got)
	}
	if len(names(full)) != 5 {
		t.Errorf("全量应看到 5 条")
	}
	// 单体越界 → 404。
	for _, id := range []uint64{allBrand.ID, apBp.ID, bpPkg.ID} {
		if _, err := svc.GetPopup(ctx, brandAP, id); AsError(err) == nil || AsError(err).Code != 404 {
			t.Errorf("越界详情应 404: id=%d err=%v", id, err)
		}
		if err := svc.DeletePopup(ctx, brandAP, id); AsError(err) == nil || AsError(err).Code != 404 {
			t.Errorf("越界删除应 404: id=%d err=%v", id, err)
		}
	}
	if _, err := svc.GetPopup(ctx, brandAP, apOnly.ID); err != nil {
		t.Errorf("范围内应可见: %v", err)
	}

	// 新建：品牌受限必须给非空且全在范围内的 brandCodes。
	for name, tc := range map[string]struct{ brands, apps []string }{
		"空品牌":     {nil, nil},
		"含越界品牌":   {[]string{"ap", "bp"}, nil},
		"越界品牌":    {[]string{"bp"}, nil},
		"品牌内但包越界": {[]string{"ap"}, []string{bpApp}},
	} {
		if _, err := mk(brandAP, name, tc.brands, tc.apps); AsError(err) == nil || AsError(err).Code != 403 {
			t.Errorf("%s 应 403: %v", name, err)
		}
	}
	if _, err := mk(brandAP, "合法", []string{"ap"}, nil); err != nil {
		t.Errorf("范围内新建应成功: %v", err)
	}
	// 更新越界定向 → 403。
	up := validInput("P1")
	up.BrandCodes = []string{"ap", "bp"}
	if _, err := svc.UpdatePopup(ctx, brandAP, apOnly.ID, up); AsError(err) == nil || AsError(err).Code != 403 {
		t.Errorf("更新为越界定向应 403: %v", err)
	}

	// 渠道受限（ap 品牌下仅 ap01001）：必须给非空 appIds 且全在范围内。
	var apCh model.Channel
	if err := svc.repo.DB().Where("application_id = ?", apApp).First(&apCh).Error; err != nil {
		t.Fatal(err)
	}
	chScope := auth.Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: false, ChannelIDs: map[uint64]bool{apCh.ID: true}}
	if _, err := mk(chScope, "渠道受限无包", []string{"ap"}, nil); AsError(err) == nil || AsError(err).Code != 403 {
		t.Errorf("渠道受限不给 appIds 应 403: %v", err)
	}
	if _, err := mk(chScope, "渠道受限越界包", []string{"ap"}, []string{apApp2}); AsError(err) == nil || AsError(err).Code != 403 {
		t.Errorf("渠道受限越界包应 403: %v", err)
	}
	if _, err := mk(chScope, "渠道受限合法", []string{"ap"}, []string{apApp}); err != nil {
		t.Errorf("渠道受限合法应成功: %v", err)
	}
	got = names(chScope)
	if !got["ap某包"] || !got["渠道受限合法"] || got["仅ap"] {
		t.Errorf("渠道受限列表可见性错误: %v", got)
	}
	// runtime-preview 越界 404。
	if _, err := svc.PopupRuntimePreview(ctx, brandAP, bpApp, ""); AsError(err) == nil || AsError(err).Code != 404 {
		t.Errorf("预览越界渠道应 404: %v", err)
	}

	// 位置开关：受限账号 403，全量可改。
	if _, err := svc.SetPopupPosition(ctx, brandAP, "P3", true, "t"); AsError(err) == nil || AsError(err).Code != 403 {
		t.Errorf("受限账号改位置开关应 403: %v", err)
	}
	if _, err := svc.SetPopupPosition(ctx, chScope, "P3", true, "t"); AsError(err) == nil || AsError(err).Code != 403 {
		t.Errorf("渠道受限账号改位置开关应 403: %v", err)
	}
	if p, err := svc.SetPopupPosition(ctx, full, "P3", true, "t"); err != nil || !p.Enabled {
		t.Errorf("全量账号应可改: %v", err)
	}
	if _, err := svc.SetPopupPosition(ctx, full, "P9", true, "t"); AsError(err) == nil || AsError(err).Code != 404 {
		t.Errorf("未知位置应 404: %v", err)
	}
	// 存在性：全量账号给不存在的包 / 品牌 → 400。
	if _, err := mk(full, "坏包", nil, []string{"com.no.such"}); AsError(err) == nil || AsError(err).Code != 400 {
		t.Errorf("不存在的渠道包应 400: %v", err)
	}
	if _, err := mk(full, "坏品牌", []string{"zz"}, nil); AsError(err) == nil || AsError(err).Code != 400 {
		t.Errorf("不存在的品牌应 400: %v", err)
	}
}

func TestListPopupsFilters(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	a := validInput("P1")
	a.Name, a.BrandCodes = "alpha", []string{"ap"}
	b := validInput("P2")
	b.Name, b.BrandCodes, b.Enabled = "beta", []string{"bp"}, false
	c := validInput("P1")
	c.Name = "gamma" // 全品牌
	for _, in := range []PopupInput{a, b, c} {
		if _, err := svc.CreatePopup(ctx, full, in, "t"); err != nil {
			t.Fatal(err)
		}
	}
	count := func(q PopupListQuery) int {
		l, err := svc.ListPopups(ctx, full, q)
		if err != nil {
			t.Fatal(err)
		}
		return len(l)
	}
	if n := count(PopupListQuery{}); n != 3 {
		t.Errorf("全部 3: %d", n)
	}
	if n := count(PopupListQuery{Position: "P1"}); n != 2 {
		t.Errorf("P1 应 2: %d", n)
	}
	if n := count(PopupListQuery{Brand: "ap"}); n != 2 { // alpha + 全品牌 gamma
		t.Errorf("brand=ap 应含全品牌弹窗共 2: %d", n)
	}
	if n := count(PopupListQuery{Status: "disabled"}); n != 1 {
		t.Errorf("disabled 应 1: %d", n)
	}
	if n := count(PopupListQuery{Keyword: "alp"}); n != 1 {
		t.Errorf("keyword 应 1: %d", n)
	}
}

func TestSeedPopupPositionsNotOverwrite(t *testing.T) {
	svc, r := newPopupTestService(t)
	ctx := context.Background()
	list, _ := svc.ListPopupPositions(ctx)
	if len(list) != 8 {
		t.Fatalf("应 seed 8 行: %d", len(list))
	}
	on := map[string]bool{}
	for _, p := range list {
		on[p.Code] = p.Enabled
	}
	for _, c := range []string{"P1", "P2", "P8"} {
		if !on[c] {
			t.Errorf("%s 默认应开", c)
		}
	}
	for _, c := range []string{"P3", "P4", "P5", "P6", "P7"} {
		if on[c] {
			t.Errorf("%s 默认应关", c)
		}
	}
	// 运营改过的开关，重启 seed 不覆盖。
	if _, err := svc.SetPopupPosition(ctx, auth.FullScope(), "P1", false, "ops"); err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePopupPositions(ctx, r.DB()); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.ListPopupPositions(ctx)
	for _, p := range list {
		if p.Code == "P1" && p.Enabled {
			t.Errorf("已存在的行不应被 seed 覆盖")
		}
	}
}

// ---------- 素材上传 ----------

func TestUploadPopupImage(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 120, 60))); err != nil {
		t.Fatal(err)
	}
	res, err := svc.UploadPopupImage(ctx, buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 120 || res.Height != 60 || res.Size != buf.Len() ||
		!strings.HasPrefix(res.Key, "popup/images/") || !strings.HasSuffix(res.Key, ".png") || res.URL == "" {
		t.Errorf("上传结果错误: %+v", res)
	}
	if _, err := svc.UploadPopupImage(ctx, []byte("not an image at all")); AsError(err) == nil || AsError(err).Code != 400 {
		t.Errorf("非图片应 400: %v", err)
	}
	if _, err := svc.UploadPopupImage(ctx, append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 3<<20)...)); AsError(err) == nil || AsError(err).Code != 400 {
		t.Errorf("超过 3MB 应 400: %v", err)
	}
}

// ---------- 评审修复:未知 popupId / 卡片归属 / 越权 / upsert / Save / displaysAll ----------

func TestIngestDropsUnknownPopupAndZeroesForeignCard(t *testing.T) {
	svc, r := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	appID := mustCreateChannel(t, svc, "ap", "ap01001")
	p1, _ := svc.CreatePopup(ctx, full, validInput("P1"), "t")
	p2, _ := svc.CreatePopup(ctx, full, validInput("P1"), "t")
	now := time.Now().UnixMilli()
	res, err := svc.IngestPopupEvents(ctx, PopupEventsRequest{BatchID: "x1", AppID: appID, Events: []PopupEventIn{
		{Event: "popup_trigger", PopupID: 99999, TS: now},                                                           // 未知弹窗:丢弃
		{Event: "popup_impression", PopupID: p1.ID, CardID: p2.Cards[0].ID, CardIndex: 2, Display: "auto", TS: now}, // 卡片属于别的弹窗:置 0
		{Event: "popup_impression", PopupID: p1.ID, CardID: p1.Cards[0].ID, CardIndex: 0, Display: "auto", TS: now},
	}})
	if err != nil || res.Accepted != 2 {
		t.Fatalf("应只接受 2 条: %v %+v", err, res)
	}
	var unknown int64
	r.DB().Model(&model.PopupStatDaily{}).Where("popup_id = ?", 99999).Count(&unknown)
	if unknown != 0 {
		t.Errorf("未知 popupId 不应入库")
	}
	var row model.PopupStatDaily
	if err := r.DB().Where("popup_id = ? AND card_index = 2", p1.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.CardID != 0 {
		t.Errorf("不属于该弹窗的 card_id 应置 0、card_index 保留: %+v", row)
	}
	// 软删后迟到的埋点仍入库(元数据含软删弹窗)。
	_ = svc.DeletePopup(ctx, full, p1.ID)
	res, _ = svc.IngestPopupEvents(ctx, PopupEventsRequest{BatchID: "x2", AppID: appID, Events: []PopupEventIn{
		{Event: "popup_trigger", PopupID: p1.ID, TS: now}}})
	if res.Accepted != 1 {
		t.Errorf("软删弹窗的迟到埋点应仍接受: %+v", res)
	}
}

func TestPopupStatsNoLeakAndNewFields(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	full := auth.FullScope()
	mustCreateChannel(t, svc, "ap", "ap01001")
	in := validInput("P1")
	in.Name, in.BrandCodes = "bp秘密", []string{"bp"}
	secret, err := svc.CreatePopup(ctx, full, in, "t")
	if err != nil {
		t.Fatal(err)
	}
	apScope := auth.Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: true}
	res, err := svc.PopupStats(ctx, apScope, PopupStatsQuery{PopupID: secret.ID})
	if err != nil || len(res.Popups) != 0 {
		t.Fatalf("越权 popupId 不应泄露名称: %v %+v", err, res.Popups)
	}
	// 全量账号可见零行,并带 tz / from / to。
	res, _ = svc.PopupStats(ctx, full, PopupStatsQuery{PopupID: secret.ID, From: "2026-10-01", To: "2026-10-03"})
	if len(res.Popups) != 1 || res.TZ != "Asia/Manila" || res.TzOffsetMinutes != 480 || res.From != "2026-10-01" || res.To != "2026-10-03" {
		t.Errorf("tz/from/to 或零行错误: %+v", res)
	}
	// 库里不存在的弹窗行不输出。
	rows := []repo.PopupStatRow{{StatDate: "2026-10-01", PopupID: 777, Event: "popup_trigger", Count: 5}}
	if out := computePopupStats(rows, map[uint64]model.Popup{}, nil); len(out) != 0 {
		t.Errorf("不存在的弹窗应跳过: %+v", out)
	}
}

func TestComputeDisplaysAll(t *testing.T) {
	rows := []repo.PopupStatRow{
		{StatDate: "d", PopupID: 1, CardID: 1, CardIndex: 0, Event: "popup_impression", Dim: "auto", Count: 10},
		{StatDate: "d", PopupID: 1, CardID: 1, CardIndex: 0, Event: "popup_impression", Dim: "tab", Count: 4},
		{StatDate: "d", PopupID: 1, CardID: 2, CardIndex: 1, Event: "popup_impression", Dim: "auto", Count: 9},
	}
	out := computePopupStats(rows, map[uint64]model.Popup{1: {ID: 1}}, nil)
	if out[0].Displays != 10 || out[0].DisplaysAll != 14 || out[0].Impressions != 23 {
		t.Errorf("displaysAll 应为 auto+tab 的 card_index=0 之和: %+v", out[0])
	}
}

func TestApplyPopupStatsMultiRowAccumulate(t *testing.T) {
	_, r := newPopupTestService(t)
	ctx := context.Background()
	counts := map[repo.PopupStatKey]int64{}
	for i := 0; i < 30; i++ {
		counts[repo.PopupStatKey{StatDate: "2026-10-01", PopupID: uint64(i%3 + 1), CardIndex: i, Event: "popup_trigger"}] = int64(i + 1)
	}
	for _, b := range []string{"b1", "b2"} {
		if dup, err := r.ApplyPopupStats(ctx, b, "app", counts); err != nil || dup {
			t.Fatal(dup, err)
		}
	}
	var total int64
	r.DB().Model(&model.PopupStatDaily{}).Select("SUM(`count`)").Scan(&total)
	if total != 2*(30*31/2) {
		t.Errorf("两批多行累加总和错误: %d", total)
	}
}

func TestUpdatePopupDoesNotResurrectDeleted(t *testing.T) {
	svc, r := newPopupTestService(t)
	ctx := context.Background()
	v, err := svc.CreatePopup(ctx, auth.FullScope(), validInput("P1"), "t")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := r.GetPopup(ctx, v.ID)
	if err := r.DeletePopup(ctx, v.ID); err != nil { // 模拟并发软删发生在读取之后
		t.Fatal(err)
	}
	p.Name = "复活?"
	err = r.SavePopup(ctx, p, func(e []model.PopupCard) ([]model.PopupCard, []uint64) { return nil, nil })
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("已软删弹窗保存应 ErrNotFound: %v", err)
	}
	if _, err := r.GetPopup(ctx, v.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("弹窗不应被复活")
	}
}

func TestPopupConfigCacheInvalidatedOnWrite(t *testing.T) {
	svc, _ := newPopupTestService(t)
	ctx := context.Background()
	appID := mustCreateChannel(t, svc, "ap", "ap01001")
	c0, _ := svc.AssemblePopupConfig(ctx, appID, "")
	pv, _ := svc.CreatePopup(ctx, auth.FullScope(), validInput("P1"), "t")
	c1, _ := svc.AssemblePopupConfig(ctx, appID, "")
	if len(c0.Popups) != 0 || len(c1.Popups) != 1 {
		t.Fatalf("写操作后缓存应失效: %d %d", len(c0.Popups), len(c1.Popups))
	}
	_, _ = svc.SetPopupEnabled(ctx, auth.FullScope(), pv.ID, false)
	if c2, _ := svc.AssemblePopupConfig(ctx, appID, ""); len(c2.Popups) != 0 {
		t.Errorf("关闭后应立即不下发")
	}
}

func TestNormalizeTabIconTooLong(t *testing.T) {
	in := validInput("P1")
	in.TabEnabled, in.TabText = true, "Hi"
	in.TabIconURL = "https://a.com/" + strings.Repeat("x", 520)
	if _, err := normalizePopupInput(in); err == nil || !strings.Contains(err.Error(), "512") {
		t.Errorf("tabIconUrl 超长应 400: %v", err)
	}
}
