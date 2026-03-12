package diff

import (
	"os"
	"testing"
)

func TestParserGitDiffFile(t *testing.T) {
	// 读取测试文件内容
	content, err := os.ReadFile("testdata/test.diff")
	if err != nil {
		t.Fatalf("failed to read test.diff: %v", err)
	}

	model := NewGitModel()
	gitDiff, err := model.ParserGitDiffFile(string(content))
	if err != nil {
		t.Fatalf("ParserGitDiffFile error: %v", err)
	}

	// 1. 验证解析出的文件数量
	if len(gitDiff.DiffFiles) != 1 {
		t.Errorf("expected 1 diff file, got %d", len(gitDiff.DiffFiles))
	}

	diffFile := gitDiff.DiffFiles[0]

	// 2. 验证文件名
	expectedFile := "readme.md"
	if diffFile.OriginFileName != expectedFile {
		t.Errorf("expected origin file %s, got %s", expectedFile, diffFile.OriginFileName)
	}
	if diffFile.NowFileName != expectedFile {
		t.Errorf("expected now file %s, got %s", expectedFile, diffFile.NowFileName)
	}

	// 3. 验证 Hunk 信息
	if len(diffFile.Hunks) != 1 {
		t.Errorf("expected 1 hunk, got %d", len(diffFile.Hunks))
	}

	hunk := diffFile.Hunks[0]
	// @@ -1,112 +1,136 @@
	if hunk.OriginStartLine != 1 || hunk.OriginLineCount != 112 {
		t.Errorf("hunk origin range mismatch: got %d,%d, want 1,112", hunk.OriginStartLine, hunk.OriginLineCount)
	}
	if hunk.NewStartLine != 1 || hunk.NewLineCount != 136 {
		t.Errorf("hunk new range mismatch: got %d,%d, want 1,136", hunk.NewStartLine, hunk.NewLineCount)
	}

	// 4. 验证具体行变动
	// 第一行是删除的: -# Transparent Context SDK
	if len(hunk.OldOriginLines) == 0 || hunk.OldOriginLines[0] != 1 {
		t.Errorf("expected line 1 to be removed, but not found in OldOriginLines")
	}

	// 接下来是新增的:
	// +# 项目介绍
	// +#### 项目定位
	// ...
	if len(hunk.NewFileLines) < 2 {
		t.Fatalf("expected at least 2 added lines, got %d", len(hunk.NewFileLines))
	}
	if hunk.NewFileLines[0] != 1 || hunk.NewFileLines[1] != 2 {
		t.Errorf("added lines mismatch: got %v, want starting with [1, 2]", hunk.NewFileLines[:2])
	}

	// 5. 验证 GetAddLinesCount 和 GetDeleteLinesCount
	addCount := diffFile.GetAddLinesCount()
	delCount := diffFile.GetDeleteLinesCount()
	t.Logf("Total added lines: %d, deleted lines: %d", addCount, delCount)

	// 根据 test.diff 的大致内容验证
	if addCount == 0 {
		t.Errorf("addCount should not be 0")
	}
}
