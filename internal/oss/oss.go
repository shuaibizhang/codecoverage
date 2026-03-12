package oss

import (
	"context"
	"fmt"
	"io"
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
}

// minioOSS 实现 OSS 接口
type minioOSS struct {
	client *minio.Client
}

// NewMinioOSS 创建 MinIO 存储实例
func NewMinioOSS(cfg Config) (OSS, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
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
