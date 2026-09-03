package csvio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hybrid-app/cli/internal/manifest"
)

// findRepoRoot 向上查找仓库根（含 settings.gradle + channels/），供测试读取真实 CSV。
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "settings.gradle")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "channels")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("未找到仓库根，跳过基于真实 CSV 的测试")
		}
		dir = parent
	}
}

// TestRoundTripByteCompatible 是核心兼容性保证：
// 对现有每份 channels/*.csv 执行 Parse → Render，结果必须与原文件逐字节一致。
// 这确保 CLI 重写 CSV 不会引入任何 git 噪声，满足 ADR-0004「字节级兼容」。
func TestRoundTripByteCompatible(t *testing.T) {
	root := findRepoRoot(t)
	for _, brand := range []string{"ap", "bp", "gp"} {
		path := filepath.Join(root, "channels", brand+".csv")
		orig, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取 %s: %v", path, err)
		}
		f := Parse(orig)
		got := Render(f.Header, f.Rows)
		if !bytes.Equal(orig, got) {
			t.Errorf("%s round-trip 不一致\n--- 原始(%d B) ---\n%q\n--- 渲染(%d B) ---\n%q",
				brand, len(orig), string(orig), len(got), string(got))
		}
	}
}

// TestHeaderPreserved 验证注释头被完整保留（两行）。
func TestHeaderPreserved(t *testing.T) {
	root := findRepoRoot(t)
	f, err := ReadFile(filepath.Join(root, "channels", "ap.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Header) != 2 {
		t.Fatalf("期望 2 行注释头，实得 %d: %v", len(f.Header), f.Header)
	}
	if f.Header[0] != "# 渠道数据表  字段: flavorName|applicationId|palCode|appName" {
		t.Errorf("注释头首行不符: %q", f.Header[0])
	}
}

// TestParseSkipsCommentsAndShortLines 复刻 Gradle loadChannels 的过滤规则。
func TestParseSkipsCommentsAndShortLines(t *testing.T) {
	in := []byte("# c1\n# c2\nflavorA|com.x.a|111|App A\n\n# mid\nbad|only|three\nflavorB|com.x.b|222|App B\n")
	f := Parse(in)
	if len(f.Rows) != 2 {
		t.Fatalf("期望 2 行数据，实得 %d", len(f.Rows))
	}
	if f.Rows[0].Flavor != "flavorA" || f.Rows[1].ApplicationId != "com.x.b" {
		t.Errorf("解析结果错误: %+v", f.Rows)
	}
	// 首个数据行之前只有 c1/c2 计入头。
	if len(f.Header) != 2 {
		t.Errorf("头行数应为 2，实得 %d: %v", len(f.Header), f.Header)
	}
}

// TestValidateDetectsDuplicateAppID 验证唯一性校验能抓住 applicationId 重复
// （CLAUDE.md 护栏 5 / ADR-0009：applicationId 是唯一标识，重复会导致两个渠道包互相覆盖安装）。
// 样本取自历史上真实出现过的脏数据：ap01035 曾与 ap01034 共用 com.arenaplus.ap01034
// （359000d 引入、66979a1 修正）。这里内联样本而不读 channels/*.csv——真实 CSV 是会变的
// 业务数据，拿它当断言样本会在数据被修正时假性变红。
func TestValidateDetectsDuplicateAppID(t *testing.T) {
	rows := []Row{
		{"ap01034", "com.arenaplus.ap01034", "1053259243279695873", "Arena Plus"},
		{"ap01035", "com.arenaplus.ap01034", "1053259242391433216", "Arena Plus"},
	}
	conflicts := Validate(rows)
	if len(conflicts) != 1 {
		t.Fatalf("应恰好报出 1 处冲突，实得: %v", conflicts)
	}
	c := conflicts[0]
	if c.Field != "applicationId" || c.Value != "com.arenaplus.ap01034" {
		t.Errorf("冲突字段/值不符，实得: %v", c)
	}
	if len(c.Flavors) != 2 || c.Flavors[0] != "ap01034" || c.Flavors[1] != "ap01035" {
		t.Errorf("冲突应列出两个 flavor（升序），实得: %v", c.Flavors)
	}
}

// TestValidateDetectsDuplicateFlavor 验证 flavor 维度同样查重（Validate 的另一半约束）。
func TestValidateDetectsDuplicateFlavor(t *testing.T) {
	rows := []Row{
		{"ap01034", "com.arenaplus.ap01034", "111", "Arena Plus"},
		{"ap01034", "com.arenaplus.dup", "222", "Arena Plus"},
	}
	conflicts := Validate(rows)
	if len(conflicts) != 1 || conflicts[0].Field != "flavor" || conflicts[0].Value != "ap01034" {
		t.Errorf("应报出 flavor 重复 ap01034，实得: %v", conflicts)
	}
}

// TestValidateClean 干净数据应无冲突。
func TestValidateClean(t *testing.T) {
	rows := []Row{
		{"a", "com.x.a", "1", "A"},
		{"b", "com.x.b", "2", "B"},
	}
	if c := Validate(rows); len(c) != 0 {
		t.Errorf("干净数据不应有冲突: %v", c)
	}
}

// TestValidateAllowsDuplicatePalCode 验证 ADR-0009：palCode 跨渠道重复不再视为冲突
// （flavor 与 applicationId 各自唯一即通过）。
func TestValidateAllowsDuplicatePalCode(t *testing.T) {
	rows := []Row{
		{"a", "com.x.a", "777", "A"},
		{"b", "com.x.b", "777", "B"}, // 同 palCode，不同 flavor/appId
	}
	if c := Validate(rows); len(c) != 0 {
		t.Errorf("palCode 重复不应再被拦截（ADR-0009）: %v", c)
	}
}

// TestRowsFromChannelsDerived 验证派生 applicationId（<品牌前缀>.<flavor>）。
func TestRowsFromChannelsDerived(t *testing.T) {
	chs := []manifest.Channel{
		{Flavor: "ap01035", ApplicationId: "com.arenaplus.ap01034", PalCode: "1", AppName: "X"}, // 故意给错
		{Flavor: "gpgzmkk042", ApplicationId: "", PalCode: "2", AppName: "Y"},
	}
	apRows := RowsFromChannelsDerived("ap", chs[:1])
	if apRows[0].ApplicationId != "com.arenaplus.ap01035" {
		t.Errorf("ap 派生错误: %q", apRows[0].ApplicationId)
	}
	gpRows := RowsFromChannelsDerived("gp", chs[1:])
	if gpRows[0].ApplicationId != "com.gamezone.gpgzmkk042" {
		t.Errorf("gp 派生错误: %q", gpRows[0].ApplicationId)
	}
	// 未知品牌无法派生时回退到 manifest 给定值。
	fallback := RowsFromChannelsDerived("zz", []manifest.Channel{{Flavor: "x", ApplicationId: "com.given.x"}})
	if fallback[0].ApplicationId != "com.given.x" {
		t.Errorf("未知品牌应回退 manifest 值: %q", fallback[0].ApplicationId)
	}
}

// TestRenderExactFormat 验证单行渲染格式（单管道、无多余空格、行尾换行）。
func TestRenderExactFormat(t *testing.T) {
	rows := []Row{{"ap01018", "com.arenaplus.ap01018", "1053259232660520961", "ArenaPlus:USA Basketball Live"}}
	got := Render([]string{"# h"}, rows)
	want := "# h\nap01018|com.arenaplus.ap01018|1053259232660520961|ArenaPlus:USA Basketball Live\n"
	if string(got) != want {
		t.Errorf("渲染格式不符\n got: %q\nwant: %q", string(got), want)
	}
}
