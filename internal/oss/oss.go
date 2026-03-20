package oss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// OSS 定义对象存储接口
type OSS interface {
	// PutObject 上传对象
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64) error
	// GetObject 获取对象
	GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, bucketName, objectName string) error
	// StatObject 获取对象元数据
	StatObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error)
	// PresignedGetObject 生成预签名下载链接
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (string, error)
	// MakeBucket 创建存储桶
	MakeBucket(ctx context.Context, bucketName string) error
}

// minioOSS 实现 OSS 接口
type minioOSS struct {
	client *minio.Client
}

// pathPrefixTransport 是一个自定义的 http.RoundTripper，用于在每个请求路径前添加前缀。
// 解决问题：
// 1. 标准的 MinIO Go SDK (minio-go) 不支持在 Endpoint 中包含路径前缀（例如 http://ip/oss/）。
// 2. 如果服务器（如 Nginx 代理）要求请求必须带有特定的路径前缀才能正确路由，直接传给 SDK 会报错。
// 3. 直接在签名计算前修改路径会破坏 S3 的签名机制。
// 本 Transport 在签名计算完成后，在网络层发送请求前动态添加前缀，既满足了服务器路由要求，又保证了签名的有效性。
type pathPrefixTransport struct {
	base   http.RoundTripper
	prefix string
}

func (t *pathPrefixTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 如果前缀不为空，将其添加到请求路径中
	if t.prefix != "" && !strings.HasPrefix(req.URL.Path, t.prefix) {
		req.URL.Path = t.prefix + req.URL.Path
		if req.URL.RawPath != "" {
			req.URL.RawPath = t.prefix + req.URL.RawPath
		}
	}
	return t.base.RoundTrip(req)
}

// NewMinioOSS creates a new Minio OSS client
func NewMinioOSS(cfg Config) (OSS, error) {
	endpoint := cfg.Endpoint
	var pathPrefix string

	// 处理包含协议头的 endpoint
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}

	// 提取路径前缀 (如 /oss/)。
	// 标准 minio-go 只能处理 host:port，如果配置了路径（如 49.233.216.158/oss/），
	// 我们需要将 /oss 提取出来并放入自定义 Transport 中处理。
	if idx := strings.Index(endpoint, "/"); idx != -1 {
		pathPrefix = endpoint[idx:]
		endpoint = endpoint[:idx]
	}
	// 去掉末尾斜杠以保持一致性
	pathPrefix = strings.TrimSuffix(pathPrefix, "/")

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	}

	// 如果存在路径前缀，使用自定义 Transport
	if pathPrefix != "" {
		opts.Transport = &pathPrefixTransport{
			base:   http.DefaultTransport,
			prefix: pathPrefix,
		}
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio client: %w", err)
	}

	return &minioOSS{client: client}, nil
}

func (m *minioOSS) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64) error {
	_, err := m.client.PutObject(ctx, bucketName, objectName, reader, objectSize, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to put object %s to bucket %s: %w", objectName, bucketName, err)
	}
	return nil
}

func (m *minioOSS) GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s from bucket %s: %w", objectName, bucketName, err)
	}
	return obj, nil
}

func (m *minioOSS) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	err := m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object %s from bucket %s: %w", objectName, bucketName, err)
	}
	return nil
}

func (m *minioOSS) StatObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	info, err := m.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("failed to stat object %s in bucket %s: %w", objectName, bucketName, err)
	}
	return info, nil
}

func (m *minioOSS) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url for %s in %s: %w", objectName, bucketName, err)
	}
	return u.String(), nil
}

func (m *minioOSS) MakeBucket(ctx context.Context, bucketName string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket %s exists: %w", bucketName, err)
	}
	if !exists {
		err = m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to make bucket %s: %w", bucketName, err)
		}
	}
	return nil
}
