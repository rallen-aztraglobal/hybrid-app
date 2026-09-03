package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var server, username, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "登录后台并保存 token 到 ~/.hybrid-pack/config.json",
		Long: `登录渠道中台后台，保存服务地址与访问令牌到 ~/.hybrid-pack/config.json（权限 0600）。

可交互输入账号密码，或用 --username/--password 非交互（CI）。
安全：仅保存服务地址、token、用户名；keystore 与任何签名口令绝不写入此文件。`,
		Example: `  hybrid-pack login --server https://admin.example.com
  hybrid-pack login --server https://admin.example.com --username ops --password ***`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if server != "" {
				cfg.Server = strings.TrimRight(strings.TrimSpace(server), "/")
			}
			if cfg.Server == "" {
				if err := huh.NewInput().Title("后台地址").Placeholder("https://admin.example.com").Value(&cfg.Server).Run(); err != nil {
					return fmt.Errorf("已取消: %w", err)
				}
				cfg.Server = strings.TrimRight(strings.TrimSpace(cfg.Server), "/")
			}
			if cfg.Server == "" {
				return fmt.Errorf("必须提供后台地址 --server")
			}

			// 账号密码：交互优先用 huh；非交互用 flag。
			if username == "" || password == "" {
				form := huh.NewForm(huh.NewGroup(
					huh.NewInput().Title("用户名").Value(&username),
					huh.NewInput().Title("密码").EchoMode(huh.EchoModePassword).Value(&password),
				))
				if err := form.Run(); err != nil {
					return fmt.Errorf("已取消: %w", err)
				}
			}
			if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
				return fmt.Errorf("用户名与密码不能为空")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cl := api.New(cfg.Server, "")
			resp, err := cl.Login(ctx, username, password)
			if err != nil {
				return fmt.Errorf("登录失败: %w", err)
			}
			cfg.Token = resp.AccessToken
			cfg.Operator = resp.Username
			if err := cfg.Save(); err != nil {
				return err
			}
			p, _ := config.Path()
			fmt.Fprintf(os.Stdout, "✓ 登录成功：%s（用户 %s）\n  凭据已保存至 %s\n", cfg.Server, cfg.Operator, p)
			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "后台基础地址，如 https://admin.example.com")
	cmd.Flags().StringVarP(&username, "username", "u", "", "用户名（非交互）")
	cmd.Flags().StringVarP(&password, "password", "p", "", "密码（非交互；CI 建议改用环境注入）")
	return cmd
}
