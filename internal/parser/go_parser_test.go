package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoCovParser_Parse(t *testing.T) {
	prefix := "github.com/shuaibizhang/codecoverage/"
	parser := NewGoCovParser(prefix)

	content := `mode: set
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:19.56,21.2 1 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:23.109,25.16 2 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:25.16,27.3 1 0
`

	res, err := parser.Parse(content)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	filePath := "internal/reports/report/storage/coder/cover_line_decoder.go"
	data, ok := res.CoverageData[filePath]
	assert.True(t, ok)
	assert.Equal(t, uint(27), data.TotalLines)
	assert.Equal(t, uint(8), data.InstrLines)
	assert.Equal(t, uint(6), data.CoverLines)

	// 验证具体的行覆盖情况
	// 第 20 行在第一个块内 (19-21)，CoverCount=1
	assert.Equal(t, int32(1), data.CoverLineData[19])
	// 第 24 行在第二个块内 (23-25)，CoverCount=1
	assert.Equal(t, int32(1), data.CoverLineData[23])
	// 第 26 行在第三个块内 (25-27)，CoverCount=0
	assert.Equal(t, int32(0), data.CoverLineData[25])
	// 第 1 行不在任何块内，应该是 NotInstrLine (-1)
	assert.Equal(t, NotInstrLine, data.CoverLineData[0])
}

func TestGoCovParser_Parse_Empty(t *testing.T) {
	parser := NewGoCovParser("")
	res, err := parser.Parse("mode: set\n")
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGoCovParser_ScanCoverageFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cov_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	f1 := filepath.Join(tempDir, "a.out")
	f2 := filepath.Join(tempDir, "b.txt")
	os.WriteFile(f1, []byte("test"), 0644)
	os.WriteFile(f2, []byte("test"), 0644)

	parser := NewGoCovParser("")
	files, err := parser.ScanCoverageFiles(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files, f1)
}

func TestGoCovParser_ParseMultiFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cov_multi_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	content := `mode: set
main.go:1.0,2.0 1 1
`
	f1 := filepath.Join(tempDir, "main.out")
	os.WriteFile(f1, []byte(content), 0644)

	parser := NewGoCovParser("")
	moduleFiles := map[string]string{
		"moduleA": f1,
	}

	res, err := parser.ParseMultiFiles(moduleFiles)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Contains(t, res.CoverageData, filepath.Join("moduleA", "main.go"))
}
