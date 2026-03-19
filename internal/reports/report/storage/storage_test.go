package storage

import (
	"context"
	"io"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
	"github.com/stretchr/testify/require"
)

// mockDataSource 模拟 datasource.DataSource 接口
type mockDataSource struct {
	data   []byte
	offset int64
}

func (m *mockDataSource) Read(p []byte) (n int, err error) {
	if m.offset >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.offset:])
	m.offset += int64(n)
	return n, nil
}

func (m *mockDataSource) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockDataSource) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = m.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(m.data)) + offset
	}
	m.offset = newOffset
	return m.offset, nil
}

func (m *mockDataSource) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *mockDataSource) WriteAt(p []byte, off int64) (n int, err error) {
	end := off + int64(len(p))
	if end > int64(len(m.data)) {
		newData := make([]byte, end)
		copy(newData, m.data)
		m.data = newData
	}
	copy(m.data[off:], p)
	return len(p), nil
}

func (m *mockDataSource) Close() error              { return nil }
func (m *mockDataSource) Truncate(size int64) error { m.data = m.data[:size]; return nil }
func (m *mockDataSource) Sync() error               { return nil }

// mockReportLock 模拟 reportlock.ReportLock 接口
type mockReportLock struct {
	locked bool
}

func (m *mockReportLock) Lock(ctx context.Context) error {
	m.locked = true
	return nil
}

func (m *mockReportLock) Unlock(ctx context.Context) error {
	m.locked = false
	return nil
}

func (m *mockReportLock) CanWrite(ctx context.Context) bool {
	return m.locked
}

// mockPartitionKey 模拟 partitionkey.PartitionKey 接口
type mockPartitionKey struct {
	offset int64
}

func (m *mockPartitionKey) Type() partitionkey.PartitionType { return partitionkey.ReportType }
func (m *mockPartitionKey) Marshal() (string, error)         { return "", nil }
func (m *mockPartitionKey) Unmarshal(data string) error      { return nil }
func (m *mockPartitionKey) RealPathPrefix() string           { return "test" }
func (m *mockPartitionKey) Offset() int64                    { return m.offset }
func (m *mockPartitionKey) SetOffset(o int64)                { m.offset = o }

func TestStorage_SetGetCoverLine(t *testing.T) {
	ctx := context.Background()
	metaSrc := &mockDataSource{}
	coverSrc := &mockDataSource{}
	lock := &mockReportLock{}
	s := NewStorage(metaSrc, coverSrc, lock)

	pk := &mockPartitionKey{}
	coverLines := []int32{1, 0, -1, 5}
	addedLines := []uint32{1, 2}

	// 测试 SetCoverLine
	newPk, err := s.SetCoverLine(ctx, pk, coverLines, addedLines)
	if err != nil {
		t.Fatalf("SetCoverLine failed: %v", err)
	}

	if newPk.Offset() != 0 {
		t.Errorf("expected offset 0, got %d", newPk.Offset())
	}

	// 测试 GetCoverLine
	gotLines, gotAdded, err := s.GetCoverLine(ctx, newPk)
	if err != nil {
		t.Fatalf("GetCoverLine failed: %v", err)
	}

	require.Equal(t, coverLines, gotLines)
	require.Equal(t, addedLines, gotAdded)
}

func TestStorage_SetLoadReport(t *testing.T) {
	ctx := context.Background()
	metaSrc := &mockDataSource{}
	coverSrc := &mockDataSource{}
	lock := &mockReportLock{}
	s := NewStorage(metaSrc, coverSrc, lock)

	// 使用真实的 CoverReportImpl 配合 Mock DataSource
	rep := &report.CoverReportImpl{
		Meta: report.MetaInfo{
			Module: "test-module",
		},
		Tree: tree.NewDirNode("*", "*"),
	}
	pk := &mockPartitionKey{}

	// 测试 SetReport
	_, err := s.SetReport(ctx, pk, rep)
	if err != nil {
		t.Fatalf("SetReport failed: %v", err)
	}

	// 测试 LoadReport
	newRep := &report.CoverReportImpl{
		Tree: tree.NewDirNode("*", "*"), // Decode 内部会重新创建，但结构需一致
	}
	err = s.LoadReport(ctx, pk, newRep)
	if err != nil {
		t.Fatalf("LoadReport failed: %v", err)
	}

	if newRep.Meta.Module != rep.Meta.Module {
		t.Errorf("expected module %s, got %s", rep.Meta.Module, newRep.Meta.Module)
	}
}
