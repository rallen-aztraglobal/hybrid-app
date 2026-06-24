// Package render 把后台 manifest「渲染回现有 Gradle 认识的输入文件」（ADR-0004）：
//
//	channels/<brand>.csv                                  ← 字节级兼容重写
//	app/src/channels/<brand>/<flavor>/res/...             ← 解压 res.zip
//	app/src/channels/<brand>/<flavor>/assets/bootstrap.json ← 域名兜底 + 配置端点
//
// 绝不修改 app/build.gradle 的任何机制；本包只生产它已经会读取的文件。
package render

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/csvio"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
)

// Options 控制渲染行为。
type Options struct {
	// DryRun 为 true 时不写盘，只计算变更并通过 Logf 汇报。
	DryRun bool
	// SkipRes 跳过 res 资源下载/解压（仅刷新 CSV 与 bootstrap.json）。
	SkipRes bool
	// Logf 进度回调（可为 nil）。
	Logf func(format string, args ...any)
}

func (o Options) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

// Result 汇总一次渲染产生的变更。
type Result struct {
	Brand          string
	CSVChanged     bool
	ChannelCount   int
	ResWritten     int // 成功解压资源的 flavor 数
	ResSkipped     int // 无资源地址 / 跳过的 flavor 数
	BootstrapCount int
	Conflicts      []csvio.Conflict
}

// Pull 渲染单个品牌的 manifest 到本地文件。
func Pull(ctx context.Context, r *repo.Repo, src api.ManifestSource, brand string, opt Options) (*Result, error) {
	m, err := src.Manifest(ctx, brand)
	if err != nil {
		return nil, fmt.Errorf("拉取 %s manifest 失败: %w", brand, err)
	}
	return RenderManifest(ctx, r, src, m, opt)
}

// RenderManifest 把已获取的 manifest 渲染到本地。
func RenderManifest(ctx context.Context, r *repo.Repo, src api.ManifestSource, m *manifest.Manifest, opt Options) (*Result, error) {
	res := &Result{Brand: m.Brand, ChannelCount: len(m.Channels)}

	// 1) 唯一性校验：拦截 applicationId / flavor 重复（CLAUDE.md 护栏 5 / ADR-0009：
	//    palCode 不再全局唯一，不参与查重）。CSV 行的 applicationId 统一用派生值（ADR-0009）。
	rows := csvio.RowsFromChannelsDerived(m.Brand, m.Channels)
	if conflicts := csvio.Validate(rows); len(conflicts) > 0 {
		res.Conflicts = conflicts
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("品牌 %s 的渠道数据存在唯一性冲突，已拒绝渲染：\n", m.Brand))
		for _, c := range conflicts {
			sb.WriteString("  - " + c.String() + "\n")
		}
		return res, fmt.Errorf("%s", sb.String())
	}

	// 2) 渲染 CSV（保留现有注释头，字节级兼容）。
	csvPath := r.ChannelsCSV(m.Brand)
	existing, err := csvio.ReadFile(csvPath)
	if err != nil {
		return res, err
	}
	header := existing.Header // 复用现有注释头（首次则为默认头）
	newData := csvio.Render(header, rows)
	oldData, _ := os.ReadFile(csvPath)
	res.CSVChanged = !bytes.Equal(oldData, newData)

	if res.CSVChanged {
		if opt.DryRun {
			opt.logf("  [dry-run] 将重写 %s（%d 渠道）", rel(r, csvPath), len(rows))
		} else {
			if err := csvio.WriteFile(csvPath, header, rows); err != nil {
				return res, err
			}
			opt.logf("  已重写 %s（%d 渠道）", rel(r, csvPath), len(rows))
		}
	} else {
		opt.logf("  %s 无变化（%d 渠道）", rel(r, csvPath), len(rows))
	}

	// 3) 逐 flavor：解压 res + 写 bootstrap.json。
	for _, ch := range m.Channels {
		// bootstrap.json：编译期兜底（ADR-0002）+ appId 解析键（ADR-0009）。
		// AppID 统一用派生值（与 CSV/BuildConfig.APPLICATION_ID 一致）；派生失败时回退到 manifest 给定值。
		appID := manifest.DeriveApplicationID(m.Brand, ch.Flavor)
		if appID == "" {
			appID = ch.ApplicationId
		}
		bs := manifest.Bootstrap{
			AppID:          appID,
			ConfigURL:      m.ConfigURL,
			Palcode:        ch.PalCode,
			DefaultDomains: ch.EffectiveDomains(m.BrandDomains),
		}
		if err := writeBootstrap(r, m.Brand, ch.Flavor, bs, opt); err != nil {
			return res, err
		}
		res.BootstrapCount++

		// res 资源。
		if opt.SkipRes || ch.ResZipURL == "" {
			res.ResSkipped++
			if ch.ResZipURL == "" {
				opt.logf("  %s 无 res.zip 地址，跳过资源", ch.Flavor)
			}
			continue
		}
		data, err := src.DownloadResZip(ctx, ch.ResZipURL, ch.ResZipSHA256)
		if err != nil {
			return res, fmt.Errorf("下载 %s 资源失败: %w", ch.Flavor, err)
		}
		if len(data) == 0 {
			res.ResSkipped++
			opt.logf("  %s 资源为空，跳过", ch.Flavor)
			continue
		}
		dst := r.FlavorResDir(m.Brand, ch.Flavor)
		if opt.DryRun {
			opt.logf("  [dry-run] 将解压 %d 字节 res.zip → %s", len(data), rel(r, dst))
		} else {
			n, err := Unzip(data, dst)
			if err != nil {
				return res, fmt.Errorf("解压 %s 资源失败: %w", ch.Flavor, err)
			}
			opt.logf("  %s 资源已更新（%d 个文件）", ch.Flavor, n)
		}
		res.ResWritten++
	}

	return res, nil
}

func writeBootstrap(r *repo.Repo, brand, flavor string, bs manifest.Bootstrap, opt Options) error {
	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 bootstrap.json 失败: %w", err)
	}
	data = append(data, '\n')
	path := r.FlavorBootstrap(brand, flavor)
	if opt.DryRun {
		opt.logf("  [dry-run] 将写 %s（%d 域名）", rel(r, path), len(bs.DefaultDomains))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 assets 目录失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("写 %s 失败: %w", path, err)
	}
	return nil
}

// Unzip 把 zip 字节安全解压到 dst 目录（先清空旧 res），返回写出的文件数。
//
// 防 zip-slip：拒绝任何会逃逸出 dst 的条目路径。
func Unzip(data []byte, dst string) (int, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("打开 zip 失败: %w", err)
	}
	// 先清空旧资源，确保「后台删了某档图标，本地也随之删除」。
	if err := os.RemoveAll(dst); err != nil {
		return 0, fmt.Errorf("清理旧资源失败: %w", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, fmt.Errorf("创建资源目录失败: %w", err)
	}
	cleanDst := filepath.Clean(dst)
	count := 0
	for _, f := range zr.File {
		// 归一化 zip 内路径分隔符，再按平台拼接。
		name := strings.ReplaceAll(f.Name, "\\", "/")
		// 容忍 zip 顶层包了一层「res/」目录：剥掉它，直接落到 dst（=.../res）。
		name = strings.TrimPrefix(name, "res/")
		if name == "" {
			continue
		}
		target := filepath.Join(cleanDst, filepath.FromSlash(name))
		if target != cleanDst && !strings.HasPrefix(target, cleanDst+string(os.PathSeparator)) {
			return count, fmt.Errorf("非法 zip 条目（路径逃逸）: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return count, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, err
		}
		if err := writeZipFile(f, target); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func writeZipFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	// 限制单文件解压上限，防 zip 炸弹。
	if _, err := io.Copy(out, io.LimitReader(rc, 64<<20)); err != nil {
		return err
	}
	return nil
}

func rel(r *repo.Repo, p string) string {
	if rp, err := filepath.Rel(r.Root, p); err == nil {
		return rp
	}
	return p
}
