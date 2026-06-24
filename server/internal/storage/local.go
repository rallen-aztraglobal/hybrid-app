package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Local 把对象写到本地磁盘，PublicURL 指向后端的 /static 静态路由（main 里挂载）。
// 适合开发与单机部署；生产换 MinIO。
type Local struct {
	root      string // 磁盘根目录
	publicURL string // 对外前缀，如 http://localhost:8080/static
}

// NewLocal 创建本地磁盘存储，自动建根目录。
func NewLocal(root, publicURL string) (*Local, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建对象存储根目录失败: %w", err)
	}
	return &Local{root: root, publicURL: publicURL}, nil
}

func (l *Local) Kind() string { return "local" }

// Root 返回磁盘根目录，供静态文件路由挂载。
func (l *Local) Root() string { return l.root }

func (l *Local) abs(key string) string {
	return filepath.Join(l.root, filepath.FromSlash(CleanKey(key)))
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) (string, error) {
	p := l.abs(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	f, err := os.Create(p)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return l.PublicURL(key), nil
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(l.abs(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
		}
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	return f, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	if err := os.Remove(l.abs(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

func (l *Local) PublicURL(key string) string {
	return joinURL(l.publicURL, key)
}
