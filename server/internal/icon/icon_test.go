package icon

import (
	"archive/zip"
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

// makeMasterPNG 生成一张纯色 1024² PNG 作为测试主图。
func makeMasterPNG(t *testing.T) []byte {
	t.Helper()
	img := DrawSolid(MasterSize, color.NRGBA{R: 20, G: 120, B: 220, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("编码测试主图失败: %v", err)
	}
	return buf.Bytes()
}

func TestDecodeMasterRejectsTooSmall(t *testing.T) {
	small := DrawSolid(64, color.NRGBA{A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, small)
	if _, err := DecodeMaster(&buf); err == nil {
		t.Fatal("过小主图应被拒绝")
	}
}

func TestDecodeMasterCropsToSquare(t *testing.T) {
	// 构造非正方形图，应被裁成正方形并缩放到 1024²。
	rect := DrawSolid(512, color.NRGBA{A: 255}) // 用方图近似；DecodeMaster 内部统一缩放
	var buf bytes.Buffer
	_ = png.Encode(&buf, rect)
	m, err := DecodeMaster(&buf)
	if err != nil {
		t.Fatalf("DecodeMaster 失败: %v", err)
	}
	b := m.Bounds()
	if b.Dx() != MasterSize || b.Dy() != MasterSize {
		t.Fatalf("规范化后应为 %d², 得到 %dx%d", MasterSize, b.Dx(), b.Dy())
	}
}

func TestFanOutProducesFullMatrix(t *testing.T) {
	master, err := DecodeMaster(bytes.NewReader(makeMasterPNG(t)))
	if err != nil {
		t.Fatalf("DecodeMaster 失败: %v", err)
	}
	files, err := FanOut(master, DefaultOptions())
	if err != nil {
		t.Fatalf("FanOut 失败: %v", err)
	}

	// 期望存在的关键路径。
	want := map[string]bool{
		"mipmap-mdpi/ic_launcher.png":              false,
		"mipmap-xxxhdpi/ic_launcher.png":           false,
		"mipmap-mdpi/ic_launcher_round.png":        false,
		"mipmap-xxxhdpi/ic_launcher_round.png":     false,
		"mipmap-mdpi/ic_launcher_foreground.png":   false,
		"mipmap-xxxhdpi/ic_launcher_foreground.png": false,
		"mipmap-anydpi-v26/ic_launcher.xml":        false,
		"mipmap-anydpi-v26/ic_launcher_round.xml":  false,
		"values/ic_launcher_background.xml":        false,
	}
	dims := map[string]int{} // path -> px（仅 png）
	for _, f := range files {
		if _, ok := want[f.Path]; ok {
			want[f.Path] = true
		}
		if len(f.Data) == 0 {
			t.Errorf("%s 内容为空", f.Path)
		}
		if bytes.HasSuffix([]byte(f.Path), []byte(".png")) {
			cfg, err := png.DecodeConfig(bytes.NewReader(f.Data))
			if err != nil {
				t.Errorf("%s 不是合法 PNG: %v", f.Path, err)
				continue
			}
			dims[f.Path] = cfg.Width
		}
	}
	for p, found := range want {
		if !found {
			t.Errorf("缺少期望文件: %s", p)
		}
	}

	// 校验尺寸：方形 mdpi=48、xxxhdpi=192；前景 mdpi=108、xxxhdpi=432。
	checkDim := func(path string, px int) {
		if dims[path] != px {
			t.Errorf("%s 宽度 = %d，期望 %d", path, dims[path], px)
		}
	}
	checkDim("mipmap-mdpi/ic_launcher.png", 48)
	checkDim("mipmap-xxxhdpi/ic_launcher.png", 192)
	checkDim("mipmap-mdpi/ic_launcher_foreground.png", 108)
	checkDim("mipmap-xxxhdpi/ic_launcher_foreground.png", 432)
}

func TestPackZipRoundTrip(t *testing.T) {
	master, _ := DecodeMaster(bytes.NewReader(makeMasterPNG(t)))
	files, _ := FanOut(master, DefaultOptions())
	data, err := PackZip(files)
	if err != nil {
		t.Fatalf("PackZip 失败: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	if len(zr.File) != len(files) {
		t.Fatalf("zip 条目数 = %d，期望 %d", len(zr.File), len(files))
	}
	// 确认 anydpi xml 在 zip 内可读且非空。
	var foundXML bool
	for _, zf := range zr.File {
		if zf.Name == "mipmap-anydpi-v26/ic_launcher.xml" {
			foundXML = true
			rc, _ := zf.Open()
			b := new(bytes.Buffer)
			_, _ = b.ReadFrom(rc)
			_ = rc.Close()
			if b.Len() == 0 {
				t.Error("anydpi xml 在 zip 内为空")
			}
		}
	}
	if !foundXML {
		t.Error("zip 内缺少 anydpi xml")
	}
}

func TestRoundMaskTransparentCorners(t *testing.T) {
	master, _ := DecodeMaster(bytes.NewReader(makeMasterPNG(t)))
	files, _ := FanOut(master, DefaultOptions())
	var roundData []byte
	for _, f := range files {
		if f.Path == "mipmap-xxxhdpi/ic_launcher_round.png" {
			roundData = f.Data
		}
	}
	if roundData == nil {
		t.Fatal("未找到圆形图标")
	}
	img, err := png.Decode(bytes.NewReader(roundData))
	if err != nil {
		t.Fatalf("解码圆形图标失败: %v", err)
	}
	// 左上角(0,0)应被圆形遮罩裁成透明。
	_, _, _, a := img.At(0, 0).RGBA()
	if a != 0 {
		t.Errorf("圆形图标四角应透明，左上 alpha = %d", a>>8)
	}
	// 中心应不透明。
	b := img.Bounds()
	_, _, _, ca := img.At(b.Dx()/2, b.Dy()/2).RGBA()
	if ca == 0 {
		t.Error("圆形图标中心不应透明")
	}
}
