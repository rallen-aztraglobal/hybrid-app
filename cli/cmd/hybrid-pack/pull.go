package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hybrid-app/cli/internal/render"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var brand string
	var dryRun, skipRes bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "拉取后台配置并渲染回现有 CSV / res / bootstrap.json",
		Long: `从后台拉取 manifest，把数据渲染回现有 Gradle 输入文件（ADR-0004）：
  - 重写 channels/<brand>.csv（保留注释头，与现状字节级兼容）
  - 下载解压 res.zip 到 app/src/channels/<brand>/<flavor>/res
  - 写每个 flavor 的 assets/bootstrap.json（域名兜底 + 配置端点，ADR-0002）

会先做唯一性校验（applicationId/palCode/flavor），有冲突即拒绝写入。

离线演练：设置环境变量 HYBRID_PACK_MANIFEST_DIR=<目录> 可用本地 <brand>.json 演练，
无需后台。配合 --dry-run 仅展示将发生的变更、不写盘。`,
		Example: `  hybrid-pack pull                 # 全部品牌
  hybrid-pack pull --brand ap      # 仅 ap
  hybrid-pack pull --brand ap --dry-run --skip-res`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mustRepo()
			if err != nil {
				return err
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			src, origin, err := newManifestSource(cfg)
			if err != nil {
				return err
			}
			brands, err := resolveBrands(brand)
			if err != nil {
				return err
			}
			pterm.Info.Printfln("数据源: %s", origin)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			var failed bool
			for _, b := range brands {
				title := fmt.Sprintf("同步 %s", b)
				if dryRun {
					title += "（dry-run）"
				}
				pterm.DefaultSection.Println(title)
				logf := func(f string, a ...any) { pterm.Println(fmt.Sprintf(f, a...)) }
				opts := render.Options{
					DryRun:  dryRun,
					SkipRes: skipRes,
					Logf:    logf,
				}
				res, err := render.Pull(ctx, r, src, b, opts)
				if err != nil {
					failed = true
					pterm.Error.Println(err.Error())
					continue
				}
				// 额外拉取该品牌的 google-services.json（有则落地、无则友好提示，不阻断）。
				if gsErr := render.PullGoogleServices(ctx, r, src, b, opts); gsErr != nil {
					pterm.Warning.Printfln("google-services.json 写入失败（%v），已跳过", gsErr)
				}
				summary := fmt.Sprintf("%s: %d 渠道", b, res.ChannelCount)
				if res.CSVChanged {
					summary += "，CSV 已更新"
				} else {
					summary += "，CSV 无变化"
				}
				summary += fmt.Sprintf("，bootstrap %d", res.BootstrapCount)
				if !skipRes {
					summary += fmt.Sprintf("，res 更新 %d/跳过 %d", res.ResWritten, res.ResSkipped)
				}
				pterm.Success.Println(summary)
			}
			if failed {
				return fmt.Errorf("部分品牌同步失败，详见上方日志")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&brand, "brand", "b", "", "大渠道（ap/bp/gp），逗号分隔多个，留空=全部")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只展示将发生的变更，不写盘")
	cmd.Flags().BoolVar(&skipRes, "skip-res", false, "跳过 res 资源下载/解压，仅刷新 CSV 与 bootstrap.json")
	return cmd
}
