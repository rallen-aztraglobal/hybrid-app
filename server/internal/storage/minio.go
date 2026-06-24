package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO 把对象存到 MinIO/S3 兼容存储。PublicURL 走配置的 public base
// （通常是 CDN 或 nginx 反代到 bucket 的地址），与 SDK 的内网 endpoint 解耦。
type MinIO struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// MinIOOptions MinIO 连接参数。
type MinIOOptions struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

// NewMinIO 连接 MinIO 并确保 bucket 存在。
func NewMinIO(ctx context.Context, o MinIOOptions) (*MinIO, error) {
	cli, err := minio.New(o.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: o.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 MinIO 客户端失败: %w", err)
	}
	exists, err := cli.BucketExists(ctx, o.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查 bucket 失败: %w", err)
	}
	if !exists {
		if err := cli.MakeBucket(ctx, o.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("创建 bucket 失败: %w", err)
		}
	}
	pub := o.PublicURL
	if pub == "" {
		scheme := "http"
		if o.UseSSL {
			scheme = "https"
		}
		pub = fmt.Sprintf("%s://%s/%s", scheme, o.Endpoint, o.Bucket)
	}
	return &MinIO{client: cli, bucket: o.Bucket, publicURL: pub}, nil
}

func (m *MinIO) Kind() string { return "minio" }

func (m *MinIO) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	key = CleanKey(key)
	ct := guessContentType(key, contentType)
	// size<0 时用 -1 触发分片上传。
	if size <= 0 {
		size = -1
	}
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: ct})
	if err != nil {
		return "", fmt.Errorf("上传对象失败: %w", err)
	}
	return m.PublicURL(key), nil
}

func (m *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, CleanKey(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("读取对象失败: %w", err)
	}
	// GetObject 是惰性的，提前 Stat 以便把不存在归一化为 ErrNotFound。
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
		}
		return nil, fmt.Errorf("读取对象元信息失败: %w", err)
	}
	return obj, nil
}

func (m *MinIO) Delete(ctx context.Context, key string) error {
	if err := m.client.RemoveObject(ctx, m.bucket, CleanKey(key), minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("删除对象失败: %w", err)
	}
	return nil
}

func (m *MinIO) PublicURL(key string) string {
	return joinURL(m.publicURL, key)
}
