package partitionkey

import (
	"strings"
	"testing"
)

func TestReportKeyPathGeneration(t *testing.T) {
	module := "my-module"
	branch := "master"
	commit := "1234567890abcdef"

	t.Run("UnitTest Path", func(t *testing.T) {
		pk := NewReportKey(UnitTest, module, branch, commit)
		path := pk.RealPath()

		// 校验前缀：unit/{date}/my-module/
		if !strings.HasPrefix(path, "unit/") {
			t.Errorf("expected prefix unit/, got %s", path)
		}
		if !strings.Contains(path, "/"+module+"/") {
			t.Errorf("path should contain module: %s", path)
		}
		// 校验文件名包含 branch 和 commit 前 8 位
		if !strings.Contains(path, branch+"_12345678") {
			t.Errorf("path should contain branch and short commit: %s", path)
		}
		if !strings.HasSuffix(path, ".cno") {
			t.Errorf("path should end with .cno: %s", path)
		}
	})

	t.Run("AutoTest Path", func(t *testing.T) {
		planID := uint64(1001)
		execID := uint64(2002)
		pk := NewAutoReportKey(module, branch, commit, planID, execID)
		path := pk.RealPath()

		// 校验前缀：auto/{date}/my-module/
		if !strings.HasPrefix(path, "auto/") {
			t.Errorf("expected prefix auto/, got %s", path)
		}
		if !strings.Contains(path, "/"+module+"/") {
			t.Errorf("path should contain module: %s", path)
		}
		// 校验文件名：1001_2002.cno
		expectedSuffix := "1001_2002.cno"
		if !strings.HasSuffix(path, expectedSuffix) {
			t.Errorf("expected filename suffix %s, got %s", expectedSuffix, path)
		}
	})
}

func TestNewCoverageKey(t *testing.T) {
	cnoPath := "unit/2023-10-27/mod/br/mod_br_commit_123.cno"
	offset := int64(1024)

	pk := NewCoverageKey(cnoPath, offset)

	if pk.Type() != CoverageDataType {
		t.Errorf("expected type %s, got %s", CoverageDataType, pk.Type())
	}

	expectedPath := "unit/2023-10-27/mod/br/mod_br_commit_123.cda"
	if pk.RealPath() != expectedPath {
		t.Errorf("expected cda path %s, got %s", expectedPath, pk.RealPath())
	}

	if pk.Offset() != offset {
		t.Errorf("expected offset %d, got %d", offset, pk.Offset())
	}
}
