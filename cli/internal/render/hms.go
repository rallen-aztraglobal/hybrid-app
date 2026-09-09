package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
)

// renderHMSChannels 渲染 app/hms-channels.json：本次品牌**全部**渠道的
// 「是否集成华为 HMS/OAID」权威表，键=派生后的 applicationId（ADR-0009），值=布尔。
//
// 为什么需要这个文件：HMS/OAID 原先只有两个判据——品牌整体开（bp）、或 flavor 以 _hw 结尾的
// 华为商店包。但存在「已上架华为商店、flavor 却不带 _hw 后缀」的老渠道（如 ap01018），
// 按名字推断会漏集成 OAID → 华为设备无 GAID、AppsFlyer 归因丢事件。故把开关收进 Console
// 的渠道配置，由后台按渠道下发有效值。
//
// 与 adjust-tokens.json 的两点差异：
//  1. 写**全量**渠道（含 false）——「未收录」在 Gradle 侧的语义是「回落旧默认规则」而非「关闭」；
//  2. **跨品牌合并**而非整体覆盖。build.gradle 的 dependencies 块遍历的是三个品牌的 allChannels，
//     单文件服务全部品牌；若按品牌整体覆盖，`pull bp` 会抹掉 ap/gp 的显式开关，随后本地
//     `./package.sh -b ap` 打 ap01018（本功能的由来）就查不到键、静默回落旧规则漏掉 OAID。
//     故只重写本品牌前缀（com.arenaplus. 等）下的键：本品牌以本次 manifest 为准（已删渠道的
//     陈旧键随之清掉），其余品牌的既有键原样保留。
//
// 返回本品牌写入的「集成 HMS」渠道数（供调用方汇报）。
func renderHMSChannels(r *repo.Repo, brand string, brandHMS bool, channels []manifest.Channel, opt Options) (int, error) {
	flags := make(map[string]bool, len(channels))
	enabled := 0
	for _, ch := range channels {
		appID := manifest.DeriveApplicationID(brand, ch.Flavor)
		if appID == "" {
			// 派生失败（未知品牌等极端情况）时回退到 manifest 给定值，与 bootstrap.json 的策略一致。
			appID = ch.ApplicationId
		}
		on := ch.EffectiveHMS(brandHMS)
		flags[appID] = on
		if on {
			enabled++
		}
	}

	dest := r.AppHMSChannelsJSON()

	// 与工作区已有文件合并：先摘掉本品牌前缀下的所有旧键（本次 manifest 是本品牌的唯一权威，
	// 顺带清掉已删渠道的陈旧键），再并入本次结果；其余品牌的键原样保留。
	merged, err := readHMSChannels(dest)
	if err != nil {
		return 0, err
	}
	if prefix := manifest.BrandPackagePrefix(brand); prefix != "" {
		for k := range merged {
			if strings.HasPrefix(k, prefix+".") {
				delete(merged, k)
			}
		}
	}
	for k, v := range flags {
		merged[k] = v
	}

	if len(merged) == 0 {
		// 三个品牌都没有任何渠道：不留空文件。Gradle 侧 file(...).exists() 判断，缺失 = 全体
		// 回落默认规则，与加开关前的存量行为一致。呼应 adjust.go / google_services.go 对残留
		// 渲染产物的清理。
		if opt.DryRun {
			opt.logf("  [dry-run] 该品牌（%s）无渠道，%s 将保持缺省/被清理", brand, rel(r, dest))
			return 0, nil
		}
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			opt.logf("  警告: 清理残留 hms-channels.json 失败（%v）", err)
		}
		return 0, nil
	}

	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("序列化 hms-channels.json 失败: %w", err)
	}
	data = append(data, '\n')

	if opt.DryRun {
		opt.logf("  [dry-run] 将写 %s（本品牌 %d/%d 个渠道集成 HMS，合计 %d 个键）", rel(r, dest), enabled, len(flags), len(merged))
		return enabled, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, fmt.Errorf("创建 app 目录失败: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return 0, fmt.Errorf("写 %s 失败: %w", dest, err)
	}
	opt.logf("  hms-channels.json 已落地 → %s（本品牌 %d/%d 个渠道集成 HMS，合计 %d 个键）", rel(r, dest), enabled, len(flags), len(merged))
	return enabled, nil
}

// readHMSChannels 读取工作区已有的 hms-channels.json；文件不存在返回空表。
// 内容损坏（手工改坏 / 半截文件）时不静默丢弃已有配置，而是报错中断本次渲染——
// 静默重建会让其他品牌的显式开关无声消失，正是本文件要防的事。
func readHMSChannels(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读 %s 失败: %w", path, err)
	}
	out := map[string]bool{}
	if len(bytes.TrimSpace(data)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("解析 %s 失败（可删除该文件后重试）: %w", path, err)
	}
	return out, nil
}
