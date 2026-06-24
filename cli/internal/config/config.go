// Package config 管理 CLI 的本地配置：~/.hybrid-pack/config.json。
//
// 仅保存与后台交互所需的非机密运行参数（服务地址、登录 token、当前操作者）。
// 安全红线（CLAUDE.md 护栏 4 / cli-go.md）：keystore 路径与任何签名口令绝不写入此文件，
// 也绝不随构建记录上传后台——签名密钥只存在于仓库本地 local.properties。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 是 ~/.hybrid-pack/config.json 的内容。
type Config struct {
	// Server 后台基础地址，如 https://admin.example.com（无尾斜杠）。
	Server string `json:"server"`
	// Token 登录后获得的访问令牌（Bearer）。
	Token string `json:"token,omitempty"`
	// Operator 当前登录用户名，用于构建记录的 operator 字段。
	Operator string `json:"operator,omitempty"`
}

// DirName 是配置目录名（位于用户主目录下）。
const DirName = ".hybrid-pack"

// FileName 是配置文件名。
const FileName = "config.json"

// Dir 返回配置目录绝对路径：~/.hybrid-pack。
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, DirName), nil
}

// Path 返回配置文件绝对路径。
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// Load 读取配置；文件不存在返回空 Config（非错误），便于首次使用。
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("读取配置 %s 失败: %w", p, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析配置 %s 失败: %w", p, err)
	}
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	return &c, nil
}

// Save 持久化配置。目录权限 0700、文件权限 0600（含 token，按机密文件处理）。
func (c *Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data = append(data, '\n')
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("写配置 %s 失败: %w", p, err)
	}
	return nil
}

// RequireServer 校验已配置后台地址，否则提示先 login。
func (c *Config) RequireServer() error {
	if strings.TrimSpace(c.Server) == "" {
		return errors.New("尚未配置后台地址，请先执行: hybrid-pack login --server <URL>")
	}
	return nil
}

// RequireAuth 校验已登录（有 server 与 token）。
func (c *Config) RequireAuth() error {
	if err := c.RequireServer(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("尚未登录或登录已失效，请先执行: hybrid-pack login --server <URL>")
	}
	return nil
}
