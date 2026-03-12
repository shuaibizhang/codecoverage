package diff

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

/*
diff --git a/readme.md b/readme.md                 (diff 头)
index 8d4e4dd..76af9bd 100644
--- a/readme.md                                     （旧文件）
+++ b/readme.md                                     （新文件）
@@ -1,112 +1,136 @@ （hunk 头）
-# Transparent Context SDK                          （删除行）
+# 项目介绍                                          （新增行）
+#### 项目定位
*/

// GitModel diff组件
type GitModel struct {
}

func NewGitModel() *GitModel {
	return &GitModel{}
}

// git文件类型
type GitFileType int

const (
	ModifiedFile GitFileType = iota // 修改文件
	NewFile                         // 新增文件
	DeleteFile                      // 删除文件
	RenameFile                      // 重命名文件
)

var (
	// diff file文件头
	diffPattern = regexp.MustCompile(`^diff --git ("?a/.+?"?) ("?b/.+?"?)$`)
	// diff hunk头
	linePattern = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
)

// DiffLineMode diff行类型
type DiffLineMode rune

const (
	// 新增行
	ADDED DiffLineMode = iota
	// 删除行
	REMOVED
	// 未更改行
	UNCHANGED
)

// git diff结构体
type GitDiff struct {
	DiffFiles []*DiffFile `json:"diff_files"`
}

func (g *GitDiff) CovertToMap() *GitDiffMap {
	res := &GitDiffMap{DiffFileMap: make(map[string]*DiffFile)}
	for _, file := range g.DiffFiles {
		res.DiffFileMap[file.NowFileName] = file
		if file.OriginFileName != file.NowFileName {
			res.DiffFileMap[file.OriginFileName] = file
		}
	}
	return res
}

type GitDiffMap struct {
	DiffFileMap map[string]*DiffFile `json:"diff_file_map"`
}

type DiffFile struct {
	// 原始文件名
	OriginFileName string `json:"origin_file_name"`
	// 当前文件名
	NowFileName string `json:"now_file_name"`
	// 文件类型
	FileType GitFileType `json:"file_type"` // new or delete or rename
	// diff hunks
	Hunks []*DiffHunk `json:"hunks"`
}

type DiffHunk struct {
	// 源文件块起始行
	OriginStartLine int `json:"origin_start_line"`
	// 源文件块行数
	OriginLineCount int `json:"origin_line_count"`
	// 修改后文件块起始行
	NewStartLine int `json:"new_start_line"`
	// 修改后文件块行数
	NewLineCount int `json:"new_line_count"`
	// 块内新增行号
	NewFileLines []int `json:"new_file_lines"`
	// 块内删除行号
	OldOriginLines []int `json:"old_origin_lines"`
	// 块内未修改的行号映射
	UnChangeLinesMap map[int]int `json:"unchange_lines_map"`
}

// GetDiffFile 根据文件名获取diff信息
func (d *GitDiffMap) GetDiffFile(fileName string) *DiffFile {
	if d == nil || d.DiffFileMap == nil {
		return nil
	}
	return d.DiffFileMap[fileName]
}

// GetFileChangeLines 获取文件中新增的行号
func (f *DiffFile) GetFileChangeLines() []int {
	var res []int
	for _, hunk := range f.Hunks {
		res = append(res, hunk.NewFileLines...)
	}
	sort.Ints(res)
	return res
}

// GetAddLinesCount 获取文件中新增的总行数
func (f *DiffFile) GetAddLinesCount() uint32 {
	var res uint32
	for _, hunk := range f.Hunks {
		res += uint32(len(hunk.NewFileLines))
	}
	return res
}

// GetDeleteLinesCount 获取文件中删除的总行数
func (f *DiffFile) GetDeleteLinesCount() uint32 {
	var res uint32
	for _, hunk := range f.Hunks {
		res += uint32(len(hunk.OldOriginLines))
	}
	return res
}

func NewHunk() *DiffHunk {
	return &DiffHunk{
		OriginStartLine:  0,
		OriginLineCount:  0,
		NewStartLine:     0,
		NewLineCount:     0,
		NewFileLines:     []int{},
		OldOriginLines:   []int{},
		UnChangeLinesMap: make(map[int]int),
	}
}

// ParserGitDiffFile 直接解析git diff 的原始数据
func (g *GitModel) ParserGitDiffFile(content string) (*GitDiff, error) {
	gitDiff := new(GitDiff)
	diffFile := new(DiffFile)
	hunk := new(DiffHunk)
	var isHunk bool
	var newLineNum, oldLineNum int
	var err error

	// 按行解析
	lines := strings.Split(content, "\n")
	// 遍历每一行
	for _, line := range lines {
		// 解析git diff文件头数据
		diffFile, err = parseFileHeader(line, &isHunk, gitDiff, diffFile)
		if err != nil {
			return nil, err
		}
		// 解析diff file的类型
		parseIsNewFile(line, diffFile)
		parseIsDeleteFile(line, diffFile)

		// 解析diff hunk头数据
		hunk, err = parseLineData(line, &isHunk, &oldLineNum, &newLineNum, hunk, diffFile)
		if err != nil {
			return nil, err
		}
		// 解析diff hunk数据
		parseHunk(line, &isHunk, &oldLineNum, &newLineNum, hunk)
	}
	return gitDiff, nil
}

// parseFileHeader 解析头部 git diff file
func parseFileHeader(line string, isHunk *bool, gitDiff *GitDiff, diffFile *DiffFile) (*DiffFile, error) {
	if strings.HasPrefix(line, "diff --git ") {
		*isHunk = false
		// 判断是否检测到开始识别新的diff文件了
		diffFile = new(DiffFile)
		gitDiff.DiffFiles = append(gitDiff.DiffFiles, diffFile)
		matches := diffPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			// 获取原始文件名、当前文件名
			diffFile.OriginFileName = strings.TrimPrefix(strings.Trim(matches[1], "\""), "a/")
			diffFile.NowFileName = strings.TrimPrefix(strings.Trim(matches[2], "\""), "b/")
		} else {
			return nil, fmt.Errorf("diff file match: line=%s", line)
		}
		// 改名操作
		if diffFile.NowFileName != diffFile.OriginFileName {
			diffFile.FileType = RenameFile
		}
	}
	return diffFile, nil
}

// parseIsNewFile 解析判断是否是新增文件，原来文件是/dev/null
func parseIsNewFile(line string, diffFile *DiffFile) {
	if line == "--- /dev/null" {
		diffFile.FileType = NewFile
	}
}

// parseIsDeleteFile 解析判断是否删除文件，当前文件是/dev/null
func parseIsDeleteFile(line string, diffFile *DiffFile) {
	if line == "+++ /dev/null" {
		diffFile.FileType = DeleteFile
	}
}

// 解析新旧文件变动的范围
func parseLineData(line string, isHunk *bool, oldLineNum, newLineNum *int, hunk *DiffHunk, diffFile *DiffFile) (*DiffHunk, error) {
	// 解析hunk头
	if strings.HasPrefix(line, "@@") {
		if diffFile == nil {
			return hunk, errors.New("diff file is nil")
		}
		*isHunk = true
		hunk = NewHunk()
		diffFile.Hunks = append(diffFile.Hunks, hunk)
		matches := linePattern.FindStringSubmatch(line)

		if len(matches) != 5 {
			return hunk, fmt.Errorf("parse @@ err,%v", line)
		}

		var err error
		hunk.OriginStartLine, err = strconv.Atoi(matches[1])
		if err != nil {
			return hunk, fmt.Errorf("parse @@ err: %v", line)
		}
		*oldLineNum = hunk.OriginStartLine
		if matches[2] != "" {
			hunk.OriginLineCount, err = strconv.Atoi(matches[2])
			if err != nil {
				return hunk, fmt.Errorf("parse @@ err: %v", line)
			}
		} else {
			hunk.OriginLineCount = 0 // 默认为1行
		}

		hunk.NewStartLine, err = strconv.Atoi(matches[3])
		if err != nil {
			return hunk, fmt.Errorf("parse @@ err: %v", line)
		}
		*newLineNum = hunk.NewStartLine
		if matches[4] != "" {
			hunk.NewLineCount, err = strconv.Atoi(matches[4])
			if err != nil {
				return hunk, fmt.Errorf("parse @@ err: %v", line)
			}
		} else {
			hunk.NewLineCount = 0 // 默认为1行
		}
	}
	return hunk, nil
}

// parseHunk 解析diff hunk数据
// 维护newLineNum、oldLineNum，用于做映射
func parseHunk(line string, isHunk *bool, oldLineNum, newLineNum *int, hunk *DiffHunk) {
	if *isHunk && isSourceLine(line) {
		m, err := lineMode(line)
		if err != nil {
			return
		}
		if hunk == nil {
			return
		}
		switch *m {
		case ADDED:
			hunk.NewFileLines = append(hunk.NewFileLines, *newLineNum)
			*newLineNum++
		case REMOVED:
			hunk.OldOriginLines = append(hunk.OldOriginLines, *oldLineNum)
			*oldLineNum++
		case UNCHANGED:
			// 块内未改变的行，记录下旧行号和新行号的映射关系
			hunk.UnChangeLinesMap[*oldLineNum] = *newLineNum
			*newLineNum++
			*oldLineNum++
		}
	}
}

// isSourceLine 判断是否是diff行，过滤掉hunk头、diff文件头、空行等
func isSourceLine(line string) bool {
	if line == `\ No newline at end of file` {
		return false
	}
	if l := len(line); l == 0 || (l >= 3 && (line[:3] == "---" || line[:3] == "+++")) {
		return false
	}
	return true
}

// lineMode 解析hunk中，每一行的变动类型（新增、删除、还是未变动）
func lineMode(line string) (*DiffLineMode, error) {
	var m DiffLineMode
	switch line[:1] {
	case " ":
		m = UNCHANGED
	case "+":
		m = ADDED
	case "-":
		m = REMOVED
	default:
		return nil, errors.New("could not parse line mode for line: \"" + line + "\"")
	}
	return &m, nil
}
