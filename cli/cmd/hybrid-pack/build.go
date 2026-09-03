package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/hybrid-app/cli/internal/build"
	"github.com/hybrid-app/cli/internal/csvio"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var brand, channels, versionName string
	var testEvents, yes bool
	var extra []string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "打包选定的小渠道（交互式或非交互）",
		Long: `对选定小渠道执行 ./gradlew assemble<Cap(flavor)>Release（Windows 用 gradlew.bat）。

复刻 package.sh 体验：交互式「选大渠道 → 多选小渠道 → 测试事件开关」。
非交互（CI）用 --brand/--channels/--test-events。task 名与 cap 逻辑与 package.sh 一致。

注意：build 直接读取本地 channels/<brand>.csv（不联网）。如需最新配置，先 hybrid-pack pull。`,
		Example: `  hybrid-pack build
  hybrid-pack build --brand ap --channels all --test-events
  hybrid-pack build --brand ap --channels ap01018,ap01034 --version 1.0.1 -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mustRepo()
			if err != nil {
				return err
			}
			// 先校验版本号（快速失败，避免走完交互/计划再报错）。
			ver := strings.TrimSpace(versionName)
			if ver != "" && !build.ValidVersionName(ver) {
				return fmt.Errorf("--version 必须为 X.Y.Z 格式（如 1.0.1），当前: %q", ver)
			}
			plan, err := resolveBuildPlan(r, brand, channels, &testEvents, ver, yes)
			if err != nil {
				return err
			}
			return runBuild(cmd.Context(), r, plan, extra)
		},
	}
	cmd.Flags().StringVarP(&brand, "brand", "b", "", "大渠道（ap/bp/gp）")
	cmd.Flags().StringVarP(&channels, "channels", "c", "", "小渠道：逗号分隔的 flavor，或 all")
	cmd.Flags().BoolVarP(&testEvents, "test-events", "t", false, "开启测试事件（-PtestEvents=true）")
	cmd.Flags().StringVar(&versionName, "version", "", "版本号 X.Y.Z（透传 -PversionName=<v>；留空用 build.gradle 默认）")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过确认")
	cmd.Flags().StringArrayVar(&extra, "gradle-arg", nil, "透传给 gradlew 的附加参数（可重复）")
	return cmd
}

// buildPlan 是一次构建的最终计划。
type buildPlan struct {
	Brand       string
	Flavors     []string
	TestEvents  bool
	VersionName string // 空 = 用 build.gradle 默认；非空透传 -PversionName
}

// resolveBuildPlan 决定品牌、小渠道、测试事件：参数齐全则非交互，否则用 huh 交互补齐。
// testEventsFlag 指向 cobra 的 flag 值，便于在交互模式覆盖；versionName 已在调用前校验。
func resolveBuildPlan(r *repo.Repo, brand, channels string, testEventsFlag *bool, versionName string, yes bool) (*buildPlan, error) {
	interactive := brand == "" || channels == ""

	// 1) 品牌
	if brand == "" {
		opts := make([]huh.Option[string], 0, 3)
		for _, b := range manifest.KnownBrands() {
			n, _ := countChannels(r, b)
			opts = append(opts, huh.NewOption(fmt.Sprintf("%s  %s  (%d 个渠道)", b, manifest.BrandName(b), n), b))
		}
		if err := huh.NewSelect[string]().Title("选择大渠道").Options(opts...).Value(&brand).Run(); err != nil {
			return nil, fmt.Errorf("已取消: %w", err)
		}
	}
	if !manifest.IsKnownBrand(brand) {
		return nil, fmt.Errorf("无效大渠道: %q（支持: %s）", brand, strings.Join(manifest.KnownBrands(), ", "))
	}

	// 读取该品牌全部小渠道。
	csvFile, err := csvio.ReadFile(r.ChannelsCSV(brand))
	if err != nil {
		return nil, err
	}
	if len(csvFile.Rows) == 0 {
		return nil, fmt.Errorf("品牌 %s 无小渠道（channels/%s.csv 为空）；可先 hybrid-pack pull", brand, brand)
	}
	// 本地脏数据预警（不阻塞构建，但提示）。
	if conflicts := csvio.Validate(csvFile.Rows); len(conflicts) > 0 {
		pterm.Warning.Println("本地 CSV 存在唯一性冲突（建议先 pull 清洗）：")
		for _, c := range conflicts {
			pterm.Println("  - " + c.String())
		}
	}

	// 2) 小渠道
	var selected []string
	switch {
	case channels == "all":
		selected = csvFile.Flavors()
	case channels != "":
		selected, err = parseChannelArg(channels, csvFile)
		if err != nil {
			return nil, err
		}
	default:
		opts := make([]huh.Option[string], 0, len(csvFile.Rows))
		for _, row := range csvFile.Rows {
			label := fmt.Sprintf("%-16s %s", row.Flavor, row.AppName)
			opts = append(opts, huh.NewOption(label, row.Flavor))
		}
		if err := huh.NewMultiSelect[string]().
			Title("选择小渠道（空格多选 / a 全选）").
			Options(opts...).
			Value(&selected).Run(); err != nil {
			return nil, fmt.Errorf("已取消: %w", err)
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("未选择任何小渠道")
		}
	}

	// 3) 测试事件（仅交互模式询问）
	if interactive && channels == "" {
		te := *testEventsFlag
		if err := huh.NewConfirm().Title("开启测试事件？").Affirmative("是").Negative("否").Value(&te).Run(); err != nil {
			return nil, fmt.Errorf("已取消: %w", err)
		}
		*testEventsFlag = te
	}

	plan := &buildPlan{Brand: brand, Flavors: selected, TestEvents: *testEventsFlag, VersionName: versionName}

	// 汇总确认
	printPlan(plan)
	if interactive && !yes {
		ok := true
		if err := huh.NewConfirm().Title("确认开始打包？").Affirmative("开始").Negative("取消").Value(&ok).Run(); err != nil {
			return nil, fmt.Errorf("已取消: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("已取消")
		}
	}
	return plan, nil
}

// parseChannelArg 解析 --channels 的逗号/空格分隔 flavor 列表，并校验都属于该品牌。
func parseChannelArg(arg string, csvFile *csvio.File) ([]string, error) {
	valid := map[string]bool{}
	for _, f := range csvFile.Flavors() {
		valid[f] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, f := range strings.FieldsFunc(arg, func(r rune) bool { return r == ',' || r == ' ' }) {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !valid[f] {
			return nil, fmt.Errorf("未知小渠道: %q（不属于该品牌）", f)
		}
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("未选择任何小渠道")
	}
	return out, nil
}

func printPlan(p *buildPlan) {
	pterm.DefaultSection.Println("打包计划")
	pterm.Printfln("  大渠道  : %s (%s)", p.Brand, manifest.BrandName(p.Brand))
	pterm.Printfln("  小渠道  : %d 个 → %s", len(p.Flavors), strings.Join(p.Flavors, ", "))
	pterm.Printfln("  测试事件: %t", p.TestEvents)
	if p.VersionName != "" {
		pterm.Printfln("  版本号  : %s", p.VersionName)
	} else {
		pterm.Printfln("  版本号  : (build.gradle 默认)")
	}
	pterm.Printfln("  构建类型: Release")
}

// runBuild 执行构建并打印产物 APK。
func runBuild(ctx context.Context, r *repo.Repo, plan *buildPlan, extra []string) error {
	opt := build.Options{
		Flavors:     plan.Flavors,
		TestEvents:  plan.TestEvents,
		VersionName: plan.VersionName,
		ExtraArgs:   extra,
	}
	pterm.Printfln("执行: %s %s", build.GradlewPath(r), strings.Join(opt.Args(), " "))
	_, err := build.Run(ctx, r, opt)
	if err != nil {
		return fmt.Errorf("打包失败: %w", err)
	}
	pterm.DefaultSection.Println("产物 APK")
	total := 0
	for _, f := range plan.Flavors {
		apks, _ := build.CollectAPKs(r, f)
		for _, a := range apks {
			pterm.Printfln("  %s", a)
			total++
		}
	}
	pterm.Success.Printfln("完成，共 %d 个 APK", total)
	return nil
}

// countChannels 返回某品牌小渠道数量（供交互菜单展示）。
func countChannels(r *repo.Repo, brand string) (int, error) {
	f, err := csvio.ReadFile(r.ChannelsCSV(brand))
	if err != nil {
		return 0, err
	}
	return len(f.Rows), nil
}
