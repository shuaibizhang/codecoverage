package datasource

import "os"

type FileDataSource struct {
	*os.File
}

func NewFileDataSource(file *os.File) *FileDataSource {
	return &FileDataSource{file}
}

// OpenFileDataSource 打开文件数据源，以读写模式打开文件
func OpenFileDataSource(path string) (*FileDataSource, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return NewFileDataSource(file), nil
}

// CreateFileDataSource 创建文件数据源，如果文件不存在则创建，存在则截断，清空现有数据
func CreateFileDataSource(path string) (*FileDataSource, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return NewFileDataSource(file), nil
}

func (f *FileDataSource) Truncate(size int64) error {
	return f.File.Truncate(size)
}

func (f *FileDataSource) Sync() error {
	return f.File.Sync()
}
