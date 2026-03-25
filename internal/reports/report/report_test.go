package report

import (
	"context"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockStorage 模拟 report.Storage 接口
type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) SetCoverLine(ctx context.Context, pk partitionkey.PartitionKey, coverLines []int32, addedLines []uint32) (partitionkey.PartitionKey, error) {
	args := m.Called(ctx, pk, coverLines, addedLines)
	return args.Get(0).(partitionkey.PartitionKey), args.Error(1)
}

func (m *mockStorage) GetCoverLine(ctx context.Context, pk partitionkey.PartitionKey) ([]int32, []uint32, error) {
	args := m.Called(ctx, pk)
	return args.Get(0).([]int32), args.Get(1).([]uint32), args.Error(2)
}

func (m *mockStorage) SetReport(ctx context.Context, pk partitionkey.PartitionKey, report CoverReport) (partitionkey.PartitionKey, error) {
	args := m.Called(ctx, pk, report)
	return args.Get(0).(partitionkey.PartitionKey), args.Error(1)
}

func (m *mockStorage) LoadReport(ctx context.Context, pk partitionkey.PartitionKey, report CoverReport) error {
	args := m.Called(ctx, pk, report)
	return args.Error(0)
}

func (m *mockStorage) GetCoverLineWithFlag(ctx context.Context, pk partitionkey.PartitionKey) ([]uint32, error) {
	args := m.Called(ctx, pk)
	return args.Get(0).([]uint32), args.Error(1)
}

func (m *mockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

// mockPartitionKey 模拟 partitionkey.PartitionKey 接口
type mockPartitionKey struct {
	offset int64
}

func (m *mockPartitionKey) Type() partitionkey.PartitionType { return partitionkey.ReportType }
func (m *mockPartitionKey) Marshal() (string, error)         { return "", nil }
func (m *mockPartitionKey) Unmarshal(data string) error      { return nil }
func (m *mockPartitionKey) RealPathPrefix() string           { return "test_path" }
func (m *mockPartitionKey) Offset() int64                    { return m.offset }
func (m *mockPartitionKey) SetOffset(o int64)                { m.offset = o }
func (m *mockPartitionKey) GetModule() string                { return "" }
func (m *mockPartitionKey) GetBranch() string                { return "" }
func (m *mockPartitionKey) GetCommit() string                { return "" }

func TestCoverReportImpl_AddFile(t *testing.T) {
	storage := new(mockStorage)
	meta := MetaInfo{Module: "test-module"}
	pk := &mockPartitionKey{}
	r := NewCoverReport(storage, meta, pk)

	path := "src/main.go"
	lines := []int32{1, 0, -1, 1} // 1: covered, 0: not covered, -1: not instrumented
	diffInfo := FileDiffInfo{
		AddedLines:  []uint32{1, 2, 4},
		AddLines:    3,
		DeleteLines: 1,
	}

	// 预期存储调用
	storage.On("SetCoverLine", mock.Anything, mock.Anything, lines, diffInfo.AddedLines).Return(&mockPartitionKey{offset: 100}, nil)

	err := r.AddFile(path, lines, diffInfo)
	assert.NoError(t, err)
	assert.True(t, r.Change)
	assert.Equal(t, uint32(1), r.Meta.TotalFiles)

	// 验证目录树结构
	node := r.findNode(path)
	assert.NotNil(t, node)
	fileNode, ok := node.(*tree.FileNode)
	assert.True(t, ok)
	assert.Equal(t, int64(100), fileNode.BlockOffset)

	// 验证统计数据
	stats, err := r.ListFileStats(path, false)
	assert.NoError(t, err)
	assert.Len(t, stats, 1)
	stat := stats[0]
	assert.Equal(t, uint32(4), stat.TotalLines)
	assert.Equal(t, uint32(3), stat.InstrLines)
	assert.Equal(t, uint32(2), stat.CoverLines)
	assert.Equal(t, uint32(66), stat.Coverage)
	assert.Equal(t, uint32(3), stat.AddLines)
	assert.Equal(t, uint32(1), stat.DeleteLines)
	assert.Equal(t, uint32(3), stat.IncrInstrLines) // Line 1 (covered), Line 2 (not covered), Line 4 (covered). Line 3 (-1) is ignored.
	assert.Equal(t, uint32(2), stat.IncrCoverLines) // Line 1 (covered), Line 4 (covered).
	assert.Equal(t, uint32(66), stat.IncrCoverage)  // 2/3 = 66%
	assert.True(t, stat.HasIncrement)

	// 验证增量模式下的过滤
	statsIncr, err := r.ListFileStats(path, true)
	assert.NoError(t, err)
	assert.Len(t, statsIncr, 1)
	assert.Equal(t, stat, statsIncr[0])

	// 添加一个没有增量的文件
	path2 := "src/other.go"
	lines2 := []int32{1, 1}
	diffInfo2 := FileDiffInfo{}
	storage.On("SetCoverLine", mock.Anything, mock.Anything, lines2, mock.Anything).Return(&mockPartitionKey{offset: 300}, nil)
	_ = r.AddFile(path2, lines2, diffInfo2)

	// 验证在增量模式下，src 目录应该只包含 main.go
	statsDir, err := r.ListFileStats("src", true)
	assert.NoError(t, err)
	assert.Len(t, statsDir, 1)
	assert.Equal(t, "main.go", statsDir[0].Name)

	// 验证在全量模式下，src 目录包含两个文件
	statsDirFull, err := r.ListFileStats("src", false)
	assert.NoError(t, err)
	assert.Len(t, statsDirFull, 2)

	storage.AssertExpectations(t)
}

func TestCoverReportImpl_UpdateFile(t *testing.T) {
	storage := new(mockStorage)
	meta := MetaInfo{Module: "test-module"}
	pk := &mockPartitionKey{}
	r := NewCoverReport(storage, meta, pk)

	path := "src/main.go"
	lines := []int32{1, 0}
	diffInfo := FileDiffInfo{}

	// 先添加文件
	storage.On("SetCoverLine", mock.Anything, mock.Anything, lines, mock.Anything).Return(&mockPartitionKey{offset: 100}, nil)
	_ = r.AddFile(path, lines, diffInfo)

	// 更新文件
	newLines := []int32{1, 1}
	storage.On("SetCoverLine", mock.Anything, mock.Anything, newLines, mock.Anything).Return(&mockPartitionKey{offset: 200}, nil)

	err := r.UpdateFile(path, newLines, diffInfo, 0)
	assert.NoError(t, err)

	node := r.findNode(path)
	fileNode := node.(*tree.FileNode)
	assert.Equal(t, int64(200), fileNode.BlockOffset)
	assert.Equal(t, uint32(100), fileNode.GetStat().Coverage)

	storage.AssertExpectations(t)
}

func TestCoverReportImpl_GetFileCoverLines(t *testing.T) {
	storage := new(mockStorage)
	meta := MetaInfo{Module: "test-module"}
	pk := &mockPartitionKey{}
	r := NewCoverReport(storage, meta, pk)

	path := "src/main.go"
	lines := []int32{1, 0}
	uintLines := []uint32{1, 0}
	storage.On("SetCoverLine", mock.Anything, mock.Anything, lines, mock.Anything).Return(&mockPartitionKey{offset: 100}, nil)
	_ = r.AddFile(path, lines, FileDiffInfo{})

	// 获取覆盖行
	storage.On("GetCoverLineWithFlag", mock.Anything, mock.MatchedBy(func(p partitionkey.PartitionKey) bool {
		return p.Offset() == 100
	})).Return(uintLines, nil)

	gotLines, err := r.GetFileCoverLines(path)
	assert.NoError(t, err)
	assert.Equal(t, uintLines, gotLines)

	storage.AssertExpectations(t)
}

func TestCoverReportImpl_Flush(t *testing.T) {
	storage := new(mockStorage)
	meta := MetaInfo{Module: "test-module"}
	pk := &mockPartitionKey{}
	r := NewCoverReport(storage, meta, pk)

	storage.On("SetReport", mock.Anything, pk, r).Return(pk, nil)

	err := r.Flush(context.Background())
	assert.NoError(t, err)

	storage.AssertExpectations(t)
}
