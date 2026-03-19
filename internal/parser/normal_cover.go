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
	CoverageData map[string]*CoverData `json:"coverage_data"` // 覆盖率数据
}
