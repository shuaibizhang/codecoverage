package coder

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/tree"
)

func TestReportAndLineDecodingAndEncoding(t *testing.T) {
	cnoPath := filepath.Join("testdata", "test.cno")
	cdaPath := filepath.Join("testdata", "test.cda")
	genCnoPath := filepath.Join("testdata", "test_gen.cno")
	genCdaPath := filepath.Join("testdata", "test_gen.cda")

	// 1. 解码 .cno 文件获取 report
	cnoFile, err := os.Open(cnoPath)
	if err != nil {
		t.Fatalf("failed to open test.cno: %v", err)
	}
	defer cnoFile.Close()

	decoder := NewReportDecoder(cnoFile)
	rep := &report.CoverReportImpl{}
	if err := decoder.Decode(rep); err != nil {
		t.Fatalf("failed to decode report: %v", err)
	}

	// 2. 加载 .cda 数据
	cdaData, err := os.ReadFile(cdaPath)
	if err != nil {
		t.Fatalf("failed to read test.cda: %v", err)
	}

	// 3. 准备编码器
	cdaBuffer := &bytes.Buffer{}
	lineEncoder := NewCoverLineEncoder()

	// 4. 遍历树节点，解码行覆盖率，并使用 Encoder 重新编码到新的 CDA 数据流中
	var processNode func(node tree.TreeNode)
	processNode = func(node tree.TreeNode) {
		if fileNode, ok := node.(*tree.FileNode); ok {
			name := fileNode.Name()
			if fileNode.GetStat().TotalLines > 0 && (filepath.Ext(name) == ".go" || filepath.Ext(name) == ".java") {
				offset := fileNode.BlockOffset
				if offset >= int64(len(cdaData)) {
					t.Errorf("file %s offset %d out of bounds", fileNode.Path(), offset)
					return
				}

				// A. 解码原始数据
				lineDecoder := NewCoverLineDecoder(cdaData[offset:])
				coverLines, addedLines, err := lineDecoder.DecodeRawCoverLine()
				if err != nil {
					t.Errorf("failed to decode lines for %s at offset %d: %v", fileNode.Path(), offset, err)
					return
				}

				// B. 更新新的偏移量 (当前 CDA Buffer 的长度)
				fileNode.BlockOffset = int64(cdaBuffer.Len())

				// C. 重新编码并写入 Buffer
				encodedBlock, err := lineEncoder.Encode(coverLines, addedLines)
				if err != nil {
					t.Errorf("failed to encode lines for %s: %v", fileNode.Path(), err)
					return
				}
				cdaBuffer.Write(encodedBlock)

				t.Logf("Processed: %-40s | Old Offset: %-6d | New Offset: %-6d | Lines: %d",
					fileNode.Path(), offset, fileNode.BlockOffset, len(coverLines))
			}
		}

		for child := range node.Children() {
			processNode(child)
		}
	}

	if rep.Tree != nil {
		processNode(rep.Tree)
	} else {
		t.Fatal("report tree is nil")
	}

	// 5. 将编码后的 CDA 数据写入文件
	if err := os.WriteFile(genCdaPath, cdaBuffer.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test_gen.cda: %v", err)
	}
	t.Logf("Successfully generated %s, size: %d", genCdaPath, cdaBuffer.Len())

	// 6. 使用 ReportEncoder 将更新后的 report 编码并写入新的 CNO 文件
	genCnoFile, err := os.Create(genCnoPath)
	if err != nil {
		t.Fatalf("failed to create test_gen.cno: %v", err)
	}
	defer genCnoFile.Close()

	reportEncoder := NewReportEncoder(genCnoFile)
	if err := reportEncoder.Encode(rep); err != nil {
		t.Fatalf("failed to encode report to test_gen.cno: %v", err)
	}
	t.Logf("Successfully generated %s", genCnoPath)
}
