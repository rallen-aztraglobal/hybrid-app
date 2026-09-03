package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hybrid-app/cli/internal/doctor"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "打包前预检：JDK / Android SDK / keystore / 后台连通性",
		Long: `逐项检查打包前置条件：
  - JDK：要求 17（本工程 jvmTarget/compileOptions 约定）
  - Android SDK：ANDROID_HOME 或 local.properties 的 sdk.dir
  - Keystore：local.properties 的 KEYSTORE_FILE/KEYSTORE_PASSWORD/KEY_ALIAS/KEY_PASSWORD
              （仅校验是否就位，绝不读取或打印任何口令）
  - 后台连通性：当前配置的后台是否可达 / 是否已登录

存在 FAIL 项时以非零退出码结束，便于 CI 拦截。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mustRepo()
			if err != nil {
				return err
			}
			cfg, _ := loadConfig() // 配置可缺省（仅影响后台连通性检查）

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			rep := doctor.Run(ctx, r, cfg)
			pterm.DefaultSection.Println("预检结果")
			for _, c := range rep.Checks {
				line := fmt.Sprintf("%s %-14s %s", c.Level.Symbol(), c.Name, c.Detail)
				switch c.Level {
				case doctor.OK:
					pterm.Success.Println(line)
				case doctor.Warn:
					pterm.Warning.Println(line)
				default:
					pterm.Error.Println(line)
				}
			}
			if rep.HasFail() {
				return fmt.Errorf("预检发现致命问题，请先修复再打包")
			}
			pterm.Success.Println("预检通过")
			return nil
		},
	}
	return cmd
}
