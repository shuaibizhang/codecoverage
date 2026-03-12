package parser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var goPattern = regexp.MustCompile(`^([^:]+):(\d+)\.\d+,(\d+)\.\d+ (\d+) (\d+)$`)

type GoCovLineInfo struct {
	FilePath      string
	StartLine     int
	EndLine       int
	InstrLineNums int
	CoverCount    int
}

// go语言覆盖率解析器
type goCovParser struct {
	prefix string
}

func NewGoCovParser(prefix string) *goCovParser {
	return &goCovParser{
		prefix: prefix,
	}
}

/*
覆盖率文件格式： 文件名:起始行.起始列,结束行.结束列 块内语句数 块内执行数
mode: set
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:19.56,21.2 1 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:23.109,25.16 2 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:25.16,27.3 1 0
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:29.2,31.32 3 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:31.32,33.29 1 1
github.com/shuaibizhang/codecoverage/internal/reports/report/storage/coder/cover_line_decoder.go:33.29,35.4 1 1
*/
func (g *goCovParser) Parse(content string) (*CovNormalInfo, error) {
	// 按行进行解析
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil, errors.New("cover content is empty")
	}

	var hasData bool
	var res CovNormalInfo
	// 每个文件的行覆盖信息map
	fileCovMap := make(map[string][]int32, 0)

	// 处理每一行
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// 解析覆盖率模式行，跳过
		if strings.HasPrefix(line, "mode") {
			continue
		}

		hasData = true
		// 获取解析到的覆盖行信息
		goCovLineInfo, err := g.parseCoverFileLine(line)
		if err != nil {
			return nil, err
		}

		// 获取对应文件的行覆盖信息数组
		covLines, ok := fileCovMap[goCovLineInfo.FilePath]
		if !ok {
			covLines = make([]int32, 0)
		}

		// 检查 covLines 长度是否比当前块结束行大
		if len(covLines) < goCovLineInfo.EndLine {
			oldLen := len(covLines)
			// 扩容长度
			covLines = append(covLines, make([]int32, goCovLineInfo.EndLine-oldLen)...)
			for j := oldLen; j < goCovLineInfo.EndLine; j++ {
				covLines[j] = NotInstrLine
			}
		}

		// 更新块内的覆盖信息
		for j := goCovLineInfo.StartLine - 1; j < goCovLineInfo.EndLine; j++ {
			if j >= 0 && j < len(covLines) {
				// 如果原本是非指令行，或者当前的覆盖次数更大，则更新
				if covLines[j] == NotInstrLine || int32(goCovLineInfo.CoverCount) > covLines[j] {
					covLines[j] = int32(goCovLineInfo.CoverCount)
				}
			}
		}
		fileCovMap[goCovLineInfo.FilePath] = covLines
	}

	if !hasData {
		return nil, errors.New("cover content is empty")
	}

	res.CoverageData = make(map[string]*CoverData)
	for filePath, coverLineData := range fileCovMap {
		covDetailData := new(CoverData)
		covDetailData.TotalLines = uint(len(coverLineData))
		covDetailData.CoverLines = uint(g.calcCoverLines(coverLineData))
		covDetailData.InstrLines = uint(g.calcInstrLines(coverLineData))
		covDetailData.CoverLineData = coverLineData
		res.CoverageData[filePath] = covDetailData
	}

	return &res, nil
}

func (g *goCovParser) calcCoverLines(covLines []int32) uint {
	var coverLines uint
	for _, coverCount := range covLines {
		if coverCount > 0 {
			coverLines++
		}
	}
	return coverLines
}

func (g *goCovParser) calcInstrLines(covLines []int32) uint {
	var instrLines uint
	for _, coverCount := range covLines {
		if coverCount != NotInstrLine {
			instrLines++
		}
	}
	return instrLines
}

func (g *goCovParser) parseCoverFileLine(line string) (*GoCovLineInfo, error) {
	matches := goPattern.FindStringSubmatch(line)
	if len(matches) != 6 {
		return nil, errors.New("invalild cover line")
	}

	// 去除前缀
	filePath := strings.TrimPrefix(matches[1], g.prefix)
	startLine, _ := strconv.Atoi(matches[2])
	endLine, _ := strconv.Atoi(matches[3])
	instrLineNums, _ := strconv.Atoi(matches[4])
	coverCount, _ := strconv.Atoi(matches[5])

	return &GoCovLineInfo{
		FilePath:      filePath,
		StartLine:     startLine,
		EndLine:       endLine,
		InstrLineNums: instrLineNums,
		CoverCount:    coverCount,
	}, nil
}

// ScanCoverageFiles 扫描覆盖率文件
func (g *goCovParser) ScanCoverageFiles(rootDir string) (map[string]string, error) {
	coverageFiles := make(map[string]string)
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".out") {
			coverageFiles[path] = d.Name()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return coverageFiles, nil
}

// ParseMultiFiles 解析多个覆盖率文件
func (g *goCovParser) ParseMultiFiles(moduleFiles map[string]string) (*CovNormalInfo, error) {
	covInfo := &CovNormalInfo{
		CoverageData: make(map[string]*CoverData),
	}

	for module, filePath := range moduleFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read module file failed: %w", err)
		}

		moduleCov, err := g.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse cover data failed: %w", err)
		}

		for path, data := range moduleCov.CoverageData {
			if module != "." {
				path = filepath.Join(module, path)
			}
			covInfo.CoverageData[path] = data
		}
	}

	return covInfo, nil
}
