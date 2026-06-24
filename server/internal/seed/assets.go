package seed

// 首次部署的资产初始化：把仓库里已存在的渠道图标 / 启动页 / 整套 res 目录，
// 经 storage 抽象（local→/static→nginx；minio 同理）注册进库，让后台卡片不再显示占位符。
//
// 关键事实（见 CLAUDE.md / 仓库结构）：线上 80 个渠道包的图已在仓库里——
//   app/src/channels/<brand>/<flavor>/res/mipmap-*/ic_launcher.png + drawable/splash_fullscreen.png。
// 运营要求图片与 APK 一样经 nginx 静态目录提供（storage 层 → STORAGE_PUBLIC_URL=/static → nginx）。
//
// 每个渠道做三件事（全部经 storage，幂等）：
//   1) 取代表性图标（最大密度 mipmap-xxxhdpi/ic_launcher.png，缺则退次大）→ 存 icons/<appId>/icon.png → channel.icon_master_url
//   2) 取 drawable/splash_fullscreen.png → 存 icons/<appId>/splash.png → channel.splash_url
//   3) 整个 res 目录打 zip → 存 icons/<appId>/res.zip → channel.icon_set_url（供 CLI pull 回灌已有图标）
// 无 res 目录的渠道：跳过并日志提示。

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/storage"
)

// iconDensityPreference 代表性图标密度优先级（最大密度优先，缺则退次大）。
var iconDensityPreference = []string{"xxxhdpi", "xxhdpi", "xhdpi", "hdpi", "mdpi"}

// AssetReport 汇总一次资产初始化结果。
type AssetReport struct {
	Processed    int      // 成功处理（至少写了一种资产）的渠道数
	SkippedNoRes int      // 因无 res 目录被跳过的渠道数
	Icons        int      // 写入图标的渠道数
	Splashes     int      // 写入启动页的渠道数
	Zips         int      // 写入 res.zip 的渠道数
	Notes        []string // 逐渠道的跳过/降级提示
}

// assetKey 按 applicationId 归档资产的对象 key。
func iconKey(appID string) string   { return "icons/" + appID + "/icon.png" }
func splashKey(appID string) string { return "icons/" + appID + "/splash.png" }
func resZipKey(appID string) string { return "icons/" + appID + "/res.zip" }

// ResolveSeedResRoot 在 seedDir 下定位「渠道 res 根目录」，兼容两种布局：
//
//	① <seedDir>/res/<brand>/<flavor>/res        （镜像精简布局，Dockerfile.api COPY 进来）
//	② <seedDir>/<brand>/<flavor>/res            （直接指向仓库 app/src/channels）
//
// 返回可用的 res 根（其下应为 <brand>/<flavor>/res），找不到返回空串。
func ResolveSeedResRoot(seedDir string) string {
	if seedDir == "" {
		return ""
	}
	candidates := []string{filepath.Join(seedDir, "res"), seedDir}
	for _, c := range candidates {
		// 命中条件：c/<brand>/<flavor>/res 至少有一个存在（用三个品牌探测）。
		for _, b := range []string{"ap", "bp", "gp"} {
			brandDir := filepath.Join(c, b)
			st, err := os.Stat(brandDir)
			if err != nil || !st.IsDir() {
				continue
			}
			entries, err := os.ReadDir(brandDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					if st, err := os.Stat(filepath.Join(brandDir, e.Name(), "res")); err == nil && st.IsDir() {
						return c
					}
				}
			}
		}
	}
	return ""
}

// ResolveSeedCSVDir 在候选位置定位含 ap/bp/gp.csv 的目录，兼容多种部署/开发布局：
//
//	① <seedDir>/channels         （镜像精简布局，Dockerfile.api COPY channels/*.csv 进来）
//	② <seedDir>                   （csv 直接平铺在 seedDir）
//	③ ../channels / channels / ./channels  （server/ 内或仓库根运行时的相对路径，兼容旧行为）
//
// 找不到返回空串。
func ResolveSeedCSVDir(seedDir string) string {
	var candidates []string
	if seedDir != "" {
		candidates = append(candidates, filepath.Join(seedDir, "channels"), seedDir)
	}
	candidates = append(candidates, "../channels", "channels", "./channels")
	for _, c := range candidates {
		hit := false
		for _, b := range []string{"ap", "bp", "gp"} {
			if fileExists(filepath.Join(c, b+".csv")) {
				hit = true
				break
			}
		}
		if hit {
			return c
		}
	}
	return ""
}

// ImportAllFromDir 把 csvDir 下的 ap/bp/gp.csv 全部导入（每个品牌一次 ImportCSV）。
// 返回各品牌报告；缺某个 csv 只跳过该品牌、不报错（与 main.runImport 行为一致）。
func ImportAllFromDir(ctx context.Context, r *repo.Repo, csvDir string) ([]*ImportReport, error) {
	var reports []*ImportReport
	for _, brand := range []string{"ap", "bp", "gp"} {
		path := filepath.Join(csvDir, brand+".csv")
		f, err := os.Open(path)
		if err != nil {
			log.Printf("[import] 跳过 %s: %v", path, err)
			continue
		}
		rep, err := ImportCSV(ctx, r, brand, f)
		_ = f.Close()
		if err != nil {
			return reports, fmt.Errorf("导入 %s 失败: %w", path, err)
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

// channelResDir 返回某渠道在 resRoot 下的 res 目录（<resRoot>/<brand>/<flavor>/res），不存在返回空串。
func channelResDir(resRoot, brandCode, flavor string) string {
	p := filepath.Join(resRoot, brandCode, flavor, "res")
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return ""
}

// InitChannelAssets 对库里所有渠道做幂等资产初始化（按 brand+flavor 定位 res 目录）。
// resRoot 应为 ResolveSeedResRoot 的返回；调用方负责确保非空。
// 幂等保证：① 对象 key 固定（按 appId 归档），重复 Put 覆盖同一 key、不堆积；
//
//	② 渠道字段已是期望的 /static URL 时跳过写库，避免无谓更新。
func InitChannelAssets(ctx context.Context, r *repo.Repo, st storage.Storage, resRoot string) (*AssetReport, error) {
	rep := &AssetReport{}
	if resRoot == "" {
		return rep, fmt.Errorf("资产源目录为空（请设置 SEED_DIR 指向含渠道 res 的目录）")
	}

	// 取全部渠道（含 brand，用于定位 <brand>/<flavor>/res）。分页拉满。
	var all []model.Channel
	page := 1
	for {
		list, total, err := r.ListChannels(ctx, repo.ChannelFilter{Page: page, PageSize: 500})
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
		if int64(len(all)) >= total || len(list) == 0 {
			break
		}
		page++
	}

	for i := range all {
		ch := &all[i]
		brandCode := ""
		if ch.Brand != nil {
			brandCode = ch.Brand.Code
		}
		if brandCode == "" {
			// 兜底：理论上 ListChannels 已 Preload Brand。
			if b, err := r.GetBrandByID(ctx, ch.BrandID); err == nil {
				brandCode = b.Code
			}
		}
		dir := channelResDir(resRoot, brandCode, ch.FlavorName)
		if dir == "" {
			rep.SkippedNoRes++
			rep.Notes = append(rep.Notes, fmt.Sprintf("跳过 %s/%s (%s)：无 res 目录", brandCode, ch.FlavorName, ch.ApplicationID))
			log.Printf("[assets] 跳过 %s/%s：在 %s 下无 res 目录", brandCode, ch.FlavorName, resRoot)
			continue
		}

		changed := map[string]any{}
		touched := false

		// 1) 代表性图标（最大密度优先）。
		if iconPath := pickRepresentativeIcon(dir); iconPath != "" {
			key := iconKey(ch.ApplicationID)
			url := st.PublicURL(key)
			if ch.IconMasterURL != url {
				if _, err := putFile(ctx, st, key, iconPath, "image/png"); err != nil {
					return nil, fmt.Errorf("%s 写图标失败: %w", ch.ApplicationID, err)
				}
				changed["icon_master_url"] = url
			} else if !objectExists(ctx, st, key) {
				// URL 已写但对象丢失（如换了存储后端）→ 重新上传，保持自愈。
				if _, err := putFile(ctx, st, key, iconPath, "image/png"); err != nil {
					return nil, fmt.Errorf("%s 重传图标失败: %w", ch.ApplicationID, err)
				}
			}
			rep.Icons++
		} else {
			rep.Notes = append(rep.Notes, fmt.Sprintf("%s/%s：res 内无 ic_launcher.png，图标未注册", brandCode, ch.FlavorName))
		}

		// 2) 启动页。
		splashPath := filepath.Join(dir, "drawable", "splash_fullscreen.png")
		if fileExists(splashPath) {
			key := splashKey(ch.ApplicationID)
			url := st.PublicURL(key)
			if ch.SplashURL != url {
				if _, err := putFile(ctx, st, key, splashPath, "image/png"); err != nil {
					return nil, fmt.Errorf("%s 写启动页失败: %w", ch.ApplicationID, err)
				}
				changed["splash_url"] = url
			} else if !objectExists(ctx, st, key) {
				if _, err := putFile(ctx, st, key, splashPath, "image/png"); err != nil {
					return nil, fmt.Errorf("%s 重传启动页失败: %w", ch.ApplicationID, err)
				}
			}
			rep.Splashes++
		}

		// 3) 整个 res 目录打 zip（供 CLI pull 回灌已有图标）。
		key := resZipKey(ch.ApplicationID)
		url := st.PublicURL(key)
		if ch.IconSetURL != url || !objectExists(ctx, st, key) {
			zipBytes, err := zipDir(dir)
			if err != nil {
				return nil, fmt.Errorf("%s 打包 res.zip 失败: %w", ch.ApplicationID, err)
			}
			if _, err := st.Put(ctx, key, bytes.NewReader(zipBytes), int64(len(zipBytes)), "application/zip"); err != nil {
				return nil, fmt.Errorf("%s 写 res.zip 失败: %w", ch.ApplicationID, err)
			}
			if ch.IconSetURL != url {
				changed["icon_set_url"] = url
			}
		}
		rep.Zips++

		if len(changed) > 0 {
			if err := r.UpdateChannelFields(ctx, ch.ID, changed); err != nil {
				return nil, fmt.Errorf("%s 更新渠道资产字段失败: %w", ch.ApplicationID, err)
			}
			touched = true
		}
		if touched {
			rep.Processed++
		}
	}
	return rep, nil
}

// pickRepresentativeIcon 在 res 目录里按密度优先级挑一张方形图标的绝对路径，找不到返回空串。
func pickRepresentativeIcon(resDir string) string {
	for _, dpi := range iconDensityPreference {
		p := filepath.Join(resDir, "mipmap-"+dpi, "ic_launcher.png")
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// putFile 把磁盘文件经 storage 写入指定 key。
func putFile(ctx context.Context, st storage.Storage, key, path, contentType string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开源文件失败: %w", err)
	}
	defer f.Close()
	var size int64 = -1
	if fi, err := f.Stat(); err == nil {
		size = fi.Size()
	}
	return st.Put(ctx, key, f, size, contentType)
}

// objectExists 判断对象是否已存在于存储（用于自愈：URL 已写但文件丢失时重传）。
func objectExists(ctx context.Context, st storage.Storage, key string) bool {
	rc, err := st.Get(ctx, key)
	if err != nil {
		return false
	}
	_ = rc.Close()
	return true
}

// zipDir 把目录下全部文件打成 zip（zip 内路径相对 dir，用 / 分隔，与 res.zip 约定一致）。
// 结果对相同输入稳定（条目按路径排序），便于幂等比对与诊断。
func zipDir(dir string) ([]byte, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range files {
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		w, err := zw.Create(rel)
		if err != nil {
			return nil, fmt.Errorf("写 zip 条目 %s 失败: %w", rel, err)
		}
		src, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", rel, err)
		}
		_, copyErr := io.Copy(w, src)
		_ = src.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("写 zip 数据 %s 失败: %w", rel, copyErr)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Summary 返回适合一行日志的摘要。
func (rep *AssetReport) Summary() string {
	return fmt.Sprintf("处理 %d 渠道（图标 %d / 启动页 %d / res.zip %d），跳过无 res %d",
		rep.Processed, rep.Icons, rep.Splashes, rep.Zips, rep.SkippedNoRes)
}
