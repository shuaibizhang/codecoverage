package oss

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestMinioOSS_PutObject(t *testing.T) {
	// 准备配置 (对应 dev.toml 中的配置)
	cfg := Config{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "admin",
		SecretAccessKey: "password123",
		UseSSL:          false,
		BucketName:      "coverage-reports",
	}

	// 1. 初始化 MinIO 客户端
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		t.Fatalf("Failed to create minio client: %v", err)
	}

	ctx := context.Background()

	// 2. 确保 Bucket 存在
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		t.Fatalf("Failed to check bucket existence: %v", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			t.Fatalf("Failed to create bucket: %v", err)
		}
	}

	// 3. 初始化我们的 OSS 包装类
	oss, err := NewMinioOSS(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize NewMinioOSS: %v", err)
	}

	// 4. 准备写入的内容
	content := "hello world, this is a test coverage report content"
	reader := strings.NewReader(content)
	objectName := "test/hello.txt"

	// 5. 写入文件
	err = oss.PutObject(ctx, cfg.BucketName, objectName, reader, int64(len(content)))
	if err != nil {
		t.Fatalf("Failed to put object: %v", err)
	}
	t.Logf("Successfully uploaded %s to %s", objectName, cfg.BucketName)

	// 6. 验证文件是否存在并读取内容
	readCloser, err := oss.GetObject(ctx, cfg.BucketName, objectName)
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}
	defer readCloser.Close()

	var buf strings.Builder
	_, err = io.Copy(&buf, readCloser)
	if err != nil {
		t.Fatalf("Failed to read object content: %v", err)
	}

	if buf.String() != content {
		t.Fatalf("Content mismatch: expected %q, got %q", content, buf.String())
	}
	t.Logf("Verified content: %s", buf.String())

	// 7. 测试删除
	err = oss.DeleteObject(ctx, cfg.BucketName, objectName)
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}
	t.Logf("Successfully deleted %s", objectName)
}
