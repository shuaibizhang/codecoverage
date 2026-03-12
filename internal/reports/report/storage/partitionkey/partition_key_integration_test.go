package partitionkey_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

func TestPartitionKeyIntegration(t *testing.T) {
	// 1. 初始化 PartitionKey (模拟从上报参数中恢复)
	module := "nuwa/cover/codecov"
	branch := "master"
	commit := "123123"

	// 因为是外部测试，我们需要通过导出函数或导出的 struct 访问
	// 但是 reportKey 是私有的，我们需要一个公开的构造方法或者模拟 RealPath 逻辑

	// 我们使用导出的构造函数
	pk := partitionkey.NewReportKey(partitionkey.IntegrateTest, module, branch, commit)

	// 为了使 RealPath 产生确定的结果，我们只能通过 Marshal/Unmarshal 恢复带有时间戳的对象
	// 或者我们直接构建满足路径的文件名

	// 让我们通过反射或者修改 pk 来注入特定的时间戳？
	// 更好的办法是利用 RealPath 的 filename 生成逻辑

	// 我们手动构建一个 key 模拟从存储中恢复的场景
	keyData := `{"test_type":"integrate","module":"nuwa/cover/codecov","branch":"master","commit":"123123","timestamp":1731935936}`
	pk = partitionkey.NewReportKey(partitionkey.IntegrateTest, module, branch, commit)
	if err := pk.Unmarshal(keyData); err != nil {
		t.Fatalf("failed to unmarshal key: %v", err)
	}

	// 2. 获取 RealPath 并定位到 testdata
	relPath := pk.RealPath()
	fullCnoPath := filepath.Join("testdata", relPath)

	t.Logf("Checking CNO path: %s", fullCnoPath)

	// 3. 读取并解码 .cno
	cnoFile, err := os.Open(fullCnoPath)
	if err != nil {
		t.Fatalf("failed to open .cno at %s: %v", fullCnoPath, err)
	}
	defer cnoFile.Close()

	decoder := coder.NewReportDecoder(cnoFile)
	rep := &report.CoverReportImpl{}
	if err := decoder.Decode(rep); err != nil {
		t.Fatalf("failed to decode report: %v", err)
	}

	// 验证元数据
	if rep.Meta.Module != module {
		t.Errorf("expected module %s, got %s", module, rep.Meta.Module)
	}

	// 4. 查找特定文件并根据物理偏移量获取行覆盖数据
	// 我们查找 internal/analysis/percent.go
	var percentNode *tree.FileNode
	var findPercent func(node tree.TreeNode)
	findPercent = func(node tree.TreeNode) {
		if fn, ok := node.(*tree.FileNode); ok && fn.Path() == "internal/analysis/percent.go" {
			percentNode = fn
			return
		}
		for child := range node.Children() {
			findPercent(child)
		}
	}
	findPercent(rep.Tree)

	if percentNode == nil {
		t.Fatal("could not find internal/analysis/percent.go in report tree")
	}

	// 5. 根据 CNO 路径和节点的 BlockOffset 创建 CoverageKey
	offset := int64(percentNode.BlockOffset)
	covKey := partitionkey.NewCoverageKey(fullCnoPath, offset)

	cdaPath := covKey.RealPath()
	t.Logf("CDA RealPath from CoverageKey: %s", cdaPath)

	// 6. 读取 CDA 并解码行覆盖率
	cdaData, err := os.ReadFile(cdaPath)
	if err != nil {
		t.Fatalf("failed to read .cda at %s: %v", cdaPath, err)
	}

	// 根据偏移量截取数据并解码
	lineDecoder := coder.NewCoverLineDecoder(cdaData[covKey.Offset():])
	coverLines, _, err := lineDecoder.DecodeRawCoverLine()
	if err != nil {
		t.Fatalf("failed to decode cover lines: %v", err)
	}

	// 校验数据：percent.go 有 52 行
	if len(coverLines) != 52 {
		t.Errorf("expected 52 lines for percent.go, got %d", len(coverLines))
	}
	t.Logf("Successfully decoded %d lines for %s using PartitionKey and Offset", len(coverLines), percentNode.Path())
}
