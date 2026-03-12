package codecoverage

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/datasource"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
)

// mockReportLock 实现 reportlock.ReportLock 接口用于集成测试
type mockReportLock struct{}

func (m *mockReportLock) Lock(ctx context.Context) error    { return nil }
func (m *mockReportLock) Unlock(ctx context.Context) error  { return nil }
func (m *mockReportLock) CanWrite(ctx context.Context) bool { return true }

func TestFullCoverageFlow(t *testing.T) {
	// 准备工作目录
	baseDir := "coverage"
	uploadDir := filepath.Join(baseDir, "upload")
	reportsDir := filepath.Join(baseDir, "reports")
	_ = os.MkdirAll(uploadDir, 0755)
	_ = os.MkdirAll(reportsDir, 0755)

	coverageOut := "coverage.out"

	// 1、执行 internal 下所有单元测试，将获取到的覆盖率数据写入 coverage.out 中
	t.Log("Step 1: Running tests in internal and generating coverage.out...")
	cmd := exec.Command("go", "test", "-v", "./internal/...", "-coverprofile="+coverageOut)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tests: %v\nOutput: %s", err, string(output))
	}

	// 2、调用 parser 组件解析覆盖率数据，将 json 文件保存到 coverage/upload 里
	t.Log("Step 2: Parsing coverage data and saving to coverage/upload...")
	content, err := os.ReadFile(coverageOut)
	if err != nil {
		t.Fatalf("Failed to read coverage.out: %v", err)
	}

	// 假设模块前缀为 github.com/shuaibizhang/codecoverage/
	prefix := "github.com/shuaibizhang/codecoverage/"
	goParser := parser.NewGoCovParser(prefix)
	normalInfo, err := goParser.Parse(string(content))
	if err != nil {
		t.Fatalf("Failed to parse coverage data: %v", err)
	}

	uploadJSON := filepath.Join(uploadDir, "coverage.json")
	jsonData, _ := json.MarshalIndent(normalInfo, "", "  ")
	if err := os.WriteFile(uploadJSON, jsonData, 0644); err != nil {
		t.Fatalf("Failed to save coverage.json: %v", err)
	}

	// 3、生成覆盖率报告，将报告持久化到 coverage/reports 里
	t.Log("Step 3: Generating coverage report and persisting to coverage/reports...")

	// 初始化存储 - 使用与 server 一致的文件名
	cnoPath := filepath.Join(reportsDir, "meta.cno")
	cdaPath := filepath.Join(reportsDir, "cover.cda")

	// 先删除旧文件，确保是全新的报告
	os.Remove(cnoPath)
	os.Remove(cdaPath)

	metaDS, err := datasource.CreateFileDataSource(cnoPath)
	if err != nil {
		t.Fatalf("Failed to create meta data source: %v", err)
	}
	defer metaDS.Close()

	coverDS, err := datasource.CreateFileDataSource(cdaPath)
	if err != nil {
		t.Fatalf("Failed to create cover data source: %v", err)
	}
	defer coverDS.Close()

	store := storage.NewStorage(metaDS, coverDS, &mockReportLock{})
	reportManager := manager.NewReportManager(store)

	// 创建报告
	ctx := context.Background()
	meta := report.MetaInfo{
		Module: "github.com/shuaibizhang/codecoverage",
		Branch: "main",
		Commit: "latest",
	}

	pk := partitionkey.NewReportKey(partitionkey.UnitTest, meta.Module, meta.Branch, meta.Commit)

	rep, err := reportManager.CreateReport(ctx, meta, pk)
	if err != nil {
		t.Fatalf("Failed to create report: %v", err)
	}

	// 填充数据
	for path, data := range normalInfo.CoverageData {
		err := rep.AddFile(path, data.CoverLineData, report.FileDiffInfo{})
		if err != nil {
			t.Errorf("Failed to add file %s to report: %v", path, err)
		}
	}

	// 持久化
	if err := rep.Flush(ctx); err != nil {
		t.Fatalf("Failed to flush report: %v", err)
	}

	// 4、验证生成的报告是否可以正常读取和解析
	t.Log("Step 4: Verifying the persisted report...")

	// 使用 Open 方法打开报告
	readMetaDS, err := datasource.OpenFileDataSource(cnoPath)
	if err != nil {
		t.Fatalf("Failed to open meta data source for reading: %v", err)
	}
	defer readMetaDS.Close()

	readCoverDS, err := datasource.OpenFileDataSource(cdaPath)
	if err != nil {
		t.Fatalf("Failed to open cover data source for reading: %v", err)
	}
	defer readCoverDS.Close()

	readStore := storage.NewStorage(readMetaDS, readCoverDS, &mockReportLock{})
	readManager := manager.NewReportManager(readStore)

	openedRep, err := readManager.Open(ctx, pk)
	if err != nil {
		t.Fatalf("Failed to open report: %v", err)
	}

	openedMeta := openedRep.GetMeta()
	if openedMeta.Module != meta.Module || openedMeta.Commit != meta.Commit {
		t.Errorf("Meta info mismatch: expected %+v, got %+v", meta, openedMeta)
	}

	// 验证文件是否存在
	for path := range normalInfo.CoverageData {
		if !openedRep.ExistFile(path) {
			t.Errorf("File %s missing in opened report", path)
		}
	}

	t.Logf("Full flow completed and verified. Results in %s and %s", uploadDir, reportsDir)
}
