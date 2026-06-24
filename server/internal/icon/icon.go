// Package icon 实现「单张主图 → Android 全密度图标」的服务端 fan-out（ADR-0005、docs/admin/03 §2/§3）。
// 纯 Go（disintegration/imaging + image/draw），禁用 cgo，保持后端二进制全静态。
//
// 产出矩阵（与现有 app/src/channels/<brand>/<flavor>/res 结构字节级兼容）：
//
//	方形     mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher.png            48/72/96/144/192
//	圆形     mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher_round.png      圆形 alpha 遮罩
//	自适应前景 mipmap-{...}/ic_launcher_foreground.png             108..432，内容居中于 ~66% 安全区
//	自适应配置 mipmap-anydpi-v26/ic_launcher.xml(+round)            指向前景 + 背景
//	背景色   values/ic_launcher_background.xml                     纯色背景（默认白）
package icon

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"

	"github.com/disintegration/imaging"
)

// MasterSize 是约定的主图边长（与 App Store 一致，留足下采样空间）。
const MasterSize = 1024

// MinMasterSize 主图最小边长，低于此放大会糊（前端也应拦截，后端兜底）。
const MinMasterSize = 256

// foregroundSafeRatio 自适应前景内容安全区占比（外圈留边给系统裁切成圆/方/squircle）。
const foregroundSafeRatio = 0.66

// 各密度的方形图标边长（px）。
var squarePx = []struct {
	DPI string
	Px  int
}{
	{"mdpi", 48},
	{"hdpi", 72},
	{"xhdpi", 96},
	{"xxhdpi", 144},
	{"xxxhdpi", 192},
}

// 各密度的自适应前景画布边长（px）。
var foregroundPx = map[string]int{
	"mdpi":    108,
	"hdpi":    162,
	"xhdpi":   216,
	"xxhdpi":  324,
	"xxxhdpi": 432,
}

// Slot 标识图标矩阵里的一个槽位，供前端九宫格展示与单槽覆盖。
type Slot struct {
	Kind string // square / round / foreground
	DPI  string // mdpi..xxxhdpi
	Px   int    // 边长
	Path string // 在 res.zip 内的相对路径
}

// Slots 返回完整槽位清单（顺序稳定：先方形、再圆形、再前景，各按密度从小到大）。
func Slots() []Slot {
	out := make([]Slot, 0, len(squarePx)*3)
	for _, s := range squarePx {
		out = append(out, Slot{Kind: "square", DPI: s.DPI, Px: s.Px,
			Path: fmt.Sprintf("mipmap-%s/ic_launcher.png", s.DPI)})
	}
	for _, s := range squarePx {
		out = append(out, Slot{Kind: "round", DPI: s.DPI, Px: s.Px,
			Path: fmt.Sprintf("mipmap-%s/ic_launcher_round.png", s.DPI)})
	}
	for _, s := range squarePx {
		fg := foregroundPx[s.DPI]
		out = append(out, Slot{Kind: "foreground", DPI: s.DPI, Px: fg,
			Path: fmt.Sprintf("mipmap-%s/ic_launcher_foreground.png", s.DPI)})
	}
	return out
}

// GeneratedFile 是 fan-out 产出的单个文件（内存中）。
type GeneratedFile struct {
	Path        string // res 内相对路径
	ContentType string
	Data        []byte
}

// Options 控制 fan-out 行为。
type Options struct {
	// BackgroundHex 自适应图标背景色（#RRGGBB），默认白色。圆形遮罩外圈也用它兜底。
	BackgroundHex string
	// IncludeAdaptive 是否生成自适应前景 + anydpi-v26 xml + 背景色。默认 true。
	IncludeAdaptive bool
	// IncludeRound 是否生成圆形图标。默认 true。
	IncludeRound bool
}

// DefaultOptions 返回默认 fan-out 选项。
func DefaultOptions() Options {
	return Options{BackgroundHex: "#FFFFFF", IncludeAdaptive: true, IncludeRound: true}
}

// DecodeMaster 解码并规范化主图：校验最小尺寸、裁成正方形、缩放到 1024²。
// 返回规范化后的 1024² 图像，供后续 fan-out 复用。
func DecodeMaster(r io.Reader) (image.Image, error) {
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("解码主图失败（需 PNG/JPEG）: %w", err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < MinMasterSize || h < MinMasterSize {
		return nil, fmt.Errorf("主图尺寸过小（%dx%d），至少需 %d×%d", w, h, MinMasterSize, MinMasterSize)
	}
	// 非正方形 → 居中裁成正方形（前端通常已裁好，这里兜底）。
	if w != h {
		side := w
		if h < w {
			side = h
		}
		src = imaging.CropCenter(src, side, side)
	}
	// 统一缩放到 1024²，保证后续各密度都是下采样（更清晰）。
	master := imaging.Resize(src, MasterSize, MasterSize, imaging.Lanczos)
	return master, nil
}

// FanOut 由规范化主图生成全部密度文件。master 应为正方形（建议 1024²）。
func FanOut(master image.Image, opt Options) ([]GeneratedFile, error) {
	if opt.BackgroundHex == "" {
		opt.BackgroundHex = "#FFFFFF"
	}
	files := make([]GeneratedFile, 0, 20)

	// ① 方形 ic_launcher
	for _, s := range squarePx {
		sq := imaging.Resize(master, s.Px, s.Px, imaging.Lanczos)
		data, err := encodePNG(sq)
		if err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{
			Path:        fmt.Sprintf("mipmap-%s/ic_launcher.png", s.DPI),
			ContentType: "image/png", Data: data,
		})

		// ② 圆形 ic_launcher_round
		if opt.IncludeRound {
			round := applyCircleMask(sq)
			rdata, err := encodePNG(round)
			if err != nil {
				return nil, err
			}
			files = append(files, GeneratedFile{
				Path:        fmt.Sprintf("mipmap-%s/ic_launcher_round.png", s.DPI),
				ContentType: "image/png", Data: rdata,
			})
		}
	}

	// ③ 自适应前景 + ④ anydpi-v26 xml + 背景色
	if opt.IncludeAdaptive {
		for dpi, px := range foregroundPx {
			inner := int(float64(px) * foregroundSafeRatio)
			fgIcon := imaging.Resize(master, inner, inner, imaging.Lanczos)
			canvas := imaging.New(px, px, color.NRGBA{0, 0, 0, 0}) // 透明画布
			canvas = imaging.PasteCenter(canvas, fgIcon)           // 内容居中留安全区
			data, err := encodePNG(canvas)
			if err != nil {
				return nil, err
			}
			files = append(files, GeneratedFile{
				Path:        fmt.Sprintf("mipmap-%s/ic_launcher_foreground.png", dpi),
				ContentType: "image/png", Data: data,
			})
		}
		files = append(files,
			GeneratedFile{Path: "mipmap-anydpi-v26/ic_launcher.xml", ContentType: "application/xml", Data: []byte(adaptiveXML(false))},
			GeneratedFile{Path: "mipmap-anydpi-v26/ic_launcher_round.xml", ContentType: "application/xml", Data: []byte(adaptiveXML(true))},
			GeneratedFile{Path: "values/ic_launcher_background.xml", ContentType: "application/xml", Data: []byte(backgroundColorXML(opt.BackgroundHex))},
		)
	}
	return files, nil
}

// SplashFile 由源图生成 drawable/splash_fullscreen.png（全屏 CENTER_CROP 用单图，原样转码保真）。
func SplashFile(r io.Reader) (GeneratedFile, error) {
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("解码 splash 源图失败: %w", err)
	}
	data, err := encodePNG(src)
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{Path: "drawable/splash_fullscreen.png", ContentType: "image/png", Data: data}, nil
}

// PackZip 把若干 GeneratedFile 打成一个 res.zip（路径即 zip 内路径）。
func PackZip(files []GeneratedFile) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.Path)
		if err != nil {
			return nil, fmt.Errorf("写 zip 条目 %s 失败: %w", f.Path, err)
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, fmt.Errorf("写 zip 数据 %s 失败: %w", f.Path, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

// ---- 内部图像工具 ----

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("PNG 编码失败: %w", err)
	}
	return buf.Bytes(), nil
}

// applyCircleMask 给方形图标套圆形 alpha 遮罩（圆外透明）。用抗锯齿的软边缘。
func applyCircleMask(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2, float64(h)/2
	radius := float64(w) / 2
	// nrgba 化源，便于逐像素改 alpha。
	srcN := imaging.Clone(src)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			r := radius
			c := srcN.NRGBAAt(x, y)
			// 软边缘：在半径 ±1px 内做线性 alpha 过渡，避免锯齿。
			edge := r - 1
			d := math.Sqrt(dx*dx + dy*dy)
			switch {
			case d <= edge:
				// 完全保留
			case d >= r:
				c.A = 0
			default:
				factor := (r - d) / (r - edge)
				c.A = uint8(float64(c.A) * factor)
			}
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// adaptiveXML 生成 mipmap-anydpi-v26/ic_launcher(.round).xml。
func adaptiveXML(round bool) string {
	_ = round // 圆形与方形指向同一前景/背景，由系统按设备形状裁切。
	return `<?xml version="1.0" encoding="utf-8"?>
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@color/ic_launcher_background" />
    <foreground android:drawable="@mipmap/ic_launcher_foreground" />
    <monochrome android:drawable="@mipmap/ic_launcher_foreground" />
</adaptive-icon>
`
}

// backgroundColorXML 生成 values/ic_launcher_background.xml 颜色资源。
func backgroundColorXML(hex string) string {
	hex = normalizeHex(hex)
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<resources>
    <color name="ic_launcher_background">%s</color>
</resources>
`, hex)
}

func normalizeHex(hex string) string {
	if hex == "" {
		return "#FFFFFF"
	}
	if hex[0] != '#' {
		hex = "#" + hex
	}
	return hex
}

// DrawSolid 在测试/兜底场景下生成纯色方图（导出供测试使用）。
func DrawSolid(px int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}
