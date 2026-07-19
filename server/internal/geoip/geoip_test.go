package geoip

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDbipURL(t *testing.T) {
	got := dbipURL(time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC))
	want := "https://download.db-ip.com/free/dbip-country-lite-2026-07.mmdb.gz"
	if got != want {
		t.Errorf("dbipURL = %s，期望 %s", got, want)
	}
	// 月份需补零。
	got = dbipURL(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	want = "https://download.db-ip.com/free/dbip-country-lite-2026-01.mmdb.gz"
	if got != want {
		t.Errorf("dbipURL 月份未补零: %s，期望 %s", got, want)
	}
}

// 库缺失是最关键的降级路径：必须安全返回「未知」，让网关判 A 面，
// 而不是 panic 或返回某个默认国家。
func TestResolverMissingDBDegradesSafely(t *testing.T) {
	r := New(filepath.Join(t.TempDir(), "nonexistent.mmdb"))
	if r.Loaded() {
		t.Fatal("库不存在时 Loaded() 应为 false")
	}
	if code, ok := r.Country(net.ParseIP("1.2.3.4")); ok || code != "" {
		t.Errorf("库缺失时应返回未知，实际 (%q, %v)", code, ok)
	}
	if !r.BuildTime().IsZero() {
		t.Error("库缺失时 BuildTime 应为零值")
	}
}

// 损坏的库同样要降级，不能因为文件存在就当它可用。
func TestResolverCorruptDBDegradesSafely(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.mmdb")
	if err := os.WriteFile(path, []byte("这不是一个 mmdb 文件"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := New(path)
	if r.Loaded() {
		t.Fatal("损坏的库不应被标记为已加载")
	}
	if _, ok := r.Country(net.ParseIP("1.2.3.4")); ok {
		t.Error("损坏的库应返回未知")
	}
}

func TestCountryNilIP(t *testing.T) {
	r := New(filepath.Join(t.TempDir(), "nonexistent.mmdb"))
	if _, ok := r.Country(nil); ok {
		t.Error("nil IP 应返回未知")
	}
}

// 端到端：真实下载 DB-IP 库并校验若干已知 IP 的归属。
// 依赖外网，故 -short 时跳过（CI 无外网时用 go test -short）。
func TestRefreshAndLookupRealDB(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要外网的真实库下载测试")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "dbip-country-lite.mmdb")
	r := New(path) // 首次：文件不存在，未加载

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := r.Refresh(ctx, time.Now()); err != nil {
		t.Fatalf("刷新真实 GeoIP 库失败: %v", err)
	}
	if !r.Loaded() {
		t.Fatal("刷新后应已加载")
	}
	if r.BuildTime().IsZero() {
		t.Error("刷新后 BuildTime 不应为零值")
	}
	// 文件应落盘，供下次启动直接加载。
	if _, err := os.Stat(path); err != nil {
		t.Errorf("刷新后库文件未落盘: %v", err)
	}

	// 几个归属稳定的公共 DNS / 保留地址。
	cases := []struct {
		ip   string
		want string
	}{
		{"8.8.8.8", "US"},         // Google DNS，解析为 US → 会被硬编码闸拦成 A 面
		{"114.114.114.114", "CN"}, // 国内 DNS，解析为 CN → 同样被硬编码闸拦下
		// Cloudflare 的 1.1.1.1 由 APNIC 分配、在 DB-IP 里登记为澳大利亚而非美国。
		// 保留此用例是为了覆盖「非强制 A 面国家」的正常解析路径。
		{"1.1.1.1", "AU"},
	}
	for _, tc := range cases {
		got, ok := r.Country(net.ParseIP(tc.ip))
		if !ok {
			t.Errorf("%s 未解析出国家", tc.ip)
			continue
		}
		if got != tc.want {
			t.Errorf("%s 解析为 %s，期望 %s", tc.ip, got, tc.want)
		}
	}

	// 私有地址查不到国家，应安全返回未知。
	if _, ok := r.Country(net.ParseIP("192.168.1.1")); ok {
		t.Error("私有地址应返回未知国家")
	}
}

// 热替换：刷新期间与刷新后，已有 Resolver 引用应始终可用，不出现空窗。
func TestRefreshHotSwapKeepsResolverUsable(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要外网的真实库下载测试")
	}
	dir := t.TempDir()
	r := New(filepath.Join(dir, "db.mmdb"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := r.Refresh(ctx, time.Now()); err != nil {
		t.Fatalf("首次刷新失败: %v", err)
	}
	first, _ := r.Country(net.ParseIP("8.8.8.8"))

	// 再刷一次（模拟 cron 月度更新），期间查询不应失败。
	if err := r.Refresh(ctx, time.Now()); err != nil {
		t.Fatalf("二次刷新失败: %v", err)
	}
	second, ok := r.Country(net.ParseIP("8.8.8.8"))
	if !ok {
		t.Fatal("热替换后查询失败")
	}
	if first != second {
		t.Errorf("热替换前后结果不一致: %s → %s", first, second)
	}
}
