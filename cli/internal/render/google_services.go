package render

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/repo"
)

// PullGoogleServices 尝试从后台拉取指定品牌的 google-services.json，写到 app/google-services.json。
//
// 「有则拉、无则跳过」语义（ADR-0012 第 6b 节）：
//   - 后端返回 nil（404 / 未配置 FCM / fixture 无文件）→ 打印跳过提示，正常返回 nil，不阻断打包。
//   - 成功拿到 JSON → 写入 app/google-services.json，打印落地路径。
//   - dry-run → 只打印将要写入的路径，不落盘。
//
// 调用方负责在每次 pull 时调用，确保「换品牌打包」时覆盖上一品牌的文件。
func PullGoogleServices(ctx context.Context, r *repo.Repo, src api.ManifestSource, brand string, opt Options) error {
	data, err := src.FetchGoogleServicesJSON(ctx, brand)
	if err != nil {
		// 理论上 Client 实现对网络异常已宽容处理（返回 nil,nil），此处作为最后兜底。
		opt.logf("  警告: 拉取 %s google-services.json 出错（%v），已跳过", brand, err)
		return nil
	}
	if len(data) == 0 {
		// 后端无该文件 → 跳过。但必须清理工作区可能残留的【上一品牌】google-services.json：
		// 构建机工作区跨任务持久化，若先打 bp（落地 bp 配置）再打 gp（无配置），残留的 bp 文件会让
		// google-services 插件按 bp 项目找 client，gp 包名不在其中而构建失败。清理后 gp 恢复无推送正常打包。
		if !opt.DryRun {
			if err := os.Remove(r.AppGoogleServicesJSON()); err != nil && !os.IsNotExist(err) {
				opt.logf("  警告: 清理残留 google-services.json 失败（%v）", err)
			}
		}
		opt.logf("  ⏭ 该品牌（%s）未配置 FCM，跳过 google-services.json（推送功能未启用）", brand)
		return nil
	}

	dest := r.AppGoogleServicesJSON()
	if opt.DryRun {
		opt.logf("  [dry-run] 将写 %s（%d 字节）", rel(r, dest), len(data))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("创建 app 目录失败: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("写 %s 失败: %w", dest, err)
	}
	opt.logf("  google-services.json 已落地 → %s（%d 字节）", rel(r, dest), len(data))
	return nil
}
