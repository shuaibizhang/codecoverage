package parser

// 覆盖率数据部分
type CoverData struct {
	TotalLines    uint    `json:"total_lines"`     // 代码总行数
	CoverLines    uint    `json:"cover_lines"`     // 覆盖的代码行数
	InstrLines    uint    `json:"instr_lines"`     // 指令行数
	CoverLineData []int32 `json:"cover_line_data"` // 覆盖的代码行数据 [-1,0,n] index： 行号，指令行-1 未覆盖 0 覆盖次数 n
}

// MetaInfo 覆盖率元数据部分
type MetaInfo struct {
	HostName      string `json:"hostname"`        // 上报主机名
	Module        string `json:"module"`          // 模块名
	Branch        string `json:"branch"`          // 分支名
	Commit        string `json:"commit"`          // 提交哈希
	BaseCommit    string `json:"base_commit"`     // 基础提交哈希
	UnittestRunID string `json:"unittest_run_id"` // 单元测试执行id
	Language      string `json:"language"`        // 编程语言
	FilePrefix    string `json:"file_prefix"`     // 文件前缀
}

// CovNormalInfo 归一化覆盖率信息，屏蔽语言差异
type CovNormalInfo struct {
	MetaInfo
	CoverageData CoverDataMap `json:"coverage_data"` // 覆盖率数据
}

type CoverDataMap map[string]*CoverData

func GetIncrChangeCoverMap(oldData, newData CoverDataMap) (CoverDataMap, uint, error) {
	// 简单的增量计算逻辑，实际可能更复杂
	incrData := make(CoverDataMap)
	var totalIncr uint
	for file, newCov := range newData {
		oldCov, ok := oldData[file]
		if !ok {
			incrData[file] = newCov
			totalIncr += newCov.CoverLines
			continue
		}
		// 比较逻辑... 这里先简化
		if newCov.CoverLines > oldCov.CoverLines {
			incrData[file] = newCov
			totalIncr += (newCov.CoverLines - oldCov.CoverLines)
		}
	}
	return incrData, totalIncr, nil
}

func CompressCommonPrefix(data CoverDataMap) (string, CoverDataMap) {
	if len(data) == 0 {
		return "", data
	}

	// 找出所有文件路径的公共前缀
	var commonPrefix string
	first := true
	for path := range data {
		if first {
			commonPrefix = path
			first = false
			continue
		}

		// 找到 commonPrefix 和 path 的公共前缀
		i := 0
		for i < len(commonPrefix) && i < len(path) && commonPrefix[i] == path[i] {
			i++
		}
		commonPrefix = commonPrefix[:i]
		if commonPrefix == "" {
			break
		}
	}

	// 如果没有公共前缀，直接返回
	if commonPrefix == "" {
		return "", data
	}

	// 如果公共前缀不是以 / 结尾，截取到最后一个 /
	lastSlash := -1
	for i, c := range commonPrefix {
		if c == '/' {
			lastSlash = i
		}
	}
	if lastSlash != -1 {
		commonPrefix = commonPrefix[:lastSlash+1]
	} else {
		commonPrefix = ""
	}

	if commonPrefix == "" {
		return "", data
	}

	// 压缩数据
	newData := make(CoverDataMap)
	for path, cov := range data {
		newPath := path[len(commonPrefix):]
		newData[newPath] = cov
	}

	return commonPrefix, newData
}
