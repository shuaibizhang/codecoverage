package datasource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileDataSource(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file.dat")

	t.Run("CreateFileDataSource", func(t *testing.T) {
		ds, err := CreateFileDataSource(testFilePath)
		if err != nil {
			t.Fatalf("failed to create file data source: %v", err)
		}
		defer ds.Close()

		// 写入数据验证
		data := []byte("hello world")
		if _, err := ds.Write(data); err != nil {
			t.Fatalf("failed to write data: %v", err)
		}

		// 检查文件是否存在
		if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
			t.Fatal("file should exist after CreateFileDataSource")
		}
	})

	t.Run("OpenFileDataSource", func(t *testing.T) {
		// 再次打开已存在的文件
		ds, err := OpenFileDataSource(testFilePath)
		if err != nil {
			t.Fatalf("failed to open file data source: %v", err)
		}
		defer ds.Close()

		// 读取数据验证内容
		buf := make([]byte, 11)
		if _, err := ds.ReadAt(buf, 0); err != nil {
			t.Fatalf("failed to read data: %v", err)
		}
		if string(buf) != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", string(buf))
		}
	})

	t.Run("CreateFileDataSource_Truncate", func(t *testing.T) {
		// 再次调用 Create，应该截断文件
		ds, err := CreateFileDataSource(testFilePath)
		if err != nil {
			t.Fatalf("failed to create file data source: %v", err)
		}
		defer ds.Close()

		info, err := ds.Stat()
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("expected empty file after truncation, got size %d", info.Size())
		}
	})

	t.Run("OpenFileDataSource_NotExist", func(t *testing.T) {
		notExistPath := filepath.Join(tempDir, "not_exist.dat")
		_, err := OpenFileDataSource(notExistPath)
		if err == nil {
			t.Fatal("expected error when opening non-existent file, got nil")
		}
	})
}
