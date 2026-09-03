package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/build"
	"github.com/hybrid-app/cli/internal/config"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/render"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	var brand, channels, versionName string
	var testEvents, yes, skipPull, noUpload bool

	cmd := &cobra.Command{
		Use:   "release",
		Short: "一条龙：pull → build → 收集产物 → 回传构建记录",
		Long: `pull 拉取并渲染最新配置 → build 打包 → 收集 APK → POST /api/build/records 回传记录。

安全：回传内容仅含品牌/渠道/状态/产物文件名/日志摘要，绝不上传 keystore、口令或本地绝对路径。`,
		Example: `  hybrid-pack release --brand ap --channels all
  hybrid-pack release --brand ap --channels ap01018,ap01034 -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mustRepo()
			if err != nil {
				return err
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if brand == "" {
				return fmt.Errorf("release 需指定 --brand（避免误打全部品牌）")
			}
			brands, err := resolveBrands(brand)
			if err != nil {
				return err
			}
			if len(brands) != 1 {
				return fmt.Errorf("release 一次仅支持一个品牌")
			}
			b := brands[0]

			ver := strings.TrimSpace(versionName)
			if ver != "" && !build.ValidVersionName(ver) {
				return fmt.Errorf("--version 必须为 X.Y.Z 格式（如 1.0.1），当前: %q", ver)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			// 1) pull（除非显式跳过）
			src, origin, err := newManifestSource(cfg)
			if err != nil {
				return err
			}
			if !skipPull {
				pterm.DefaultSection.Printfln("同步配置（%s）", origin)
				if _, err := render.Pull(ctx, r, src, b, render.Options{
					Logf: func(f string, a ...any) { pterm.Println(fmt.Sprintf(f, a...)) },
				}); err != nil {
					return fmt.Errorf("pull 失败: %w", err)
				}
			}

			// 2) build
			plan, err := resolveBuildPlan(r, b, channels, &testEvents, ver, yes)
			if err != nil {
				return err
			}

			// 记录版本号：显式 --version 优先，否则回读 build.gradle 默认。
			recVersion := plan.VersionName
			if recVersion == "" {
				recVersion = build.ReadVersionName(r)
			}
			rec := manifest.BuildRecord{
				BrandCode:   b,
				Flavors:     plan.Flavors,
				TestEvents:  plan.TestEvents,
				Status:      manifest.StatusRunning,
				Operator:    cfg.Operator,
				VersionName: recVersion,
			}

			pterm.DefaultSection.Println("打包")
			buildRes, buildErr := build.Run(ctx, r, build.Options{
				Flavors:          plan.Flavors,
				TestEvents:       plan.TestEvents,
				VersionName:      plan.VersionName,
				CaptureTailLines: 40,
			})
			if buildErr != nil {
				rec.Status = manifest.StatusFailed
				if buildRes != nil {
					rec.LogExcerpt = buildRes.LogTail
				}
				_ = uploadRecord(ctx, cfg, rec, noUpload)
				return fmt.Errorf("打包失败: %w", buildErr)
			}
			rec.LogExcerpt = buildRes.LogTail

			// 3) 收集产物（仅文件名，不含本地绝对路径）
			pterm.DefaultSection.Println("产物 APK")
			var names []string
			for _, f := range plan.Flavors {
				apks, _ := build.CollectAPKs(r, f)
				for _, a := range apks {
					pterm.Printfln("  %s", a)
					names = append(names, filepath.Base(a))
				}
			}
			rec.APKNames = names
			rec.Status = manifest.StatusSuccess
			pterm.Success.Printfln("完成，共 %d 个 APK", len(names))

			// 4) 回传记录
			if err := uploadRecord(ctx, cfg, rec, noUpload); err != nil {
				pterm.Warning.Printfln("构建成功，但回传记录失败: %v", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&brand, "brand", "b", "", "大渠道（ap/bp/gp）·必填")
	cmd.Flags().StringVarP(&channels, "channels", "c", "", "小渠道：逗号分隔的 flavor，或 all")
	cmd.Flags().StringVar(&versionName, "version", "", "版本号 X.Y.Z（透传 -PversionName=<v>；留空用 build.gradle 默认）")
	cmd.Flags().BoolVarP(&testEvents, "test-events", "t", false, "开启测试事件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过确认")
	cmd.Flags().BoolVar(&skipPull, "skip-pull", false, "跳过 pull，直接用本地配置打包")
	cmd.Flags().BoolVar(&noUpload, "no-upload", false, "不回传构建记录到后台")
	return cmd
}

func uploadRecord(ctx context.Context, cfg *config.Config, rec manifest.BuildRecord, noUpload bool) error {
	if noUpload {
		pterm.Info.Println("已跳过回传构建记录（--no-upload）")
		return nil
	}
	// fixture 模式或未登录时静默跳过，不阻塞本地打包。
	if strings.TrimSpace(cfg.Server) == "" || strings.TrimSpace(cfg.Token) == "" {
		pterm.Info.Println("未登录后台，跳过回传构建记录")
		return nil
	}
	cl := api.New(cfg.Server, cfg.Token)
	id, err := cl.PostBuildRecord(ctx, rec)
	if err != nil {
		return err
	}
	if id > 0 {
		pterm.Success.Printfln("构建记录已回传 #%d", id)
	} else {
		pterm.Success.Println("构建记录已回传")
	}
	return nil
}
