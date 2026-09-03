package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/runner"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// 服务器端构建（ADR-0008）：runner 在构建机上常驻，轮询 build-job 队列。
// 鉴权/地址可来自 login 的 ~/.hybrid-pack/config.json，或以下环境变量（构建机更常用）。
const (
	envServer      = "HYBRID_PACK_SERVER"
	envToken       = "HYBRID_PACK_TOKEN"
	envRunnerID    = "HYBRID_PACK_RUNNER_ID"
	envArtifactDir = "HYBRID_PACK_ARTIFACT_DIR"
	envArtifactURL = "HYBRID_PACK_ARTIFACT_BASE_URL"
)

func newRunnerCmd() *cobra.Command {
	var serverFlag, tokenFlag, runnerID, artifactDir, artifactBaseURL string
	var pollSeconds int
	var once bool

	cmd := &cobra.Command{
		Use:   "runner",
		Short: "构建机守护：轮询 build-job 队列→pull+打包+签名→落产物→回传（ADR-0008）",
		Long: `服务器端构建执行体（ADR-0008）：在独立构建机/容器内常驻，循环执行：
  领取任务 → pull（拉最新配置渲染 CSV/res/bootstrap）→ ./gradlew assemble<Flavor>Release(+签名)
  → 把 APK 落到产物目录（默认 ` + runner.DefaultArtifactDir + `/<brand>/<flavor>/<versionName>/）
  → 回传状态/日志/产物给后端。版本号用 job.versionName（透传 -PversionName）。

鉴权与地址来源（优先级：命令行 > 环境变量 > ~/.hybrid-pack/config.json）：
  --server / ` + envServer + `        后端地址
  --token  / ` + envToken + `         访问令牌（构建机建议用环境注入）

签名（keystore 只作构建机 secret，绝不上传 / 绝不打印口令）：
  ` + runner.EnvKeystoreFile + `      keystore 文件路径
  ` + runner.EnvKeystorePassword + `  store 口令
  ` + runner.EnvKeyAlias + `          key alias
  ` + runner.EnvKeyPassword + `       key 口令
  四项齐全 → runner 注入进 local.properties 供 Gradle 签名；否则要求构建机已配好 local.properties。`,
		Example: `  # 常驻轮询（构建机）
  HYBRID_PACK_SERVER=https://admin.example.com HYBRID_PACK_TOKEN=*** \
  HYBRID_PACK_KEYSTORE_FILE=/secrets/release.jks HYBRID_PACK_KEYSTORE_PASSWORD=*** \
  HYBRID_PACK_KEY_ALIAS=release HYBRID_PACK_KEY_PASSWORD=*** \
  hybrid-pack runner --artifact-dir /var/www/apks

  # 单次取活（CI / 手动触发）
  hybrid-pack runner --once`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mustRepo()
			if err != nil {
				return err
			}

			server, token, err := resolveRunnerAuth(serverFlag, tokenFlag)
			if err != nil {
				return err
			}
			cl := api.New(server, token)

			rid := firstNonEmpty(runnerID, os.Getenv(envRunnerID))
			adir := firstNonEmpty(artifactDir, os.Getenv(envArtifactDir))
			aurl := firstNonEmpty(artifactBaseURL, os.Getenv(envArtifactURL))

			opt := runner.Options{
				RunnerID:        rid,
				ArtifactDir:     adir,
				ArtifactBaseURL: aurl,
				PollInterval:    time.Duration(pollSeconds) * time.Second,
				Once:            once,
				Keystore:        runner.KeystoreFromEnv(),
				Source:          cl,
				Logf:            func(f string, a ...any) { pterm.Println(fmt.Sprintf(f, a...)) },
			}

			// 优雅退出：SIGINT/SIGTERM 取消主循环（当前任务交由 ctx 传播）。
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			pterm.Info.Printfln("后端: %s", server)
			if err := runner.Run(ctx, r, cl, opt); err != nil {
				if ctx.Err() != nil {
					pterm.Info.Println("收到退出信号，runner 已停止")
					return nil
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverFlag, "server", "", "后端地址（缺省取环境 "+envServer+" 或登录配置）")
	cmd.Flags().StringVar(&tokenFlag, "token", "", "访问令牌（缺省取环境 "+envToken+" 或登录配置）")
	cmd.Flags().StringVar(&runnerID, "runner-id", "", "构建机标识（缺省取环境 "+envRunnerID+" 或主机名）")
	cmd.Flags().StringVar(&artifactDir, "artifact-dir", "", "产物根目录（缺省 "+runner.DefaultArtifactDir+"）")
	cmd.Flags().StringVar(&artifactBaseURL, "artifact-base-url", "", "产物下载前缀（nginx 暴露，如 /apks 或 https://host/apks）")
	cmd.Flags().IntVar(&pollSeconds, "poll", 5, "队列为空时的轮询间隔（秒）")
	cmd.Flags().BoolVar(&once, "once", false, "只领取并处理一个任务后退出（CI/手动触发）")
	return cmd
}

// resolveRunnerAuth 按「命令行 > 环境 > 登录配置」解析后端地址与令牌。
func resolveRunnerAuth(serverFlag, tokenFlag string) (server, token string, err error) {
	server = firstNonEmpty(serverFlag, os.Getenv(envServer))
	token = firstNonEmpty(tokenFlag, os.Getenv(envToken))
	if server == "" || token == "" {
		cfg, cErr := loadConfig()
		if cErr == nil {
			if server == "" {
				server = cfg.Server
			}
			if token == "" {
				token = cfg.Token
			}
		}
	}
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if server == "" {
		return "", "", fmt.Errorf("缺少后端地址：用 --server 或环境 %s，或先 hybrid-pack login", envServer)
	}
	if token == "" {
		return "", "", fmt.Errorf("缺少访问令牌：用 --token 或环境 %s，或先 hybrid-pack login", envToken)
	}
	return server, token, nil
}

// firstNonEmpty 返回首个非空白字符串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
