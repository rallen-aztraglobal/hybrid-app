package seed

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/storage"
)

// newTestRepo 起一个 sqlite 内存库 + 已 seed 三品牌的 Repo。
func newTestRepo(t *testing.T) *repo.Repo {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	if err := EnsureBrands(context.Background(), db); err != nil {
		t.Fatalf("seed 品牌失败: %v", err)
	}
	return repo.New(db)
}

// writePNG 写一张纯色 PNG 到 path（自动建目录）。
func writePNG(t *testing.T, path string, px int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建目录失败: %v", err)
	}
	// 颜色随尺寸变化，保证不同密度文件字节不同（便于断言「取最大密度」）。
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	c := color.NRGBA{R: uint8(px % 256), G: 100, B: 200, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("编码 PNG 失败: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("写 PNG 失败: %v", err)
	}
}

// makeSeedLayout 在 root 下造「精简布局①」：root/channels/*.csv + root/res/<brand>/<flavor>/res/...
// 为 ap 造两个渠道：ap01018（全资源）、ap01034（仅 csv、无 res 目录，用于验证「无 res 跳过」）。
func makeSeedLayout(t *testing.T, root string) {
	t.Helper()
	csvDir := filepath.Join(root, "channels")
	if err := os.MkdirAll(csvDir, 0o755); err != nil {
		t.Fatalf("建 csv 目录失败: %v", err)
	}
	apCSV := "" +
		"# 注释行\n" +
		"ap01018|com.arenaplus.ap01018|PAL18|ArenaPlus One\n" +
		"ap01034|com.arenaplus.ap01034|PAL34|ArenaPlus Two\n"
	if err := os.WriteFile(filepath.Join(csvDir, "ap.csv"), []byte(apCSV), 0o644); err != nil {
		t.Fatalf("写 ap.csv 失败: %v", err)
	}
	// bp/gp 空 csv（ImportAllFromDir 会跳过缺失的，这里给空文件也可）。
	for _, b := range []string{"bp", "gp"} {
		if err := os.WriteFile(filepath.Join(csvDir, b+".csv"), []byte("# empty\n"), 0o644); err != nil {
			t.Fatalf("写 %s.csv 失败: %v", b, err)
		}
	}

	// ap01018 的 res：多密度 ic_launcher + splash。
	resBase := filepath.Join(root, "res", "ap", "ap01018", "res")
	writePNG(t, filepath.Join(resBase, "mipmap-xxxhdpi", "ic_launcher.png"), 192)
	writePNG(t, filepath.Join(resBase, "mipmap-hdpi", "ic_launcher.png"), 72)
	writePNG(t, filepath.Join(resBase, "drawable", "splash_fullscreen.png"), 256)
	// ap01034 故意不建 res 目录。
}

// TestInitChannelAssetsEndToEnd 覆盖：导入 → 资产初始化 → URL/磁盘文件/无res跳过/幂等。
func TestInitChannelAssetsEndToEnd(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	seedRoot := t.TempDir()
	makeSeedLayout(t, seedRoot)

	// 导入 CSV。
	csvDir := ResolveSeedCSVDir(seedRoot)
	if csvDir == "" {
		t.Fatal("未解析到 csv 目录")
	}
	if _, err := ImportAllFromDir(ctx, r, csvDir); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if n, _ := r.CountChannels(ctx); n != 2 {
		t.Fatalf("期望导入 2 渠道，实际 %d", n)
	}

	// 本地存储，public base 模拟生产 /static。
	diskRoot := t.TempDir()
	st, err := storage.NewLocal(diskRoot, "/static")
	if err != nil {
		t.Fatalf("建本地存储失败: %v", err)
	}

	resRoot := ResolveSeedResRoot(seedRoot)
	if resRoot == "" {
		t.Fatal("未解析到 res 根目录")
	}
	rep, err := InitChannelAssets(ctx, r, st, resRoot)
	if err != nil {
		t.Fatalf("资产初始化失败: %v", err)
	}

	// ap01034 无 res → 应被跳过。
	if rep.SkippedNoRes != 1 {
		t.Fatalf("期望跳过 1 个无 res 渠道，实际 %d", rep.SkippedNoRes)
	}
	if rep.Icons != 1 || rep.Splashes != 1 || rep.Zips != 1 {
		t.Fatalf("期望 icons/splashes/zips 各 1，实际 %d/%d/%d", rep.Icons, rep.Splashes, rep.Zips)
	}

	// ap01018 的字段应为 /static URL。
	ch, err := findChannelByAppID(ctx, r, "com.arenaplus.ap01018")
	if err != nil {
		t.Fatalf("查 ap01018 失败: %v", err)
	}
	wantIcon := "/static/icons/com.arenaplus.ap01018/icon.png"
	wantSplash := "/static/icons/com.arenaplus.ap01018/splash.png"
	wantZip := "/static/icons/com.arenaplus.ap01018/res.zip"
	if ch.IconMasterURL != wantIcon {
		t.Errorf("IconMasterURL=%q 期望 %q", ch.IconMasterURL, wantIcon)
	}
	if ch.SplashURL != wantSplash {
		t.Errorf("SplashURL=%q 期望 %q", ch.SplashURL, wantSplash)
	}
	if ch.IconSetURL != wantZip {
		t.Errorf("IconSetURL=%q 期望 %q", ch.IconSetURL, wantZip)
	}

	// 磁盘上确实落了文件。
	for _, rel := range []string{
		"icons/com.arenaplus.ap01018/icon.png",
		"icons/com.arenaplus.ap01018/splash.png",
		"icons/com.arenaplus.ap01018/res.zip",
	} {
		if !fileExists(filepath.Join(diskRoot, rel)) {
			t.Errorf("磁盘缺文件: %s", rel)
		}
	}

	// res.zip 内应含两张图标 + splash（共 3 条）。
	zipBytes, err := os.ReadFile(filepath.Join(diskRoot, "icons/com.arenaplus.ap01018/res.zip"))
	if err != nil {
		t.Fatalf("读 res.zip 失败: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("解析 res.zip 失败: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"mipmap-xxxhdpi/ic_launcher.png", "mipmap-hdpi/ic_launcher.png", "drawable/splash_fullscreen.png"} {
		if !names[want] {
			t.Errorf("res.zip 缺条目: %s（实有 %v）", want, names)
		}
	}

	// 代表性图标应取最大密度（xxxhdpi=192），与该密度源文件字节一致。
	gotIcon, _ := os.ReadFile(filepath.Join(diskRoot, "icons/com.arenaplus.ap01018/icon.png"))
	srcIcon, _ := os.ReadFile(filepath.Join(resRoot, "ap", "ap01018", "res", "mipmap-xxxhdpi", "ic_launcher.png"))
	if !bytes.Equal(gotIcon, srcIcon) {
		t.Errorf("图标未取最大密度源文件（字节不一致）")
	}

	// 幂等：再跑一次，不应再有字段变更（Processed=0），且文件数不变。
	rep2, err := InitChannelAssets(ctx, r, st, resRoot)
	if err != nil {
		t.Fatalf("二次资产初始化失败: %v", err)
	}
	if rep2.Processed != 0 {
		t.Errorf("幂等失败：二次运行 Processed=%d，期望 0", rep2.Processed)
	}
	// 仍应统计到 1 套资产（验证再次扫描到、但未写库）。
	if rep2.Icons != 1 || rep2.Zips != 1 {
		t.Errorf("二次运行资产计数异常：icons=%d zips=%d", rep2.Icons, rep2.Zips)
	}
}

// findChannelByAppID 测试辅助：按 appId 取渠道。
func findChannelByAppID(ctx context.Context, r *repo.Repo, appID string) (*model.Channel, error) {
	return r.GetChannelByApplicationID(ctx, appID)
}
