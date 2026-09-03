package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hybrid-app/cli/internal/render"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var brand string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "本地 CSV 与后台的漂移检测",
		Long: `对比本地 channels/<brand>.csv 与后台 manifest，列出差异：
  - 仅后台有（本地缺，pull 会新增）
  - 仅本地有（后台已删/未启用，pull 会移除）
  - 字段变化（applicationId/palCode/appName 不一致）
并提示本地 CSV 自身的唯一性冲突（脏数据）。`,
		Example: `  hybrid-pack status
  hybrid-pack status --brand ap`,
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

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			allInSync := true
			var hadErr bool
			for _, b := range brands {
				d, err := render.Status(ctx, r, src, b)
				if err != nil {
					hadErr = true
					pterm.Error.Println(err.Error())
					continue
				}
				printDrift(d)
				if !d.InSync() || len(d.LocalConflicts) > 0 {
					allInSync = false
				}
			}
			if hadErr {
				return fmt.Errorf("部分品牌漂移检测失败")
			}
			if allInSync {
				pterm.Success.Println("本地与后台完全一致")
			} else {
				pterm.Warning.Println("存在漂移，执行 hybrid-pack pull 同步到本地")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&brand, "brand", "b", "", "大渠道（ap/bp/gp），留空=全部")
	return cmd
}

func printDrift(d *render.Drift) {
	pterm.DefaultSection.Printfln("%s", d.Brand)
	if d.InSync() && len(d.LocalConflicts) == 0 {
		pterm.Success.Printfln("  一致")
		return
	}
	for _, f := range d.OnlyRemote {
		pterm.Printfln("  + %s（仅后台，pull 将新增）", f)
	}
	for _, f := range d.OnlyLocal {
		pterm.Printfln("  - %s（仅本地，pull 将移除）", f)
	}
	for f, diffs := range d.Changed {
		for _, fd := range diffs {
			pterm.Printfln("  ~ %s.%s: 本地=%q 后台=%q", f, fd.Field, fd.Local, fd.Remote)
		}
	}
	for _, c := range d.LocalConflicts {
		pterm.Warning.Printfln("  ! 本地冲突 %s", c.String())
	}
}
