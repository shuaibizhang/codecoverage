package datasource

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/shuaibizhang/codecoverage/internal/oss"
)

type ossDataSource struct {
	ossCli     oss.OSS
	bucketName string
	objectName string
	localFile  *os.File
	isDirty    bool
}

func NewOSSDataSource(ctx context.Context, ossCli oss.OSS, bucketName, objectName string) (DataSource, error) {
	// 1. 创建本地临时文件
	tmpFile, err := os.CreateTemp("", "oss-ds-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// 2. 尝试从 OSS 下载对象
	_, err = ossCli.StatObject(ctx, bucketName, objectName)
	if err == nil {
		reader, err := ossCli.GetObject(ctx, bucketName, objectName)
		if err == nil {
			defer reader.Close()
			_, err = io.Copy(tmpFile, reader)
			if err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				return nil, fmt.Errorf("failed to download from oss: %w", err)
			}
			// 重置偏移量到开头
			_, _ = tmpFile.Seek(0, io.SeekStart)
		} else {
			fmt.Printf("Warning: failed to get object %s from bucket %s despite stat success: %v\n", objectName, bucketName, err)
		}
	} else {
		// 更加健壮地检查是否为 404/NoSuchKey
		errResp := minio.ToErrorResponse(err)
		errMsg := err.Error()
		isNotFound := errResp.Code == "NoSuchKey" ||
			errResp.Code == "NoSuchBucket" ||
			errResp.StatusCode == 404 ||
			strings.Contains(errMsg, "NoSuchKey") ||
			strings.Contains(errMsg, "does not exist")

		if isNotFound {
			// 如果对象或桶不存在，视为正常情况（创建新报告），直接使用空文件
			fmt.Printf("Info: object %s not found in bucket %s, starting with empty file\n", objectName, bucketName)
		} else {
			// 其他错误（如权限、网络等），报错退出
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, fmt.Errorf("failed to stat object %s from oss (code: %s, status: %d): %w",
				objectName, errResp.Code, errResp.StatusCode, err)
		}
	}

	return &ossDataSource{
		ossCli:     ossCli,
		bucketName: bucketName,
		objectName: objectName,
		localFile:  tmpFile,
	}, nil
}

func (s *ossDataSource) Read(p []byte) (n int, err error) {
	return s.localFile.Read(p)
}

func (s *ossDataSource) Write(p []byte) (n int, err error) {
	n, err = s.localFile.Write(p)
	if n > 0 {
		s.isDirty = true
	}
	return n, err
}

func (s *ossDataSource) Seek(offset int64, whence int) (int64, error) {
	return s.localFile.Seek(offset, whence)
}

func (s *ossDataSource) ReadAt(p []byte, off int64) (n int, err error) {
	return s.localFile.ReadAt(p, off)
}

func (s *ossDataSource) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = s.localFile.WriteAt(p, off)
	if n > 0 {
		s.isDirty = true
	}
	return n, err
}

func (s *ossDataSource) Truncate(size int64) error {
	err := s.localFile.Truncate(size)
	if err == nil {
		s.isDirty = true
	}
	return err
}

func (s *ossDataSource) Sync() error {
	if !s.isDirty {
		return nil
	}

	// 1. 先同步本地文件到磁盘
	if err := s.localFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync local file: %w", err)
	}

	// 2. 上传到 OSS
	// 重新打开文件进行上传，避免影响当前的 Seek 指针
	uploadFile, err := os.Open(s.localFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open local file for upload: %w", err)
	}
	defer uploadFile.Close()

	info, err := uploadFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	err = s.ossCli.PutObject(context.Background(), s.bucketName, s.objectName, uploadFile, info.Size())
	if err != nil {
		return fmt.Errorf("failed to upload to oss: %w", err)
	}

	s.isDirty = false
	return nil
}

func (s *ossDataSource) Close() error {
	// 关闭前同步
	if err := s.Sync(); err != nil {
		fmt.Printf("Warning: failed to sync before close object %s in bucket %s: %v\n", s.objectName, s.bucketName, err)
	}

	localPath := s.localFile.Name()
	err := s.localFile.Close()
	// 删除临时文件
	os.Remove(localPath)
	return err
}
