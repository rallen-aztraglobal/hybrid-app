package render

import (
	"os"
	"testing"

	"github.com/hybrid-app/cli/internal/repo"
)

// 带标识 → 每个 flavor 写入原样一行；下个任务不带 → 残留被清掉，且不碰同目录的 bootstrap.json。
func TestSyncADIRegistration(t *testing.T) {
	r := &repo.Repo{Root: t.TempDir()}
	flavors := []string{"bpom3410", "bpom3411"}

	if err := SyncADIRegistration(r, "bp", flavors, " C7V2CUSU2IP4YAAAAAAAAAAAAA\n", Options{}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	for _, f := range flavors {
		got, err := os.ReadFile(r.FlavorADIRegistration("bp", f))
		if err != nil {
			t.Fatalf("%s 未写出注册文件: %v", f, err)
		}
		if string(got) != "C7V2CUSU2IP4YAAAAAAAAAAAAA" {
			t.Errorf("%s 内容应为 trim 后的标识，实际 %q", f, got)
		}
	}

	bootstrap := r.FlavorBootstrap("bp", "bpom3410")
	if err := os.WriteFile(bootstrap, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SyncADIRegistration(r, "bp", flavors, "", Options{}); err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	for _, f := range flavors {
		if _, err := os.Stat(r.FlavorADIRegistration("bp", f)); !os.IsNotExist(err) {
			t.Errorf("%s 的残留注册文件应被清理", f)
		}
	}
	if _, err := os.Stat(bootstrap); err != nil {
		t.Errorf("清理不应动 bootstrap.json: %v", err)
	}
	// 本来就没有文件（甚至没有 assets 目录）时再清一次也不报错。
	if err := SyncADIRegistration(r, "bp", []string{"never_built"}, "", Options{}); err != nil {
		t.Errorf("无残留时清理不应报错: %v", err)
	}
}
