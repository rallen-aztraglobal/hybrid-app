package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hybrid-app/cli/internal/manifest"
)

// ManifestSource 抽象「manifest 从哪来」：可以是真实后台，也可以是本地 fixture。
// 这样 pull 在无后台时也能用本地 JSON / 现有 CSV 演练（dry-run），验证渲染逻辑。
type ManifestSource interface {
	Manifest(ctx context.Context, brand string) (*manifest.Manifest, error)
	// DownloadResZip 下载资源 zip；fixture 源对此通常返回「无资源」。
	DownloadResZip(ctx context.Context, rawURL, expectedSHA string) ([]byte, error)
}

// 编译期断言：真实 Client 满足 ManifestSource。
var _ ManifestSource = (*Client)(nil)

// FixtureSource 从本地目录读取 manifest JSON：<dir>/<brand>.json。
// 由环境变量 HYBRID_PACK_MANIFEST_DIR 触发，用于离线演练与自测（见 cli-go.md 自测要求）。
type FixtureSource struct {
	Dir string
}

// FixtureDirFromEnv 返回 fixture 目录（若设置了 HYBRID_PACK_MANIFEST_DIR）。
func FixtureDirFromEnv() string {
	return os.Getenv("HYBRID_PACK_MANIFEST_DIR")
}

func (f *FixtureSource) Manifest(_ context.Context, brand string) (*manifest.Manifest, error) {
	p := filepath.Join(f.Dir, brand+".json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("读取 fixture %s 失败: %w", p, err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析 fixture %s 失败: %w", p, err)
	}
	if m.Brand == "" {
		m.Brand = brand
	}
	return &m, nil
}

// DownloadResZip 对 fixture 源：若 rawURL 是本地存在的文件路径则读之，否则返回 nil（无资源）。
func (f *FixtureSource) DownloadResZip(_ context.Context, rawURL, _ string) ([]byte, error) {
	if rawURL == "" {
		return nil, nil
	}
	if data, err := os.ReadFile(rawURL); err == nil {
		return data, nil
	}
	// fixture 模式下无法访问对象存储时不报错，交由上层「跳过资源」。
	return nil, nil
}
