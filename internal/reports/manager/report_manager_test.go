package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/stretchr/testify/assert"
)

// mockReportLock 模拟 reportlock.ReportLock 接口，避免依赖 Redis
type mockReportLock struct{}

func (m *mockReportLock) Lock(ctx context.Context) error    { return nil }
func (m *mockReportLock) Unlock(ctx context.Context) error  { return nil }
func (m *mockReportLock) CanWrite(ctx context.Context) bool { return true }

// setupRealStorage 创建真实的存储实例
func setupRealStorage(t *testing.T) (report.Storage, func()) {
	tmpDir, err := os.MkdirTemp("", "report_test")
	if err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(tmpDir, "test.cno")
	coverPath := filepath.Join(tmpDir, "test.cda")

	metaFile, err := datasource.CreateFileDataSource(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	coverFile, err := datasource.CreateFileDataSource(coverPath)
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewStorage(metaFile, coverFile, &mockReportLock{})

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, cleanup
}

func testStorageFactory(s report.Storage) StorageFactory {
	return func(ctx context.Context, pk partitionkey.PartitionKey) (report.Storage, error) {
		return s, nil
	}
}

// mockPartitionKey 模拟 partitionkey.PartitionKey
type mockPartitionKey struct {
	offset int64
}

func (m *mockPartitionKey) Type() partitionkey.PartitionType { return partitionkey.ReportType }
func (m *mockPartitionKey) Marshal() (string, error)         { return "", nil }
func (m *mockPartitionKey) Unmarshal(data string) error      { return nil }
func (m *mockPartitionKey) RealPathPrefix() string           { return "test_report" }
func (m *mockPartitionKey) Offset() int64                    { return m.offset }
func (m *mockPartitionKey) SetOffset(o int64)                { m.offset = o }

func TestReportManager_CreateReport(t *testing.T) {
	storage, cleanup := setupRealStorage(t)
	defer cleanup()

	mgr := NewReportManager(testStorageFactory(storage))
	ctx := context.Background()
	meta := report.MetaInfo{Module: "test"}
	pk := &mockPartitionKey{}

	rep, err := mgr.CreateReport(ctx, meta, pk)
	assert.NoError(t, err)
	assert.NotNil(t, rep)
	assert.Equal(t, meta.Module, rep.GetMeta().Module)
}

func TestReportManager_Open(t *testing.T) {
	s, cleanup := setupRealStorage(t)
	defer cleanup()

	mgr := NewReportManager(testStorageFactory(s))
	ctx := context.Background()
	pk := &mockPartitionKey{}

	// 使用 mgr.CreateReport 创建报告
	meta := report.MetaInfo{Module: "test-open"}
	rep, err := mgr.CreateReport(ctx, meta, pk)
	assert.NoError(t, err)

	// 需要调用 Flush 才会真正写入存储
	err = rep.Flush(ctx)
	assert.NoError(t, err)

	// 重要：在重新打开之前，需要重置文件偏移量到开头，
	// 因为 setupRealStorage 中创建的 FileDataSource 共享同一个文件句柄，
	// Flush 之后文件指针在末尾。
	impl, _ := s.(interface{ GetMetaSource() datasource.DataSource })
	if impl != nil {
		impl.GetMetaSource().Seek(0, 0)
	}

	openedRep, err := mgr.Open(ctx, pk)
	assert.NoError(t, err)
	assert.NotNil(t, openedRep)
	assert.Equal(t, meta.Module, openedRep.GetMeta().Module)
}

func TestReportManager_MergeSameCommitReport(t *testing.T) {
	s, cleanup := setupRealStorage(t)
	defer cleanup()

	mgr := NewReportManager(testStorageFactory(s))
	ctx := context.Background()

	pk := &mockPartitionKey{}
	base := report.NewCoverReport(s, report.MetaInfo{Module: "base"}, pk)
	other := report.NewCoverReport(s, report.MetaInfo{Module: "other"}, pk)

	// 准备 other 报告的数据
	path := "main.go"
	otherLines := []int32{1, 1} // 全覆盖
	err := other.AddFile(path, otherLines, report.FileDiffInfo{})
	assert.NoError(t, err)

	// 准备 base 报告的数据 (部分覆盖)
	baseLines := []int32{1, 0}
	err = base.AddFile(path, baseLines, report.FileDiffInfo{})
	assert.NoError(t, err)

	// 执行合并
	err = mgr.MergeSameCommitReport(ctx, base, other)
	assert.NoError(t, err)

	// 验证 base 的统计数据是否更新
	node := base.FindNode(path)
	assert.NotNil(t, node)
	stat := node.GetStat()
	assert.Equal(t, uint32(100), stat.Coverage)

	// 验证合并后的行数据
	mergedLines, err := base.GetFileCoverLines(path)
	assert.NoError(t, err)
	expectedUintLines := []uint32{1, 1}
	assert.Equal(t, expectedUintLines, mergedLines)
}
