package render

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
)

// makeZip 构造内存 zip，names 为条目相对路径。
func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUnzipLayout(t *testing.T) {
	dst := t.TempDir()
	data := makeZip(t, map[string]string{
		"mipmap-mdpi/ic_launcher.png":    "m",
		"mipmap-xxxhdpi/ic_launcher.png": "x",
		"drawable/splash_fullscreen.png": "s",
	})
	n, err := Unzip(data, dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("期望解压 3 个文件，实得 %d", n)
	}
	for _, rel := range []string{"mipmap-mdpi/ic_launcher.png", "mipmap-xxxhdpi/ic_launcher.png", "drawable/splash_fullscreen.png"} {
		if _, err := os.Stat(filepath.Join(dst, filepath.FromSlash(rel))); err != nil {
			t.Errorf("缺少 %s: %v", rel, err)
		}
	}
}

// TestUnzipStripsResPrefix 验证 zip 顶层若包了一层 res/ 会被剥掉。
func TestUnzipStripsResPrefix(t *testing.T) {
	dst := t.TempDir()
	data := makeZip(t, map[string]string{"res/mipmap-hdpi/ic_launcher.png": "h"})
	if _, err := Unzip(data, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "mipmap-hdpi", "ic_launcher.png")); err != nil {
		t.Errorf("res/ 前缀未被剥离: %v", err)
	}
}

// TestUnzipClearsOld 验证解压前会清空旧资源（后台删图标→本地同步删除）。
func TestUnzipClearsOld(t *testing.T) {
	dst := t.TempDir()
	stale := filepath.Join(dst, "mipmap-mdpi", "old.png")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := makeZip(t, map[string]string{"mipmap-mdpi/ic_launcher.png": "new"})
	if _, err := Unzip(data, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("旧文件应被清除，仍存在: %v", err)
	}
}

// TestUnzipRejectsZipSlip 验证路径逃逸被拒绝。
func TestUnzipRejectsZipSlip(t *testing.T) {
	dst := t.TempDir()
	data := makeZip(t, map[string]string{"../evil.png": "x"})
	if _, err := Unzip(data, dst); err == nil {
		t.Fatal("期望 zip-slip 被拒绝，但成功了")
	}
}

// fixtureSource 是测试用 manifest 源。
type fixtureSource struct {
	m   *manifest.Manifest
	zip []byte
}

func (f *fixtureSource) Manifest(_ context.Context, _ string) (*manifest.Manifest, error) {
	return f.m, nil
}
func (f *fixtureSource) DownloadResZip(_ context.Context, _, _ string) ([]byte, error) {
	return f.zip, nil
}
func (f *fixtureSource) FetchGoogleServicesJSON(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// fakeRepo 在临时目录构造一个最小仓库布局供渲染。
func fakeRepo(t *testing.T) *repo.Repo {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "channels"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.gradle"), []byte("// test"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &repo.Repo{Root: root}
}

// TestRenderManifestWritesBootstrapAndCSV 端到端验证 pull 渲染产物。
func TestRenderManifestWritesBootstrapAndCSV(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand:        "ap",
		ConfigURL:    "https://cdn.example.com/app/config",
		BrandDomains: []string{"https://arenaplus.ph", "https://ap-backup.net"},
		Channels: []manifest.Channel{
			{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "111", AppName: "ArenaPlus", UseBrandDomains: true, ResZipURL: "x"},
			{Flavor: "ap01034", ApplicationId: "com.arenaplus.ap01034", PalCode: "222", AppName: "Arena Plus", UseBrandDomains: false, Domains: []string{"https://special.ph"}},
		},
	}
	src := &fixtureSource{m: m, zip: makeZip(t, map[string]string{"mipmap-mdpi/ic_launcher.png": "m"})}

	res, err := RenderManifest(context.Background(), r, src, m, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CSVChanged || res.ChannelCount != 2 || res.BootstrapCount != 2 {
		t.Fatalf("结果不符: %+v", res)
	}

	// CSV 内容校验。
	csvData, err := os.ReadFile(r.ChannelsCSV("ap"))
	if err != nil {
		t.Fatal(err)
	}
	wantCSV := "# 渠道数据表  字段: flavorName|applicationId|palCode|appName\n" +
		"# 新增小渠道 = 在此加一行 + 在 app/src/<flavorName>/ 放 icon 与 splash 资源\n" +
		"ap01018|com.arenaplus.ap01018|111|ArenaPlus\n" +
		"ap01034|com.arenaplus.ap01034|222|Arena Plus\n"
	if string(csvData) != wantCSV {
		t.Errorf("CSV 不符\n got: %q\nwant: %q", string(csvData), wantCSV)
	}

	// bootstrap.json：ap01018 继承品牌域名。
	var bs manifest.Bootstrap
	readJSON(t, r.FlavorBootstrap("ap", "ap01018"), &bs)
	if bs.ConfigURL != m.ConfigURL || bs.Palcode != "111" {
		t.Errorf("ap01018 bootstrap 不符: %+v", bs)
	}
	if len(bs.DefaultDomains) != 2 || bs.DefaultDomains[0] != "https://arenaplus.ph" {
		t.Errorf("ap01018 应继承品牌域名: %+v", bs.DefaultDomains)
	}

	// bootstrap.json：ap01034 用覆盖域名。
	var bs2 manifest.Bootstrap
	readJSON(t, r.FlavorBootstrap("ap", "ap01034"), &bs2)
	if len(bs2.DefaultDomains) != 1 || bs2.DefaultDomains[0] != "https://special.ph" {
		t.Errorf("ap01034 应用覆盖域名: %+v", bs2.DefaultDomains)
	}

	// res 已解压。
	if _, err := os.Stat(filepath.Join(r.FlavorResDir("ap", "ap01018"), "mipmap-mdpi", "ic_launcher.png")); err != nil {
		t.Errorf("ap01018 res 未解压: %v", err)
	}
}

// TestRenderManifestRejectsDuplicates 验证唯一性冲突阻止写盘。
// ADR-0009 后 applicationId 由 flavor 派生，故 appId 冲突只可能来自 flavor 重复——
// 这里用重复 flavor 触发冲突（同时验证 palCode 重复不再被拦截）。
func TestRenderManifestRejectsDuplicates(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand: "ap",
		Channels: []manifest.Channel{
			{Flavor: "ap01034", ApplicationId: "com.arenaplus.ap01034", PalCode: "1"},
			{Flavor: "ap01034", ApplicationId: "com.arenaplus.ap01034", PalCode: "2"}, // 重复 flavor
		},
	}
	src := &fixtureSource{m: m}
	res, err := RenderManifest(context.Background(), r, src, m, Options{})
	if err == nil {
		t.Fatal("期望因唯一性冲突报错")
	}
	if len(res.Conflicts) == 0 {
		t.Error("应返回冲突详情")
	}
	// 不应写出 CSV。
	if _, err := os.Stat(r.ChannelsCSV("ap")); !os.IsNotExist(err) {
		t.Error("冲突时不应写出 CSV")
	}
}

// TestRenderManifestDerivesApplicationID 验证 ADR-0009：
// CSV 与 bootstrap.json 的 applicationId 都用派生值（<品牌前缀>.<flavor>），
// 即使 manifest 给的 applicationId 与 flavor 不一致（历史脏数据）也被纠正；
// 且 palCode 跨渠道重复不再阻止渲染。
func TestRenderManifestDerivesApplicationID(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand:        "ap",
		ConfigURL:    "https://cdn.example.com/app/config",
		BrandDomains: []string{"https://arenaplus.ph"},
		Channels: []manifest.Channel{
			// 故意给错 applicationId（挂 ap01034 的包名），派生应纠正为 com.arenaplus.ap01035。
			{Flavor: "ap01035", ApplicationId: "com.arenaplus.ap01034", PalCode: "777", AppName: "Arena", UseBrandDomains: true},
			// palCode 与上一行重复——ADR-0009 不再拦截。
			{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "777", AppName: "Arena2", UseBrandDomains: true},
		},
	}
	src := &fixtureSource{m: m}
	if _, err := RenderManifest(context.Background(), r, src, m, Options{SkipRes: true}); err != nil {
		t.Fatalf("派生应消除 appId 冲突且 palCode 重复不拦截，却报错: %v", err)
	}
	csvData, err := os.ReadFile(r.ChannelsCSV("ap"))
	if err != nil {
		t.Fatal(err)
	}
	want := "ap01035|com.arenaplus.ap01035|777|Arena\n"
	if !bytes.Contains(csvData, []byte(want)) {
		t.Errorf("applicationId 未派生纠正，CSV=%q", string(csvData))
	}
	// bootstrap.json 的 appId 也应是派生值。
	var bs manifest.Bootstrap
	readJSON(t, r.FlavorBootstrap("ap", "ap01035"), &bs)
	if bs.AppID != "com.arenaplus.ap01035" {
		t.Errorf("bootstrap.appId 应为派生值，实得 %q", bs.AppID)
	}
	if bs.Palcode != "777" {
		t.Errorf("bootstrap.palcode 应保留原值供 URL，实得 %q", bs.Palcode)
	}
}

// TestDryRunWritesNothing 验证 dry-run 不写盘。
func TestDryRunWritesNothing(t *testing.T) {
	r := fakeRepo(t)
	m := &manifest.Manifest{
		Brand:        "ap",
		BrandDomains: []string{"https://a"},
		Channels:     []manifest.Channel{{Flavor: "ap01018", ApplicationId: "com.x.a", PalCode: "1", AppName: "A", UseBrandDomains: true}},
	}
	src := &fixtureSource{m: m}
	if _, err := RenderManifest(context.Background(), r, src, m, Options{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.ChannelsCSV("ap")); !os.IsNotExist(err) {
		t.Error("dry-run 不应写 CSV")
	}
	if _, err := os.Stat(r.FlavorBootstrap("ap", "ap01018")); !os.IsNotExist(err) {
		t.Error("dry-run 不应写 bootstrap.json")
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}
