package datasource

import "io"

// 数据源接口
type DataSource interface {
	io.Reader
	io.Writer
	io.Seeker
	io.ReaderAt
	io.WriterAt
	io.Closer

	Truncate(size int64) error
	Sync() error
}
