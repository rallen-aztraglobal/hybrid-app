package main

import (
	"fmt"
	"strings"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/config"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
	"github.com/spf13/cobra"
)

// version 由 -ldflags "-X main.version=..." 注入；默认 dev。
var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "hybrid-pack",
		Short: "渠道中台跨平台打包 CLI",
		Long: `hybrid-pack —— 渠道中台的跨平台打包工具（替代 package.sh）。

把后台统一管理的「渠道清单 / 图标资源 / 域名配置」渲染回现有 Gradle 输入：
  channels/<brand>.csv（字节级兼容重写）
  app/src/channels/<brand>/<flavor>/res（解压图标资源）
  app/src/channels/<brand>/<flavor>/assets/bootstrap.json（域名兜底 + 配置端点）
随后跨平台调用 ./gradlew 打包。绝不修改 app/build.gradle 的构建机制。

常用流程：
  hybrid-pack login --server https://admin.example.com   # 登录
  hybrid-pack pull --brand ap                            # 拉配置渲染回本地
  hybrid-pack build --brand ap --channels all --version 1.0.1  # 打包（透传 -PversionName）
  hybrid-pack release --brand ap --channels all          # pull→build→回传记录
  hybrid-pack runner --once                              # 构建机：领取 build-job 队列任务并出包（ADR-0008）`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	root.CompletionOptions.HiddenDefaultCmd = true

	root.AddCommand(
		newLoginCmd(),
		newPullCmd(),
		newBuildCmd(),
		newReleaseCmd(),
		newRunnerCmd(),
		newStatusCmd(),
		newDoctorCmd(),
	)
	return root
}

// ---- 共享辅助 ----

// mustRepo 定位仓库根。
func mustRepo() (*repo.Repo, error) {
	r, err := repo.Find("")
	if err != nil {
		return nil, err
	}
	return r, nil
}

// loadConfig 读取 CLI 配置。
func loadConfig() (*config.Config, error) {
	return config.Load()
}

// newManifestSource 选择 manifest 来源：
//   - 若设置 HYBRID_PACK_MANIFEST_DIR → 用本地 fixture（离线演练/自测，见 cli-go.md）；
//   - 否则用真实后台（需已登录）。
func newManifestSource(cfg *config.Config) (api.ManifestSource, string, error) {
	if dir := api.FixtureDirFromEnv(); dir != "" {
		return &api.FixtureSource{Dir: dir}, "fixture:" + dir, nil
	}
	if err := cfg.RequireAuth(); err != nil {
		return nil, "", err
	}
	return api.New(cfg.Server, cfg.Token), cfg.Server, nil
}

// resolveBrands 把 --brand 参数解析为有序品牌列表；空则返回全部已知品牌。
func resolveBrands(brand string) ([]string, error) {
	brand = strings.TrimSpace(brand)
	if brand == "" || brand == "all" {
		return manifest.KnownBrands(), nil
	}
	var out []string
	for _, b := range strings.FieldsFunc(brand, func(r rune) bool { return r == ',' || r == ' ' }) {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if !manifest.IsKnownBrand(b) {
			return nil, fmt.Errorf("未知大渠道: %q（支持: %s）", b, strings.Join(manifest.KnownBrands(), ", "))
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return manifest.KnownBrands(), nil
	}
	return out, nil
}
