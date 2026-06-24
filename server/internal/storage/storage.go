// Package storage 抽象对象存储（图标主图、资源 zip、APK 产物、CDN 配置快照）。
// 提供两种实现：local（本地磁盘，开发默认，无外部依赖）与 minio（生产）。
// handler/service 只依赖 Storage 接口，便于测试与切换。
package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Storage 是对象存储统一接口。key 用 "/" 分隔的相对路径（如 channels/12/icon/master.png）。
type Storage interface {
	// Put 上传对象，返回可对外访问的 URL（拼接 public base）。
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (publicURL string, err error)
	// Get 读取对象内容。调用方负责 Close。
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete 删除对象。
	Delete(ctx context.Context, key string) error
	// PublicURL 返回某 key 对外可访问的 URL（不校验是否存在）。
	PublicURL(key string) string
	// Kind 返回实现类型标识（local/minio），便于日志与诊断。
	Kind() string
}

// CleanKey 规范化对象 key：去掉前导 /，折叠重复 /。
func CleanKey(key string) string {
	key = strings.TrimPrefix(key, "/")
	for strings.Contains(key, "//") {
		key = strings.ReplaceAll(key, "//", "/")
	}
	return key
}

// joinURL 拼接 base 与 key，保证恰好一个 /。
func joinURL(base, key string) string {
	base = strings.TrimRight(base, "/")
	key = CleanKey(key)
	if base == "" {
		return "/" + key
	}
	return base + "/" + key
}

// guessContentType 在调用方未指定时按扩展名猜测。
func guessContentType(key, given string) string {
	if given != "" {
		return given
	}
	lower := strings.ToLower(key)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".zip"):
		return "application/zip"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	case strings.HasSuffix(lower, ".xml"):
		return "application/xml"
	case strings.HasSuffix(lower, ".apk"):
		return "application/vnd.android.package-archive"
	default:
		return "application/octet-stream"
	}
}

// ErrNotFound 在对象不存在时返回（各实现统一包装）。
var ErrNotFound = fmt.Errorf("object not found")
